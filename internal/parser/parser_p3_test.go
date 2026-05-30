package parser

import (
	"reflect"
	"testing"
)

func TestAdGuard(t *testing.T) {
	res := mustParse(t, "adguard", `! Title: test
||ads.example.com^
||track.example.com^$third-party
@@||allow.example.com^
||1.2.3.4^
example.org
##.banner
||*.wild.com^
`)
	rs := res.RuleSet
	wantSuffix := []string{"ads.example.com", "track.example.com", "example.org"}
	if !reflect.DeepEqual(rs.DomainSuffix, wantSuffix) {
		t.Errorf("DomainSuffix = %v, want %v", rs.DomainSuffix, wantSuffix)
	}
	if !reflect.DeepEqual(rs.IPCIDR, []string{"1.2.3.4/32"}) {
		t.Errorf("IPCIDR = %v", rs.IPCIDR)
	}
	if res.Skipped["exception"] != 1 || res.Skipped["cosmetic"] != 1 || res.Skipped["wildcard"] != 1 {
		t.Errorf("Skipped = %v", res.Skipped)
	}
}

func TestHosts(t *testing.T) {
	res := mustParse(t, "hosts", `# hosts
0.0.0.0 ads.example.com
127.0.0.1 a.com b.com
address=/dnsmasq.example/0.0.0.0
0.0.0.0 localhost
`)
	rs := res.RuleSet
	wantDomain := []string{"ads.example.com", "a.com", "b.com"}
	if !reflect.DeepEqual(rs.Domain, wantDomain) {
		t.Errorf("Domain = %v, want %v", rs.Domain, wantDomain)
	}
	if !reflect.DeepEqual(rs.DomainSuffix, []string{"dnsmasq.example"}) {
		t.Errorf("DomainSuffix = %v", rs.DomainSuffix)
	}
	if res.Skipped["nonhost"] != 1 { // "localhost" has no dot
		t.Errorf("Skipped = %v, want nonhost x1", res.Skipped)
	}
}

func TestSingboxLogical(t *testing.T) {
	res := mustParse(t, "singbox", `{"version":1,"rules":[
		{"domain_suffix":["leaf.com"]},
		{"type":"logical","mode":"and","rules":[
			{"domain_suffix":["a.com"]},{"port":[443]}
		]}
	]}`)
	rs := res.RuleSet
	if !reflect.DeepEqual(rs.DomainSuffix, []string{"leaf.com"}) {
		t.Errorf("DomainSuffix = %v", rs.DomainSuffix)
	}
	if len(rs.Logical) != 1 || rs.Logical[0].Mode != "and" || len(rs.Logical[0].Rules) != 2 {
		t.Fatalf("Logical = %+v", rs.Logical)
	}
}

func TestDetectAdGuardAndHosts(t *testing.T) {
	if got := Detect([]byte("! Title\n||ads.com^"), "u"); got != "adguard" {
		t.Errorf("adguard detect = %q", got)
	}
	if got := Detect([]byte("0.0.0.0 ads.com\n0.0.0.0 b.com"), "u"); got != "hosts" {
		t.Errorf("hosts detect = %q", got)
	}
}
