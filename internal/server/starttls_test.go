package server

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/smtp"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"mailproxy/internal/config"
)

// ---------------- STARTTLS 双监听测试环境（不改动 server_test.go 现有 helper） ----------------

// dualConfig 双监听配置模板，{{STARTTLS}} 占位符会被替换为预留的 STARTTLS 端口。
func dualConfig(backendAddr, mode string) string {
	host, port, _ := net.SplitHostPort(backendAddr)
	return fmt.Sprintf(`
server:
  listen: "127.0.0.1:0"
  starttls_listen: "{{STARTTLS}}"
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
log: { level: info }
validate_on_start: false
`, mode, host, port)
}

// startDualProxy 启动代理实例，返回隐式 TLS 地址与预留的 STARTTLS 地址
// （模板不含 {{STARTTLS}} 占位符时 STARTTLS 不启用，stAddr 仅作为「应未监听」的探测地址）。
func startDualProxy(t *testing.T, yamlCfg string) (implicitAddr, stAddr string, stop func()) {
	t.Helper()
	certPath, keyPath, certPEM := genSelfSignedCert(t)
	testCertPEM = certPEM
	yamlCfg = strings.ReplaceAll(yamlCfg, "{{CERT}}", certPath)
	yamlCfg = strings.ReplaceAll(yamlCfg, "{{KEY}}", keyPath)

	// 预留一个随机端口作为 STARTTLS 监听地址
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	stAddr = ln.Addr().String()
	ln.Close()
	yamlCfg = strings.ReplaceAll(yamlCfg, "{{STARTTLS}}", stAddr)

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

	// 预留一个随机端口作为隐式 TLS 监听地址
	ln2, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	implicitAddr = ln2.Addr().String()
	ln2.Close()
	cfg.Server.Listen = implicitAddr

	provider := config.NewProvider(path, cfg)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	srv := New(provider, logger, nil)

	errCh := make(chan error, 1)
	go func() { errCh <- srv.Run(t.Context()) }()

	waitReady(t, implicitAddr)
	if cfg.Server.StartTLSListen != "" {
		waitReady(t, stAddr)
	}
	return implicitAddr, stAddr, func() {
		srv.Stop()
		select {
		case <-errCh:
		case <-time.After(5 * time.Second):
		}
	}
}

func waitReady(t *testing.T, addr string) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		c, err := net.Dial("tcp", addr)
		if err == nil {
			c.Close()
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("端口限时未就绪:", addr)
}

