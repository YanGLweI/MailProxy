// Package config 定义 MailProxy 的配置结构、YAML 加载、校验与原子热加载。
package config

import (
	"fmt"
	"os"
	"sync/atomic"
	"time"

	"gopkg.in/yaml.v3"
)

// 鉴权模式。
const (
	// ModeNone: 模式A，免鉴权，所有邮件使用 DefaultBackend 转发。
	ModeNone = "none"
	// ModeAuth: 模式B，代理侧登录认证，按账号/MAIL FROM 路由到后端配置。
	ModeAuth = "auth"
)

// 后端连接安全模式。
const (
	SecuritySSL      = "ssl"      // 隐式 TLS（如 465）
	SecurityStartTLS = "starttls" // 明文连接后 STARTTLS 升级
	SecurityNone     = "none"     // 明文（仅限测试）
)

// Duration 支持 YAML 直接写 "30s" 这类时间字符串。
type Duration time.Duration

// UnmarshalYAML 解析时间字符串（如 30s、2m）。
func (d *Duration) UnmarshalYAML(value *yaml.Node) error {
	var s string
	if err := value.Decode(&s); err != nil {
		return err
	}
	dur, err := time.ParseDuration(s)
	if err != nil {
		return fmt.Errorf("invalid duration %q: %w", s, err)
	}
	*d = Duration(dur)
	return nil
}

func (d Duration) Duration() time.Duration { return time.Duration(d) }

type Config struct {
	Server          ServerConfig `yaml:"server"`
	Auth            AuthConfig   `yaml:"auth"`
	Backends        []Backend    `yaml:"backends"`
	Accounts        []Account    `yaml:"accounts"`
	Routes          []Route      `yaml:"routes"`
	Log             LogConfig    `yaml:"log"`
	Metrics         Metrics      `yaml:"metrics"`
	ValidateOnStart bool         `yaml:"validate_on_start"`
}

type ServerConfig struct {
	Listen           string   `yaml:"listen"` // 对外监听地址，如 ":465"
	Hostname         string   `yaml:"hostname"`
	TLS              TLSFiles `yaml:"tls"`
	MaxConnections   int      `yaml:"max_connections"`
	MaxMessageBytes  int64    `yaml:"max_message_bytes"`
	HandshakeTimeout Duration `yaml:"handshake_timeout"`
	IOTimeout        Duration `yaml:"io_timeout"`
	IPWhitelist      []string `yaml:"ip_whitelist"`
}

type TLSFiles struct {
	Cert string `yaml:"cert"`
	Key  string `yaml:"key"`
}

type AuthConfig struct {
	Mode           string `yaml:"mode"`            // none | auth
	DefaultBackend string `yaml:"default_backend"` // mode=none 时使用的后端
}

// Backend 一组后端真实 SMTP 账号配置。
type Backend struct {
	ID       string `yaml:"id"`   // 唯一标识，供账号/路由引用
	Name     string `yaml:"name"` // 备注名称
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	Security string `yaml:"security"` // ssl | starttls | none
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

// Account 代理侧登录账号（mode=auth），映射到一组后端配置。
// BackendID 为空时按 routes 的 MAIL FROM 规则路由。
type Account struct {
	Username  string `yaml:"username"`
	Password  string `yaml:"password"`
	BackendID string `yaml:"backend"`
}

// Route 按 MAIL FROM 发件人地址精确匹配后端配置。
type Route struct {
	From      string `yaml:"from"`
	BackendID string `yaml:"backend"`
}

type LogConfig struct {
	Level string `yaml:"level"` // info | warn | error
	File  string `yaml:"file"`  // 为空则仅控制台输出
}

type Metrics struct {
	Enabled bool   `yaml:"enabled"`
	Listen  string `yaml:"listen"`
}

// Load 读取并解析配置文件（不做引用完整性校验）。
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	cfg := &Config{}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	applyDefaults(cfg)
	return cfg, nil
}

func applyDefaults(cfg *Config) {
	if cfg.Server.Listen == "" {
		cfg.Server.Listen = ":465"
	}
	if cfg.Server.Hostname == "" {
		cfg.Server.Hostname = "mailproxy"
	}
	if cfg.Server.MaxConnections <= 0 {
		cfg.Server.MaxConnections = 100
	}
	if cfg.Server.MaxMessageBytes <= 0 {
		cfg.Server.MaxMessageBytes = 50 << 20 // 50MB
	}
	if cfg.Server.HandshakeTimeout <= 0 {
		cfg.Server.HandshakeTimeout = Duration(30 * time.Second)
	}
	if cfg.Server.IOTimeout <= 0 {
		cfg.Server.IOTimeout = Duration(120 * time.Second)
	}
	if cfg.Auth.Mode == "" {
		cfg.Auth.Mode = ModeNone
	}
	if cfg.Log.Level == "" {
		cfg.Log.Level = "info"
	}
}

