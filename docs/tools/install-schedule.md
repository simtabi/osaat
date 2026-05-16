# `osaat install-schedule`

**Status:** skeleton — full body in Phase 3.

Install (or uninstall) a recurring scan via the platform scheduler.

## Usage

```sh
osaat install-schedule --weekly [--out <dir>] [--profile <name>] [--dry-run]
osaat install-schedule --uninstall
```

## What gets written

### macOS (launchd)

```
~/Library/LaunchAgents/com.simtabi.osaat.<profile>.plist
```

Loaded with `launchctl bootstrap gui/$UID`. The plist runs `osaat scan`
with the same flags `install-schedule` was invoked with.

### Linux (systemd user)

```
~/.config/systemd/user/osaat.service
~/.config/systemd/user/osaat.timer
```

Enabled with `systemctl --user enable --now osaat.timer`.

## Flags

| Flag | Default | Effect |
|---|---|---|
| `--weekly` | _required if not `--daily`/`--monthly`_ | Schedule cadence. |
| `--daily` | | Run daily. |
| `--monthly` | | Run monthly (1st of month). |
| `--out <dir>` | `~/backup/osaat-{date}/` | Output dir for each scheduled run; `{date}` expands at run time. |
| `--profile <name>` | _none_ | Scan profile to load. |
| `--dry-run` | `false` | Print what would be written; do nothing. |
| `--uninstall` | `false` | Remove the schedule. |

## Examples

```sh
# Weekly audit every Monday morning into a date-stamped dir
osaat install-schedule --weekly --profile imani-mbp

# Preview without writing
osaat install-schedule --weekly --dry-run

# Tear it down
osaat install-schedule --uninstall
```
