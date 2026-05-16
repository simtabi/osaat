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

## Quick start

```sh
# Interactive wizard
osaat scan

# Headless
osaat scan --os macos --format json,markdown --out ./out/
```

The wizard opens automatically when stdin is a TTY and no flags are
passed. At the end of the wizard you get a non-interactive command you can
run next time.

## What gets captured

For every detected app: name, author, vendor URL, installation source
(App Store / Homebrew / pkg / DMG / direct download / system / sandbox /
unknown), original download URL, version, install date, last-used date,
size on disk, signing status, Apple Silicon compatibility (macOS), and a
reinstall command.

License keys, when detectable, go to a separate `secrets.json` —
unredacted and grouped by category — never to the audit report. Optional
`age` encryption is supported.

## Documentation

- [Installation](docs/installation.md)
- [Configuration](docs/configuration.md)
- [Architecture](docs/architecture.md)
- [Release process](docs/release.md)
- [Migration / shipping checklist](docs/shipping-checklist.md)
- Per-command docs: [scan](docs/tools/scan.md) · [diff](docs/tools/diff.md) · [restore-help](docs/tools/restore-help.md) · [install-schedule](docs/tools/install-schedule.md) · [backup](docs/tools/backup.md)

## License

MIT — see [LICENSE](LICENSE). Copyright © 2026 Simtabi LLC.
