// Package parser turns raw upstream rule bytes into the neutral IR
// (model.RuleSet). Each upstream format implements the Parser interface and
// registers itself via init(); the generator selects one by name from
// sources.yaml.
package parser

import (
	"fmt"
	"sort"

	"github.com/lazyaix/sing-box-geosite/internal/model"
)

// Result is what a Parser returns: the parsed rules plus a record of rule
// types it deliberately dropped. Surfacing Skipped (instead of swallowing it
// like the original pandas script) is a core goal — silent drops hide upstream
// format changes.
type Result struct {
	RuleSet *model.RuleSet
	Skipped map[string]int // unsupported rule type -> count
}

// Parser converts one upstream rule format into the neutral IR.
type Parser interface {
	// Name is the format identifier used in sources.yaml (format: <name>).
	Name() string
	// Parse converts raw source bytes into a Result.
	Parse(content []byte) (*Result, error)
}

var registry = map[string]Parser{}

func register(p Parser) { registry[p.Name()] = p }

// Get returns the parser registered for the named format.
func Get(format string) (Parser, error) {
	p, ok := registry[format]
	if !ok {
		return nil, fmt.Errorf("unknown format %q (known: %v)", format, Formats())
	}
	return p, nil
}

// Formats lists all registered format names, sorted.
func Formats() []string {
	out := make([]string, 0, len(registry))
	for name := range registry {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}
