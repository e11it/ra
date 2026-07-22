package ra

import (
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"strings"
)

const anonymousUser = "anon"

type identitySource struct {
	header         string
	trustedProxies []netip.Prefix
}

func newIdentitySource(cfg *Config) (*identitySource, error) {
	if cfg == nil {
		return nil, fmt.Errorf("identity config is nil")
	}
	header := strings.TrimSpace(cfg.Identity.AuthenticatedUserHeader)
	if header == "" {
		return nil, fmt.Errorf("identity authenticated user header is empty")
	}

	source := &identitySource{
		header:         header,
		trustedProxies: make([]netip.Prefix, 0, len(cfg.Identity.TrustedProxies)),
	}
	for i, raw := range cfg.Identity.TrustedProxies {
		prefix, err := parseTrustedProxy(raw)
		if err != nil {
			return nil, fmt.Errorf("parse trusted proxy [%d] %q: %w", i, raw, err)
		}
		source.trustedProxies = append(source.trustedProxies, prefix)
	}
	return source, nil
}

func parseTrustedProxy(raw string) (netip.Prefix, error) {
	value := strings.TrimSpace(raw)
	if prefix, err := netip.ParsePrefix(value); err == nil {
		return prefix.Masked(), nil
	}
	addr, err := netip.ParseAddr(value)
	if err != nil {
		return netip.Prefix{}, fmt.Errorf("expected ip or cidr: %w", err)
	}
	return netip.PrefixFrom(addr, addr.BitLen()), nil
}

func (s *identitySource) username(req *http.Request) string {
	if s == nil || req == nil || !s.isTrustedPeer(req.RemoteAddr) {
		return anonymousUser
	}
	username := strings.TrimSpace(req.Header.Get(s.header))
	if username == "" {
		return anonymousUser
	}
	return username
}

func (s *identitySource) isTrustedPeer(remoteAddr string) bool {
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		host = remoteAddr
	}
	addr, err := netip.ParseAddr(strings.Trim(host, "[]"))
	if err != nil {
		return false
	}
	addr = addr.Unmap()
	for _, prefix := range s.trustedProxies {
		if prefix.Contains(addr) {
			return true
		}
	}
	return false
}
