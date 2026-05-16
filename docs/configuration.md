# Configuration

`osaat` is configured three ways, in increasing order of precedence:

1. Defaults baked into the binary (incl. OS-aware paths).
2. A named profile loaded with `--profile <name>`.
3. CLI flags on the command line.

When stdin is a TTY and no scan-shaping flags are passed, the
interactive wizard runs on top of all of the above and lets the user
adjust the resolved values before the scan starts.

## Default paths

| Purpose | Path |
|---|---|
| Generated outputs | `<Documents>/osaat/<YYYY-MM-DD>/` |
| Config directory | `$HOME/.config/osaat/` |
| Named profiles | `$HOME/.config/osaat/profiles/<name>.toml` |
| Daily logs | `$HOME/.config/osaat/logs/osaat-<YYYY-MM-DD>.log` |

`<Documents>` resolves to:

- macOS, Windows: `$HOME/Documents` (or `%USERPROFILE%\Documents`).
- Linux / BSD: `$XDG_DOCUMENTS_DIR`, else `$HOME/Documents`.

The config directory is `$HOME/.config/osaat/` on every platform — the
tool keeps a uniform layout rather than following per-OS conventions,
so a user moving between machines finds the same paths.

Profile files and the secrets file are written with mode `600`; the
logs directory and log files are created with mode `700` / `600`. The
public-facing report files (`report.json`, `report.pdf`, etc.) are
`644` so they can be shared or version-controlled if desired.

## Profiles

Profiles live at `~/.config/osaat/profiles/<name>.toml`:

```toml
schema = "osaat.profile/v1"
os = ["macos"]
formats = ["pdf", "markdown", "txt", "json"]
out = "~/Documents/osaat/{date}"

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

`{date}` in `out` expands to `YYYY-MM-DD` at scan time so a daily
profile produces distinct output directories.

Profile values only override flags the user did NOT explicitly pass on
the command line — explicit flags always win.

## Saving a profile

The wizard's last page asks for a profile name. Enter a name (e.g.
`imani-mbp`) and the answers are persisted to
`~/.config/osaat/profiles/imani-mbp.toml` automatically. Replay any
time with:

```sh
osaat scan --profile imani-mbp
```

## Logs

The daily log file at `~/.config/osaat/logs/osaat-<YYYY-MM-DD>.log`
captures:

- Scan start / completion timestamps and durations
- Record counts
- Collector warnings (e.g. unsigned binaries, plist parse skips)
- Reporter and secrets-write events

The logger scrubs every record before it hits disk:

- `$HOME` is replaced with `~` in every string attribute and message.
- Hostname-shaped attribute keys (`hostname`, `host`, `machine`,
  `node`) are replaced with `[redacted]`.
- License keys and secret values are never passed to the logger by any
  code path.

`osaat` does not send telemetry over the network. Nothing leaves the
machine unless you explicitly copy a file off it.

## Environment variables

| Variable | Effect |
|---|---|
| `XDG_DOCUMENTS_DIR` | On Linux / BSD, override the auto-detected Documents directory. |
| `NO_COLOR` | Disable styled output. |
