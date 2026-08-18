package server

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"io"
	"log/slog"
	"math/big"
	"net"
	"net/smtp"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/emersion/go-sasl"
	gosmtp "github.com/emersion/go-smtp"

	"mailproxy/internal/config"
)

// ---------------- mock 后端 SMTP 服务器 ----------------

type mockMail struct {
	From string
	To   []string
	Data string
}

type mockBackend struct {
	mu    sync.Mutex
	mails []mockMail
}

func (b *mockBackend) NewSession(*gosmtp.Conn) (gosmtp.Session, error) {
	return &mockSession{b: b}, nil
}

func (b *mockBackend) recorded() []mockMail {
	b.mu.Lock()
	defer b.mu.Unlock()
	return append([]mockMail(nil), b.mails...)
}

type mockSession struct {
	b     *mockBackend
	from  string
	rcpts []string
}

func (s *mockSession) Reset()        { s.from = ""; s.rcpts = nil }
func (s *mockSession) Logout() error { return nil }
func (s *mockSession) Mail(from string, _ *gosmtp.MailOptions) error {
	s.from = from
	s.rcpts = nil
	return nil
}
func (s *mockSession) Rcpt(to string, _ *gosmtp.RcptOptions) error {
	s.rcpts = append(s.rcpts, to)
	return nil
}
func (s *mockSession) Data(r io.Reader) error {
	data, err := io.ReadAll(r)
	if err != nil {
		return err
	}
	s.b.mu.Lock()
	s.b.mails = append(s.b.mails, mockMail{From: s.from, To: s.rcpts, Data: string(data)})
	s.b.mu.Unlock()
	return nil
}

// AuthMechanisms/Auth 使 mock 后端支持 PLAIN 登录，模拟真实 SMTP 服务商。
func (s *mockSession) AuthMechanisms() []string { return []string{"PLAIN"} }

func (s *mockSession) Auth(mech string) (sasl.Server, error) {
	if mech != "PLAIN" {
		return nil, gosmtp.ErrAuthUnknownMechanism
	}
	return sasl.NewPlainServer(func(identity, username, password string) error {
		return nil // mock 后端接受任意凭据
	}), nil
}

func startMockBackend(t *testing.T) (*mockBackend, string) {
	t.Helper()
	b := &mockBackend{}
	srv := gosmtp.NewServer(gosmtp.BackendFunc(b.NewSession))
	srv.Domain = "mock.backend"
	srv.AllowInsecureAuth = true
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	go srv.Serve(ln)
	t.Cleanup(func() { srv.Close() })
	return b, ln.Addr().String()
}

// ---------------- 代理测试环境 ----------------

func genSelfSignedCert(t *testing.T) (certPath, keyPath string, certPEM []byte) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	tmpl := x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "127.0.0.1"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(24 * time.Hour),
		IPAddresses:           []net.IP{net.ParseIP("127.0.0.1")},
		DNSNames:              []string{"localhost"},
		BasicConstraintsValid: true,
		IsCA:                  true,
	}
	der, err := x509.CreateCertificate(rand.Reader, &tmpl, &tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	certPath = filepath.Join(dir, "server.crt")
	keyPath = filepath.Join(dir, "server.key")
	certPEM = pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	keyDER, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		t.Fatal(err)
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})
	if err := os.WriteFile(certPath, certPEM, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(keyPath, keyPEM, 0o600); err != nil {
		t.Fatal(err)
	}
	return certPath, keyPath, certPEM
}

