// Command generator builds sing-box rule sets from upstream sources defined in
// sources.yaml. P0 runs the full pipeline (fetch -> parse -> normalize ->
// compile json + srs) sequentially for the configured categories; P1 adds
// concurrent fetching and more parsers.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/lazyaix/sing-box-geosite/internal/compile"
	"github.com/lazyaix/sing-box-geosite/internal/config"
	"github.com/lazyaix/sing-box-geosite/internal/model"
	"github.com/lazyaix/sing-box-geosite/internal/parser"
	"github.com/lazyaix/sing-box-geosite/internal/source"
)

func main() {
	var (
		configPath string
		outDir     string
		srsVersion uint
	)
	flag.StringVar(&configPath, "config", "rules/sources.yaml", "path to sources config")
	flag.StringVar(&outDir, "out", "rule", "output directory")
	flag.UintVar(&srsVersion, "srs-version", 1, "srs binary format version (1, 2 or 3)")
	flag.Parse()

	if err := run(configPath, outDir, uint8(srsVersion)); err != nil {
		log.Fatalf("fatal: %v", err)
	}
}

func run(configPath, outDir string, srsVersion uint8) error {
	cfg, err := config.Load(configPath)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return err
	}

	ctx := context.Background()
	var failures int
	for _, src := range cfg.Sources {
		if err := build(ctx, src, outDir, srsVersion); err != nil {
			log.Printf("[FAIL] %s: %v", src.Category, err)
			failures++
		}
	}
	if failures > 0 {
		return fmt.Errorf("%d of %d sources failed", failures, len(cfg.Sources))
	}
	log.Printf("done: %d sources -> %s", len(cfg.Sources), outDir)
	return nil
}

func build(ctx context.Context, src config.Source, outDir string, srsVersion uint8) error {
	format := src.Format
	if format == "" || format == "auto" {
		// P0 only ships one parser; P1 adds content sniffing for real auto-detect.
		format = "quantumultx"
	}
	p, err := parser.Get(format)
	if err != nil {
		return err
	}

	merged := &model.RuleSet{}
	for _, url := range src.URLs {
		start := time.Now()
		data, err := source.Fetch(ctx, url)
		if err != nil {
			return fmt.Errorf("fetch %s: %w", url, err)
		}
		res, err := p.Parse(data)
		if err != nil {
			return fmt.Errorf("parse %s: %w", url, err)
		}
		merged.Merge(res.RuleSet)
		if len(res.Skipped) > 0 {
			log.Printf("  [skip] %s: unsupported types %v", src.Category, sortedSkip(res.Skipped))
		}
		log.Printf("  fetched %s (%d entries, %s)", url, res.RuleSet.Count(), time.Since(start).Round(time.Millisecond))
	}
	merged.Normalize()

	jsonBytes, err := compile.JSON(merged, srsVersion)
	if err != nil {
		return fmt.Errorf("render json: %w", err)
	}
	jsonPath := filepath.Join(outDir, src.Category+".json")
	if err := os.WriteFile(jsonPath, append(jsonBytes, '\n'), 0o644); err != nil {
		return err
	}
	srsPath := filepath.Join(outDir, src.Category+".srs")
	if err := compile.SRS(jsonBytes, srsPath, srsVersion); err != nil {
		return fmt.Errorf("compile srs: %w", err)
	}

	log.Printf("[ OK ] %s -> %s + %s (%d entries)", src.Category, jsonPath, srsPath, merged.Count())
	return nil
}

func sortedSkip(m map[string]int) []string {
	out := make([]string, 0, len(m))
	for k, v := range m {
		out = append(out, fmt.Sprintf("%s x%d", k, v))
	}
	sort.Strings(out)
	return out
}
