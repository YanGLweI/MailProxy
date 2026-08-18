// Package acl 实现客户端 IP 白名单访问控制，防止代理被当作开放中继。
package acl

import (
	"fmt"
	"net"
	"net/netip"
)

// Whitelist 客户端 IP 白名单。为空表示不限制。
type Whitelist struct {
	nets []netip.Prefix
}

// NewWhitelist 解析白名单条目，支持单 IP（自动补 /32 或 /128）与 CIDR。
func NewWhitelist(entries []string) (*Whitelist, error) {
	wl := &Whitelist{}
	for _, e := range entries {
		p, err := netip.ParsePrefix(e)
		if err != nil {
			addr, err2 := netip.ParseAddr(e)
			if err2 != nil {
				return nil, fmt.Errorf("非法白名单条目 %q: %w", e, err)
			}
			bits := 32
			if addr.Is6() {
				bits = 128
			}
			p = netip.PrefixFrom(addr, bits)
		}
		wl.nets = append(wl.nets, p)
	}
	return wl, nil
}

// Empty 白名单是否未配置（不限制访问）。
func (wl *Whitelist) Empty() bool { return wl == nil || len(wl.nets) == 0 }

// Allows 判断客户端 IP 是否在白名单内。
func (wl *Whitelist) Allows(ip netip.Addr) bool {
	if wl.Empty() {
		return true
	}
	ip = ip.Unmap()
	for _, p := range wl.nets {
		if p.Contains(ip) {
			return true
		}
	}
	return false
}

// IPFromAddr 从 net.Addr 提取客户端 IP。
func IPFromAddr(addr net.Addr) (netip.Addr, bool) {
	ta, ok := addr.(*net.TCPAddr)
	if !ok {
		return netip.Addr{}, false
	}
	a, ok := netip.AddrFromSlice(ta.IP)
	if !ok {
		return netip.Addr{}, false
	}
	return a.Unmap(), true
}
