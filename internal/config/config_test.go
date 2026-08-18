package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

const validYAML = `
server:
  listen: ":10465"
  hostname: test.local
  tls: { cert: c.crt, key: c.key }
  max_connections: 10
  handshake_timeout: 5s
  io_timeout: 10s
  ip_whitelist: ["127.0.0.1"]
auth:
  mode: none
  default_backend: default
backends:
  - id: default
    name: mock
    host: 127.0.0.1
    port: 2525
    security: none
    username: u@example.com
    password: p
log: { level: info }
validate_on_start: false
`

func writeTemp(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestLoadAndValidate(t *testing.T) {
	cfg, err := Load(writeTemp(t, validYAML))
	if err != nil {
		t.Fatal(err)
	}
	if err := cfg.Validate(); err != nil {
		t.Fatal(err)
	}
	if cfg.Server.IOTimeout.Duration() != 10*time.Second {
		t.Errorf("io_timeout 解析错误: %v", cfg.Server.IOTimeout.Duration())
	}
	if cfg.Server.MaxMessageBytes != 50<<20 {
		t.Errorf("默认 max_message_bytes 应为 50MB, got %d", cfg.Server.MaxMessageBytes)
	}
	if _, ok := cfg.BackendByID("default"); !ok {
		t.Error("BackendByID 未找到 default")
	}
}

func TestValidateErrors(t *testing.T) {
	cases := map[string]string{
		"重复backend id": `
server: { tls: { cert: a, key: b } }
auth: { mode: none, default_backend: x }
backends:
  - { id: x, host: h, port: 1, security: ssl, username: u, password: p }
  - { id: x, host: h, port: 1, security: ssl, username: u, password: p }
`,
		"default_backend不存在": `
server: { tls: { cert: a, key: b } }
auth: { mode: none, default_backend: missing }
backends:
  - { id: x, host: h, port: 1, security: ssl, username: u, password: p }
`,
		"非法security": `
server: { tls: { cert: a, key: b } }
auth: { mode: none, default_backend: x }
backends:
  - { id: x, host: h, port: 1, security: weird, username: u, password: p }
`,
		"账号引用不存在backend": `
server: { tls: { cert: a, key: b } }
auth: { mode: auth }
backends:
  - { id: x, host: h, port: 1, security: ssl, username: u, password: p }
accounts:
  - { username: a1, password: p1, backend: missing }
`,
		"auth模式无账号": `
server: { tls: { cert: a, key: b } }
auth: { mode: auth }
backends:
  - { id: x, host: h, port: 1, security: ssl, username: u, password: p }
`,
		"缺少证书配置": `
auth: { mode: none, default_backend: x }
backends:
  - { id: x, host: h, port: 1, security: ssl, username: u, password: p }
`,
	}
	for name, y := range cases {
		cfg, err := Load(writeTemp(t, y))
		if err != nil {
			t.Fatalf("%s: 加载失败: %v", name, err)
		}
		if err := cfg.Validate(); err == nil {
			t.Errorf("%s: 期望校验失败", name)
		}
	}
}

func TestAuthModeRoutingRefs(t *testing.T) {
	y := `
server: { tls: { cert: a, key: b } }
auth: { mode: auth }
backends:
  - { id: x, host: h, port: 1, security: ssl, username: u, password: p }
accounts:
  - { username: a1, password: p1 }
  - { username: a2, password: p2, backend: x }
routes:
  - { from: "noreply@e.com", backend: x }
`
	cfg, err := Load(writeTemp(t, y))
	if err != nil {
		t.Fatal(err)
	}
	if err := cfg.Validate(); err != nil {
		t.Fatal(err)
	}
	acct, ok := cfg.AccountByUsername("a2")
	if !ok || acct.BackendID != "x" {
		t.Errorf("AccountByUsername 结果错误: %+v", acct)
	}
	route, ok := cfg.RouteByFrom("noreply@e.com")
	if !ok || route.BackendID != "x" {
		t.Errorf("RouteByFrom 结果错误: %+v", route)
	}
	if _, ok := cfg.RouteByFrom("other@e.com"); ok {
		t.Error("未配置的 from 不应匹配")
	}
}

func TestProviderReload(t *testing.T) {
	path := writeTemp(t, validYAML)
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	p := NewProvider(path, cfg)

	// 热加载修改后的配置
	updated := validYAML + `
routes: []
`
	if err := os.WriteFile(path, []byte(updated), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := p.Reload(); err != nil {
		t.Fatal(err)
	}
	if p.Get().Server.MaxConnections != 10 {
		t.Error("热加载后配置未生效")
	}

	// 非法新配置：保留旧配置
	if err := os.WriteFile(path, []byte("backends: []"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := p.Reload(); err == nil {
		t.Fatal("期望非法配置热加载失败")
	}
	if p.Get().Server.MaxConnections != 10 {
		t.Error("热加载失败后应沿用旧配置")
	}
}
