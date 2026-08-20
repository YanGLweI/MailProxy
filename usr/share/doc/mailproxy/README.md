# MailProxy

Go 实现的 SMTP 邮件代理网关。统一对接后端邮箱服务商，业务侧只需连接代理服务器即可发信。

<div align="center">

## 解决什么问题

✓ **统一接入点** - 所有业务程序对接同一台 SMTP 服务器，无需感知后端邮箱服务商差异  
✓ **多账号路由** - 根据发件人自动选择最优后端账号，提升送达率  
✓ **零代码改造** - 只需修改 SMTP 地址配置，现有发邮件逻辑完全不用变  
✓ **安全合规** - IP 白名单 + 可选认证，防止被滥用为开放中继  
✓ **企业邮箱兼容** - 自动处理「信封发件人必须等于认证账号」限制（阿里/腾讯/网易等）

</div>

<br/>

<div align="center">

```
┌─────────────┐                    ┌──────────────────────────────────────┐
│ 业务程序     │                    │         MailProxy 代理               │
│ (Email App) │                    └──────────────────────────────────────┘
└──────┬──────┘                              ▲           │
       │                                      │           │
       │ SMTP over SSL (465)                 │           │
       │ EHLO → AUTH → MAIL FROM → DATA      │           │
       ├─────────────────────────────────────┤           │
       │                                     │           │
       │        ┌────────────────────────┐   │           │
       └───────►│  路由引擎              │   │           │
                │  • 信封发件人匹配      │◄──┤  配置    │
                │  • 认证账号映射        │   │  (YAML)  │
                │  • IP 白名单校验       │   │          │
                └──────────┬─────────────┘   │          │
                           │                  │          │
                           ▼                  │          │
                    ┌─────────────────────────┴───┐      │
                    │  后端转发队列               │      │
                    └─────────────────────────────┘      │
                           │                              │
                           │ SMTP/SMTPS (465/587)         │
                           │ AUTH PLAIN/LOGIN             │
                           ▼                              ▼
              ┌──────────────────────────┐    ┌──────────────────────────┐
              │ 阿里云企业邮箱            │    │ 其他 SMTP 服务端           │
              │ smtp.qiye.aliyun.com     │    │ QQ/网易/Gmail/自定义     │
              └──────────────────────────┘    └──────────────────────────┘
```

*图：MailProxy 作为中央枢纽，统一管理多个后端 SMTP 服务*

</div>

---

<br/>

## 功能特性

- 对外提供 **SMTP over SSL（465 端口）** 标准 SMTP 服务，兼容 `EHLO/HELO、AUTH、MAIL FROM、RCPT TO、DATA、QUIT`
- 可选 **STARTTLS 监听（587 端口，`server.starttls_listen`）**：兼容仅支持 STARTTLS、不支持隐式 TLS 的客户端（如 Veeam Backup & Replication），默认不启用，独立连接计数池、与 465 互不影响
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
# 1. 生成自签名证书（仅供快速跑通；生产建议替换为自有 CA/公共 CA 证书，见「SSL 证书」）
./deploy/gen-cert.sh

# 2. 准备配置
cp config.example.yaml config.yaml
chmod 600 config.yaml       # 配置内含授权码明文
vim config.yaml             # 填写后端邮箱账号等

# 3. 启动
./mailproxy -config config.yaml
```

**业务接入只需改 3 个参数：**

| 参数 | 修改前                    | 修改后                      |
|------|---------------------------|-----------------------------|
| SMTP 主机 | `smtp.qiye.aliyun.com`    | `mailproxy.internal:465`    |
| SSL 证书 | 逐个管理各服务商证书      | 只需信任 mailproxy 一个证书 |
| 开发成本 | 不同服务商 API 差异适配    | 统一 SMTP 协议，一次开发     |

### 连通性自测

**推荐先验证连接再发送测试邮件：**

```bash
# 验证 465 TLS 握手与 SMTP 交互（自签证书加 -startls 无关，直接用 s_client）
openssl s_client -connect 127.0.0.1:465 -quiet

# 验证 587 STARTTLS 监听（启用 starttls_listen 后）
openssl s_client -starttls smtp -connect 127.0.0.1:587 -quiet

# 经代理发送一封测试邮件
go run ./testtools/sendmail \
  -addr 127.0.0.1:10465 \
  -from sender@example.com -to someone@example.com
