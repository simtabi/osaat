# Library stack

**Status:** ratified, 2026-05-16

The principle: **use existing libraries, don't rebuild**. Where a library
doesn't exist for a task (e.g. parsing `mas list` output), shell out via
`os/exec` and parse — no third-party wrapper needed for those.

## The stack

| Concern | Library | Version pin | Rationale |
|---|---|---|---|
| CLI framework | `github.com/spf13/cobra` | latest stable | Standard for Go CLIs (kubectl, gh, docker, hugo). Subcommands, flag parsing, shell completions. |
| Interactive forms | `github.com/charmbracelet/huh` | latest stable | Form-first API (Select, MultiSelect, Confirm, Input, Note). Built on bubbletea. |
| TUI runtime / custom screens | `github.com/charmbracelet/bubbletea` | latest stable | Powers `huh` (transitive); used directly for the live scan view and per-app review panel where forms aren't the right metaphor. |
| Styling | `github.com/charmbracelet/lipgloss` | latest stable | Transitive dep of huh; reused for ad-hoc colored output. |
| Progress bars | `github.com/schollz/progressbar/v3` | latest stable | Light, non-bubbletea-flavored; used for headless scans where the wizard isn't running. |
| macOS plist parsing | `howett.net/plist` | latest stable | The Go plist library. Reads `Info.plist` and `system_profiler -xml` output. Pure Go. |
| Cross-platform sys info | `github.com/shirou/gopsutil/v4` | v4 | Host facts, disk usage, process listings. psutil port. |
| File encryption | `filippo.io/age` | latest stable | Modern, simple, audited. Filippo Valsorda. Stdlib-style API. |
| TOML profiles | `github.com/pelletier/go-toml/v2` | v2 | Spec-compliant TOML reader/writer. |
| Test assertions | `github.com/stretchr/testify` | latest stable | Standard. |
| Lint | `golangci-lint` (CLI) | latest stable | Standard meta-linter; not a Go import. |
| Cross-platform release | `goreleaser` (CLI) | latest stable | Builds macOS+Linux binaries from one config; publishes to GitHub Releases + Homebrew tap. |

Stdlib for: `encoding/json`, `encoding/csv`, `encoding/xml`, `html/template`,
`archive/tar`, `os/exec`, `path/filepath`, `text/template`, `testing`,
`net/url`, `runtime` (GOOS detection).

## Why `huh` + `bubbletea` and not just one

- `huh` is the right tool for **forms** (multi-page wizard with Select,
  MultiSelect, Confirm, Input). It compiles down to a `bubbletea` program
  internally.
- `bubbletea` is the right tool for **custom screens** where a form isn't
  the metaphor — e.g. the live scan view with rolling counters, the
  per-app review where the user accepts/rejects each detected key, a
  side-by-side diff view.
- Sharing the runtime means one keymap/theme/event-loop. We don't pay the
  cost of two TUI frameworks.
- Both are by Charm (`charmbracelet/*`), actively maintained, MIT-licensed.

## Why `cobra` and not stdlib `flag`

- We have ≥6 subcommands (`scan`, `diff`, `restore-help`,
  `install-schedule`, `backup`, `version`). stdlib `flag` doesn't do
  subcommands without significant glue.
- Cobra gives us shell completions (`osaat completion zsh`) for free —
  worth it on its own.
- Cost: one mature dep that everything else in the Go CLI world already
  uses.

## Why `gopsutil` and not just `os/exec`

- Disk usage per path is non-trivially platform-specific. `du -sh` works
  but spawning N processes for N apps is slow on Linux.
- `gopsutil` gives us `disk.Usage(path)` cross-platform.
- Host facts (`host.Info()`, `host.HostID()`) populate the audit metadata
  block consistently.
- For installed-package listing, `gopsutil` does NOT help — we shell out
  to `dpkg-query`, `rpm`, `pacman`, etc. and parse output. That's fine;
  the alternative would be reading distro-specific databases directly,
  which is more brittle than the official CLIs.

## Why `filippo.io/age` and not `gpg`

- Pure Go (no system dep on `gpg` binary).
- Small API. We need `Encrypt(recipients, in, out)` and that's it.
- Modern crypto, no key-management baggage.
- We can still accept `gpg` recipients in the wizard later as a
  `age-plugin-gpg` use case; not needed for v0.1.

## Why `goreleaser` and not `go build`

- We ship 4+ binaries per release (macOS arm64, macOS amd64, Linux amd64,
  Linux arm64). Hand-rolling that in a workflow is ~80 lines of YAML.
  GoReleaser is one `.goreleaser.yaml`.
- It writes the Homebrew tap formula on every release automatically.
- It generates checksums + SBOMs + signed-release metadata.

## Versioning policy

- Pin minor versions in `go.mod` (`v0.4.0` form, not `v0.4`).
- Dependabot weekly on Mon 06:00 ET (per global CLAUDE.md).
- Re-run `go test` + `golangci-lint run` on every Dependabot PR before
  merge.

## Libraries explicitly NOT chosen

| Considered | Why not |
|---|---|
| `AlecAivazis/survey` | Solid alternative to `huh`. Less stylized; doesn't share runtime with `bubbletea`. Picked `huh` for ecosystem consistency. |
| `manifoldco/promptui` | Templating-heavy; lacks multi-select-with-validation. |
| `c-bata/go-prompt` | REPL-flavored; great for a shell, overkill for a wizard. |
| `BurntSushi/toml` | Mature but no longer the primary recommendation; `pelletier/go-toml/v2` is current. |
| `urfave/cli` | Comparable to cobra; cobra has the larger gravity well. |
| `viper` | Config layering we don't need for v0.1. Profiles in TOML + cobra flags suffice. |
| `textual` (Python) | Wrong language. Listed only to record that "full screen TUI app" was considered and rejected as overkill. |
