// Package server 实现对外的 SMTP over SSL 代理服务。
package server

import (
	"context"
	"crypto/tls"
	"fmt"
	"log/slog"
	"net"
	"sync/atomic"
	"time"

	smtp "github.com/emersion/go-smtp"

	"mailproxy/internal/acl"
	"mailproxy/internal/config"
	"mailproxy/internal/metrics"
)

// Server 对外 SMTP over SSL 代理服务。
type Server struct {
	provider *config.Provider
	logger   *slog.Logger
	metrics  *metrics.Metrics // 可为 nil（未启用）

	smtp *smtp.Server
	st   *startTLSRunner // 可选 STARTTLS 监听（未配置时为 nil）
}

func New(provider *config.Provider, logger *slog.Logger, m *metrics.Metrics) *Server {
	return &Server{provider: provider, logger: logger, metrics: m}
}

// Run 监听 TLS 端口并服务，直到 srv.Stop 或 ctx 取消。
func (s *Server) Run(ctx context.Context) error {
	cfg := s.provider.Get()

	tlsCfg, err := loadTLSConfig(cfg)
	if err != nil {
		return err
	}

	s.smtp = newSMTPServer(cfg, s)
	ln, err := tls.Listen("tcp", cfg.Server.Listen, tlsCfg)
	if err != nil {
		return fmt.Errorf("listen %s: %w", cfg.Server.Listen, err)
	}
	s.logger.Info("SMTP over SSL 代理服务已启动",
		"listen", cfg.Server.Listen,
		"auth_mode", cfg.Auth.Mode,
		"max_connections", cfg.Server.MaxConnections,
	)

	// 可选 STARTTLS 监听：独立运行，启动失败仅记日志，不影响主服务
	st, err := maybeStartStartTLS(s, cfg, tlsCfg)
	if err != nil {
		s.logger.Error("STARTTLS 监听启动失败", "listen", cfg.Server.StartTLSListen, "error", err)
	} else if st != nil {
		s.st = st
		go st.serve()
		s.logger.Info("STARTTLS 监听已启动", "listen", cfg.Server.StartTLSListen)
	}

	ln = s.wrapListener(ln, cfg.Server.HandshakeTimeout.Duration())
	return s.smtp.Serve(ln)
}

// Stop 停止接受新连接，等待现有会话结束后返回。
func (s *Server) Stop() error {
	if s.smtp == nil {
		return nil
	}
	err := s.smtp.Shutdown(context.Background())
	if s.st != nil {
		if err2 := s.st.stop(context.Background()); err2 != nil && err == nil {
			err = err2
		}
	}
	return err
}

func loadTLSConfig(cfg *config.Config) (*tls.Config, error) {
	cert, err := tls.LoadX509KeyPair(cfg.Server.TLS.Cert, cfg.Server.TLS.Key)
	if err != nil {
		return nil, fmt.Errorf("加载服务端证书失败：%w", err)
	}
	// === 提升 TLS 兼容性，支持更多加密套件 ===
	return &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS12,
		// === 允许更广泛的加密套件，提升客户端兼容性 ===
		CipherSuites: []uint16{
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_RSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
		},
		// === 不需要客户端证书验证 ===
		ClientAuth: tls.NoClientCert,
	}, nil
}

func newSMTPServer(cfg *config.Config, s *Server) *smtp.Server {
	srv := smtp.NewServer(smtp.BackendFunc(s.newSession))
	srv.Domain = cfg.Server.Hostname
	srv.ReadTimeout = cfg.Server.IOTimeout.Duration()
	srv.WriteTimeout = cfg.Server.IOTimeout.Duration()
	srv.MaxMessageBytes = cfg.Server.MaxMessageBytes
	srv.MaxRecipients = 500
	srv.AllowInsecureAuth = true // 监听本身已是 TLS；兼容个别不检查 TLS 的旧客户端
	srv.EnableSMTPUTF8 = true
	// === 增大最大行长度，支持超大 Base64 编码内容 ===
	srv.MaxLineLength = 10000 // 从默认的 2000 增大到 10000 字节
	return srv
}

// wrappedListener 负责 TLS 握手超时、IP 白名单与最大并发连接数控制。
type wrappedListener struct {
	net.Listener
	srv        *Server
	handshake  time.Duration
	active     atomic.Int64
	warnedOnce atomic.Bool
}

func (s *Server) wrapListener(ln net.Listener, handshake time.Duration) net.Listener {
	return &wrappedListener{Listener: ln, srv: s, handshake: handshake}
}

func (wl *wrappedListener) Accept() (net.Conn, error) {
	for {
		conn, err := wl.Listener.Accept()
		if err != nil {
			return nil, err
		}
		if accepted := wl.check(conn); accepted != nil {
			return accepted, nil
		}
		// 被拒绝的连接已处理完毕，继续接受下一个
	}
}

// check 完成握手超时、白名单、连接数检查；返回 nil 表示已拒绝并关闭连接。
func (wl *wrappedListener) check(conn net.Conn) net.Conn {
	cfg := wl.srv.provider.Get()
	remote := conn.RemoteAddr().String()

	// TLS 握手超时
	if wl.handshake > 0 {
		_ = conn.SetDeadline(time.Now().Add(wl.handshake))
	}
	tlsConn, ok := conn.(*tls.Conn)
	if !ok {
		conn.Close()
		return nil
	}
	if err := tlsConn.Handshake(); err != nil {
		// === 记录详细的 TLS 错误信息，便于调试 ===
		wl.srv.logger.Warn("TLS 握手失败", "remote", remote, "error", err)
		conn.Close()
		return nil
	}
	_ = conn.SetDeadline(time.Time{})

	// IP 白名单（防开放中继）
	wlObj, err := acl.NewWhitelist(cfg.Server.IPWhitelist)
	switch {
	case err != nil:
		if !wl.warnedOnce.Swap(true) {
			wl.srv.logger.Error("IP 白名单解析失败，已放行全部客户端", "error", err)
		}
	case !wlObj.Empty():
		ip, ok := acl.IPFromAddr(conn.RemoteAddr())
		if ok && !wlObj.Allows(ip) {
			wl.srv.logger.Warn("拒绝白名单外客户端", "remote", remote, "client_ip", ip.String())
			_, _ = conn.Write([]byte("554 5.7.1 Access denied: client IP not whitelisted\r\n"))
			conn.Close()
			return nil
		}
	}

	// 最大并发连接数
	if int(wl.active.Load()) >= cfg.Server.MaxConnections {
		wl.srv.logger.Warn("连接数已达上限，拒绝新连接", "remote", remote, "max", cfg.Server.MaxConnections)
		_, _ = conn.Write([]byte("421 4.7.0 Too many connections, try again later\r\n"))
		conn.Close()
		return nil
	}

	wl.active.Add(1)
	if wl.srv.metrics != nil {
		wl.srv.metrics.ActiveConnections.Inc()
	}
	return &countedConn{Conn: conn, wl: wl}
}

// countedConn 在连接关闭时回收计数。
type countedConn struct {
	net.Conn
	wl     *wrappedListener
	closed atomic.Bool
}

func (c *countedConn) Close() error {
	if c.closed.CompareAndSwap(false, true) {
		c.wl.active.Add(-1)
		if c.wl.srv.metrics != nil {
			c.wl.srv.metrics.ActiveConnections.Dec()
		}
	}
	return c.Conn.Close()
}
