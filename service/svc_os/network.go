package svc_os

import (
	"bytes"
	"net"
	"sort"
)

// LANIPv4 returns a private IPv4 address from an active non-loopback interface.
// Prefer 192.168/16 home networks, then other private networks, deterministically.
func LANIPv4() string {
	interfaces, err := net.Interfaces()
	if err != nil {
		return ""
	}
	var candidates []net.IP
	for _, iface := range interfaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		addresses, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, address := range addresses {
			ip, _, err := net.ParseCIDR(address.String())
			if err == nil {
				candidates = append(candidates, ip)
			}
		}
	}
	return preferredLANIPv4(candidates)
}

func preferredLANIPv4(candidates []net.IP) string {
	var private []net.IP
	for _, candidate := range candidates {
		ip := candidate.To4()
		if ip != nil && ip.IsPrivate() {
			private = append(private, ip)
		}
	}
	sort.Slice(private, func(i, j int) bool {
		a, b := private[i], private[j]
		a192, b192 := a[0] == 192, b[0] == 192
		if a192 != b192 {
			return a192
		}
		return bytes.Compare(a, b) < 0
	})
	if len(private) == 0 {
		return ""
	}
	return private[0].String()
}
