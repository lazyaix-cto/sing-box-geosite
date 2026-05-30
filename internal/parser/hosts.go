package parser

import (
	"bufio"
	"bytes"
	"net/netip"
	"strings"

	"github.com/lazyaix/sing-box-geosite/internal/model"
)

func init() { register(hosts{}) }

// hosts parses hosts files (`IP hostname...`) and dnsmasq lines
// (`address=/domain/ip`, `server=/domain/...`). Hosts hostnames become exact
// domains; dnsmasq domains become domain_suffix (dnsmasq `/domain/` matches the
// domain and its subdomains).
type hosts struct{}

func (hosts) Name() string { return "hosts" }

func (hosts) Parse(content []byte) (*Result, error) {
	rs := &model.RuleSet{}
	skipped := map[string]int{}

	sc := bufio.NewScanner(bytes.NewReader(content))
	sc.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "!") {
			continue
		}
		if i := strings.IndexByte(line, '#'); i >= 0 { // trailing comment
			line = strings.TrimSpace(line[:i])
		}

		if strings.HasPrefix(line, "address=/") || strings.HasPrefix(line, "server=/") {
			parts := strings.Split(line, "/")
			matched := false
			for j := 1; j < len(parts)-1; j++ { // first and last are directive/target
				if looksLikeDomain(parts[j]) {
					rs.DomainSuffix = append(rs.DomainSuffix, parts[j])
					matched = true
				}
			}
			if !matched {
				skipped["dnsmasq"]++
			}
			continue
		}

		fields := strings.Fields(line)
		if len(fields) >= 2 {
			if _, err := netip.ParseAddr(fields[0]); err == nil {
				for _, h := range fields[1:] {
					if looksLikeDomain(h) {
						rs.Domain = append(rs.Domain, h)
					} else {
						skipped["nonhost"]++
					}
				}
				continue
			}
		}
		if len(fields) == 1 && looksLikeDomain(fields[0]) {
			rs.Domain = append(rs.Domain, fields[0])
			continue
		}
		skipped["unparsed"]++
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	return &Result{RuleSet: rs, Skipped: skipped}, nil
}
