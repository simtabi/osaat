# Installation

## Homebrew (macOS, Linux)

```sh
brew install simtabi/tap/osaat
```

The tap is configured automatically on first install.

## Go

```sh
go install github.com/simtabi/osaat/cmd/osaat@latest
```

Requires Go 1.24 or newer.

## Direct binary

Download the archive for your platform from the
[latest release](https://github.com/simtabi/osaat/releases/latest).
Every release ships these architectures:

| OS | Architectures |
|---|---|
| macOS | `amd64` (Intel), `arm64` (Apple Silicon) |
| Linux | `amd64`, `arm64`, `386` (32-bit x86), `armv7` (32-bit ARM) |
| Windows | `amd64`, `arm64`, `386` |
| FreeBSD | `amd64`, `arm64`, `386` |

Archive names follow `osaat_<version>_<os>_<arch>.tar.gz` (or `.zip` for
Windows). ARM 32-bit builds carry a `v7` suffix
(`..._linux_armv7.tar.gz`).

Extract and move the `osaat` binary onto your `PATH`.

## Zero-dependency Bash fallback (macOS only)

If you need an audit *before* installing Go or Homebrew (e.g. on a
brand-new Mac you're about to migrate to), use the Bash fallback
script. It produces a JSON report compatible with the Go binary's
schema.

```sh
curl -fsSL -o bash-fallback.sh \
  https://raw.githubusercontent.com/simtabi/osaat/main/scripts/bash-fallback.sh
chmod +x bash-fallback.sh
./bash-fallback.sh --out ~/backup/pre-migration/
```

See [`docs/tools/bash-fallback.md`](tools/bash-fallback.md) for
details on its capabilities and limitations.

## Source

```sh
git clone https://github.com/simtabi/osaat.git
cd osaat
make build
./bin/osaat version
```

## Verify the install

```sh
osaat version
osaat --help
```

## Uninstall

```sh
# Homebrew
brew uninstall osaat
brew untap simtabi/tap

# Go
rm $(which osaat)

# Direct binary
rm /usr/local/bin/osaat   # or wherever you put it
```

User configuration at `~/.config/osaat/` is left in place by uninstall —
remove it manually if you don't want to keep profiles.
