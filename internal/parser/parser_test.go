package parser

import (
	"reflect"
	"testing"
)

func mustParse(t *testing.T, format string, content string) *Result {
	t.Helper()
	p, err := Get(format)
	if err != nil {
		t.Fatalf("Get(%q): %v", format, err)
	}
	res, err := p.Parse([]byte(content))
	if err != nil {
		t.Fatalf("Parse(%q): %v", format, err)
	}
	return res
}

func TestQuantumultX(t *testing.T) {
	res := mustParse(t, "quantumultx", `# comment
HOST,a.com,Policy
HOST-SUFFIX,b.com,Policy
HOST-KEYWORD,kw,Policy
IP-CIDR,1.2.3.0/24,Policy,no-resolve
IP-ASN,123,Policy
`)
	rs := res.RuleSet
	if !reflect.DeepEqual(rs.Domain, []string{"a.com"}) {
		t.Errorf("Domain = %v", rs.Domain)
	}
	if !reflect.DeepEqual(rs.DomainSuffix, []string{"b.com"}) {
		t.Errorf("DomainSuffix = %v", rs.DomainSuffix)
	}
	if !reflect.DeepEqual(rs.DomainKeyword, []string{"kw"}) {
		t.Errorf("DomainKeyword = %v", rs.DomainKeyword)
	}
	if !reflect.DeepEqual(rs.IPCIDR, []string{"1.2.3.0/24"}) {
		t.Errorf("IPCIDR = %v", rs.IPCIDR)
	}
	if res.Skipped["IP-ASN"] != 1 {
		t.Errorf("Skipped = %v, want IP-ASN x1", res.Skipped)
	}
}

func TestClash(t *testing.T) {
	res := mustParse(t, "clash", `payload:
  - '+.example.com'
  - 'exact.com'
  - 'DOMAIN-SUFFIX,suf.com'
  - 'IP-CIDR,10.0.0.0/8,no-resolve'
`)
	rs := res.RuleSet
	if !reflect.DeepEqual(rs.Domain, []string{"exact.com"}) {
		t.Errorf("Domain = %v", rs.Domain)
	}
	if !reflect.DeepEqual(rs.DomainSuffix, []string{"example.com", "suf.com"}) {
		t.Errorf("DomainSuffix = %v", rs.DomainSuffix)
	}
	if !reflect.DeepEqual(rs.IPCIDR, []string{"10.0.0.0/8"}) {
		t.Errorf("IPCIDR = %v", rs.IPCIDR)
	}
}

func TestSingbox(t *testing.T) {
	res := mustParse(t, "singbox", `{"version":1,"rules":[{"domain":["a.com"],"domain_suffix":["b.com"]}]}`)
	rs := res.RuleSet
	if !reflect.DeepEqual(rs.Domain, []string{"a.com"}) || !reflect.DeepEqual(rs.DomainSuffix, []string{"b.com"}) {
		t.Errorf("got Domain=%v Suffix=%v", rs.Domain, rs.DomainSuffix)
	}
}

func TestDomainList(t *testing.T) {
	res := mustParse(t, "domainlist", `# c
full:exact.com
domain:sub.com
keyword:kw
+.plus.com
plain.com
1.2.3.4
`)
	rs := res.RuleSet
	if !reflect.DeepEqual(rs.Domain, []string{"exact.com"}) {
		t.Errorf("Domain = %v", rs.Domain)
	}
	if !reflect.DeepEqual(rs.DomainSuffix, []string{"sub.com", "plus.com", "plain.com"}) {
		t.Errorf("DomainSuffix = %v", rs.DomainSuffix)
	}
	if !reflect.DeepEqual(rs.DomainKeyword, []string{"kw"}) {
		t.Errorf("DomainKeyword = %v", rs.DomainKeyword)
	}
	if !reflect.DeepEqual(rs.IPCIDR, []string{"1.2.3.4/32"}) {
		t.Errorf("IPCIDR = %v", rs.IPCIDR)
	}
}

func TestDetect(t *testing.T) {
	cases := []struct {
		content, url, want string
	}{
		{"payload:\n  - x.com", "u", "clash"},
		{"{\n  \"version\": 1, \"rules\": []}", "u", "singbox"},
		{"# NAME\nHOST,a.com,P", "u.list", "quantumultx"},
		{"a.com\nb.com", "u.txt", "domainlist"},
	}
	for _, c := range cases {
		if got := Detect([]byte(c.content), c.url); got != c.want {
			t.Errorf("Detect(%q) = %q, want %q", c.content, got, c.want)
		}
	}
}
