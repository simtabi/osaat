// Package macos implements the macOS application collector. It is
// designed to run on darwin; on other platforms Collect() returns a
// clear error rather than producing partial data.
package macos

import (
	"context"
	"fmt"
	"log/slog"
	"runtime"

	"github.com/simtabi/osaat/internal/audit"
	"github.com/simtabi/osaat/internal/collectors"
)

// Collector is the macOS implementation of collectors.Collector.
// Construct it with New() and customize behavior via the Option helpers.
type Collector struct {
	runCmd   collectors.RunCmd
	log      *slog.Logger
	insights map[string]bool
}

// Option mutates a Collector during construction.
type Option func(*Collector)

// WithRunCmd injects a subprocess runner. Tests use this to return
// canned output without spawning real processes.
func WithRunCmd(r collectors.RunCmd) Option {
	return func(c *Collector) { c.runCmd = r }
}

// WithLogger replaces the default slog.Default logger.
func WithLogger(l *slog.Logger) Option {
	return func(c *Collector) { c.log = l }
}

// WithInsights gates the optional per-app enrichers. Tokens recognized:
//   - "forgotten":    runs the last-used enricher (kMDItemLastUsedDate).
//   - "apple-silicon": runs the lipo -archs enricher.
//
// An unknown token is silently ignored — the wizard may pass through
// new tokens that newer macOS versions support.
func WithInsights(tokens []string) Option {
	return func(c *Collector) {
		c.insights = make(map[string]bool, len(tokens))
		for _, t := range tokens {
			c.insights[t] = true
		}
	}
}

// New returns a macOS collector with sane defaults.
func New(opts ...Option) *Collector {
	c := &Collector{
		runCmd:   collectors.DefaultRunCmd,
		log:      slog.Default(),
		insights: map[string]bool{},
	}
	for _, o := range opts {
		o(c)
	}
	return c
}

// Name returns the collector's identifier used in logs and metadata.
func (c *Collector) Name() string { return "macos" }

// Collect runs the discovery pass and every enabled enricher, returning
// the merged record set. An enricher that fails is logged and skipped —
// the scan does not abort because one source was unavailable.
func (c *Collector) Collect(ctx context.Context) ([]audit.AppRecord, error) {
	if runtime.GOOS != "darwin" {
		return nil, fmt.Errorf("macos collector requires darwin, running on %s", runtime.GOOS)
	}

	records, err := c.discover(ctx)
	if err != nil {
		return nil, fmt.Errorf("discover: %w", err)
	}
	c.log.Info("discovered apps", "count", len(records))

	// Order matters: source-setting enrichers run before purely
	// additive ones so that later sources can override the baseline.
	// Insight enrichers are gated on the --insights set.
	pipeline := []struct {
		name      string
		fn        func(context.Context, []audit.AppRecord) error
		insightOn string // empty = always runs
	}{
		{"system_profiler", c.enrichFromSystemProfiler, ""},
		{"mas", c.enrichFromMas, ""},
		{"brew", c.enrichFromBrew, ""},
		{"pkgutil", c.enrichFromPkgutil, ""},
		{"mdls", c.enrichFromMdls, ""},
		{"quarantine", c.enrichFromQuarantine, ""},
		{"codesign", c.enrichFromCodesign, ""},
		{"last_used", c.enrichFromLastUsed, "forgotten"},
		{"lipo", c.enrichFromLipo, "apple-silicon"},
	}
	for _, step := range pipeline {
		if step.insightOn != "" && !c.insights[step.insightOn] {
			continue
		}
		if err := step.fn(ctx, records); err != nil {
			c.log.Warn("enricher failed; continuing", "enricher", step.name, "err", err)
		}
	}

	return records, nil
}
