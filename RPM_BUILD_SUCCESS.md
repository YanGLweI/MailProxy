# MailProxy v1.0.4 RPM 构建完成报告

## ✅ 构建状态：成功

**时间**: 2026-08-20  
**版本号**: 1.0.4  
**架构**: x86_64  
**产物大小**: 3.1 MB (源码 9.4 MB)

---

## 📦 产出文件

### RPM 包位置
```
/Users/yeung/Projects/MailProxy/dist/mailproxy-1.0.4-1.el9.x86_64.rpm
```

### RPM 包信息
```bash
Name        : mailproxy
Version     : 1.0.4
Release     : 1.el9
Architecture: x86_64
Summary     : MailProxy SMTP relay gateway
Size        : 9,455,373 bytes
License     : Proprietary
```

### 安装文件列表
```
/etc/mailproxy/                    # 配置目录
/etc/mailproxy/certs/              # TLS 证书目录
/etc/mailproxy/config.yaml         # 主配置文件
/usr/bin/mailproxy                 # 二进制程序
/usr/lib/systemd/system/mailproxy.service  # systemd 服务
/var/log/mailproxy/               # 日志目录
```

---

## 🔧 核心修复功能

本次 v1.0.4 版本包含以下关键修复：

### 1. **支持超大 HTML 邮件** ⭐⭐⭐
- **问题**: go-smtp 库默认限制每行 2000 字节，Base64 编码的 HTML 会触发 "too long a line" 错误
- **解决**: `MaxLineLength` 从 2000 → 10000 字节
- **影响**: 可处理高达 10KB 的 Base64/MIME 单行编码

### 2. **TLS 兼容性优化** ⭐⭐
- **扩展加密套件**: 支持更多 ECDHE 和 RSA 算法
- **详细错误日志**: TLS 握手失败时记录完整错误信息
- **提升客户端兼容性**: 适配不同版本的 TLS 实现

### 3. **增强监控和诊断** ⭐⭐
- **详细的邮件处理日志**: 记录每个阶段的状态
- **STARTTLS 独立标记**: 区分 465 和 587 端口连接
- **超时和错误分类**: 更清晰的错误提示

---

## 🚀 正式环境部署步骤

### 方式 A: 手动替换二进制（最快）

```bash
# 1. 传输到生产服务器
scp /Users/yeung/Projects/MailProxy/dist/mailproxy-1.0.4-1.el9.x86_64.rpm \
    root@production-server:/tmp/

# 2. SSH 到生产服务器
ssh root@production-server << 'EOF'
# 进入临时目录
cd /tmp

# 备份当前版本
cp /usr/local/bin/mailproxy /usr/local/bin/mailproxy.v1.0.3.bak.$(date +%Y%m%d)

# 安装新版本
rpm -Uvh mailproxy-1.0.4-1.el9.x86_64.rpm

# 重启服务
systemctl restart mailproxy

# 确认运行正常
systemctl status mailproxy

# 监控日志
journalctl -u mailproxy -f
EOF
```

### 方式 B: 直接复制二进制（紧急修复）

如果时间紧迫，可以直接使用之前编译的二进制：

```bash
# 传输 Linux 二进制
scp /Users/yeung/Projects/MailProxy/mailproxy_linux_amd64 \
    root@production-server:/tmp/mailproxy_v1.0.4

# SSH 执行
ssh root@production-server << 'EOF'
systemctl stop mailproxy
mv /usr/local/bin/mailproxy /usr/local/bin/mailproxy.v1.0.3.bak.$(date +%Y%m%d)
mv /tmp/mailproxy_v1.0.4 /usr/local/bin/mailproxy
chmod +x /usr/local/bin/mailproxy
systemctl start mailproxy
journalctl -u mailproxy -f
EOF
```

---

## 📋 验证检查清单

安装后请确认以下内容：

### 基础检查
- [ ] `systemctl status mailproxy` 显示 active (running)
- [ ] `/etc/mailproxy/config.yaml` 权限为 0640，属主 root:mailproxy
- [ ] `/etc/mailproxy/certs/` 目录存在且有证书文件

### IP 白名单验证
```bash
grep -A 10 "ip_whitelist:" /etc/mailproxy/config.yaml
# 确保包含业务平台 IP：10.66.254.155
```

### 测试发送大邮件
```bash
# 发送与正式环境相同大小的 HTML 邮件
# 观察日志应包含:
# "开始读取邮件数据"
# "邮件发送成功"
# 不应出现："too long a line in input stream"
```

### 日志检查
```bash
journalctl -u mailproxy -f | grep -E "(客户端已连接|邮件发送成功|too long)"
```

---

## ⚠️ 常见问题排查

### 问题 1: 服务无法启动
**症状**: `systemctl status mailproxy` 显示异常

**解决**:
```bash
# 查看详细错误
journalctl -u mailproxy -b --no-pager | tail -50

# 恢复备份
cp /usr/local/bin/mailproxy.v1.0.3.bak.* /usr/local/bin/mailproxy
systemctl start mailproxy
```

### 问题 2: "too long a line" 仍出现
**可能原因**: 仍有超过 10KB 的单行数据

**解决**:
1. 编辑 `internal/server/server.go`
2. 将 `srv.MaxLineLength = 10000` 改为更大的值（如 20000）
3. 重新构建并部署

### 问题 3: TLS 握手失败
**症状**: 日志中只有 "客户端已连接"，之后无日志

**解决**:
```bash
# 检查 IP 白名单
grep -A 10 "ip_whitelist:" /etc/mailproxy/config.yaml

# 添加具体 IP（如果需要）
vi /etc/mailproxy/config.yaml
# 在 ip_whitelist 中添加：
#   - 10.66.254.155
systemctl reload mailproxy
```

---

## 📊 预期效果对比

| 场景 | v1.0.3 | v1.0.4 (期望) |
|------|--------|--------------|
| 小邮件 (< 1MB) | ✅ 成功 | ✅ 成功 |
| 中等邮件 (1-20MB) | ❌ "too long a line" | ✅ 成功 |
| 大型 HTML 邮件 (> 20MB) | ❌ broken pipe | ✅ 成功 |
| 报告邮件发送 | ❌ 失败 | ✅ 成功 |

---

## 🔄 回滚方案

如遇到问题需要回滚到 v1.0.3：

```bash
ssh production-server << 'EOF'
# 停止服务
systemctl stop mailproxy

# 恢复备份
mv /usr/local/bin/mailproxy.v1.0.3.bak.$(date +%Y%m%d) /usr/local/bin/mailproxy
chmod +x /usr/local/bin/mailproxy

# 重启服务
systemctl start mailproxy

# 确认正常运行
systemctl status mailproxy
EOF
```

---

## 📝 下一步行动

### 立即行动
1. ✅ 将 RPM 包复制到生产服务器
2. ✅ 在维护窗口期间升级
3. ✅ 监控日志和业务反馈

### 长期优化
1. 监控系统指标（如启用 metrics）
2. 收集用户反馈
3. 考虑 CI/CD 自动化

---

**文档生成时间**: 2026-08-20 10:40  
**构建工具**: RockyLinux 9 Docker 容器 + rpmbuild  
**镜像源**: docker.m.daocloud.io/rockylinux/rockylinux:9
