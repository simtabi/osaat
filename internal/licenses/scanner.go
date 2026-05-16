// Package licenses implements license-key extraction in three modes:
// best_effort (heuristic plist scan), checklist (pointers only), and
// aggressive (best_effort plus a manual-keychain note). All three
// produce a *secrets.File which is written separately from report.json.
package licenses

import (
	"context"
	"fmt"

	"github.com/simtabi/osaat/internal/audit"
	"github.com/simtabi/osaat/internal/secrets"
)

// Scanner is the contract every license-extraction mode implements.
type Scanner interface {
	// Mode is the short token used on the --license-mode CLI flag.
	Mode() string

	// Scan inspects the records and returns a populated secrets.File.
	// Implementations must never write license values into the input
	// records — the keys belong in the secrets file only.
	Scan(ctx context.Context, records []audit.AppRecord) (*secrets.File, error)
}

// For returns the Scanner for a mode name, or an error for unknown modes.
func For(mode string) (Scanner, error) {
	switch mode {
	case "none", "":
		return nil, nil
	case "checklist":
		return NewChecklistScanner(), nil
	case "best-effort", "best_effort":
		return NewBestEffortScanner(), nil
	case "aggressive":
		return NewAggressiveScanner(), nil
	default:
		return nil, fmt.Errorf("unknown license-mode %q (use none|checklist|best-effort|aggressive)", mode)
	}
}
