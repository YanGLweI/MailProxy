#!/usr/bin/env bash
# 构建 MailProxy RPM 包（x86_64）。
# 流程：macOS 本机交叉编译 Linux 二进制 -> Docker 容器内 rpmbuild 打包。
# 产物：dist/mailproxy-<version>-1.el*.x86_64.rpm
# 依赖：Go 1.25+、Docker（构建机无需 Linux / rpmbuild 环境）。
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
VERSION="$(tr -d '[:space:]' < "$ROOT/VERSION")"
DIST="$ROOT/dist"
RPMTOP="$DIST/rpmbuild"
IMAGE="rockylinux/rockylinux:9"

command -v go >/dev/null || { echo "错误：未找到 go，请先安装 Go 1.25+" >&2; exit 1; }
command -v docker >/dev/null || { echo "错误：未找到 docker，请先安装 Docker" >&2; exit 1; }
docker info >/dev/null 2>&1 || { echo "错误：Docker 未运行，请先启动 Docker" >&2; exit 1; }

echo "==> 交叉编译 Linux x86_64 二进制 (v$VERSION)"
rm -rf "$RPMTOP"
mkdir -p "$RPMTOP"/{SOURCES,SPECS,BUILD,BUILDROOT,RPMS,SRPMS}
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -trimpath -ldflags "-s -w" \
    -o "$RPMTOP/SOURCES/mailproxy" "$ROOT"

echo "==> 准备打包源文件"
cp "$ROOT/deploy/mailproxy.spec"        "$RPMTOP/SPECS/mailproxy.spec"
cp "$ROOT/deploy/rpm/config.yaml"       "$RPMTOP/SOURCES/config.yaml"
cp "$ROOT/deploy/rpm/mailproxy.service" "$RPMTOP/SOURCES/mailproxy.service"
cp "$ROOT/README.md"                    "$RPMTOP/SOURCES/README.md"

echo "==> Docker 内执行 rpmbuild ($IMAGE, linux/amd64)"
# 强制 amd64 平台：arm64 镜像内的 rpmbuild 无法构建 x86_64 包
docker run --rm --platform linux/amd64 \
    -v "$RPMTOP:/build" \
    -w /build \
    "$IMAGE" \
    bash -c "
        command -v rpmbuild >/dev/null || dnf install -y -q rpm-build
        rpmbuild \
            --define '_topdir /build' \
            --define '_version $VERSION' \
            -bb SPECS/mailproxy.spec
    "

RPM_FILE="$(find "$RPMTOP/RPMS/x86_64" -name "mailproxy-*.rpm" | head -n1)"
cp "$RPM_FILE" "$DIST/"
echo ""
echo "==> 构建完成：dist/$(basename "$RPM_FILE")"
echo "    安装：rpm -ivh dist/$(basename "$RPM_FILE")"
echo "    升级：rpm -Uvh dist/$(basename "$RPM_FILE")"
