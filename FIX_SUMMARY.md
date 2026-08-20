# 最终修复方案 - 解决超大邮件发送问题

## 问题根本原因

经过深入分析，确定问题是由于 **go-smtp 库的 `lineLimitReader` 限制了每行最大长度为 2000 字节**（默认值），而某些大型 HTML 邮件的 Base64/MIME 编码内容产生了超过 2000 字节的单行数据，导致 SMTP 协议层主动断开连接。

### 错误信息
```
time=2026-08-20T10:03:34.237+08:00 level=WARN msg=读取邮件数据失败 error="smtp: too long a line in input stream" from=lw.yang@ho-brostech.com
```

### 技术细节

根据 go-smtp v0.25.0 源码分析：
- 默认 `MaxLineLength = 2000` (RFC 5321 标准的两倍)
- 在 DATA 阶段使用 `lineLimitReader` 包装输入流
- 当检测到单行超过限制时，返回 `ErrTooLongLine` 错误并关闭连接

## 实施的修复

### 修改 1：增大 MaxLineLength (server.go)

**文件**: `/Users/yeung/Projects/MailProxy/internal/server/server.go`

```go
func newSMTPServer(cfg *config.Config, s *Server) *smtp.Server {
    srv := smtp.NewServer(smtp.BackendFunc(s.newSession))
    srv.Domain = cfg.Server.Hostname
    srv.ReadTimeout = cfg.Server.IOTimeout.Duration()
    srv.WriteTimeout = cfg.Server.IOTimeout.Duration()
    srv.MaxMessageBytes = cfg.Server.MaxMessageBytes
    srv.MaxRecipients = 500
    srv.AllowInsecureAuth = true
    srv.EnableSMTPUTF8 = true
    // === 增大最大行长度，支持超大 Base64 编码内容 ===
    srv.MaxLineLength = 10000 // 从默认的 2000 增大到 10000 字节
    return srv
}
```

### 修改 2：增大 MaxLineLength (starttls.go)

**文件**: `/Users/yeung/Projects/MailProxy/internal/server/starttls.go`

同样的修改应用于 STARTTLS 监听器，确保两个端口行为一致。

### 其他优化保留

之前添加的以下优化也一并包含在内：
- ✅ TLS 加密套件扩展
- ✅ TLS 握手错误详细日志
- ✅ 邮件数据处理日志增强
- ✅ STARTTLS 连接独立日志标记

## 测试验证

### 步骤 1：停止旧服务
```bash
pkill mailproxy
```

### 步骤 2：启动新版本
```bash
cd /Users/yeung/Projects/MailProxy
./mailproxy_final -config config.yaml
```

### 步骤 3：发送大型 HTML 邮件测试

根据您的业务平台或 SMTP 客户端，发送与正式环境相同大小的 HTML 邮件。

### 预期的成功日志序列

```log
time=2026-08-20TXX:XX:XX level=INFO msg=客户端已连接 client_ip=10.60.1.191 remote=...
time=2026-08-20TXX:XX:XX level=INFO msg=开始读取邮件数据 from=... rcpts_count=3 max_message_bytes=52428800
time=2026-08-20TXX:XX:XX level=INFO msg=邮件发送成功 ... (无错误)
```

### 如果仍然失败

如果出现以下情况：
1. **仍然看到 "too long a line"** → 说明有超过 10KB 的单行数据，需要进一步调优
2. **其他错误** → 查看具体错误信息，可能需要调整 TLS 配置

### 如果需要更大行数限制

如果发现仍有超长行问题，可以继续增加：

```go
srv.MaxLineLength = 20000  // 20KB
// 或者
srv.MaxLineLength = 0      // 禁用长度限制（不推荐，可能有安全风险）
```

## 生产环境部署

### 方式 1：替换现有二进制
```bash
# 在生产服务器上执行
scp /Users/yeung/Projects/MailProxy/mailproxy_final root@production-server:/tmp/mailproxy_fixed

ssh production-server << 'EOF'
systemctl stop mailproxy
cp /usr/local/bin/mailproxy /usr/local/bin/mailproxy.bak_$(date +%Y%m%d)
mv /tmp/mailproxy_fixed /usr/local/bin/mailproxy
chmod +x /usr/local/bin/mailproxy
systemctl start mailproxy
journalctl -u mailproxy -f
EOF
```

### 方式 2：热重载配置
```bash
# 如果只是调整配置文件，可以热重载
systemctl reload mailproxy
```

## 回滚方案

如果遇到不可预见的问题：

```bash
# 恢复旧版本
systemctl stop mailproxy
mv /usr/local/bin/mailproxy.bak_* /usr/local/bin/mailproxy
chmod +x /usr/local/bin/mailproxy
systemctl start mailproxy
```

## 监控建议

部署后请密切观察：
1. 大型邮件处理的成功率
2. 是否有 "too long a line" 或其他异常日志
3. 邮件传输的平均耗时

## 技术总结

本次修复的核心改进：
1. **增大 MaxLineLength 从 2000→10000 字节** - 直接解决超长行问题
2. **TLS 兼容性优化** - 提升不同客户端的连接能力
3. **详细日志记录** - 便于后续问题排查

预计影响：
- ✅ 能够处理高达 10KB 的 Base64/MIME 单行编码
- ✅ 显著提升大型 HTML 邮件的成功率
- ✅ 保持向后兼容性，不影响小邮件处理

---

**最后更新时间**: 2026-08-20
**版本**: mailproxy_final (v1.0.4-fix)
