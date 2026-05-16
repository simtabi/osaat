// Package audit defines the canonical AppRecord shape and the
// orchestrator that runs collectors and reporters.
package audit

import "time"

// SchemaReportV1 is the value of the "schema" field at the top of a
// report.json produced by this version of osaat. Readers should check
// this before deserializing.
const SchemaReportV1 = "osaat.report/v1"

// AppRecord is the canonical per-application row. Every collector
// returns a slice of these and every reporter consumes them.
//
// All optional fields carry omitempty so a Linux record does not drag
// macOS-only fields and vice versa.
type AppRecord struct {
	// Identity
	Name     string `json:"name"`
	BundleID string `json:"bundle_id,omitempty"` // CFBundleIdentifier (macOS)
	PkgID    string `json:"pkg_id,omitempty"`    // dpkg / rpm / pacman name (Linux/Unix)
	Version  string `json:"version,omitempty"`

	// Provenance
	Author        string `json:"author,omitempty"`
	VendorURL     string `json:"vendor_url,omitempty"`
	Source        Source `json:"source"`
	DownloadURL   string `json:"download_url,omitempty"`
	InstallerHash string `json:"installer_sha256,omitempty"`

	// Filesystem
	Path        string     `json:"path,omitempty"`
	SizeBytes   int64      `json:"size_bytes,omitempty"`
	InstalledAt *time.Time `json:"installed_at,omitempty"`
	LastUsedAt  *time.Time `json:"last_used_at,omitempty"`

	// Security
	SigningStatus SigningStatus `json:"signing_status,omitempty"`
	SigningTeam   string        `json:"signing_team,omitempty"`

	// macOS-specific
	AppleSilicon *bool `json:"apple_silicon,omitempty"`

	// Restore
	ReinstallCmd string `json:"reinstall_cmd,omitempty"`
	AppStoreID   string `json:"app_store_id,omitempty"`

	// Audit-time warnings
	CollectorNotes []string `json:"collector_notes,omitempty"`
}

// Note appends a soft warning to CollectorNotes if the message is not
// already present.
func (r *AppRecord) Note(msg string) {
	for _, existing := range r.CollectorNotes {
		if existing == msg {
			return
		}
	}
	r.CollectorNotes = append(r.CollectorNotes, msg)
}

// Source is the installation channel a record came from.
type Source string

const (
	SourceAppStore    Source = "app_store"
	SourceBrewFormula Source = "homebrew_formula"
	SourceBrewCask    Source = "homebrew_cask"
	SourcePkg         Source = "pkg_installer"
	SourceDMG         Source = "dmg_or_direct"
	SourceSystem      Source = "system"
	SourceSandbox     Source = "sandbox"
	SourceDpkg        Source = "dpkg"
	SourceRpm         Source = "rpm"
	SourcePacman      Source = "pacman"
	SourceSnap        Source = "snap"
	SourceFlatpak     Source = "flatpak"
	SourceAppImage    Source = "appimage"
	SourceBSDPkg      Source = "bsd_pkg"
	SourceUnknown     Source = "unknown"
)

// SigningStatus is a coarse summary of whether the binary is signed
// and how strong that signature is.
type SigningStatus string

const (
	SigningSigned   SigningStatus = "signed"
	SigningAdHoc    SigningStatus = "ad_hoc"
	SigningUnsigned SigningStatus = "unsigned"
	SigningUnknown  SigningStatus = "unknown"
)
