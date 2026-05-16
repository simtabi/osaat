# `osaat backup`

Bundle a scan output directory into a single age-encrypted tar
archive — or decrypt one back to a directory.

## Usage

```sh
# Create
osaat backup --from <dir> --age-recipient <age1...> --out <file.tar.age>

# Decrypt + extract
osaat backup --decrypt --in <file.tar.age> --age-key <path> --out <dir>
```

## What's in the archive

By default the archive contains only the known osaat output set,
so a stray note you dropped in the same directory doesn't end up in
your backup. Pass `--include-extras` to copy everything.

```
report.pdf, report.md, report.txt, report.json, report.csv, report.html
secrets.json or secrets.json.age
Brewfile
mas-apps.txt
apt-packages.txt, dnf-packages.txt, pacman-packages.txt
RESTORE.md
SHA256SUMS
osaat-metadata.json
```

Files outside this list (other than via `--include-extras`) are
silently skipped.

## Flags

| Flag | Default | Effect |
|---|---|---|
| `--from <dir>` | _required (create)_ | Source directory — typically the output of `osaat scan`. |
| `--age-recipient <key>` | _required (create)_ | age recipient public key (`age1...`). |
| `--decrypt` | `false` | Switch to decrypt mode. |
| `--in <file>` | _required (decrypt)_ | Encrypted archive path. |
| `--age-key <path>` | `~/.age/key.txt` (if present) | age private key file (the form `age-keygen` produces). |
| `--out <file>` (create) | `<from>.tar.age` next to `--from` | Output path or directory. |
| `--out <dir>` (decrypt) | _required_ | Extraction directory. Created if missing (mode 700). |
| `--include-extras` | `false` | Archive every regular file in `--from`, not just the known set. |

## Round-trip

The format is exact:

```sh
osaat backup --from old/ --age-recipient $RCPT --out bundle.tar.age
osaat backup --decrypt --in bundle.tar.age --age-key ~/.age/key.txt --out new/
diff -r old/ new/
# (exit 0, no output)
```

Every regular file round-trips byte-for-byte. Directories aren't
walked, so the source directory must be flat.

## Examples

```sh
# Create — bundle today's scan
osaat backup \
    --from ~/Documents/osaat/2026-05-16 \
    --age-recipient $(cat ~/.age/recipient.txt) \
    --out ~/Documents/osaat/2026-05-16.tar.age

# Decrypt to verify
osaat backup --decrypt \
    --in ~/Documents/osaat/2026-05-16.tar.age \
    --age-key ~/.age/key.txt \
    --out /tmp/check-2026-05-16
```

## Exit codes

| Code | Meaning |
|---|---|
| 0 | Archive written or extracted. |
| 1 | I/O or crypto error. |
| 2 | Flag misuse — see stderr. |

## Why age and not gpg

age is pure Go (no system binary requirement), has a small explicit
API, and produces a tightly defined wire format. We ship the
library statically as part of `osaat` — there's nothing else to
install. If you keep your secrets in gpg today, generate an age
recipient just for this use case; it's a small one-time setup.
