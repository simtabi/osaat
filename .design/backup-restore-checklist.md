# Backup / restore checklist

**Status:** ratified, 2026-05-16
**Becomes:** `docs/shipping-checklist.md` once the project is scaffolded.

This runbook is what the tool's output supports. Every step that says "run
`osaat ...`" produces an artifact that a later step consumes.

## Pre-migration (run on the OLD machine)

### 1. Run a full audit with restore manifest

```bash
osaat scan \
    --os macos \
    --format json,markdown \
    --license-mode best-effort \
    --age-recipient "$AGE_RECIPIENT" \
    --insights forgotten,apple-silicon \
    --with-restore \
    --out ~/backup/osaat-$(date +%F)/
```

Produces:

```
~/backup/osaat-2026-05-16/
├── report.json             # canonical audit; no keys
├── REPORT.md               # human-readable table
├── secrets.json.age        # encrypted; full keys
├── Brewfile                # `brew bundle` rehydrates these
├── mas-apps.txt            # `xargs -L1 mas install <` rehydrates these
├── RESTORE.md              # per-app manual-install checklist
└── osaat-metadata.json     # host info, tool version, scan duration
```

### 2. Bundle the whole output

```bash
osaat backup \
    --from ~/backup/osaat-2026-05-16/ \
    --age-recipient "$AGE_RECIPIENT" \
    --out ~/backup/osaat-2026-05-16.tar.age
```

One encrypted archive. Stash on iCloud Drive / external SSD / wherever.

### 3. Manual steps the tool can't do

- [ ] **Keychain export.** Open Keychain Access → File → Export Items
  → save `login.keychain-db.p12`. Move this to the backup. There is no
  CLI equivalent — Apple gates this behind the GUI.
- [ ] **Dotfiles.** `~/.zshrc`, `~/.bashrc`, `~/.gitconfig`,
  `~/.config/`, `~/.aws/`, `~/.kube/`, etc. — copy whatever you care
  about.
- [ ] **SSH keys.** `~/.ssh/` (mode 700).
- [ ] **GPG keys.** `~/.gnupg/`.
- [ ] **Email receipts.** Gmail search:
  `from:(noreply OR receipt OR support) (license OR serial OR key)`.
  Archive a copy somewhere offline.
- [ ] **Browser data.** Bookmarks, history, saved passwords from each
  browser via its native export.
- [ ] **iMessage / Photos / Mail.** Use Apple Migration Assistant or
  iCloud — outside the scope of `osaat`.

### 4. Spot-check the audit before you wipe the machine

```bash
# Look at the report
osaat diff /dev/null ~/backup/osaat-2026-05-16/report.json | less

# Eyeball it for known apps that aren't there
grep -i -E "1password|adobe|jetbrains|paid-tool" ~/backup/osaat-2026-05-16/REPORT.md
```

If something's missing, fix the collector before migration. After the wipe
you can't audit the old machine.

## Post-migration (run on the NEW machine)

### 1. Install prerequisites

```bash
xcode-select --install                          # Xcode Command Line Tools
/bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"
brew install mas age osaat                      # mas + age + the tool itself
```

If `osaat` isn't on a tap yet, install from GitHub release:

```bash
brew install simtabi/tap/osaat
# or download the binary directly:
curl -fsSL https://github.com/simtabi/osaat/releases/latest/download/osaat-darwin-arm64.tar.gz | tar -xz
```

### 2. Restore the bundle

```bash
mkdir -p ~/backup
age -d -i ~/.age/key.txt ~/backup/osaat-2026-05-16.tar.age | tar -x -C ~/backup/
cd ~/backup/osaat-2026-05-16/
```

### 3. Bulk reinstall

```bash
# Homebrew formulae + casks
brew bundle --file=./Brewfile

# Mac App Store apps (mas sign-in is required first time)
mas signin --dialog                             # opens GUI for password
xargs -L1 mas install < ./mas-apps.txt
```

### 4. Per-app manual installs

Open `RESTORE.md`. It has one section per app that wasn't covered by
`brew bundle` or `mas install`. For each app:

- Vendor URL (where to download)
- Original download URL (where you got it last time)
- License-key reminder (pointer into `secrets.json`)
- Reinstall command if known

### 5. Restore secrets

```bash
age -d -i ~/.age/key.txt ./secrets.json.age > ./secrets.json
# Now manually paste each license key into the corresponding app on first launch.
```

### 6. Verify

```bash
# Audit the new machine
osaat scan --os macos --format json --out ~/backup/osaat-new/

# Compare to old
osaat diff \
    ~/backup/osaat-2026-05-16/report.json \
    ~/backup/osaat-new/report.json
```

The diff highlights apps present on the OLD machine but missing on the NEW.
Anything in that list either needs another install pass, or was
intentionally dropped — annotate it in your notes.

### 7. Set up ongoing audits

```bash
osaat install-schedule --weekly --out ~/backup/osaat-{date}/
```

Writes a launchd plist (macOS) or systemd user unit (Linux). Drift between
audits surfaces via `osaat diff` against the previous week's output.

## Recovery without `osaat` installed

`scripts/bash-fallback.sh` produces a `report.json` that is schema-compatible
with the Go binary's output. Use case: you're on a fresh Mac, Homebrew
isn't installed yet, and you want to audit before you start installing
things. Limitations: macOS only, no license extraction, no Apple Silicon
column, no scheduling.

## What to keep, what to throw away

| Artifact | Keep for | Why |
|---|---|---|
| `report.json` | indefinitely | Diffable; small; not sensitive |
| `secrets.json.age` | indefinitely | Encrypted; only useful with your age key |
| `Brewfile`, `mas-apps.txt` | until next audit | Regenerated each scan |
| `RESTORE.md`, `REPORT.md` | until next audit | Regenerated each scan |
| `*.tar.age` bundle | until next migration | One archive per machine-state snapshot |
