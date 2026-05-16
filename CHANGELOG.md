# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- Initial scaffold: Go module, Cobra subcommand skeleton (`scan`, `diff`,
  `restore-help`, `install-schedule`, `backup`, `version`), GitHub Actions
  CI and release workflows, GoReleaser config, full Simtabi OSS document
  tree, and Dependabot configuration (weekly, Monday 06:00
  America/New_York).
- macOS application collector with seven enrichers:
  - filesystem walk of `/Applications`, `~/Applications`,
    `/System/Applications`, and the legacy `Utilities` subdirs
  - `Info.plist` decode via `howett.net/plist`, tolerant of fields
    that ship a dict where a string was expected
  - `system_profiler -xml SPApplicationsDataType` parse for source
    (App Store / identified developer / Apple) and signing authority
  - `mdls -name kMDItemWhereFroms` for download URL when present
  - `xattr -p com.apple.quarantine` for Gatekeeper agent (Safari /
    Chrome / etc.) recorded as a collector note
  - `mas list` cross-reference for App Store IDs and reinstall command
    (skipped if `mas` is not installed)
  - `brew list --cask --versions` cross-reference for cask names and
    reinstall command (skipped if `brew` is not installed)
  - `pkgutil --pkgs` exact-match cross-reference by `CFBundleIdentifier`
  - `codesign -dv --verbose=4` for signing status, team identifier,
    and primary authority (uses combined stdout+stderr capture since
    codesign writes to stderr)
- Canonical `AppRecord` data model under `internal/audit/`, with stable
  `osaat.report/v1` schema header and `omitempty` on every
  platform-specific field.
- JSON reporter (`internal/reporters/json.go`) with deterministic sort
  by case-insensitive name then bundle id, and a host metadata block
  identifying the tool version + commit + hostname.
- `osaat scan` wired to the orchestrator: detects darwin → macos when
  `--os auto`, runs the collector with structured logging, writes
  `report.<format>` files to `--out`. Currently supports `--format
  json`; CSV / Markdown / HTML and the wizard land in Phase 2.

- CSV, Markdown, and HTML reporters. CSV uses a stable column order
  matching the prompt spec; Markdown emits a summary by source +
  signing status followed by the main table; HTML is a single
  self-contained file with inline styling and vanilla-JS column
  sorting + a filter input.
- Restore manifest generation under `internal/restore/`. With
  `--with-restore`, `osaat scan` writes:
  - `Brewfile` — `cask "name"` for every Homebrew cask record and
    `mas "Name", id: N` for every App Store record with a known ID.
  - `mas-apps.txt` — one App Store ID per line, with name comments.
  - `RESTORE.md` — per-app checklist for everything outside Homebrew
    and Mac App Store (direct downloads, .pkg installers, App Store
    apps without a known ID, unknown source).
- `osaat diff <old.json> <new.json>` compares two reports. Records
  are matched on `BundleID` (macOS), `PkgID` (Linux/Unix), or
  `Name+Version` when neither is set. Outputs in `text` (default),
  `json` (schema `osaat.diff/v1`), or `markdown`. Exits 0 when clean,
  1 when differences are found.
- License extraction with three scanner modes:
  - `--license-mode checklist`: emits `look_here` / `look_for`
    pointers for every non-system app; no file reads.
  - `--license-mode best-effort`: reads
    `~/Library/Preferences/<BundleID>.plist` and
    `~/Library/Application Support/<BundleID>/<licen[cs]e>.plist`,
    matches field names and value patterns against license-shaped
    regexes, and emits findings tagged `high` / `medium` / `low`
    confidence.
  - `--license-mode aggressive`: best-effort plus a manual Keychain
    pointer in the checklist for the user to follow up on.
- `internal/secrets/`: canonical secrets file (schema
  `osaat.secrets/v1`) with categories grouped by source. License
  keys are stored in full, never redacted — protection comes from
  encryption at rest, not masking.
- Optional `--age-recipient <age1...>` encrypts the secrets file via
  `filippo.io/age`, producing `secrets.json.age` instead of
  `secrets.json`. Plain `secrets.json` is written with mode 600.
- `report.json` schema unchanged — license keys do not appear there.
- Go toolchain bumped to 1.24 (required by `filippo.io/age`); CI
  matrix updated to `1.24` and `1.25`.

- Interactive wizard via `charmbracelet/huh` — auto-triggers when
  stdin is a TTY and no scan-shaping flags are passed. Five groups:
  scope, output path + formats, license + encryption, insights, save
  profile. Always prompts for the output directory. Prints the
  equivalent non-interactive command on completion.
- `internal/wizard/scanview.go` — `charmbracelet/bubbletea` model for
  a live scan view (phase label, counters, elapsed time, ✓/✗ final
  state). Wired into the wizard form via `huh` (which itself runs on
  bubbletea); a future Phase 3 pass will route per-app progress
  events into this model.
- Named profiles in `~/.config/osaat/profiles/<name>.toml`
  (`osaat.profile/v1` schema). `--profile <name>` loads defaults
  whose flag wasn't explicitly set; profiles persist whatever the
  wizard's last "Save profile" page captures.
