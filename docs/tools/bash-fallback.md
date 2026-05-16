# `bash-fallback.sh`

A zero-dependency Bash script that audits installed macOS applications
and writes a JSON report compatible with the Go binary's
`osaat.report/v1` schema. Use it when you need an inventory *before*
bootstrapping Go, Homebrew, or `osaat` itself — for example, on a
brand-new Mac that's about to receive a Migration Assistant transfer.

## What it does

- Walks `/Applications`, `~/Applications`, `/System/Applications`,
  and their `Utilities/` subdirs.
- For each `.app` bundle, reads `kMDItemDisplayName`,
  `kMDItemCFBundleIdentifier`, `kMDItemVersion`, and
  `kMDItemWhereFroms` via `mdls`.
- Captures install date (file mtime), size (via `du -sk`), and
  signing status (via `codesign`).
- Emits a JSON document with the same envelope shape as
  `osaat scan --format json`.

## What it does not do

- License-key scanning (use the Go binary's `--license-mode`).
- `age` encryption.
- PDF / Markdown / TXT / CSV / HTML reporters.
- `mas` / `brew` cross-reference (the `source` field is `system` for
  apps under `/System/Applications/` and `unknown` for everything else).
- Linux or Unix collectors.

## Compatibility

- Bash 3.2 — the default `/bin/bash` on macOS, unchanged since
  10.4. The script avoids Bash 4+ features (no associative arrays,
  no `${var^^}`, no `mapfile`).
- POSIX tools (`awk`, `sed`, `grep`, `stat`, `du`).
- macOS-specific tools that ship with the base system: `mdls`,
  `codesign`, `hostname`, `uname`, `xattr`.

No Go, Homebrew, or third-party Python required.

## Usage

```sh
# Write report.json to stdout
./scripts/bash-fallback.sh > report.json

# Write to a specific directory
./scripts/bash-fallback.sh --out ~/backup/

# Help
./scripts/bash-fallback.sh --help
```

A 300-app scan takes roughly 2 minutes on Apple Silicon (slower than
the Go binary's ~80 seconds because each per-app field reads through
a separate subprocess call to `mdls` / `codesign` / `stat`).

## Diffing against the Go binary's output

The schema is identical, so `osaat diff` consumes both interchangeably:

```sh
./scripts/bash-fallback.sh > old.json
osaat scan --format json --out ./new
osaat diff old.json ./new/report.json
```

This is useful as a sanity check after installing `osaat` proper — you
can confirm the Go collector reports the same record set the
fallback did.

## When to prefer the Go binary

The fallback exists for the cold-start recovery scenario. For every
other case (license scanning, encrypted secrets, restoration
manifest, scheduled audits, Linux/Unix support, diffing across
machines), use the Go binary.
