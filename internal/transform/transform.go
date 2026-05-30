// Package transform applies behavior-preserving optimization passes to the IR:
// drop invalid regex, collapse domains/suffixes made redundant by a keyword or
// a broader suffix, and aggregate CIDRs. Every pass only removes rules that are
// already covered by a rule that stays, so matching behavior is unchanged.
package transform

import (
	"regexp"

	"github.com/lazyaix-cto/sing-box-geosite/internal/model"
)

// Stats records what the passes changed, for transparent logging.
type Stats struct {
	KeywordCollapsed int
	SuffixCollapsed  int
	CIDRBefore       int
	CIDRAfter        int
	CIDRDropped      int
	RegexDropped     int
}

// Changed reports whether any pass altered the rule set.
func (s Stats) Changed() bool {
	return s.KeywordCollapsed != 0 || s.SuffixCollapsed != 0 ||
		s.CIDRDropped != 0 || s.RegexDropped != 0 || s.CIDRBefore != s.CIDRAfter
}

// Process runs all passes in place and returns what changed. Order matters:
// normalize first so passes see deduped input; keyword collapse before suffix
// collapse; CIDR aggregation last; a final normalize keeps output sorted.
func Process(rs *model.RuleSet) Stats {
	rs.Normalize()

	var st Stats
	st.RegexDropped = dropInvalidRegex(rs)
	st.KeywordCollapsed = collapseByKeyword(rs)
	st.SuffixCollapsed = collapseBySuffix(rs)

	st.CIDRBefore = len(rs.IPCIDR) + len(rs.SourceIPCIDR)
	st.CIDRDropped = mergeCIDRField(&rs.IPCIDR) + mergeCIDRField(&rs.SourceIPCIDR)
	st.CIDRAfter = len(rs.IPCIDR) + len(rs.SourceIPCIDR)

	rs.Normalize()
	return st
}

// dropInvalidRegex removes domain_regex entries that don't compile (RE2, the
// same engine sing-box uses), so one bad upstream regex can't fail the whole
// category at srs-compile time.
func dropInvalidRegex(rs *model.RuleSet) int {
	if len(rs.DomainRegex) == 0 {
		return 0
	}
	out := rs.DomainRegex[:0]
	dropped := 0
	for _, r := range rs.DomainRegex {
		if _, err := regexp.Compile(r); err != nil {
			dropped++
			continue
		}
		out = append(out, r)
	}
	rs.DomainRegex = out
	return dropped
}
