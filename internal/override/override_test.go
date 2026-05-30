package override

import (
	"sort"
	"testing"

	"github.com/lazyaix-cto/sing-box-geosite/internal/model"
)

func TestApply(t *testing.T) {
	o := &Override{
		Add:     patch{DomainSuffix: []string{"new.com"}, Domain: []string{"add.exact"}},
		Exclude: patch{DomainSuffix: []string{"bad.com"}},
	}
	rs := &model.RuleSet{
		DomainSuffix: []string{"bad.com", "ok.com"},
	}
	o.Apply(rs)

	sort.Strings(rs.DomainSuffix)
	want := []string{"new.com", "ok.com"}
	if len(rs.DomainSuffix) != len(want) {
		t.Fatalf("DomainSuffix = %v, want %v", rs.DomainSuffix, want)
	}
	for i := range want {
		if rs.DomainSuffix[i] != want[i] {
			t.Errorf("DomainSuffix = %v, want %v", rs.DomainSuffix, want)
		}
	}
	if len(rs.Domain) != 1 || rs.Domain[0] != "add.exact" {
		t.Errorf("Domain = %v, want [add.exact]", rs.Domain)
	}
}

func TestLoadAbsent(t *testing.T) {
	o, err := Load(t.TempDir(), "Nonexistent")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if o != nil {
		t.Errorf("expected nil override, got %+v", o)
	}
}