```

**预期输出示例：**

```bash
$ go run ./testtools/sendmail ...
✅ 连接成功：127.0.0.1:10465
✅ AUTH LOGIN 认证成功
✅ MAIL FROM:<sender@example.com>
✅ RCPT TO:<someone@example.com>
✅ DATA 已发送 (Message-ID: <abc123@mailproxy>)
✅ 状态：250 OK
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

后端账号、代理账号、路由规则、白名单、日志等变更即时生效；`server.listen`、`server.starttls_listen` 与 TLS 证书变更需要重启。热加载校验失败时沿用旧配置并记录错误日志。

## systemd 部署

```
install -m 755 mailproxy /usr/local/bin/mailproxy
install -d /etc/mailproxy
install -m 600 config.yaml /etc/mailproxy/config.yaml
install -m 644 deploy/mailproxy.service /etc/systemd/system/
systemctl daemon-reload
systemctl enable --now mailproxy
```

`deploy/mailproxy.service` 已通过 `AmbientCapabilities=CAP_NET_BIND_SERVICE` 解决非 root 绑定 465 特权端口的问题（请确保以 root 启动 unit，capability 会下发给服务用户）。

## RPM 打包部署（RHEL/CentOS/Rocky 系）

构建（macOS 本机即可，依赖 Go 与 Docker，无需 Linux 环境）：

```bash
bash deploy/build-rpm.sh        # 产物 dist/mailproxy-<version>-1.el*.x86_64.rpm
```

安装（目标服务器，root）：

```bash
rpm -ivh mailproxy-1.0.3-1.el9.x86_64.rpm
```

安装时自动完成：

- 创建系统用户/组 `mailproxy`（nologin）
- 自动生成自签名 TLS 证书到 `/etc/mailproxy/certs/`，供服务快速启动；**生产环境建议替换为自有 CA 或公共 CA 签发的证书**，见下方「SSL 证书」
- 安装配置到 `/etc/mailproxy/config.yaml`（权限 640 root:mailproxy，服务用户经组权限可读）、systemd unit 到 `/usr/lib/systemd/system/`
- 设置开机自启（`systemctl enable`），但**不启动**服务

安装完成后按终端提示操作：

```bash
vim /etc/mailproxy/config.yaml      # 填写后端邮箱账号/授权码、IP 白名单等
systemctl start mailproxy           # 手动启动
systemctl status mailproxy          # 查看状态
```

升级与卸载：

```bash
rpm -Uvh mailproxy-<new>.rpm        # 升级；已修改的 config.yaml 不被覆盖（新版落地为 config.yaml.rpmnew），证书不重置
rpm -e mailproxy                    # 卸载：停止并禁用服务、删除 mailproxy 用户；config.yaml 按 rpm 惯例保留
```

打包相关文件：`deploy/mailproxy.spec`、`deploy/build-rpm.sh`、`deploy/rpm/`（rpm 专用配置与 unit 模板）。

## SSL 证书

### 安装自动生成自签名证书

为开箱即用，**RPM 安装时会自动生成自签名证书**（手工部署则执行 `deploy/gen-cert.sh`），服务可立即启动。但自签名证书不在任何客户端的默认信任链中，每个接入的业务平台都要单独导入证书才能通过 TLS 校验。

**生产环境建议替换为自己企业 CA 或公共 CA 签发的证书**：

- 企业 CA：把根 CA 证书一次性导入各业务平台信任库后，后续证书续期、轮换业务侧零改动
- 公共 CA（如 Let's Encrypt、DigiCert 等）：绝大多数客户端默认信任，业务平台通常无需任何配置

### 替换步骤

1. 用 CA 签发服务器证书，**SAN 必须包含业务程序实际连接所用的主机名**（与 `server.hostname` 一致），否则客户端主机名校验会失败
2. 若由中间 CA 签发，把中间证书拼接到服务器证书后面（叶子证书在前）再写入 `server.crt`，MailProxy 会随握手下发完整证书链：

   ```bash
   cat your_server.crt intermediate.crt > server.crt
   ```

