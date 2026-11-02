# Installation

## Build from source

osaat is built from source. Binary releases are on hold until the maintainer
decides otherwise, so this page offers no download, package-manager or
`go install` path. You need Go 1.25 or newer (the version `go.mod` names),
`git` and `make`.

```sh
git clone https://github.com/simtabi/osaat ~/src/osaat
make -C ~/src/osaat build      # -> ~/src/osaat/bin/osaat
install -m 0755 ~/src/osaat/bin/osaat ~/.local/bin/osaat
```

Any directory on `PATH` works in place of `~/.local/bin`. On Windows, or anywhere
without `make`, build with Go directly and put the binary on your `PATH`:

```sh
go -C ~/src/osaat build -o bin/osaat ./cmd/osaat
```

A version is a tag: check one out to build exactly that release.

```sh
git -C ~/src/osaat fetch --tags
git -C ~/src/osaat checkout v0.1.1
make -C ~/src/osaat build
```

### Updating

```sh
git -C ~/src/osaat pull --ff-only
make -C ~/src/osaat build
install -m 0755 ~/src/osaat/bin/osaat ~/.local/bin/osaat
```

`make build` stamps the version from `git describe`, so `osaat version` reports
the tag or commit the binary was built from, and a build from a branch is still
identifiable.

### Earlier binary releases

0.1.1 was published with binaries and archives for macOS, Linux, Windows and
FreeBSD, Linux `.deb`/`.rpm`/`.apk` packages, checksums with a build-provenance
attestation, and a Homebrew cask in `simtabi/homebrew-tap`. Those stay where they
are, and an existing install keeps working, but none of those channels is updated
while releases are on hold: build from source to get anything newer.

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

## Verify the install

```sh
osaat version
osaat --help
```

## Uninstall

```sh
# A binary built from source
rm ~/.local/bin/osaat     # or wherever you installed it

# An earlier Homebrew install (0.1.1)
brew uninstall osaat
brew untap simtabi/tap
```

User configuration at `~/.config/osaat/` is left in place by uninstall —
remove it manually if you don't want to keep profiles.
