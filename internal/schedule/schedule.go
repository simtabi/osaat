// Package schedule renders and installs platform scheduler units that
// run `osaat scan` on a recurring cadence — launchd on macOS, systemd
// user units on Linux.
//
// The package is purely declarative for unit generation (text/template)
// and shells out to launchctl / systemctl only to load and unload the
// generated units. Generation is testable without root or systemctl.
package schedule

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"strings"
	"time"
)

// Cadence is how often the scheduled scan should run.
type Cadence string

const (
	CadenceDaily   Cadence = "daily"
	CadenceWeekly  Cadence = "weekly"
	CadenceMonthly Cadence = "monthly"
)

// Validate reports whether c is one of the supported cadences.
func (c Cadence) Validate() error {
	switch c {
	case CadenceDaily, CadenceWeekly, CadenceMonthly:
		return nil
	default:
		return fmt.Errorf("unsupported cadence %q (use daily|weekly|monthly)", c)
	}
}

// InstallOptions configures one scheduled audit.
type InstallOptions struct {
	// BinaryPath is the absolute path to the osaat binary that the
	// scheduler should invoke. Caller is responsible for resolving via
	// os.Executable() and verifying it's not the per-run go-build
	// temp binary.
	BinaryPath string

	// Cadence picks the timing pattern (06:00 local time always —
	// borrowed from the global CLAUDE.md Dependabot convention).
	Cadence Cadence

	// OutDir is the --out value the scheduled invocation will use.
	// Supports the literal `{date}` template which the runner expands
	// to YYYY-MM-DD at execution time (the platform scheduler can't
	// expand it for us). Defaults to a documents-path placeholder.
	OutDir string

	// Profile is an optional --profile <name> argument.
	Profile string

	// ExtraArgs are appended after the standard scan arguments. Use
	// for things like --insights or --license-mode that aren't part
	// of a profile.
	ExtraArgs []string

	// Label is the launchd label / systemd unit basename. Defaults
	// to "osaat" (or "com.simtabi.osaat.<profile>" on launchd when a
	// profile is named).
	Label string

	// DryRun reports the plan but doesn't write to disk or load any
	// scheduler unit.
	DryRun bool
}

// UninstallOptions controls schedule removal.
type UninstallOptions struct {
	Label  string
	DryRun bool
}

// Action describes a single planned filesystem or scheduler change.
type Action struct {
	// Kind is "write", "remove", "load", or "unload".
	Kind string
	// Path is the file path (write/remove) or a textual description
	// of the load/unload command (load/unload).
	Path string
	// Mode is the on-disk file mode for write actions; zero otherwise.
	Mode os.FileMode
	// Body is the file content for write actions; empty for the
	// other kinds. Tests assert against this.
	Body string
}

// Plan is the full set of actions a Manager will perform.
type Plan struct {
	Actions []Action
	DryRun  bool
}

// Manager installs or uninstalls one scheduled scan. Each platform
// has its own implementation under this package.
type Manager interface {
	// Plan returns the actions Install would perform without
	// touching disk or the scheduler. Used for --dry-run output.
	Plan(opts InstallOptions) (Plan, error)
	// Install writes the unit file(s) and loads them with the
	// platform scheduler.
	Install(ctx context.Context, opts InstallOptions) (Plan, error)
	// Uninstall removes and unloads the unit(s).
	Uninstall(ctx context.Context, opts UninstallOptions) (Plan, error)
}

// For returns the Manager for the current platform.
func For() (Manager, error) {
	switch runtime.GOOS {
	case "darwin":
		return NewLaunchd(), nil
	case "linux":
		return NewSystemd(), nil
	default:
		return nil, fmt.Errorf("install-schedule is not supported on %s yet", runtime.GOOS)
	}
}

// scheduleTime returns the (Weekday, Day-of-Month) selectors for a
// cadence. Hour is always 6, minute is always 0 — matches the
// Dependabot Mon 06:00 convention from the global CLAUDE.md.
func scheduleTime(c Cadence) (weekday, dayOfMonth int) {
	switch c {
	case CadenceWeekly:
		return 1, 0 // Monday
	case CadenceMonthly:
		return 0, 1 // 1st of month
	default:
		return 0, 0
	}
}

// defaultOutDir returns the placeholder used when InstallOptions
// leaves OutDir blank. {date} is intentionally embedded so the
// scheduled run produces a per-day directory.
func defaultOutDir(home string) string {
	return strings.TrimRight(home, "/") + "/Documents/osaat/{date}"
}

// expandTime is a no-op for now — the scheduler doesn't expand
// {date}; the scan binary does at invocation time. The function
// exists so future enhancements (e.g. expanding template fields in
// the binary path) have a single hook.
func expandTime(s string, _ time.Time) string {
	return s
}
