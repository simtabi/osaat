# Configuration

**Status:** skeleton — full body in Phase 2 (profiles).

`osaat` is configured three ways, in increasing order of precedence:

1. Defaults baked into the binary.
2. A named profile loaded with `--profile <name>`.
3. CLI flags on the command line.

## Profiles

Profiles live at `~/.config/osaat/profiles/<name>.toml`
(`$XDG_CONFIG_HOME/osaat/profiles/<name>.toml` if `XDG_CONFIG_HOME` is
set).

```toml
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

`{date}` in `out` expands to `YYYY-MM-DD` at scan time so a daily profile
produces distinct output directories.

## Environment variables

| Variable | Effect |
|---|---|
| `XDG_CONFIG_HOME` | Override the config directory. |
| `OSAAT_PROFILE` | Default profile when `--profile` is not passed. |
| `OSAAT_AGE_KEY` | Path to your age private key (overrides `--age-key`). |
| `NO_COLOR` | Disable styled output. |

## Generating a profile

The wizard offers to save a profile on the last page. Headless equivalent:

```sh
osaat profile save imani-mbp \
    --os macos \
    --format json,markdown \
    --license-mode best-effort \
    --insights forgotten,apple-silicon
```

(`profile save` is documented in [tools/scan.md](tools/scan.md) once
Phase 2 lands.)
