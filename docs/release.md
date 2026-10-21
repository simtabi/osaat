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
- An update to `Casks/osaat.rb` in `simtabi/homebrew-tap`. With the deploy
  key it is pushed straight to `main` (deploy keys are the tap ruleset's one
  bypass actor), so `brew install simtabi/tap/osaat` serves the new version at
  once. Without the key it falls back to a pull request, served once merged.

Scoop and Winget manifests are built but not published by default. Scoop publishes when the
repository variable `SKIP_SCOOP` is `false` (unset, it follows `SKIP_WINDOWS_PKGS`); Winget publishes
when `SKIP_WINDOWS_PKGS` is `false` and also needs a `simtabi/winget-pkgs` fork.

## Credentials

The tap and bucket are written with the write deploy keys `HOMEBREW_TAP_DEPLOY_KEY` (on `simtabi/homebrew-tap`) and `SCOOP_BUCKET_DEPLOY_KEY` (on `simtabi/scoop-bucket`), each a repository secret that reaches its one repository and nothing else. While
both are set, no app token is minted. Without them, the workflow falls back to a
per-run Refresh Bot token limited to those two repositories (contents and pull
request write), from `REFRESH_APP_CLIENT_ID` and `REFRESH_APP_PRIVATE_KEY`. The
release itself uses `GITHUB_TOKEN`.

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
