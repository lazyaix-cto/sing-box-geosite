// Package asn resolves autonomous-system numbers (from IP-ASN rules) to their
// announced CIDR prefixes, using the public, secret-free ipverse/asn-ip dataset
// hosted on GitHub. Results are cached so a shared ASN is fetched once.
//
// The data source is intentionally swappable: replace baseURL / the fetch func
// to use an online BGP API or an offline GeoLite2-ASN database instead.
package asn

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/lazyaix-cto/sing-box-geosite/internal/source"
)

const baseURL = "https://raw.githubusercontent.com/ipverse/asn-ip/master/as"

type fetchFunc func(ctx context.Context, url string) ([]byte, error)

// Resolver maps ASNs to CIDR prefixes with an in-memory cache.
type Resolver struct {
	mu    sync.Mutex
	cache map[string][]string
	fetch fetchFunc
}

// NewResolver returns a Resolver backed by the ipverse dataset over HTTP.
func NewResolver() *Resolver {
	return &Resolver{cache: map[string][]string{}, fetch: source.Fetch}
}

// Resolve returns the ipv4 + ipv6 CIDR prefixes announced by asnRaw (which may
// be "20473" or "AS20473").
func (r *Resolver) Resolve(ctx context.Context, asnRaw string) ([]string, error) {
	asn := normalize(asnRaw)
	if asn == "" {
		return nil, fmt.Errorf("invalid asn %q", asnRaw)
	}
	r.mu.Lock()
	if v, ok := r.cache[asn]; ok {
		r.mu.Unlock()
		return v, nil
	}
	r.mu.Unlock()

	cidrs, err := r.fetchASN(ctx, asn)
	if err != nil {
		return nil, err
	}
	r.mu.Lock()
	r.cache[asn] = cidrs
	r.mu.Unlock()
	return cidrs, nil
}

func (r *Resolver) fetchASN(ctx context.Context, asn string) ([]string, error) {
	var out []string
	var lastErr error
	got := false
	// ipv6-aggregated.txt is absent for some ASNs; a single file is enough.
	for _, name := range []string{"ipv4-aggregated.txt", "ipv6-aggregated.txt"} {
		data, err := r.fetch(ctx, fmt.Sprintf("%s/%s/%s", baseURL, asn, name))
		if err != nil {
			lastErr = err
			continue
		}
		got = true
		out = append(out, parseCIDRLines(data)...)
	}
	if !got {
		return nil, fmt.Errorf("fetch AS%s: %w", asn, lastErr)
	}
	return out, nil
}

func parseCIDRLines(data []byte) []string {
	var out []string
	sc := bufio.NewScanner(bytes.NewReader(data))
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		out = append(out, line)
	}
	return out
}

func normalize(s string) string {
	s = strings.TrimPrefix(strings.ToUpper(strings.TrimSpace(s)), "AS")
	if s == "" {
		return ""
	}
	for _, c := range s {
		if c < '0' || c > '9' {
			return ""
		}
	}
	return s
}
