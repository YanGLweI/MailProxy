// Package relay 负责与后端真实 SMTP 服务器建连并转发完整邮件报文。
// 基础版不做队列与重试，后端错误原样透传给调用方。
package relay

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/smtp"
	"net/textproto"
	"strconv"
	"strings"
	"time"

	"mailproxy/internal/config"
)

// BackendRef 转发时使用的一组后端 SMTP 配置。
type BackendRef struct {
	ID       string
	Name     string
	Host     string
	Port     int
	Security string
	Username string
	Password string
	// RewriteFrom 为 true 时由调用方把信封 MAIL FROM 改写为 Username。
	RewriteFrom bool
}

// RefFromBackend 从配置结构构造转发引用。
func RefFromBackend(b config.Backend) BackendRef {
	return BackendRef{
		ID:          b.ID,
		Name:        b.Name,
		Host:        b.Host,
		Port:        b.Port,
		Security:    b.Security,
		Username:    b.Username,
		Password:    b.Password,
		RewriteFrom: b.RewriteFrom,
	}
}

// Message 一封待转发的邮件（信封 + 原始报文）。
type Message struct {
	From string
	To   []string
	Data []byte
}

// BackendError 后端 SMTP 返回的错误，携带 SMTP 应答码供透传给客户端。
type BackendError struct {
	Code int // 0 表示无明确应答码（按连接类临时错误处理）
	Msg  string
}

func (e *BackendError) Error() string {
	if e.Code > 0 {
		return fmt.Sprintf("backend smtp %d: %s", e.Code, e.Msg)
	}
	return "backend smtp: " + e.Msg
}

// Send 建立到后端的连接并发送一封邮件，成功后 QUIT 释放连接。
func Send(ctx context.Context, b BackendRef, msg Message) error {
	c, err := dial(ctx, b)
	if err != nil {
		return err
	}
	defer c.Close()

	if err := c.Hello("mailproxy"); err != nil {
		return &BackendError{Code: codeOf(err), Msg: "EHLO: " + errMsg(err)}
	}
	if b.Security == config.SecurityStartTLS {
		if err := c.StartTLS(&tls.Config{ServerName: b.Host}); err != nil {
			return &BackendError{Code: codeOf(err), Msg: "STARTTLS: " + errMsg(err)}
		}
	}
	if b.Username != "" {
		auth := smtp.PlainAuth("", b.Username, b.Password, b.Host)
		if err := c.Auth(auth); err != nil {
			return &BackendError{Code: codeOf(err), Msg: "AUTH: " + errMsg(err)}
		}
	}
	if err := c.Mail(msg.From); err != nil {
		return &BackendError{Code: codeOf(err), Msg: "MAIL FROM: " + errMsg(err)}
	}
	for _, to := range msg.To {
		if err := c.Rcpt(to); err != nil {
			return &BackendError{Code: codeOf(err), Msg: "RCPT TO <" + to + ">: " + errMsg(err)}
		}
	}
	w, err := c.Data()
	if err != nil {
		return &BackendError{Code: codeOf(err), Msg: "DATA: " + errMsg(err)}
	}
	if _, err := w.Write(msg.Data); err != nil {
		return &BackendError{Msg: "write message: " + errMsg(err)}
	}
	if err := w.Close(); err != nil {
		return &BackendError{Code: codeOf(err), Msg: errMsg(err)}
	}
	_ = c.Quit()
	return nil
}

// CheckBackend 对后端做一次连通性检测：建连 + EHLO + AUTH + QUIT。
func CheckBackend(ctx context.Context, b BackendRef) error {
	c, err := dial(ctx, b)
	if err != nil {
		return err
	}
	defer c.Close()

	if err := c.Hello("mailproxy"); err != nil {
		return fmt.Errorf("EHLO: %w", err)
	}
	if b.Security == config.SecurityStartTLS {
		if err := c.StartTLS(&tls.Config{ServerName: b.Host}); err != nil {
			return fmt.Errorf("STARTTLS: %w", err)
		}
	}
	if b.Username != "" {
		if err := c.Auth(smtp.PlainAuth("", b.Username, b.Password, b.Host)); err != nil {
			return fmt.Errorf("AUTH: %w", err)
		}
	}
	return c.Quit()
}

// dial 按 security 模式建立后端连接并包装为 smtp.Client。
func dial(ctx context.Context, b BackendRef) (*smtp.Client, error) {
	addr := net.JoinHostPort(b.Host, strconv.Itoa(b.Port))
	d := net.Dialer{}
	var conn net.Conn
	var err error

	switch b.Security {
	case config.SecuritySSL:
		conn, err = tls.DialWithDialer(&d, "tcp", addr, &tls.Config{ServerName: b.Host})
	case config.SecurityStartTLS, config.SecurityNone:
		conn, err = d.DialContext(ctx, "tcp", addr)
	default:
		return nil, &BackendError{Msg: "unknown security mode: " + b.Security}
	}
	if err != nil {
		return nil, &BackendError{Msg: "connect " + addr + ": " + errMsg(err)}
	}

	deadline, ok := ctx.Deadline()
	if !ok {
		deadline = time.Now().Add(2 * time.Minute)
	}
	_ = conn.SetDeadline(deadline)

	c, err := smtp.NewClient(conn, b.Host)
	if err != nil {
		conn.Close()
		return nil, &BackendError{Msg: "smtp handshake with " + addr + ": " + errMsg(err)}
	}
	return c, nil
}

// codeOf 从 net/smtp / textproto 错误中提取 3 位 SMTP 应答码。
func codeOf(err error) int {
	var te *textproto.Error
	if errors.As(err, &te) {
		return te.Code
	}
	return 0
}

// errMsg 提取错误的可读文本，去掉多行应答中的换行。
func errMsg(err error) string {
	if err == nil {
		return ""
	}
	s := strings.TrimSpace(err.Error())
	return strings.ReplaceAll(s, "\n", " ")
}
