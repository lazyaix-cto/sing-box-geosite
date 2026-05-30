package asn

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"testing"
)

func TestResolve(t *testing.T) {
	calls := 0
	r := &Resolver{
		cache: map[string][]string{},
		fetch: func(ctx context.Context, url string) ([]byte, error) {
			calls++
			switch {
			case strings.HasSuffix(url, "/123/ipv4-aggregated.txt"):
				return []byte("# AS123\n1.2.0.0/16\n3.4.0.0/16\n"), nil
			case strings.HasSuffix(url, "/123/ipv6-aggregated.txt"):
				return []byte("2001:db8::/32\n"), nil
			}
			return nil, fmt.Errorf("404")
		},
	}

	got, err := r.Resolve(context.Background(), "AS123")
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"1.2.0.0/16", "3.4.0.0/16", "2001:db8::/32"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Resolve = %v, want %v", got, want)
	}

	// Second call must hit the cache (no extra fetches).
	before := calls
	if _, err := r.Resolve(context.Background(), "123"); err != nil {
		t.Fatal(err)
	}
	if calls != before {
		t.Errorf("expected cache hit, got %d extra fetches", calls-before)
	}
}

func TestResolveInvalid(t *testing.T) {
	r := NewResolver()
	if _, err := r.Resolve(context.Background(), "not-an-asn"); err == nil {
		t.Error("expected error for invalid asn")
	}
}
