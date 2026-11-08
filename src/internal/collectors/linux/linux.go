// Package linux implements the Linux application collector. It runs
// the package managers it can find (dpkg, rpm, pacman) and merges
// their output into a single AppRecord list. Phase 4b adds snap,
// flatpak, AppImage, and desktop-file enrichment.
package linux

import (
	"context"
	"fmt"
	"log/slog"
	"runtime"
	"strings"

	"github.com/simtabi/osaat/internal/audit"
	"github.com/simtabi/osaat/internal/collectors"
)

// Collector is the Linux implementation of collectors.Collector.
type Collector struct {
	runCmd      collectors.RunCmd
	log         *slog.Logger
	insights    map[string]bool
	progress    ProgressFn
}

// ProgressFn is the same shape as the macOS collector's: invoked at
// every sub-collector boundary with that collector's short name
// ("dpkg", "rpm", "pacman", "snap", ...).
type ProgressFn func(phase string)

// Option mutates a Collector during construction.
type Option func(*Collector)

// WithRunCmd injects a subprocess runner for testing.
func WithRunCmd(r collectors.RunCmd) Option {
	return func(c *Collector) { c.runCmd = r }
}

// WithLogger replaces the default slog.Default logger.
func WithLogger(l *slog.Logger) Option {
	return func(c *Collector) { c.log = l }
}

// WithInsights records insight tokens for parity with macos.Collector;
// the Linux collector consults this for future enrichers (forgotten
// apps from atime, etc.) — currently a no-op stub.
func WithInsights(tokens []string) Option {
	return func(c *Collector) {
		c.insights = make(map[string]bool, len(tokens))
		for _, t := range tokens {
			c.insights[t] = true
		}
	}
}

// WithProgressFn registers the phase-event callback.
func WithProgressFn(fn ProgressFn) Option {
	return func(c *Collector) { c.progress = fn }
}

// New returns a Linux collector with sane defaults.
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

// Name returns "linux".
func (c *Collector) Name() string { return "linux" }

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

// Collect runs every available package-manager sub-collector and
// returns the merged record set. Missing tools are skipped silently
// — a Debian box doesn't have `pacman`, an Arch box doesn't have
// `dpkg-query`. The orchestrator never errors when a tool is absent.
func (c *Collector) Collect(ctx context.Context) ([]audit.AppRecord, error) {
	if runtime.GOOS != "linux" {
		return nil, fmt.Errorf("linux collector requires linux, running on %s", runtime.GOOS)
	}

	subs := []struct {
		name     string
		exe      string
		fn       func(context.Context) ([]audit.AppRecord, error)
	}{
		{"dpkg", "dpkg-query", c.collectDpkg},
		{"rpm", "rpm", c.collectRpm},
		{"pacman", "pacman", c.collectPacman},
		{"snap", "snap", c.collectSnap},
		{"flatpak", "flatpak", c.collectFlatpak},
		{"appimage", "", c.collectAppImage}, // no required tool; always runs on Linux
	}

	var all []audit.AppRecord
	for _, s := range subs {
		// s.exe == "" means the sub-collector doesn't shell out to an
		// optional tool — it always runs (e.g. AppImage filesystem walk).
		if s.exe != "" && !collectors.LookupExe(s.exe) {
			continue
		}
		c.emitProgress(s.name)
		recs, err := s.fn(ctx)
		if err != nil {
			c.log.Warn("sub-collector failed; continuing", "sub", s.name, "err", err)
			continue
		}
		c.log.Info("sub-collector returned", "sub", s.name, "count", len(recs))
		all = append(all, recs...)
	}

	c.emitProgress("finalizing")
	return all, nil
}

// extractEmailFromMaintainer splits a Debian/RPM-style maintainer
// string "Foo Bar <foo@bar.com>" into the bare name. The email half
// is dropped — we already capture vendor URLs separately when
// available.
func extractEmailFromMaintainer(s string) (name, email string) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "", ""
	}
	open := strings.LastIndex(s, "<")
	close := strings.LastIndex(s, ">")
	if open >= 0 && close > open {
		email = strings.TrimSpace(s[open+1 : close])
		name = strings.TrimSpace(s[:open])
	} else {
		name = s
	}
	return name, email
}

// _ keeps audit imported in case future helpers need it after the
// Phase 4a sub-collectors are added piecemeal.
var _ = audit.SourceUnknown
