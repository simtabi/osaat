# Wizard UX

**Status:** ratified, 2026-05-16

## When the wizard runs

| Condition | Behavior |
|---|---|
| `stdin.IsTerminal() && no flags passed` | Wizard runs |
| `--interactive` flag passed | Wizard runs (forced) |
| `--non-interactive` flag passed | Wizard refuses (CI-safe) |
| Any other flag passed | Strict flag mode; no wizard |
| `osaat <subcommand> --help` | Cobra help; no wizard |

This rule means `osaat scan` opens the wizard, `osaat scan --format json`
runs headless, and CI never gets surprised.

## Flow

The wizard is one `huh.Form` composed of multiple groups (pages). User can
go back with `<- arrow` and forward with `->`. `esc` cancels and exits with
no side effects.

### Page 1 — Scope

```
┌─ osaat: audit installed applications ─────────────────────────────┐
│                                                                   │
│  Which collectors should run?  (space to toggle, enter to confirm) │
│  ❯ ◉ macOS                                                        │
│    ○ Linux  (dpkg / rpm / pacman / snap / flatpak)                │
│    ○ Unix   (BSD pkg, generic POSIX)                              │
│                                                                   │
│  Auto-detected: darwin (macOS arm64)                              │
└───────────────────────────────────────────────────────────────────┘
```

Default selects the auto-detected OS; user can add others (e.g. running
under macOS but auditing a mounted Linux drive).

### Page 2 — Output

```
┌─ output ──────────────────────────────────────────────────────────┐
│                                                                   │
│  Output formats:  (multi-select)                                  │
│    ◉ JSON      (report.json — the canonical data)                 │
│    ◉ Markdown  (REPORT.md — for README / sharing)                 │
│    ○ CSV       (report.csv — for spreadsheet)                     │
│    ○ HTML      (report.html — for opening in a browser)           │
│                                                                   │
│  Output directory:                                                │
│  > ~/backup/osaat-2026-05-16/                                     │
│                                                                   │
│  Also write a restoration manifest?  ◉ yes  ○ no                  │
│  (Brewfile + mas-list + apt list + RESTORE.md)                    │
└───────────────────────────────────────────────────────────────────┘
```

### Page 3 — License extraction

```
┌─ license keys ────────────────────────────────────────────────────┐
│                                                                   │
│  License extraction mode:                                         │
│  ❯ best-effort   scan plists + AppSupport, flag unknowns          │
│    checklist     emit "look here" pointers, no extraction         │
│    aggressive    also dump Keychain (prompts password per item)   │
│    none          skip entirely                                    │
│                                                                   │
│  Encrypt secrets.json with age?  ○ no  ◉ yes                      │
│  age recipient:                                                   │
│  > age1qz8tnxw7p...                                               │
│                                                                   │
│  Note: secrets.json holds full unredacted keys, organized by      │
│  category. report.json never includes keys. With encryption       │
│  enabled, only secrets.json.age is written.                       │
└───────────────────────────────────────────────────────────────────┘
```

### Page 4 — Insights

```
┌─ insights (optional columns) ─────────────────────────────────────┐
│                                                                   │
│  ◉ Forgotten apps (not used in N months)                          │
│      N = [ 6 ] months                                             │
│  ◉ Apple Silicon compatibility column (macOS only)                │
│  ○ Login items and LaunchAgents/Daemons                           │
│  ○ Browser extensions (Safari / Chrome / Firefox)                 │
└───────────────────────────────────────────────────────────────────┘
```

### Page 5 — Save as profile

```
┌─ save profile ────────────────────────────────────────────────────┐
│                                                                   │
│  Save these answers as a profile for next time?  ◉ yes  ○ no      │
│  Profile name:                                                    │
│  > imani-mbp                                                      │
│                                                                   │
│  Will be written to ~/.config/osaat/profiles/imani-mbp.toml       │
└───────────────────────────────────────────────────────────────────┘
```

### Scan view

After confirmation, `huh` hands off to `bubbletea` for the live scan view:

```
Scanning /Applications                ████████████████████░░░  142/172

  Found    : 142 apps
  App Store: 18    Brew cask: 41    pkg: 8    direct: 75
  Notes    : 3 unsigned   1 Intel-only   2 forgotten (> 6 months)

  Currently inspecting: Adobe Creative Cloud.app
```

This is where `bubbletea` earns its keep over `huh`: live multi-section
updates with the spinner and counters refreshing in place.

### Done view

```
✓ Scan complete in 38.4s

  Outputs:
    ~/backup/osaat-2026-05-16/report.json
    ~/backup/osaat-2026-05-16/REPORT.md
    ~/backup/osaat-2026-05-16/secrets.json.age
    ~/backup/osaat-2026-05-16/Brewfile
    ~/backup/osaat-2026-05-16/mas-apps.txt
    ~/backup/osaat-2026-05-16/RESTORE.md

  Run this scan non-interactively next time:

    osaat scan \
        --os macos \
        --format json,markdown \
        --license-mode best-effort \
        --age-recipient age1qz8tnxw7p... \
        --insights forgotten,apple-silicon \
        --insights-forgotten-months 6 \
        --with-restore \
        --out ~/backup/osaat-2026-05-16/

  Or load the saved profile:

    osaat scan --profile imani-mbp
```

The replay command is the teaching moment — it's how users discover the
flag set without reading `--help`.

## Profile file (TOML)

```toml
# ~/.config/osaat/profiles/imani-mbp.toml
schema = "osaat.profile/v1"
os = ["macos"]
formats = ["json", "markdown"]
out = "~/backup/osaat-{date}/"

[license]
mode = "best-effort"
age_recipient = "age1qz8tnxw7p..."

[insights]
forgotten = true
forgotten_months = 6
apple_silicon = true

[restore]
enabled = true
```

`{date}` is a documented template variable for output paths so a daily
profile produces distinct directories.
