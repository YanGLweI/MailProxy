package server

import (
	"context"
	"crypto/subtle"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/emersion/go-sasl"
	smtp "github.com/emersion/go-smtp"

	"mailproxy/internal/acl"
	"mailproxy/internal/config"
	"mailproxy/internal/relay"
)

// newSession 是 go-smtp 的 Backend 入口：每个客户端连接创建一个会话。
func (s *Server) newSession(c *smtp.Conn) (smtp.Session, error) {
	cfg := s.provider.Get()

	clientIP := ""
	if ip, ok := acl.IPFromAddr(c.Conn().RemoteAddr()); ok {
		clientIP = ip.String()
	}
	s.logger.Info("客户端已连接", "client_ip", clientIP, "remote", c.Conn().RemoteAddr().String())

	ses := &session{
		srv:      s,
		clientIP: clientIP,
		started:  time.Now(),
	}
	if cfg.Auth.Mode == config.ModeAuth {
		return &authSession{session: ses}, nil
	}
	return &noopAuthSession{session: ses}, nil
}

// session 实现 smtp.Session，处理 RSET/MAIL FROM/RCPT TO/DATA/QUIT。
// mode=none 时由 noopAuthSession 包装，接受并忽略任意 AUTH；
// mode=auth 时由 authSession 包装，做真实代理账号认证。
type session struct {
	srv      *Server
	clientIP string
	started  time.Time

	mu       sync.Mutex
	authUser string   // mode=auth 时登录的代理账号
	from     string   // 当前事务的发件人
	rcpts    []string // 当前事务的收件人
}

func (s *session) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.from = ""
	s.rcpts = nil
}

func (s *session) Logout() error {
	s.srv.logger.Debug("客户端断开", "client_ip", s.clientIP,
		"conn_duration", time.Since(s.started).Round(time.Millisecond).String())
	return nil
}

func (s *session) Mail(from string, _ *smtp.MailOptions) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.from = from
	s.rcpts = nil
	return nil
}

func (s *session) Rcpt(to string, _ *smtp.RcptOptions) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.from == "" {
		return &smtp.SMTPError{Code: 503, Message: "Error: need MAIL command"}
	}
	s.rcpts = append(s.rcpts, to)
	return nil
}

// Data 读取完整邮件报文并转发到后端，报文不落盘，转发完成即释放。
func (s *session) Data(r io.Reader) error {
	cfg := s.srv.provider.Get()

	s.mu.Lock()
	from, rcpts, authUser := s.from, s.rcpts, s.authUser
	s.mu.Unlock()

	if from == "" || len(rcpts) == 0 {
		return &smtp.SMTPError{Code: 503, Message: "Error: need MAIL and RCPT commands"}
	}

	buf, err := io.ReadAll(io.LimitReader(r, cfg.Server.MaxMessageBytes+1))
	if err != nil {
		return smtpErrf(451, "read message data: %v", err)
	}
	if int64(len(buf)) > cfg.Server.MaxMessageBytes {
		return &smtp.SMTPError{Code: 552, Message: "Message exceeds maximum size"}
	}

	backend, rerr := s.resolveBackend(cfg, from)
	if rerr != nil {
		s.logSend(backend.ID, authUser, from, rcpts, relay.MessageID(buf),
			time.Since(s.started), rerr)
		return rerr
	}

	msgID := relay.MessageID(buf)
	start := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), cfg.Server.IOTimeout.Duration())
	defer cancel()

	sendErr := relay.Send(ctx, backend, relay.Message{From: from, To: rcpts, Data: buf})
	s.logSend(backend.ID, authUser, from, rcpts, msgID, time.Since(start), sendErr)
	if sendErr != nil {
		if s.srv.metrics != nil {
			s.srv.metrics.SendTotal.WithLabelValues("fail").Inc()
			s.srv.metrics.BackendErrorTotal.WithLabelValues(backend.ID).Inc()
		}
		return toSMTPError(sendErr)
	}
	if s.srv.metrics != nil {
		s.srv.metrics.SendTotal.WithLabelValues("success").Inc()
	}
	return nil
}

// resolveBackend 按当前配置解析本次发送使用的后端。
// mode=none: 固定使用 default_backend；
// mode=auth: 登录账号绑定的 backend -> routes 按 MAIL FROM 精确匹配 -> 均不匹配则拒绝。
func (s *session) resolveBackend(cfg *config.Config, from string) (relay.BackendRef, *smtp.SMTPError) {
	if cfg.Auth.Mode == config.ModeNone {
		b, ok := cfg.BackendByID(cfg.Auth.DefaultBackend)
		if !ok {
			return relay.BackendRef{}, &smtp.SMTPError{Code: 451, Message: "Default backend not configured"}
		}
		return relay.RefFromBackend(b), nil
	}

	s.mu.Lock()
	authUser := s.authUser
	s.mu.Unlock()
	if authUser == "" {
		return relay.BackendRef{}, smtp.ErrAuthRequired
	}
	acct, ok := cfg.AccountByUsername(authUser)
	if !ok {
		return relay.BackendRef{}, &smtp.SMTPError{Code: 550, Message: "Proxy account no longer exists"}
	}
	if acct.BackendID != "" {
		b, ok := cfg.BackendByID(acct.BackendID)
		if !ok {
			return relay.BackendRef{}, &smtp.SMTPError{Code: 451, Message: "Mapped backend not configured"}
		}
		return relay.RefFromBackend(b), nil
	}
	if route, ok := cfg.RouteByFrom(from); ok {
		b, _ := cfg.BackendByID(route.BackendID)
		return relay.RefFromBackend(b), nil
	}
	return relay.BackendRef{}, &smtp.SMTPError{
		Code:    550,
		Message: "No backend route matched for sender <" + from + ">",
	}
}

