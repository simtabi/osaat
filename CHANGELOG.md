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

### Known limitations (deferred)

- `--license-mode` flag is recognized but emits a warning; full
  implementation is Phase 2.
- CSV / Markdown / HTML reporters return a "not implemented yet"
  error; Phase 2.
- Linux / Unix collectors return a "not implemented yet" error; Phase 4.
- No concurrency: collectors run sequentially per app. A 300-app scan
  takes about 70 seconds on Apple Silicon. Phase 2 or 3 adds goroutines.

[Unreleased]: https://github.com/simtabi/osaat/compare/v0.0.0...HEAD
