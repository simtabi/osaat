# Phase 5 — release v0.1.0

**Goal:** the project is public on GitHub, the Homebrew tap formula
works, and a tagged release ships binaries for macOS arm64/amd64 and
Linux amd64/arm64.

This is the **only phase that touches remote state.** Every action that
modifies GitHub, the Homebrew tap, or any external system requires
explicit "yes, do this" from the user (per global CLAUDE.md). The
checklist below is what the user authorizes step by step — `osaat` does
not run it for them.

## First-release D-list (mirrors Simtabi CLAUDE.md)

### 0. Pre-flight (local, no remote action)
- [ ] `make test && make lint && make snapshot` all green.
- [ ] `CHANGELOG.md` rolled: `## [Unreleased]` → `## [0.1.0] - 2026-MM-DD`.
- [ ] `docs/` reviewed — every linked file exists, internal links
      resolve, no stale TODOs in published docs.
- [ ] `README.md` install instructions reference the (not-yet-pushed)
      tag.
- [ ] Spot-check generated `dist/` binaries:
      ```
      ./dist/osaat-darwin-arm64/osaat version
      ./dist/osaat-darwin-amd64/osaat version
      ./dist/osaat-linux-amd64/osaat version
      ./dist/osaat-linux-arm64/osaat version
      ```
- [ ] Verify external URLs in metadata with `curl -sSL --max-time 5`
      per global CLAUDE.md "Verification before pinning" rule.

### 1. Make repo public on GitHub
- [ ] User authorizes; user runs:
      ```
      gh repo create simtabi/osaat \
          --public \
          --description "Audit and back up installed applications on macOS, Linux, and Unix." \
          --homepage "https://opensource.simtabi.com/products/osaat" \
          --source=. \
          --remote=origin \
          --push
      ```
- [ ] Topics: `oss`, `golang`, `cli`, `macos`, `linux`, `audit`,
      `backup`, `migration`.
- [ ] Enable Issues + Discussions.

### 2. Homebrew tap
- [ ] Repo `simtabi/homebrew-tap` exists (one-time org setup).
- [ ] Add `TAP_GITHUB_TOKEN` secret to `osaat`'s
      Actions secrets.
- [ ] Configure `.goreleaser.yaml` `brews:` block to point at the tap.

### 3. Cut the tag
- [ ] User authorizes (per global CLAUDE.md, every push and every tag
      requires explicit "yes").
- [ ] User runs:
      ```
      git tag -a v0.1.0 -m "Initial public release"
      git push origin v0.1.0
      ```
- [ ] GitHub Actions runs `release.yml`:
      - `goreleaser release` builds + publishes binaries
      - Homebrew tap PR opens automatically
      - Release notes generated from `CHANGELOG.md`

### 4. Verify release
- [ ] `https://github.com/simtabi/osaat/releases/tag/v0.1.0`
      shows 4 archives + checksums.
- [ ] `brew install simtabi/tap/osaat` works on a clean Mac.
- [ ] `osaat scan` produces a clean audit on that Mac.
- [ ] Run `osaat scan` and inspect output one more time.

### 5. Announce
- [ ] Post to whichever channels Simtabi uses for OSS announcements.
- [ ] Mark v0.1.0 in the project memory under
      `~/.claude/projects/.../memory/`.

## What "done" means

- `brew install simtabi/tap/osaat` → working binary on macOS.
- `go install github.com/simtabi/osaat/cmd/osaat@v0.1.0` →
  working binary on any Go-supported platform.
- Direct binary download from the GitHub Release works.
- A new user can read the README + `docs/installation.md` and have
  `osaat scan` running in under 5 minutes.

## What v0.1.0 doesn't include

(Documented in `CHANGELOG.md` under "Known limitations" so future
issues don't get filed against missing features.)

- Windows support.
- GUI.
- Cloud-sync destinations.
- License extraction on Linux.
- Vendor-URL enrichment via remote APIs (e.g. App Store metadata
  lookup). All sources are local-only.

## What lands in v0.2+

- Windows collector.
- More license-key heuristics on Linux.
- Optional vendor-URL enrichment via the App Store API.
- Web UI for browsing audit history (separate repo).
