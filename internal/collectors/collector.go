// Package collectors holds the Collector interface and shared helpers
// used by OS-specific collector implementations under collectors/<os>/.
package collectors

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/simtabi/osaat/internal/audit"
)

// Collector is the contract every OS-specific collector implements.
type Collector interface {
	Name() string
	Collect(ctx context.Context) ([]audit.AppRecord, error)
}

// RunCmd is the dependency-injection point for running subprocesses.
// Tests replace it with a function that returns canned bytes; production
// uses DefaultRunCmd.
type RunCmd func(ctx context.Context, name string, args ...string) ([]byte, error)

// DefaultRunCmd runs the command with a 30 second timeout if the context
// has no deadline of its own, captures stdout, and returns wrapped errors
// that include stderr for diagnostics.
func DefaultRunCmd(ctx context.Context, name string, args ...string) ([]byte, error) {
	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, 30*time.Second)
		defer cancel()
	}

	cmd := exec.CommandContext(ctx, name, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		stderrSnip := strings.TrimSpace(stderr.String())
		if len(stderrSnip) > 256 {
			stderrSnip = stderrSnip[:256] + "..."
		}
		return nil, fmt.Errorf("%s %s: %w (stderr: %s)", name, strings.Join(args, " "), err, stderrSnip)
	}
	return stdout.Bytes(), nil
}

// LookupExe reports whether an executable is on PATH. Used by collectors
// to gate optional enrichers without making them error.
func LookupExe(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

// RunCmdCombined captures stdout and stderr from the command and
// returns whatever was written even when the command exits non-zero.
// Used for tools that write their primary output to stderr (codesign)
// or tools whose non-zero exit codes still carry useful data.
//
// The ctx must carry the cancellation policy — RunCmdCombined does
// not impose its own timeout.
func RunCmdCombined(ctx context.Context, name string, args ...string) []byte {
	cmd := exec.CommandContext(ctx, name, args...)
	out, _ := cmd.CombinedOutput()
	return out
}
