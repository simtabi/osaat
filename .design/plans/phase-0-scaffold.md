# Phase 0 — scaffold

**Goal:** project renamed, Go module initialized, full Simtabi OSS tree
in place, CI/release workflows runnable, Cobra subcommand skeleton
compiles and prints `--help`. No audit logic yet.

## Tasks

### Rename
- [ ] Confirm `macos-backup-script/` has only `.design/`, `.claude/`,
      and `.DS_Store` (the empty case we already know).
- [ ] `mv` the folder to `osaat/`. Update CLAUDE.md
      references if any.

### Git
- [ ] `git init` inside `osaat/`.
- [ ] `git config user.email "19682005+imanimanyara@users.noreply.github.com"`
- [ ] `git config user.name "Imani Manyara"`
- [ ] First commit will be `Initial release` once everything below is
      in place (per Simtabi CLAUDE.md).

### Go module
- [ ] `go mod init github.com/simtabi/osaat`
- [ ] `go 1.22` directive in `go.mod`.
- [ ] Add deps (don't yet wire them — just `go get`):
      `cobra`, `huh`, `bubbletea`, `lipgloss`, `progressbar/v3`,
      `howett.net/plist`, `gopsutil/v4`, `filippo.io/age`,
      `pelletier/go-toml/v2`, `stretchr/testify`.

### Simtabi OSS scaffold (root)
- [ ] `LICENSE` — MIT, `Copyright (c) 2026 Simtabi LLC`.
      Fetch via `claude-config fetch` if available, else `curl` from
      a canonical source. **Disk-to-disk** per global CLAUDE.md.
- [ ] `README.md` — tagline, install (`brew install simtabi/tap/osaat`
      + `go install`), pointers into `docs/`.
- [ ] `CHANGELOG.md` — Keep-a-Changelog skeleton, `## [Unreleased]`.
- [ ] `CONTRIBUTING.md` — code style, test layout, PR expectations.
- [ ] `SECURITY.md` — disclosure address `opensource@simtabi.com`.
- [ ] `CODE_OF_CONDUCT.md` — Contributor Covenant 2.1 (fetch
      disk-to-disk; do not stream the body).
- [ ] `.editorconfig` — UTF-8, LF, 4-space Go.
- [ ] `.gitignore` — Go (`bin/`, `dist/`, `*.test`, `*.out`,
      `vendor/`), plus `.DS_Store`, `.idea/`, `.vscode/`,
      `secrets.json`, `*.age`, `*.tar.age`.

### `.github/`
- [ ] `.github/dependabot.yml` — `gomod` + `github-actions`,
      weekly Monday 06:00 America/New_York.
- [ ] `.github/workflows/ci.yml`:
      - matrix: Go `1.22`, `1.23`
      - steps: `go mod tidy`, `go vet`, `golangci-lint run`,
        `go test ./...`
      - cache `~/.cache/go-build` + `~/go/pkg/mod`
- [ ] `.github/workflows/release.yml`:
      - trigger: tag `v*`
      - uses `goreleaser/goreleaser-action`
      - publishes binaries + Homebrew tap update
      - requires `tap` GitHub Environment + `TAP_GITHUB_TOKEN` secret
        (per Simtabi shipping-checklist)
- [ ] `.github/ISSUE_TEMPLATE/bug_report.yml`,
      `feature_request.yml`, `config.yml` (UPPERCASE folder name per
      GitHub conventions).
- [ ] `.github/PULL_REQUEST_TEMPLATE.md`.

### `.goreleaser.yaml`
- [ ] Builds: macOS arm64+amd64, Linux amd64+arm64.
- [ ] Archives: tar.gz with the binary + LICENSE + README.
- [ ] Checksums.
- [ ] Homebrew tap publish to `simtabi/homebrew-tap` (deferred to
      Phase 5; placeholder section commented out).

### `docs/`
Empty bodies are fine for now — just the structure with frontmatter and
one paragraph each. They'll be filled out as the relevant phase lands.
- [ ] `docs/README.md` — index linking to each doc.
- [ ] `docs/installation.md`
- [ ] `docs/configuration.md`
- [ ] `docs/architecture.md` (copy from `.design/architecture.md` and
      simplify for end-users; the design doc keeps the implementation
      detail).
- [ ] `docs/release.md`
- [ ] `docs/shipping-checklist.md` (copy from
      `.design/backup-restore-checklist.md`).
- [ ] `docs/tools/scan.md`
- [ ] `docs/tools/diff.md`
- [ ] `docs/tools/restore-help.md`
- [ ] `docs/tools/install-schedule.md`
- [ ] `docs/tools/backup.md`

### Cobra skeleton
- [ ] `cmd/osaat/main.go` — `rootCmd` with persistent flags
      (`--out`, `--profile`, `--non-interactive`, `--verbose`).
- [ ] `cmd/osaat/scan.go` — stub: prints "scan stub", parses flags.
- [ ] `cmd/osaat/diff.go` — stub.
- [ ] `cmd/osaat/restore_help.go` — stub.
- [ ] `cmd/osaat/install_schedule.go` — stub.
- [ ] `cmd/osaat/backup.go` — stub.
- [ ] `cmd/osaat/version.go` — prints `internal/version.Version`.
- [ ] `internal/version/version.go` — `const Version = "0.0.0-dev"`;
      will be overridden by `-ldflags` from goreleaser.

### Tests
- [ ] `cmd/osaat/main_test.go` — `TestRootHelp` runs `osaat --help`
      and asserts each subcommand appears.
- [ ] `Makefile` — `make test`, `make lint`, `make build`,
      `make snapshot` (calls `goreleaser release --snapshot --clean`).

### Memory + housekeeping
- [ ] Create `~/.claude/projects/.../memory/MEMORY.md` index for this
      project (or update if it exists). Per global CLAUDE.md memory
      conventions.
- [ ] Run `make lint` + `make test` and capture output.
- [ ] Run `go build ./...` and capture.

### Definition of done

```bash
$ osaat --help                # works, lists subcommands
$ osaat scan --help           # works, shows flags
$ osaat version               # prints 0.0.0-dev
$ make test && make lint      # both green
$ make snapshot               # produces ./dist/ binaries for darwin + linux
```

No `git push`. No tag. Local-only commit on `main`.

## Non-goals for Phase 0

- Any audit logic. `scan` prints "scan stub".
- Wizard. Even if cobra is wired, the wizard is empty.
- Reporters. They don't exist yet.
