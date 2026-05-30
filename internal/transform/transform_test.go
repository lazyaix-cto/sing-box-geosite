package transform

import (
	"sort"
	"testing"

	"github.com/lazyaix/sing-box-geosite/internal/model"
)

func equalSet(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	x := append([]string(nil), a...)
	y := append([]string(nil), b...)
	sort.Strings(x)
	sort.Strings(y)
	for i := range x {
		if x[i] != y[i] {
			return false
		}
	}
	return true
}

func TestCollapseBySuffix(t *testing.T) {
	rs := &model.RuleSet{
		Domain:       []string{"a.openai.com", "keep.org"},
		DomainSuffix: []string{"openai.com", "sub.openai.com"},
	}
	Process(rs)
	if !equalSet(rs.Domain, []string{"keep.org"}) {
		t.Errorf("Domain = %v, want [keep.org]", rs.Domain)
	}
	if !equalSet(rs.DomainSuffix, []string{"openai.com"}) {
		t.Errorf("DomainSuffix = %v, want [openai.com]", rs.DomainSuffix)
	}
}

func TestCollapseByKeyword(t *testing.T) {
	rs := &model.RuleSet{
		DomainKeyword: []string{"openai"},
		DomainSuffix:  []string{"openai.com", "other.net"},
		Domain:        []string{"x.openai.com.cdn.net"},
	}
	Process(rs)
	if len(rs.Domain) != 0 {
		t.Errorf("Domain = %v, want empty", rs.Domain)
	}
	if !equalSet(rs.DomainSuffix, []string{"other.net"}) {
		t.Errorf("DomainSuffix = %v, want [other.net]", rs.DomainSuffix)
	}
	if !equalSet(rs.DomainKeyword, []string{"openai"}) {
		t.Errorf("DomainKeyword = %v", rs.DomainKeyword)
	}
}

func TestMergeCIDR(t *testing.T) {
	rs := &model.RuleSet{
		IPCIDR: []string{"1.2.3.0/25", "1.2.3.128/25", "1.2.3.0/24", "1.2.3.4"},
	}
	st := Process(rs)
	if !equalSet(rs.IPCIDR, []string{"1.2.3.0/24"}) {
		t.Errorf("IPCIDR = %v, want [1.2.3.0/24]", rs.IPCIDR)
	}
	if st.CIDRAfter != 1 {
		t.Errorf("CIDRAfter = %d, want 1", st.CIDRAfter)
	}
}

func TestDropInvalidRegex(t *testing.T) {
	rs := &model.RuleSet{
		DomainRegex: []string{`^a.*\.com$`, `(unclosed`},
	}
	st := Process(rs)
	if !equalSet(rs.DomainRegex, []string{`^a.*\.com$`}) {
		t.Errorf("DomainRegex = %v", rs.DomainRegex)
	}
	if st.RegexDropped != 1 {
		t.Errorf("RegexDropped = %d, want 1", st.RegexDropped)
	}
}