// dialStartTLS 明文 TCP 拨号 STARTTLS 端口并完成 EHLO（尚未升级）。
func dialStartTLS(t *testing.T, addr string) *smtp.Client {
	t.Helper()
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatal("连接 STARTTLS 端口失败:", err)
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

// upgradeStartTLS 执行 STARTTLS 升级，信任测试自签证书。
func upgradeStartTLS(t *testing.T, c *smtp.Client) {
	t.Helper()
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(testCertPEM) {
		t.Fatal("无法解析测试证书")
	}
	if err := c.StartTLS(&tls.Config{RootCAs: pool, ServerName: "127.0.0.1"}); err != nil {
		t.Fatal("STARTTLS 升级失败:", err)
	}
}

// ---------------- 测试用例 ----------------

// STARTTLS 端口：升级前不通告 AUTH，升级后可正常转发且报文完整。
func TestStartTLSRelay(t *testing.T) {
	mock, backendAddr := startMockBackend(t)
	_, stAddr, stop := startDualProxy(t, dualConfig(backendAddr, config.ModeNone))
	defer stop()

	c := dialStartTLS(t, stAddr)
	defer c.Close()

	// 升级前不得通告 AUTH（防止凭据明文传输）
	if ok, _ := c.Extension("AUTH"); ok {
		t.Error("STARTTLS 升级前不应通告 AUTH")
	}
	upgradeStartTLS(t, c)
	if ok, _ := c.Extension("AUTH"); !ok {
		t.Error("STARTTLS 升级后应通告 AUTH")
	}

	if err := sendMail(t, c, "sender@example.com", []string{"rcpt@example.com"}, testBody); err != nil {
		t.Fatal("发送失败:", err)
	}
	mails := mock.recorded()
	if len(mails) != 1 {
		t.Fatalf("mock 后端应收到 1 封邮件, got %d", len(mails))
	}
	if !strings.Contains(mails[0].Data, "hello proxy") {
		t.Errorf("报文未完整透传: %q", mails[0].Data)
	}
}

// STARTTLS 端口：mode=auth 下升级前 AUTH 不可用，升级后代理账号登录发信成功。
func TestStartTLSAuthMode(t *testing.T) {
	mock, backendAddr := startMockBackend(t)
	_, stAddr, stop := startDualProxy(t, dualConfig(backendAddr, config.ModeAuth))
	defer stop()

	c := dialStartTLS(t, stAddr)
	if ok, _ := c.Extension("AUTH"); ok {
		t.Fatal("升级前不应通告 AUTH")
	}
	if err := c.Auth(smtp.PlainAuth("", "biz1", "secret1", "127.0.0.1")); err == nil {
		t.Fatal("STARTTLS 升级前 AUTH 应失败")
	}
	c.Close() // net/smtp 在 AUTH 失败后发 QUIT 并关闭连接，重连走升级流程

	c = dialStartTLS(t, stAddr)
	defer c.Close()
	upgradeStartTLS(t, c)
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

// STARTTLS 端口：白名单外客户端被 554 拒绝。
func TestStartTLSWhitelist(t *testing.T) {
	_, backendAddr := startMockBackend(t)
	yamlCfg := strings.Replace(dualConfig(backendAddr, config.ModeNone),
		"  io_timeout: 15s",
		"  io_timeout: 15s\n  ip_whitelist: [\"192.0.2.0/24\"]", 1)
	_, stAddr, stop := startDualProxy(t, yamlCfg)
	defer stop()

	conn, err := net.Dial("tcp", stAddr)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	// 白名单拒绝时代理先写 554 再关闭，smtp.NewClient 读 greeting 应报错
	if _, err := smtp.NewClient(conn, "127.0.0.1"); err == nil {
		t.Fatal("白名单外客户端应被拒绝")
	} else if !strings.Contains(err.Error(), "554") {
		t.Errorf("期望 554 拒绝应答, got: %v", err)
	}
}

// 关键回归：启用 STARTTLS 后，隐式 TLS 与 STARTTLS 两个监听共存互不影响。
func TestBothListenersWork(t *testing.T) {
	mock, backendAddr := startMockBackend(t)
	implicitAddr, stAddr, stop := startDualProxy(t, dualConfig(backendAddr, config.ModeNone))
	defer stop()

	// 隐式 TLS（465 形态）
	c1 := dialProxy(t, implicitAddr)
	if err := sendMail(t, c1, "implicit@example.com", []string{"rcpt@example.com"}, testBody); err != nil {
		t.Fatal("隐式 TLS 发送失败:", err)
	}
	c1.Close()

	// STARTTLS（587 形态）
	c2 := dialStartTLS(t, stAddr)
	upgradeStartTLS(t, c2)
	if err := sendMail(t, c2, "starttls@example.com", []string{"rcpt@example.com"}, testBody); err != nil {
		t.Fatal("STARTTLS 发送失败:", err)
	}
	c2.Close()

	if n := len(mock.recorded()); n != 2 {
		t.Fatalf("mock 后端应收到 2 封邮件, got %d", n)
	}
}

// starttls_listen 留空时不监听 STARTTLS 端口（默认关闭，向后兼容）。
func TestStartTLSDisabledByDefault(t *testing.T) {
	_, backendAddr := startMockBackend(t)
	_, stAddr, stop := startDualProxy(t, proxyConfig(backendAddr, config.ModeNone))
	defer stop()

	if _, err := net.Dial("tcp", stAddr); err == nil {
		t.Fatal("starttls_listen 留空时 STARTTLS 端口不应监听")
	}
}
