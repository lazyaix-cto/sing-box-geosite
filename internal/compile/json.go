// Package compile renders the IR into sing-box outputs: source-format JSON and
// the binary .srs (compiled in-process via sing-box's own writer).
package compile

import (
	"encoding/json"

	"github.com/lazyaix-cto/sing-box-geosite/internal/model"
)

// sourceRule mirrors a sing-box headless rule. Leaf rules carry domain/ip
// fields; a logical rule carries Type="logical", Mode and nested Rules.
type sourceRule struct {
	Type          string       `json:"type,omitempty"`
	Mode          string       `json:"mode,omitempty"`
	Rules         []sourceRule `json:"rules,omitempty"`
	Domain        []string     `json:"domain,omitempty"`
	DomainSuffix  []string     `json:"domain_suffix,omitempty"`
	DomainKeyword []string     `json:"domain_keyword,omitempty"`
	DomainRegex   []string     `json:"domain_regex,omitempty"`
	IPCIDR        []string     `json:"ip_cidr,omitempty"`
	SourceIPCIDR  []string     `json:"source_ip_cidr,omitempty"`
	Port          []uint16     `json:"port,omitempty"`
}

type sourceRuleSet struct {
	Version uint8        `json:"version"`
	Rules   []sourceRule `json:"rules"`
}

func leaf(rs *model.RuleSet) sourceRule {
	return sourceRule{
		Domain:        rs.Domain,
		DomainSuffix:  rs.DomainSuffix,
		DomainKeyword: rs.DomainKeyword,
		DomainRegex:   rs.DomainRegex,
		IPCIDR:        rs.IPCIDR,
		SourceIPCIDR:  rs.SourceIPCIDR,
		Port:          rs.Port,
	}
}

func leafEmpty(r sourceRule) bool {
	return len(r.Domain)+len(r.DomainSuffix)+len(r.DomainKeyword)+len(r.DomainRegex)+
		len(r.IPCIDR)+len(r.SourceIPCIDR)+len(r.Port) == 0
}

// JSON renders rs as sing-box source-format JSON for the given rule-set format
// version. The merged leaf fields become one default rule; each logical rule
// becomes a separate {type:"logical"} rule.
func JSON(rs *model.RuleSet, version uint8) ([]byte, error) {
	rules := make([]sourceRule, 0, 1+len(rs.Logical))
	if l := leaf(rs); !leafEmpty(l) {
		rules = append(rules, l)
	}
	for _, lr := range rs.Logical {
		sub := make([]sourceRule, 0, len(lr.Rules))
		for _, s := range lr.Rules {
			sub = append(sub, leaf(s))
		}
		mode := lr.Mode
		if mode == "" {
			mode = "and"
		}
		rules = append(rules, sourceRule{Type: "logical", Mode: mode, Rules: sub})
	}
	doc := sourceRuleSet{Version: version, Rules: rules}
	return json.MarshalIndent(doc, "", "  ")
}
