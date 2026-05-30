// Package config loads sources.yaml, the declarative replacement for the
// original project's bare links.txt. Each source carries an explicit category
// (output filename) and optional format, instead of deriving the name from a
// URL's basename.
package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Source is one output rule set, built by merging one or more upstream URLs.
type Source struct {
	Category string   `yaml:"category"` // output basename: <category>.json/.srs
	Format   string   `yaml:"format"`   // parser name; "" or "auto" -> auto-detect
	URLs     []string `yaml:"urls"`
}

// Config is the top-level sources.yaml document.
type Config struct {
	Sources []Source `yaml:"sources"`
}

// Load reads and validates the config at path.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var c Config
	if err := yaml.Unmarshal(data, &c); err != nil {
		return nil, err
	}
	if len(c.Sources) == 0 {
		return nil, fmt.Errorf("no sources defined")
	}
	for i, s := range c.Sources {
		if s.Category == "" {
			return nil, fmt.Errorf("sources[%d]: category is required", i)
		}
		if len(s.URLs) == 0 {
			return nil, fmt.Errorf("source %q: at least one url is required", s.Category)
		}
	}
	return &c, nil
}
