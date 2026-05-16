package licenses

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"howett.net/plist"

	"github.com/simtabi/osaat/internal/audit"
	"github.com/simtabi/osaat/internal/secrets"
)

// NewBestEffortScanner walks Preferences plists and Application
// Support directories for fields that look like license keys.
// Findings are tagged with a confidence level so the reader can
// triage high-signal hits from likely noise.
func NewBestEffortScanner() *BestEffortScanner {
	return &BestEffortScanner{
		// Regexes are compiled once and reused across records.
		keyFieldRe:   regexp.MustCompile(`(?i)(licen[cs]e|serial|registr|activation|product[\s_-]*key|unlock[\s_-]*code)`),
		emailFieldRe: regexp.MustCompile(`(?i)(registered[\s_-]*email|owner[\s_-]*email|licen[cs]e[\s_-]*email)`),
		valuePatterns: []*regexp.Regexp{
			regexp.MustCompile(`^[A-Fa-f0-9]{8}-[A-Fa-f0-9]{4}-[A-Fa-f0-9]{4}-[A-Fa-f0-9]{4}-[A-Fa-f0-9]{12}$`),
			regexp.MustCompile(`^[A-Z0-9]{4,}(-[A-Z0-9]{4,}){2,}$`),
			regexp.MustCompile(`^[A-Fa-f0-9]{32,}$`),
		},
	}
}

// BestEffortScanner extracts license-shaped values from local plists.
type BestEffortScanner struct {
	keyFieldRe    *regexp.Regexp
	emailFieldRe  *regexp.Regexp
	valuePatterns []*regexp.Regexp
}

// Mode implements Scanner.
func (s *BestEffortScanner) Mode() string { return "best-effort" }

// Scan implements Scanner.
func (s *BestEffortScanner) Scan(_ context.Context, records []audit.AppRecord) (*secrets.File, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	hostname, _ := os.Hostname()

	out := &secrets.File{
		Schema:      secrets.SchemaSecretsV1,
		GeneratedAt: time.Now().UTC(),
		Host:        hostname,
		LicenseMode: s.Mode(),
	}

	bySource := map[string][]secrets.Entry{}

	for _, r := range records {
		if !needsChecklistEntry(r) {
			continue
		}
		for _, entry := range s.scanRecord(r, home) {
			bySource[categoryFor(r)] = append(bySource[categoryFor(r)], entry)
		}
	}

	for _, name := range []string{"App Store", "Homebrew Cask", "Standalone (direct download)", "Installer packages", "Other"} {
		if entries, ok := bySource[name]; ok {
			out.Categories = append(out.Categories, secrets.Category{Name: name, Entries: entries})
		}
	}

	// Apps with no automated findings get a manual-checklist entry so
	// the user knows we looked.
	checklist := NewChecklistScanner()
	if cf, _ := checklist.Scan(context.Background(), records); cf != nil {
		out.ManualChecklist = cf.ManualChecklist
	}

	return out, nil
}

// categoryFor maps a record's Source to one of the named buckets in
// the secrets file.
func categoryFor(r audit.AppRecord) string {
	switch r.Source {
	case audit.SourceAppStore:
		return "App Store"
	case audit.SourceBrewCask:
		return "Homebrew Cask"
	case audit.SourceDMG:
		return "Standalone (direct download)"
	case audit.SourcePkg:
		return "Installer packages"
	default:
		return "Other"
	}
}

// scanRecord inspects the well-known per-app locations and returns
// any license-shaped findings.
func (s *BestEffortScanner) scanRecord(r audit.AppRecord, home string) []secrets.Entry {
	var found []secrets.Entry

	// 1. ~/Library/Preferences/<BundleID>.plist
	if r.BundleID != "" {
		prefPath := filepath.Join(home, "Library", "Preferences", r.BundleID+".plist")
		found = append(found, s.scanPlistFile(r, prefPath)...)
	}

	// 2. ~/Library/Application Support/<BundleID>/* and /<Name>/*
	candidates := []string{}
	if r.BundleID != "" {
		candidates = append(candidates, filepath.Join(home, "Library", "Application Support", r.BundleID))
	}
	if r.Name != "" {
		candidates = append(candidates, filepath.Join(home, "Library", "Application Support", r.Name))
	}
	for _, dir := range candidates {
		found = append(found, s.scanAppSupportDir(r, dir)...)
	}

	return found
}

// scanAppSupportDir walks the top level of dir and scans files whose
// name suggests they hold a license. Does NOT recurse — limits the
// scan to predictable cost.
func (s *BestEffortScanner) scanAppSupportDir(r audit.AppRecord, dir string) []secrets.Entry {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var found []secrets.Entry
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := strings.ToLower(e.Name())
		if !s.keyFieldRe.MatchString(name) {
			continue
		}
		path := filepath.Join(dir, e.Name())
		if strings.HasSuffix(name, ".plist") {
			found = append(found, s.scanPlistFile(r, path)...)
		}
	}
	return found
}

// scanPlistFile parses path as a plist (binary or XML) and looks for
// keys with names that match keyFieldRe. Returns one Entry per match.
func (s *BestEffortScanner) scanPlistFile(r audit.AppRecord, path string) []secrets.Entry {
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()

	var raw map[string]any
	if err := plist.NewDecoder(f).Decode(&raw); err != nil {
		return nil
	}

	var entries []secrets.Entry
	for k, v := range raw {
		strVal, ok := v.(string)
		if !ok || strVal == "" {
			continue
		}

		nameMatch := s.keyFieldRe.MatchString(k)
		emailMatch := s.emailFieldRe.MatchString(k)
		valueMatch := s.valuePatternsMatch(strVal)

		switch {
		case nameMatch && valueMatch:
			entries = append(entries, makeEntry(r, path, k, strVal, secrets.ConfidenceHigh))
		case nameMatch:
			entries = append(entries, makeEntry(r, path, k, strVal, secrets.ConfidenceMedium))
		case emailMatch:
			e := makeEntry(r, path, k, "", secrets.ConfidenceMedium)
			e.LicenseEmail = strVal
			e.LicenseKey = ""
			entries = append(entries, e)
		case valueMatch:
			// Value looks like a key but the field name is generic —
			// could easily be an internal UUID. Low confidence.
			entries = append(entries, makeEntry(r, path, k, strVal, secrets.ConfidenceLow))
		}
	}
	return entries
}

func (s *BestEffortScanner) valuePatternsMatch(v string) bool {
	for _, pat := range s.valuePatterns {
		if pat.MatchString(v) {
			return true
		}
	}
	return false
}

func makeEntry(r audit.AppRecord, path, key, value string, conf secrets.Confidence) secrets.Entry {
	return secrets.Entry{
		AppName:    r.Name,
		BundleID:   r.BundleID,
		AppStoreID: r.AppStoreID,
		LicenseKey: value,
		Source:     fmt.Sprintf("%s#%s", path, key),
		Confidence: conf,
	}
}

// scanPlistFile is also useful for tests — exported alias.
var _ = (fs.FS)(nil) // keep io/fs in scope for future expansion
