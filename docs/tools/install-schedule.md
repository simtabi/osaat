# `osaat install-schedule`

Install (or uninstall) a recurring `osaat scan` via the platform
scheduler — `launchd` on macOS, `systemd --user` on Linux. Every
scheduled run fires at 06:00 local time (matches the Dependabot /
Simtabi-wide schedule convention).

## Usage

```sh
osaat install-schedule --weekly [flags]
osaat install-schedule --daily [flags]
osaat install-schedule --monthly [flags]
osaat install-schedule --uninstall
```

## Flags

| Flag | Default | Effect |
|---|---|---|
| `--weekly` | _one required_ | Run Monday at 06:00. |
| `--daily` |  | Run every day at 06:00. |
| `--monthly` |  | Run on the 1st of each month at 06:00. |
| `--uninstall` | `false` | Remove the schedule. |
| `--dry-run` | `false` | Print the plan; don't write anything. |
| `--profile <name>` | _none_ | Scan profile to load at run time. |
| `--out <dir>` | `~/Documents/osaat/{date}` | Output directory. `{date}` expands to YYYY-MM-DD at run time. |
| `--label <name>` | `com.simtabi.osaat` / `osaat` | Scheduler unit name. |
| `--extra-arg <arg>` | _none_ | Repeatable. Each value is appended to the scheduled scan command. |

## What gets written

### macOS (launchd)

`~/Library/LaunchAgents/<label>.plist` — a LaunchAgent that fires
`osaat scan --non-interactive --out <OUT_DIR>` on the configured
cadence. Standard streams are redirected to
`~/.config/osaat/logs/launchd-{stdout,stderr}.log`.

Loaded with:

```sh
launchctl bootstrap gui/$UID <plist>
```

### Linux (systemd --user)

Two files:

- `~/.config/systemd/user/<label>.service` — `Type=oneshot` invoking
  `osaat scan ...`
- `~/.config/systemd/user/<label>.timer` — `OnCalendar=...` per
  cadence, `Persistent=true` so missed runs catch up after a reboot.

Enabled with:

```sh
systemctl --user daemon-reload
systemctl --user enable --now <label>.timer
```

## Examples

```sh
# Weekly audit, using a saved profile
osaat install-schedule --weekly --profile imani-mbp

# Daily audit with one extra flag
osaat install-schedule --daily --extra-arg --license-mode=best-effort

# Preview the plan without writing
osaat install-schedule --monthly --dry-run

# Remove the schedule
osaat install-schedule --uninstall
```

## Exit codes

| Code | Meaning |
|---|---|
| 0 | Schedule installed / removed (or plan printed for `--dry-run`). |
| 1 | Filesystem error, missing `launchctl` / `systemctl`, or load failure. |
| 2 | Flag misuse — see stderr. |

## Notes

- The scheduler needs an absolute, stable path to the `osaat` binary.
  `install-schedule` resolves the current binary via `os.Executable()`
  and follows any leading symlink, so installing from
  `./bin/osaat` writes a plist that points at the resolved path.
- The unit file is regenerated each run — re-running with different
  flags safely overwrites the old plist or service+timer pair.
- Uninstall is idempotent. Running it twice is a no-op the second time.
