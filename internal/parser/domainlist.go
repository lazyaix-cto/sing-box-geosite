package parser

import (
	"bufio"
	"bytes"
	"strings"

	"github.com/lazyaix/sing-box-geosite/internal/model"
)

func init() { register(domainList{}) }

// domainList parses plain domain lists (one entry per line). It supports the
// v2fly/dlc attribute prefixes full:/domain:/keyword:/regexp: and +./.
// suffixes. A bare line is treated as a domain_suffix, matching the v2fly
// "domain" convention (the domain and its subdomains).
type domainList struct{}

func (domainList) Name() string { return "domainlist" }

func (domainList) Parse(content []byte) (*Result, error) {
	rs := &model.RuleSet{}
	skipped := map[string]int{}

	sc := bufio.NewScanner(bytes.NewReader(content))
	sc.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "!") {
			continue
		}
		// Drop inline attributes/comments, e.g. v2fly "domain.com @ads".
		if i := strings.IndexAny(line, " \t"); i >= 0 {
			line = line[:i]
		}
		switch {
		case strings.HasPrefix(line, "full:"):
			rs.Domain = append(rs.Domain, line[len("full:"):])
		case strings.HasPrefix(line, "domain:"):
			rs.DomainSuffix = append(rs.DomainSuffix, line[len("domain:"):])
		case strings.HasPrefix(line, "keyword:"):
			rs.DomainKeyword = append(rs.DomainKeyword, line[len("keyword:"):])
		case strings.HasPrefix(line, "regexp:"):
			rs.DomainRegex = append(rs.DomainRegex, line[len("regexp:"):])
		case strings.HasPrefix(line, "+."):
			rs.DomainSuffix = append(rs.DomainSuffix, line[2:])
		case strings.HasPrefix(line, "."):
			rs.DomainSuffix = append(rs.DomainSuffix, line[1:])
		default:
			if cidr, ok := toCIDR(line); ok {
				rs.IPCIDR = append(rs.IPCIDR, cidr)
			} else if looksLikeDomain(line) {
				rs.DomainSuffix = append(rs.DomainSuffix, line)
			} else {
				skipped["unparsed"]++
			}
		}
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	return &Result{RuleSet: rs, Skipped: skipped}, nil
}
