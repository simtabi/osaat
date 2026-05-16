# Canonical prompt

**Status:** ratified, 2026-05-16
**Version:** v3

## The prompt

> Build a Go CLI named `osaat` (project: `osaat`) that audits all
> user-installed applications on a macOS, Linux, or generic Unix machine and
> emits a structured inventory plus a restoration manifest for migrating to a
> new machine. The tool ships as a single static binary per platform
> (macOS arm64/amd64, Linux amd64/arm64), distributed via GitHub Releases and a
> Homebrew tap.
>
> **Per detected app, capture:**
>
> | Column | Source |
> |---|---|
> | Application name | App bundle name / package name |
> | Author / publisher | Code-signing identity, package author, plist metadata |
> | Vendor URL | Code-sign developer ID lookup, plist, package metadata, App Store listing |
> | Installation source | App Store / Homebrew (formula vs cask) / pkg / DMG / direct download / system / sandbox / unknown |
> | Original download URL | `mdls kMDItemWhereFroms`, `com.apple.quarantine` xattr, brew tap source, mas product page |
> | Version | `CFBundleShortVersionString` / package version |
> | Bundle identifier / package ID | `CFBundleIdentifier` / dpkg / rpm name |
> | Install date | File mtime / pkg receipt date |
> | Last-used date | `kMDItemLastUsedDate` (macOS) / atime (Linux) |
> | Size on disk | `du -sh` equivalent |
> | Signing status | `codesign -dv` / GPG sig / unsigned |
> | Apple Silicon compat (macOS) | `lipo -archs` |
> | Reinstall command | `brew install --cask <x>` / `mas install <id>` / `apt install <pkg>` / vendor URL |
>
> **License keys** are written to a separate, well-organized `secrets.json`
> with full unredacted values, structured by category (App Store / Homebrew /
> standalone / Keychain-found / manual-checklist). The audit `report.json`
> does **not** include keys. Three scanner modes selectable per run:
> `best-effort`, `checklist`, `aggressive`. `secrets.json` is opt-in encrypted
> via `age` when the user provides a recipient.
>
> **Reports:** JSON, CSV, Markdown table, HTML. **Restoration manifest:**
> Brewfile + mas list + apt/dnf/pacman install lists + per-app
> manual-restore notes. **Backup bundle:** single encrypted tar containing
> audit + secrets + manifests.
>
> **UX:** when `stdin.IsTerminal() && len(flags) == 0`, open an interactive
> wizard built on `charmbracelet/huh` (forms with select, multi-select,
> confirm, input). For custom screens beyond `huh`'s scope (live scan view,
> per-app review panel), drop down to `charmbracelet/bubbletea` directly.
> The wizard collects every setting a flag invocation would set, runs the
> scan with a progress bar, and prints the equivalent non-interactive command
> at the end. Wizard answers can be saved as named **profiles**
> (`~/.config/osaat/profiles/<name>.toml`) and replayed with
> `osaat scan --profile <name>`.
>
> **Diff:** `osaat diff old.json new.json` for inventory drift between
> machines or over time. **Schedule:** `osaat install-schedule --weekly`
> writes a launchd plist (macOS) or systemd user unit (Linux) for ongoing
> audits.

## Decision log

| Date | Decision | Notes |
|---|---|---|
| 2026-05-16 | Cross-platform from day one, per-OS collectors | DRY/KISS — one `AppRecord` model, N collectors |
| 2026-05-16 | Language = Go | Single static binary; user override of earlier Python recommendation |
| 2026-05-16 | Project rename `macos-backup-script` → `osaat` | Cross-platform scope; folder rename happens in Phase 0 |
| 2026-05-16 | Binary name = `osaat` | Short, matches project initials |
| 2026-05-16 | TUI = `huh` (forms) + `bubbletea` (custom screens) | Charm ecosystem; same runtime |
| 2026-05-16 | Secrets in separate `secrets.json`, full keys, well-organized | Not redacted within `secrets.json`; optional age encryption |
| 2026-05-16 | All four v0.1 bundles in scope | UX + Security + Insight + Automation |
