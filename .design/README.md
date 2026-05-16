# Design — `osaat`

This directory holds the design and architecture notes that drive the build.
End-user documentation will live at `docs/` once the project is scaffolded in
Phase 0; that tree is separate from this one.

## Index

| Document | Purpose |
|---|---|
| [prompt.md](prompt.md) | Canonical product prompt — what we're building and why |
| [architecture.md](architecture.md) | System architecture, modules, data flow |
| [data-model.md](data-model.md) | `AppRecord` and `secrets.json` schemas |
| [wizard-ux.md](wizard-ux.md) | Interactive wizard flow |
| [library-stack.md](library-stack.md) | Go library choices, including how `bubbletea` factors in |
| [backup-restore-checklist.md](backup-restore-checklist.md) | Migration runbook (becomes `docs/shipping-checklist.md` post-scaffold) |
| [plans/README.md](plans/README.md) | Phased build plan index |

## Quick facts

- Project name: **`osaat`** (folder will be renamed from `macos-backup-script` in Phase 0).
- Binary: **`osaat`**.
- Language: Go 1.22+, single static binary per platform.
- Owner: Simtabi LLC. License: MIT.
- Canonical URLs:
  - Product: <https://opensource.simtabi.com/products/osaat>
  - Docs: <https://opensource.simtabi.com/documentation/osaat>
  - Repo: <https://github.com/simtabi/osaat>
  - Issues: <https://github.com/simtabi/osaat/issues>

## Conventions for this directory

- Filenames are lowercase-kebab.
- Cross-document links use relative paths so they resolve on GitHub.
- A ratified decision carries an absolute date; a tentative one is marked `**status:** draft`.
- These docs are revised as decisions evolve — they are working artifacts,
  not frozen specs. The phase plans under `plans/` are the authoritative
  checklist of what to actually build.
