package server

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"sync/atomic"

	smtp "github.com/emersion/go-smtp"

	"mailproxy/internal/acl"
	"mailproxy/internal/config"
)

// startTLSRunner 独立运行「明文 + STARTTLS」监听，服务仅支持 STARTTLS、
// 不支持 465 隐式 TLS 的客户端（如 Veeam Backup & Replication）。
// 与隐式 TLS 监听完全解耦：独立的连接计数池（两端口限额互不挤占），
// 启动/运行失败仅记日志，不导致主服务退出。
type startTLSRunner struct {
	srv  *Server
	smtp *smtp.Server
	ln   net.Listener
}

// maybeStartStartTLS 按配置构造 STARTTLS 监听；starttls_listen 留空时返回 nil（不启用）。
func maybeStartStartTLS(s *Server, cfg *config.Config, tlsCfg *tls.Config) (*startTLSRunner, error) {
	if cfg.Server.StartTLSListen == "" {
		return nil, nil
	}
	ln, err := net.Listen("tcp", cfg.Server.StartTLSListen)
	if err != nil {
		return nil, fmt.Errorf("listen %s: %w", cfg.Server.StartTLSListen, err)
	}

	srv := smtp.NewServer(smtp.BackendFunc(s.newSession))
	srv.Domain = cfg.Server.Hostname
	srv.ReadTimeout = cfg.Server.IOTimeout.Duration()
	srv.WriteTimeout = cfg.Server.IOTimeout.Duration()
	srv.MaxMessageBytes = cfg.Server.MaxMessageBytes
	srv.MaxRecipients = 500
	srv.EnableSMTPUTF8 = true
	// === 增大最大行长度，支持超大 Base64 编码内容 ===
	srv.MaxLineLength = 10000 // 从默认的 2000 增大到 10000 字节
	srv.TLSConfig = tlsCfg // go-smtp 据此通告 STARTTLS 并完成升级
	// AllowInsecureAuth 保持默认 false：AUTH 仅在 STARTTLS 升级后允许

	return &startTLSRunner{
		srv:  s,
		smtp: srv,
		ln:   &stListener{Listener: ln, srv: s},
	}, nil
}

// serve 运行 STARTTLS 监听；异常仅记 Error 日志，不向上返回以免影响主服务。
func (r *startTLSRunner) serve() {
	if err := r.smtp.Serve(r.ln); err != nil && !errors.Is(err, net.ErrClosed) {
		r.srv.logger.Error("STARTTLS 监听异常退出", "listen", r.ln.Addr().String(), "error", err)
	}
}

// stop 优雅关闭 STARTTLS 监听，等待现有会话结束。
func (r *startTLSRunner) stop(ctx context.Context) error {
	return r.smtp.Shutdown(ctx)
}

// stListener 包装明文监听器，Accept 后做 IP 白名单与独立并发连接数控制。
// 逻辑与 wrappedListener 对齐但独立实现、不共享状态，保证新端口与 465 互不影响。
type stListener struct {
	net.Listener
	srv        *Server
	active     atomic.Int64
	warnedOnce atomic.Bool
}

func (l *stListener) Accept() (net.Conn, error) {
	for {
		conn, err := l.Listener.Accept()
		if err != nil {
			return nil, err
		}
		if accepted := l.check(conn); accepted != nil {
			return accepted, nil
		}
		// 被拒绝的连接已处理完毕，继续接受下一个
	}
}

// check 完成白名单与连接数检查；返回 nil 表示已拒绝并关闭连接。
func (l *stListener) check(conn net.Conn) net.Conn {
	cfg := l.srv.provider.Get()
	remote := conn.RemoteAddr().String()

	// === 记录新连接 ===
	l.srv.logger.Info("STARTTLS: 客户端已连接", "remote", remote)

	// IP 白名单（防开放中继）
	wlObj, err := acl.NewWhitelist(cfg.Server.IPWhitelist)
	switch {
	case err != nil:
		if !l.warnedOnce.Swap(true) {
			l.srv.logger.Error("IP 白名单解析失败，已放行全部客户端", "error", err)
		}
	case !wlObj.Empty():
		ip, ok := acl.IPFromAddr(conn.RemoteAddr())
		if ok && !wlObj.Allows(ip) {
			l.srv.logger.Warn("拒绝白名单外客户端", "remote", remote, "client_ip", ip.String())
			_, _ = conn.Write([]byte("554 5.7.1 Access denied: client IP not whitelisted\r\n"))
			conn.Close()
			return nil
		}
	}

	// 最大并发连接数（独立计数池）
	if int(l.active.Load()) >= cfg.Server.MaxConnections {
		l.srv.logger.Warn("连接数已达上限，拒绝新连接", "remote", remote, "max", cfg.Server.MaxConnections)
		_, _ = conn.Write([]byte("421 4.7.0 Too many connections, try again later\r\n"))
		conn.Close()
		return nil
	}

	l.active.Add(1)
	if l.srv.metrics != nil {
		l.srv.metrics.ActiveConnections.Inc()
	}
	return &stConn{Conn: conn, l: l}
}

// stConn 在连接关闭时回收计数。
type stConn struct {
	net.Conn
	l      *stListener
	closed atomic.Bool
}

func (c *stConn) Close() error {
	if c.closed.CompareAndSwap(false, true) {
		c.l.active.Add(-1)
		if c.l.srv.metrics != nil {
			c.l.srv.metrics.ActiveConnections.Dec()
		}
	}
	return c.Conn.Close()
}
