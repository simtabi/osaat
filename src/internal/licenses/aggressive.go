package licenses

import (
	"context"

	"github.com/simtabi/osaat/internal/audit"
	"github.com/simtabi/osaat/internal/secrets"
)

// NewAggressiveScanner runs the best-effort scanner and adds a
// guidance entry to the manual checklist directing the user at the
// macOS Keychain for any apps that still need verification. A full
// `security dump-keychain` integration is deferred (it requires
// interactive password prompts and produces noisy output) — for now
// we point at the right command and let the user run it manually.
func NewAggressiveScanner() *AggressiveScanner {
	return &AggressiveScanner{
		base: NewBestEffortScanner(),
	}
}

// AggressiveScanner is the "do as much as possible" license mode.
type AggressiveScanner struct {
	base *BestEffortScanner
}

// Mode implements Scanner.
func (s *AggressiveScanner) Mode() string { return "aggressive" }

// Scan implements Scanner.
func (s *AggressiveScanner) Scan(ctx context.Context, records []audit.AppRecord) (*secrets.File, error) {
	out, err := s.base.Scan(ctx, records)
	if err != nil {
		return nil, err
	}
	out.LicenseMode = s.Mode()
	out.ManualChecklist = append(out.ManualChecklist, secrets.ChecklistItem{
		AppName:  "(all apps — Keychain)",
		LookHere: "macOS Keychain — `security dump-keychain login.keychain > ~/keychain.txt` (requires you to confirm each item via GUI dialog)",
		LookFor:  "Generic-password entries whose Service name matches an app's bundle id; entries tagged `kSecAttrLabel` containing 'license' / 'serial' / 'key'.",
	})
	return out, nil
}