3. 覆盖证书文件并设置权限（服务以非 root 用户运行，须保证服务用户可读）：

   ```bash
   # RPM 安装环境
   cp server.crt /etc/mailproxy/certs/server.crt
   cp server.key /etc/mailproxy/certs/server.key
   chmod 644 /etc/mailproxy/certs/server.crt
   chmod 600 /etc/mailproxy/certs/server.key
   chown root:mailproxy /etc/mailproxy/certs/server.*
   systemctl restart mailproxy      # 证书变更需重启，kill -HUP 热加载不生效
   ```

### 第三方业务平台信任证书

业务平台通过 465（SSL）连接代理时会校验服务端证书。使用自签名证书时，常见报错：Java 程序报 `PKIX path building failed` / `unable to find valid certification path`，其他平台报「证书不受信任」「SSL 握手失败」等。按优先级选择处理方式：

1. **（推荐）换用业务平台已信任的 CA 签发的证书**——企业内统一分发了根 CA 的用自有 CA，否则用公共 CA 证书，业务平台零配置或仅导入一次根证书
2. **把证书导入业务平台的信任库**——自签名证书导入 `server.crt` 本身；CA 签发证书导入其**根 CA 证书**（而非服务器证书，这样换证书后无需再动业务侧）。Java 平台示例（如 EventLog Analyzer 等自带 JRE 的产品，导入其 JRE 的 cacerts）：

   ```bash
   keytool -importcert -alias mailproxy -file server.crt \
     -keystore $APP_HOME/jre/lib/security/cacerts -storepass changeit
   ```

3. **关闭业务平台的证书校验**——仅作为临时手段，存在中间人风险，生产不建议

注意：证书更新/轮换后，凡是按方式 2 单独导入过证书的平台都需要重新导入，这也是建议改用 CA 证书的主要原因。

## 业务程序接入

只需把业务程序的 SMTP 服务器改为本代理：

- SMTP 地址：代理服务器 IP
- 端口：465（SSL）
- 认证：`auth.mode: auth` 时填配置中的代理账号/密码；`none` 时任意填写或留空（代理接受并忽略任意 AUTH）
- SSL 证书：自签名证书需业务平台单独导入信任；建议直接使用自有 CA/公共 CA 证书，详见「SSL 证书」章节

**仅支持 STARTTLS 的客户端**（如 Veeam Backup & Replication：官方不支持 465 隐式 TLS，勾选 "Connect using SSL" 仍走 STARTTLS 握手，直连 465 会超时失败）：启用 `server.starttls_listen: ":587"` 并重启后，按如下配置接入：

- 端口：587，客户端勾选 SSL/STARTTLS 选项（Veeam 即 "Connect using SSL"）
- 防火墙放行 587
- 465 隐式 TLS 与 587 STARTTLS 双监听共存，其他业务程序零影响

## 设计约束（基础版）

- 不做邮件存储：只中转，转发完成即释放报文
- 不做队列与失败重试：后端故障透传错误，由业务程序自行重试（二期规划）
- 无 Web 管理界面：仅文件配置

### 二期迭代规划

邮件队列与本地暂存重试、单客户端/单账号限流、TLS 证书自动更新、HTTP 健康检查、黑名单过滤、Web 配置界面。

## 风险与注意事项

1. **465 特权端口**：Linux 下需 root 或 `cap_net_bind_service` 能力（systemd unit 已处理）
2. **SSL 证书**：安装自动生成的自签名证书需各业务平台逐一导入信任，否则 SSL 握手失败；生产建议替换为自有 CA 或公共 CA 签发的证书（见「SSL 证书」）
3. **服务商限制**：第三方邮箱的发送频率/额度限制无法绕过，超限错误会透传给业务程序
4. **密码安全**：配置文件保存授权码明文（预留加密存储扩展点），务必限制文件权限（手工部署 `chmod 600`；rpm 安装为 `640 root:mailproxy`，服务用户需经组权限读取）

## 测试

```bash
go test ./...
```

包含配置加载/校验、路由解析、IP 白名单单测，以及基于内存 mock SMTP 后端的全链路集成测试（TLS 监听、免鉴权转发、强制 AUTH 忽略、账号映射、LOGIN 认证、信封发件人改写、MAIL FROM 路由拒绝、白名单拒绝、后端故障透传、STARTTLS 升级前后鉴权控制、STARTTLS 白名单拒绝、双监听共存回归、默认不启用回归）。
