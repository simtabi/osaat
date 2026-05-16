# osaat

Audit and back up installed applications on macOS, Linux, and Unix.
Produces a structured inventory plus a restoration manifest you can run on
a new machine.

The binary is `osaat`.

## Install

Homebrew (macOS, Linux):

```sh
brew install simtabi/tap/osaat
```

Go (any supported platform):

```sh
go install github.com/simtabi/osaat/cmd/osaat@latest
```

Direct binary: see the [latest release](https://github.com/simtabi/osaat/releases/latest).
Pre-built binaries cover macOS (Intel + Apple Silicon), Linux (amd64,
arm64, 386, armv7), Windows (amd64, arm64, 386), and FreeBSD (amd64,
arm64, 386).

**Zero-dep Bash fallback (macOS only):** for the cold-start case
where Go and Homebrew aren't installed yet, use
[`scripts/bash-fallback.sh`](scripts/bash-fallback.sh). It produces a
JSON report compatible with the Go binary's schema. See
[docs/tools/bash-fallback.md](docs/tools/bash-fallback.md).

## Quick start

```sh
# Interactive wizard — auto-opens when stdin is a TTY and no flags are passed
osaat scan

# Headless
osaat scan --os macos --format pdf,markdown,txt,json --out ~/backup/
```

The wizard collects every setting, runs the scan, and prints the
equivalent non-interactive command at the end. Wizard answers can be
saved as named profiles (`osaat scan --profile <name>`).

## What gets captured

For every detected app: name, author, vendor URL, installation source
(App Store / Homebrew / pkg / DMG / direct download / system / sandbox /
unknown), original download URL, version, install date, last-used date,
size on disk, signing status, Apple Silicon compatibility (macOS), and a
reinstall command.

License keys, when detectable, go to a separate `secrets.json` —
unredacted and grouped by category — never to the audit report. Optional
`age` encryption is supported via `--age-recipient`.

## File locations

| What | Where |
|---|---|
| Generated audit outputs | `<Documents>/osaat/<YYYY-MM-DD>/` by default (overridable via `--out` or the wizard) |
| Daily log file (mode 600) | `~/.config/osaat/logs/osaat-<YYYY-MM-DD>.log` |
| Named profiles (mode 600) | `~/.config/osaat/profiles/<name>.toml` |
| Secrets file (mode 600) | `<output>/secrets.json` or `<output>/secrets.json.age` |
| Output integrity checksums | `<output>/SHA256SUMS` |

The `Documents` folder is auto-detected per OS:

- macOS / Windows: `$HOME/Documents/osaat` (or `%USERPROFILE%\Documents\osaat`).
- Linux / BSD: `$XDG_DOCUMENTS_DIR/osaat`, falling back to `$HOME/Documents/osaat`.

## Privacy

`osaat` never sends data over the network. Logs are written to disk
with $HOME paths replaced by `~` and hostname-shaped attributes
redacted, so a stolen log file doesn't identify the machine.

## Output formats

| Format | File | Use |
|---|---|---|
| PDF | `report.pdf` | Print-ready, paginated. Default. |
| Markdown | `report.md` | Renders cleanly on GitHub or in editors. Default. |
| Plain text | `report.txt` | grep-friendly, no rendering deps. Default. |
| JSON | `report.json` | Machine-readable. Required for `osaat diff`. Default. |
| CSV | `report.csv` | Spreadsheet imports. |
| HTML | `report.html` | Self-contained file with sortable table + filter input. |

## Documentation

- [Installation](docs/installation.md)
- [Configuration](docs/configuration.md) — profiles, environment, paths
- [Architecture](docs/architecture.md)
- [Release process](docs/release.md)
- [Migration / shipping checklist](docs/shipping-checklist.md)
- Per-command docs:
  [scan](docs/tools/scan.md) ·
  [diff](docs/tools/diff.md) ·
  [restore-help](docs/tools/restore-help.md) ·
  [install-schedule](docs/tools/install-schedule.md) ·
  [backup](docs/tools/backup.md)

## License

MIT — see [LICENSE](LICENSE). Copyright © 2026 Simtabi LLC.
