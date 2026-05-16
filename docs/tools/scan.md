# `osaat scan`

**Status:** skeleton — full body in Phase 1 (macOS collector) and Phase 2 (wizard, all formats, licenses).

Scan the system, produce an audit report and optionally a restoration manifest.

## Usage

```sh
osaat scan [flags]
```

When stdin is a TTY and no flags are passed, `scan` opens an interactive
wizard. Otherwise it runs headlessly with the flag values.

## Flags

| Flag | Default | Effect |
|---|---|---|
| `--os <macos\|linux\|unix\|auto>` | `auto` | Collector set to run. `auto` uses `runtime.GOOS`. |
| `--format <list>` | `json` | Comma-separated: `json`, `csv`, `markdown`, `html`. |
| `--out <dir>` | `./osaat-out/` | Output directory. |
| `--license-mode <mode>` | `none` | `best-effort`, `checklist`, `aggressive`, or `none`. |
| `--age-recipient <key>` | _none_ | When set, write `secrets.json.age` instead of `secrets.json`. |
| `--insights <list>` | _none_ | Comma-separated: `forgotten`, `apple-silicon`. |
| `--insights-forgotten-months <N>` | `6` | Months of inactivity that mark an app as forgotten. |
| `--with-restore` | `false` | Also emit Brewfile, mas list, apt/dnf/pacman lists, `RESTORE.md`. |
| `--profile <name>` | _none_ | Load defaults from `~/.config/osaat/profiles/<name>.toml`. |
| `--interactive` | `false` | Force the wizard even if flags are passed. |
| `--non-interactive` | `false` | Forbid the wizard. CI-safe. |
| `--verbose` | `false` | Verbose log output. |

## Examples

Interactive:

```sh
osaat scan
```

Headless macOS scan:

```sh
osaat scan --os macos --format json,markdown --with-restore --out ./out/
```

Daily encrypted backup:

```sh
osaat scan --profile imani-mbp --out ~/backup/osaat-$(date +%F)/
```

## Output

```
<out>/
├── report.json          canonical audit; no keys
├── report.csv           if csv format selected
├── REPORT.md            if markdown format selected
├── report.html          if html format selected
├── secrets.json[.age]   if license-mode != none
├── Brewfile             if --with-restore
├── mas-apps.txt         if --with-restore
├── apt-packages.txt     if --with-restore and dpkg detected
├── RESTORE.md           if --with-restore
└── osaat-metadata.json  always
```

## Exit codes

| Code | Meaning |
|---|---|
| 0 | Success. |
| 1 | One or more collectors errored. Partial output may have been written. |
| 2 | Flag misuse — see stderr. |
| 130 | User cancelled the wizard. |
