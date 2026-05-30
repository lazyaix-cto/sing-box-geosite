// Command generator builds sing-box rule sets from upstream sources defined in
// sources.yaml. The pipeline is fetch -> parse -> override -> optimize ->
// compile (json + in-process srs). Sources are built concurrently; a failed
// source is reported but never aborts the others.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"

	"golang.org/x/sync/errgroup"

	"github.com/lazyaix/sing-box-geosite/internal/compile"
	"github.com/lazyaix/sing-box-geosite/internal/config"
	"github.com/lazyaix/sing-box-geosite/internal/model"
	"github.com/lazyaix/sing-box-geosite/internal/override"
	"github.com/lazyaix/sing-box-geosite/internal/parser"
	"github.com/lazyaix/sing-box-geosite/internal/source"
	"github.com/lazyaix/sing-box-geosite/internal/transform"
)

type options struct {
	outDir       string
	overridesDir string
	srsVersion   uint8
	optimize     bool
}

func main() {
	var (
		configPath  string
		opt         options
		srsVersion  uint
		concurrency int
	)
	flag.StringVar(&configPath, "config", "rules/sources.yaml", "path to sources config")
	flag.StringVar(&opt.outDir, "out", "rule", "output directory")
	flag.StringVar(&opt.overridesDir, "overrides", "rules/overrides", "per-category override directory")
	flag.UintVar(&srsVersion, "srs-version", 1, "srs binary format version (1, 2 or 3)")
	flag.BoolVar(&opt.optimize, "optimize", true, "apply dedup/collapse/cidr-merge passes")
	flag.IntVar(&concurrency, "concurrency", 8, "max concurrent sources")
	flag.Parse()
	opt.srsVersion = uint8(srsVersion)

	if err := run(configPath, opt, concurrency); err != nil {
		log.Fatalf("fatal: %v", err)
	}
}

type outcome struct {
	category string
	count    int
	err      error
}

func run(configPath string, opt options, concurrency int) error {
	cfg, err := config.Load(configPath)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	if err := os.MkdirAll(opt.outDir, 0o755); err != nil {
		return err
	}

	ctx := context.Background()
	outcomes := make([]outcome, len(cfg.Sources))

	var g errgroup.Group
	if concurrency > 0 {
		g.SetLimit(concurrency)
	}
	for i, src := range cfg.Sources {
		g.Go(func() error {
			count, err := build(ctx, src, opt)
			// Capture per-source; return nil so one failure never cancels siblings.
			outcomes[i] = outcome{src.Category, count, err}
			return nil
		})
	}
	_ = g.Wait()

	var failures, total int
	for _, o := range outcomes {
		if o.err != nil {
			log.Printf("[FAIL] %-12s %v", o.category, o.err)
			failures++
			continue
		}
		total += o.count
	}
	log.Printf("done: %d/%d sources ok, %d entries total -> %s",
		len(cfg.Sources)-failures, len(cfg.Sources), total, opt.outDir)
	if failures > 0 {
		return fmt.Errorf("%d source(s) failed", failures)
	}
	return nil
}

func build(ctx context.Context, src config.Source, opt options) (int, error) {
	merged := &model.RuleSet{}
	skipped := map[string]int{}
	var detected string

	for _, url := range src.URLs {
		data, err := source.Fetch(ctx, url)
		if err != nil {
			return 0, fmt.Errorf("fetch %s: %w", url, err)
		}
		format := src.Format
		if format == "" || format == "auto" {
			format = parser.Detect(data, url)
			detected = format
		}
		p, err := parser.Get(format)
		if err != nil {
			return 0, err
		}
		res, err := p.Parse(data)
		if err != nil {
			return 0, fmt.Errorf("parse %s: %w", url, err)
		}
		merged.Merge(res.RuleSet)
		for k, v := range res.Skipped {
			skipped[k] += v
		}
	}

	ov, err := override.Load(opt.overridesDir, src.Category)
	if err != nil {
		return 0, fmt.Errorf("load override: %w", err)
	}
	if ov != nil {
		ov.Apply(merged)
	}

	var st transform.Stats
	if opt.optimize {
		st = transform.Process(merged)
	} else {
		merged.Normalize()
	}

	jsonBytes, err := compile.JSON(merged, opt.srsVersion)
	if err != nil {
		return 0, fmt.Errorf("render json: %w", err)
	}
	jsonPath := filepath.Join(opt.outDir, src.Category+".json")
	if err := os.WriteFile(jsonPath, append(jsonBytes, '\n'), 0o644); err != nil {
		return 0, err
	}
	srsPath := filepath.Join(opt.outDir, src.Category+".srs")
	if err := compile.SRS(jsonBytes, srsPath, opt.srsVersion); err != nil {
		return 0, fmt.Errorf("compile srs: %w", err)
	}

	logResult(src.Category, merged.Count(), detected, ov != nil, st, skipped)
	return merged.Count(), nil
}

func logResult(category string, count int, detected string, overridden bool, st transform.Stats, skipped map[string]int) {
	note := ""
	if detected != "" {
		note += " (auto:" + detected + ")"
	}
	if overridden {
		note += " (override)"
	}
	if count == 0 {
		log.Printf("[WARN] %-12s produced 0 entries%s", category, note)
	} else {
		log.Printf("[ OK ] %-12s %6d entries%s", category, count, note)
	}
	if st.Changed() {
		log.Printf("        %-12s optimized: -kw %d, -suffix %d, cidr %d->%d, -regex %d",
			category, st.KeywordCollapsed, st.SuffixCollapsed, st.CIDRBefore, st.CIDRAfter, st.RegexDropped)
	}
	if len(skipped) > 0 {
		log.Printf("        %-12s skipped %v", category, sortedSkip(skipped))
	}
}

func sortedSkip(m map[string]int) []string {
	out := make([]string, 0, len(m))
	for k, v := range m {
		out = append(out, fmt.Sprintf("%s x%d", k, v))
	}
	sort.Strings(out)
	return out
}
