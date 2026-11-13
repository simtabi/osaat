# Installation

Two ways, in this order: download the binary from the GitHub release, or build
from source. The GitHub release is the only channel: this page offers no
package-manager or `go install` path.

## Download a release

Each [GitHub release](https://github.com/simtabi/osaat/releases) carries bare
binaries `osaat_<os>_<arch>` (`.exe` on Windows; `macos`, not `darwin`), archives,
Linux `.deb`/`.rpm`/`.apk` packages, and `checksums.txt`. Download the one for
your platform and the checksums, verify, and put it on your `PATH`:

```sh
gh release download --repo simtabi/osaat --pattern 'osaat_macos_universal' --pattern checksums.txt
shasum -a 256 -c checksums.txt --ignore-missing
install -m 0755 osaat_macos_universal ~/.local/bin/osaat
```

Every asset in `checksums.txt` carries a build-provenance attestation:

```sh
gh attestation verify osaat_macos_universal --repo simtabi/osaat
```

## Build from source

You need Go 1.25 or newer (the version `src/go.mod` names), `git` and `make`.

```sh
git clone https://github.com/simtabi/osaat ~/src/osaat
make -C ~/src/osaat build      # -> ~/src/osaat/build/macos/osaat (build/linux/ on Linux)
install -m 0755 ~/src/osaat/build/macos/osaat ~/.local/bin/osaat
```

`make build` writes `build/<os>/osaat`, where `<os>` is `macos`, `linux`,
`freebsd` or `windows` (with `.exe`). `make install` does both steps, copying
the binary to `~/.local/bin` (set `BINDIR=` for another directory). Any
directory on `PATH` works in place of `~/.local/bin`.

The Go module lives in `src/` (`github.com/simtabi/osaat/src`). On Windows, or
anywhere without `make`, build with Go directly and put the binary on your
`PATH`:

```sh
go -C ~/src/osaat/src build -o ../build/windows/osaat.exe ./cmd/osaat
```

Use `../build/linux/osaat` (or `macos`, `freebsd`) on other systems, so the
binary lands in the same per-OS folder `make build` uses.

A version is a tag: check one out to build exactly that release.

```sh
git -C ~/src/osaat fetch --tags
git -C ~/src/osaat checkout v0.1.1
make -C ~/src/osaat build
```

### Updating

```sh
git -C ~/src/osaat pull --ff-only
make -C ~/src/osaat install
```

`make build` stamps the version from `git describe`, so `osaat version` reports
the tag or commit the binary was built from, and a build from a branch is still
identifiable.

### Frozen channels

0.1.1 was also published as a Homebrew cask in `simtabi/homebrew-tap`. That cask
stays where it is and an existing install keeps working, but it is frozen: no
release updates it. Download from the GitHub release, or build from source, to
get anything newer.

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
