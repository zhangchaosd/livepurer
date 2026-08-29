package request

import (
	"fmt"
	"net"
	"net/netip"
	"net/url"
	"strings"
)

// ValidatePublicURL rejects malformed URLs and destinations on local or private networks.
// Callers must still disable redirects or validate every redirected request.
func ValidatePublicURL(raw string, allowedSchemes ...string) (*url.URL, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return nil, fmt.Errorf("parse URL: %w", err)
	}
	if u.User != nil || u.Hostname() == "" {
		return nil, fmt.Errorf("URL must contain a host and no user info")
	}

	allowed := make(map[string]struct{}, len(allowedSchemes))
	for _, scheme := range allowedSchemes {
		allowed[scheme] = struct{}{}
	}
	if _, ok := allowed[strings.ToLower(u.Scheme)]; !ok {
		return nil, fmt.Errorf("unsupported URL scheme %q", u.Scheme)
	}

	host := strings.TrimSuffix(strings.ToLower(u.Hostname()), ".")
	if host == "localhost" || strings.HasSuffix(host, ".localhost") {
		return nil, fmt.Errorf("local destinations are not allowed")
	}
	if ip, err := netip.ParseAddr(host); err == nil {
		if !isPublicIP(ip) {
			return nil, fmt.Errorf("non-public destination is not allowed")
		}
		return u, nil
	}

	ips, err := net.LookupIP(host)
	if err != nil {
		return nil, fmt.Errorf("resolve host: %w", err)
	}
	if len(ips) == 0 {
		return nil, fmt.Errorf("host has no addresses")
	}
	for _, ip := range ips {
		addr, ok := netip.AddrFromSlice(ip)
		if !ok || !isPublicIP(addr.Unmap()) {
			return nil, fmt.Errorf("non-public destination is not allowed")
		}
	}
	return u, nil
}

func isPublicIP(ip netip.Addr) bool {
	return ip.IsValid() && !ip.IsLoopback() && !ip.IsPrivate() && !ip.IsLinkLocalUnicast() &&
		!ip.IsLinkLocalMulticast() && !ip.IsMulticast() && !ip.IsUnspecified()
}
