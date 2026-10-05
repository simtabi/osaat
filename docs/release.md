# Release process

How a tagged release is cut, what it publishes, and how the package
managers are updated.

## Versioning

Semantic versioning. Pre-1.0 means minor version bumps may include
breaking changes; patch releases are bug fixes only.

## Cutting a release

1. Ensure `main` is green: `make test && make lint && make snapshot`.
2. Roll `CHANGELOG.md`: replace `## [Unreleased]` with
   `## [X.Y.Z] - YYYY-MM-DD` and create a fresh `## [Unreleased]`
   section above it.
3. Update the compare link at the bottom of `CHANGELOG.md`.
4. Land the roll through a pull request; never push to `main`.
5. Tag the merge commit on `main` and push the tag:
   `git tag -a vX.Y.Z -m "Release vX.Y.Z" && git push origin vX.Y.Z`.

The module lives at the repository root (`github.com/simtabi/osaat`), so the
one `vX.Y.Z` tag is all `go install` needs; there is no `src/` alias tag.

The release workflow (`.github/workflows/release.yml`) triggers on a
`vX.Y.Z` tag and runs GoReleaser. It publishes, from that one tag:

- Bare binaries `osaat_<os>_<arch>` and archives `osaat_<os>_<arch>.tar.gz`
  (`.zip` on Windows) for macOS (amd64, arm64, universal), Linux (amd64,
  arm64, 386, armv7), Windows (amd64, arm64, 386) and FreeBSD (amd64, arm64,
  386). Names are version-less and use `macos`, not `darwin`.
- Linux `.deb`, `.rpm` and `.apk` packages.
- A reproducible source tarball, alongside GitHub's own source links.
- One SPDX SBOM per archive.
- `checksums.txt` (SHA-256, bare file names) over every asset.
- A signed build-provenance attestation for every asset in `checksums.txt`.
- The GitHub Release, whose body is the tagged version's `CHANGELOG.md`
  section (`scripts/extract-changelog.sh`).
- A pull request against `simtabi/homebrew-tap` updating `Casks/osaat.rb`.
  The tap only accepts pull requests, so `brew install simtabi/tap/osaat`
  serves the new version once that pull request is merged.

Scoop and Winget manifests are built but not published until the repository
variable `SKIP_WINDOWS_PKGS` is set to `false`; Winget also needs a
`simtabi/winget-pkgs` fork.

## Credentials

No long-lived token is stored for the release. The workflow mints a
per-run token from the Refresh Bot GitHub App, limited to
`simtabi/homebrew-tap` and `simtabi/scoop-bucket` with contents and pull
request write access, from the org-level `REFRESH_APP_CLIENT_ID` variable and
`REFRESH_APP_PRIVATE_KEY` secret. The release itself uses `GITHUB_TOKEN`.

## Trusted publishing (PyPI / npm / etc.)

Not applicable — `osaat` is a Go binary distributed via GitHub Releases
and the Homebrew tap. No package-registry publishing keys are needed.

## Rollback

If a release ships with a regression:

1. Cut a patch release (`vX.Y.Z+1`) with the fix.
2. Mark the broken release as "broken — see vX.Y.Z+1" in its GitHub
   release notes.
3. Open a GitHub Security Advisory if the regression is security-relevant.

Do **not** delete or rewrite a published tag. Releases are append-only.
