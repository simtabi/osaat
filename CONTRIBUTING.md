# Contributing

Thanks for your interest in `osaat`. This document is the
short version of how to land a change.

## Ground rules

- Be kind. The [Code of Conduct](CODE_OF_CONDUCT.md) applies everywhere.
- Security issues go to <opensource@simtabi.com>, not the public issue
  tracker. See [SECURITY.md](SECURITY.md).

## Getting set up

```sh
git clone https://github.com/simtabi/osaat.git
cd osaat
make test
```

You need:

- Go 1.22 or newer
- `make`
- `golangci-lint` for `make lint` (optional for development; required in CI)
- `goreleaser` for `make snapshot` (optional)

## Coding conventions

- Standard Go: `gofmt`, `go vet`, `golangci-lint run` all clean.
- One package per directory; package name matches the directory name.
- Errors are wrapped with `fmt.Errorf("doing X: %w", err)`. No `errors.New`
  with dynamic data — use `Errorf`.
- Tests live next to the code they test (`foo.go` + `foo_test.go`).
- Table-driven tests using `stretchr/testify` are the default style.
- Public symbols have a comment that starts with the symbol name.
- Avoid global state. Pass `context.Context` as the first arg of any
  function that can block or be cancelled.
- Shell-outs go through a single helper (`runCmd`) so they're injectable
  in tests.

## Commit messages

Imperative subject ≤ 72 characters. Body explains the **why**, not the
**what**. No emoji. No `Co-Authored-By:` trailers unless explicitly
requested.

```
Add macOS app store detection via mas list

The system_profiler XML output marks App Store apps as "Mac App Store"
but doesn't expose the App Store ID. Cross-reference `mas list` so the
reinstall command can be `mas install <id>` rather than a manual store
search.
```

## Pull requests

- One topic per PR. Mixed PRs get reverted.
- All checks green before requesting review.
- Update `CHANGELOG.md` under `## [Unreleased]` for user-visible changes.
- Update or add documentation under `docs/` when behavior changes.
- Tests for new behavior, regression tests for fixes.

## Reporting bugs

Use the bug-report issue template. Include:

- `osaat version` output
- OS and version (`sw_vers` on macOS, `lsb_release -a` on Linux)
- The command you ran
- What you expected to happen
- What happened instead
- A minimal reproduction if possible

## Suggesting features

Use the feature-request issue template. Frame the request in terms of the
problem you're trying to solve — implementation is part of the
discussion.
