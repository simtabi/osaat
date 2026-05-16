# Phase 1 — macOS collector + JSON report (MVP)

**Goal:** `osaat scan --os macos --format json -o ./out/` produces a real
`report.json` for the developer's Mac. End-to-end pipeline works for one
OS + one reporter.

## Tasks

### `internal/audit/record.go`
- [ ] Implement `AppRecord`, `Source`, `SigningStatus` per
      [data-model.md](../data-model.md). All JSON tags `omitempty`
      where the field is OS-specific.
- [ ] Unit tests for JSON round-trip.

### `internal/collectors/collector.go`
- [ ] `type Collector interface { Collect(ctx) ([]AppRecord, error); Name() string }`
- [ ] Shared helpers: `runCmd(ctx, name, args...) ([]byte, error)`
      with a 30s default timeout, error-wrapping.
- [ ] `dedupe(records []AppRecord) []AppRecord` — by `BundleID` or
      path. Test with overlapping fixtures.

### `internal/collectors/macos/macos.go`
The orchestrator. Returns merged records from all sub-collectors below.

```
sub-collectors (each returns []AppRecord, all merged by BundleID):
├── applications.go     walk /Applications, ~/Applications, /System/Applications
│                       read Info.plist via howett.net/plist
│                       fields: Name, BundleID, Version, Path, InstalledAt, SizeBytes
├── system_profiler.go  shell out to `system_profiler -xml SPApplicationsDataType`
│                       parse via howett.net/plist
│                       fields: Source ("Obtained from"), Author
├── mdls.go             per app, shell out to `mdls -name kMDItemWhereFroms <path>`
│                       fields: DownloadURL
├── quarantine.go       per app, `xattr -p com.apple.quarantine <path>`
│                       fields: DownloadURL (fallback if mdls returns nothing)
├── mas.go              shell out to `mas list` once
│                       fields: AppStoreID, ReinstallCmd ("mas install <id>")
├── brew.go             shell out to `brew list --cask --versions`
│                       and `brew list --formula --versions`
│                       fields: ReinstallCmd, Source = brew_cask|brew_formula
├── pkgutil.go          shell out to `pkgutil --pkgs`
│                       fields: marks records as Source = pkg_installer
├── codesign.go         per app, `codesign -dv --verbose=4 <path>` (2>&1)
│                       fields: SigningStatus, SigningTeam, Author (Authority=)
└── lipo.go             per app, `lipo -archs <path>/Contents/MacOS/<binary>`
                        fields: AppleSilicon = ptr.Bool(true|false)
                        (gated; Phase 1 uses a stub returning nil to keep scope tight)
```

Phase 1 implements **applications.go**, **system_profiler.go**, **mdls.go**,
**quarantine.go**, **mas.go**, **brew.go**, **pkgutil.go**, **codesign.go**.
The full `lipo.go` and forgotten-apps date logic are Phase 3.

### `internal/reporters/json.go`
- [ ] `type Reporter interface { Write(records []AppRecord, w io.Writer) error }`
- [ ] JSON reporter writes a top-level object:
      ```json
      { "schema": "osaat.report/v1", "generated_at": "...", "host": {...},
        "records": [ ... ] }
      ```
- [ ] Stable order: by `Name` (case-insensitive), then `BundleID`.

### `cmd/osaat/scan.go`
- [ ] Wire flags:
      `--os <macos|linux|unix|auto>`, `--format <json,...>`,
      `--out <dir>`, `--license-mode <none|best-effort|...>`,
      `--non-interactive`.
- [ ] For Phase 1, license-mode is accepted but only `none` actually
      works; others print "TODO".
- [ ] Detect OS via `runtime.GOOS` when `--os auto`.
- [ ] Call `audit.Run(ctx, opts)` → `[]AppRecord`.
- [ ] For each requested format, instantiate reporter, write to
      `<out>/report.<ext>`.
- [ ] Exit code: 0 on success, 1 on collector errors, 2 on misuse.

### Fixtures (`testdata/`)
- [ ] One real Info.plist (e.g. from a known app, anonymized if
      needed).
- [ ] A 50-line slice of `system_profiler SPApplicationsDataType -xml`
      output captured from the dev machine.
- [ ] A sample `mas list` output line.
- [ ] A sample `codesign -dv` output block.

### Tests
- [ ] Each sub-collector has a unit test against its fixture, with
      no real subprocess — collectors take an injectable `runCmd` so
      tests can return canned output.
- [ ] `audit.Run` integration test runs the macOS collector against
      a temp fixture dir and asserts the merged output.
- [ ] Reporter test: round-trip a small `[]AppRecord` through JSON
      and assert canonical shape.

### Definition of done

```bash
$ osaat scan --os macos --format json --license-mode none --out ./out
[osaat] scanning /Applications, ~/Applications, /System/Applications...
[osaat] 142 apps found.
[osaat] wrote ./out/report.json

$ jq '.records | length' ./out/report.json
142

$ jq '.records[] | select(.source == "app_store") | .name' ./out/report.json | head
"Keynote"
"Pages"
"Numbers"
...
```

`make test && make lint` green. No wizard (still Phase 2).

## Non-goals for Phase 1

- License extraction (Phase 2).
- Wizard (Phase 2).
- CSV/MD/HTML reporters (Phase 2).
- Restoration manifest (Phase 2).
- Linux / Unix collectors (Phase 4).
- Apple Silicon column (Phase 3).
- Schedule install (Phase 3).
