# Phase 4 — Linux + Unix collectors

**Goal:** `osaat scan --os linux` works on dpkg, rpm, pacman, snap,
flatpak, and AppImage systems. `osaat scan --os unix` covers BSD pkg and
generic POSIX fallback. The CI matrix grows.

## Tasks

### `internal/collectors/linux/linux.go` (orchestrator)
- [ ] Detects which package managers are present (`exec.LookPath`).
- [ ] Runs each available sub-collector concurrently.
- [ ] Merges into `[]AppRecord` deduping by `PkgID`.

### `internal/collectors/linux/dpkg.go` (Debian / Ubuntu)
- [ ] `dpkg-query -W -f='${Package}\t${Version}\t${Maintainer}\t${Installed-Size}\t${Status}\n'`
- [ ] Parse, filter for `installed` status.
- [ ] Reinstall: `apt install <Package>=<Version>` (or `<Package>`
      when version pinning is overkill).

### `internal/collectors/linux/rpm.go` (Fedora / RHEL / openSUSE)
- [ ] `rpm -qa --qf '%{NAME}\t%{VERSION}\t%{VENDOR}\t%{SIZE}\t%{INSTALLTIME}\n'`
- [ ] Reinstall: `dnf install <Name>` (or `zypper install` based on
      `/etc/os-release ID`).

### `internal/collectors/linux/pacman.go` (Arch / Manjaro)
- [ ] `pacman -Q` for names; `pacman -Qi <name>` for details when
      records need enrichment.
- [ ] Reinstall: `pacman -S <name>`.

### `internal/collectors/linux/snap.go`
- [ ] `snap list` parsed by columns.
- [ ] Reinstall: `snap install <name>` (record channel when known).

### `internal/collectors/linux/flatpak.go`
- [ ] `flatpak list --app --columns=application,version,arch,branch,installation`
- [ ] Reinstall: `flatpak install <remote> <application>`.

### `internal/collectors/linux/appimage.go`
- [ ] Scan `~/.local/bin/`, `~/Applications/`, `~/AppImages/`, and a
      configurable list for `*.AppImage` files.
- [ ] Fields from file metadata only (size, mtime, executable name).
      AppImages don't have a package manager — `DownloadURL` left empty
      unless `xattr -p user.xdg.origin.url` is set (some download
      managers set it).

### `internal/collectors/linux/desktop_files.go`
- [ ] Cross-reference `/usr/share/applications/*.desktop` and
      `~/.local/share/applications/*.desktop` to enrich records with
      `Name`, vendor URL, and icon path. Augments package-manager
      records.

### `internal/collectors/unix/`
- [ ] `bsd_pkg.go` — `pkg info -a` (FreeBSD), `pkg_info` (NetBSD/OpenBSD).
- [ ] `pkgsrc.go` — `pkg_info` on systems that use pkgsrc.
- [ ] `unix.go` — orchestrator; same shape as Linux.

### Insight columns on Linux
- [ ] Forgotten apps via `atime` of the package's primary binary
      (`dpkg -L <pkg> | head` to find a binary).
- [ ] Apple Silicon column: nil. (Documented in the reporter as
      "macOS only".)

### Restore manifest extensions
- [ ] `internal/restore/apt_list.go` — emits `apt-packages.txt`
      consumable by `xargs -L1 apt install`.
- [ ] `internal/restore/dnf_list.go` — `dnf-packages.txt`.
- [ ] `internal/restore/pacman_list.go` — `pacman-packages.txt`.
- [ ] `RESTORE.md` updated to surface these per-distro lists.

### CI
- [ ] `.github/workflows/ci.yml` matrix adds `ubuntu-latest` and
      `fedora` (via container) runners.
- [ ] Smoke test on each: install a known package, run `osaat scan
      --os linux`, assert it appears in `report.json`.

### Tests
- [ ] Per-collector fixture tests (canned `dpkg-query` output, etc.).
- [ ] One end-to-end test per supported package manager — guarded by
      build tag `linux` so they only run in matrix jobs.

### Definition of done

```bash
$ osaat scan --os linux --format json --out ./out
[osaat] dpkg: 1840 pkgs   snap: 17   flatpak: 9   AppImage: 3
[osaat] wrote ./out/report.json

$ osaat scan --os auto                # auto-detect on whatever host
[osaat] detected: linux/amd64 (Ubuntu 24.04)
```

`make test && make lint` green; Linux CI matrix green.

## Non-goals for Phase 4

- Windows.
- License extraction on Linux (deferred; far less standardized than
  macOS plists).
- Public release (Phase 5).
