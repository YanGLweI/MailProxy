package acl

import (
	"net"
	"net/netip"
	"testing"
)

func TestNewWhitelistInvalid(t *testing.T) {
	if _, err := NewWhitelist([]string{"not-an-ip"}); err == nil {
		t.Fatal("期望非法条目报错")
	}
}

func TestWhitelistAllows(t *testing.T) {
	wl, err := NewWhitelist([]string{"127.0.0.1", "10.0.0.0/8", "192.168.1.0/24", "::1"})
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		ip   string
		want bool
	}{
		{"127.0.0.1", true},
		{"127.0.0.2", false},
		{"10.1.2.3", true},
		{"11.0.0.1", false},
		{"192.168.1.55", true},
		{"192.168.2.55", false},
		{"::1", true},
		{"8.8.8.8", false},
		{"::ffff:10.1.1.1", true}, // IPv4-mapped 地址应先 Unmap 再匹配
	}
	for _, c := range cases {
		if got := wl.Allows(netip.MustParseAddr(c.ip)); got != c.want {
			t.Errorf("Allows(%s) = %v, want %v", c.ip, got, c.want)
		}
	}
}

func TestWhitelistEmptyAllowsAll(t *testing.T) {
	wl, err := NewWhitelist(nil)
	if err != nil {
		t.Fatal(err)
	}
	if !wl.Empty() || !wl.Allows(netip.MustParseAddr("203.0.113.9")) {
		t.Error("空白名单应放行全部")
	}
}

func TestIPFromAddr(t *testing.T) {
	ip, ok := IPFromAddr(&net.TCPAddr{IP: net.ParseIP("10.2.3.4"), Port: 1234})
	if !ok || ip.String() != "10.2.3.4" {
		t.Errorf("IPFromAddr = %v, %v", ip, ok)
	}
	if _, ok := IPFromAddr(&net.UDPAddr{}); ok {
		t.Error("非 TCP 地址应返回 false")
	}
}
