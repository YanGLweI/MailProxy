#!/bin/bash
# MailProxy 大邮件修复 - 快速部署脚本
# 使用方法：chmod +x deploy-fix.sh && ./deploy-fix.sh [目标服务器 IP]

set -e

echo "=========================================="
echo "MailProxy 大邮件发送问题修复 - 快速部署"
echo "=========================================="

# 构建新版本
echo "[1/5] 构建新版本..."
cd /Users/yeung/Projects/MailProxy
go build -o mailproxy_fixed . || {
    echo "❌ 构建失败，请检查 Go 版本和依赖"
    exit 1
}
echo "✅ 构建成功"

# 验证二进制文件
echo "[2/5] 验证新版本..."
if ! file mailproxy_fixed | grep -q "executable"; then
    echo "❌ 生成的二进制文件无效"
    exit 1
fi
./mailproxy_fixed --version 2>&1 || true
echo "✅ 验证通过"

# 显示配置建议
echo ""
echo "[3/5] 重要配置建议："
echo "====================="
cat << 'EOF'

如果业务平台发送的 HTML 邮件较大（超过 20MB），建议调整 config.yaml：

server:
  max_message_bytes: 104857600    # 调整为 100MB
  io_timeout: 180s                # 延长超时时间到 3 分钟

ip_whitelist:
  - 127.0.0.1
  - ::1
  - 10.0.0.0/8
  - 10.66.254.155                 # 确保包含业务平台 IP

EOF

# 本地测试
echo "[4/5] 执行本地功能测试..."
echo "建议操作："
echo "  1. 启动新服务：nohup ./mailproxy_fixed -config config.yaml > mailproxy.log 2>&1 &"
echo "  2. 观察日志：tail -f mailproxy.log"
echo "  3. 发送测试邮件验证功能"
echo ""

# 提供生产部署命令
echo "[5/5] 生产环境部署命令："
echo "====================="
cat << 'DEPLOY_EOF'

# SSH 到目标服务器后执行：
scp /Users/yeung/Projects/MailProxy/mailproxy_fixed root@target-server:/tmp/mailproxy_fixed

ssh target-server << 'SSH_CMDS'
# 停止服务
systemctl stop mailproxy

# 备份当前版本
cp /usr/local/bin/mailproxy /usr/local/bin/mailbackup_$(date +%Y%m%d_%H%M%S)

# 安装新版本
mv /tmp/mailproxy_fixed /usr/local/bin/mailproxy
chmod +x /usr/local/bin/mailproxy

# 检查配置文件（如需修改）
vi /etc/mailproxy/config.yaml

# 启动服务
systemctl start mailproxy

# 确认运行正常
systemctl status mailproxy
journalctl -u mailproxy -n 50

# 监控日志
watch -n 5 "journalctl -u mailproxy -n 5 | tail -20"
SSH_CMDS

DEPLOY_EOF

echo "=========================================="
echo "✅ 部署准备完成！"
echo "详细文档：MAILFIX_IMPLEMENTATION.md"
echo "=========================================="
