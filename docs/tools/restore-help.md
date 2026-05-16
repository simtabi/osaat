# `osaat restore-help`

**Status:** skeleton — full body in Phase 2.

Emit a per-app manual-install checklist from a previously generated audit
report. Useful after `brew bundle` and `mas install` cover the obvious
cases, when you still need to track down the long tail.

## Usage

```sh
osaat restore-help --from <report.json> [--out RESTORE.md]
```

## Output

A Markdown document with one section per detected app whose reinstall
command is not covered by `brew bundle` or `mas install`. Each section
contains:

- Vendor URL
- Original download URL (from `kMDItemWhereFroms` / quarantine xattr)
- Pointer to the relevant entry in `secrets.json` (when license-mode
  was something other than `none`)
- The reinstall command if known

## Examples

```sh
osaat restore-help --from ~/backup/osaat-2026-05-16/report.json \
    --out ~/backup/osaat-2026-05-16/RESTORE.md
```
