package parser

import (
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/lazyaix/sing-box-geosite/internal/model"
)

func init() { register(clash{}) }

// clash parses Clash rule-provider YAML (a top-level `payload:` list). It
// handles all three behaviors transparently: items with a known TYPE prefix are
// treated as classical (delegated to applyClassical); bare items are treated as
// domain-behavior (+./. -> suffix, plain -> domain, CIDR -> ip_cidr).
type clash struct{}

func (clash) Name() string { return "clash" }

type clashPayload struct {
	Payload []string `yaml:"payload"`
}

func (clash) Parse(content []byte) (*Result, error) {
	var doc clashPayload
	if err := yaml.Unmarshal(content, &doc); err != nil {
		return nil, err
	}
	rs := &model.RuleSet{}
	skipped := map[string]int{}

	for _, raw := range doc.Payload {
		item := strings.Trim(strings.TrimSpace(raw), "'\"")
		if item == "" || strings.HasPrefix(item, "#") {
			continue
		}
		if idx := strings.IndexByte(item, ','); idx >= 0 {
			typ := strings.ToUpper(strings.TrimSpace(item[:idx]))
			val := strings.TrimSpace(item[idx+1:])
			if c := strings.IndexByte(val, ','); c >= 0 { // strip trailing options
				val = strings.TrimSpace(val[:c])
			}
			if val == "" || !applyClassical(rs, typ, val) {
				skipped[typ]++
			}
			continue
		}
		applyClashDomain(rs, item)
	}
	return &Result{RuleSet: rs, Skipped: skipped}, nil
}

func applyClashDomain(rs *model.RuleSet, item string) {
	switch {
	case strings.HasPrefix(item, "+."):
		rs.DomainSuffix = append(rs.DomainSuffix, item[2:])
	case strings.HasPrefix(item, "*."):
		// single-level wildcard; approximated as a suffix on the remainder.
		rs.DomainSuffix = append(rs.DomainSuffix, item[2:])
	case strings.HasPrefix(item, "."):
		rs.DomainSuffix = append(rs.DomainSuffix, item[1:])
	default:
		if cidr, ok := toCIDR(item); ok {
			rs.IPCIDR = append(rs.IPCIDR, cidr)
		} else {
			rs.Domain = append(rs.Domain, item)
		}
	}
}
