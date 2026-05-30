package parser

import (
	"net/netip"
	"strings"
)

// toCIDR normalizes s to a CIDR string when it is an IP or CIDR; ok is false
// for anything else (e.g. a domain). A bare address becomes a /32 or /128.
func toCIDR(s string) (string, bool) {
	if p, err := netip.ParsePrefix(s); err == nil {
		return p.String(), true
	}
	if a, err := netip.ParseAddr(s); err == nil {
		if a.Is4() {
			return a.String() + "/32", true
		}
		return a.String() + "/128", true
	}
	return "", false
}

// looksLikeDomain is a cheap heuristic to avoid turning junk lines into rules.
func looksLikeDomain(s string) bool {
	return strings.Contains(s, ".") && !strings.ContainsAny(s, " \t/\\")
}
