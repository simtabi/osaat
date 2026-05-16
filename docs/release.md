# Release process

**Status:** skeleton — full body in Phase 5.

## Versioning

Semantic versioning. Pre-1.0 means minor version bumps may include
breaking changes; patch releases are bug fixes only.

## Cutting a release

1. Ensure `main` is green: `make test && make lint && make snapshot`.
2. Roll `CHANGELOG.md`: replace `## [Unreleased]` with
   `## [X.Y.Z] - YYYY-MM-DD` and create a fresh `## [Unreleased]`
   section above it.
3. Update the compare link at the bottom of `CHANGELOG.md`.
4. Commit: `chore: release vX.Y.Z`.
5. Tag: `git tag -a vX.Y.Z -m "Release vX.Y.Z"`.
6. Push: `git push && git push --tags`.

The release workflow (`.github/workflows/release.yml`) triggers on the
tag and runs GoReleaser. Artifacts:

- Four binaries: darwin/{amd64,arm64} and linux/{amd64,arm64}
- `checksums.txt` (SHA256)
- A draft GitHub Release with changelog entries pulled from the
  conventional-commit prefixes since the last tag
- A Homebrew tap update pull request against `simtabi/homebrew-tap`

## First-release D-list

For `v0.1.0` only:

1. Make `simtabi/osaat` public on GitHub.
2. Create `simtabi/homebrew-tap` repo if it doesn't exist.
3. Mint a fine-grained PAT with `Contents: write` on the tap and add it
   as `TAP_GITHUB_TOKEN` in this repo's Actions secrets.
4. Uncomment the `brews:` block in `.goreleaser.yaml`.
5. Cut `v0.1.0`.

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
