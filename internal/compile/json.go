// Package compile renders the IR into sing-box outputs: source-format JSON and
// the binary .srs (compiled in-process via sing-box's own writer).
package compile

import (
	"encoding/json"

	"github.com/lazyaix/sing-box-geosite/internal/model"
)

// sourceRule mirrors a sing-box headless rule. All domain/ip fields live in one
// rule object — the canonical form `sing-box rule-set convert` emits.
type sourceRule struct {
	Domain        []string `json:"domain,omitempty"`
	DomainSuffix  []string `json:"domain_suffix,omitempty"`
	DomainKeyword []string `json:"domain_keyword,omitempty"`
	DomainRegex   []string `json:"domain_regex,omitempty"`
	IPCIDR        []string `json:"ip_cidr,omitempty"`
	SourceIPCIDR  []string `json:"source_ip_cidr,omitempty"`
	Port          []uint16 `json:"port,omitempty"`
}

type sourceRuleSet struct {
	Version uint8        `json:"version"`
	Rules   []sourceRule `json:"rules"`
}

// JSON renders rs as sing-box source-format JSON for the given rule-set format
// version (1/2/3). An empty rule set yields an empty rules array.
func JSON(rs *model.RuleSet, version uint8) ([]byte, error) {
	doc := sourceRuleSet{Version: version, Rules: []sourceRule{}}
	if !rs.IsEmpty() {
		doc.Rules = []sourceRule{{
			Domain:        rs.Domain,
			DomainSuffix:  rs.DomainSuffix,
			DomainKeyword: rs.DomainKeyword,
			DomainRegex:   rs.DomainRegex,
			IPCIDR:        rs.IPCIDR,
			SourceIPCIDR:  rs.SourceIPCIDR,
			Port:          rs.Port,
		}}
	}
	return json.MarshalIndent(doc, "", "  ")
}
