# Data model

**Status:** ratified, 2026-05-16

Two top-level artifacts:

- `report.json` — array of `AppRecord`. No license keys.
- `secrets.json` — `SecretsFile`. Full keys, well-organized. Optional age encryption.

## `AppRecord`

```go
// internal/audit/record.go
package audit

import "time"

type AppRecord struct {
    // Identity
    Name            string `json:"name"`
    BundleID        string `json:"bundle_id,omitempty"`        // CFBundleIdentifier (macOS)
    PkgID           string `json:"pkg_id,omitempty"`           // dpkg / rpm / pacman name (Linux/Unix)
    Version         string `json:"version,omitempty"`

    // Provenance
    Author          string `json:"author,omitempty"`           // codesign authority / package author
    VendorURL       string `json:"vendor_url,omitempty"`
    Source          Source `json:"source"`                     // enum below
    DownloadURL     string `json:"download_url,omitempty"`     // kMDItemWhereFroms or quarantine xattr
    InstallerHash   string `json:"installer_sha256,omitempty"` // when discoverable

    // Filesystem
    Path            string    `json:"path,omitempty"`
    SizeBytes       int64     `json:"size_bytes,omitempty"`
    InstalledAt     time.Time `json:"installed_at,omitempty"`  // file mtime / pkg receipt date
    LastUsedAt      time.Time `json:"last_used_at,omitempty"`  // kMDItemLastUsedDate / atime

    // Security
    SigningStatus   SigningStatus `json:"signing_status,omitempty"`
    SigningTeam     string        `json:"signing_team,omitempty"`

    // macOS-specific
    AppleSilicon    *bool    `json:"apple_silicon,omitempty"`  // nil on non-macOS; lipo -archs

    // Restore
    ReinstallCmd    string   `json:"reinstall_cmd,omitempty"`  // e.g. "brew install --cask 1password"
    AppStoreID      string   `json:"app_store_id,omitempty"`   // mas reinstall id

    // Audit
    CollectorNotes  []string `json:"collector_notes,omitempty"` // soft warnings: "unsigned", "outside /Applications", etc.
}

type Source string

const (
    SourceAppStore    Source = "app_store"
    SourceBrewFormula Source = "homebrew_formula"
    SourceBrewCask    Source = "homebrew_cask"
    SourcePkg         Source = "pkg_installer"
    SourceDMG         Source = "dmg_or_direct"
    SourceSystem      Source = "system"          // /System/Applications
    SourceSandbox     Source = "sandbox"         // ~/Library/Containers
    SourceDpkg        Source = "dpkg"
    SourceRpm         Source = "rpm"
    SourcePacman      Source = "pacman"
    SourceSnap        Source = "snap"
    SourceFlatpak     Source = "flatpak"
    SourceAppImage    Source = "appimage"
    SourceBSDPkg      Source = "bsd_pkg"
    SourceUnknown     Source = "unknown"
)

type SigningStatus string

const (
    SigningSigned    SigningStatus = "signed"
    SigningAdHoc     SigningStatus = "ad_hoc"
    SigningUnsigned  SigningStatus = "unsigned"
    SigningUnknown   SigningStatus = "unknown"
)
```

## `SecretsFile`

Grouped by where the key was found, not by app. That way you can scan the
file and see "all my App Store receipts in one block, all my Homebrew cask
keys in another, all my Keychain finds in a third."

```go
// internal/secrets/schema.go
package secrets

import "time"

type SecretsFile struct {
    Schema      string         `json:"schema"`               // "osaat.secrets/v1"
    GeneratedAt time.Time      `json:"generated_at"`
    Host        string         `json:"host"`
    LicenseMode string         `json:"license_mode"`         // best_effort | checklist | aggressive
    Categories  []Category     `json:"categories"`
    ManualChecklist []ChecklistItem `json:"manual_checklist,omitempty"`
}

type Category struct {
    Name    string   `json:"name"`                          // "App Store", "Homebrew Cask", "Standalone", "Keychain"
    Entries []Entry  `json:"entries"`
}

type Entry struct {
    AppName       string            `json:"app"`
    BundleID      string            `json:"bundle_id,omitempty"`
    AppStoreID    string            `json:"app_store_id,omitempty"`
    LicenseKey    string            `json:"license_key,omitempty"`     // full value, no redaction
    LicenseEmail  string            `json:"license_email,omitempty"`
    Source        string            `json:"source"`                    // file path or "keychain:<service>"
    Confidence    Confidence        `json:"confidence"`                // high | medium | low
    Extra         map[string]string `json:"extra,omitempty"`           // anything else worth keeping
}

type ChecklistItem struct {
    AppName    string `json:"app"`
    LookHere   string `json:"look_here"`     // e.g. "~/Library/Application Support/Foo/license.plist"
    LookFor    string `json:"look_for"`      // e.g. "search Gmail for 'Foo receipt'"
}

type Confidence string

const (
    ConfidenceHigh   Confidence = "high"     // exact-match key field in plist
    ConfidenceMedium Confidence = "medium"   // pattern-matched (UUID, hex, dash-separated)
    ConfidenceLow    Confidence = "low"      // included but probably noise
)
```

## Why this shape

- **Stable schema field** (`osaat.secrets/v1`, `osaat.report/v1`) so future
  versions can migrate cleanly.
- **`omitempty` everywhere** so a Linux record doesn't drag macOS-only
  fields, keeping JSON compact and readable.
- **`*bool` for `AppleSilicon`** so we can tell "unknown / not applicable"
  from "Intel only" — `false` is a real value here.
- **`CollectorNotes []string`** is a soft-warnings field; the orchestrator
  appends as it merges sources. Surfaces in reporters as "Notes" column.
- **Secrets grouped by category, not app** so the file is scannable as a
  recovery checklist, which is how it'll actually be used.
- **`Confidence` on every Entry** is the contract between best-effort
  extraction and the reader. "medium" means "could be a license, could be a
  UUID — go verify". No quiet false positives.
