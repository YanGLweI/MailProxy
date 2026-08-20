# MailProxy RPM 完整打包流程指南

本教程适用于 macOS 开发机交叉编译 x86_64 Linux 二进制并构建 Rocky Linux/CentOS/RHEL 系的 RPM 包。全程无需安装 Docker Desktop、无需本地 Linux 环境、无需 Linux 虚拟机。

---

## 前置条件

| 组件 | 要求 | 检查命令 |
|---|---|---|
| macOS 芯片 | Apple Silicon (M1/M2/M3) 或 Intel | `uname -m` → arm64 / x86_64 |
| Go | ≥1.25.0 | `go version` |
| Homebrew (可选) | 用于安装 lima | `/usr/bin/ruby -e "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/master/install)"` (未装过才需要) |
| curl/tar/git | 系统自带 | `which curl tar git` |
| 网络 | 可访问 GitHub/Docker Hub(镜像站优先) | `ping github.com` |

---

## 第一阶段：搭建临时容器环境 (约 5 分钟)

### Step 1: 下载 colima 二进制 (ARM64)

```bash
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

### Step 2: 安装 limactl (虚拟化层)

```bash
brew install lima
limactl --version
```

**输出示例:**
```
limactl version 2.2.0
```

### Step 3: 下载 Docker CLI 静态二进制

```bash
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
 Version:           28.3.3
...
```

### Step 4: 启动 colima 虚拟机并加载镜像

```bash
export PATH="$HOME/docker:$PATH"   # 让 colima 找到 docker
colima start --cpu 2 --memory 4 --disk 20
```

等待输出:`READY. Run 'limactl shell colima' to open the shell.`

### Step 5: 拉取构建所需镜像 (指定 amd64 平台)

```bash
docker pull --platform linux/amd64 rockylinux/rockylinux:9
```

**提示:**如果直连超时，使用镜像代理:
```bash
docker pull --platform linux/amd64 docker.m.daocloud.io/rockylinux/rockylinux:9
```

---

## 第二阶段：准备源码与构建脚本

### Step 6: 克隆或打开项目目录

```bash
cd /Users/yeung/Projects/MailProxy   # 替换为你的项目路径
cat VERSION                          # 确认版本号 (如 1.0.4)
```

### Step 7: 查看核心构建脚本

```bash
cat deploy/build-rpm.sh
```

**关键流程:**

1. 读取 VERSION 文件中的版本号
2. `CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build ...` → 交叉编译 Linux x86_64 静态二进制
3. 将 spec/specs/source 复制到 `dist/rpmbuild/SOURCES/`
4. `docker run --platform linux/amd64 rockylinux:9 rpmbuild ...` → 在容器中执行 rpm 构建

---

## 第三阶段：一键构建 RPM 包

### Step 8: 直接运行构建脚本

```bash
export PATH="$HOME/docker:$PATH"    # 确保找到 docker
bash deploy/build-rpm.sh
```

**完整输出示例:**
```
==> 交叉编译 Linux x86_64 二进制 (v1.0.4)
==> 准备打包源文件
==> Docker 内执行 rpmbuild (rockylinux/rockylinux:9, linux/amd64)
Building target platforms: x86_64
Building for target x86_64
...
Wrote: /build/RPMS/x86_64/mailproxy-1.0.4-1.el9.x86_64.rpm

==> 构建完成：dist/mailproxy-1.0.4-1.el9.x86_64.rpm
    安装：rpm -ivh dist/mailproxy-1.0.4-1.el9.x86_64.rpm
    升级：rpm -Uvh dist/mailproxy-1.0.4-1.el9.x86_64.rpm
```

**产物位置:**
- `dist/mailproxy-1.0.4-1.el9.x86_64.rpm`
- 临时构建目录:`dist/rpmbuild/`(可删除)

---

## 第四阶段：验证 RPM 包功能

### Step 9: 创建带 systemd 的测试容器

```bash
docker run -d --name mp-test --privileged --tmpfs /run --tmpfs /run/lock \
    -v $(pwd)/dist:/dist:ro \
    rockylinux/rockylinux:9 /sbin/init >/dev/null 2>&1

sleep 25 && echo "TEST_CONTAINER_READY"
```

### Step 10: 执行安装与验证

```bash
docker exec mp-test bash -c '
dnf install -y -q openssl >/dev/null 2>&1
rpm -i /dist/mailproxy-1.0.4-1.el9.x86_64.rpm >/dev/null 2>&1

echo "=== 文件落位 ==="
ls -l /etc/mailproxy/config.yaml
ls -ld /etc/mailproxy/certs
file /usr/bin/mailproxy

echo "=== 服务状态 ==="
sed -i "s/^validate_on_start: true/validate_on_start: false/" /etc/mailproxy/config.yaml
systemctl start mailproxy
sleep 4
systemctl is-enabled mailproxy
systemctl is-active mailproxy

echo "=== 进程检查 ==="
ps -o user,pid,comm -C mailproxy

