package parser

import (
	"bufio"
	"bytes"
	"strconv"
	"strings"

	"github.com/lazyaix/sing-box-geosite/internal/model"
)

func init() { register(quantumultX{}) }

// quantumultX parses blackmatrix7-style QuantumultX / Clash-classical rule
// lists: one `TYPE,value[,policy|options]` per line, `#` or `;` for comments.
// It is the dominant format among the upstream sources in sources.yaml.
type quantumultX struct{}

func (quantumultX) Name() string { return "quantumultx" }

func (quantumultX) Parse(content []byte) (*Result, error) {
	rs := &model.RuleSet{}
	skipped := map[string]int{}

	sc := bufio.NewScanner(bytes.NewReader(content))
	sc.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}
		fields := strings.Split(line, ",")
		if len(fields) < 2 {
			continue
		}
		typ := strings.ToUpper(strings.TrimSpace(fields[0]))
		val := strings.TrimSpace(fields[1])
		if val == "" {
			continue
		}
		switch typ {
		case "HOST", "DOMAIN":
			rs.Domain = append(rs.Domain, val)
		case "HOST-SUFFIX", "DOMAIN-SUFFIX":
			rs.DomainSuffix = append(rs.DomainSuffix, val)
		case "HOST-KEYWORD", "DOMAIN-KEYWORD":
			rs.DomainKeyword = append(rs.DomainKeyword, val)
		case "DOMAIN-REGEX":
			rs.DomainRegex = append(rs.DomainRegex, val)
		case "IP-CIDR", "IP-CIDR6", "IP6-CIDR":
			rs.IPCIDR = append(rs.IPCIDR, val)
		case "SRC-IP-CIDR", "SOURCE-IP-CIDR":
			rs.SourceIPCIDR = append(rs.SourceIPCIDR, val)
		case "DST-PORT", "PORT":
			if p, err := strconv.ParseUint(val, 10, 16); err == nil {
				rs.Port = append(rs.Port, uint16(p))
			} else {
				skipped[typ]++
			}
		default:
			// IP-ASN, GEOIP, USER-AGENT, URL-REGEX, logical AND/OR, ... — not
			// representable as-is in a sing-box headless rule-set. Counted, not
			// silently dropped. (URL-REGEX deliberately excluded: sing-box
			// domain_regex matches the host only, not a full URL.) P2/P3 will
			// handle GEOIP/IP-ASN/logical explicitly.
			skipped[typ]++
		}
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	return &Result{RuleSet: rs, Skipped: skipped}, nil
}
