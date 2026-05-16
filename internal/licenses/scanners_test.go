package licenses

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/simtabi/osaat/internal/audit"
	"github.com/simtabi/osaat/internal/secrets"
)

func TestForReturnsNilForNoneMode(t *testing.T) {
	for _, mode := range []string{"none", ""} {
		s, err := For(mode)
		if err != nil {
			t.Errorf("For(%q) returned error: %v", mode, err)
		}
		if s != nil {
			t.Errorf("For(%q) should return nil scanner; got %T", mode, s)
		}
	}
}

func TestForRejectsUnknownMode(t *testing.T) {
	_, err := For("bogus")
	if err == nil {
		t.Fatal("expected error for unknown mode")
	}
	if !strings.Contains(err.Error(), "unknown license-mode") {
		t.Errorf("error should mention 'unknown license-mode': %v", err)
	}
}

func TestChecklistScannerSkipsSystemApps(t *testing.T) {
	s := NewChecklistScanner()
	records := []audit.AppRecord{
		{Name: "Safari", BundleID: "com.apple.Safari", Source: audit.SourceSystem},
		{Name: "Photoshop", BundleID: "com.adobe.photoshop", Source: audit.SourceDMG},
		{Name: "BrewLib", Source: audit.SourceBrewFormula},
	}
	got, err := s.Scan(context.Background(), records)
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if len(got.ManualChecklist) != 1 {
		t.Fatalf("expected 1 checklist entry (Photoshop only); got %d: %+v", len(got.ManualChecklist), got.ManualChecklist)
	}
	if got.ManualChecklist[0].AppName != "Photoshop" {
		t.Errorf("checklist entry: %+v", got.ManualChecklist[0])
	}
}

func writePlist(t *testing.T, path string, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
}

const licensePlistBody = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>LicenseKey</key>
    <string>ABCD-EFGH-IJKL-MNOP-QRST</string>
    <key>RegisteredEmail</key>
    <string>user@example.com</string>
    <key>SomeUUID</key>
    <string>0123abcd-4567-89ef-0123-456789abcdef</string>
    <key>UnrelatedSetting</key>
    <true/>
</dict>
</plist>
`

func TestBestEffortFindsLicenseInPreferences(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)

	prefsPath := filepath.Join(tmpHome, "Library", "Preferences", "com.example.testapp.plist")
	writePlist(t, prefsPath, licensePlistBody)

	s := NewBestEffortScanner()
	records := []audit.AppRecord{
		{Name: "TestApp", BundleID: "com.example.testapp", Source: audit.SourceDMG},
	}
	got, err := s.Scan(context.Background(), records)
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}

	var licenseEntry, emailEntry, uuidEntry *secrets.Entry
	for ci := range got.Categories {
		for ei := range got.Categories[ci].Entries {
			e := &got.Categories[ci].Entries[ei]
			switch {
			case strings.Contains(e.Source, "LicenseKey"):
				licenseEntry = e
			case strings.Contains(e.Source, "RegisteredEmail"):
				emailEntry = e
			case strings.Contains(e.Source, "SomeUUID"):
				uuidEntry = e
			}
		}
	}

	if licenseEntry == nil {
		t.Fatalf("did not find LicenseKey entry; got categories: %+v", got.Categories)
	}
	if licenseEntry.LicenseKey != "ABCD-EFGH-IJKL-MNOP-QRST" {
		t.Errorf("license key: got %q", licenseEntry.LicenseKey)
	}
	if licenseEntry.Confidence != secrets.ConfidenceHigh {
		t.Errorf("license confidence: got %q, want high (field name + value pattern both match)", licenseEntry.Confidence)
	}

	if emailEntry == nil {
		t.Errorf("did not find email entry")
	} else if emailEntry.LicenseEmail != "user@example.com" {
		t.Errorf("email: got %q", emailEntry.LicenseEmail)
	}

	if uuidEntry == nil {
		t.Logf("uuid-only entry not surfaced (acceptable for SomeUUID — field name is generic)")
	} else if uuidEntry.Confidence != secrets.ConfidenceLow {
		t.Errorf("SomeUUID confidence: got %q, want low (value matches but field name is generic)", uuidEntry.Confidence)
	}
}

func TestBestEffortReturnsEmptyWhenNothingFound(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)

	s := NewBestEffortScanner()
	records := []audit.AppRecord{
		{Name: "NoFilesApp", BundleID: "com.example.nothing", Source: audit.SourceDMG},
	}
	got, err := s.Scan(context.Background(), records)
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if got.LicenseMode != "best-effort" {
		t.Errorf("mode: %q", got.LicenseMode)
	}
	for _, c := range got.Categories {
		if len(c.Entries) > 0 {
			t.Errorf("expected no entries for app with no plists; got %+v", c.Entries)
		}
	}
	if len(got.ManualChecklist) == 0 {
		t.Error("expected at least the manual checklist to be populated")
	}
}

func TestAggressiveIncludesKeychainNote(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)

	s := NewAggressiveScanner()
	got, err := s.Scan(context.Background(), []audit.AppRecord{
		{Name: "AnApp", BundleID: "com.example.a", Source: audit.SourceDMG},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.LicenseMode != "aggressive" {
		t.Errorf("mode: %q", got.LicenseMode)
	}
	var foundKeychainNote bool
	for _, item := range got.ManualChecklist {
		if strings.Contains(item.LookHere, "Keychain") {
			foundKeychainNote = true
			break
		}
	}
	if !foundKeychainNote {
		t.Errorf("aggressive mode should include a Keychain pointer in the checklist")
	}
}
