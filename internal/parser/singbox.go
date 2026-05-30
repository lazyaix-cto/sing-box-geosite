package parser

import (
	"encoding/json"

	"github.com/lazyaix/sing-box-geosite/internal/model"
)

func init() { register(singbox{}) }

// singbox parses a native sing-box rule-set in source format, letting us
// re-aggregate existing rule sets. Logical rules are counted as skipped (P2
// will preserve them) rather than flattened into wrong semantics.
type singbox struct{}

func (singbox) Name() string { return "singbox" }

type sbRule struct {
	Type          string   `json:"type"`
	Domain        []string `json:"domain"`
	DomainSuffix  []string `json:"domain_suffix"`
	DomainKeyword []string `json:"domain_keyword"`
	DomainRegex   []string `json:"domain_regex"`
	IPCIDR        []string `json:"ip_cidr"`
	SourceIPCIDR  []string `json:"source_ip_cidr"`
	Port          []uint16 `json:"port"`
	Rules         []sbRule `json:"rules"` // present on logical rules
}

type sbDoc struct {
	Version uint8    `json:"version"`
	Rules   []sbRule `json:"rules"`
}

func (singbox) Parse(content []byte) (*Result, error) {
	var doc sbDoc
	if err := json.Unmarshal(content, &doc); err != nil {
		return nil, err
	}
	rs := &model.RuleSet{}
	skipped := map[string]int{}
	for _, r := range doc.Rules {
		if r.Type == "logical" || len(r.Rules) > 0 {
			skipped["logical"]++
			continue
		}
		rs.Domain = append(rs.Domain, r.Domain...)
		rs.DomainSuffix = append(rs.DomainSuffix, r.DomainSuffix...)
		rs.DomainKeyword = append(rs.DomainKeyword, r.DomainKeyword...)
		rs.DomainRegex = append(rs.DomainRegex, r.DomainRegex...)
		rs.IPCIDR = append(rs.IPCIDR, r.IPCIDR...)
		rs.SourceIPCIDR = append(rs.SourceIPCIDR, r.SourceIPCIDR...)
		rs.Port = append(rs.Port, r.Port...)
	}
	return &Result{RuleSet: rs, Skipped: skipped}, nil
}
