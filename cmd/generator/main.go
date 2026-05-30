// Command generator builds sing-box rule sets from upstream sources defined in
// sources.yaml. The pipeline is fetch -> parse -> normalize -> compile (json +
// in-process srs). Sources are built concurrently; a failed source is reported
// but never aborts the others.
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
	"github.com/lazyaix/sing-box-geosite/internal/parser"
	"github.com/lazyaix/sing-box-geosite/internal/source"
)

func main() {
	var (
		configPath  string
		outDir      string
		srsVersion  uint
		concurrency int
	)
	flag.StringVar(&configPath, "config", "rules/sources.yaml", "path to sources config")
	flag.StringVar(&outDir, "out", "rule", "output directory")
	flag.UintVar(&srsVersion, "srs-version", 1, "srs binary format version (1, 2 or 3)")
	flag.IntVar(&concurrency, "concurrency", 8, "max concurrent sources")
	flag.Parse()

	if err := run(configPath, outDir, uint8(srsVersion), concurrency); err != nil {
		log.Fatalf("fatal: %v", err)
	}
}

type outcome struct {
	category string
	count    int
	err      error
}

func run(configPath, outDir string, srsVersion uint8, concurrency int) error {
	cfg, err := config.Load(configPath)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	if err := os.MkdirAll(outDir, 0o755); err != nil {
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
			count, err := build(ctx, src, outDir, srsVersion)
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
		len(cfg.Sources)-failures, len(cfg.Sources), total, outDir)
	if failures > 0 {
		return fmt.Errorf("%d source(s) failed", failures)
	}
	return nil
}

func build(ctx context.Context, src config.Source, outDir string, srsVersion uint8) (int, error) {
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
	merged.Normalize()

	jsonBytes, err := compile.JSON(merged, srsVersion)
	if err != nil {
		return 0, fmt.Errorf("render json: %w", err)
	}
	jsonPath := filepath.Join(outDir, src.Category+".json")
	if err := os.WriteFile(jsonPath, append(jsonBytes, '\n'), 0o644); err != nil {
		return 0, err
	}
	srsPath := filepath.Join(outDir, src.Category+".srs")
	if err := compile.SRS(jsonBytes, srsPath, srsVersion); err != nil {
		return 0, fmt.Errorf("compile srs: %w", err)
	}

	note := ""
	if detected != "" {
		note = " (auto:" + detected + ")"
	}
	if merged.Count() == 0 {
		log.Printf("[WARN] %-12s produced 0 entries%s", src.Category, note)
	} else {
		log.Printf("[ OK ] %-12s %5d entries%s", src.Category, merged.Count(), note)
	}
	if len(skipped) > 0 {
		log.Printf("        %-12s skipped %v", src.Category, sortedSkip(skipped))
	}
	return merged.Count(), nil
}

func sortedSkip(m map[string]int) []string {
	out := make([]string, 0, len(m))
	for k, v := range m {
		out = append(out, fmt.Sprintf("%s x%d", k, v))
	}
	sort.Strings(out)
	return out
}
