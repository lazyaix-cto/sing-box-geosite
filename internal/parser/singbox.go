package parser

import (
	"encoding/json"

	"github.com/lazyaix-cto/sing-box-geosite/internal/model"
)

func init() { register(singbox{}) }

// singbox parses a native sing-box rule-set in source format, letting us
// re-aggregate existing rule sets. Logical rules are preserved into the IR.
type singbox struct{}

func (singbox) Name() string { return "singbox" }

type sbRule struct {
	Type          string   `json:"type"`
	Mode          string   `json:"mode"`
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
			lr := model.LogicalRule{Mode: r.Mode}
			for _, nr := range r.Rules {
				lr.Rules = append(lr.Rules, leafToRuleSet(nr))
			}
			if len(lr.Rules) > 0 {
				rs.Logical = append(rs.Logical, lr)
			} else {
				skipped["logical"]++
			}
			continue
		}
		leaf := leafToRuleSet(r)
		rs.Merge(leaf)
	}
	return &Result{RuleSet: rs, Skipped: skipped}, nil
}

func leafToRuleSet(r sbRule) *model.RuleSet {
	return &model.RuleSet{
		Domain:        r.Domain,
		DomainSuffix:  r.DomainSuffix,
		DomainKeyword: r.DomainKeyword,
		DomainRegex:   r.DomainRegex,
		IPCIDR:        r.IPCIDR,
		SourceIPCIDR:  r.SourceIPCIDR,
		Port:          r.Port,
	}
}
