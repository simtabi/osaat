package licenses

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/simtabi/osaat/internal/audit"
	"github.com/simtabi/osaat/internal/secrets"
)

// NewChecklistScanner emits per-app pointers ("look here") without
// trying to read any files. Useful when the user prefers manual
// verification or doesn't trust automated heuristics.
func NewChecklistScanner() *ChecklistScanner { return &ChecklistScanner{} }

// ChecklistScanner is the lowest-risk license-mode.
type ChecklistScanner struct{}

// Mode implements Scanner.
func (s *ChecklistScanner) Mode() string { return "checklist" }

// Scan implements Scanner.
func (s *ChecklistScanner) Scan(_ context.Context, records []audit.AppRecord) (*secrets.File, error) {
	home, _ := os.UserHomeDir()
	hostname, _ := os.Hostname()

	out := &secrets.File{
		Schema:      secrets.SchemaSecretsV1,
		GeneratedAt: time.Now().UTC(),
		Host:        hostname,
		LicenseMode: s.Mode(),
	}

	for _, r := range records {
		if !needsChecklistEntry(r) {
			continue
		}
		out.ManualChecklist = append(out.ManualChecklist, checklistItemFor(r, home))
	}
	return out, nil
}

// needsChecklistEntry decides whether to bother emitting a pointer
// for this record. System apps and Homebrew formulae don't need
// license-key recovery.
func needsChecklistEntry(r audit.AppRecord) bool {
	switch r.Source {
	case audit.SourceSystem, audit.SourceSandbox, audit.SourceBrewFormula:
		return false
	}
	return true
}

func checklistItemFor(r audit.AppRecord, home string) secrets.ChecklistItem {
	lookHere := []string{}
	if r.BundleID != "" {
		lookHere = append(lookHere,
			fmt.Sprintf("%s/Library/Preferences/%s.plist", home, r.BundleID),
			fmt.Sprintf("%s/Library/Application Support/%s/", home, r.BundleID),
		)
	}
	if r.Name != "" {
		lookHere = append(lookHere,
			fmt.Sprintf("%s/Library/Application Support/%s/", home, r.Name),
		)
	}
	if len(lookHere) == 0 {
		lookHere = []string{"(no candidate paths — read the app's documentation)"}
	}

	lookFor := fmt.Sprintf("Search email receipts for %q; check vendor account dashboards.", r.Name)
	if r.VendorURL != "" {
		lookFor += " Vendor: " + r.VendorURL
	}

	return secrets.ChecklistItem{
		AppName:  r.Name,
		LookHere: strings.Join(lookHere, " | "),
		LookFor:  lookFor,
	}
}
