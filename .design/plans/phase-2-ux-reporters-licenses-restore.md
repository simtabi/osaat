# Phase 2 — wizard + reporters + licenses + restore + diff

**Goal:** the UX layer lands and the macOS scan becomes useful for real
backup work. Every format ships, the wizard ties it all together, and
`osaat diff` lets you compare two reports.

## Tasks

### `internal/wizard/wizard.go`
- [ ] huh `Form` composed of the 5 groups described in
      [../wizard-ux.md](../wizard-ux.md).
- [ ] Returns a populated `scan.Options` struct.
- [ ] Renders the "next-time" replay command at the end.
- [ ] Auto-trigger rule in `cmd/osaat/scan.go`:
      ```go
      if isatty.IsTerminal(os.Stdin.Fd()) && !flagsProvided(cmd) && !opts.NonInteractive {
          opts = wizard.Run(ctx, defaults)
      }
      ```

### `internal/wizard/scanview.go` (bubbletea)
- [ ] Live progress view shown during scan in TTY mode.
- [ ] Multi-section: progress bar, found counters, current app name,
      notes counter.
- [ ] Falls back to `schollz/progressbar` lines when not a TTY.

### `internal/profiles/`
- [ ] `profile.go` — load/save TOML per schema in
      [../wizard-ux.md](../wizard-ux.md).
- [ ] `store.go` — directory `~/.config/osaat/profiles/` (XDG_CONFIG_HOME-aware).
- [ ] `--profile <name>` flag wired into `cmd/osaat/scan.go`.

### Reporters
- [ ] `internal/reporters/csv.go` — flat CSV, one row per app. Columns
      ordered to match the prompt's spec.
- [ ] `internal/reporters/markdown.go` — pipe-delimited table; small
      preamble with host metadata; collapsible per-source sections.
- [ ] `internal/reporters/html.go` — single-file HTML with sortable
      table (no JS framework; vanilla `<script>` for sort).
      Use stdlib `html/template`.

### License scanners
- [ ] `internal/licenses/scanner.go` — `type Scanner interface
      { Scan(ctx, records) (Secrets, error); Mode() string }`.
- [ ] `internal/licenses/best_effort.go`:
      - walks `~/Library/Application Support/<app>/`,
        `~/Library/Preferences/<bundle-id>.plist`
      - regex for likely keys (UUID, hex≥16, dash-separated
        groups, email-shaped fields)
      - confidence: high if field name matches /licen[cs]e|serial|key|registration/, medium if pattern-only, low otherwise
- [ ] `internal/licenses/checklist.go`:
      - emit one ChecklistItem per detected app with `look_here`
        pointing at the conventional plist path; no extraction.
- [ ] `internal/licenses/keychain.go`:
      - shells `security dump-keychain login.keychain` and parses
      - **interactive**: prompts user per Keychain item via huh confirm
      - records `Source: "keychain:<service>"`
- [ ] All three scanners produce a `Secrets` value that
      `internal/secrets` writes to disk.

### `internal/secrets/`
- [ ] `schema.go` — types per [../data-model.md](../data-model.md).
- [ ] `writer.go` — writes `secrets.json` (plain) OR `secrets.json.age`
      (encrypted to the recipient passed in).
- [ ] **Full keys, no redaction.** Confirmed in
      [../prompt.md](../prompt.md) decision log.

### Restore manifest
- [ ] `internal/restore/brewfile.go` — emits Homebrew Brewfile
      from records with `Source == brew_cask|brew_formula`.
- [ ] `internal/restore/mas_list.go` — emits `mas-apps.txt` (one
      App Store ID per line) and a parallel `mas-apps.md` with
      app names for human review.
- [ ] `internal/restore/manifest.go` — `RESTORE.md` per-app
      manual-install table for apps that can't be rehydrated by
      `brew bundle` or `mas install`.
- [ ] `cmd/osaat/scan.go` — `--with-restore` flag (default `true`
      when wizard sets it).

### `osaat diff`
- [ ] `cmd/osaat/diff.go` — load two `report.json` files, diff
      `[]AppRecord` by `BundleID || PkgID || (Name+Version)`.
- [ ] Output formats: text (default), JSON (`--format json`),
      markdown (`--format markdown`).
- [ ] Categories: added, removed, version-changed, source-changed.

### `scripts/bash-fallback.sh`
- [ ] Pure Bash 3.2-compatible (macOS default).
- [ ] Produces a `report.json` schema-compatible with the Go binary.
- [ ] No license extraction, no encryption — recovery-only.
- [ ] `shellcheck` clean.

### Tests
- [ ] Wizard test: feed a scripted answer sequence, assert resulting
      `Options` struct.
- [ ] License scanners: fixture-driven (sample plist with a key field,
      sample security-dump output).
- [ ] Diff tests: matrix of added/removed/changed.
- [ ] Bash fallback: compare its output against the Go binary's JSON
      schema on the dev machine.

### Definition of done

```bash
$ osaat scan
[wizard opens; user walks through; scan runs with live view]

$ ls ./out
report.json  report.csv  REPORT.md  report.html
secrets.json.age
Brewfile  mas-apps.txt  mas-apps.md  RESTORE.md

$ osaat diff ./old/report.json ./new/report.json
+ 1Password (8.10 → 8.11)
- Skitch
~ Adobe Photoshop  (source: dmg → brew_cask)
```

`make test && make lint` green.

## Non-goals for Phase 2

- Forgotten apps / Apple Silicon column (Phase 3).
- Scheduling (Phase 3).
- Encrypted backup bundle (Phase 3).
- Linux/Unix collectors (Phase 4).
- Public release (Phase 5).
