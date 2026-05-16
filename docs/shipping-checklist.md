# Shipping checklist — migrating to a new machine

**Status:** skeleton — full body in Phase 3 / Phase 4.
A more detailed working version lives at
[`.design/backup-restore-checklist.md`](../.design/backup-restore-checklist.md).

## On the OLD machine

```sh
# 1. Full audit with restore manifest
osaat scan \
    --license-mode best-effort \
    --age-recipient "$AGE_RECIPIENT" \
    --insights forgotten,apple-silicon \
    --with-restore \
    --out ~/backup/osaat-$(date +%F)/

# 2. Bundle the output
osaat backup \
    --from ~/backup/osaat-$(date +%F)/ \
    --age-recipient "$AGE_RECIPIENT" \
    --out ~/backup/osaat-$(date +%F).tar.age
```

Then manually:

- Export Keychain (Keychain Access → File → Export Items)
- Copy `~/.ssh/`, `~/.gnupg/`, dotfiles
- Save paid-software email receipts

## On the NEW machine

```sh
# 1. Prerequisites
xcode-select --install
/bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"
brew install mas age simtabi/tap/osaat

# 2. Restore the bundle
mkdir -p ~/backup
age -d -i ~/.age/key.txt ~/backup/osaat-YYYY-MM-DD.tar.age | tar -x -C ~/backup/
cd ~/backup/osaat-YYYY-MM-DD/

# 3. Bulk reinstall
brew bundle --file=./Brewfile
mas signin --dialog
xargs -L1 mas install < ./mas-apps.txt

# 4. Per-app manual installs — open RESTORE.md
```

## Verify

```sh
osaat scan --out ~/backup/osaat-new/
osaat diff ~/backup/osaat-YYYY-MM-DD/report.json ~/backup/osaat-new/report.json
```

Anything in the diff that's marked `-` (present on the old machine,
missing on the new) needs another install pass or an explicit "intentionally
dropped" note.

## Ongoing audits

```sh
osaat install-schedule --weekly --out ~/backup/osaat-{date}/
```

Writes a launchd plist (macOS) or systemd user unit (Linux) that runs the
scan weekly. Drift between weeks surfaces via `osaat diff`.