// Validate 校验配置完整性与引用合法性。
func (c *Config) Validate() error {
	if c.Auth.Mode != ModeNone && c.Auth.Mode != ModeAuth {
		return fmt.Errorf("auth.mode 必须是 none 或 auth，当前为 %q", c.Auth.Mode)
	}
	if len(c.Backends) == 0 {
		return fmt.Errorf("backends 不能为空，至少配置一组后端 SMTP 账号")
	}
	ids := make(map[string]struct{}, len(c.Backends))
	for i, b := range c.Backends {
		if b.ID == "" {
			return fmt.Errorf("backends[%d] 缺少 id", i)
		}
		if _, dup := ids[b.ID]; dup {
			return fmt.Errorf("backend id 重复: %q", b.ID)
		}
		ids[b.ID] = struct{}{}
		if b.Host == "" || b.Port <= 0 {
			return fmt.Errorf("backend %q 的 host/port 配置不完整", b.ID)
		}
		switch b.Security {
		case SecuritySSL, SecurityStartTLS, SecurityNone:
		case "":
			return fmt.Errorf("backend %q 缺少 security 配置（ssl/starttls/none）", b.ID)
		default:
			return fmt.Errorf("backend %q 的 security 非法: %q", b.ID, b.Security)
		}
		if b.Username == "" || b.Password == "" {
			return fmt.Errorf("backend %q 的 username/password 配置不完整", b.ID)
		}
	}
	if c.Server.TLS.Cert == "" || c.Server.TLS.Key == "" {
		return fmt.Errorf("server.tls.cert 与 server.tls.key 必须配置")
	}
	if c.Auth.Mode == ModeNone {
		if c.Auth.DefaultBackend == "" {
			return fmt.Errorf("auth.mode=none 时必须配置 auth.default_backend")
		}
		if _, ok := ids[c.Auth.DefaultBackend]; !ok {
			return fmt.Errorf("auth.default_backend 引用的 backend %q 不存在", c.Auth.DefaultBackend)
		}
	}
	if c.Auth.Mode == ModeAuth && len(c.Accounts) == 0 {
		return fmt.Errorf("auth.mode=auth 时必须配置至少一个 accounts 代理账号")
	}
	users := make(map[string]struct{}, len(c.Accounts))
	for i, a := range c.Accounts {
		if a.Username == "" || a.Password == "" {
			return fmt.Errorf("accounts[%d] 的 username/password 配置不完整", i)
		}
		if _, dup := users[a.Username]; dup {
			return fmt.Errorf("代理账号重复: %q", a.Username)
		}
		users[a.Username] = struct{}{}
		if a.BackendID != "" {
			if _, ok := ids[a.BackendID]; !ok {
				return fmt.Errorf("account %q 引用的 backend %q 不存在", a.Username, a.BackendID)
			}
		}
	}
	for i, r := range c.Routes {
		if r.From == "" {
			return fmt.Errorf("routes[%d] 缺少 from", i)
		}
		if r.BackendID == "" {
			return fmt.Errorf("routes[%d] 缺少 backend", i)
		}
		if _, ok := ids[r.BackendID]; !ok {
			return fmt.Errorf("route from=%q 引用的 backend %q 不存在", r.From, r.BackendID)
		}
	}
	return nil
}

// BackendByID 按 id 查找后端配置。
func (c *Config) BackendByID(id string) (Backend, bool) {
	for _, b := range c.Backends {
		if b.ID == id {
			return b, true
		}
	}
	return Backend{}, false
}

// AccountByUsername 按用户名查找代理账号。
func (c *Config) AccountByUsername(username string) (Account, bool) {
	for _, a := range c.Accounts {
		if a.Username == username {
			return a, true
		}
	}
	return Account{}, false
}

// RouteByFrom 按 MAIL FROM 地址精确匹配路由规则。
func (c *Config) RouteByFrom(from string) (Route, bool) {
	for _, r := range c.Routes {
		if r.From == from {
			return r, true
		}
	}
	return Route{}, false
}

// WarnIfInsecureFilePerms 检测含明文密码的配置文件权限，过宽时输出告警日志。
func (c *Config) WarnIfInsecureFilePerms(path string, warnf func(format string, args ...any)) {
	info, err := os.Stat(path)
	if err != nil {
		return
	}
	perm := info.Mode().Perm()
	if perm&0o077 != 0 {
		warnf("配置文件 %s 权限为 %v，其他用户可读，建议执行 chmod 600 %s（配置内含邮箱授权码明文）",
			path, perm, path)
	}
}

// Provider 持有配置的原子快照，支持 SIGHUP 热加载。
type Provider struct {
	path string
	cur  atomic.Pointer[Config]
}

func NewProvider(path string, initial *Config) *Provider {
	p := &Provider{path: path}
	p.cur.Store(initial)
	return p
}

// Get 返回当前配置快照。
func (p *Provider) Get() *Config { return p.cur.Load() }

// Reload 重新读取并校验配置文件；校验失败时保留旧配置并返回错误。
func (p *Provider) Reload() (*Config, error) {
	cfg, err := Load(p.path)
	if err != nil {
		return p.Get(), err
	}
	if err := cfg.Validate(); err != nil {
		return p.Get(), err
	}
	p.cur.Store(cfg)
	return cfg, nil
}
