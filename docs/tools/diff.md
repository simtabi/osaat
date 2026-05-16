# `osaat diff`

**Status:** skeleton — full body in Phase 2.

Diff two audit reports.

## Usage

```sh
osaat diff <old.json> <new.json> [--format <text|json|markdown>]
```

Each report is matched record-by-record on `BundleID` (macOS) or `PkgID`
(Linux/Unix), falling back to `Name + Version` when neither is set.

## Categories

| Marker | Meaning |
|---|---|
| `+` | Present in the new report, missing from the old. |
| `-` | Present in the old report, missing from the new. |
| `~` | Present in both, with at least one differing field (version, source, signing, etc.). |

## Examples

Daily drift:

```sh
osaat diff ~/backup/osaat-2026-05-09/report.json ~/backup/osaat-2026-05-16/report.json
```

Pre/post migration check:

```sh
osaat diff ~/backup/old-mac/report.json ~/backup/new-mac/report.json
```

Machine-readable:

```sh
osaat diff --format json a.json b.json | jq '.changes[] | select(.kind == "removed")'
```

## Exit codes

| Code | Meaning |
|---|---|
| 0 | No differences. |
| 1 | Differences found. |
| 2 | Flag misuse or invalid input. |
