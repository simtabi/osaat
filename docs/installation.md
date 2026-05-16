# Installation

**Status:** skeleton — full body in Phase 5 (release).

## Homebrew (macOS, Linux)

```sh
brew install simtabi/tap/osaat
```

The tap is configured automatically on first install.

## Go

```sh
go install github.com/simtabi/osaat/cmd/osaat@latest
```

Requires Go 1.22 or newer.

## Direct binary

Download the archive for your platform from the
[latest release](https://github.com/simtabi/osaat/releases/latest):

- `osaat_<version>_darwin_arm64.tar.gz` — Apple Silicon Macs
- `osaat_<version>_darwin_amd64.tar.gz` — Intel Macs
- `osaat_<version>_linux_amd64.tar.gz` — Linux x86_64
- `osaat_<version>_linux_arm64.tar.gz` — Linux ARM64

Extract and move the `osaat` binary onto your `PATH`.

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
