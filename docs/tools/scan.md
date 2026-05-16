# `osaat scan`

Scan the system, produce an audit report, and optionally a restoration
manifest.

## Usage

```sh
osaat scan [flags]
```

When stdin is a TTY and no scan-shaping flags are passed, `scan` opens
an interactive wizard built on the `charmbracelet/huh` form library.
The wizard collects every setting, runs the scan, prints the equivalent
non-interactive command at the end, and optionally saves the answers
as a named profile.

Pass `--non-interactive` to forbid the wizard (CI-safe), or
`--interactive` to force it.

## Flags

| Flag | Default | Effect |
|---|---|---|
| `--os <macos\|linux\|unix\|auto>` | `auto` | Collector set. `auto` resolves to the host OS. |
| `--format <list>` | `pdf,markdown,txt,json` | Comma-separated: `pdf`, `markdown` (or `md`), `txt`, `json`, `csv`, `html`. |
| `--out <dir>` | `<Documents>/osaat/<YYYY-MM-DD>` | Output directory. Created if missing. |
| `--license-mode <mode>` | `none` | `none`, `checklist`, `best-effort`, or `aggressive`. |
| `--age-recipient <key>` | _none_ | Write `secrets.json.age` instead of `secrets.json`. |
| `--insights <list>` | _none_ | Comma-separated: `forgotten`, `apple-silicon` (Phase 3). |
| `--insights-forgotten-months <N>` | `6` | Months of inactivity that flag an app as forgotten. |
| `--with-restore` | `false` | Also emit Brewfile, mas list, RESTORE.md. |
| `--profile <name>` | _none_ | Load defaults from `~/.config/osaat/profiles/<name>.toml`. |
| `--interactive` | `false` | Force the wizard even if flags are passed. |
| `--non-interactive` | `false` | Forbid the wizard. CI-safe. |
| `--verbose` | `false` | Debug-level log output to stderr + file. |
| `--quiet` | `false` | Suppress per-file `wrote ...` lines. |

## Examples

Interactive (default when run from a terminal):

```sh
osaat scan
```

Headless macOS scan with all default formats:

```sh
osaat scan --os macos --with-restore
# defaults --out to ~/Documents/osaat/<today>
```

Encrypted secrets:

```sh
osaat scan --license-mode best-effort \
    --age-recipient $(cat ~/.age/recipient.txt)
```

Replay a saved profile:

```sh
osaat scan --profile imani-mbp
```

## Output

```
<out>/
├── report.pdf           if pdf format selected (default)
├── report.md            if markdown format selected (default)
├── report.txt           if txt format selected (default)
├── report.json          canonical audit; no keys (default; required by osaat diff)
├── report.csv           if csv format selected
├── report.html          if html format selected
├── Brewfile             if --with-restore
├── mas-apps.txt         if --with-restore
├── RESTORE.md           if --with-restore
├── secrets.json[.age]   if license-mode != none (mode 600 when plain)
└── SHA256SUMS           always — sha256sum-compatible digest of every file above
```

## Logs

Every run appends to `~/.config/osaat/logs/osaat-<YYYY-MM-DD>.log`
(mode 600). The logger scrubs `$HOME` paths to `~` and redacts
hostname-shaped fields so the file is safe to share or version-control.

## Exit codes

| Code | Meaning |
|---|---|
| 0 | Success. |
| 1 | One or more collectors or reporters errored. |
| 2 | Flag misuse — see stderr. |
| 130 | User cancelled the wizard (`ctrl-c` / `esc`). |
