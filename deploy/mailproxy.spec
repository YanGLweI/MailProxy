# MailProxy RPM spec
# 构建：由 deploy/build-rpm.sh 预编译二进制后调用 rpmbuild，
# 版本号通过 --define "_version x.y.z" 注入（来源 VERSION 文件）。
%global debug_package %{nil}
%global _build_id_links none
# 最小化基础镜像无 systemd-rpm-macros 时的兜底定义
%{!?_systemd_unitdir: %global _systemd_unitdir /usr/lib/systemd/system}

Name:           mailproxy
Version:        %{_version}
Release:        1%{?dist}
Summary:        MailProxy SMTP relay gateway
License:        Proprietary
BuildArch:      x86_64

Requires:       openssl
Requires(pre):  shadow-utils

%description
Go 实现的 SMTP 邮件代理网关：业务程序统一连接本代理发信，
代理转发到后端真实 SMTP 服务器，支持多账号配置与路由策略。

%prep
# 二进制与资源文件由构建脚本预置到 SOURCES，无需解包；
# -T 不解包、-c 创建并进入 BUILD 目录，供 %doc 使用
%setup -T -c
cp %{_sourcedir}/README.md .

%install
install -Dpm 0755 %{_sourcedir}/mailproxy %{buildroot}%{_bindir}/mailproxy
install -Dpm 0644 %{_sourcedir}/mailproxy.service %{buildroot}%{_systemd_unitdir}/mailproxy.service
install -dm 0750 %{buildroot}%{_sysconfdir}/mailproxy/certs
install -pm 0640 %{_sourcedir}/config.yaml %{buildroot}%{_sysconfdir}/mailproxy/config.yaml
install -dm 0750 %{buildroot}%{_localstatedir}/log/mailproxy

%pre
# 幂等创建服务用户（系统账号，禁止登录）
getent group mailproxy >/dev/null || groupadd -r mailproxy
getent passwd mailproxy >/dev/null || \
    useradd -r -g mailproxy -d %{_sysconfdir}/mailproxy -s /sbin/nologin \
    -c "MailProxy service account" mailproxy
exit 0

%post
# 首次安装生成自签名 TLS 证书；升级时已存在则跳过，不重置用户证书
if [ ! -f %{_sysconfdir}/mailproxy/certs/server.crt ]; then
    openssl req -x509 -newkey rsa:2048 -sha256 -days 3650 -nodes \
        -keyout %{_sysconfdir}/mailproxy/certs/server.key \
        -out %{_sysconfdir}/mailproxy/certs/server.crt \
        -subj "/CN=mailproxy.internal" \
        -addext "subjectAltName=DNS:mailproxy.internal,DNS:localhost,IP:127.0.0.1" \
        >/dev/null 2>&1 || true
fi
chown mailproxy:mailproxy %{_sysconfdir}/mailproxy/certs/server.key %{_sysconfdir}/mailproxy/certs/server.crt 2>/dev/null || true
chmod 600 %{_sysconfdir}/mailproxy/certs/server.key 2>/dev/null || true
chmod 644 %{_sysconfdir}/mailproxy/certs/server.crt 2>/dev/null || true

# 配置开机自启，但不启动服务：需先修改配置文件
if [ -d /run/systemd/system ]; then
    systemctl daemon-reload >/dev/null 2>&1 || true
    systemctl enable mailproxy >/dev/null 2>&1 || true
fi

cat <<'EOF'
======================================================================
 MailProxy 安装完成，服务已设置开机自启但尚未启动。

 下一步：
   1. 修改配置文件（填写后端邮箱账号/授权码、IP 白名单等）：
        vim /etc/mailproxy/config.yaml
   2. 手动启动服务：
        systemctl start mailproxy
   3. 查看运行状态：
        systemctl status mailproxy

 说明：
   - TLS 自签名证书已生成于 /etc/mailproxy/certs/（业务程序需信任
     server.crt；如需自有证书直接覆盖该目录后重启服务）
   - 日志文件：/var/log/mailproxy/mailproxy.log
   - 配置热加载：systemctl reload mailproxy
======================================================================
EOF
exit 0

%preun
if [ "$1" = "0" ]; then
    # 完全卸载：停止并禁用服务
    if [ -d /run/systemd/system ]; then
        systemctl stop mailproxy >/dev/null 2>&1 || true
        systemctl disable mailproxy >/dev/null 2>&1 || true
    fi
fi
exit 0

%postun
if [ -d /run/systemd/system ]; then
    systemctl daemon-reload >/dev/null 2>&1 || true
fi
if [ "$1" = "0" ]; then
    # 完全卸载：删除服务用户（进程/文件占用时忽略失败）
    userdel mailproxy >/dev/null 2>&1 || true
fi
exit 0

%files
%doc README.md
%{_bindir}/mailproxy
%{_systemd_unitdir}/mailproxy.service
%dir %{_sysconfdir}/mailproxy
%dir %attr(0750, root, mailproxy) %{_sysconfdir}/mailproxy/certs
%config(noreplace) %attr(0640, root, mailproxy) %{_sysconfdir}/mailproxy/config.yaml
%dir %attr(0750, mailproxy, mailproxy) %{_localstatedir}/log/mailproxy
