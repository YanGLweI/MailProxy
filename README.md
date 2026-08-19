# MailProxy

Go 实现的 SMTP 邮件代理网关。内部业务程序不直接对接真实邮箱 SMTP 服务器，统一连接本代理；代理接收业务侧 SMTP 请求后，转发到后端真实 SMTP 服务器完成发送，支持多套邮箱账号配置与路由策略。

业务程序视角：把本代理当成一台 SMTP 服务器，填写代理服务 IP + 465（SSL），使用标准 SMTP 协议发信，无需感知后端真实邮箱服务商信息，也无需修改原有发邮件逻辑。

## 功能特性

- 对外提供 **SMTP over SSL（465 端口）** 标准 SMTP 服务，兼容 `EHLO/HELO、AUTH、MAIL FROM、RCPT TO、DATA、QUIT`
- 多组后端邮箱账号配置（host / port / ssl / starttls / 账号 / 授权码 / 备注名）
- 两种鉴权模式（`auth.mode` 配置切换）：
  - `none`：免鉴权，可信内网直接发信，所有邮件使用固定后端配置转发；客户端若强制发起 AUTH，接受任意凭据并忽略（兼容不协商能力的第三方平台）
  - `auth`：代理侧账号登录（支持 AUTH PLAIN/LOGIN），按账号映射后端配置；账号未绑定后端时按 `MAIL FROM` 路由规则匹配，不匹配则拒绝（550）
- 可选信封发件人改写（后端 `rewrite_from`）：对要求「信封发件人==认证账号」的后端（如企业邮箱），转发前把 MAIL FROM 改写为后端账号，报文头 From 不变
- 后端转发支持 SSL/TLS（465）与 STARTTLS（587），后端错误原样透传给业务客户端，不静默丢邮件
- 配置文件（YAML）管理，**SIGHUP 热加载**，无需重启（监听地址/证书变更除外）
- 启动时对每组后端做连通性 + 认证检测
- IP 白名单访问控制，防止被当作开放中继
- 并发连接数上限、TLS 握手超时、收发超时，避免僵死连接
- 结构化日志（控制台 + 文件，级别可配），每次发信记录来源 IP、账号、后端、收发件人、Message-ID、结果与耗时
- 可选 Prometheus 指标（`mailproxy_send_total`、`mailproxy_backend_error_total`、`mailproxy_active_connections`）
- 单二进制部署，systemd 托管，SIGTERM 优雅关闭（等待现有连接处理完成）

## 构建

```bash
go build -o mailproxy .
```

Linux 部署可直接在本机交叉编译：

```bash
GOOS=linux GOARCH=amd64 go build -o mailproxy .
```

## 快速开始

```bash
# 1. 生成自签名证书（内网使用；业务程序需信任 certs/server.crt）
./deploy/gen-cert.sh

# 2. 准备配置
cp config.example.yaml config.yaml
chmod 600 config.yaml       # 配置内含授权码明文
vim config.yaml             # 填写后端邮箱账号等

# 3. 启动
./mailproxy -config config.yaml
```

启动参数：

| 参数 | 说明 |
|---|---|
| `-config` | 配置文件路径，默认 `config.yaml` |
| `-log` | 日志文件路径，覆盖配置中的 `log.file` |

配置字段含义见 [config.example.yaml](config.example.yaml) 中的逐项注释。

### 配置热加载

```bash
kill -HUP $(pidof mailproxy)
# 或 systemd: systemctl reload mailproxy
```

后端账号、代理账号、路由规则、白名单、日志等变更即时生效；`server.listen` 与 TLS 证书变更需要重启。热加载校验失败时沿用旧配置并记录错误日志。

### 连通性自测

```bash
# 验证 465 TLS 握手与 SMTP 交互（自签证书加 -starttls 无关，直接用 s_client）
openssl s_client -connect 127.0.0.1:465 -quiet

# 经代理发送一封测试邮件
go run ./testtools/sendmail \
  -addr 127.0.0.1:10465 \
  -from sender@example.com -to someone@example.com
```

## systemd 部署

```bash
install -m 755 mailproxy /usr/local/bin/mailproxy
install -d /etc/mailproxy
install -m 600 config.yaml /etc/mailproxy/config.yaml
install -m 644 deploy/mailproxy.service /etc/systemd/system/
systemctl daemon-reload
systemctl enable --now mailproxy
```

`deploy/mailproxy.service` 已通过 `AmbientCapabilities=CAP_NET_BIND_SERVICE` 解决非 root 绑定 465 特权端口的问题（请确保以 root 启动 unit，capability 会下发给服务用户）。

## 业务程序接入

只需把业务程序的 SMTP 服务器改为本代理：

- SMTP 地址：代理服务器 IP
- 端口：465（SSL）
- 认证：`auth.mode: auth` 时填配置中的代理账号/密码；`none` 时任意填写或留空（代理接受并忽略任意 AUTH）
- 自签证书：业务程序需信任代理证书，或按既有方式处理证书校验

## 设计约束（基础版）

- 不做邮件存储：只中转，转发完成即释放报文
- 不做队列与失败重试：后端故障透传错误，由业务程序自行重试（二期规划）
- 无 Web 管理界面：仅文件配置

### 二期迭代规划

邮件队列与本地暂存重试、单客户端/单账号限流、TLS 证书自动更新、HTTP 健康检查、黑名单过滤、Web 配置界面。

## 风险与注意事项

1. **465 特权端口**：Linux 下需 root 或 `cap_net_bind_service` 能力（systemd unit 已处理）
2. **SSL 证书**：内网自签名证书需业务程序信任，否则 SSL 握手失败
3. **服务商限制**：第三方邮箱的发送频率/额度限制无法绕过，超限错误会透传给业务程序
4. **密码安全**：配置文件保存授权码明文（预留加密存储扩展点），务必限制文件权限为 600

## 测试

```bash
go test ./...
```

包含配置加载/校验、路由解析、IP 白名单单测，以及基于内存 mock SMTP 后端的全链路集成测试（TLS 监听、免鉴权转发、强制 AUTH 忽略、账号映射、LOGIN 认证、信封发件人改写、MAIL FROM 路由拒绝、白名单拒绝、后端故障透传）。