- Two new reporters:
  - **PDF** (`report.pdf`) — paginated A4 layout with title block,
    per-source summary, table with zebra rows and footer page
    numbers, built on the pure-Go `go-pdf/fpdf` library.
  - **Plain text** (`report.txt`) — labeled key-value blocks per
    record; grep-friendly, no rendering dependencies.
- New default format set: `pdf`, `markdown`, `txt`, `json`. CSV and
  HTML remain available.
- OS-aware default paths via `internal/paths`:
  - Outputs: `<Documents>/osaat/<YYYY-MM-DD>/` on every OS.
  - Config: `$HOME/.config/osaat/` on every OS (uniform layout
    regardless of platform conventions).
- Privacy-aware file logger (`internal/logging`):
  - Daily log at `~/.config/osaat/logs/osaat-<YYYY-MM-DD>.log` (mode 600).
  - `$HOME` paths are replaced with `~` in every record.
  - Hostname-shaped attribute keys are replaced with `[redacted]`.
  - No network telemetry. Logs are local-only.
  - In wizard mode the logger writes only to the file so the form
    rendering isn't corrupted by stderr output.
- `SHA256SUMS` written to the output directory listing the digest of
  every file produced — supports verifiable restore on the new
  machine.
- `--quiet` flag for headless runs that suppresses the per-file
  `wrote ...` lines.
- Cross-platform release matrix (`.goreleaser.yaml`):
  - macOS: amd64, arm64.
  - Linux: amd64, arm64, 386, armv7.
  - Windows: amd64, arm64, 386.
  - FreeBSD: amd64, arm64, 386.
  - Windows archives ship as `.zip`; the rest are `.tar.gz`.
- New deps: `charmbracelet/huh`, `charmbracelet/bubbletea`,
  `charmbracelet/lipgloss`, `pelletier/go-toml/v2`, `go-pdf/fpdf`,
  `mattn/go-isatty`.

- `scripts/bash-fallback.sh` — zero-dependency Bash 3.2 script that
  audits macOS applications and emits an `osaat.report/v1`-compatible
  JSON file. Uses `mdls`, `codesign`, `du`, `stat`, and `xattr` — no
  Go, no Homebrew, no Python. For the cold-start case where the Go
  binary isn't yet installed. End-to-end on the dev Mac: 296 apps,
  ~120 seconds, output round-trips through `osaat diff` against the
  Go binary's report.
- `docs/tools/bash-fallback.md` documents its capabilities and
  intentional limits (no licenses, no encryption, macOS only).
- Two macOS insight enrichers, gated on the `--insights` flag:
  - **Forgotten apps** (`--insights forgotten`): reads
    `kMDItemLastUsedDate` via `mdls` and flags apps whose last-used
    date is older than `--insights-forgotten-months` (default 6).
    Apps that have never been opened are flagged when their install
    date is past the same cutoff. System / sandbox / Homebrew-formula
    apps are excluded. End-to-end on the dev Mac: 87 of 296 apps
    flagged with a 6-month cutoff.
  - **Apple Silicon compatibility** (`--insights apple-silicon`):
    runs `lipo -archs` on each app's main executable; sets
    `apple_silicon` to true / false and adds an `"intel-only binary"`
    collector note for Intel-only apps. Left `null` when lipo can't
    locate the executable.
- `audit.MarkForgotten(records, months, now)` helper applies the
  forgotten flag in a post-collection pass. Returns the count for
  logging.
- `AppRecord.InstalledAt` and `AppRecord.LastUsedAt` are now
  `*time.Time` so they're cleanly omitted from JSON when unset
  (previously serialized as `"0001-01-01T00:00:00Z"`).
- Concurrent per-app enrichers — `mdls`, `xattr`, `codesign`,
  `last_used`, and `lipo` now run via a bounded worker pool
  (`DefaultParallelism = 16`, overridable with
  `macos.WithParallelism`). Each goroutine writes to a distinct
  record index, so the slice update is race-free under
  `go test -race`. Per-record work is preserved exactly — no
  reordering, no dropped fields. End-to-end on the dev Mac:
  ~57 seconds for the same 296-app scan that previously took ~98s,
  a roughly 1.7× speedup. CPU utilization climbed from ~45% to ~83%.
- Progress-callback hook on the macOS collector
  (`macos.WithProgressFn`). The callback fires at every enricher
  boundary with the enricher's short name. The wizard wires this to
  the bubbletea `ScanModel` so users see the current phase update
  live during a wizard-mode run.
- Bubbletea `ScanView` is now started by `osaat scan` in wizard
  mode: a `tea.Program` runs in a goroutine, receives `PhaseMsg`
  events through the progress hook, exits on `DoneMsg` once the
  collector returns, and yields the terminal back for the post-scan
  summary.

### Known limitations (deferred)

- The bubbletea ScanModel exists but does not yet receive per-app
  progress events from the macOS collector — the collector runs to
  completion in one go and emits only phase-boundary log entries.
  Phase 3 wires per-app updates alongside the concurrency pass.
- Linux / Unix collectors return a "not implemented yet" error;
  Phase 4.
- No concurrency in the Go collector: per-app commands run
  sequentially. A 300-app scan takes about 80 seconds on Apple
  Silicon. Phase 3 adds goroutines.

[Unreleased]: https://github.com/simtabi/osaat/compare/v0.0.0...HEAD
