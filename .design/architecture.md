# Architecture

**Status:** ratified, 2026-05-16

## Overview

`osaat` is a Go CLI that collects an application inventory from the host OS,
normalizes it into a single `AppRecord` model, and emits reports + a
restoration manifest. An interactive wizard sits above the CLI and only
produces the same flag set a scripted invocation would; the CLI is the source
of truth.

```
┌───────────────────────────────────────────────────────────────────────┐
│  cmd/osaat (cobra)                                                    │
│    scan │ diff │ restore-help │ install-schedule │ backup │ version   │
└────────┬──────────────────────────────────────────────────────────────┘
         │
         ▼
┌───────────────────────────────────────────────────────────────────────┐
│  wizard (huh, optional)                                               │
│    only when stdin is a TTY AND no flags were passed                  │
│    output: a populated *cobra.Command flag set + a replay command     │
└────────┬──────────────────────────────────────────────────────────────┘
         │
         ▼
┌───────────────────────────────────────────────────────────────────────┐
│  audit orchestrator                                                   │
│    selects collector(s) per --os flag (or runtime.GOOS detection)     │
└─┬──────────────────────────┬─────────────────────────┬────────────────┘
  │                          │                         │
  ▼                          ▼                         ▼
┌────────────┐         ┌────────────┐            ┌────────────┐
│ collectors │         │ collectors │            │ collectors │
│ /macos     │         │ /linux     │            │ /unix      │
└──────┬─────┘         └──────┬─────┘            └──────┬─────┘
       │                      │                         │
       └──────────┬───────────┴─────────────┬──────────┘
                  ▼                         ▼
       ┌────────────────────┐     ┌────────────────────┐
       │ licenses (3 modes) │     │ insight extras     │
       │  best_effort       │     │  forgotten apps    │
       │  checklist         │     │  apple silicon     │
       │  keychain          │     │  signing status    │
       └─────────┬──────────┘     └─────────┬──────────┘
                 │                          │
                 └───────────┬──────────────┘
                             ▼
                  ┌────────────────────┐
                  │ []AppRecord        │
                  │  + Secrets         │
                  └─────────┬──────────┘
                            ▼
       ┌──────────────────────────────────────┐
       │ reporters   │ restore   │ schedule   │
       │  json       │  brewfile │  launchd   │
       │  csv        │  mas list │  systemd   │
       │  markdown   │  apt list │            │
       │  html       │  manifest │            │
       │             │  archive  │            │
       │             │  (age)    │            │
       └──────────────────────────────────────┘
```

## Module boundaries

| Package | Responsibility | Allowed deps |
|---|---|---|
| `cmd/osaat` | argparse via cobra; wire wizard → flags → orchestrator | cobra, internal/* |
| `internal/audit` | `AppRecord` struct; orchestrator that calls collectors and reporters | stdlib + internal/collectors, internal/reporters |
| `internal/collectors/{macos,linux,unix}` | OS-specific data collection; return `[]AppRecord` | howett.net/plist (macos), gopsutil, os/exec |
| `internal/licenses` | Three scanner modes; populate `Secrets` | os/exec (security), howett.net/plist |
| `internal/reporters` | Format `[]AppRecord` into JSON/CSV/MD/HTML | stdlib only |
| `internal/restore` | Brewfile/mas/apt manifest + encrypted archive | filippo.io/age, archive/tar |
| `internal/wizard` | huh-based form; optional bubbletea screens | charmbracelet/huh, charmbracelet/bubbletea, charmbracelet/lipgloss |
| `internal/profiles` | TOML save/load for named profiles | pelletier/go-toml/v2 |
| `internal/schedule` | launchd plist + systemd unit generators | text/template only |
| `internal/secrets` | Schema + age crypt for `secrets.json` | filippo.io/age |
| `internal/version` | Single source-of-truth version string | stdlib |

Rules:

1. `internal/reporters` never imports collectors. It only knows `AppRecord`.
2. `internal/collectors/*` never import each other. Cross-platform shared
   helpers live in `internal/collectors/collector.go`.
3. `cmd/osaat` is the only package allowed to read CLI flags. Everything
   below takes options structs.
4. Wizard output is a populated options struct, not direct calls into the
   orchestrator. Tested by running both flag-mode and wizard-mode against
   the same fixture and asserting identical orchestrator inputs.

## Data flow

1. **Entry.** `cmd/osaat scan` parses flags via cobra. If TTY + no flags,
   it invokes `wizard.Run()` which returns a fully-populated options
   struct.
2. **OS selection.** `audit.Run(opts)` consults `opts.OS` (override) or
   `runtime.GOOS` to choose the collector set. `--os all` runs every
   collector that detects its prerequisites are present.
3. **Collection.** Each collector returns `[]AppRecord`. A merge step
   dedupes by `BundleID` (macOS) or `PkgID` (Linux/Unix), preferring
   richer sources (e.g. `system_profiler` over a raw `/Applications`
   filesystem walk).
4. **License pass.** If `--license-mode != none`, the chosen scanner
   walks each record and populates the `Secrets` map keyed by record.
5. **Insight pass.** Forgotten-apps and Apple Silicon columns are
   computed last; they need data from the collection pass.
6. **Reporters.** For each `--format`, the corresponding reporter writes
   to disk under `--out`.
7. **Restore manifest.** If `--with-restore` (default in wizard mode),
   `restore.Manifest(records).Write(...)` emits Brewfile, mas-list,
   apt/dnf/pacman list, and `RESTORE.md`.
8. **Backup bundle.** If `--backup`, the whole output dir is tar+age'd.

## Extension points

- **New collector:** drop a file under `internal/collectors/<os>/`,
  implement the `Collector` interface, register in the orchestrator's
  switch.
- **New reporter:** drop a file under `internal/reporters/`, implement
  the `Reporter` interface, register by format name.
- **New license scanner:** drop a file under `internal/licenses/`,
  implement `Scanner`, register by mode name.

## What lives outside Go

- `scripts/bash-fallback.sh` — zero-dep macOS-only subset that produces
  a JSON shaped identically to a real `osaat scan --format json`. For
  the "I'm on a fresh Mac with nothing but bash" recovery scenario.
- Everything else: pure Go.

## Out of scope for v0.1

- Windows. (Folder layout is flexible enough to add `collectors/windows/`
  later, but no work done now.)
- GUI. The wizard is the UX ceiling.
- Cloud sync / hosted backup. Output is local files; user moves them.
