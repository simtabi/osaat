# Architecture

**Status:** skeleton — full body in Phase 1.
Implementation-level detail lives in
[`.design/architecture.md`](../.design/architecture.md).

`osaat` is structured as a thin CLI on top of a small set of cleanly
separated packages:

```
cmd/osaat        — Cobra entrypoint, flag parsing
internal/audit   — AppRecord model, orchestrator
internal/collectors/{macos,linux,unix}  — OS-specific data collection
internal/licenses  — three license-extraction modes
internal/reporters — JSON / CSV / Markdown / HTML output
internal/restore   — Brewfile, mas list, manifest, encrypted archive
internal/wizard    — huh-based interactive form
internal/profiles  — TOML save/load
internal/schedule  — launchd + systemd unit generators
internal/secrets   — secrets.json schema + age encryption
internal/version   — version string (set at build time)
```

## Boundaries

- `internal/reporters` never imports collectors. It knows only the
  canonical `AppRecord` shape.
- `internal/collectors/*` never import each other. Shared helpers live in
  `internal/collectors/collector.go`.
- `cmd/osaat` is the only package allowed to read CLI flags. Everything
  below it takes options structs.
- The wizard produces a populated options struct identical to what the
  flag mode would build, so both code paths are interchangeable.

## Data flow

1. The CLI parses flags, or runs the wizard if stdin is a TTY and no
   flags were passed.
2. The orchestrator picks collectors based on the requested OS.
3. Collectors return `[]AppRecord`, which the orchestrator merges and
   deduplicates.
4. The selected license scanner enriches records and produces a
   `SecretsFile`.
5. Reporters render the records in each requested format.
6. The restore generator produces a Brewfile, package-manager install
   lists, and `RESTORE.md`.
7. Optionally the whole output directory is wrapped in a `tar.age`
   bundle.

## Extending

- **New collector:** new file under `internal/collectors/<os>/`,
  implement `Collector`, register in the orchestrator.
- **New reporter:** new file under `internal/reporters/`, implement
  `Reporter`, register by format name.
- **New license scanner:** new file under `internal/licenses/`,
  implement `Scanner`, register by mode name.

The deeper rationale for each of these choices is in
[`.design/architecture.md`](../.design/architecture.md) and
[`.design/library-stack.md`](../.design/library-stack.md).