// logSend 记录一次发信事件（成功或失败）。
func (s *session) logSend(backendID, authUser, from string,
	rcpts []string, msgID string, cost time.Duration, sendErr error) {

	attrs := []any{
		"client_ip", s.clientIP,
		"backend", backendID,
		"from", from,
		"to", strings.Join(rcpts, ","),
		"message_id", msgID,
		"cost", cost.Round(time.Millisecond).String(),
	}
	if authUser != "" {
		attrs = append(attrs, "proxy_account", authUser)
	}
	if sendErr != nil {
		attrs = append(attrs, "error", sendErr.Error())
		s.srv.logger.Error("邮件发送失败", attrs...)
		return
	}
	s.srv.logger.Info("邮件发送成功", attrs...)
}

// toSMTPError 把后端错误映射为返回给业务客户端的 SMTP 应答。
// 有明确应答码的透传原始码与文案；连接类故障按临时错误 451 返回，提示客户端重试。
func toSMTPError(err error) *smtp.SMTPError {
	if be, ok := err.(*relay.BackendError); ok {
		if be.Code >= 400 {
			return &smtp.SMTPError{Code: be.Code, Message: be.Msg}
		}
	}
	return &smtp.SMTPError{Code: 451, Message: "Backend unavailable: " + errMsgOf(err)}
}

func smtpErrf(code int, format string, args ...any) *smtp.SMTPError {
	return &smtp.SMTPError{Code: code, Message: fmt.Sprintf(format, args...)}
}

func errMsgOf(err error) string {
	if err == nil {
		return ""
	}
	return strings.ReplaceAll(strings.TrimSpace(err.Error()), "\n", " ")
}

// authSession 在 mode=auth 时为会话附加 SMTP AUTH（PLAIN）能力。
type authSession struct {
	*session
}

var _ smtp.AuthSession = (*authSession)(nil)

func (a *authSession) AuthMechanisms() []string {
	return []string{"PLAIN"}
}

func (a *authSession) Auth(mech string) (sasl.Server, error) {
	if mech != "PLAIN" {
		return nil, smtp.ErrAuthUnknownMechanism
	}
	return sasl.NewPlainServer(func(identity, username, password string) error {
		cfg := a.srv.provider.Get()
		acct, ok := cfg.AccountByUsername(username)
		if !ok || subtle.ConstantTimeCompare([]byte(password), []byte(acct.Password)) != 1 {
			return smtp.ErrAuthFailed
		}
		a.mu.Lock()
		a.authUser = username
		a.mu.Unlock()
		a.srv.logger.Info("代理账号登录成功", "proxy_account", username, "client_ip", a.clientIP)
		return nil
	}), nil
}

// noopAuthSession 在 mode=none 时为会话附加“接受并忽略”的 AUTH 能力：
// 对外宣告 PLAIN/LOGIN 并接受任意凭据，兼容不协商 EHLO 能力、强制发起 AUTH 的客户端。
// 凭据不校验、不写入 authUser，转发仍固定走 default_backend。
type noopAuthSession struct {
	*session
}

var _ smtp.AuthSession = (*noopAuthSession)(nil)

func (n *noopAuthSession) AuthMechanisms() []string {
	return []string{"PLAIN", "LOGIN"}
}

func (n *noopAuthSession) Auth(mech string) (sasl.Server, error) {
	accept := func(username string) {
		n.srv.logger.Debug("mode=none 忽略客户端 AUTH",
			"mechanism", mech, "username", username, "client_ip", n.clientIP)
	}
	switch mech {
	case "PLAIN":
		return sasl.NewPlainServer(func(identity, username, password string) error {
			accept(username)
			return nil
		}), nil
	case "LOGIN":
		return &loginServer{accept: accept}, nil
	}
	return nil, smtp.ErrAuthUnknownMechanism
}

// loginServer 是 SASL LOGIN 机制的最小服务端实现（go-sasl 仅提供客户端），
// 配合 noopAuthSession 接受任意凭据。
type loginServer struct {
	step   int
	user   string
	accept func(username string)
}

func (l *loginServer) Next(response []byte) (challenge []byte, done bool, err error) {
	switch l.step {
	case 0: // AUTH LOGIN，initial response 可能直接携带用户名
		if response == nil {
			l.step = 1
			return []byte("Username:"), false, nil
		}
		l.user = string(response)
		l.step = 2
		return []byte("Password:"), false, nil
	case 1: // 客户端发送用户名
		l.user = string(response)
		l.step = 2
		return []byte("Password:"), false, nil
	case 2: // 客户端发送密码，认证结束
		l.step = 3
		l.accept(l.user)
		return nil, true, nil
	default:
		return nil, true, sasl.ErrUnexpectedClientResponse
	}
}
