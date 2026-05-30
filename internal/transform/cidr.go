package transform

import (
	"net/netip"

	"go4.org/netipx"
)

// mergeCIDRField aggregates CIDR/IP strings into the minimal covering set of
// prefixes (overlapping and adjacent ranges are merged). Invalid entries are
// dropped and counted rather than left to fail srs compilation. Returns the
// number dropped.
func mergeCIDRField(field *[]string) int {
	in := *field
	if len(in) == 0 {
		return 0
	}
	var b netipx.IPSetBuilder
	dropped := 0
	for _, s := range in {
		p, err := netip.ParsePrefix(s)
		if err != nil {
			a, err2 := netip.ParseAddr(s)
			if err2 != nil {
				dropped++
				continue
			}
			p = netip.PrefixFrom(a, a.BitLen())
		}
		b.AddPrefix(p.Masked())
	}
	set, err := b.IPSet()
	if err != nil {
		return dropped
	}
	prefixes := set.Prefixes()
	out := make([]string, 0, len(prefixes))
	for _, p := range prefixes {
		out = append(out, p.String())
	}
	*field = out
	return dropped
}
