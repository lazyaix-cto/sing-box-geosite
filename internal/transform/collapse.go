package transform

import (
	"strings"

	"github.com/lazyaix-cto/sing-box-geosite/internal/model"
)

// collapseByKeyword drops domain/domain_suffix entries that contain a
// domain_keyword as a substring: such entries already match the keyword rule,
// so removing them is behavior-preserving.
func collapseByKeyword(rs *model.RuleSet) int {
	if len(rs.DomainKeyword) == 0 {
		return 0
	}
	removed := 0
	rs.Domain, removed = filterContaining(rs.Domain, rs.DomainKeyword, removed)
	rs.DomainSuffix, removed = filterContaining(rs.DomainSuffix, rs.DomainKeyword, removed)
	return removed
}

func filterContaining(in, keywords []string, removed int) ([]string, int) {
	out := in[:0]
	for _, v := range in {
		drop := false
		for _, k := range keywords {
			if k != "" && strings.Contains(v, k) {
				drop = true
				break
			}
		}
		if drop {
			removed++
		} else {
			out = append(out, v)
		}
	}
	return out, removed
}

// collapseBySuffix drops entries already covered by a domain_suffix:
//   - a domain equal to, or a sub-domain of, some suffix;
//   - a domain_suffix that is a sub-domain of a shorter suffix.
//
// Coverage is label-aligned (sub-domain), a safe subset of sing-box's string
// suffix matching. Because any dropped entry is covered by a *kept* minimal
// suffix (coverage is transitive), behavior is preserved.
func collapseBySuffix(rs *model.RuleSet) int {
	if len(rs.DomainSuffix) == 0 {
		return 0
	}
	suffixSet := make(map[string]struct{}, len(rs.DomainSuffix))
	for _, s := range rs.DomainSuffix {
		suffixSet[s] = struct{}{}
	}
	removed := 0

	out := rs.Domain[:0]
	for _, d := range rs.Domain {
		if coveredBySuffix(d, suffixSet, true) {
			removed++
		} else {
			out = append(out, d)
		}
	}
	rs.Domain = out

	outS := rs.DomainSuffix[:0]
	for _, s := range rs.DomainSuffix {
		if coveredBySuffix(s, suffixSet, false) {
			removed++
		} else {
			outS = append(outS, s)
		}
	}
	rs.DomainSuffix = outS

	return removed
}

// coveredBySuffix reports whether name is covered by an entry in suffixSet.
// includeSelf counts an exact match (used for domains); otherwise only strictly
// shorter, label-aligned suffixes count (used when collapsing suffixes against
// each other, so a suffix never eliminates itself).
func coveredBySuffix(name string, suffixSet map[string]struct{}, includeSelf bool) bool {
	if includeSelf {
		if _, ok := suffixSet[name]; ok {
			return true
		}
	}
	for i := 0; i < len(name); i++ {
		if name[i] == '.' {
			if _, ok := suffixSet[name[i+1:]]; ok {
				return true
			}
		}
	}
	return false
}
