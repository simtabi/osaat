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

### Known limitations (deferred)

- The interactive wizard, named profiles, and bubbletea live scan
  view are Phase 2b.2.
- `scripts/bash-fallback.sh` (zero-dep macOS recovery) is Phase 2c.
- Linux / Unix collectors return a "not implemented yet" error;
  Phase 4.
- No concurrency: collectors run sequentially per app. A 300-app scan
  takes about 80 seconds on Apple Silicon. Phase 3 adds goroutines.

[Unreleased]: https://github.com/simtabi/osaat/compare/v0.0.0...HEAD
