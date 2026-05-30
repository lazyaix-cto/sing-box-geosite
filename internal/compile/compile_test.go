package compile

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lazyaix/sing-box-geosite/internal/model"
)

// TestLogicalCompiles drives a logical rule through JSON -> in-process srs ->
// read-back verification, proving sing-box accepts the output end to end.
func TestLogicalCompiles(t *testing.T) {
	rs := &model.RuleSet{
		DomainSuffix: []string{"example.com"},
		Logical: []model.LogicalRule{{
			Mode: "and",
			Rules: []*model.RuleSet{
				{DomainSuffix: []string{"a.com"}},
				{Port: []uint16{443}},
			},
		}},
	}
	js, err := JSON(rs, 1)
	if err != nil {
		t.Fatalf("JSON: %v", err)
	}
	if !strings.Contains(string(js), `"logical"`) || !strings.Contains(string(js), `"and"`) {
		t.Fatalf("json missing logical rule:\n%s", js)
	}

	path := filepath.Join(t.TempDir(), "out.srs")
	if err := SRS(js, path, 1); err != nil {
		t.Fatalf("SRS: %v", err)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(b) < 4 || string(b[:3]) != "SRS" {
		t.Fatalf("bad srs magic: %x", b[:min(4, len(b))])
	}
}

func TestLeafOnlyCompiles(t *testing.T) {
	rs := &model.RuleSet{DomainSuffix: []string{"example.com"}, IPCIDR: []string{"1.2.3.0/24"}}
	js, err := JSON(rs, 1)
	if err != nil {
		t.Fatal(err)
	}
	if err := SRS(js, filepath.Join(t.TempDir(), "o.srs"), 1); err != nil {
		t.Fatalf("SRS: %v", err)
	}
}
