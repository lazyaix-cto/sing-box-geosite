// Package model defines the neutral intermediate representation (IR) that
// every parser produces and every compiler consumes. Decoupling input formats
// from output formats through one shared type is the core of the design: a new
// upstream format only needs a parser, a new output only needs a compiler.
package model

import "sort"

// RuleSet is the format-agnostic intermediate representation.
//
// P0 keeps IP fields as plain strings. P2 will switch IPCIDR/SourceIPCIDR to
// net/netip prefixes so we can do real CIDR aggregation instead of string
// dedup.
type RuleSet struct {
	Domain        []string
	DomainSuffix  []string
	DomainKeyword []string
	DomainRegex   []string
	IPCIDR        []string
	SourceIPCIDR  []string
	Port          []uint16
}

// Merge appends another rule set's entries into r.
func (r *RuleSet) Merge(other *RuleSet) {
	if other == nil {
		return
	}
	r.Domain = append(r.Domain, other.Domain...)
	r.DomainSuffix = append(r.DomainSuffix, other.DomainSuffix...)
	r.DomainKeyword = append(r.DomainKeyword, other.DomainKeyword...)
	r.DomainRegex = append(r.DomainRegex, other.DomainRegex...)
	r.IPCIDR = append(r.IPCIDR, other.IPCIDR...)
	r.SourceIPCIDR = append(r.SourceIPCIDR, other.SourceIPCIDR...)
	r.Port = append(r.Port, other.Port...)
}

// Normalize deduplicates and sorts every field so output is deterministic,
// keeping git diffs minimal across runs. P2 adds cross-category collapse
// (drop domains covered by a suffix) and CIDR merging on top of this.
func (r *RuleSet) Normalize() {
	r.Domain = dedupSortStrings(r.Domain)
	r.DomainSuffix = dedupSortStrings(r.DomainSuffix)
	r.DomainKeyword = dedupSortStrings(r.DomainKeyword)
	r.DomainRegex = dedupSortStrings(r.DomainRegex)
	r.IPCIDR = dedupSortStrings(r.IPCIDR)
	r.SourceIPCIDR = dedupSortStrings(r.SourceIPCIDR)
	r.Port = dedupSortPorts(r.Port)
}

// IsEmpty reports whether the rule set has no entries at all.
func (r *RuleSet) IsEmpty() bool {
	return r.Count() == 0
}

// Count returns the total number of entries across all fields.
func (r *RuleSet) Count() int {
	return len(r.Domain) + len(r.DomainSuffix) + len(r.DomainKeyword) +
		len(r.DomainRegex) + len(r.IPCIDR) + len(r.SourceIPCIDR) + len(r.Port)
}

func dedupSortStrings(in []string) []string {
	if len(in) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(in))
	out := make([]string, 0, len(in))
	for _, v := range in {
		if v == "" {
			continue
		}
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	sort.Strings(out)
	return out
}

func dedupSortPorts(in []uint16) []uint16 {
	if len(in) == 0 {
		return nil
	}
	seen := make(map[uint16]struct{}, len(in))
	out := make([]uint16, 0, len(in))
	for _, v := range in {
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}
