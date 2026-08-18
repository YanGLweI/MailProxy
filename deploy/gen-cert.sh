#!/usr/bin/env bash
# 生成自签名 TLS 证书，供 MailProxy 465 端口 SMTP over SSL 使用。
# 用法: ./deploy/gen-cert.sh [输出目录] [主机名]
# 注意: 内网自签名证书需要业务程序信任（导入证书或关闭证书校验），否则会 SSL 握手失败。
set -euo pipefail

OUT_DIR="${1:-certs}"
HOSTNAME_="${2:-mailproxy.internal}"

mkdir -p "$OUT_DIR"
openssl req -x509 -newkey rsa:2048 -sha256 -days 3650 -nodes \
  -keyout "$OUT_DIR/server.key" -out "$OUT_DIR/server.crt" \
  -subj "/CN=${HOSTNAME_}" \
  -addext "subjectAltName=DNS:${HOSTNAME_},DNS:localhost,IP:127.0.0.1"

chmod 600 "$OUT_DIR/server.key"
chmod 644 "$OUT_DIR/server.crt"

echo "证书已生成:"
echo "  $OUT_DIR/server.crt (证书, 需分发给业务程序信任)"
echo "  $OUT_DIR/server.key (私钥, 仅限代理服务器读取)"
