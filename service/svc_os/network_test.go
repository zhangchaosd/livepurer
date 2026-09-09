package svc_os

import (
	"net"
	"testing"
)

func TestPreferredLANIPv4(t *testing.T) {
	for _, tt := range []struct {
		name string
		ips  []string
		want string
	}{
		{"prefer home network", []string{"10.0.0.2", "172.16.0.3", "192.168.1.9"}, "192.168.1.9"},
		{"stable ordering", []string{"192.168.2.4", "192.168.1.9", "192.168.1.3"}, "192.168.1.3"},
		{"other private network", []string{"172.20.1.4", "10.1.1.2"}, "10.1.1.2"},
		{"ignore non LAN", []string{"127.0.0.1", "0.0.0.0", "169.254.1.2", "192.0.2.1", "8.8.8.8", "::1", "fd00::1", "invalid"}, ""},
		{"no interfaces", nil, ""},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var ips []net.IP
			for _, value := range tt.ips {
				ips = append(ips, net.ParseIP(value))
			}
			if got := preferredLANIPv4(ips); got != tt.want {
				t.Fatalf("got %q, want %q", got, tt.want)
			}
		})
	}
}
