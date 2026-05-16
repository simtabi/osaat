# Phased build plan

**Status:** ratified, 2026-05-16

Each phase is a self-contained unit of work that ends with a runnable
artifact and a green CI. Phases ship in order; nothing later depends on
work that isn't done yet.

| Phase | File | What ships |
|---|---|---|
| 0 | [phase-0-scaffold.md](phase-0-scaffold.md) | Renamed repo + full Simtabi OSS tree + Go module + CI/release + Cobra skeleton |
| 1 | [phase-1-macos-collector.md](phase-1-macos-collector.md) | `osaat scan --os macos --format json` produces a real `report.json` |
| 2 | [phase-2-ux-reporters-licenses-restore.md](phase-2-ux-reporters-licenses-restore.md) | Wizard + CSV/MD/HTML + license modes + Brewfile/mas/manifest + diff |
| 3 | [phase-3-insight-automation.md](phase-3-insight-automation.md) | Forgotten apps + Apple Silicon + launchd/systemd scheduling + encrypted backup bundle |
| 4 | [phase-4-linux-unix-collectors.md](phase-4-linux-unix-collectors.md) | Linux (dpkg/rpm/pacman/snap/flatpak/AppImage) + Unix/BSD collectors |
| 5 | [phase-5-release.md](phase-5-release.md) | Public repo, Homebrew tap, `v0.1.0` |

## Rules across phases

1. **Every phase ends green.** CI passes, no broken-but-stubbed
   subcommands surfaced to the user.
2. **No phase pushes to GitHub.** Public release is Phase 5 only. Up to
   that point everything is local.
3. **No phase rewrites prior history.** Append-only commits per global
   CLAUDE.md.
4. **A phase that misses its scope gets a follow-up file**
   (e.g. `phase-2.1-foo.md`), not bloat into the next phase.
5. **`MEMORY.md` is updated** at the end of each phase with any
   non-obvious decisions made during that phase.

## Out of scope across all phases (v0.1)

- Windows support.
- GUI.
- Cloud sync / hosted backup destination.
- Pluggable third-party "vendor lookup" services (e.g. paid license
  databases). The tool reads what's on the machine; it doesn't call
  external APIs.

## Definition of "done" for v0.1

- Single `osaat` binary per platform on GitHub Releases.
- Homebrew tap `simtabi/tap` formula works: `brew install simtabi/tap/osaat`.
- `osaat scan` produces all four output formats + the restore manifest.
- Wizard works in a TTY; flag mode works headlessly.
- Linux collector handles at least dpkg+rpm+pacman+snap+flatpak.
- `docs/` is complete (installation, configuration, architecture,
  release, shipping-checklist, tools/*).
- README on github.com/simtabi/osaat renders cleanly.
