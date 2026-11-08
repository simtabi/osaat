// Package unix implements the Unix/BSD application collector — a
// thin wrapper around per-OS pkg tools. Each supported BSD has its
// own sub-collector under this package; runtime.GOOS picks the
// right one.
//
// Currently supported:
//   - freebsd: `pkg query` (modern FreeBSD pkgng)
//   - openbsd, netbsd, dragonfly: `pkg_info`
//
// Linux package managers are in internal/collectors/linux. macOS is
// in internal/collectors/macos. This package is for the BSDs only.
package unix

import (
	"context"
	"fmt"
	"log/slog"
	"runtime"

	"github.com/simtabi/osaat/internal/audit"
	"github.com/simtabi/osaat/internal/collectors"
)

// Collector is the Unix/BSD implementation of collectors.Collector.
type Collector struct {
	runCmd   collectors.RunCmd
	log      *slog.Logger
	progress ProgressFn
}

// ProgressFn matches the other collectors' shape.
type ProgressFn func(phase string)

// Option mutates a Collector during construction.
type Option func(*Collector)

// WithRunCmd injects a subprocess runner for tests.
func WithRunCmd(r collectors.RunCmd) Option {
	return func(c *Collector) { c.runCmd = r }
}

// WithLogger replaces the default slog.Default logger.
func WithLogger(l *slog.Logger) Option {
	return func(c *Collector) { c.log = l }
}

// WithProgressFn registers the phase-event callback.
func WithProgressFn(fn ProgressFn) Option {
	return func(c *Collector) { c.progress = fn }
}

// New returns a Unix/BSD collector with sane defaults.
func New(opts ...Option) *Collector {
	c := &Collector{
		runCmd: collectors.DefaultRunCmd,
		log:    slog.Default(),
	}
	for _, o := range opts {
		o(c)
	}
	return c
}

// Name returns "unix".
func (c *Collector) Name() string { return "unix" }

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

// Collect dispatches to the appropriate sub-collector based on
// runtime.GOOS. Returns a clear error on unsupported OSes so the
// failure surfaces in the user's terminal rather than producing
// silent empty output.
func (c *Collector) Collect(ctx context.Context) ([]audit.AppRecord, error) {
	switch runtime.GOOS {
	case "freebsd":
		c.emitProgress("freebsd-pkg")
		return c.collectFreeBSDPkg(ctx)
	case "openbsd", "netbsd", "dragonfly":
		c.emitProgress("pkg_info")
		return c.collectPkgInfo(ctx)
	default:
		return nil, fmt.Errorf("unix collector does not support %s yet (supported: freebsd, openbsd, netbsd, dragonfly)", runtime.GOOS)
	}
}
