# Phase 3 — insight columns + automation + backup bundle

**Goal:** the v0.1 feature surface is complete. Forgotten apps and Apple
Silicon columns are populated. Scheduled audits work. The encrypted
backup bundle (`osaat backup`) produces a single archive.

## Tasks

### Insights — forgotten apps
- [ ] `internal/collectors/macos/last_used.go`:
      shell `mdls -name kMDItemLastUsedDate` per app.
      Populate `AppRecord.LastUsedAt`.
- [ ] Linux equivalent: file `atime` of the package's main binary.
      Slated for Phase 4 (need access to dpkg metadata for the
      primary binary path).
- [ ] `internal/audit/insights.go` — `MarkForgotten(records, months)`:
      sets a synthetic note `"forgotten: not used in N months"` in
      `CollectorNotes` when `LastUsedAt` is older than the cutoff or
      missing.
- [ ] Reporter changes:
      - JSON: no schema change; readers consume `LastUsedAt` directly.
      - CSV: append `last_used_at` column.
      - Markdown: section "Forgotten apps" lists them with last-used
        date if known.
      - HTML: same; flagged row class.

### Insights — Apple Silicon (macOS)
- [ ] Promote the stubbed `internal/collectors/macos/lipo.go` to a
      real implementation. For each app:
      ```
      lipo -archs <Contents/MacOS/<binary>>
      ```
      Parse the output (space-separated archs). Set
      `AppRecord.AppleSilicon` = `arm64 in archs`.
- [ ] If the app isn't a Mach-O binary (e.g. Electron with a top-level
      script), traverse `Contents/MacOS/*` and prefer the binary named
      after `CFBundleExecutable`.
- [ ] Reporter changes:
      - CSV: append `apple_silicon` column.
      - Markdown: column with values `arm64 / intel-only / unknown`.
      - HTML: same; flagged row class for intel-only.

### Scheduling — launchd (macOS)
- [ ] `internal/schedule/launchd.go`:
      generate a `~/Library/LaunchAgents/com.simtabi.osaat.<profile>.plist`
      with a `StartCalendarInterval` block.
- [ ] Templating via stdlib `text/template`.
- [ ] `osaat install-schedule --weekly` writes the plist and runs
      `launchctl load` (with `--dry-run` to preview).
- [ ] `osaat install-schedule --uninstall` removes it.
- [ ] Wizard offers "schedule this audit weekly?" as an opt-in.

### Scheduling — systemd (Linux)
- [ ] `internal/schedule/systemd.go`:
      generate `~/.config/systemd/user/osaat.service` +
      `osaat.timer` (OnCalendar=weekly).
- [ ] `osaat install-schedule` runs `systemctl --user
      enable --now osaat.timer` (with `--dry-run`).
- [ ] `osaat install-schedule --uninstall` removes and disables.

### Backup bundle
- [ ] `internal/restore/archive.go`:
      build a tar of `report.json`, `REPORT.md`, `secrets.json.age`,
      `Brewfile`, `mas-apps.txt`, `RESTORE.md`,
      `osaat-metadata.json`. Wrap with `filippo.io/age` to
      recipient(s) passed in. Stream — never materialize uncompressed
      tar in memory.
- [ ] `cmd/osaat/backup.go`:
      `osaat backup --from <dir> --age-recipient <key> --out
      <file.tar.age>`.
- [ ] If `--from` omits a `secrets.json[.age]`, warn but proceed
      (audit-only bundles are valid).
- [ ] `osaat backup --decrypt --in <file.tar.age> --age-key
      <path> --out <dir>` round-trips.

### Tests
- [ ] `mdls` and `lipo` collector tests with fixtures.
- [ ] launchd plist + systemd unit generator tests against golden
      files.
- [ ] Backup round-trip test: build a bundle, decrypt, verify
      bit-identical files come out.

### Definition of done

```bash
$ osaat scan --insights forgotten,apple-silicon --insights-forgotten-months 6
[scan runs; report.json now includes last_used_at + apple_silicon per app]

$ osaat install-schedule --weekly --out ~/backup/osaat-{date}/
✓ wrote ~/Library/LaunchAgents/com.simtabi.osaat.default.plist
✓ launchctl loaded

$ osaat backup --from ~/backup/osaat-2026-05-16/ \
       --age-recipient $(cat ~/.age/recipient.txt) \
       --out ~/backup/osaat-2026-05-16.tar.age
✓ wrote 1 archive (412 KB)

$ osaat backup --decrypt \
       --in ~/backup/osaat-2026-05-16.tar.age \
       --age-key ~/.age/key.txt \
       --out /tmp/restore-check
✓ extracted 7 files
```

`make test && make lint` green.

## Non-goals for Phase 3

- Linux / Unix collectors (Phase 4).
- Public release (Phase 5).
