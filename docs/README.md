# Documentation

End-user documentation for `osaat`. Project-internal design
notes live in [`.design/`](../.design/README.md) and are not shipped with
releases.

## Overview

| Document | What it covers |
|---|---|
| [installation.md](installation.md) | How to install on macOS, Linux, BSD |
| [configuration.md](configuration.md) | Config file format, environment variables, profiles |
| [architecture.md](architecture.md) | What the tool does and how it does it, at a level useful for contributors |
| [release.md](release.md) | How a release is cut and how the Homebrew tap is updated |
| [shipping-checklist.md](shipping-checklist.md) | Step-by-step migration runbook |

## Per-command reference

| Command | Doc |
|---|---|
| `osaat scan` | [tools/scan.md](tools/scan.md) |
| `osaat diff` | [tools/diff.md](tools/diff.md) |
| `osaat restore-help` | [tools/restore-help.md](tools/restore-help.md) |
| `osaat install-schedule` | [tools/install-schedule.md](tools/install-schedule.md) |
| `osaat backup` | [tools/backup.md](tools/backup.md) |

## Status

This documentation tree is created in Phase 0 with skeletons. Each
document gets its body in the phase that ships the feature it covers.
The header of each file records the current status.