echo "=== 端口检查 ==="
(exec 3<>/dev/tcp/127.0.0.1/465 && echo "PASS:465 端口正常") || echo "FAIL:465 端口异常"

echo "=== 日志 ==="
tail -5 /var/log/mailproxy/mailproxy.log
'
```

**预期成功输出:**
```
=== 文件落位 ===
-rw-r----- 1 root mailproxy 4268 /etc/mailproxy/config.yaml
drwxr-x--- 2 root mailproxy /etc/mailproxy/certs
/usr/bin/mailproxy: ELF 64-bit LSB executable, x86-64...

=== 服务状态 ===
enabled
active

=== 进程检查 ===
mailpro+     261 mailproxy

=== 端口检查 ===
PASS:465 端口正常

=== 日志 ===
time=... level=INFO msg="MailProxy 启动中"
time=... level=INFO msg="SMTP over SSL 代理服务已启动" listen=:465
```

---

## 第五阶段：升级回滚与常见问题排查

### Step 11: 升级验证 (新旧版本共存)

```bash
# 模拟旧版 1.0.0 安装
rpm -e mailproxy >/dev/null 2>&1
rpm -i /dist/mailproxy-1.0.0-1.el9.x86_64.rpm >/dev/null 2>&1

# 手动修改配置后升级新版
echo "# CUSTOM EDIT" >> /etc/mailproxy/config.yaml
rpm -U /dist/mailproxy-1.0.4-1.el9.x86_64.rpm >/dev/null 2>&1

# 验证保留
grep "CUSTOM EDIT" /etc/mailproxy/config.yaml && echo "配置保留 OK"
[ -f /etc/mailproxy/config.yaml.rpmnew ] && echo "差异配置落地 .rpmnew OK"
```

### Step 12: 常见问题排查速查表

| 现象 | 可能原因 | 解决命令 |
|---|---|---|
| `启动即退出 exit 1` | 配置文件权限不对 (0600 root:组) | `chmod 640 /etc/mailproxy/config.yaml` |
| `open server.crt: permission denied` | certs 目录属主 root:root | `chown root:mailproxy /etc/mailproxy/certs` |
| `Backend SMTP 连通性检测失败` | 授权码无效/网络不通 | `validate_on_start: false` 跳过检测 |
| `Docker 拉取镜像超时` | 国内网络问题 | `docker pull --platform linux/amd64 docker.m.daocloud.io/rockylinux/rockylinux:9` |
| `rpmbuild: failed to stat ... No such file or directory` | Docker 挂载卷不可见 | 确保主机 HOME 目录挂载到 VM(使用 colima default) |
| `colima: dependency check failed for docker` | PATH 中找不到 docker | `export PATH="$HOME/docker:$PATH"; colima start` |

---

## 附录 A: Spec 文件关键设计

```spec
%global _build_id_links none
# 兜底定义宏(最小化基础镜像无 rpm-macros)
%{!?_systemd_unitdir: %global _systemd_unitdir /usr/lib/systemd/system}

%pre
# 幂等创建服务用户
getent group mailproxy || groupadd -r mailproxy
getent passwd mailproxy || \
    useradd -r -g mailproxy -d %{_sysconfdir}/mailproxy \
        -s /sbin/nologin -c "MailProxy service" mailproxy

%files
# 正确的文件权限
%config(noreplace) %attr(0640, root, mailproxy) %{_sysconfdir}/mailproxy/config.yaml
%dir %attr(0750, root, mailproxy) %{_sysconfdir}/mailproxy/certs
%dir %attr(0750, mailproxy, mailproxy) %{_localstatedir}/log/mailproxy
```

---

## 附录 B: 环境变量与性能优化

| 变量 | 作用 | 推荐值 |
|---|---|---|
| CGO_ENABLED | 禁用 CGO(减少依赖) | 0 |
| GOOS/GOARCH | 目标架构 | linux/amd64 |
| ldflags | 去掉 debug 符号,减小体积 | `-s -w` |
| trimpath | 去掉构建路径,提升安全性 | true |

---

## 附录 C: 清理残留资源

```bash
# 停止临时容器
docker rm -f mp-sys mp-rocky-systemd mp-build >/dev/null 2>&1

# 删除 colima VM (可选)
colima stop; colima delete --purge

# 清理临时工具 (可选)
rm -rf ~/bin/colima ~/docker /tmp/docker.tgz /tmp/docker.tgz

# 卸载 colima (可选)
brew uninstall cola  # 通过 brew 安装的 colima 才能这样操作
```

---

## 下一步

- **发布流程**:更新 VERSION → `git tag v1.0.4` → GitHub Release 上传 rpm
- **自动化**:结合 GitHub Actions + QEMU 多架构模拟器实现全自动 CI/CD
- **签名**:为 RPM 包添加 GPG 签名并配置 GPGKEY 进行校验
- **文档**:生成 RELEASE_NOTES.md 列出新增功能与兼容性问题

---

**全文结束。建议保存此文档作为本地知识库，后续遇到类似环境可直接按步骤复现。**
