# MailProxy v1.0.4 正式环境部署指南

##  版本信息

**版本号**: v1.0.4  
**修复内容**: 
- ✅ 支持超大 HTML 邮件（MaxLineLength 从 2000 → 10000 字节）
- ✅ TLS 加密套件扩展，提升兼容性
- ✅ 详细错误日志记录

## 📦 两种部署方式

### 方式 1：直接编译二进制文件（最快）

如果您有 Linux 服务器或能访问构建环境：

```bash
# 在正式服务器上直接执行
cd /opt/mailproxy  # 或您的安装目录

# 停止服务
systemctl stop mailproxy

# 备份当前版本
cp /usr/local/bin/mailproxy /usr/local/bin/mailproxy.bak.$(date +%Y%m%d)

# 下载新版本二进制
# 从本地复制或使用之前开发的二进制
./deploy/update-binary.sh mailproxy_final  # 如果有这个脚本

# 或直接替换为之前的 binary
cp /Users/yeung/Projects/MailProxy/mailproxy_final /usr/local/bin/mailproxy
chmod +x /usr/local/bin/mailproxy

# 启动服务
systemctl start mailproxy

# 查看日志
journalctl -u mailproxy -f
```

### 方式 2：RPM 包升级（标准方式）

#### 步骤 1：在开发环境构建 RPM

```bash
cd /Users/yeung/Projects/MailProxy

# 确保 VERSION 文件是 1.0.4
cat VERSION  # 应该显示 "1.0.4"

# 启动 colima/docker
colima start  # 或者 open /Applications/Docker.app

# 运行构建脚本
bash deploy/build-rpm.sh
```

**预期输出**:
```
==> 交叉编译 Linux x86_64 二进制 (v1.0.4)
==> 准备打包源文件
==> Docker 内执行 rpmbuild (rockylinux/rockylinux:9, linux/amd64)
==> 构建完成：dist/mailproxy-1.0.4-1.el9.x86_64.rpm
```

#### 步骤 2：传输到正式环境

```bash
# 在开发服务器执行
scp dist/mailproxy-1.0.4-1.el9.x86_64.rpm root@production-server:/tmp/

# 或在生产服务器执行
ssh production-server << 'EOF'
# 进入临时目录
cd /tmp

# 升级
rpm -Uvh mailproxy-1.0.4-1.el9.x86_64.rpm

# 重启服务
systemctl restart mailproxy

# 确认运行正常
systemctl status mailproxy

# 监控日志
journalctl -u mailproxy -f
EOF
```

## 🔧 配置检查清单

### 1. IP 白名单验证
确保 `/etc/mailproxy/config.yaml` 包含业务平台 IP:
```yaml
server:
  ip_whitelist:
    - 127.0.0.1
    - ::1
    - 10.0.0.0/8
    - 10.66.254.155  # ← 确保这行存在
```

修改后热重载:
```bash
systemctl reload mailproxy
```

### 2. 邮件大小设置
如果发送的邮件较大，考虑调整：
```yaml
server:
  max_message_bytes: 104857600  # 100MB
  io_timeout: 180s              # 3 分钟超时
```

### 3. 日志级别
开发测试时可设为 debug，生产环境保持 info:
```yaml
log:
  level: info
```

## 📊 验证测试

### 步骤 1：基本连接测试

```bash
# 使用 telnet 或 openssl 测试端口
openssl s_client -connect 10.66.254.155:465 -starttls smtp
```

### 步骤 2：发送测试邮件

```bash
# 使用业务平台发送测试报告邮件
# 或使用邮件客户端
mailq  # 查看队列状态
```

### 步骤 3：观察日志

```bash
# 实时日志
journalctl -u mailproxy -f

# 关键日志序列（成功的标志）:
time=... level=INFO msg="开始读取邮件数据" from=... rcpts_count=3 max_message_bytes=52428800
time=... level=INFO msg="邮件发送成功" backend=aliyun cost=1.234s proxy_account=test
```

### 步骤 4：对比测试

| 场景 | 旧版本 | 新版本期望 |
|------|--------|-----------|
| 小邮件 (< 1MB) | ✅ 成功 | ✅ 成功 |
| 大邮件 (HTML, > 10MB) | ❌ "too long a line" | ✅ 成功 |
| 报告邮件发送 | ❌ broken pipe | ✅ 成功 |

## ⚠️ 故障排查

### 问题 1: "too long a line" 仍然出现

**原因**: 仍有单行超过 10KB 的内容

**解决**:
1. 编辑 `/etc/mailproxy/internal/server/server.go`
2. 将 `srv.MaxLineLength = 10000` 改为更大的值（如 20000）
3. 重新构建并部署

### 问题 2: TLS 握手失败

**症状**: 日志中只有 "客户端已连接"，之后无日志

**可能原因**: 
- IP 不在白名单中
- TLS 证书不匹配

**解决**:
```bash
# 检查 IP 白名单
grep -A 10 "ip_whitelist:" /etc/mailproxy/config.yaml

# 检查 TLS 日志
journalctl -u mailproxy | grep "TLS 握手"

# 添加具体 IP（如果需要）
vi /etc/mailproxy/config.yaml
systemctl reload mailproxy
```

### 问题 3: 服务无法启动

**症状**: `systemctl status mailproxy` 显示异常

**解决**:
```bash
# 查看详细错误
journalctl -u mailproxy -b --no-pager | tail -50

# 检查配置文件语法
/usr/local/bin/mailproxy -config /etc/mailproxy/config.yaml

# 恢复备份
cp /usr/local/bin/mailproxy.bak.* /usr/local/bin/mailproxy
systemctl start mailproxy
```

## 📝 回滚方案

如果遇到问题需要回滚：

```bash
# 在生产服务器执行
ssh production-server << 'EOF'
# 停止服务
systemctl stop mailproxy

# 恢复备份
cp /usr/local/bin/mailproxy.bak.$(date +%Y%m%d) /usr/local/bin/mailproxy
chmod +x /usr/local/bin/mailproxy

# 重启服务
systemctl start mailproxy

# 确认正常运行
systemctl status mailproxy
EOF
```

## 📞 支持联系人

- **开发人员**: @您的用户名
- **GitHub Issues**: https://github.com/your-org/mailproxy/issues
- **紧急联系**: 如有严重问题，请随时联系

---

**最后更新**: 2026-08-20  
**版本**: v1.0.4 (Bugfix Release)
