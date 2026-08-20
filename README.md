# MailProxy

Go 实现的 SMTP 邮件代理网关。统一对接后端邮箱服务商，业务侧只需连接代理服务器即可发信。

[![Release](https://img.shields.io/github/release/YanGLweI/MailProxy.svg?style=flat-square)](https://github.com/YanGLweI/MailProxy/releases)
[![License](https://img.shields.io/badge/license-MIT-blue.svg?style=flat-square)](LICENSE)
[![Go Version](https://img.shields.io/badge/go-1.21-blue.svg?style=flat-square)](https://go.dev/)
[![RPM](https://img.shields.io/badge/RPM-available-brightgreen.svg?style=flat-square)](https://github.com/YanGLweI/MailProxy/releases)

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

- 🚀 **高性能** - 基于 Go 并发模型，支持高并发连接
- 🔒 **安全认证** - IP 白名单 + 可选 AUTH 认证，防止被滥用为开放中继
- 🔄 **智能路由** - 根据发件人自动选择最优后端账号，提升送达率
- 🎯 **零代码改造** - 只需修改 SMTP 地址配置，现有发邮件逻辑完全不用变
- 🔐 **TLS 加密** - 支持 SMTP over SSL (465) 和 STARTTLS (587)，确保传输安全
- ♻️ **热重载** - 配置文件 SIGHUP 热加载，无需重启服务
- 📊 **监控指标** - Prometheus 指标导出，实时监控系统状态
- 📝 **详细日志** - 结构化日志记录，每次发信全流程追踪
- 🎁 **企业邮箱兼容** - 自动处理「信封发件人必须等于认证账号」限制（阿里/腾讯/网易等）
- 🏗️ **易部署** - 单二进制文件 + systemd 托管，提供 RPM 包一键安装

---

<br/>

## 快速开始

```bash
# 1. 下载最新版本 (Linux AMD64)
curl -LO https://github.com/YanGLweI/MailProxy/releases/latest/download/mailproxy-1.0.4-linux-amd64.tar.gz
tar -xzf mailproxy-*.tar.gz
chmod +x mailproxy

# 或使用 RPM 包 (RHEL/CentOS/Rocky)
curl -LO https://github.com/YanGLweI/MailProxy/releases/latest/download/mailproxy-1.0.4-1.el9.x86_64.rpm
sudo rpm -ivh mailproxy-1.0.4-1.el9.x86_64.rpm
```

**立即体验：**

```bash
# 生成自签名证书
./deploy/gen-cert.sh

# 准备配置
cp config.example.yaml config.yaml
vim config.yaml             # 填写后端邮箱账号等

# 启动服务
./mailproxy -config config.yaml
```

**业务接入只需改 3 个参数：**

**业务接入只需改 3 个参数：**

| 参数 | 修改前                    | 修改后                      |
|------|---------------------------|-----------------------------|
| SMTP 主机 | `smtp.qiye.aliyun.com`    | `mailproxy.internal:465`    |
| SSL 证书 | 逐个管理各服务商证书      | 只需信任 mailproxy 一个证书 |  
| 开发成本 | 不同服务商 API 差异适配    | 统一 SMTP 协议，一次开发     |

### 构建源码

如需从源码构建：

```bash
# 本地构建
go build -o mailproxy .

# 交叉编译 Linux AMD64
GOOS=linux GOARCH=amd64 go build -o mailproxy .

# 运行测试
go test ./...
```

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

### 🚀 快速构建 (macOS 本机即可)

```bash
bash deploy/build-rpm.sh        # 产物 dist/mailproxy-<version>-1.el*.x86_64.rpm
```

**前置要求:**
- ✅ Go 1.25+ `go version` → go1.25.0 darwin/arm64  
- ✅ Docker CLI (colima 或 Docker Desktop) `command -v docker`
- ✅ colima 已运行 `limactl status` → running
- ✅ Rocky Linux 9 amd64 镜像已拉取 `docker images | grep rockylinux`

> 💡 **说明**: 如果 `build-rpm.sh` 执行失败，请先确保所有前置要求都满足。

---

### ⚠️ 重要注意事项

#### 问题 1: Docker Hub 认证超时
**现象**: `failed to authorize: failed to fetch anonymous token`  
**原因**: 国内网络访问 Docker Hub 受限  
**解决**: 配置镜像代理或使用其他镜像站

**方法 A: 使用 DaoCloud 镜像加速**
```bash
export REGISTRY_MIRROR=https://docker.m.daocloud.io
docker pull --platform linux/amd64 rockylinux/rockylinux:9
```

**方法 B: 提前在服务器拉取并导出导入**
```bash
# macOS 本地
docker save rockylinux/rockylinux:9 > rockylinux.tar

# 上传到服务器
scp rockylinux.tar server:/tmp/

# 服务器上加载
docker load < /tmp/rockylinux.tar
```

#### 问题 2: macOS ARM64 的卷挂载限制  
**现象**: 跨平台卷挂载导致文件不可见  
**原因**: Docker Desktop 在 arm64 host 上无法完全支持 x86_64 容器的卷挂载  

**解决方案**:  
1. 构建阶段使用正确的 `--platform linux/amd64`
2. 验证阶段改用 `rpm -qip` 检查元数据而非安装测试
3. 生产环境直接上传 RPM 到目标服务器安装

#### 问题 3: RockyLinux 基础镜像无 systemd  
**现象**: 测试容器无法启动 systemd  
**原因**: 官方镜像最小化设计  

**解决方案**: 使用完整测试 Dockerfile 或仅做元数据验证

---

### 📦 详细构建流程与每步操作指南

如需从头完整了解如何从 macOS 构建 x86_64 Linux RPM 包，请参考以下步骤:

#### 🔧 前置环境与依赖

| 组件 | 要求 | 检查命令 |
|---|---|---|
| macOS 芯片 | Apple Silicon (M1/M2/M3) 或 Intel | `uname -m` → arm64 / x86_64 |
| Go | ≥1.25.0 | `go version` |
| Homebrew (可选) | 用于安装 lima | `/usr/bin/ruby -e "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/master/install)"` (未装过才需要) |
| curl/tar/git | 系统自带 | `which curl tar git` |
| 网络 | 可访问 GitHub/Docker Hub(镜像站优先) | `ping github.com` |

> ⚠️ **重要提示**: 
> - ARM64 Mac 需要下载对应版本的 Docker CLI(见 Step 3)
> - 推荐使用 [colima](https://github.com/abiosoft/colima) 替代 Docker Desktop  
> - Docker Hub 直连可能超时，建议配置镜像代理

#### 🏗️ 阶段一：搭建临时容器环境（约 5 分钟）

**Step 1: 下载 colima 二进制 **(ARM64)

```
mkdir -p ~/bin && \
curl -sL -o ~/bin/colima \
    https://github.com/abiosoft/colima/releases/latest/download/colima-Darwin-arm64 && \
chmod +x ~/bin/colima && \
~/bin/colima version
```

**输出示例:**
```
colima version v0.10.3
git commit: ...
```

**Step 2: 安装 limactl **(虚拟化层)

```
brew install lima
limactl --version
```

**输出示例:**
```
limactl version 2.2.0
```

**Step 3: 下载 Docker CLI 静态二进制**

```
mkdir -p ~/docker && \
curl -sL -o /tmp/docker.tgz \
    https://download.docker.com/mac/static/stable/aarch64/docker-28.3.3.tgz && \
tar -xzf /tmp/docker.tgz -C /tmp && \
cp /tmp/docker/docker ~/docker/ && \
~/docker/docker version
```

**输出示例:**
```
Client: Docker Engine - Community
 Version:           27.5.1-rc1
...
```

**替代方案：使用 Homebrew 安装 **(如果已配置)
```
# 方法 A: 检查 colima 自带的 docker 命令
colima status
# 如果显示 running，则 ~/.local/bin/docker 应该可用

# 方法 B: 导出 docker socket
export DOCKER_HOST=unix://$HOME/.colima/default/docker.sock
docker info
```

**Step 4: 启动 colima 虚拟机并加载镜像**

```
export PATH="$HOME/docker:$PATH"   # 让 colima 找到 docker
colima start --cpu 2 --memory 4 --disk 20
```

等待输出:`READY. Run 'limactl shell colima' to open the shell.`

**验证 Docker 工作正常:**
```
limactl status colima
# 应该看到：running

docker images
# 此时可以没有任何镜像，后续会拉取

docker run --rm hello-world
# 测试运行一个小容器
```

**Step 5: 拉取构建所需镜像 **(指定 amd64 平台)

```
docker pull --platform linux/amd64 rockylinux/rockylinux:9
```

**提示**:如果直连超时，使用镜像代理:
```
docker pull --platform linux/amd64 docker.m.daocloud.io/rockylinux/rockylinux:9
```

**验证镜像已成功拉取**:
```
docker images | grep rockylinux
# 应该看到:
# rockylinux/rockylinux     9      <IMAGE-ID>   <DATE>   710MB
# docker.m.daocloud.io/rockylinux/rockylinux   9    <same-ID>
```

---

#### 📝 阶段二：准备源码与构建脚本

**Step 6: 打开项目目录**

```
cd /Users/yeung/Projects/MailProxy   # 替换为你的项目路径
cat VERSION                          # 确认版本号 (如 1.0.4)
```

**Step 7: 查看核心构建脚本**

```
cat deploy/build-rpm.sh
```

**关键流程**:

1. 读取 VERSION 文件中的版本号
2. `CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build ...` → 交叉编译 Linux x86_64 静态二进制
3. 将 spec/specs/source 复制到 `dist/rpmbuild/SOURCES/`
4. `docker run --platform linux/amd64 rockylinux:9 rpmbuild ...` → 在容器中执行 rpm 构建

**rpmbuild 目录结构详解**:
```
dist/rpmbuild/
├── SOURCES/          # 源代码文件 (binary + config + service)
│   ├── mailproxy     # Go 编译的 x86_64 二进制
│   ├── config.yaml   # 默认配置文件  
│   ├── mailproxy.service  # systemd unit 模板
│   └── README.md     # 产品说明文档
├── SPECS/            # RPM spec 文件定义包结构
│   └── mailproxy.spec
├── BUILD/            # 编译工作目录 (空目录)
├── BUILDROOT/        # 安装包根目录 (包含安装后的文件布局)
├── RPMS/x86_64/      # 最终生成的 RPM 包
│   └── mailproxy-1.0.4-1.el9.x86_64.rpm
└── SRPMS/            # 源码 RPM (如果有)
    └── mailproxy-1.0.4-1.el9.src.rpm
```

---

#### 🎯 阶段三：一键构建 RPM 包

**Step 8: 直接运行构建脚本**

```
export PATH="$HOME/docker:$PATH"    # 确保找到 docker
bash deploy/build-rpm.sh
```

**完整输出示例**(优化版):
```
==> 交叉编译 Linux x86_64 二进制 (v1.0.4)
==> 准备打包源文件
==> Docker 内执行 rpmbuild (rockylinux/rockylinux:9, linux/amd64)

Installed:
  rpm-build-4.16.1.3-40.el9.x86_64
  ... (更多依赖包)

Executing(%prep): /bin/sh -e /var/tmp/rpm-tmp.XXX
+ umask 022
+ cd /build/BUILD
+ rm -rf mailproxy-1.0.4
...

Executing(%install): /bin/sh -e /var/tmp/rpm-tmp.YYY
+ umask 022
+ '[' /build/BUILDROOT/mailproxy-1.0.4-1.el9.x86_64 '!=' / ']'
...
+ install -Dpm 0755 /build/SOURCES/mailproxy /build/BUILDROOT/mailproxy-1.0.4-1.el9.x86_64/usr/bin/mailproxy
+ install -Dpm 0644 /build/SOURCES/mailproxy.service /build/BUILDROOT/mailproxy-1.0.4-1.el9.x86_64/usr/lib/systemd/system/mailproxy.service
...

Wrote: /build/RPMS/x86_64/mailproxy-1.0.4-1.el9.x86_64.rpm

==> 构建完成：dist/mailproxy-1.0.4-1.el9.x86_64.rpm
    安装：rpm -ivh dist/mailproxy-1.0.4-1.el9.x86_64.rpm
    升级：rpm -Uvh dist/mailproxy-1.0.4-1.el9.x86_64.rpm
```

**成功标志**:
- ✅ 最后一行显示 "构建完成"
- ✅ 显示了正确的 rpm 文件名
- ✅ shell 返回码为 0 (`echo $?`)

**产物位置**:
- `dist/mailproxy-1.0.4-1.el9.x86_64.rpm` (主包)
- `dist/rpmbuild/RPMS/x86_64/mailproxy-1.0.4-1.el9.x86_64.rpm` (原始位置)
- `dist/rpmbuild/` 临时目录 (可删除)

**时间预期**: 
- 首次构建：约 2-3 分钟 (需要下载 rpmbuild 依赖)
- 重复构建：约 30-60 秒 (所有依赖已缓存)

**常见问题处理**:

| 错误信息 | 原因 | 解决方案 |
|---|---|---|
| `error: No such file or directory` | Docker CLI 不可见或 PATH 不对 | `export PATH="$HOME/docker:$PATH"` |
| `failed to resolve reference` | Docker Hub 网络问题 | 使用镜像代理，或提前用`docker save`导出 |
| `architecture mismatch` | 使用了 arm64 镜像而非 amd64 | 确保 docker run 带 `--platform linux/amd64` |
| `go.mod not found` | 不在项目根目录 | `cd` 到 MailProxy 项目根目录 |
| `container exited abnormally` | Docker 资源不足 | 增加 `colima start --memory 8 --disk 40` |

---

#### ✅ 阶段四：验证 RPM 包功能

> ⚠️ **重要提示**: 
> 在 macOS ARM64 上使用 Docker 进行 x86_64 容器测试存在跨平台卷挂载限制。
> 建议在生产服务器上测试实际安装流程。

**Step 9: 本地元数据验证 **(推荐)

```
# 检查基本元数据
rpm -qip dist/mailproxy-1.0.4-1.el9.x86_64.rpm

# 检查包内容列表
rpm -qplf dist/mailproxy-1.0.4-1.el9.x86_64.rpm

# 列出所有文件及其权限
ls -lh dist/mailproxy-1.0.4-1.el9.x86_64.rpm
```

**预期成功输出**:
```
Name        : mailproxy
Version     : 1.0.4
Release     : 1.el9
Architecture: x86_64
Install Date: (not installed)
Group       : Unspecified
Size        : 9461599
License     : Proprietary
Summary     : MailProxy SMTP relay gateway
Description :
Go 实现的 SMTP 邮件代理网关：业务程序统一连接本代理发信，
代理转发到后端真实 SMTP 服务器，支持多账号配置与路由策略。
```

✅ **通过标准**:
- Version 正确 (与 VERSION 文件一致)
- Architecture 是 x86_64
- Size 约 9MB (9,461,599 bytes)
- Description 包含正确信息
- 无错误信息

**Step 10: 高级验证 - 在测试容器中模拟安装**

由于跨平台限制，此步骤仅在相同架构下可靠运行。如必须测试，参考以下方法:

```
# 需要先构建完整测试环境
cd testtools
docker build -f Dockerfile.testenv -t mailproxy-test-complete . --platform linux/amd64

# 然后拷贝文件到容器内测试 (而不是卷挂载)
docker run -d --name mp-test-via-copy mailproxy-test-complete sleep infinity
docker cp dist/mailproxy-*.rpm mp-test-via-copy:/tmp/
docker exec mp-test-via-copy bash -c "rpm -ivh /tmp/mailproxy-*.rpm"
docker stop mp-test-via-copy
```

⚠️ **注意**: 这种方法比卷挂载更可靠，因为避免了跨平台文件系统转换。

```

*图：MailProxy 作为中央枢纽，统一管理多个后端 SMTP 服务*

```

---

<br/>

## 功能特性

- 🚀 **高性能** - 基于 Go 并发模型，支持高并发连接
- 🔒 **安全认证** - IP 白名单 + 可选 AUTH 认证，防止被滥用为开放中继
- 🔄 **智能路由** - 根据发件人自动选择最优后端账号，提升送达率
- 🎯 **零代码改造** - 只需修改 SMTP 地址配置，现有发邮件逻辑完全不用变
- 🔐 **TLS 加密** - 支持 SMTP over SSL (465) 和 STARTTLS (587)，确保传输安全
- ♻️ **热重载** - 配置文件 SIGHUP 热加载，无需重启服务
- 📊 **监控指标** - Prometheus 指标导出，实时监控系统状态
- 📝 **详细日志** - 结构化日志记录，每次发信全流程追踪
- 🎁 **企业邮箱兼容** - 自动处理「信封发件人必须等于认证账号」限制（阿里/腾讯/网易等）
- 🏗️ **易部署** - 单二进制文件 + systemd 托管，提供 RPM 包一键安装

---

<br/>

## 快速开始

```
# 1. 下载最新版本 (Linux AMD64)
curl -LO https://github.com/YanGLweI/MailProxy/releases/latest/download/mailproxy-1.0.4-linux-amd64.tar.gz
tar -xzf mailproxy-*.tar.gz
chmod +x mailproxy

# 或使用 RPM 包 (RHEL/CentOS/Rocky)
curl -LO https://github.com/YanGLweI/MailProxy/releases/latest/download/mailproxy-1.0.4-1.el9.x86_64.rpm
sudo rpm -ivh mailproxy-1.0.4-1.el9.x86_64.rpm
```

**立即体验：**

```
# 生成自签名证书
./deploy/gen-cert.sh

# 准备配置
cp config.example.yaml config.yaml
vim config.yaml             # 填写后端邮箱账号等

# 启动服务
./mailproxy -config config.yaml
```

**业务接入只需改 3 个参数：**

**业务接入只需改 3 个参数：**

| 参数 | 修改前                    | 修改后                      |
|------|---------------------------|-----------------------------|
| SMTP 主机 | `smtp.qiye.aliyun.com`    | `mailproxy.internal:465`    |
| SSL 证书 | 逐个管理各服务商证书      | 只需信任 mailproxy 一个证书 |  
| 开发成本 | 不同服务商 API 差异适配    | 统一 SMTP 协议，一次开发     |

### 构建源码

如需从源码构建：

```
# 本地构建
go build -o mailproxy .

# 交叉编译 Linux AMD64
GOOS=linux GOARCH=amd64 go build -o mailproxy .

# 运行测试
go test ./...
```

### 连通性自测

**推荐先验证连接再发送测试邮件：**

```
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

```
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

```
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

### 🚀 快速构建 (macOS 本机即可)

```
bash deploy/build-rpm.sh        # 产物 dist/mailproxy-<version>-1.el*.x86_64.rpm
```

**前置要求:**
- ✅ Go 1.25+ `go version` → go1.25.0 darwin/arm64  
- ✅ Docker CLI (colima 或 Docker Desktop) `command -v docker`
- ✅ colima 已运行 `limactl status` → running
- ✅ Rocky Linux 9 amd64 镜像已拉取 `docker images | grep rockylinux`

> 💡 **说明**: 如果 `build-rpm.sh` 执行失败，请先确保所有前置要求都满足。

---

### ⚠️ 重要注意事项

#### 问题 1: Docker Hub 认证超时
**现象**: `failed to authorize: failed to fetch anonymous token`  
**原因**: 国内网络访问 Docker Hub 受限  
**解决**: 配置镜像代理或使用其他镜像站

**方法 A: 使用 DaoCloud 镜像加速**
```
export REGISTRY_MIRROR=https://docker.m.daocloud.io
docker pull --platform linux/amd64 rockylinux/rockylinux:9
```

**方法 B: 提前在服务器拉取并导出导入**
```
# macOS 本地
docker save rockylinux/rockylinux:9 > rockylinux.tar

# 上传到服务器
scp rockylinux.tar server:/tmp/

# 服务器上加载
docker load < /tmp/rockylinux.tar
```

#### 问题 2: macOS ARM64 的卷挂载限制  
**现象**: 跨平台卷挂载导致文件不可见  
**原因**: Docker Desktop 在 arm64 host 上无法完全支持 x86_64 容器的卷挂载  

**解决方案**:  
1. 构建阶段使用正确的 `--platform linux/amd64`
2. 验证阶段改用 `rpm -qip` 检查元数据而非安装测试
3. 生产环境直接上传 RPM 到目标服务器安装

#### 问题 3: RockyLinux 基础镜像无 systemd  
**现象**: 测试容器无法启动 systemd  
**原因**: 官方镜像最小化设计  

**解决方案**: 使用完整测试 Dockerfile 或仅做元数据验证

---

### 📦 详细构建流程与每步操作指南

如需从头完整了解如何从 macOS 构建 x86_64 Linux RPM 包，请参考以下步骤:

#### 🔧 前置环境与依赖

| 组件 | 要求 | 检查命令 |
|---|---|---|
| macOS 芯片 | Apple Silicon (M1/M2/M3) 或 Intel | `uname -m` → arm64 / x86_64 |
| Go | ≥1.25.0 | `go version` |
| Homebrew (可选) | 用于安装 lima | `/usr/bin/ruby -e "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/master/install)"` (未装过才需要) |
| curl/tar/git | 系统自带 | `which curl tar git` |
| 网络 | 可访问 GitHub/Docker Hub(镜像站优先) | `ping github.com` |

> ⚠️ **重要提示**: 
> - ARM64 Mac 需要下载对应版本的 Docker CLI(见 Step 3)
> - 推荐使用 [colima](https://github.com/abiosoft/colima) 替代 Docker Desktop  
> - Docker Hub 直连可能超时，建议配置镜像代理

#### 🏗️ 阶段一：搭建临时容器环境（约 5 分钟）

**Step 1: 下载 colima 二进制 **(ARM64)

```
mkdir -p ~/bin && \
curl -sL -o ~/bin/colima \
    https://github.com/abiosoft/colima/releases/latest/download/colima-Darwin-arm64 && \
chmod +x ~/bin/colima && \
~/bin/colima version
```

**输出示例:**
```
colima version v0.10.3
git commit: ...
```

**Step 2: 安装 limactl **(虚拟化层)

```
brew install lima
limactl --version
```

**输出示例:**
```
limactl version 2.2.0
```

**Step 3: 下载 Docker CLI 静态二进制**

```
mkdir -p ~/docker && \
curl -sL -o /tmp/docker.tgz \
    https://download.docker.com/mac/static/stable/aarch64/docker-28.3.3.tgz && \
tar -xzf /tmp/docker.tgz -C /tmp && \
cp /tmp/docker/docker ~/docker/ && \
~/docker/docker version
```

**输出示例:**
```
Client: Docker Engine - Community
 Version:           27.5.1-rc1
...
```

**替代方案：使用 Homebrew 安装 **(如果已配置)
```
# 方法 A: 检查 colima 自带的 docker 命令
colima status
# 如果显示 running，则 ~/.local/bin/docker 应该可用

# 方法 B: 导出 docker socket
export DOCKER_HOST=unix://$HOME/.colima/default/docker.sock
docker info
```

**Step 4: 启动 colima 虚拟机并加载镜像**

```
export PATH="$HOME/docker:$PATH"   # 让 colima 找到 docker
colima start --cpu 2 --memory 4 --disk 20
```

等待输出:`READY. Run 'limactl shell colima' to open the shell.`

**验证 Docker 工作正常:**
```
limactl status colima
# 应该看到：running

docker images
# 此时可以没有任何镜像，后续会拉取

docker run --rm hello-world
# 测试运行一个小容器
```

**Step 5: 拉取构建所需镜像 **(指定 amd64 平台)

```
docker pull --platform linux/amd64 rockylinux/rockylinux:9
```

**提示**:如果直连超时，使用镜像代理:
```
docker pull --platform linux/amd64 docker.m.daocloud.io/rockylinux/rockylinux:9
```

**验证镜像已成功拉取**:
```
docker images | grep rockylinux
# 应该看到:
# rockylinux/rockylinux     9      <IMAGE-ID>   <DATE>   710MB
# docker.m.daocloud.io/rockylinux/rockylinux   9    <same-ID>
```

---

#### 📝 阶段二：准备源码与构建脚本

**Step 6: 打开项目目录**

```
cd /Users/yeung/Projects/MailProxy   # 替换为你的项目路径
cat VERSION                          # 确认版本号 (如 1.0.4)
```

**Step 7: 查看核心构建脚本**

```
cat deploy/build-rpm.sh
```

**关键流程**:

1. 读取 VERSION 文件中的版本号
2. `CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build ...` → 交叉编译 Linux x86_64 静态二进制
3. 将 spec/specs/source 复制到 `dist/rpmbuild/SOURCES/`
4. `docker run --platform linux/amd64 rockylinux:9 rpmbuild ...` → 在容器中执行 rpm 构建

**rpmbuild 目录结构详解**:
```
dist/rpmbuild/
├── SOURCES/          # 源代码文件 (binary + config + service)
│   ├── mailproxy     # Go 编译的 x86_64 二进制
│   ├── config.yaml   # 默认配置文件  
│   ├── mailproxy.service  # systemd unit 模板
│   └── README.md     # 产品说明文档
├── SPECS/            # RPM spec 文件定义包结构
│   └── mailproxy.spec
├── BUILD/            # 编译工作目录 (空目录)
├── BUILDROOT/        # 安装包根目录 (包含安装后的文件布局)
├── RPMS/x86_64/      # 最终生成的 RPM 包
│   └── mailproxy-1.0.4-1.el9.x86_64.rpm
└── SRPMS/            # 源码 RPM (如果有)
    └── mailproxy-1.0.4-1.el9.src.rpm
```

---

#### 🎯 阶段三：一键构建 RPM 包

**Step 8: 直接运行构建脚本**

```
export PATH="$HOME/docker:$PATH"    # 确保找到 docker
bash deploy/build-rpm.sh
```

**完整输出示例**(优化版):
```
==> 交叉编译 Linux x86_64 二进制 (v1.0.4)
==> 准备打包源文件
==> Docker 内执行 rpmbuild (rockylinux/rockylinux:9, linux/amd64)

Installed:
  rpm-build-4.16.1.3-40.el9.x86_64
  ... (更多依赖包)

Executing(%prep): /bin/sh -e /var/tmp/rpm-tmp.XXX
+ umask 022
+ cd /build/BUILD
+ rm -rf mailproxy-1.0.4
...

Executing(%install): /bin/sh -e /var/tmp/rpm-tmp.YYY
+ umask 022
+ '[' /build/BUILDROOT/mailproxy-1.0.4-1.el9.x86_64 '!=' / ']'
...
+ install -Dpm 0755 /build/SOURCES/mailproxy /build/BUILDROOT/mailproxy-1.0.4-1.el9.x86_64/usr/bin/mailproxy
+ install -Dpm 0644 /build/SOURCES/mailproxy.service /build/BUILDROOT/mailproxy-1.0.4-1.el9.x86_64/usr/lib/systemd/system/mailproxy.service
...

Wrote: /build/RPMS/x86_64/mailproxy-1.0.4-1.el9.x86_64.rpm

==> 构建完成：dist/mailproxy-1.0.4-1.el9.x86_64.rpm
    安装：rpm -ivh dist/mailproxy-1.0.4-1.el9.x86_64.rpm
    升级：rpm -Uvh dist/mailproxy-1.0.4-1.el9.x86_64.rpm
```

**成功标志**:
- ✅ 最后一行显示 "构建完成"
- ✅ 显示了正确的 rpm 文件名
- ✅ shell 返回码为 0 (`echo $?`)

**产物位置**:
- `dist/mailproxy-1.0.4-1.el9.x86_64.rpm` (主包)
- `dist/rpmbuild/RPMS/x86_64/mailproxy-1.0.4-1.el9.x86_64.rpm` (原始位置)
- `dist/rpmbuild/` 临时目录 (可删除)

**时间预期**: 
- 首次构建：约 2-3 分钟 (需要下载 rpmbuild 依赖)
- 重复构建：约 30-60 秒 (所有依赖已缓存)

**常见问题处理**:

| 错误信息 | 原因 | 解决方案 |
|---|---|---|
| `error: No such file or directory` | Docker CLI 不可见或 PATH 不对 | `export PATH="$HOME/docker:$PATH"` |
| `failed to resolve reference` | Docker Hub 网络问题 | 使用镜像代理，或提前用`docker save`导出 |
| `architecture mismatch` | 使用了 arm64 镜像而非 amd64 | 确保 docker run 带 `--platform linux/amd64` |
| `go.mod not found` | 不在项目根目录 | `cd` 到 MailProxy 项目根目录 |
| `container exited abnormally` | Docker 资源不足 | 增加 `colima start --memory 8 --disk 40` |

---

#### ✅ 阶段四：验证 RPM 包功能

> ⚠️ **重要提示**: 
> 在 macOS ARM64 上使用 Docker 进行 x86_64 容器测试存在跨平台卷挂载限制。
> 建议在生产服务器上测试实际安装流程。

**Step 9: 本地元数据验证 **(推荐)

```
# 检查基本元数据
rpm -qip dist/mailproxy-1.0.4-1.el9.x86_64.rpm

# 检查包内容列表
rpm -qplf dist/mailproxy-1.0.4-1.el9.x86_64.rpm

# 列出所有文件及其权限
ls -lh dist/mailproxy-1.0.4-1.el9.x86_64.rpm
```

**预期成功输出**:
```
Name        : mailproxy
Version     : 1.0.4
Release     : 1.el9
Architecture: x86_64
Install Date: (not installed)
Group       : Unspecified
Size        : 9461599
License     : Proprietary
Summary     : MailProxy SMTP relay gateway
Description :
Go 实现的 SMTP 邮件代理网关：业务程序统一连接本代理发信，
代理转发到后端真实 SMTP 服务器，支持多账号配置与路由策略。
```

✅ **通过标准**:
- Version 正确 (与 VERSION 文件一致)
- Architecture 是 x86_64
- Size 约 9MB (9,461,599 bytes)
- Description 包含正确信息
- 无错误信息

**Step 10: 高级验证 - 在测试容器中模拟安装**

由于跨平台限制，此步骤仅在相同架构下可靠运行。如必须测试，参考以下方法:

```
# 需要先构建完整测试环境
cd testtools
docker build -f Dockerfile.testenv -t mailproxy-test-complete . --platform linux/amd64

# 然后拷贝文件到容器内测试 (而不是卷挂载)
docker run -d --name mp-test-via-copy mailproxy-test-complete sleep infinity
docker cp dist/mailproxy-*.rpm mp-test-via-copy:/tmp/
docker exec mp-test-via-copy bash -c "rpm -ivh /tmp/mailproxy-*.rpm"
docker stop mp-test-via-copy
```

⚠️ **注意**: 这种方法比卷挂载更可靠，因为避免了跨平台文件系统转换。

```

*图：MailProxy 作为中央枢纽，统一管理多个后端 SMTP 服务*

```

---

<br/>

## 功能特性

- 🚀 **高性能** - 基于 Go 并发模型，支持高并发连接
- 🔒 **安全认证** - IP 白名单 + 可选 AUTH 认证，防止被滥用为开放中继
- 🔄 **智能路由** - 根据发件人自动选择最优后端账号，提升送达率
- 🎯 **零代码改造** - 只需修改 SMTP 地址配置，现有发邮件逻辑完全不用变
- 🔐 **TLS 加密** - 支持 SMTP over SSL (465) 和 STARTTLS (587)，确保传输安全
- ♻️ **热重载** - 配置文件 SIGHUP 热加载，无需重启服务
- 📊 **监控指标** - Prometheus 指标导出，实时监控系统状态
- 📝 **详细日志** - 结构化日志记录，每次发信全流程追踪
- 🎁 **企业邮箱兼容** - 自动处理「信封发件人必须等于认证账号」限制（阿里/腾讯/网易等）
- 🏗️ **易部署** - 单二进制文件 + systemd 托管，提供 RPM 包一键安装

---

<br/>

## 快速开始

```
# 1. 下载最新版本 (Linux AMD64)
curl -LO https://github.com/YanGLweI/MailProxy/releases/latest/download/mailproxy-1.0.4-linux-amd64.tar.gz
tar -xzf mailproxy-*.tar.gz
chmod +x mailproxy

# 或使用 RPM 包 (RHEL/CentOS/Rocky)
curl -LO https://github.com/YanGLweI/MailProxy/releases/latest/download/mailproxy-1.0.4-1.el9.x86_64.rpm
sudo rpm -ivh mailproxy-1.0.4-1.el9.x86_64.rpm
```

**立即体验：**

```
# 生成自签名证书
./deploy/gen-cert.sh

# 准备配置
cp config.example.yaml config.yaml
vim config.yaml             # 填写后端邮箱账号等

# 启动服务
./mailproxy -config config.yaml
```

**业务接入只需改 3 个参数：**

**业务接入只需改 3 个参数：**

| 参数 | 修改前                    | 修改后                      |
|------|---------------------------|-----------------------------|
| SMTP 主机 | `smtp.qiye.aliyun.com`    | `mailproxy.internal:465`    |
| SSL 证书 | 逐个管理各服务商证书      | 只需信任 mailproxy 一个证书 |  
| 开发成本 | 不同服务商 API 差异适配    | 统一 SMTP 协议，一次开发     |

### 构建源码

如需从源码构建：

```
# 本地构建
go build -o mailproxy .

# 交叉编译 Linux AMD64
GOOS=linux GOARCH=amd64 go build -o mailproxy .

# 运行测试
go test ./...
```

### 连通性自测

**推荐先验证连接再发送测试邮件：**

```
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

```
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

```
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

### 🚀 快速构建 (macOS 本机即可)

```
bash deploy/build-rpm.sh        # 产物 dist/mailproxy-<version>-1.el*.x86_64.rpm
```

**前置要求:**
- ✅ Go 1.25+ `go version` → go1.25.0 darwin/arm64  
- ✅ Docker CLI (colima 或 Docker Desktop) `command -v docker`
- ✅ colima 已运行 `limactl status` → running
- ✅ Rocky Linux 9 amd64 镜像已拉取 `docker images | grep rockylinux`

> 💡 **说明**: 如果 `build-rpm.sh` 执行失败，请先确保所有前置要求都满足。

---

### ⚠️ 重要注意事项

#### 问题 1: Docker Hub 认证超时
**现象**: `failed to authorize: failed to fetch anonymous token`  
**原因**: 国内网络访问 Docker Hub 受限  
**解决**: 配置镜像代理或使用其他镜像站

**方法 A: 使用 DaoCloud 镜像加速**
```
export REGISTRY_MIRROR=https://docker.m.daocloud.io
docker pull --platform linux/amd64 rockylinux/rockylinux:9
```

**方法 B: 提前在服务器拉取并导出导入**
```
# macOS 本地
docker save rockylinux/rockylinux:9 > rockylinux.tar

# 上传到服务器
scp rockylinux.tar server:/tmp/

# 服务器上加载
docker load < /tmp/rockylinux.tar
```

#### 问题 2: macOS ARM64 的卷挂载限制  
**现象**: 跨平台卷挂载导致文件不可见  
**原因**: Docker Desktop 在 arm64 host 上无法完全支持 x86_64 容器的卷挂载  

**解决方案**:  
1. 构建阶段使用正确的 `--platform linux/amd64`
2. 验证阶段改用 `rpm -qip` 检查元数据而非安装测试
3. 生产环境直接上传 RPM 到目标服务器安装

#### 问题 3: RockyLinux 基础镜像无 systemd  
**现象**: 测试容器无法启动 systemd  
**原因**: 官方镜像最小化设计  

**解决方案**: 使用完整测试 Dockerfile 或仅做元数据验证

---

### 📦 详细构建流程与每步操作指南

如需从头完整了解如何从 macOS 构建 x86_64 Linux RPM 包，请参考以下步骤:

#### 🔧 前置环境与依赖

| 组件 | 要求 | 检查命令 |
|---|---|---|
| macOS 芯片 | Apple Silicon (M1/M2/M3) 或 Intel | `uname -m` → arm64 / x86_64 |
| Go | ≥1.25.0 | `go version` |
| Homebrew (可选) | 用于安装 lima | `/usr/bin/ruby -e "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/master/install)"` (未装过才需要) |
| curl/tar/git | 系统自带 | `which curl tar git` |
| 网络 | 可访问 GitHub/Docker Hub(镜像站优先) | `ping github.com` |

> ⚠️ **重要提示**: 
> - ARM64 Mac 需要下载对应版本的 Docker CLI(见 Step 3)
> - 推荐使用 [colima](https://github.com/abiosoft/colima) 替代 Docker Desktop  
> - Docker Hub 直连可能超时，建议配置镜像代理

#### 🏗️ 阶段一：搭建临时容器环境（约 5 分钟）

**Step 1: 下载 colima 二进制 **(ARM64)

```
mkdir -p ~/bin && \
curl -sL -o ~/bin/colima \
    https://github.com/abiosoft/colima/releases/latest/download/colima-Darwin-arm64 && \
chmod +x ~/bin/colima && \
~/bin/colima version
```

**输出示例:**
```
colima version v0.10.3
git commit: ...
```

**Step 2: 安装 limactl **(虚拟化层)

```
brew install lima
limactl --version
```

**输出示例:**
```
limactl version 2.2.0
```

**Step 3: 下载 Docker CLI 静态二进制**

```
mkdir -p ~/docker && \
curl -sL -o /tmp/docker.tgz \
    https://download.docker.com/mac/static/stable/aarch64/docker-28.3.3.tgz && \
tar -xzf /tmp/docker.tgz -C /tmp && \
cp /tmp/docker/docker ~/docker/ && \
~/docker/docker version
```

**输出示例:**
```
Client: Docker Engine - Community
 Version:           27.5.1-rc1
...
```

**替代方案：使用 Homebrew 安装 **(如果已配置)
```
# 方法 A: 检查 colima 自带的 docker 命令
colima status
# 如果显示 running，则 ~/.local/bin/docker 应该可用

# 方法 B: 导出 docker socket
export DOCKER_HOST=unix://$HOME/.colima/default/docker.sock
docker info
```

**Step 4: 启动 colima 虚拟机并加载镜像**

```
export PATH="$HOME/docker:$PATH"   # 让 colima 找到 docker
colima start --cpu 2 --memory 4 --disk 20
```

等待输出:`READY. Run 'limactl shell colima' to open the shell.`

**验证 Docker 工作正常:**
```
limactl status colima
# 应该看到：running

docker images
# 此时可以没有任何镜像，后续会拉取

docker run --rm hello-world
# 测试运行一个小容器
```

**Step 5: 拉取构建所需镜像 **(指定 amd64 平台)

```
docker pull --platform linux/amd64 rockylinux/rockylinux:9
```

**提示**:如果直连超时，使用镜像代理:
```
docker pull --platform linux/amd64 docker.m.daocloud.io/rockylinux/rockylinux:9
```

**验证镜像已成功拉取**:
```
docker images | grep rockylinux
# 应该看到:
# rockylinux/rockylinux     9      <IMAGE-ID>   <DATE>   710MB
# docker.m.daocloud.io/rockylinux/rockylinux   9    <same-ID>
```

---

#### 📝 阶段二：准备源码与构建脚本

**Step 6: 打开项目目录**

```
cd /Users/yeung/Projects/MailProxy   # 替换为你的项目路径
cat VERSION                          # 确认版本号 (如 1.0.4)
```

**Step 7: 查看核心构建脚本**

```
cat deploy/build-rpm.sh
```

**关键流程**:

1. 读取 VERSION 文件中的版本号
2. `CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build ...` → 交叉编译 Linux x86_64 静态二进制
3. 将 spec/specs/source 复制到 `dist/rpmbuild/SOURCES/`
4. `docker run --platform linux/amd64 rockylinux:9 rpmbuild ...` → 在容器中执行 rpm 构建

**rpmbuild 目录结构详解**:
```
dist/rpmbuild/
├── SOURCES/          # 源代码文件 (binary + config + service)
│   ├── mailproxy     # Go 编译的 x86_64 二进制
│   ├── config.yaml   # 默认配置文件  
│   ├── mailproxy.service  # systemd unit 模板
│   └── README.md     # 产品说明文档
├── SPECS/            # RPM spec 文件定义包结构
│   └── mailproxy.spec
├── BUILD/            # 编译工作目录 (空目录)
├── BUILDROOT/        # 安装包根目录 (包含安装后的文件布局)
├── RPMS/x86_64/      # 最终生成的 RPM 包
│   └── mailproxy-1.0.4-1.el9.x86_64.rpm
└── SRPMS/            # 源码 RPM (如果有)
    └── mailproxy-1.0.4-1.el9.src.rpm
```

---

#### 🎯 阶段三：一键构建 RPM 包

**Step 8: 直接运行构建脚本**

```
export PATH="$HOME/docker:$PATH"    # 确保找到 docker
bash deploy/build-rpm.sh
```

**完整输出示例**(优化版):
```
==> 交叉编译 Linux x86_64 二进制 (v1.0.4)
==> 准备打包源文件
==> Docker 内执行 rpmbuild (rockylinux/rockylinux:9, linux/amd64)

Installed:
  rpm-build-4.16.1.3-40.el9.x86_64
  ... (更多依赖包)

Executing(%prep): /bin/sh -e /var/tmp/rpm-tmp.XXX
+ umask 022
+ cd /build/BUILD
+ rm -rf mailproxy-1.0.4
...

Executing(%install): /bin/sh -e /var/tmp/rpm-tmp.YYY
+ umask 022
+ '[' /build/BUILDROOT/mailproxy-1.0.4-1.el9.x86_64 '!=' / ']'
...
+ install -Dpm 0755 /build/SOURCES/mailproxy /build/BUILDROOT/mailproxy-1.0.4-1.el9.x86_64/usr/bin/mailproxy
+ install -Dpm 0644 /build/SOURCES/mailproxy.service /build/BUILDROOT/mailproxy-1.0.4-1.el9.x86_64/usr/lib/systemd/system/mailproxy.service
...

Wrote: /build/RPMS/x86_64/mailproxy-1.0.4-1.el9.x86_64.rpm

==> 构建完成：dist/mailproxy-1.0.4-1.el9.x86_64.rpm
    安装：rpm -ivh dist/mailproxy-1.0.4-1.el9.x86_64.rpm
    升级：rpm -Uvh dist/mailproxy-1.0.4-1.el9.x86_64.rpm
```

**成功标志**:
- ✅ 最后一行显示 "构建完成"
- ✅ 显示了正确的 rpm 文件名
- ✅ shell 返回码为 0 (`echo $?`)

**产物位置**:
- `dist/mailproxy-1.0.4-1.el9.x86_64.rpm` (主包)
- `dist/rpmbuild/RPMS/x86_64/mailproxy-1.0.4-1.el9.x86_64.rpm` (原始位置)
- `dist/rpmbuild/` 临时目录 (可删除)

**时间预期**: 
- 首次构建：约 2-3 分钟 (需要下载 rpmbuild 依赖)
- 重复构建：约 30-60 秒 (所有依赖已缓存)

**常见问题处理**:

| 错误信息 | 原因 | 解决方案 |
|---|---|---|
| `error: No such file or directory` | Docker CLI 不可见或 PATH 不对 | `export PATH="$HOME/docker:$PATH"` |
| `failed to resolve reference` | Docker Hub 网络问题 | 使用镜像代理，或提前用`docker save`导出 |
| `architecture mismatch` | 使用了 arm64 镜像而非 amd64 | 确保 docker run 带 `--platform linux/amd64` |
| `go.mod not found` | 不在项目根目录 | `cd` 到 MailProxy 项目根目录 |
| `container exited abnormally` | Docker 资源不足 | 增加 `colima start --memory 8 --disk 40` |

---

#### ✅ 阶段四：验证 RPM 包功能

> ⚠️ **重要提示**: 
> 在 macOS ARM64 上使用 Docker 进行 x86_64 容器测试存在跨平台卷挂载限制。
> 建议在生产服务器上测试实际安装流程。

**Step 9: 本地元数据验证 **(推荐)

```
# 检查基本元数据
rpm -qip dist/mailproxy-1.0.4-1.el9.x86_64.rpm

# 检查包内容列表
rpm -qplf dist/mailproxy-1.0.4-1.el9.x86_64.rpm

# 列出所有文件及其权限
ls -lh dist/mailproxy-1.0.4-1.el9.x86_64.rpm
```

**预期成功输出**:
```
Name        : mailproxy
Version     : 1.0.4
Release     : 1.el9
Architecture: x86_64
Install Date: (not installed)
Group       : Unspecified
Size        : 9461599
License     : Proprietary
Summary     : MailProxy SMTP relay gateway
Description :
Go 实现的 SMTP 邮件代理网关：业务程序统一连接本代理发信，
代理转发到后端真实 SMTP 服务器，支持多账号配置与路由策略。
```

✅ **通过标准**:
- Version 正确 (与 VERSION 文件一致)
- Architecture 是 x86_64
- Size 约 9MB (9,461,599 bytes)
- Description 包含正确信息
- 无错误信息

**Step 10: 高级验证 - 在测试容器中模拟安装**

由于跨平台限制，此步骤仅在相同架构下可靠运行。如必须测试，参考以下方法:

```
# 需要先构建完整测试环境
cd testtools
docker build -f Dockerfile.testenv -t mailproxy-test-complete . --platform linux/amd64

# 然后拷贝文件到容器内测试 (而不是卷挂载)
docker run -d --name mp-test-via-copy mailproxy-test-complete sleep infinity
docker cp dist/mailproxy-*.rpm mp-test-via-copy:/tmp/
docker exec mp-test-via-copy bash -c "rpm -ivh /tmp/mailproxy-*.rpm"
docker stop mp-test-via-copy
```

⚠️ **注意**: 这种方法比卷挂载更可靠，因为避免了跨平台文件系统转换。

```

*图：MailProxy 作为中央枢纽，统一管理多个后端 SMTP 服务*

```

---

<br/>

## 功能特性

- 🚀 **高性能** - 基于 Go 并发模型，支持高并发连接
- 🔒 **安全认证** - IP 白名单 + 可选 AUTH 认证，防止被滥用为开放中继
- 🔄 **智能路由** - 根据发件人自动选择最优后端账号，提升送达率
- 🎯 **零代码改造** - 只需修改 SMTP 地址配置，现有发邮件逻辑完全不用变
- 🔐 **TLS 加密** - 支持 SMTP over SSL (465) 和 STARTTLS (587)，确保传输安全
- ♻️ **热重载** - 配置文件 SIGHUP 热加载，无需重启服务
- 📊 **监控指标** - Prometheus 指标导出，实时监控系统状态
- 📝 **详细日志** - 结构化日志记录，每次发信全流程追踪
- 🎁 **企业邮箱兼容** - 自动处理「信封发件人必须等于认证账号」限制（阿里/腾讯/网易等）
- 🏗️ **易部署** - 单二进制文件 + systemd 托管，提供 RPM 包一键安装

---

<br/>

## 快速开始

```
# 1. 下载最新版本 (Linux AMD64)
curl -LO https://github.com/YanGLweI/MailProxy/releases/latest/download/mailproxy-1.0.4-linux-amd64.tar.gz
tar -xzf mailproxy-*.tar.gz
chmod +x mailproxy

# 或使用 RPM 包 (RHEL/CentOS/Rocky)
curl -LO https://github.com/YanGLweI/MailProxy/releases/latest/download/mailproxy-1.0.4-1.el9.x86_64.rpm
sudo rpm -ivh mailproxy-1.0.4-1.el9.x86_64.rpm
```

**立即体验：**

```
# 生成自签名证书
./deploy/gen-cert.sh

# 准备配置
cp config.example.yaml config.yaml
vim config.yaml             # 填写后端邮箱账号等

# 启动服务
./mailproxy -config config.yaml
```

**业务接入只需改 3 个参数：**

**业务接入只需改 3 个参数：**

| 参数 | 修改前                    | 修改后                      |
|------|---------------------------|-----------------------------|
| SMTP 主机 | `smtp.qiye.aliyun.com`    | `mailproxy.internal:465`    |
| SSL 证书 | 逐个管理各服务商证书      | 只需信任 mailproxy 一个证书 |  
| 开发成本 | 不同服务商 API 差异适配    | 统一 SMTP 协议，一次开发     |

### 构建源码

如需从源码构建：

```
# 本地构建
go build -o mailproxy .

# 交叉编译 Linux AMD64
GOOS=linux GOARCH=amd64 go build -o mailproxy .

# 运行测试
go test ./...
```

### 连通性自测

**推荐先验证连接再发送测试邮件：**

```
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

```
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

```
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

### 🚀 快速构建 (macOS 本机即可)

```
bash deploy/build-rpm.sh        # 产物 dist/mailproxy-<version>-1.el*.x86_64.rpm
```

**前置要求:**
- ✅ Go 1.25+ `go version` → go1.25.0 darwin/arm64  
- ✅ Docker CLI (colima 或 Docker Desktop) `command -v docker`
- ✅ colima 已运行 `limactl status` → running
- ✅ Rocky Linux 9 amd64 镜像已拉取 `docker images | grep rockylinux`

> 💡 **说明**: 如果 `build-rpm.sh` 执行失败，请先确保所有前置要求都满足。

---

### ⚠️ 重要注意事项

#### 问题 1: Docker Hub 认证超时
**现象**: `failed to authorize: failed to fetch anonymous token`  
**原因**: 国内网络访问 Docker Hub 受限  
**解决**: 配置镜像代理或使用其他镜像站

**方法 A: 使用 DaoCloud 镜像加速**
```
export REGISTRY_MIRROR=https://docker.m.daocloud.io
docker pull --platform linux/amd64 rockylinux/rockylinux:9
```

**方法 B: 提前在服务器拉取并导出导入**
```
# macOS 本地
docker save rockylinux/rockylinux:9 > rockylinux.tar

# 上传到服务器
scp rockylinux.tar server:/tmp/

# 服务器上加载
docker load < /tmp/rockylinux.tar
```

#### 问题 2: macOS ARM64 的卷挂载限制  
**现象**: 跨平台卷挂载导致文件不可见  
**原因**: Docker Desktop 在 arm64 host 上无法完全支持 x86_64 容器的卷挂载  

**解决方案**:  
1. 构建阶段使用正确的 `--platform linux/amd64`
2. 验证阶段改用 `rpm -qip` 检查元数据而非安装测试
3. 生产环境直接上传 RPM 到目标服务器安装

#### 问题 3: RockyLinux 基础镜像无 systemd  
**现象**: 测试容器无法启动 systemd  
**原因**: 官方镜像最小化设计  

**解决方案**: 使用完整测试 Dockerfile 或仅做元数据验证

---

### 📦 详细构建流程与每步操作指南

如需从头完整了解如何从 macOS 构建 x86_64 Linux RPM 包，请参考以下步骤:

#### 🔧 前置环境与依赖

| 组件 | 要求 | 检查命令 |
|---|---|---|
| macOS 芯片 | Apple Silicon (M1/M2/M3) 或 Intel | `uname -m` → arm64 / x86_64 |
| Go | ≥1.25.0 | `go version` |
| Homebrew (可选) | 用于安装 lima | `/usr/bin/ruby -e "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/master/install)"` (未装过才需要) |
| curl/tar/git | 系统自带 | `which curl tar git` |
| 网络 | 可访问 GitHub/Docker Hub(镜像站优先) | `ping github.com` |

> ⚠️ **重要提示**: 
> - ARM64 Mac 需要下载对应版本的 Docker CLI(见 Step 3)
> - 推荐使用 [colima](https://github.com/abiosoft/colima) 替代 Docker Desktop  
> - Docker Hub 直连可能超时，建议配置镜像代理

#### 🏗️ 阶段一：搭建临时容器环境（约 5 分钟）

**Step 1: 下载 colima 二进制 **(ARM64)

```
mkdir -p ~/bin && \
curl -sL -o ~/bin/colima \
    https://github.com/abiosoft/colima/releases/latest/download/colima-Darwin-arm64 && \
chmod +x ~/bin/colima && \
~/bin/colima version
```

**输出示例:**
```
colima version v0.10.3
git commit: ...
```

**Step 2: 安装 limactl **(虚拟化层)

```
brew install lima
limactl --version
```

**输出示例:**
```
limactl version 2.2.0
```

**Step 3: 下载 Docker CLI 静态二进制**

```
mkdir -p ~/docker && \
curl -sL -o /tmp/docker.tgz \
    https://download.docker.com/mac/static/stable/aarch64/docker-28.3.3.tgz && \
tar -xzf /tmp/docker.tgz -C /tmp && \
cp /tmp/docker/docker ~/docker/ && \
~/docker/docker version
```

**输出示例:**
```
Client: Docker Engine - Community
 Version:           27.5.1-rc1
...
```

**替代方案：使用 Homebrew 安装 **(如果已配置)
```
# 方法 A: 检查 colima 自带的 docker 命令
colima status
# 如果显示 running，则 ~/.local/bin/docker 应该可用

# 方法 B: 导出 docker socket
export DOCKER_HOST=unix://$HOME/.colima/default/docker.sock
docker info
```

**Step 4: 启动 colima 虚拟机并加载镜像**

```
export PATH="$HOME/docker:$PATH"   # 让 colima 找到 docker
colima start --cpu 2 --memory 4 --disk 20
```

等待输出:`READY. Run 'limactl shell colima' to open the shell.`

**验证 Docker 工作正常:**
```
limactl status colima
# 应该看到：running

docker images
# 此时可以没有任何镜像，后续会拉取

docker run --rm hello-world
# 测试运行一个小容器
```

**Step 5: 拉取构建所需镜像 **(指定 amd64 平台)

```
docker pull --platform linux/amd64 rockylinux/rockylinux:9
```

**提示**:如果直连超时，使用镜像代理:
```
docker pull --platform linux/amd64 docker.m.daocloud.io/rockylinux/rockylinux:9
```

**验证镜像已成功拉取**:
```
docker images | grep rockylinux
# 应该看到:
# rockylinux/rockylinux     9      <IMAGE-ID>   <DATE>   710MB
# docker.m.daocloud.io/rockylinux/rockylinux   9    <same-ID>
```

---

#### 📝 阶段二：准备源码与构建脚本

**Step 6: 打开项目目录**

```
cd /Users/yeung/Projects/MailProxy   # 替换为你的项目路径
cat VERSION                          # 确认版本号 (如 1.0.4)
```

**Step 7: 查看核心构建脚本**

```
cat deploy/build-rpm.sh
```

**关键流程**:

1. 读取 VERSION 文件中的版本号
2. `CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build ...` → 交叉编译 Linux x86_64 静态二进制
3. 将 spec/specs/source 复制到 `dist/rpmbuild/SOURCES/`
4. `docker run --platform linux/amd64 rockylinux:9 rpmbuild ...` → 在容器中执行 rpm 构建

**rpmbuild 目录结构详解**:
```
dist/rpmbuild/
├── SOURCES/          # 源代码文件 (binary + config + service)
│   ├── mailproxy     # Go 编译的 x86_64 二进制
│   ├── config.yaml   # 默认配置文件  
│   ├── mailproxy.service  # systemd unit 模板
│   └── README.md     # 产品说明文档
├── SPECS/            # RPM spec 文件定义包结构
│   └── mailproxy.spec
├── BUILD/            # 编译工作目录 (空目录)
├── BUILDROOT/        # 安装包根目录 (包含安装后的文件布局)
├── RPMS/x86_64/      # 最终生成的 RPM 包
│   └── mailproxy-1.0.4-1.el9.x86_64.rpm
└── SRPMS/            # 源码 RPM (如果有)
    └── mailproxy-1.0.4-1.el9.src.rpm
```

---

#### 🎯 阶段三：一键构建 RPM 包

**Step 8: 直接运行构建脚本**

```
export PATH="$HOME/docker:$PATH"    # 确保找到 docker
bash deploy/build-rpm.sh
```

**完整输出示例**(优化版):
```
==> 交叉编译 Linux x86_64 二进制 (v1.0.4)
==> 准备打包源文件
==> Docker 内执行 rpmbuild (rockylinux/rockylinux:9, linux/amd64)

Installed:
  rpm-build-4.16.1.3-40.el9.x86_64
  ... (更多依赖包)

Executing(%prep): /bin/sh -e /var/tmp/rpm-tmp.XXX
+ umask 022
+ cd /build/BUILD
+ rm -rf mailproxy-1.0.4
...

Executing(%install): /bin/sh -e /var/tmp/rpm-tmp.YYY
+ umask 022
+ '[' /build/BUILDROOT/mailproxy-1.0.4-1.el9.x86_64 '!=' / ']'
...
+ install -Dpm 0755 /build/SOURCES/mailproxy /build/BUILDROOT/mailproxy-1.0.4-1.el9.x86_64/usr/bin/mailproxy
+ install -Dpm 0644 /build/SOURCES/mailproxy.service /build/BUILDROOT/mailproxy-1.0.4-1.el9.x86_64/usr/lib/systemd/system/mailproxy.service
...

Wrote: /build/RPMS/x86_64/mailproxy-1.0.4-1.el9.x86_64.rpm

==> 构建完成：dist/mailproxy-1.0.4-1.el9.x86_64.rpm
    安装：rpm -ivh dist/mailproxy-1.0.4-1.el9.x86_64.rpm
    升级：rpm -Uvh dist/mailproxy-1.0.4-1.el9.x86_64.rpm
```

**成功标志**:
- ✅ 最后一行显示 "构建完成"
- ✅ 显示了正确的 rpm 文件名
- ✅ shell 返回码为 0 (`echo $?`)

**产物位置**:
- `dist/mailproxy-1.0.4-1.el9.x86_64.rpm` (主包)
- `dist/rpmbuild/RPMS/x86_64/mailproxy-1.0.4-1.el9.x86_64.rpm` (原始位置)
- `dist/rpmbuild/` 临时目录 (可删除)

**时间预期**: 
- 首次构建：约 2-3 分钟 (需要下载 rpmbuild 依赖)
- 重复构建：约 30-60 秒 (所有依赖已缓存)

**常见问题处理**:

| 错误信息 | 原因 | 解决方案 |
|---|---|---|
| `error: No such file or directory` | Docker CLI 不可见或 PATH 不对 | `export PATH="$HOME/docker:$PATH"` |
| `failed to resolve reference` | Docker Hub 网络问题 | 使用镜像代理，或提前用`docker save`导出 |
| `architecture mismatch` | 使用了 arm64 镜像而非 amd64 | 确保 docker run 带 `--platform linux/amd64` |
| `go.mod not found` | 不在项目根目录 | `cd` 到 MailProxy 项目根目录 |
| `container exited abnormally` | Docker 资源不足 | 增加 `colima start --memory 8 --disk 40` |

---

#### ✅ 阶段四：验证 RPM 包功能

> ⚠️ **重要提示**: 
> 在 macOS ARM64 上使用 Docker 进行 x86_64 容器测试存在跨平台卷挂载限制。
> 建议在生产服务器上测试实际安装流程。

**Step 9: 本地元数据验证 **(推荐)

```
# 检查基本元数据
rpm -qip dist/mailproxy-1.0.4-1.el9.x86_64.rpm

# 检查包内容列表
rpm -qplf dist/mailproxy-1.0.4-1.el9.x86_64.rpm

# 列出所有文件及其权限
ls -lh dist/mailproxy-1.0.4-1.el9.x86_64.rpm
```

**预期成功输出**:
```
Name        : mailproxy
Version     : 1.0.4
Release     : 1.el9
Architecture: x86_64
Install Date: (not installed)
Group       : Unspecified
Size        : 9461599
License     : Proprietary
Summary     : MailProxy SMTP relay gateway
Description :
Go 实现的 SMTP 邮件代理网关：业务程序统一连接本代理发信，
代理转发到后端真实 SMTP 服务器，支持多账号配置与路由策略。
```

✅ **通过标准**:
- Version 正确 (与 VERSION 文件一致)
- Architecture 是 x86_64
- Size 约 9MB (9,461,599 bytes)
- Description 包含正确信息
- 无错误信息

**Step 10: 高级验证 - 在测试容器中模拟安装**

由于跨平台限制，此步骤仅在相同架构下可靠运行。如必须测试，参考以下方法:

```
# 需要先构建完整测试环境
cd testtools
docker build -f Dockerfile.testenv -t mailproxy-test-complete . --platform linux/amd64

# 然后拷贝文件到容器内测试 (而不是卷挂载)
docker run -d --name mp-test-via-copy mailproxy-test-complete sleep infinity
docker cp dist/mailproxy-*.rpm mp-test-via-copy:/tmp/
docker exec mp-test-via-copy bash -c "rpm -ivh /tmp/mailproxy-*.rpm"
docker stop mp-test-via-copy
```

⚠️ **注意**: 这种方法比卷挂载更可靠，因为避免了跨平台文件系统转换。

```

*图：MailProxy 作为中央枢纽，统一管理多个后端 SMTP 服务*

```

---

<br/>

## 功能特性

- 🚀 **高性能** - 基于 Go 并发模型，支持高并发连接
- 🔒 **安全认证** - IP 白名单 + 可选 AUTH 认证，防止被滥用为开放中继
- 🔄 **智能路由** - 根据发件人自动选择最优后端账号，提升送达率
- 🎯 **零代码改造** - 只需修改 SMTP 地址配置，现有发邮件逻辑完全不用变
- 🔐 **TLS 加密** - 支持 SMTP over SSL (465) 和 STARTTLS (587)，确保传输安全
- ♻️ **热重载** - 配置文件 SIGHUP 热加载，无需重启服务
- 📊 **监控指标** - Prometheus 指标导出，实时监控系统状态
- 📝 **详细日志** - 结构化日志记录，每次发信全流程追踪
- 🎁 **企业邮箱兼容** - 自动处理「信封发件人必须等于认证账号」限制（阿里/腾讯/网易等）
- 🏗️ **易部署** - 单二进制文件 + systemd 托管，提供 RPM 包一键安装

---

<br/>

## 快速开始

```
# 1. 下载最新版本 (Linux AMD64)
curl -LO https://github.com/YanGLweI/MailProxy/releases/latest/download/mailproxy-1.0.4-linux-amd64.tar.gz
tar -xzf mailproxy-*.tar.gz
chmod +x mailproxy

# 或使用 RPM 包 (RHEL/CentOS/Rocky)
curl -LO https://github.com/YanGLweI/MailProxy/releases/latest/download/mailproxy-1.0.4-1.el9.x86_64.rpm
sudo rpm -ivh mailproxy-1.0.4-1.el9.x86_64.rpm
```

**立即体验：**

```
# 生成自签名证书
./deploy/gen-cert.sh

# 准备配置
cp config.example.yaml config.yaml
vim config.yaml             # 填写后端邮箱账号等

# 启动服务
./mailproxy -config config.yaml
```

**业务接入只需改 3 个参数：**

**业务接入只需改 3 个参数：**

| 参数 | 修改前                    | 修改后                      |
|------|---------------------------|-----------------------------|
| SMTP 主机 | `smtp.qiye.aliyun.com`    | `mailproxy.internal:465`    |
| SSL 证书 | 逐个管理各服务商证书      | 只需信任 mailproxy 一个证书 |  
| 开发成本 | 不同服务商 API 差异适配    | 统一 SMTP 协议，一次开发     |

### 构建源码

如需从源码构建：

```
# 本地构建
go build -o mailproxy .

# 交叉编译 Linux AMD64
GOOS=linux GOARCH=amd64 go build -o mailproxy .

# 运行测试
go test ./...
```

### 连通性自测

**推荐先验证连接再发送测试邮件：**

```
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

```
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

```
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

### 🚀 快速构建 (macOS 本机即可)

```
bash deploy/build-rpm.sh        # 产物 dist/mailproxy-<version>-1.el*.x86_64.rpm
```

**前置要求:**
- ✅ Go 1.25+ `go version` → go1.25.0 darwin/arm64  
- ✅ Docker CLI (colima 或 Docker Desktop) `command -v docker`
- ✅ colima 已运行 `limactl status` → running
- ✅ Rocky Linux 9 amd64 镜像已拉取 `docker images | grep rockylinux`

> 💡 **说明**: 如果 `build-rpm.sh` 执行失败，请先确保所有前置要求都满足。

---

### ⚠️ 重要注意事项

#### 问题 1: Docker Hub 认证超时
**现象**: `failed to authorize: failed to fetch anonymous token`  
**原因**: 国内网络访问 Docker Hub 受限  
**解决**: 配置镜像代理或使用其他镜像站

**方法 A: 使用 DaoCloud 镜像加速**
```
export REGISTRY_MIRROR=https://docker.m.daocloud.io
docker pull --platform linux/amd64 rockylinux/rockylinux:9
```

**方法 B: 提前在服务器拉取并导出导入**
```
# macOS 本地
docker save rockylinux/rockylinux:9 > rockylinux.tar

# 上传到服务器
scp rockylinux.tar server:/tmp/

# 服务器上加载
docker load < /tmp/rockylinux.tar
```

#### 问题 2: macOS ARM64 的卷挂载限制  
**现象**: 跨平台卷挂载导致文件不可见  
**原因**: Docker Desktop 在 arm64 host 上无法完全支持 x86_64 容器的卷挂载  

**解决方案**:  
1. 构建阶段使用正确的 `--platform linux/amd64`
2. 验证阶段改用 `rpm -qip` 检查元数据而非安装测试
3. 生产环境直接上传 RPM 到目标服务器安装

#### 问题 3: RockyLinux 基础镜像无 systemd  
**现象**: 测试容器无法启动 systemd  
**原因**: 官方镜像最小化设计  

**解决方案**: 使用完整测试 Dockerfile 或仅做元数据验证

---

### 📦 详细构建流程与每步操作指南

如需从头完整了解如何从 macOS 构建 x86_64 Linux RPM 包，请参考以下步骤:

#### 🔧 前置环境与依赖

| 组件 | 要求 | 检查命令 |
|---|---|---|
| macOS 芯片 | Apple Silicon (M1/M2/M3) 或 Intel | `uname -m` → arm64 / x86_64 |
| Go | ≥1.25.0 | `go version` |
| Homebrew (可选) | 用于安装 lima | `/usr/bin/ruby -e "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/master/install)"` (未装过才需要) |
| curl/tar/git | 系统自带 | `which curl tar git` |
| 网络 | 可访问 GitHub/Docker Hub(镜像站优先) | `ping github.com` |

> ⚠️ **重要提示**: 
> - ARM64 Mac 需要下载对应版本的 Docker CLI(见 Step 3)
> - 推荐使用 [colima](https://github.com/abiosoft/colima) 替代 Docker Desktop  
> - Docker Hub 直连可能超时，建议配置镜像代理

#### 🏗️ 阶段一：搭建临时容器环境（约 5 分钟）

**Step 1: 下载 colima 二进制 **(ARM64)

```
mkdir -p ~/bin && \
curl -sL -o ~/bin/colima \
    https://github.com/abiosoft/colima/releases/latest/download/colima-Darwin-arm64 && \
chmod +x ~/bin/colima && \
~/bin/colima version
```

**输出示例:**
```
colima version v0.10.3
git commit: ...
```

**Step 2: 安装 limactl **(虚拟化层)

```
brew install lima
limactl --version
```

**输出示例:**
```
limactl version 2.2.0
```

**Step 3: 下载 Docker CLI 静态二进制**

```
mkdir -p ~/docker && \
curl -sL -o /tmp/docker.tgz \
    https://download.docker.com/mac/static/stable/aarch64/docker-28.3.3.tgz && \
tar -xzf /tmp/docker.tgz -C /tmp && \
cp /tmp/docker/docker ~/docker/ && \
~/docker/docker version
```

**输出示例:**
```
Client: Docker Engine - Community
 Version:           27.5.1-rc1
...
```

**替代方案：使用 Homebrew 安装 **(如果已配置)
```
# 方法 A: 检查 colima 自带的 docker 命令
colima status
# 如果显示 running，则 ~/.local/bin/docker 应该可用

# 方法 B: 导出 docker socket
export DOCKER_HOST=unix://$HOME/.colima/default/docker.sock
docker info
```

**Step 4: 启动 colima 虚拟机并加载镜像**

```
export PATH="$HOME/docker:$PATH"   # 让 colima 找到 docker
colima start --cpu 2 --memory 4 --disk 20
```

等待输出:`READY. Run 'limactl shell colima' to open the shell.`

**验证 Docker 工作正常:**
```
limactl status colima
# 应该看到：running

docker images
# 此时可以没有任何镜像，后续会拉取

docker run --rm hello-world
# 测试运行一个小容器
```

**Step 5: 拉取构建所需镜像 **(指定 amd64 平台)

```
docker pull --platform linux/amd64 rockylinux/rockylinux:9
```

**提示**:如果直连超时，使用镜像代理:
```
docker pull --platform linux/amd64 docker.m.daocloud.io/rockylinux/rockylinux:9
```

**验证镜像已成功拉取**:
```
docker images | grep rockylinux
# 应该看到:
# rockylinux/rockylinux     9      <IMAGE-ID>   <DATE>   710MB
# docker.m.daocloud.io/rockylinux/rockylinux   9    <same-ID>
```

---

#### 📝 阶段二：准备源码与构建脚本

**Step 6: 打开项目目录**

```
cd /Users/yeung/Projects/MailProxy   # 替换为你的项目路径
cat VERSION                          # 确认版本号 (如 1.0.4)
```

**Step 7: 查看核心构建脚本**

```
cat deploy/build-rpm.sh
```

**关键流程**:

1. 读取 VERSION 文件中的版本号
2. `CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build ...` → 交叉编译 Linux x86_64 静态二进制
3. 将 spec/specs/source 复制到 `dist/rpmbuild/SOURCES/`
4. `docker run --platform linux/amd64 rockylinux:9 rpmbuild ...` → 在容器中执行 rpm 构建

**rpmbuild 目录结构详解**:
```
dist/rpmbuild/
├── SOURCES/          # 源代码文件 (binary + config + service)
│   ├── mailproxy     # Go 编译的 x86_64 二进制
│   ├── config.yaml   # 默认配置文件  
│   ├── mailproxy.service  # systemd unit 模板
│   └── README.md     # 产品说明文档
├── SPECS/            # RPM spec 文件定义包结构
│   └── mailproxy.spec
├── BUILD/            # 编译工作目录 (空目录)
├── BUILDROOT/        # 安装包根目录 (包含安装后的文件布局)
├── RPMS/x86_64/      # 最终生成的 RPM 包
│   └── mailproxy-1.0.4-1.el9.x86_64.rpm
└── SRPMS/            # 源码 RPM (如果有)
    └── mailproxy-1.0.4-1.el9.src.rpm
```

---

#### 🎯 阶段三：一键构建 RPM 包

**Step 8: 直接运行构建脚本**

```
export PATH="$HOME/docker:$PATH"    # 确保找到 docker
bash deploy/build-rpm.sh
```

**完整输出示例**(优化版):
```
==> 交叉编译 Linux x86_64 二进制 (v1.0.4)
==> 准备打包源文件
==> Docker 内执行 rpmbuild (rockylinux/rockylinux:9, linux/amd64)

Installed:
  rpm-build-4.16.1.3-40.el9.x86_64
  ... (更多依赖包)

Executing(%prep): /bin/sh -e /var/tmp/rpm-tmp.XXX
+ umask 022
+ cd /build/BUILD
+ rm -rf mailproxy-1.0.4
...

Executing(%install): /bin/sh -e /var/tmp/rpm-tmp.YYY
+ umask 022
+ '[' /build/BUILDROOT/mailproxy-1.0.4-1.el9.x86_64 '!=' / ']'
...
+ install -Dpm 0755 /build/SOURCES/mailproxy /build/BUILDROOT/mailproxy-1.0.4-1.el9.x86_64/usr/bin/mailproxy
+ install -Dpm 0644 /build/SOURCES/mailproxy.service /build/BUILDROOT/mailproxy-1.0.4-1.el9.x86_64/usr/lib/systemd/system/mailproxy.service
...

Wrote: /build/RPMS/x86_64/mailproxy-1.0.4-1.el9.x86_64.rpm

==> 构建完成：dist/mailproxy-1.0.4-1.el9.x86_64.rpm
    安装：rpm -ivh dist/mailproxy-1.0.4-1.el9.x86_64.rpm
    升级：rpm -Uvh dist/mailproxy-1.0.4-1.el9.x86_64.rpm
```

**成功标志**:
- ✅ 最后一行显示 "构建完成"
- ✅ 显示了正确的 rpm 文件名
- ✅ shell 返回码为 0 (`echo $?`)

**产物位置**:
- `dist/mailproxy-1.0.4-1.el9.x86_64.rpm` (主包)
- `dist/rpmbuild/RPMS/x86_64/mailproxy-1.0.4-1.el9.x86_64.rpm` (原始位置)
- `dist/rpmbuild/` 临时目录 (可删除)

**时间预期**: 
- 首次构建：约 2-3 分钟 (需要下载 rpmbuild 依赖)
- 重复构建：约 30-60 秒 (所有依赖已缓存)

**常见问题处理**:

| 错误信息 | 原因 | 解决方案 |
|---|---|---|
| `error: No such file or directory` | Docker CLI 不可见或 PATH 不对 | `export PATH="$HOME/docker:$PATH"` |
| `failed to resolve reference` | Docker Hub 网络问题 | 使用镜像代理，或提前用`docker save`导出 |
| `architecture mismatch` | 使用了 arm64 镜像而非 amd64 | 确保 docker run 带 `--platform linux/amd64` |
| `go.mod not found` | 不在项目根目录 | `cd` 到 MailProxy 项目根目录 |
| `container exited abnormally` | Docker 资源不足 | 增加 `colima start --memory 8 --disk 40` |

---

#### ✅ 阶段四：验证 RPM 包功能

> ⚠️ **重要提示**: 
> 在 macOS ARM64 上使用 Docker 进行 x86_64 容器测试存在跨平台卷挂载限制。
> 建议在生产服务器上测试实际安装流程。

**Step 9: 本地元数据验证 **(推荐)

```
# 检查基本元数据
rpm -qip dist/mailproxy-1.0.4-1.el9.x86_64.rpm

# 检查包内容列表
rpm -qplf dist/mailproxy-1.0.4-1.el9.x86_64.rpm

# 列出所有文件及其权限
ls -lh dist/mailproxy-1.0.4-1.el9.x86_64.rpm
```

**预期成功输出**:
```
Name        : mailproxy
Version     : 1.0.4
Release     : 1.el9
Architecture: x86_64
Install Date: (not installed)
Group       : Unspecified
Size        : 9461599
License     : Proprietary
Summary     : MailProxy SMTP relay gateway
Description :
Go 实现的 SMTP 邮件代理网关：业务程序统一连接本代理发信，
代理转发到后端真实 SMTP 服务器，支持多账号配置与路由策略。
```

✅ **通过标准**:
- Version 正确 (与 VERSION 文件一致)
- Architecture 是 x86_64
- Size 约 9MB (9,461,599 bytes)
- Description 包含正确信息
- 无错误信息

**Step 10: 高级验证 - 在测试容器中模拟安装**

由于跨平台限制，此步骤仅在相同架构下可靠运行。如必须测试，参考以下方法:

```
# 需要先构建完整测试环境
cd testtools
docker build -f Dockerfile.testenv -t mailproxy-test-complete . --platform linux/amd64

# 然后拷贝文件到容器内测试 (而不是卷挂载)
docker run -d --name mp-test-via-copy mailproxy-test-complete sleep infinity
docker cp dist/mailproxy-*.rpm mp-test-via-copy:/tmp/
docker exec mp-test-via-copy bash -c "rpm -ivh /tmp/mailproxy-*.rpm"
docker stop mp-test-via-copy
```

⚠️ **注意**: 这种方法比卷挂载更可靠，因为避免了跨平台文件系统转换。

```

*图：MailProxy 作为中央枢纽，统一管理多个后端 SMTP 服务*

```

---

<br/>

## 功能特性

- 🚀 **高性能** - 基于 Go 并发模型，支持高并发连接
- 🔒 **安全认证** - IP 白名单 + 可选 AUTH 认证，防止被滥用为开放中继
- 🔄 **智能路由** - 根据发件人自动选择最优后端账号，提升送达率
- 🎯 **零代码改造** - 只需修改 SMTP 地址配置，现有发邮件逻辑完全不用变
- 🔐 **TLS 加密** - 支持 SMTP over SSL (465) 和 STARTTLS (587)，确保传输安全
- ♻️ **热重载** - 配置文件 SIGHUP 热加载，无需重启服务
- 📊 **监控指标** - Prometheus 指标导出，实时监控系统状态
- 📝 **详细日志** - 结构化日志记录，每次发信全流程追踪
- 🎁 **企业邮箱兼容** - 自动处理「信封发件人必须等于认证账号」限制（阿里/腾讯/网易等）
- 🏗️ **易部署** - 单二进制文件 + systemd 托管，提供 RPM 包一键安装

---

<br/>

## 快速开始

```
# 1. 下载最新版本 (Linux AMD64)
curl -LO https://github.com/YanGLweI/MailProxy/releases/latest/download/mailproxy-1.0.4-linux-amd64.tar.gz
tar -xzf mailproxy-*.tar.gz
chmod +x mailproxy

# 或使用 RPM 包 (RHEL/CentOS/Rocky)
curl -LO https://github.com/YanGLweI/MailProxy/releases/latest/download/mailproxy-1.0.4-1.el9.x86_64.rpm
sudo rpm -ivh mailproxy-1.0.4-1.el9.x86_64.rpm
```

**立即体验：**

```
# 生成自签名证书
./deploy/gen-cert.sh

# 准备配置
cp config.example.yaml config.yaml
vim config.yaml             # 填写后端邮箱账号等

# 启动服务
./mailproxy -config config.yaml
```

**业务接入只需改 3 个参数：**

**业务接入只需改 3 个参数：**

| 参数 | 修改前                    | 修改后                      |
|------|---------------------------|-----------------------------|
| SMTP 主机 | `smtp.qiye.aliyun.com`    | `mailproxy.internal:465`    |
| SSL 证书 | 逐个管理各服务商证书      | 只需信任 mailproxy 一个证书 |  
| 开发成本 | 不同服务商 API 差异适配    | 统一 SMTP 协议，一次开发     |

### 构建源码

如需从源码构建：

```
# 本地构建
go build -o mailproxy .

# 交叉编译 Linux AMD64
GOOS=linux GOARCH=amd64 go build -o mailproxy .

# 运行测试
go test ./...
```

### 连通性自测

**推荐先验证连接再发送测试邮件：**

```
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

```
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

```
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

### 🚀 快速构建 (macOS 本机即可)

```
bash deploy/build-rpm.sh        # 产物 dist/mailproxy-<version>-1.el*.x86_64.rpm
```

**前置要求:**
- ✅ Go 1.25+ `go version` → go1.25.0 darwin/arm64  
- ✅ Docker CLI (colima 或 Docker Desktop) `command -v docker`
- ✅ colima 已运行 `limactl status` → running
- ✅ Rocky Linux 9 amd64 镜像已拉取 `docker images | grep rockylinux`

> 💡 **说明**: 如果 `build-rpm.sh` 执行失败，请先确保所有前置要求都满足。

---

### ⚠️ 重要注意事项

#### 问题 1: Docker Hub 认证超时
**现象**: `failed to authorize: failed to fetch anonymous token`  
**原因**: 国内网络访问 Docker Hub 受限  
**解决**: 配置镜像代理或使用其他镜像站

**方法 A: 使用 DaoCloud 镜像加速**
```
export REGISTRY_MIRROR=https://docker.m.daocloud.io
docker pull --platform linux/amd64 rockylinux/rockylinux:9
```

**方法 B: 提前在服务器拉取并导出导入**
```
# macOS 本地
docker save rockylinux/rockylinux:9 > rockylinux.tar

# 上传到服务器
scp rockylinux.tar server:/tmp/

# 服务器上加载
docker load < /tmp/rockylinux.tar
```

#### 问题 2: macOS ARM64 的卷挂载限制  
**现象**: 跨平台卷挂载导致文件不可见  
**原因**: Docker Desktop 在 arm64 host 上无法完全支持 x86_64 容器的卷挂载  

**解决方案**:  
1. 构建阶段使用正确的 `--platform linux/amd64`
2. 验证阶段改用 `rpm -qip` 检查元数据而非安装测试
3. 生产环境直接上传 RPM 到目标服务器安装

#### 问题 3: RockyLinux 基础镜像无 systemd  
**现象**: 测试容器无法启动 systemd  
**原因**: 官方镜像最小化设计  

**解决方案**: 使用完整测试 Dockerfile 或仅做元数据验证

---

### 📦 详细构建流程与每步操作指南

如需从头完整了解如何从 macOS 构建 x86_64 Linux RPM 包，请参考以下步骤:

#### 🔧 前置环境与依赖

| 组件 | 要求 | 检查命令 |
|---|---|---|
| macOS 芯片 | Apple Silicon (M1/M2/M3) 或 Intel | `uname -m` → arm64 / x86_64 |
| Go | ≥1.25.0 | `go version` |
| Homebrew (可选) | 用于安装 lima | `/usr/bin/ruby -e "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/master/install)"` (未装过才需要) |
| curl/tar/git | 系统自带 | `which curl tar git` |
| 网络 | 可访问 GitHub/Docker Hub(镜像站优先) | `ping github.com` |

> ⚠️ **重要提示**: 
> - ARM64 Mac 需要下载对应版本的 Docker CLI(见 Step 3)
> - 推荐使用 [colima](https://github.com/abiosoft/colima) 替代 Docker Desktop  
> - Docker Hub 直连可能超时，建议配置镜像代理

#### 🏗️ 阶段一：搭建临时容器环境（约 5 分钟）

**Step 1: 下载 colima 二进制 **(ARM64)

```
mkdir -p ~/bin && \
curl -sL -o ~/bin/colima \
    https://github.com/abiosoft/colima/releases/latest/download/colima-Darwin-arm64 && \
chmod +x ~/bin/colima && \
~/bin/colima version
```

**输出示例:**
```
colima version v0.10.3
git commit: ...
```

**Step 2: 安装 limactl **(虚拟化层)

```
brew install lima
limactl --version
```

**输出示例:**
```
limactl version 2.2.0
```

**Step 3: 下载 Docker CLI 静态二进制**

```
mkdir -p ~/docker && \
curl -sL -o /tmp/docker.tgz \
    https://download.docker.com/mac/static/stable/aarch64/docker-28.3.3.tgz && \
tar -xzf /tmp/docker.tgz -C /tmp && \
cp /tmp/docker/docker ~/docker/ && \
~/docker/docker version
```

**输出示例:**
```
Client: Docker Engine - Community
 Version:           27.5.1-rc1
...
```

**替代方案：使用 Homebrew 安装 **(如果已配置)
```
# 方法 A: 检查 colima 自带的 docker 命令
colima status
# 如果显示 running，则 ~/.local/bin/docker 应该可用

# 方法 B: 导出 docker socket
export DOCKER_HOST=unix://$HOME/.colima/default/docker.sock
docker info
```

**Step 4: 启动 colima 虚拟机并加载镜像**

```
export PATH="$HOME/docker:$PATH"   # 让 colima 找到 docker
colima start --cpu 2 --memory 4 --disk 20
```

等待输出:`READY. Run 'limactl shell colima' to open the shell.`

**验证 Docker 工作正常:**
```
limactl status colima
# 应该看到：running

docker images
# 此时可以没有任何镜像，后续会拉取

docker run --rm hello-world
# 测试运行一个小容器
```

**Step 5: 拉取构建所需镜像 **(指定 amd64 平台)

```
docker pull --platform linux/amd64 rockylinux/rockylinux:9
```

**提示**:如果直连超时，使用镜像代理:
```
docker pull --platform linux/amd64 docker.m.daocloud.io/rockylinux/rockylinux:9
```

**验证镜像已成功拉取**:
```
docker images | grep rockylinux
# 应该看到:
# rockylinux/rockylinux     9      <IMAGE-ID>   <DATE>   710MB
# docker.m.daocloud.io/rockylinux/rockylinux   9    <same-ID>
```

---

#### 📝 阶段二：准备源码与构建脚本

**Step 6: 打开项目目录**

```
cd /Users/yeung/Projects/MailProxy   # 替换为你的项目路径
cat VERSION                          # 确认版本号 (如 1.0.4)
```

**Step 7: 查看核心构建脚本**

```
cat deploy/build-rpm.sh
```

**关键流程**:

1. 读取 VERSION 文件中的版本号
2. `CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build ...` → 交叉编译 Linux x86_64 静态二进制
3. 将 spec/specs/source 复制到 `dist/rpmbuild/SOURCES/`
4. `docker run --platform linux/amd64 rockylinux:9 rpmbuild ...` → 在容器中执行 rpm 构建

**rpmbuild 目录结构详解**:
```
dist/rpmbuild/
├── SOURCES/          # 源代码文件 (binary + config + service)
│   ├── mailproxy     # Go 编译的 x86_64 二进制
│   ├── config.yaml   # 默认配置文件  
│   ├── mailproxy.service  # systemd unit 模板
│   └── README.md     # 产品说明文档
├── SPECS/            # RPM spec 文件定义包结构
│   └── mailproxy.spec
├── BUILD/            # 编译工作目录 (空目录)
├── BUILDROOT/        # 安装包根目录 (包含安装后的文件布局)
├── RPMS/x86_64/      # 最终生成的 RPM 包
│   └── mailproxy-1.0.4-1.el9.x86_64.rpm
└── SRPMS/            # 源码 RPM (如果有)
    └── mailproxy-1.0.4-1.el9.src.rpm
```

---

#### 🎯 阶段三：一键构建 RPM 包

**Step 8: 直接运行构建脚本**

```
export PATH="$HOME/docker:$PATH"    # 确保找到 docker
bash deploy/build-rpm.sh
```

**完整输出示例**(优化版):
```
==> 交叉编译 Linux x86_64 二进制 (v1.0.4)
==> 准备打包源文件
==> Docker 内执行 rpmbuild (rockylinux/rockylinux:9, linux/amd64)

Installed:
  rpm-build-4.16.1.3-40.el9.x86_64
  ... (更多依赖包)

Executing(%prep): /bin/sh -e /var/tmp/rpm-tmp.XXX
+ umask 022
+ cd /build/BUILD
+ rm -rf mailproxy-1.0.4
...

Executing(%install): /bin/sh -e /var/tmp/rpm-tmp.YYY
+ umask 022
+ '[' /build/BUILDROOT/mailproxy-1.0.4-1.el9.x86_64 '!=' / ']'
...
+ install -Dpm 0755 /build/SOURCES/mailproxy /build/BUILDROOT/mailproxy-1.0.4-1.el9.x86_64/usr/bin/mailproxy
+ install -Dpm 0644 /build/SOURCES/mailproxy.service /build/BUILDROOT/mailproxy-1.0.4-1.el9.x86_64/usr/lib/systemd/system/mailproxy.service
...

Wrote: /build/RPMS/x86_64/mailproxy-1.0.4-1.el9.x86_64.rpm

==> 构建完成：dist/mailproxy-1.0.4-1.el9.x86_64.rpm
    安装：rpm -ivh dist/mailproxy-1.0.4-1.el9.x86_64.rpm
    升级：rpm -Uvh dist/mailproxy-1.0.4-1.el9.x86_64.rpm
```

**成功标志**:
- ✅ 最后一行显示 "构建完成"
- ✅ 显示了正确的 rpm 文件名
- ✅ shell 返回码为 0 (`echo $?`)

**产物位置**:
- `dist/mailproxy-1.0.4-1.el9.x86_64.rpm` (主包)
- `dist/rpmbuild/RPMS/x86_64/mailproxy-1.0.4-1.el9.x86_64.rpm` (原始位置)
- `dist/rpmbuild/` 临时目录 (可删除)

**时间预期**: 
- 首次构建：约 2-3 分钟 (需要下载 rpmbuild 依赖)
- 重复构建：约 30-60 秒 (所有依赖已缓存)

**常见问题处理**:

| 错误信息 | 原因 | 解决方案 |
|---|---|---|
| `error: No such file or directory` | Docker CLI 不可见或 PATH 不对 | `export PATH="$HOME/docker:$PATH"` |
| `failed to resolve reference` | Docker Hub 网络问题 | 使用镜像代理，或提前用`docker save`导出 |
| `architecture mismatch` | 使用了 arm64 镜像而非 amd64 | 确保 docker run 带 `--platform linux/amd64` |
| `go.mod not found` | 不在项目根目录 | `cd` 到 MailProxy 项目根目录 |
| `container exited abnormally` | Docker 资源不足 | 增加 `colima start --memory 8 --disk 40` |

---

#### ✅ 阶段四：验证 RPM 包功能

> ⚠️ **重要提示**: 
> 在 macOS ARM64 上使用 Docker 进行 x86_64 容器测试存在跨平台卷挂载限制。
> 建议在生产服务器上测试实际安装流程。

**Step 9: 本地元数据验证 **(推荐)

```
# 检查基本元数据
rpm -qip dist/mailproxy-1.0.4-1.el9.x86_64.rpm

# 检查包内容列表
rpm -qplf dist/mailproxy-1.0.4-1.el9.x86_64.rpm

# 列出所有文件及其权限
ls -lh dist/mailproxy-1.0.4-1.el9.x86_64.rpm
```

**预期成功输出**:
```
Name        : mailproxy
Version     : 1.0.4
Release     : 1.el9
Architecture: x86_64
Install Date: (not installed)
Group       : Unspecified
Size        : 9461599
License     : Proprietary
Summary     : MailProxy SMTP relay gateway
Description :
Go 实现的 SMTP 邮件代理网关：业务程序统一连接本代理发信，
代理转发到后端真实 SMTP 服务器，支持多账号配置与路由策略。
```

✅ **通过标准**:
- Version 正确 (与 VERSION 文件一致)
- Architecture 是 x86_64
- Size 约 9MB (9,461,599 bytes)
- Description 包含正确信息
- 无错误信息

**Step 10: 高级验证 - 在测试容器中模拟安装**

由于跨平台限制，此步骤仅在相同架构下可靠运行。如必须测试，参考以下方法:

```
# 需要先构建完整测试环境
cd testtools
docker build -f Dockerfile.testenv -t mailproxy-test-complete . --platform linux/amd64

# 然后拷贝文件到容器内测试 (而不是卷挂载)
docker run -d --name mp-test-via-copy mailproxy-test-complete sleep infinity
docker cp dist/mailproxy-*.rpm mp-test-via-copy:/tmp/
docker exec mp-test-via-copy bash -c "rpm -ivh /tmp/mailproxy-*.rpm"
docker stop mp-test-via-copy
```

⚠️ **注意**: 这种方法比卷挂载更可靠，因为避免了跨平台文件系统转换。

```

*图：MailProxy 作为中央枢纽，统一管理多个后端 SMTP 服务*

```

---

<br/>

## 功能特性

- 🚀 **高性能** - 基于 Go 并发模型，支持高并发连接
- 🔒 **安全认证** - IP 白名单 + 可选 AUTH 认证，防止被滥用为开放中继
- 🔄 **智能路由** - 根据发件人自动选择最优后端账号，提升送达率
- 🎯 **零代码改造** - 只需修改 SMTP 地址配置，现有发邮件逻辑完全不用变
- 🔐 **TLS 加密** - 支持 SMTP over SSL (465) 和 STARTTLS (587)，确保传输安全
- ♻️ **热重载** - 配置文件 SIGHUP 热加载，无需重启服务
- 📊 **监控指标** - Prometheus 指标导出，实时监控系统状态
- 📝 **详细日志** - 结构化日志记录，每次发信全流程追踪
- 🎁 **企业邮箱兼容** - 自动处理「信封发件人必须等于认证账号」限制（阿里/腾讯/网易等）
- 🏗️ **易部署** - 单二进制文件 + systemd 托管，提供 RPM 包一键安装

---

<br/>

## 快速开始

```
# 1. 下载最新版本 (Linux AMD64)
curl -LO https://github.com/YanGLweI/MailProxy/releases/latest/download/mailproxy-1.0.4-linux-amd64.tar.gz
tar -xzf mailproxy-*.tar.gz
chmod +x mailproxy

# 或使用 RPM 包 (RHEL/CentOS/Rocky)
curl -LO https://github.com/YanGLweI/MailProxy/releases/latest/download/mailproxy-1.0.4-1.el9.x86_64.rpm
sudo rpm -ivh mailproxy-1.0.4-1.el9.x86_64.rpm
```

**立即体验：**

```
# 生成自签名证书
./deploy/gen-cert.sh

# 准备配置
cp config.example.yaml config.yaml
vim config.yaml             # 填写后端邮箱账号等

# 启动服务
./mailproxy -config config.yaml
```

**业务接入只需改 3 个参数：**

**业务接入只需改 3 个参数：**

| 参数 | 修改前                    | 修改后                      |
|------|---------------------------|-----------------------------|
| SMTP 主机 | `smtp.qiye.aliyun.com`    | `mailproxy.internal:465`    |
| SSL 证书 | 逐个管理各服务商证书      | 只需信任 mailproxy 一个证书 |  
| 开发成本 | 不同服务商 API 差异适配    | 统一 SMTP 协议，一次开发     |

### 构建源码

如需从源码构建：

```
# 本地构建
go build -o mailproxy .

# 交叉编译 Linux AMD64
GOOS=linux GOARCH=amd64 go build -o mailproxy .

# 运行测试
go test ./...
```

### 连通性自测

**推荐先验证连接再发送测试邮件：**

```
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

```
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

```
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

### 🚀 快速构建 (macOS 本机即可)

```
bash deploy/build-rpm.sh        # 产物 dist/mailproxy-<version>-1.el*.x86_64.rpm
```

**前置要求:**
- ✅ Go 1.25+ `go version` → go1.25.0 darwin/arm64  
- ✅ Docker CLI (colima 或 Docker Desktop) `command -v docker`
- ✅ colima 已运行 `limactl status` → running
- ✅ Rocky Linux 9 amd64 镜像已拉取 `docker images | grep rockylinux`

> 💡 **说明**: 如果 `build-rpm.sh` 执行失败，请先确保所有前置要求都满足。

---

### ⚠️ 重要注意事项

#### 问题 1: Docker Hub 认证超时
**现象**: `failed to authorize: failed to fetch anonymous token`  
**原因**: 国内网络访问 Docker Hub 受限  
**解决**: 配置镜像代理或使用其他镜像站

**方法 A: 使用 DaoCloud 镜像加速**
```
export REGISTRY_MIRROR=https://docker.m.daocloud.io
docker pull --platform linux/amd64 rockylinux/rockylinux:9
```

**方法 B: 提前在服务器拉取并导出导入**
```
# macOS 本地
docker save rockylinux/rockylinux:9 > rockylinux.tar

# 上传到服务器
scp rockylinux.tar server:/tmp/

# 服务器上加载
docker load < /tmp/rockylinux.tar
```

#### 问题 2: macOS ARM64 的卷挂载限制  
**现象**: 跨平台卷挂载导致文件不可见  
**原因**: Docker Desktop 在 arm64 host 上无法完全支持 x86_64 容器的卷挂载  

**解决方案**:  
1. 构建阶段使用正确的 `--platform linux/amd64`
2. 验证阶段改用 `rpm -qip` 检查元数据而非安装测试
3. 生产环境直接上传 RPM 到目标服务器安装

#### 问题 3: RockyLinux 基础镜像无 systemd  
**现象**: 测试容器无法启动 systemd  
**原因**: 官方镜像最小化设计  

**解决方案**: 使用完整测试 Dockerfile 或仅做元数据验证

---

### 📦 详细构建流程与每步操作指南

如需从头完整了解如何从 macOS 构建 x86_64 Linux RPM 包，请参考以下步骤:

#### 🔧 前置环境与依赖

| 组件 | 要求 | 检查命令 |
|---|---|---|
| macOS 芯片 | Apple Silicon (M1/M2/M3) 或 Intel | `uname -m` → arm64 / x86_64 |
| Go | ≥1.25.0 | `go version` |
| Homebrew (可选) | 用于安装 lima | `/usr/bin/ruby -e "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/master/install)"` (未装过才需要) |
| curl/tar/git | 系统自带 | `which curl tar git` |
| 网络 | 可访问 GitHub/Docker Hub(镜像站优先) | `ping github.com` |

> ⚠️ **重要提示**: 
> - ARM64 Mac 需要下载对应版本的 Docker CLI(见 Step 3)
> - 推荐使用 [colima](https://github.com/abiosoft/colima) 替代 Docker Desktop  
> - Docker Hub 直连可能超时，建议配置镜像代理

#### 🏗️ 阶段一：搭建临时容器环境（约 5 分钟）

**Step 1: 下载 colima 二进制 **(ARM64)

```
mkdir -p ~/bin && \
curl -sL -o ~/bin/colima \
    https://github.com/abiosoft/colima/releases/latest/download/colima-Darwin-arm64 && \
chmod +x ~/bin/colima && \
~/bin/colima version
```

**输出示例:**
```
colima version v0.10.3
git commit: ...
```

**Step 2: 安装 limactl **(虚拟化层)

```
brew install lima
limactl --version
```

**输出示例:**
```
limactl version 2.2.0
```

**Step 3: 下载 Docker CLI 静态二进制**

```
mkdir -p ~/docker && \
curl -sL -o /tmp/docker.tgz \
    https://download.docker.com/mac/static/stable/aarch64/docker-28.3.3.tgz && \
tar -xzf /tmp/docker.tgz -C /tmp && \
cp /tmp/docker/docker ~/docker/ && \
~/docker/docker version
```

**输出示例:**
```
Client: Docker Engine - Community
 Version:           27.5.1-rc1
...
```

**替代方案：使用 Homebrew 安装 **(如果已配置)
```
# 方法 A: 检查 colima 自带的 docker 命令
colima status
# 如果显示 running，则 ~/.local/bin/docker 应该可用

# 方法 B: 导出 docker socket
export DOCKER_HOST=unix://$HOME/.colima/default/docker.sock
docker info
```

**Step 4: 启动 colima 虚拟机并加载镜像**

```
export PATH="$HOME/docker:$PATH"   # 让 colima 找到 docker
colima start --cpu 2 --memory 4 --disk 20
```

等待输出:`READY. Run 'limactl shell colima' to open the shell.`

**验证 Docker 工作正常:**
```
limactl status colima
# 应该看到：running

docker images
# 此时可以没有任何镜像，后续会拉取

docker run --rm hello-world
# 测试运行一个小容器
```

**Step 5: 拉取构建所需镜像 **(指定 amd64 平台)

```
docker pull --platform linux/amd64 rockylinux/rockylinux:9
```

**提示**:如果直连超时，使用镜像代理:
```
docker pull --platform linux/amd64 docker.m.daocloud.io/rockylinux/rockylinux:9
```

**验证镜像已成功拉取**:
```
docker images | grep rockylinux
# 应该看到:
# rockylinux/rockylinux     9      <IMAGE-ID>   <DATE>   710MB
# docker.m.daocloud.io/rockylinux/rockylinux   9    <same-ID>
```

---

#### 📝 阶段二：准备源码与构建脚本

**Step 6: 打开项目目录**

```
cd /Users/yeung/Projects/MailProxy   # 替换为你的项目路径
cat VERSION                          # 确认版本号 (如 1.0.4)
```

**Step 7: 查看核心构建脚本**

```
cat deploy/build-rpm.sh
```

**关键流程**:

1. 读取 VERSION 文件中的版本号
2. `CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build ...` → 交叉编译 Linux x86_64 静态二进制
3. 将 spec/specs/source 复制到 `dist/rpmbuild/SOURCES/`
4. `docker run --platform linux/amd64 rockylinux:9 rpmbuild ...` → 在容器中执行 rpm 构建

**rpmbuild 目录结构详解**:
```
dist/rpmbuild/
├── SOURCES/          # 源代码文件 (binary + config + service)
│   ├── mailproxy     # Go 编译的 x86_64 二进制
│   ├── config.yaml   # 默认配置文件  
│   ├── mailproxy.service  # systemd unit 模板
│   └── README.md     # 产品说明文档
├── SPECS/            # RPM spec 文件定义包结构
│   └── mailproxy.spec
├── BUILD/            # 编译工作目录 (空目录)
├── BUILDROOT/        # 安装包根目录 (包含安装后的文件布局)
├── RPMS/x86_64/      # 最终生成的 RPM 包
│   └── mailproxy-1.0.4-1.el9.x86_64.rpm
└── SRPMS/            # 源码 RPM (如果有)
    └── mailproxy-1.0.4-1.el9.src.rpm
```

---

#### 🎯 阶段三：一键构建 RPM 包

**Step 8: 直接运行构建脚本**

```
export PATH="$HOME/docker:$PATH"    # 确保找到 docker
bash deploy/build-rpm.sh
```

**完整输出示例**(优化版):
```
==> 交叉编译 Linux x86_64 二进制 (v1.0.4)
==> 准备打包源文件
==> Docker 内执行 rpmbuild (rockylinux/rockylinux:9, linux/amd64)

Installed:
  rpm-build-4.16.1.3-40.el9.x86_64
  ... (更多依赖包)

Executing(%prep): /bin/sh -e /var/tmp/rpm-tmp.XXX
+ umask 022
+ cd /build/BUILD
+ rm -rf mailproxy-1.0.4
...

Executing(%install): /bin/sh -e /var/tmp/rpm-tmp.YYY
+ umask 022
+ '[' /build/BUILDROOT/mailproxy-1.0.4-1.el9.x86_64 '!=' / ']'
...
+ install -Dpm 0755 /build/SOURCES/mailproxy /build/BUILDROOT/mailproxy-1.0.4-1.el9.x86_64/usr/bin/mailproxy
+ install -Dpm 0644 /build/SOURCES/mailproxy.service /build/BUILDROOT/mailproxy-1.0.4-1.el9.x86_64/usr/lib/systemd/system/mailproxy.service
...

Wrote: /build/RPMS/x86_64/mailproxy-1.0.4-1.el9.x86_64.rpm

==> 构建完成：dist/mailproxy-1.0.4-1.el9.x86_64.rpm
    安装：rpm -ivh dist/mailproxy-1.0.4-1.el9.x86_64.rpm
    升级：rpm -Uvh dist/mailproxy-1.0.4-1.el9.x86_64.rpm
```

**成功标志**:
- ✅ 最后一行显示 "构建完成"
- ✅ 显示了正确的 rpm 文件名
- ✅ shell 返回码为 0 (`echo $?`)

**产物位置**:
- `dist/mailproxy-1.0.4-1.el9.x86_64.rpm` (主包)
- `dist/rpmbuild/RPMS/x86_64/mailproxy-1.0.4-1.el9.x86_64.rpm` (原始位置)
- `dist/rpmbuild/` 临时目录 (可删除)

**时间预期**: 
- 首次构建：约 2-3 分钟 (需要下载 rpmbuild 依赖)
- 重复构建：约 30-60 秒 (所有依赖已缓存)

**常见问题处理**:

| 错误信息 | 原因 | 解决方案 |
|---|---|---|
| `error: No such file or directory` | Docker CLI 不可见或 PATH 不对 | `export PATH="$HOME/docker:$PATH"` |
| `failed to resolve reference` | Docker Hub 网络问题 | 使用镜像代理，或提前用`docker save`导出 |
| `architecture mismatch` | 使用了 arm64 镜像而非 amd64 | 确保 docker run 带 `--platform linux/amd64` |
| `go.mod not found` | 不在项目根目录 | `cd` 到 MailProxy 项目根目录 |
| `container exited abnormally` | Docker 资源不足 | 增加 `colima start --memory 8 --disk 40` |

---

#### ✅ 阶段四：验证 RPM 包功能

> ⚠️ **重要提示**: 
> 在 macOS ARM64 上使用 Docker 进行 x86_64 容器测试存在跨平台卷挂载限制。
> 建议在生产服务器上测试实际安装流程。

**Step 9: 本地元数据验证 **(推荐)

```
# 检查基本元数据
rpm -qip dist/mailproxy-1.0.4-1.el9.x86_64.rpm

# 检查包内容列表
rpm -qplf dist/mailproxy-1.0.4-1.el9.x86_64.rpm

# 列出所有文件及其权限
ls -lh dist/mailproxy-1.0.4-1.el9.x86_64.rpm
```

**预期成功输出**:
```
Name        : mailproxy
Version     : 1.0.4
Release     : 1.el9
Architecture: x86_64
Install Date: (not installed)
Group       : Unspecified
Size        : 9461599
License     : Proprietary
Summary     : MailProxy SMTP relay gateway
Description :
Go 实现的 SMTP 邮件代理网关：业务程序统一连接本代理发信，
代理转发到后端真实 SMTP 服务器，支持多账号配置与路由策略。
```

✅ **通过标准**:
- Version 正确 (与 VERSION 文件一致)
- Architecture 是 x86_64
- Size 约 9MB (9,461,599 bytes)
- Description 包含正确信息
- 无错误信息

**Step 10: 高级验证 - 在测试容器中模拟安装**

由于跨平台限制，此步骤仅在相同架构下可靠运行。如必须测试，参考以下方法:

```
# 需要先构建完整测试环境
cd testtools
docker build -f Dockerfile.testenv -t mailproxy-test-complete . --platform linux/amd64

# 然后拷贝文件到容器内测试 (而不是卷挂载)
docker run -d --name mp-test-via-copy mailproxy-test-complete sleep infinity
docker cp dist/mailproxy-*.rpm mp-test-via-copy:/tmp/
docker exec mp-test-via-copy bash -c "rpm -ivh /tmp/mailproxy-*.rpm"
docker stop mp-test-via-copy
```

⚠️ **注意**: 这种方法比卷挂载更可靠，因为避免了跨平台文件系统转换。

```

*图：MailProxy 作为中央枢纽，统一管理多个后端 SMTP 服务*

```

---

<br/>

## 功能特性

- 🚀 **高性能** - 基于 Go 并发模型，支持高并发连接
- 🔒 **安全认证** - IP 白名单 + 可选 AUTH 认证，防止被滥用为开放中继
- 🔄 **智能路由** - 根据发件人自动选择最优后端账号，提升送达率
- 🎯 **零代码改造** - 只需修改 SMTP 地址配置，现有发邮件逻辑完全不用变
- 🔐 **TLS 加密** - 支持 SMTP over SSL (465) 和 STARTTLS (587)，确保传输安全
- ♻️ **热重载** - 配置文件 SIGHUP 热加载，无需重启服务
- 📊 **监控指标** - Prometheus 指标导出，实时监控系统状态
- 📝 **详细日志** - 结构化日志记录，每次发信全流程追踪
- 🎁 **企业邮箱兼容** - 自动处理「信封发件人必须等于认证账号」限制（阿里/腾讯/网易等）
- 🏗️ **易部署** - 单二进制文件 + systemd 托管，提供 RPM 包一键安装

---

<br/>

## 快速开始

```
# 1. 下载最新版本 (Linux AMD64)
curl -LO https://github.com/YanGLweI/MailProxy/releases/latest/download/mailproxy-1.0.4-linux-amd64.tar.gz
tar -xzf mailproxy-*.tar.gz
chmod +x mailproxy

# 或使用 RPM 包 (RHEL/CentOS/Rocky)
curl -LO https://github.com/YanGLweI/MailProxy/releases/latest/download/mailproxy-1.0.4-1.el9.x86_64.rpm
sudo rpm -ivh mailproxy-1.0.4-1.el9.x86_64.rpm
```

**立即体验：**

```
# 生成自签名证书
./deploy/gen-cert.sh

# 准备配置
cp config.example.yaml config.yaml
vim config.yaml             # 填写后端邮箱账号等

# 启动服务
./mailproxy -config config.yaml
```

**业务接入只需改 3 个参数：**

**业务接入只需改 3 个参数：**

| 参数 | 修改前                    | 修改后                      |
|------|---------------------------|-----------------------------|
| SMTP 主机 | `smtp.qiye.aliyun.com`    | `mailproxy.internal:465`    |
| SSL 证书 | 逐个管理各服务商证书      | 只需信任 mailproxy 一个证书 |  
| 开发成本 | 不同服务商 API 差异适配    | 统一 SMTP 协议，一次开发     |

### 构建源码

如需从源码构建：

```
# 本地构建
go build -o mailproxy .

# 交叉编译 Linux AMD64
GOOS=linux GOARCH=amd64 go build -o mailproxy .

# 运行测试
go test ./...
```

### 连通性自测

**推荐先验证连接再发送测试邮件：**

```
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

```
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

```
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

### 🚀 快速构建 (macOS 本机即可)

```
bash deploy/build-rpm.sh        # 产物 dist/mailproxy-<version>-1.el*.x86_64.rpm
```

**前置要求:**
- ✅ Go 1.25+ `go version` → go1.25.0 darwin/arm64  
- ✅ Docker CLI (colima 或 Docker Desktop) `command -v docker`
- ✅ colima 已运行 `limactl status` → running
- ✅ Rocky Linux 9 amd64 镜像已拉取 `docker images | grep rockylinux`

> 💡 **说明**: 如果 `build-rpm.sh` 执行失败，请先确保所有前置要求都满足。

---

### ⚠️ 重要注意事项

#### 问题 1: Docker Hub 认证超时
**现象**: `failed to authorize: failed to fetch anonymous token`  
**原因**: 国内网络访问 Docker Hub 受限  
**解决**: 配置镜像代理或使用其他镜像站

**方法 A: 使用 DaoCloud 镜像加速**
```
export REGISTRY_MIRROR=https://docker.m.daocloud.io
docker pull --platform linux/amd64 rockylinux/rockylinux:9
```

**方法 B: 提前在服务器拉取并导出导入**
```
# macOS 本地
docker save rockylinux/rockylinux:9 > rockylinux.tar

# 上传到服务器
scp rockylinux.tar server:/tmp/

# 服务器上加载
docker load < /tmp/rockylinux.tar
```

#### 问题 2: macOS ARM64 的卷挂载限制  
**现象**: 跨平台卷挂载导致文件不可见  
**原因**: Docker Desktop 在 arm64 host 上无法完全支持 x86_64 容器的卷挂载  

**解决方案**:  
1. 构建阶段使用正确的 `--platform linux/amd64`
2. 验证阶段改用 `rpm -qip` 检查元数据而非安装测试
3. 生产环境直接上传 RPM 到目标服务器安装

#### 问题 3: RockyLinux 基础镜像无 systemd  
**现象**: 测试容器无法启动 systemd  
**原因**: 官方镜像最小化设计  

**解决方案**: 使用完整测试 Dockerfile 或仅做元数据验证

---

### 📦 详细构建流程与每步操作指南

如需从头完整了解如何从 macOS 构建 x86_64 Linux RPM 包，请参考以下步骤:

#### 🔧 前置环境与依赖

| 组件 | 要求 | 检查命令 |
|---|---|---|
| macOS 芯片 | Apple Silicon (M1/M2/M3) 或 Intel | `uname -m` → arm64 / x86_64 |
| Go | ≥1.25.0 | `go version` |
| Homebrew (可选) | 用于安装 lima | `/usr/bin/ruby -e "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/master/install)"` (未装过才需要) |
| curl/tar/git | 系统自带 | `which curl tar git` |
| 网络 | 可访问 GitHub/Docker Hub(镜像站优先) | `ping github.com` |

> ⚠️ **重要提示**: 
> - ARM64 Mac 需要下载对应版本的 Docker CLI(见 Step 3)
> - 推荐使用 [colima](https://github.com/abiosoft/colima) 替代 Docker Desktop  
> - Docker Hub 直连可能超时，建议配置镜像代理

#### 🏗️ 阶段一：搭建临时容器环境（约 5 分钟）

**Step 1: 下载 colima 二进制 **(ARM64)

```
mkdir -p ~/bin && \
curl -sL -o ~/bin/colima \
    https://github.com/abiosoft/colima/releases/latest/download/colima-Darwin-arm64 && \
chmod +x ~/bin/colima && \
~/bin/colima version
```

**输出示例:**
```
colima version v0.10.3
git commit: ...
```

**Step 2: 安装 limactl **(虚拟化层)

```
brew install lima
limactl --version
```

**输出示例:**
```
limactl version 2.2.0
```

**Step 3: 下载 Docker CLI 静态二进制**

```
mkdir -p ~/docker && \
curl -sL -o /tmp/docker.tgz \
    https://download.docker.com/mac/static/stable/aarch64/docker-28.3.3.tgz && \
tar -xzf /tmp/docker.tgz -C /tmp && \
cp /tmp/docker/docker ~/docker/ && \
~/docker/docker version
```

**输出示例:**
```
Client: Docker Engine - Community
 Version:           27.5.1-rc1
...
```

**替代方案：使用 Homebrew 安装 **(如果已配置)
```
# 方法 A: 检查 colima 自带的 docker 命令
colima status
# 如果显示 running，则 ~/.local/bin/docker 应该可用

# 方法 B: 导出 docker socket
export DOCKER_HOST=unix://$HOME/.colima/default/docker.sock
docker info
```

**Step 4: 启动 colima 虚拟机并加载镜像**

```
export PATH="$HOME/docker:$PATH"   # 让 colima 找到 docker
colima start --cpu 2 --memory 4 --disk 20
```

等待输出:`READY. Run 'limactl shell colima' to open the shell.`

**验证 Docker 工作正常:**
```
limactl status colima
# 应该看到：running

docker images
# 此时可以没有任何镜像，后续会拉取

docker run --rm hello-world
# 测试运行一个小容器
```

**Step 5: 拉取构建所需镜像 **(指定 amd64 平台)

```
docker pull --platform linux/amd64 rockylinux/rockylinux:9
```

**提示**:如果直连超时，使用镜像代理:
```
docker pull --platform linux/amd64 docker.m.daocloud.io/rockylinux/rockylinux:9
```

**验证镜像已成功拉取**:
```
docker images | grep rockylinux
# 应该看到:
# rockylinux/rockylinux     9      <IMAGE-ID>   <DATE>   710MB
# docker.m.daocloud.io/rockylinux/rockylinux   9    <same-ID>
```

---

#### 📝 阶段二：准备源码与构建脚本

**Step 6: 打开项目目录**

```
cd /Users/yeung/Projects/MailProxy   # 替换为你的项目路径
cat VERSION                          # 确认版本号 (如 1.0.4)
```

**Step 7: 查看核心构建脚本**

```
cat deploy/build-rpm.sh
```

**关键流程**:

1. 读取 VERSION 文件中的版本号
2. `CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build ...` → 交叉编译 Linux x86_64 静态二进制
3. 将 spec/specs/source 复制到 `dist/rpmbuild/SOURCES/`
4. `docker run --platform linux/amd64 rockylinux:9 rpmbuild ...` → 在容器中执行 rpm 构建

**rpmbuild 目录结构详解**:
```
dist/rpmbuild/
├── SOURCES/          # 源代码文件 (binary + config + service)
│   ├── mailproxy     # Go 编译的 x86_64 二进制
│   ├── config.yaml   # 默认配置文件  
│   ├── mailproxy.service  # systemd unit 模板
│   └── README.md     # 产品说明文档
├── SPECS/            # RPM spec 文件定义包结构
│   └── mailproxy.spec
├── BUILD/            # 编译工作目录 (空目录)
├── BUILDROOT/        # 安装包根目录 (包含安装后的文件布局)
├── RPMS/x86_64/      # 最终生成的 RPM 包
│   └── mailproxy-1.0.4-1.el9.x86_64.rpm
└── SRPMS/            # 源码 RPM (如果有)
    └── mailproxy-1.0.4-1.el9.src.rpm
```

---

#### 🎯 阶段三：一键构建 RPM 包

**Step 8: 直接运行构建脚本**

```
export PATH="$HOME/docker:$PATH"    # 确保找到 docker
bash deploy/build-rpm.sh
```

**完整输出示例**(优化版):
```
==> 交叉编译 Linux x86_64 二进制 (v1.0.4)
==> 准备打包源文件
==> Docker 内执行 rpmbuild (rockylinux/rockylinux:9, linux/amd64)

Installed:
  rpm-build-4.16.1.3-40.el9.x86_64
  ... (更多依赖包)

Executing(%prep): /bin/sh -e /var/tmp/rpm-tmp.XXX
+ umask 022
+ cd /build/BUILD
+ rm -rf mailproxy-1.0.4
...

Executing(%install): /bin/sh -e /var/tmp/rpm-tmp.YYY
+ umask 022
+ '[' /build/BUILDROOT/mailproxy-1.0.4-1.el9.x86_64 '!=' / ']'
...
+ install -Dpm 0755 /build/SOURCES/mailproxy /build/BUILDROOT/mailproxy-1.0.4-1.el9.x86_64/usr/bin/mailproxy
+ install -Dpm 0644 /build/SOURCES/mailproxy.service /build/BUILDROOT/mailproxy-1.0.4-1.el9.x86_64/usr/lib/systemd/system/mailproxy.service
...

Wrote: /build/RPMS/x86_64/mailproxy-1.0.4-1.el9.x86_64.rpm

==> 构建完成：dist/mailproxy-1.0.4-1.el9.x86_64.rpm
    安装：rpm -ivh dist/mailproxy-1.0.4-1.el9.x86_64.rpm
    升级：rpm -Uvh dist/mailproxy-1.0.4-1.el9.x86_64.rpm
```

**成功标志**:
- ✅ 最后一行显示 "构建完成"
- ✅ 显示了正确的 rpm 文件名
- ✅ shell 返回码为 0 (`echo $?`)

**产物位置**:
- `dist/mailproxy-1.0.4-1.el9.x86_64.rpm` (主包)
- `dist/rpmbuild/RPMS/x86_64/mailproxy-1.0.4-1.el9.x86_64.rpm` (原始位置)
- `dist/rpmbuild/` 临时目录 (可删除)

**时间预期**: 
- 首次构建：约 2-3 分钟 (需要下载 rpmbuild 依赖)
- 重复构建：约 30-60 秒 (所有依赖已缓存)

**常见问题处理**:

| 错误信息 | 原因 | 解决方案 |
|---|---|---|
| `error: No such file or directory` | Docker CLI 不可见或 PATH 不对 | `export PATH="$HOME/docker:$PATH"` |
| `failed to resolve reference` | Docker Hub 网络问题 | 使用镜像代理，或提前用`docker save`导出 |
| `architecture mismatch` | 使用了 arm64 镜像而非 amd64 | 确保 docker run 带 `--platform linux/amd64` |
| `go.mod not found` | 不在项目根目录 | `cd` 到 MailProxy 项目根目录 |
| `container exited abnormally` | Docker 资源不足 | 增加 `colima start --memory 8 --disk 40` |

---

#### ✅ 阶段四：验证 RPM 包功能

> ⚠️ **重要提示**: 
> 在 macOS ARM64 上使用 Docker 进行 x86_64 容器测试存在跨平台卷挂载限制。
> 建议在生产服务器上测试实际安装流程。

**Step 9: 本地元数据验证 **(推荐)

```
# 检查基本元数据
rpm -qip dist/mailproxy-1.0.4-1.el9.x86_64.rpm

# 检查包内容列表
rpm -qplf dist/mailproxy-1.0.4-1.el9.x86_64.rpm

# 列出所有文件及其权限
ls -lh dist/mailproxy-1.0.4-1.el9.x86_64.rpm
```

**预期成功输出**:
```
Name        : mailproxy
Version     : 1.0.4
Release     : 1.el9
Architecture: x86_64
Install Date: (not installed)
Group       : Unspecified
Size        : 9461599
License     : Proprietary
Summary     : MailProxy SMTP relay gateway
Description :
Go 实现的 SMTP 邮件代理网关：业务程序统一连接本代理发信，
代理转发到后端真实 SMTP 服务器，支持多账号配置与路由策略。
```

✅ **通过标准**:
- Version 正确 (与 VERSION 文件一致)
- Architecture 是 x86_64
- Size 约 9MB (9,461,599 bytes)
- Description 包含正确信息
- 无错误信息

**Step 10: 高级验证 - 在测试容器中模拟安装**

由于跨平台限制，此步骤仅在相同架构下可靠运行。如必须测试，参考以下方法:

```
# 需要先构建完整测试环境
cd testtools
docker build -f Dockerfile.testenv -t mailproxy-test-complete . --platform linux/amd64

# 然后拷贝文件到容器内测试 (而不是卷挂载)
docker run -d --name mp-test-via-copy mailproxy-test-complete sleep infinity
docker cp dist/mailproxy-*.rpm mp-test-via-copy:/tmp/
docker exec mp-test-via-copy bash -c "rpm -ivh /tmp/mailproxy-*.rpm"
docker stop mp-test-via-copy
```

⚠️ **注意**: 这种方法比卷挂载更可靠，因为避免了跨平台文件系统转换。

```

*图：MailProxy 作为中央枢纽，统一管理多个后端 SMTP 服务*

```

---

<br/>

## 功能特性

- 🚀 **高性能** - 基于 Go 并发模型，支持高并发连接
- 🔒 **安全认证** - IP 白名单 + 可选 AUTH 认证，防止被滥用为开放中继
- 🔄 **智能路由** - 根据发件人自动选择最优后端账号，提升送达率
- 🎯 **零代码改造** - 只需修改 SMTP 地址配置，现有发邮件逻辑完全不用变
- 🔐 **TLS 加密** - 支持 SMTP over SSL (465) 和 STARTTLS (587)，确保传输安全
- ♻️ **热重载** - 配置文件 SIGHUP 热加载，无需重启服务
- 📊 **监控指标** - Prometheus 指标导出，实时监控系统状态
- 📝 **详细日志** - 结构化日志记录，每次发信全流程追踪
- 🎁 **企业邮箱兼容** - 自动处理「信封发件人必须等于认证账号」限制（阿里/腾讯/网易等）
- 🏗️ **易部署** - 单二进制文件 + systemd 托管，提供 RPM 包一键安装

---

<br/>

## 快速开始

```
# 1. 下载最新版本 (Linux AMD64)
curl -LO https://github.com/YanGLweI/MailProxy/releases/latest/download/mailproxy-1.0.4-linux-amd64.tar.gz
tar -xzf mailproxy-*.tar.gz
chmod +x mailproxy

# 或使用 RPM 包 (RHEL/CentOS/Rocky)
curl -LO https://github.com/YanGLweI/MailProxy/releases/latest/download/mailproxy-1.0.4-1.el9.x86_64.rpm
sudo rpm -ivh mailproxy-1.0.4-1.el9.x86_64.rpm
```

**立即体验：**

```
# 生成自签名证书
./deploy/gen-cert.sh

# 准备配置
cp config.example.yaml config.yaml
vim config.yaml             # 填写后端邮箱账号等

# 启动服务
./mailproxy -config config.yaml
```

**业务接入只需改 3 个参数：**

**业务接入只需改 3 个参数：**

| 参数 | 修改前                    | 修改后                      |
|------|---------------------------|-----------------------------|
| SMTP 主机 | `smtp.qiye.aliyun.com`    | `mailproxy.internal:465`    |
| SSL 证书 | 逐个管理各服务商证书      | 只需信任 mailproxy 一个证书 |  
| 开发成本 | 不同服务商 API 差异适配    | 统一 SMTP 协议，一次开发     |

### 构建源码

如需从源码构建：

```
# 本地构建
go build -o mailproxy .

# 交叉编译 Linux AMD64
GOOS=linux GOARCH=amd64 go build -o mailproxy .

# 运行测试
go test ./...
```

### 连通性自测

**推荐先验证连接再发送测试邮件：**

```
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

```
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

```
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

### 🚀 快速构建 (macOS 本机即可)

```
bash deploy/build-rpm.sh        # 产物 dist/mailproxy-<version>-1.el*.x86_64.rpm
```

**前置要求:**
- ✅ Go 1.25+ `go version` → go1.25.0 darwin/arm64  
- ✅ Docker CLI (colima 或 Docker Desktop) `command -v docker`
- ✅ colima 已运行 `limactl status` → running
- ✅ Rocky Linux 9 amd64 镜像已拉取 `docker images | grep rockylinux`

> 💡 **说明**: 如果 `build-rpm.sh` 执行失败，请先确保所有前置要求都满足。

---

### ⚠️ 重要注意事项

#### 问题 1: Docker Hub 认证超时
**现象**: `failed to authorize: failed to fetch anonymous token`  
**原因**: 国内网络访问 Docker Hub 受限  
**解决**: 配置镜像代理或使用其他镜像站

**方法 A: 使用 DaoCloud 镜像加速**
```
export REGISTRY_MIRROR=https://docker.m.daocloud.io
docker pull --platform linux/amd64 rockylinux/rockylinux:9
```

**方法 B: 提前在服务器拉取并导出导入**
```
# macOS 本地
docker save rockylinux/rockylinux:9 > rockylinux.tar

# 上传到服务器
scp rockylinux.tar server:/tmp/

# 服务器上加载
docker load < /tmp/rockylinux.tar
```

#### 问题 2: macOS ARM64 的卷挂载限制  
**现象**: 跨平台卷挂载导致文件不可见  
**原因**: Docker Desktop 在 arm64 host 上无法完全支持 x86_64 容器的卷挂载  

**解决方案**:  
1. 构建阶段使用正确的 `--platform linux/amd64`
2. 验证阶段改用 `rpm -qip` 检查元数据而非安装测试
3. 生产环境直接上传 RPM 到目标服务器安装

#### 问题 3: RockyLinux 基础镜像无 systemd  
**现象**: 测试容器无法启动 systemd  
**原因**: 官方镜像最小化设计  

**解决方案**: 使用完整测试 Dockerfile 或仅做元数据验证

---

### 📦 详细构建流程与每步操作指南

如需从头完整了解如何从 macOS 构建 x86_64 Linux RPM 包，请参考以下步骤:

#### 🔧 前置环境与依赖

| 组件 | 要求 | 检查命令 |
|---|---|---|
| macOS 芯片 | Apple Silicon (M1/M2/M3) 或 Intel | `uname -m` → arm64 / x86_64 |
| Go | ≥1.25.0 | `go version` |
| Homebrew (可选) | 用于安装 lima | `/usr/bin/ruby -e "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/master/install)"` (未装过才需要) |
| curl/tar/git | 系统自带 | `which curl tar git` |
| 网络 | 可访问 GitHub/Docker Hub(镜像站优先) | `ping github.com` |

> ⚠️ **重要提示**: 
> - ARM64 Mac 需要下载对应版本的 Docker CLI(见 Step 3)
> - 推荐使用 [colima](https://github.com/abiosoft/colima) 替代 Docker Desktop  
> - Docker Hub 直连可能超时，建议配置镜像代理

#### 🏗️ 阶段一：搭建临时容器环境（约 5 分钟）

**Step 1: 下载 colima 二进制 **(ARM64)

```
mkdir -p ~/bin && \
curl -sL -o ~/bin/colima \
    https://github.com/abiosoft/colima/releases/latest/download/colima-Darwin-arm64 && \
chmod +x ~/bin/colima && \
~/bin/colima version
```

**输出示例:**
```
colima version v0.10.3
git commit: ...
```

**Step 2: 安装 limactl **(虚拟化层)

```
brew install lima
limactl --version
```

**输出示例:**
```
limactl version 2.2.0
```

**Step 3: 下载 Docker CLI 静态二进制**

```
mkdir -p ~/docker && \
curl -sL -o /tmp/docker.tgz \
    https://download.docker.com/mac/static/stable/aarch64/docker-28.3.3.tgz && \
tar -xzf /tmp/docker.tgz -C /tmp && \
cp /tmp/docker/docker ~/docker/ && \
~/docker/docker version
```

**输出示例:**
```
Client: Docker Engine - Community
 Version:           27.5.1-rc1
...
```

**替代方案：使用 Homebrew 安装 **(如果已配置)
```
# 方法 A: 检查 colima 自带的 docker 命令
colima status
# 如果显示 running，则 ~/.local/bin/docker 应该可用

# 方法 B: 导出 docker socket
export DOCKER_HOST=unix://$HOME/.colima/default/docker.sock
docker info
```

**Step 4: 启动 colima 虚拟机并加载镜像**

```
export PATH="$HOME/docker:$PATH"   # 让 colima 找到 docker
colima start --cpu 2 --memory 4 --disk 20
```

等待输出:`READY. Run 'limactl shell colima' to open the shell.`

**验证 Docker 工作正常:**
```
limactl status colima
# 应该看到：running

docker images
# 此时可以没有任何镜像，后续会拉取

docker run --rm hello-world
# 测试运行一个小容器
```

**Step 5: 拉取构建所需镜像 **(指定 amd64 平台)

```
docker pull --platform linux/amd64 rockylinux/rockylinux:9
```

**提示**:如果直连超时，使用镜像代理:
```
docker pull --platform linux/amd64 docker.m.daocloud.io/rockylinux/rockylinux:9
```

**验证镜像已成功拉取**:
```
docker images | grep rockylinux
# 应该看到:
# rockylinux/rockylinux     9      <IMAGE-ID>   <DATE>   710MB
# docker.m.daocloud.io/rockylinux/rockylinux   9    <same-ID>
```

---

#### 📝 阶段二：准备源码与构建脚本

**Step 6: 打开项目目录**

```
cd /Users/yeung/Projects/MailProxy   # 替换为你的项目路径
cat VERSION                          # 确认版本号 (如 1.0.4)
```

**Step 7: 查看核心构建脚本**

```
cat deploy/build-rpm.sh
```

**关键流程**:

1. 读取 VERSION 文件中的版本号
2. `CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build ...` → 交叉编译 Linux x86_64 静态二进制
3. 将 spec/specs/source 复制到 `dist/rpmbuild/SOURCES/`
4. `docker run --platform linux/amd64 rockylinux:9 rpmbuild ...` → 在容器中执行 rpm 构建

**rpmbuild 目录结构详解**:
```
dist/rpmbuild/
├── SOURCES/          # 源代码文件 (binary + config + service)
│   ├── mailproxy     # Go 编译的 x86_64 二进制
│   ├── config.yaml   # 默认配置文件  
│   ├── mailproxy.service  # systemd unit 模板
│   └── README.md     # 产品说明文档
├── SPECS/            # RPM spec 文件定义包结构
│   └── mailproxy.spec
├── BUILD/            # 编译工作目录 (空目录)
├── BUILDROOT/        # 安装包根目录 (包含安装后的文件布局)
├── RPMS/x86_64/      # 最终生成的 RPM 包
│   └── mailproxy-1.0.4-1.el9.x86_64.rpm
└── SRPMS/            # 源码 RPM (如果有)
    └── mailproxy-1.0.4-1.el9.src.rpm
```

---

#### 🎯 阶段三：一键构建 RPM 包

**Step 8: 直接运行构建脚本**

```
export PATH="$HOME/docker:$PATH"    # 确保找到 docker
bash deploy/build-rpm.sh
```

**完整输出示例**(优化版):
```
==> 交叉编译 Linux x86_64 二进制 (v1.0.4)
==> 准备打包源文件
==> Docker 内执行 rpmbuild (rockylinux/rockylinux:9, linux/amd64)

Installed:
  rpm-build-4.16.1.3-40.el9.x86_64
  ... (更多依赖包)

Executing(%prep): /bin/sh -e /var/tmp/rpm-tmp.XXX
+ umask 022
+ cd /build/BUILD
+ rm -rf mailproxy-1.0.4
...

Executing(%install): /bin/sh -e /var/tmp/rpm-tmp.YYY
+ umask 022
+ '[' /build/BUILDROOT/mailproxy-1.0.4-1.el9.x86_64 '!=' / ']'
...
+ install -Dpm 0755 /build/SOURCES/mailproxy /build/BUILDROOT/mailproxy-1.0.4-1.el9.x86_64/usr/bin/mailproxy
+ install -Dpm 0644 /build/SOURCES/mailproxy.service /build/BUILDROOT/mailproxy-1.0.4-1.el9.x86_64/usr/lib/systemd/system/mailproxy.service
...

Wrote: /build/RPMS/x86_64/mailproxy-1.0.4-1.el9.x86_64.rpm

==> 构建完成：dist/mailproxy-1.0.4-1.el9.x86_64.rpm
    安装：rpm -ivh dist/mailproxy-1.0.4-1.el9.x86_64.rpm
    升级：rpm -Uvh dist/mailproxy-1.0.4-1.el9.x86_64.rpm
```

**成功标志**:
- ✅ 最后一行显示 "构建完成"
- ✅ 显示了正确的 rpm 文件名
- ✅ shell 返回码为 0 (`echo $?`)

**产物位置**:
- `dist/mailproxy-1.0.4-1.el9.x86_64.rpm` (主包)
- `dist/rpmbuild/RPMS/x86_64/mailproxy-1.0.4-1.el9.x86_64.rpm` (原始位置)
- `dist/rpmbuild/` 临时目录 (可删除)

**时间预期**: 
- 首次构建：约 2-3 分钟 (需要下载 rpmbuild 依赖)
- 重复构建：约 30-60 秒 (所有依赖已缓存)

**常见问题处理**:

| 错误信息 | 原因 | 解决方案 |
|---|---|---|
| `error: No such file or directory` | Docker CLI 不可见或 PATH 不对 | `export PATH="$HOME/docker:$PATH"` |
| `failed to resolve reference` | Docker Hub 网络问题 | 使用镜像代理，或提前用`docker save`导出 |
| `architecture mismatch` | 使用了 arm64 镜像而非 amd64 | 确保 docker run 带 `--platform linux/amd64` |
| `go.mod not found` | 不在项目根目录 | `cd` 到 MailProxy 项目根目录 |
| `container exited abnormally` | Docker 资源不足 | 增加 `colima start --memory 8 --disk 40` |

---

#### ✅ 阶段四：验证 RPM 包功能

> ⚠️ **重要提示**: 
> 在 macOS ARM64 上使用 Docker 进行 x86_64 容器测试存在跨平台卷挂载限制。
> 建议在生产服务器上测试实际安装流程。

**Step 9: 本地元数据验证 **(推荐)

```
# 检查基本元数据
rpm -qip dist/mailproxy-1.0.4-1.el9.x86_64.rpm

# 检查包内容列表
rpm -qplf dist/mailproxy-1.0.4-1.el9.x86_64.rpm

# 列出所有文件及其权限
ls -lh dist/mailproxy-1.0.4-1.el9.x86_64.rpm
```

**预期成功输出**:
```
Name        : mailproxy
Version     : 1.0.4
Release     : 1.el9
Architecture: x86_64
Install Date: (not installed)
Group       : Unspecified
Size        : 9461599
License     : Proprietary
Summary     : MailProxy SMTP relay gateway
Description :
Go 实现的 SMTP 邮件代理网关：业务程序统一连接本代理发信，
代理转发到后端真实 SMTP 服务器，支持多账号配置与路由策略。
```

✅ **通过标准**:
- Version 正确 (与 VERSION 文件一致)
- Architecture 是 x86_64
- Size 约 9MB (9,461,599 bytes)
- Description 包含正确信息
- 无错误信息

**Step 10: 高级验证 - 在测试容器中模拟安装**

由于跨平台限制，此步骤仅在相同架构下可靠运行。如必须测试，参考以下方法:

```
# 需要先构建完整测试环境
cd testtools
docker build -f Dockerfile.testenv -t mailproxy-test-complete . --platform linux/amd64

# 然后拷贝文件到容器内测试 (而不是卷挂载)
docker run -d --name mp-test-via-copy mailproxy-test-complete sleep infinity
docker cp dist/mailproxy-*.rpm mp-test-via-copy:/tmp/
docker exec mp-test-via-copy bash -c "rpm -ivh /tmp/mailproxy-*.rpm"
docker stop mp-test-via-copy
```

⚠️ **注意**: 这种方法比卷挂载更可靠，因为避免了跨平台文件系统转换。

```

*图：MailProxy 作为中央枢纽，统一管理多个后端 SMTP 服务*

```

---

<br/>

## 功能特性

- 🚀 **高性能** - 基于 Go 并发模型，支持高并发连接
- 🔒 **安全认证** - IP 白名单 + 可选 AUTH 认证，防止被滥用为开放中继
- 🔄 **智能路由** - 根据发件人自动选择最优后端账号，提升送达率
- 🎯 **零代码改造** - 只需修改 SMTP 地址配置，现有发邮件逻辑完全不用变
- 🔐 **TLS 加密** - 支持 SMTP over SSL (465) 和 STARTTLS (587)，确保传输安全
- ♻️ **热重载** - 配置文件 SIGHUP 热加载，无需重启服务
- 📊 **监控指标** - Prometheus 指标导出，实时监控系统状态
- 📝 **详细日志** - 结构化日志记录，每次发信全流程追踪
- 🎁 **企业邮箱兼容** - 自动处理「信封发件人必须等于认证账号」限制（阿里/腾讯/网易等）
- 🏗️ **易部署** - 单二进制文件 + systemd 托管，提供 RPM 包一键安装

---

<br/>

## 快速开始

```
# 1. 下载最新版本 (Linux AMD64)
curl -LO https://github.com/YanGLweI/MailProxy/releases/latest/download/mailproxy-1.0.4-linux-amd64.tar.gz
tar -xzf mailproxy-*.tar.gz
chmod +x mailproxy

# 或使用 RPM 包 (RHEL/CentOS/Rocky)
curl -LO https://github.com/YanGLweI/MailProxy/releases/latest/download/mailproxy-1.0.4-1.el9.x86_64.rpm
sudo rpm -ivh mailproxy-1.0.4-1.el9.x86_64.rpm
```

**立即体验：**

```
# 生成自签名证书
./deploy/gen-cert.sh

# 准备配置
cp config.example.yaml config.yaml
vim config.yaml             # 填写后端邮箱账号等

# 启动服务
./mailproxy -config config.yaml
```

**业务接入只需改 3 个参数：**

**业务接入只需改 3 个参数：**

| 参数 | 修改前                    | 修改后                      |
|------|---------------------------|-----------------------------|
| SMTP 主机 | `smtp.qiye.aliyun.com`    | `mailproxy.internal:465`    |
| SSL 证书 | 逐个管理各服务商证书      | 只需信任 mailproxy 一个证书 |  
| 开发成本 | 不同服务商 API 差异适配    | 统一 SMTP 协议，一次开发     |

### 构建源码

如需从源码构建：

```
# 本地构建
go build -o mailproxy .

# 交叉编译 Linux AMD64
GOOS=linux GOARCH=amd64 go build -o mailproxy .

# 运行测试
go test ./...
```

### 连通性自测

**推荐先验证连接再发送测试邮件：**

```
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

```
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

```
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

### 🚀 快速构建 (macOS 本机即可)

```
bash deploy/build-rpm.sh        # 产物 dist/mailproxy-<version>-1.el*.x86_64.rpm
```

**前置要求:**
- ✅ Go 1.25+ `go version` → go1.25.0 darwin/arm64  
- ✅ Docker CLI (colima 或 Docker Desktop) `command -v docker`
- ✅ colima 已运行 `limactl status` → running
- ✅ Rocky Linux 9 amd64 镜像已拉取 `docker images | grep rockylinux`

> 💡 **说明**: 如果 `build-rpm.sh` 执行失败，请先确保所有前置要求都满足。

---

### ⚠️ 重要注意事项

#### 问题 1: Docker Hub 认证超时
**现象**: `failed to authorize: failed to fetch anonymous token`  
**原因**: 国内网络访问 Docker Hub 受限  
**解决**: 配置镜像代理或使用其他镜像站

**方法 A: 使用 DaoCloud 镜像加速**
```
export REGISTRY_MIRROR=https://docker.m.daocloud.io
docker pull --platform linux/amd64 rockylinux/rockylinux:9
```

**方法 B: 提前在服务器拉取并导出导入**
```
# macOS 本地
docker save rockylinux/rockylinux:9 > rockylinux.tar

# 上传到服务器
scp rockylinux.tar server:/tmp/

# 服务器上加载
docker load < /tmp/rockylinux.tar
```

#### 问题 2: macOS ARM64 的卷挂载限制  
**现象**: 跨平台卷挂载导致文件不可见  
**原因**: Docker Desktop 在 arm64 host 上无法完全支持 x86_64 容器的卷挂载  

**解决方案**:  
1. 构建阶段使用正确的 `--platform linux/amd64`
2. 验证阶段改用 `rpm -qip` 检查元数据而非安装测试
3. 生产环境直接上传 RPM 到目标服务器安装

#### 问题 3: RockyLinux 基础镜像无 systemd  
**现象**: 测试容器无法启动 systemd  
**原因**: 官方镜像最小化设计  

**解决方案**: 使用完整测试 Dockerfile 或仅做元数据验证

---

### 📦 详细构建流程与每步操作指南

如需从头完整了解如何从 macOS 构建 x86_64 Linux RPM 包，请参考以下步骤:

#### 🔧 前置环境与依赖

| 组件 | 要求 | 检查命令 |
|---|---|---|
| macOS 芯片 | Apple Silicon (M1/M2/M3) 或 Intel | `uname -m` → arm64 / x86_64 |
| Go | ≥1.25.0 | `go version` |
| Homebrew (可选) | 用于安装 lima | `/usr/bin/ruby -e "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/master/install)"` (未装过才需要) |
| curl/tar/git | 系统自带 | `which curl tar git` |
| 网络 | 可访问 GitHub/Docker Hub(镜像站优先) | `ping github.com` |

> ⚠️ **重要提示**: 
> - ARM64 Mac 需要下载对应版本的 Docker CLI(见 Step 3)
> - 推荐使用 [colima](https://github.com/abiosoft/colima) 替代 Docker Desktop  
> - Docker Hub 直连可能超时，建议配置镜像代理

#### 🏗️ 阶段一：搭建临时容器环境（约 5 分钟）

**Step 1: 下载 colima 二进制 **(ARM64)

```
mkdir -p ~/bin && \
curl -sL -o ~/bin/colima \
    https://github.com/abiosoft/colima/releases/latest/download/colima-Darwin-arm64 && \
chmod +x ~/bin/colima && \
~/bin/colima version
```

**输出示例:**
```
colima version v0.10.3
git commit: ...
```

**Step 2: 安装 limactl **(虚拟化层)

```
brew install lima
limactl --version
```

**输出示例:**
```
limactl version 2.2.0
```

**Step 3: 下载 Docker CLI 静态二进制**

```
mkdir -p ~/docker && \
curl -sL -o /tmp/docker.tgz \
    https://download.docker.com/mac/static/stable/aarch64/docker-28.3.3.tgz && \
tar -xzf /tmp/docker.tgz -C /tmp && \
cp /tmp/docker/docker ~/docker/ && \
~/docker/docker version
```

**输出示例:**
```
Client: Docker Engine - Community
 Version:           27.5.1-rc1
...
```

**替代方案：使用 Homebrew 安装 **(如果已配置)
```
# 方法 A: 检查 colima 自带的 docker 命令
colima status
# 如果显示 running，则 ~/.local/bin/docker 应该可用

# 方法 B: 导出 docker socket
export DOCKER_HOST=unix://$HOME/.colima/default/docker.sock
docker info
```

**Step 4: 启动 colima 虚拟机并加载镜像**

```
export PATH="$HOME/docker:$PATH"   # 让 colima 找到 docker
colima start --cpu 2 --memory 4 --disk 20
```

等待输出:`READY. Run 'limactl shell colima' to open the shell.`

**验证 Docker 工作正常:**
```
limactl status colima
# 应该看到：running

docker images
# 此时可以没有任何镜像，后续会拉取

docker run --rm hello-world
# 测试运行一个小容器
```

**Step 5: 拉取构建所需镜像 **(指定 amd64 平台)

```
docker pull --platform linux/amd64 rockylinux/rockylinux:9
```

**提示**:如果直连超时，使用镜像代理:
```
docker pull --platform linux/amd64 docker.m.daocloud.io/rockylinux/rockylinux:9
```

**验证镜像已成功拉取**:
```
docker images | grep rockylinux
# 应该看到:
# rockylinux/rockylinux     9      <IMAGE-ID>   <DATE>   710MB
# docker.m.daocloud.io/rockylinux/rockylinux   9    <same-ID>
```

---

#### 📝 阶段二：准备源码与构建脚本

**Step 6: 打开项目目录**

```
cd /Users/yeung/Projects/MailProxy   # 替换为你的项目路径
cat VERSION                          # 确认版本号 (如 1.0.4)
```

**Step 7: 查看核心构建脚本**

```
cat deploy/build-rpm.sh
```

**关键流程**:

1. 读取 VERSION 文件中的版本号
2. `CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build ...` → 交叉编译 Linux x86_64 静态二进制
3. 将 spec/specs/source 复制到 `dist/rpmbuild/SOURCES/`
4. `docker run --platform linux/amd64 rockylinux:9 rpmbuild ...` → 在容器中执行 rpm 构建

**rpmbuild 目录结构详解**:
```
dist/rpmbuild/
├── SOURCES/          # 源代码文件 (binary + config + service)
│   ├── mailproxy     # Go 编译的 x86_64 二进制
│   ├── config.yaml   # 默认配置文件  
│   ├── mailproxy.service  # systemd unit 模板
│   └── README.md     # 产品说明文档
├── SPECS/            # RPM spec 文件定义包结构
│   └── mailproxy.spec
├── BUILD/            # 编译工作目录 (空目录)
├── BUILDROOT/        # 安装包根目录 (包含安装后的文件布局)
├── RPMS/x86_64/      # 最终生成的 RPM 包
│   └── mailproxy-1.0.4-1.el9.x86_64.rpm
└── SRPMS/            # 源码 RPM (如果有)
    └── mailproxy-1.0.4-1.el9.src.rpm
```

---

#### 🎯 阶段三：一键构建 RPM 包

**Step 8: 直接运行构建脚本**

```
export PATH="$HOME/docker:$PATH"    # 确保找到 docker
bash deploy/build-rpm.sh
```

**完整输出示例**(优化版):
```
==> 交叉编译 Linux x86_64 二进制 (v1.0.4)
==> 准备打包源文件
==> Docker 内执行 rpmbuild (rockylinux/rockylinux:9, linux/amd64)

Installed:
  rpm-build-4.16.1.3-40.el9.x86_64
  ... (更多依赖包)

Executing(%prep): /bin/sh -e /var/tmp/rpm-tmp.XXX
+ umask 022
+ cd /build/BUILD
+ rm -rf mailproxy-1.0.4
...

Executing(%install): /bin/sh -e /var/tmp/rpm-tmp.YYY
+ umask 022
+ '[' /build/BUILDROOT/mailproxy-1.0.4-1.el9.x86_64 '!=' / ']'
...
+ install -Dpm 0755 /build/SOURCES/mailproxy /build/BUILDROOT/mailproxy-1.0.4-1.el9.x86_64/usr/bin/mailproxy
+ install -Dpm 0644 /build/SOURCES/mailproxy.service /build/BUILDROOT/mailproxy-1.0.4-1.el9.x86_64/usr/lib/systemd/system/mailproxy.service
...

Wrote: /build/RPMS/x86_64/mailproxy-1.0.4-1.el9.x86_64.rpm

==> 构建完成：dist/mailproxy-1.0.4-1.el9.x86_64.rpm
    安装：rpm -ivh dist/mailproxy-1.0.4-1.el9.x86_64.rpm
    升级：rpm -Uvh dist/mailproxy-1.0.4-1.el9.x86_64.rpm
```

**成功标志**:
- ✅ 最后一行显示 "构建完成"
- ✅ 显示了正确的 rpm 文件名
- ✅ shell 返回码为 0 (`echo $?`)

**产物位置**:
- `dist/mailproxy-1.0.4-1.el9.x86_64.rpm` (主包)
- `dist/rpmbuild/RPMS/x86_64/mailproxy-1.0.4-1.el9.x86_64.rpm` (原始位置)
- `dist/rpmbuild/` 临时目录 (可删除)

**时间预期**: 
- 首次构建：约 2-3 分钟 (需要下载 rpmbuild 依赖)
- 重复构建：约 30-60 秒 (所有依赖已缓存)

**常见问题处理**:

| 错误信息 | 原因 | 解决方案 |
|---|---|---|
| `error: No such file or directory` | Docker CLI 不可见或 PATH 不对 | `export PATH="$HOME/docker:$PATH"` |
| `failed to resolve reference` | Docker Hub 网络问题 | 使用镜像代理，或提前用`docker save`导出 |
| `architecture mismatch` | 使用了 arm64 镜像而非 amd64 | 确保 docker run 带 `--platform linux/amd64` |
| `go.mod not found` | 不在项目根目录 | `cd` 到 MailProxy 项目根目录 |
| `container exited abnormally` | Docker 资源不足 | 增加 `colima start --memory 8 --disk 40` |

---

#### ✅ 阶段四：验证 RPM 包功能

> ⚠️ **重要提示**: 
> 在 macOS ARM64 上使用 Docker 进行 x86_64 容器测试存在跨平台卷挂载限制。
> 建议在生产服务器上测试实际安装流程。

**Step 9: 本地元数据验证 **(推荐)

```
# 检查基本元数据
rpm -qip dist/mailproxy-1.0.4-1.el9.x86_64.rpm

# 检查包内容列表
rpm -qplf dist/mailproxy-1.0.4-1.el9.x86_64.rpm

# 列出所有文件及其权限
ls -lh dist/mailproxy-1.0.4-1.el9.x86_64.rpm
```

**预期成功输出**:
```
Name        : mailproxy
Version     : 1.0.4
Release     : 1.el9
Architecture: x86_64
Install Date: (not installed)
Group       : Unspecified
Size        : 9461599
License     : Proprietary
Summary     : MailProxy SMTP relay gateway
Description :
Go 实现的 SMTP 邮件代理网关：业务程序统一连接本代理发信，
代理转发到后端真实 SMTP 服务器，支持多账号配置与路由策略。
```

✅ **通过标准**:
- Version 正确 (与 VERSION 文件一致)
- Architecture 是 x86_64
- Size 约 9MB (9,461,599 bytes)
- Description 包含正确信息
- 无错误信息

**Step 10: 高级验证 - 在测试容器中模拟安装**

由于跨平台限制，此步骤仅在相同架构下可靠运行。如必须测试，参考以下方法:

```
# 需要先构建完整测试环境
cd testtools
docker build -f Dockerfile.testenv -t mailproxy-test-complete . --platform linux/amd64

# 然后拷贝文件到容器内测试 (而不是卷挂载)
docker run -d --name mp-test-via-copy mailproxy-test-complete sleep infinity
docker cp dist/mailproxy-*.rpm mp-test-via-copy:/tmp/
docker exec mp-test-via-copy bash -c "rpm -ivh /tmp/mailproxy-*.rpm"
docker stop mp-test-via-copy
```

⚠️ **注意**: 这种方法比卷挂载更可靠，因为避免了跨平台文件系统转换。

```

*图：MailProxy 作为中央枢纽，统一管理多个后端 SMTP 服务*

```

---

<br/>

## 功能特性

- 🚀 **高性能** - 基于 Go 并发模型，支持高并发连接
- 🔒 **安全认证** - IP 白名单 + 可选 AUTH 认证，防止被滥用为开放中继
- 🔄 **智能路由** - 根据发件人自动选择最优后端账号，提升送达率
- 🎯 **零代码改造** - 只需修改 SMTP 地址配置，现有发邮件逻辑完全不用变
- 🔐 **TLS 加密** - 支持 SMTP over SSL (465) 和 STARTTLS (587)，确保传输安全
- ♻️ **热重载** - 配置文件 SIGHUP 热加载，无需重启服务
- 📊 **监控指标** - Prometheus 指标导出，实时监控系统状态
- 📝 **详细日志** - 结构化日志记录，每次发信全流程追踪
- 🎁 **企业邮箱兼容** - 自动处理「信封发件人必须等于认证账号」限制（阿里/腾讯/网易等）
- 🏗️ **易部署** - 单二进制文件 + systemd 托管，提供 RPM 包一键安装

---

<br/>

## 快速开始

```
# 1. 下载最新版本 (Linux AMD64)
curl -LO https://github.com/YanGLweI/MailProxy/releases/latest/download/mailproxy-1.0.4-linux-amd64.tar.gz
tar -xzf mailproxy-*.tar.gz
chmod +x mailproxy

# 或使用 RPM 包 (RHEL/CentOS/Rocky)
curl -LO https://github.com/YanGLweI/MailProxy/releases/latest/download/mailproxy-1.0.4-1.el9.x86_64.rpm
sudo rpm -ivh mailproxy-1.0.4-1.el9.x86_64.rpm
```

**立即体验：**

```
# 生成自签名证书
./deploy/gen-cert.sh

# 准备配置
cp config.example.yaml config.yaml
vim config.yaml             # 填写后端邮箱账号等

# 启动服务
./mailproxy -config config.yaml
```

**业务接入只需改 3 个参数：**

**业务接入只需改 3 个参数：**

| 参数 | 修改前                    | 修改后                      |
|------|---------------------------|-----------------------------|
| SMTP 主机 | `smtp.qiye.aliyun.com`    | `mailproxy.internal:465`    |
| SSL 证书 | 逐个管理各服务商证书      | 只需信任 mailproxy 一个证书 |  
| 开发成本 | 不同服务商 API 差异适配    | 统一 SMTP 协议，一次开发     |

### 构建源码

如需从源码构建：

```
# 本地构建
go build -o mailproxy .

# 交叉编译 Linux AMD64
GOOS=linux GOARCH=amd64 go build -o mailproxy .

# 运行测试
go test ./...
```

### 连通性自测

**推荐先验证连接再发送测试邮件：**

```
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

```
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

```
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

### 🚀 快速构建 (macOS 本机即可)

```
bash deploy/build-rpm.sh        # 产物 dist/mailproxy-<version>-1.el*.x86_64.rpm
```

**前置要求:**
- ✅ Go 1.25+ `go version` → go1.25.0 darwin/arm64  
- ✅ Docker CLI (colima 或 Docker Desktop) `command -v docker`
- ✅ colima 已运行 `limactl status` → running
- ✅ Rocky Linux 9 amd64 镜像已拉取 `docker images | grep rockylinux`

> 💡 **说明**: 如果 `build-rpm.sh` 执行失败，请先确保所有前置要求都满足。

---

### ⚠️ 重要注意事项

#### 问题 1: Docker Hub 认证超时
**现象**: `failed to authorize: failed to fetch anonymous token`  
**原因**: 国内网络访问 Docker Hub 受限  
**解决**: 配置镜像代理或使用其他镜像站

**方法 A: 使用 DaoCloud 镜像加速**
```
export REGISTRY_MIRROR=https://docker.m.daocloud.io
docker pull --platform linux/amd64 rockylinux/rockylinux:9
```

**方法 B: 提前在服务器拉取并导出导入**
```
# macOS 本地
docker save rockylinux/rockylinux:9 > rockylinux.tar

# 上传到服务器
scp rockylinux.tar server:/tmp/

# 服务器上加载
docker load < /tmp/rockylinux.tar
```

#### 问题 2: macOS ARM64 的卷挂载限制  
**现象**: 跨平台卷挂载导致文件不可见  
**原因**: Docker Desktop 在 arm64 host 上无法完全支持 x86_64 容器的卷挂载  

**解决方案**:  
1. 构建阶段使用正确的 `--platform linux/amd64`
2. 验证阶段改用 `rpm -qip` 检查元数据而非安装测试
3. 生产环境直接上传 RPM 到目标服务器安装

#### 问题 3: RockyLinux 基础镜像无 systemd  
**现象**: 测试容器无法启动 systemd  
**原因**: 官方镜像最小化设计  

**解决方案**: 使用完整测试 Dockerfile 或仅做元数据验证

---

### 📦 详细构建流程与每步操作指南

如需从头完整了解如何从 macOS 构建 x86_64 Linux RPM 包，请参考以下步骤:

#### 🔧 前置环境与依赖

| 组件 | 要求 | 检查命令 |
|---|---|---|
| macOS 芯片 | Apple Silicon (M1/M2/M3) 或 Intel | `uname -m` → arm64 / x86_64 |
| Go | ≥1.25.0 | `go version` |
| Homebrew (可选) | 用于安装 lima | `/usr/bin/ruby -e "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/master/install)"` (未装过才需要) |
| curl/tar/git | 系统自带 | `which curl tar git` |
| 网络 | 可访问 GitHub/Docker Hub(镜像站优先) | `ping github.com` |

> ⚠️ **重要提示**: 
> - ARM64 Mac 需要下载对应版本的 Docker CLI(见 Step 3)
> - 推荐使用 [colima](https://github.com/abiosoft/colima) 替代 Docker Desktop  
> - Docker Hub 直连可能超时，建议配置镜像代理

#### 🏗️ 阶段一：搭建临时容器环境（约 5 分钟）

**Step 1: 下载 colima 二进制 **(ARM64)

```
mkdir -p ~/bin && \
curl -sL -o ~/bin/colima \
    https://github.com/abiosoft/colima/releases/latest/download/colima-Darwin-arm64 && \
chmod +x ~/bin/colima && \
~/bin/colima version
```

**输出示例:**
```
colima version v0.10.3
git commit: ...
```

**Step 2: 安装 limactl **(虚拟化层)

```
brew install lima
limactl --version
```

**输出示例:**
```
limactl version 2.2.0
```

**Step 3: 下载 Docker CLI 静态二进制**

```
mkdir -p ~/docker && \
curl -sL -o /tmp/docker.tgz \
    https://download.docker.com/mac/static/stable/aarch64/docker-28.3.3.tgz && \
tar -xzf /tmp/docker.tgz -C /tmp && \
cp /tmp/docker/docker ~/docker/ && \
~/docker/docker version
```

**输出示例:**
```
Client: Docker Engine - Community
 Version:           27.5.1-rc1
...
```

**替代方案：使用 Homebrew 安装 **(如果已配置)
```
# 方法 A: 检查 colima 自带的 docker 命令
colima status
# 如果显示 running，则 ~/.local/bin/docker 应该可用

# 方法 B: 导出 docker socket
export DOCKER_HOST=unix://$HOME/.colima/default/docker.sock
docker info
```

**Step 4: 启动 colima 虚拟机并加载镜像**

```
export PATH="$HOME/docker:$PATH"   # 让 colima 找到 docker
colima start --cpu 2 --memory 4 --disk 20
```

等待输出:`READY. Run 'limactl shell colima' to open the shell.`

**验证 Docker 工作正常:**
```
limactl status colima
# 应该看到：running

docker images
# 此时可以没有任何镜像，后续会拉取

docker run --rm hello-world
# 测试运行一个小容器
```

**Step 5: 拉取构建所需镜像 **(指定 amd64 平台)

```
docker pull --platform linux/amd64 rockylinux/rockylinux:9
```

**提示**:如果直连超时，使用镜像代理:
```
docker pull --platform linux/amd64 docker.m.daocloud.io/rockylinux/rockylinux:9
```

**验证镜像已成功拉取**:
```
docker images | grep rockylinux
# 应该看到:
# rockylinux/rockylinux     9      <IMAGE-ID>   <DATE>   710MB
# docker.m.daocloud.io/rockylinux/rockylinux   9    <same-ID>
```

---

#### 📝 阶段二：准备源码与构建脚本

**Step 6: 打开项目目录**

```
cd /Users/yeung/Projects/MailProxy   # 替换为你的项目路径
cat VERSION                          # 确认版本号 (如 1.0.4)
```

**Step 7: 查看核心构建脚本**

```
cat deploy/build-rpm.sh
```

**关键流程**:

1. 读取 VERSION 文件中的版本号
2. `CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build ...` → 交叉编译 Linux x86_64 静态二进制
3. 将 spec/specs/source 复制到 `dist/rpmbuild/SOURCES/`
4. `docker run --platform linux/amd64 rockylinux:9 rpmbuild ...` → 在容器中执行 rpm 构建

**rpmbuild 目录结构详解**:
```
dist/rpmbuild/
├── SOURCES/          # 源代码文件 (binary + config + service)
│   ├── mailproxy     # Go 编译的 x86_64 二进制
│   ├── config.yaml   # 默认配置文件  
│   ├── mailproxy.service  # systemd unit 模板
│   └── README.md     # 产品说明文档
├── SPECS/            # RPM spec 文件定义包结构
│   └── mailproxy.spec
├── BUILD/            # 编译工作目录 (空目录)
├── BUILDROOT/        # 安装包根目录 (包含安装后的文件布局)
├── RPMS/x86_64/      # 最终生成的 RPM 包
│   └── mailproxy-1.0.4-1.el9.x86_64.rpm
└── SRPMS/            # 源码 RPM (如果有)
    └── mailproxy-1.0.4-1.el9.src.rpm
```

---

#### 🎯 阶段三：一键构建 RPM 包

**Step 8: 直接运行构建脚本**

```
export PATH="$HOME/docker:$PATH"    # 确保找到 docker
bash deploy/build-rpm.sh
```

**完整输出示例**(优化版):
```
==> 交叉编译 Linux x86_64 二进制 (v1.0.4)
==> 准备打包源文件
==> Docker 内执行 rpmbuild (rockylinux/rockylinux:9, linux/amd64)

Installed:
  rpm-build-4.16.1.3-40.el9.x86_64
  ... (更多依赖包)

Executing(%prep): /bin/sh -e /var/tmp/rpm-tmp.XXX
+ umask 022
+ cd /build/BUILD
+ rm -rf mailproxy-1.0.4
...

Executing(%install): /bin/sh -e /var/tmp/rpm-tmp.YYY
+ umask 022
+ '[' /build/BUILDROOT/mailproxy-1.0.4-1.el9.x86_64 '!=' / ']'
...
+ install -Dpm 0755 /build/SOURCES/mailproxy /build/BUILDROOT/mailproxy-1.0.4-1.el9.x86_64/usr/bin/mailproxy
+ install -Dpm 0644 /build/SOURCES/mailproxy.service /build/BUILDROOT/mailproxy-1.0.4-1.el9.x86_64/usr/lib/systemd/system/mailproxy.service
...

Wrote: /build/RPMS/x86_64/mailproxy-1.0.4-1.el9.x86_64.rpm

==> 构建完成：dist/mailproxy-1.0.4-1.el9.x86_64.rpm
    安装：rpm -ivh dist/mailproxy-1.0.4-1.el9.x86_64.rpm
    升级：rpm -Uvh dist/mailproxy-1.0.4-1.el9.x86_64.rpm
```

**成功标志**:
- ✅ 最后一行显示 "构建完成"
- ✅ 显示了正确的 rpm 文件名
- ✅ shell 返回码为 0 (`echo $?`)

**产物位置**:
- `dist/mailproxy-1.0.4-1.el9.x86_64.rpm` (主包)
- `dist/rpmbuild/RPMS/x86_64/mailproxy-1.0.4-1.el9.x86_64.rpm` (原始位置)
- `dist/rpmbuild/` 临时目录 (可删除)

**时间预期**: 
- 首次构建：约 2-3 分钟 (需要下载 rpmbuild 依赖)
- 重复构建：约 30-60 秒 (所有依赖已缓存)

**常见问题处理**:

| 错误信息 | 原因 | 解决方案 |
|---|---|---|
| `error: No such file or directory` | Docker CLI 不可见或 PATH 不对 | `export PATH="$HOME/docker:$PATH"` |
| `failed to resolve reference` | Docker Hub 网络问题 | 使用镜像代理，或提前用`docker save`导出 |
| `architecture mismatch` | 使用了 arm64 镜像而非 amd64 | 确保 docker run 带 `--platform linux/amd64` |
| `go.mod not found` | 不在项目根目录 | `cd` 到 MailProxy 项目根目录 |
| `container exited abnormally` | Docker 资源不足 | 增加 `colima start --memory 8 --disk 40` |

---

#### ✅ 阶段四：验证 RPM 包功能

> ⚠️ **重要提示**: 
> 在 macOS ARM64 上使用 Docker 进行 x86_64 容器测试存在跨平台卷挂载限制。
> 建议在生产服务器上测试实际安装流程。

**Step 9: 本地元数据验证 **(推荐)

```
# 检查基本元数据
rpm -qip dist/mailproxy-1.0.4-1.el9.x86_64.rpm

# 检查包内容列表
rpm -qplf dist/mailproxy-1.0.4-1.el9.x86_64.rpm

# 列出所有文件及其权限
ls -lh dist/mailproxy-1.0.4-1.el9.x86_64.rpm
```

**预期成功输出**:
```
Name        : mailproxy
Version     : 1.0.4
Release     : 1.el9
Architecture: x86_64
Install Date: (not installed)
Group       : Unspecified
Size        : 9461599
License     : Proprietary
Summary     : MailProxy SMTP relay gateway
Description :
Go 实现的 SMTP 邮件代理网关：业务程序统一连接本代理发信，
代理转发到后端真实 SMTP 服务器，支持多账号配置与路由策略。
```

✅ **通过标准**:
- Version 正确 (与 VERSION 文件一致)
- Architecture 是 x86_64
- Size 约 9MB (9,461,599 bytes)
- Description 包含正确信息
- 无错误信息

**Step 10: 高级验证 - 在测试容器中模拟安装**

由于跨平台限制，此步骤仅在相同架构下可靠运行。如必须测试，参考以下方法:

```
# 需要先构建完整测试环境
cd testtools
docker build -f Dockerfile.testenv -t mailproxy-test-complete . --platform linux/amd64

# 然后拷贝文件到容器内测试 (而不是卷挂载)
docker run -d --name mp-test-via-copy mailproxy-test-complete sleep infinity
docker cp dist/mailproxy-*.rpm mp-test-via-copy:/tmp/
docker exec mp-test-via-copy bash -c "rpm -ivh /tmp/mailproxy-*.rpm"
docker stop mp-test-via-copy
```

⚠️ **注意**: 这种方法比卷挂载更可靠，因为避免了跨平台文件系统转换。

```

*图：MailProxy 作为中央枢纽，统一管理多个后端 SMTP 服务*

```

---

<br/>

## 功能特性

- 🚀 **高性能** - 基于 Go 并发模型，支持高并发连接
- 🔒 **安全认证** - IP 白名单 + 可选 AUTH 认证，防止被滥用为开放中继
- 🔄 **智能路由** - 根据发件人自动选择最优后端账号，提升送达率
- 🎯 **零代码改造** - 只需修改 SMTP 地址配置，现有发邮件逻辑完全不用变
- 🔐 **TLS 加密** - 支持 SMTP over SSL (465) 和 STARTTLS (587)，确保传输安全
- ♻️ **热重载** - 配置文件 SIGHUP 热加载，无需重启服务
- 📊 **监控指标** - Prometheus 指标导出，实时监控系统状态
- 📝 **详细日志** - 结构化日志记录，每次发信全流程追踪
- 🎁 **企业邮箱兼容** - 自动处理「信封发件人必须等于认证账号」限制（阿里/腾讯/网易等）
- 🏗️ **易部署** - 单二进制文件 + systemd 托管，提供 RPM 包一键安装

---

<br/>

## 快速开始

```
# 1. 下载最新版本 (Linux AMD64)
curl -LO https://github.com/YanGLweI/MailProxy/releases/latest/download/mailproxy-1.0.4-linux-amd64.tar.gz
tar -xzf mailproxy-*.tar.gz
chmod +x mailproxy

# 或使用 RPM 包 (RHEL/CentOS/Rocky)
curl -LO https://github.com/YanGLweI/MailProxy/releases/latest/download/mailproxy-1.0.4-1.el9.x86_64.rpm
sudo rpm -ivh mailproxy-1.0.4-1.el9.x86_64.rpm
```

**立即体验：**

```
# 生成自签名证书
./deploy/gen-cert.sh

# 准备配置
cp config.example.yaml config.yaml
vim config.yaml             # 填写后端邮箱账号等

# 启动服务
./mailproxy -config config.yaml
```

**业务接入只需改 3 个参数：**

**业务接入只需改 3 个参数：**

| 参数 | 修改前                    | 修改后                      |
|------|---------------------------|-----------------------------|
| SMTP 主机 | `smtp.qiye.aliyun.com`    | `mailproxy.internal:465`    |
| SSL 证书 | 逐个管理各服务商证书      | 只需信任 mailproxy 一个证书 |  
| 开发成本 | 不同服务商 API 差异适配    | 统一 SMTP 协议，一次开发     |

### 构建源码

如需从源码构建：

```
# 本地构建
go build -o mailproxy .

# 交叉编译 Linux AMD64
GOOS=linux GOARCH=amd64 go build -o mailproxy .

# 运行测试
go test ./...
```

### 连通性自测

**推荐先验证连接再发送测试邮件：**

```
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

```
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

```
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

### 🚀 快速构建 (macOS 本机即可)

```
bash deploy/build-rpm.sh        # 产物 dist/mailproxy-<version>-1.el*.x86_64.rpm
```

**前置要求:**
- ✅ Go 1.25+ `go version` → go1.25.0 darwin/arm64  
- ✅ Docker CLI (colima 或 Docker Desktop) `command -v docker`
- ✅ colima 已运行 `limactl status` → running
- ✅ Rocky Linux 9 amd64 镜像已拉取 `docker images | grep rockylinux`

> 💡 **说明**: 如果 `build-rpm.sh` 执行失败，请先确保所有前置要求都满足。

---

### ⚠️ 重要注意事项

#### 问题 1: Docker Hub 认证超时
**现象**: `failed to authorize: failed to fetch anonymous token`  
**原因**: 国内网络访问 Docker Hub 受限  
**解决**: 配置镜像代理或使用其他镜像站

**方法 A: 使用 DaoCloud 镜像加速**
```
export REGISTRY_MIRROR=https://docker.m.daocloud.io
docker pull --platform linux/amd64 rockylinux/rockylinux:9
```

**方法 B: 提前在服务器拉取并导出导入**
```
# macOS 本地
docker save rockylinux/rockylinux:9 > rockylinux.tar

# 上传到服务器
scp rockylinux.tar server:/tmp/

# 服务器上加载
docker load < /tmp/rockylinux.tar
```

#### 问题 2: macOS ARM64 的卷挂载限制  
**现象**: 跨平台卷挂载导致文件不可见  
**原因**: Docker Desktop 在 arm64 host 上无法完全支持 x86_64 容器的卷挂载  

**解决方案**:  
1. 构建阶段使用正确的 `--platform linux/amd64`
2. 验证阶段改用 `rpm -qip` 检查元数据而非安装测试
3. 生产环境直接上传 RPM 到目标服务器安装

#### 问题 3: RockyLinux 基础镜像无 systemd  
**现象**: 测试容器无法启动 systemd  
**原因**: 官方镜像最小化设计  

**解决方案**: 使用完整测试 Dockerfile 或仅做元数据验证

---

### 📦 详细构建流程与每步操作指南

如需从头完整了解如何从 macOS 构建 x86_64 Linux RPM 包，请参考以下步骤:

#### 🔧 前置环境与依赖

| 组件 | 要求 | 检查命令 |
|---|---|---|
| macOS 芯片 | Apple Silicon (M1/M2/M3) 或 Intel | `uname -m` → arm64 / x86_64 |
| Go | ≥1.25.0 | `go version` |
| Homebrew (可选) | 用于安装 lima | `/usr/bin/ruby -e "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/master/install)"` (未装过才需要) |
| curl/tar/git | 系统自带 | `which curl tar git` |
| 网络 | 可访问 GitHub/Docker Hub(镜像站优先) | `ping github.com` |

> ⚠️ **重要提示**: 
> - ARM64 Mac 需要下载对应版本的 Docker CLI(见 Step 3)
> - 推荐使用 [colima](https://github.com/abiosoft/colima) 替代 Docker Desktop  
> - Docker Hub 直连可能超时，建议配置镜像代理

#### 🏗️ 阶段一：搭建临时容器环境（约 5 分钟）

**Step 1: 下载 colima 二进制 **(ARM64)

```
mkdir -p ~/bin && \
curl -sL -o ~/bin/colima \
    https://github.com/abiosoft/colima/releases/latest/download/colima-Darwin-arm64 && \
chmod +x ~/bin/colima && \
~/bin/colima version
```

**输出示例:**
```
colima version v0.10.3
git commit: ...
```

**Step 2: 安装 limactl **(虚拟化层)

```
brew install lima
limactl --version
```

**输出示例:**
```
limactl version 2.2.0
```

**Step 3: 下载 Docker CLI 静态二进制**

```
mkdir -p ~/docker && \
curl -sL -o /tmp/docker.tgz \
    https://download.docker.com/mac/static/stable/aarch64/docker-28.3.3.tgz && \
tar -xzf /tmp/docker.tgz -C /tmp && \
cp /tmp/docker/docker ~/docker/ && \
~/docker/docker version
```

**输出示例:**
```
Client: Docker Engine - Community
 Version:           27.5.1-rc1
...
```

**替代方案：使用 Homebrew 安装 **(如果已配置)
```
# 方法 A: 检查 colima 自带的 docker 命令
colima status
# 如果显示 running，则 ~/.local/bin/docker 应该可用

# 方法 B: 导出 docker socket
export DOCKER_HOST=unix://$HOME/.colima/default/docker.sock
docker info
```

**Step 4: 启动 colima 虚拟机并加载镜像**

```
export PATH="$HOME/docker:$PATH"   # 让 colima 找到 docker
colima start --cpu 2 --memory 4 --disk 20
```

等待输出:`READY. Run 'limactl shell colima' to open the shell.`

**验证 Docker 工作正常:**
```
limactl status colima
# 应该看到：running

docker images
# 此时可以没有任何镜像，后续会拉取

docker run --rm hello-world
# 测试运行一个小容器
```

**Step 5: 拉取构建所需镜像 **(指定 amd64 平台)

```
docker pull --platform linux/amd64 rockylinux/rockylinux:9
```

**提示**:如果直连超时，使用镜像代理:
```
docker pull --platform linux/amd64 docker.m.daocloud.io/rockylinux/rockylinux:9
```

**验证镜像已成功拉取**:
```
docker images | grep rockylinux
# 应该看到:
# rockylinux/rockylinux     9      <IMAGE-ID>   <DATE>   710MB
# docker.m.daocloud.io/rockylinux/rockylinux   9    <same-ID>
```

---

#### 📝 阶段二：准备源码与构建脚本

**Step 6: 打开项目目录**

```
cd /Users/yeung/Projects/MailProxy   # 替换为你的项目路径
cat VERSION                          # 确认版本号 (如 1.0.4)
```

**Step 7: 查看核心构建脚本**

```
cat deploy/build-rpm.sh
```

**关键流程**:

1. 读取 VERSION 文件中的版本号
2. `CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build ...` → 交叉编译 Linux x86_64 静态二进制
3. 将 spec/specs/source 复制到 `dist/rpmbuild/SOURCES/`
4. `docker run --platform linux/amd64 rockylinux:9 rpmbuild ...` → 在容器中执行 rpm 构建

**rpmbuild 目录结构详解**:
```
dist/rpmbuild/
├── SOURCES/          # 源代码文件 (binary + config + service)
│   ├── mailproxy     # Go 编译的 x86_64 二进制
│   ├── config.yaml   # 默认配置文件  
│   ├── mailproxy.service  # systemd unit 模板
│   └── README.md     # 产品说明文档
├── SPECS/            # RPM spec 文件定义包结构
│   └── mailproxy.spec
├── BUILD/            # 编译工作目录 (空目录)
├── BUILDROOT/        # 安装包根目录 (包含安装后的文件布局)
├── RPMS/x86_64/      # 最终生成的 RPM 包
│   └── mailproxy-1.0.4-1.el9.x86_64.rpm
└── SRPMS/            # 源码 RPM (如果有)
    └── mailproxy-1.0.4-1.el9.src.rpm
```

---

#### 🎯 阶段三：一键构建 RPM 包

**Step 8: 直接运行构建脚本**

```
export PATH="$HOME/docker:$PATH"    # 确保找到 docker
bash deploy/build-rpm.sh
```

**完整输出示例**(优化版):
```
==> 交叉编译 Linux x86_64 二进制 (v1.0.4)
==> 准备打包源文件
==> Docker 内执行 rpmbuild (rockylinux/rockylinux:9, linux/amd64)

Installed:
  rpm-build-4.16.1.3-40.el9.x86_64
  ... (更多依赖包)

Executing(%prep): /bin/sh -e /var/tmp/rpm-tmp.XXX
+ umask 022
+ cd /build/BUILD
+ rm -rf mailproxy-1.0.4
...

Executing(%install): /bin/sh -e /var/tmp/rpm-tmp.YYY
+ umask 022
+ '[' /build/BUILDROOT/mailproxy-1.0.4-1.el9.x86_64 '!=' / ']'
...
+ install -Dpm 0755 /build/SOURCES/mailproxy /build/BUILDROOT/mailproxy-1.0.4-1.el9.x86_64/usr/bin/mailproxy
+ install -Dpm 0644 /build/SOURCES/mailproxy.service /build/BUILDROOT/mailproxy-1.0.4-1.el9.x86_64/usr/lib/systemd/system/mailproxy.service
...

Wrote: /build/RPMS/x86_64/mailproxy-1.0.4-1.el9.x86_64.rpm

==> 构建完成：dist/mailproxy-1.0.4-1.el9.x86_64.rpm
    安装：rpm -ivh dist/mailproxy-1.0.4-1.el9.x86_64.rpm
    升级：rpm -Uvh dist/mailproxy-1.0.4-1.el9.x86_64.rpm
```

**成功标志**:
- ✅ 最后一行显示 "构建完成"
- ✅ 显示了正确的 rpm 文件名
- ✅ shell 返回码为 0 (`echo $?`)

**产物位置**:
- `dist/mailproxy-1.0.4-1.el9.x86_64.rpm` (主包)
- `dist/rpmbuild/RPMS/x86_64/mailproxy-1.0.4-1.el9.x86_64.rpm` (原始位置)
- `dist/rpmbuild/` 临时目录 (可删除)

**时间预期**: 
- 首次构建：约 2-3 分钟 (需要下载 rpmbuild 依赖)
- 重复构建：约 30-60 秒 (所有依赖已缓存)

**常见问题处理**:

| 错误信息 | 原因 | 解决方案 |
|---|---|---|
| `error: No such file or directory` | Docker CLI 不可见或 PATH 不对 | `export PATH="$HOME/docker:$PATH"` |
| `failed to resolve reference` | Docker Hub 网络问题 | 使用镜像代理，或提前用`docker save`导出 |
| `architecture mismatch` | 使用了 arm64 镜像而非 amd64 | 确保 docker run 带 `--platform linux/amd64` |
| `go.mod not found` | 不在项目根目录 | `cd` 到 MailProxy 项目根目录 |
| `container exited abnormally` | Docker 资源不足 | 增加 `colima start --memory 8 --disk 40` |

---

#### ✅ 阶段四：验证 RPM 包功能

> ⚠️ **重要提示**: 
> 在 macOS ARM64 上使用 Docker 进行 x86_64 容器测试存在跨平台卷挂载限制。
> 建议在生产服务器上测试实际安装流程。

**Step 9: 本地元数据验证 **(推荐)
