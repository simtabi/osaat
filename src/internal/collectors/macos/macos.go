// Package macos implements the macOS application collector. It is
// designed to run on darwin; on other platforms Collect() returns a
// clear error rather than producing partial data.
package macos

import (
	"context"
	"fmt"
	"log/slog"
	"runtime"
	"sync"

	"github.com/simtabi/osaat/internal/audit"
	"github.com/simtabi/osaat/internal/collectors"
)

// DefaultParallelism is the worker-pool size used by per-app enrichers
// when no override is supplied. Picked empirically: each per-app
// subprocess (`mdls`, `codesign`, `lipo`) is IO-bound and short, so
// running more workers than CPUs is fine — the bottleneck is process
// spawn overhead and the filesystem cache, not CPU.
const DefaultParallelism = 16

// ProgressFn is the optional hook called at every enricher boundary.
// The supplied label is the enricher's short name ("mdls", "codesign",
// "last_used", etc.). Callers use it to drive a live scan view.
type ProgressFn func(phase string)

// Collector is the macOS implementation of collectors.Collector.
// Construct it with New() and customize behavior via the Option helpers.
type Collector struct {
	runCmd      collectors.RunCmd
	log         *slog.Logger
	insights    map[string]bool
	parallelism int
	progress    ProgressFn
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

// WithParallelism overrides DefaultParallelism. Values <= 0 fall back
// to the default. Set to 1 to force sequential execution (useful in
// tests that assert deterministic logging order).
func WithParallelism(n int) Option {
	return func(c *Collector) {
		if n > 0 {
			c.parallelism = n
		}
	}
}

// WithProgressFn registers a callback invoked at each enricher
// boundary. Use it to drive a bubbletea live scan view from outside
// the collector package.
func WithProgressFn(fn ProgressFn) Option {
	return func(c *Collector) { c.progress = fn }
}

// New returns a macOS collector with sane defaults.
func New(opts ...Option) *Collector {
	c := &Collector{
		runCmd:      collectors.DefaultRunCmd,
		log:         slog.Default(),
		insights:    map[string]bool{},
		parallelism: DefaultParallelism,
	}
	for _, o := range opts {
		o(c)
	}
	return c
}

// Name returns the collector's identifier used in logs and metadata.
func (c *Collector) Name() string { return "macos" }

// emitProgress invokes the registered ProgressFn if any, swallowing
// panics from misbehaving callbacks — the scan must continue even if
// the live view is broken.
func (c *Collector) emitProgress(phase string) {
	if c.progress == nil {
		return
	}
	defer func() {
		if r := recover(); r != nil {
			c.log.Warn("progress callback panicked", "phase", phase, "recover", fmt.Sprint(r))
		}
	}()
	c.progress(phase)
}

// Collect runs the discovery pass and every enabled enricher, returning
// the merged record set. An enricher that fails is logged and skipped —
// the scan does not abort because one source was unavailable.
func (c *Collector) Collect(ctx context.Context) ([]audit.AppRecord, error) {
	if runtime.GOOS != "darwin" {
		return nil, fmt.Errorf("macos collector requires darwin, running on %s", runtime.GOOS)
	}

	c.emitProgress("discovering applications")
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
		c.emitProgress(step.name)
		if err := step.fn(ctx, records); err != nil {
			c.log.Warn("enricher failed; continuing", "enricher", step.name, "err", err)
		}
	}

	c.emitProgress("finalizing")
	return records, nil
}

// runPerApp runs work concurrently over every record using the
// configured parallelism. The callback receives the record index and
// is responsible for writing only to records[i] — different goroutines
// never race because they target disjoint indices.
func (c *Collector) runPerApp(ctx context.Context, records []audit.AppRecord, work func(ctx context.Context, i int)) {
	limit := c.parallelism
	if limit <= 0 {
		limit = DefaultParallelism
	}
	if limit > len(records) && len(records) > 0 {
		limit = len(records)
	}
	if limit <= 1 {
		// Sequential path keeps tests deterministic and avoids the
		// goroutine overhead when only one worker is allowed.
		for i := range records {
			if ctx.Err() != nil {
				return
			}
			work(ctx, i)
		}
		return
	}

	sem := make(chan struct{}, limit)
	var wg sync.WaitGroup
	for i := range records {
		if ctx.Err() != nil {
			break
		}
		i := i
		wg.Add(1)
		sem <- struct{}{}
		go func() {
			defer wg.Done()
			defer func() { <-sem }()
			work(ctx, i)
		}()
	}
	wg.Wait()
}
