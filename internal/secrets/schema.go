// Package secrets holds the well-organized container for license
// keys, App Store IDs, and similar reinstall-time secrets. The file
// is separate from report.json by design: report.json is the
// shareable inventory; this is the private companion.
package secrets

import "time"

// SchemaSecretsV1 is the value of the "schema" field at the top of a
// secrets file emitted by this version of osaat.
const SchemaSecretsV1 = "osaat.secrets/v1"

// File is the top-level container written to secrets.json (or
// secrets.json.age when encrypted).
type File struct {
	Schema          string          `json:"schema"`
	GeneratedAt     time.Time       `json:"generated_at"`
	Host            string          `json:"host"`
	LicenseMode     string          `json:"license_mode"`
	Categories      []Category      `json:"categories"`
	ManualChecklist []ChecklistItem `json:"manual_checklist,omitempty"`
}

// Category groups entries by where they came from — "App Store",
// "Homebrew", "Standalone", "Keychain", "Plist scan".
type Category struct {
	Name    string  `json:"name"`
	Entries []Entry `json:"entries"`
}

// Entry is one (app, license key) row. LicenseKey is recorded in full
// — the secrets file does not redact.
type Entry struct {
	AppName      string            `json:"app"`
	BundleID     string            `json:"bundle_id,omitempty"`
	AppStoreID   string            `json:"app_store_id,omitempty"`
	LicenseKey   string            `json:"license_key,omitempty"`
	LicenseEmail string            `json:"license_email,omitempty"`
	Source       string            `json:"source"` // file path or "keychain:<service>"
	Confidence   Confidence        `json:"confidence"`
	Extra        map[string]string `json:"extra,omitempty"`
}

// ChecklistItem points at a likely location the user should check by
// hand — used by the `checklist` scanner mode and by `best_effort`
// for apps where it found nothing automatically.
type ChecklistItem struct {
	AppName  string `json:"app"`
	LookHere string `json:"look_here"`
	LookFor  string `json:"look_for"`
}

// Confidence flags how likely an Entry's LicenseKey is to actually be
// a license rather than an internal identifier.
type Confidence string

const (
	// ConfidenceHigh: matched a field whose name explicitly says license/serial/registration.
	ConfidenceHigh Confidence = "high"
	// ConfidenceMedium: value pattern matched (UUID, dashed groups, long hex) but field name was generic.
	ConfidenceMedium Confidence = "medium"
	// ConfidenceLow: included for completeness but probably noise.
	ConfidenceLow Confidence = "low"
)

// IsEmpty reports whether a File contains no useful data — used to
// decide whether to bother writing the file at all.
func (f *File) IsEmpty() bool {
	if f == nil {
		return true
	}
	if len(f.ManualChecklist) > 0 {
		return false
	}
	for _, c := range f.Categories {
		if len(c.Entries) > 0 {
			return false
		}
	}
	return true
}
