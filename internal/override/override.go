// Package override applies per-category customization patches loaded from
// rules/overrides/<Category>.yaml: add entries (e.g. your own domains) and
// exclude entries (e.g. false positives) the upstream can't be changed for.
package override

import (
	"errors"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"

	"github.com/lazyaix/sing-box-geosite/internal/model"
)

type patch struct {
	Domain        []string `yaml:"domain"`
	DomainSuffix  []string `yaml:"domain_suffix"`
	DomainKeyword []string `yaml:"domain_keyword"`
	DomainRegex   []string `yaml:"domain_regex"`
	IPCIDR        []string `yaml:"ip_cidr"`
}

// Override is the parsed <Category>.yaml document.
type Override struct {
	Add     patch `yaml:"add"`
	Exclude patch `yaml:"exclude"`
}

// Load reads dir/<category>.yaml (or .yml). Returns (nil, nil) when no override
// file exists for the category.
func Load(dir, category string) (*Override, error) {
	for _, ext := range []string{".yaml", ".yml"} {
		data, err := os.ReadFile(filepath.Join(dir, category+ext))
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return nil, err
		}
		var o Override
		if err := yaml.Unmarshal(data, &o); err != nil {
			return nil, err
		}
		return &o, nil
	}
	return nil, nil
}

// Apply adds then excludes entries in place. Exclusion is exact-match removal
// per field.
func (o *Override) Apply(rs *model.RuleSet) {
	rs.Domain = append(rs.Domain, o.Add.Domain...)
	rs.DomainSuffix = append(rs.DomainSuffix, o.Add.DomainSuffix...)
	rs.DomainKeyword = append(rs.DomainKeyword, o.Add.DomainKeyword...)
	rs.DomainRegex = append(rs.DomainRegex, o.Add.DomainRegex...)
	rs.IPCIDR = append(rs.IPCIDR, o.Add.IPCIDR...)

	rs.Domain = removeAll(rs.Domain, o.Exclude.Domain)
	rs.DomainSuffix = removeAll(rs.DomainSuffix, o.Exclude.DomainSuffix)
	rs.DomainKeyword = removeAll(rs.DomainKeyword, o.Exclude.DomainKeyword)
	rs.DomainRegex = removeAll(rs.DomainRegex, o.Exclude.DomainRegex)
	rs.IPCIDR = removeAll(rs.IPCIDR, o.Exclude.IPCIDR)
}

func removeAll(in, drop []string) []string {
	if len(in) == 0 || len(drop) == 0 {
		return in
	}
	set := make(map[string]struct{}, len(drop))
	for _, d := range drop {
		set[d] = struct{}{}
	}
	out := in[:0]
	for _, v := range in {
		if _, ok := set[v]; !ok {
			out = append(out, v)
		}
	}
	return out
}
