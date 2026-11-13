# Release process

How a tagged release is cut and what it publishes.

Binaries ship on the GitHub release only: no Homebrew, Scoop, Winget, `go install`,
image or any other channel. A release builds in one of two ways:

```sh
git push origin vX.Y.Z                                            # builds and publishes vX.Y.Z
gh workflow run release.yml --repo simtabi/osaat -f tag=vX.Y.Z    # the same, for a tag already pushed
```

Because a pushed `vX.Y.Z` tag publishes, push one only for final code. A tag is
final: it is never moved, deleted or reused. Test builds use `vX.Y.Z-rc.N`, which
publishes a GitHub pre-release.

## Versioning

Semantic versioning. Pre-1.0 means minor version bumps may include
breaking changes; patch releases are bug fixes only.

## Cutting a release

1. Ensure `main` is green: `make test && make lint && make cross`.
   `make snapshot` additionally runs GoReleaser locally without publishing
   (output in `.cache/goreleaser/`).
2. Roll `CHANGELOG.md`: replace `## [Unreleased]` with
   `## [X.Y.Z] - YYYY-MM-DD` and create a fresh `## [Unreleased]`
   section above it.
3. Update the compare link at the bottom of `CHANGELOG.md`.
4. Land the roll through a pull request; never push to `main`.
5. Tag the merge commit on `main` and push the tag. The push builds and
   publishes the release:

   ```sh
   git -C ~/src/osaat tag -a vX.Y.Z -m "Release vX.Y.Z"
   git -C ~/src/osaat push origin vX.Y.Z
   ```

6. To build a release for a tag that is already on GitHub (a first release of
   an older tag, or a re-run after a failed one), request it instead:

   ```sh
   gh workflow run release.yml --repo simtabi/osaat -f tag=vX.Y.Z
   ```

   The workflow refuses anything that is not a `vX.Y.Z` (or `vX.Y.Z-pre`) tag
   already on GitHub, and a final version with no `## [X.Y.Z]` section in
   `CHANGELOG.md`, before a single artifact is built.

The Go module lives in `src/` (`github.com/simtabi/osaat/src`), and Go resolves
a module in a subdirectory only through a tag carrying that prefix, so a release
needs `src/vX.Y.Z` on the same commit as `vX.Y.Z`. The release workflow creates
it before GoReleaser runs (it matches no release trigger, so it starts no second run); never push it by hand, and never move it once it
exists.

Local builds never touch release assets: `make cross` writes
`build/<os>/osaat_<os>_<arch>[.exe]` and `build/checksums.txt`, `make dist` adds
`build/archives/`, and GoReleaser writes its own scratch output to
`.cache/goreleaser/`. None of them publishes.

The release workflow (`.github/workflows/release.yml`) is a thin caller of the
shared `simtabi/.github` `go-release.yml`, pinned to a commit. It runs on those
two triggers only, never on a branch push, and runs GoReleaser
(`--config src/.goreleaser.yaml`, from the repository root) against the tag. It
refuses a GoReleaser config holding any other publishing block. It publishes,
on the GitHub release for that one tag:

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
  section, followed by a link to the full changelog. The shared workflow reads
  it back after publishing and fails if the body is shorter than the section.
  `src/scripts/extract-changelog.sh` prints the same section locally.

## Frozen channels

Until 2026-11-13 a release also updated `Casks/osaat.rb` in
`simtabi/homebrew-tap`, and built Scoop and Winget manifests behind the
`SKIP_SCOOP` / `SKIP_WINDOWS_PKGS` variables. Those channels are frozen: the
0.1.1 cask stays in the tap and keeps working, but no release updates it, and no
Scoop or Winget manifest is published. The GoReleaser blocks that wrote them are
gone, and `TestGoReleaserPublishesNowhereElse` fails if one comes back.

## Credentials

None beyond the run's own `GITHUB_TOKEN`, which creates the release, uploads
the assets and pushes `src/vX.Y.Z`. The tap and bucket deploy keys
(`HOMEBREW_TAP_DEPLOY_KEY`, `SCOOP_BUCKET_DEPLOY_KEY`) and the Refresh Bot token
are no longer read by this repository's release.

## Trusted publishing (PyPI / npm / etc.)

Not applicable — `osaat` is a Go binary distributed on GitHub Releases only.
No package-registry publishing keys are needed.

## Rollback

If a release ships with a regression:

1. Cut a patch release (`vX.Y.Z+1`) with the fix.
2. Mark the broken release as "broken — see vX.Y.Z+1" in its GitHub
   release notes.
3. Open a GitHub Security Advisory if the regression is security-relevant.

Do **not** delete or rewrite a published tag. Releases are append-only.