// startProxy 启动一个监听随机端口的代理实例。
func startProxy(t *testing.T, yamlCfg string) (addr string, stop func()) {
	t.Helper()
	certPath, keyPath, certPEM := genSelfSignedCert(t)
	testCertPEM = certPEM
	yamlCfg = strings.ReplaceAll(yamlCfg, "{{CERT}}", certPath)
	yamlCfg = strings.ReplaceAll(yamlCfg, "{{KEY}}", keyPath)

	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte(yamlCfg), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := config.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := cfg.Validate(); err != nil {
		t.Fatal(err)
	}

	// 预留一个随机端口作为监听地址
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr = ln.Addr().String()
	ln.Close()
	cfg.Server.Listen = addr

	provider := config.NewProvider(path, cfg)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	srv := New(provider, logger, nil)

	errCh := make(chan error, 1)
	go func() { errCh <- srv.Run(t.Context()) }()

	// 等待端口就绪
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		c, err := net.Dial("tcp", addr)
		if err == nil {
			c.Close()
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	return addr, func() {
		srv.Stop()
		select {
		case <-errCh:
		case <-time.After(5 * time.Second):
		}
	}
}

// testCertPEM 保存当前测试的自签证书，供客户端构造信任池。
var testCertPEM []byte

func proxyConfig(backendAddr, mode string) string {
	host, port, _ := net.SplitHostPort(backendAddr)
	return fmt.Sprintf(`
server:
  listen: "127.0.0.1:0"
  hostname: proxy.test
  tls: { cert: "{{CERT}}", key: "{{KEY}}" }
  io_timeout: 15s
auth:
  mode: %s
  default_backend: default
backends:
  - id: default
    name: mock
    host: %s
    port: %s
    security: none
    username: backend-user@example.com
    password: backend-pass
accounts:
  - { username: biz1, password: secret1, backend: default }
  - { username: biz2, password: secret2 }
routes:
  - { from: "routed@example.com", backend: default }
log: { level: info }
validate_on_start: false
`, mode, host, port)
}

// dialProxy 建立到代理的 TLS+SMTP 客户端连接，信任测试自签证书。
func dialProxy(t *testing.T, addr string) *smtp.Client {
	t.Helper()
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(testCertPEM) {
		t.Fatal("无法解析测试证书")
	}
	conn, err := tls.Dial("tcp", addr, &tls.Config{RootCAs: pool, ServerName: "127.0.0.1"})
	if err != nil {
		t.Fatal("TLS 连接失败:", err)
	}
	c, err := smtp.NewClient(conn, "127.0.0.1")
	if err != nil {
		t.Fatal("SMTP 握手失败:", err)
	}
	if err := c.Hello("client.test"); err != nil {
		t.Fatal("EHLO 失败:", err)
	}
	return c
}

func sendMail(t *testing.T, c *smtp.Client, from string, to []string, body string) error {
	t.Helper()
	if err := c.Mail(from); err != nil {
		return err
	}
	for _, rcpt := range to {
		if err := c.Rcpt(rcpt); err != nil {
			return err
		}
	}
	w, err := c.Data()
	if err != nil {
		return err
	}
	if _, err := w.Write([]byte(body)); err != nil {
		return err
	}
	return w.Close()
}

const testBody = "From: sender@example.com\r\n" +
	"To: rcpt@example.com\r\n" +
	"Subject: hello\r\n" +
	"Message-ID: <test-123@client.test>\r\n" +
	"\r\nhello proxy\r\n"

// ---------------- 测试用例 ----------------

// 模式A：免鉴权，固定后端转发，报文完整透传。
func TestNoneModeRelay(t *testing.T) {
	mock, backendAddr := startMockBackend(t)
	addr, stop := startProxy(t, proxyConfig(backendAddr, config.ModeNone))
	defer stop()

	c := dialProxy(t, addr)
	defer c.Close()

	if err := sendMail(t, c, "sender@example.com", []string{"rcpt@example.com"}, testBody); err != nil {
		t.Fatal("发送失败:", err)
	}

	mails := mock.recorded()
	if len(mails) != 1 {
		t.Fatalf("mock 后端应收到 1 封邮件, got %d", len(mails))
	}
	m := mails[0]
	if m.From != "sender@example.com" || len(m.To) != 1 || m.To[0] != "rcpt@example.com" {
		t.Errorf("信封不一致: %+v", m)
	}
	if !strings.Contains(m.Data, "hello proxy") || !strings.Contains(m.Data, "Message-ID: <test-123@client.test>") {
		t.Errorf("报文未完整透传: %q", m.Data)
	}
}

// 模式B：代理账号登录并映射后端；错误密码被拒绝。
func TestAuthModeAccountMapping(t *testing.T) {
	mock, backendAddr := startMockBackend(t)
	addr, stop := startProxy(t, proxyConfig(backendAddr, config.ModeAuth))
	defer stop()

	// 错误密码
	c := dialProxy(t, addr)
	if err := c.Auth(smtp.PlainAuth("", "biz1", "wrong", "127.0.0.1")); err == nil {
		t.Error("错误密码应认证失败")
	}
	c.Close()

	// 正确密码 + 发送
	c = dialProxy(t, addr)
	defer c.Close()
	if err := c.Auth(smtp.PlainAuth("", "biz1", "secret1", "127.0.0.1")); err != nil {
		t.Fatal("认证失败:", err)
	}
	if err := sendMail(t, c, "sender@example.com", []string{"rcpt@example.com"}, testBody); err != nil {
		t.Fatal("发送失败:", err)
	}
	if len(mock.recorded()) != 1 {
		t.Fatal("mock 后端应收到 1 封邮件")
	}
}

// 模式B：账号未绑定后端时按 MAIL FROM 路由；不匹配返回 550。
func TestAuthModeFromRouting(t *testing.T) {
	mock, backendAddr := startMockBackend(t)
	addr, stop := startProxy(t, proxyConfig(backendAddr, config.ModeAuth))
	defer stop()

	c := dialProxy(t, addr)
	defer c.Close()
	if err := c.Auth(smtp.PlainAuth("", "biz2", "secret2", "127.0.0.1")); err != nil {
		t.Fatal("认证失败:", err)
	}

	// 匹配 routes 的 from
	if err := sendMail(t, c, "routed@example.com", []string{"rcpt@example.com"}, testBody); err != nil {
		t.Fatal("路由匹配时应发送成功:", err)
	}
	if len(mock.recorded()) != 1 {
		t.Fatal("mock 后端应收到 1 封邮件")
	}

	// 不匹配任何路由：DATA 应返回 5xx
	err := sendMail(t, c, "unknown@example.com", []string{"rcpt@example.com"}, testBody)
	if err == nil {
		t.Fatal("路由不匹配时应拒绝发送")
	}
	if !strings.Contains(err.Error(), "550") {
		t.Errorf("期望 550 拒绝, got: %v", err)
	}
}

// IP 白名单外的客户端在 TLS 握手后被拒绝。
func TestWhitelistDeny(t *testing.T) {
	_, backendAddr := startMockBackend(t)
	yamlCfg := strings.Replace(proxyConfig(backendAddr, config.ModeNone),
		"  io_timeout: 15s",
		"  io_timeout: 15s\n  ip_whitelist: [\"192.0.2.0/24\"]", 1)
	addr, stop := startProxy(t, yamlCfg)
	defer stop()

	pool := x509.NewCertPool()
	pool.AppendCertsFromPEM(testCertPEM)
	conn, err := tls.Dial("tcp", addr, &tls.Config{RootCAs: pool, ServerName: "127.0.0.1"})
	if err != nil {
		t.Fatal("TLS 连接失败:", err)
	}
	defer conn.Close()
	// 白名单拒绝时代理先写 554 再关闭，smtp.NewClient 读 greeting 应报错
	if _, err := smtp.NewClient(conn, "127.0.0.1"); err == nil {
		t.Fatal("白名单外客户端应被拒绝")
	} else if !strings.Contains(err.Error(), "554") {
		t.Errorf("期望 554 拒绝应答, got: %v", err)
	}
}

// 后端故障时错误透传给客户端（临时错误 4xx），不静默丢弃。
func TestBackendDownPassThrough(t *testing.T) {
	// 指向一个不存在的后端端口
	yamlCfg := `
server:
  listen: "127.0.0.1:0"
  hostname: proxy.test
  tls: { cert: "{{CERT}}", key: "{{KEY}}" }
  io_timeout: 10s
auth: { mode: none, default_backend: default }
backends:
  - { id: default, name: dead, host: 127.0.0.1, port: 1, security: none, username: u, password: p }
validate_on_start: false
`
	addr, stop := startProxy(t, yamlCfg)
	defer stop()

	c := dialProxy(t, addr)
	defer c.Close()
	err := sendMail(t, c, "sender@example.com", []string{"rcpt@example.com"}, testBody)
	if err == nil {
		t.Fatal("后端不可达时应返回错误")
	}
	code := err.Error()
	if !strings.HasPrefix(code, "4") && !strings.Contains(code, "451") && !strings.Contains(code, "45") {
		t.Errorf("后端连接故障应返回 4xx 临时错误, got: %v", err)
	}
}
