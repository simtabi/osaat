# `osaat backup`

**Status:** skeleton — full body in Phase 3.

Bundle a scan output directory into a single encrypted archive
(`tar.age`) for safe storage or transfer.

## Usage

```sh
# Create
osaat backup --from <dir> --age-recipient <key> --out <file.tar.age>

# Decrypt + extract
osaat backup --decrypt --in <file.tar.age> --age-key <path> --out <dir>
```

## What's in the archive

```
report.json
REPORT.md
report.csv               (if present in source)
report.html              (if present)
secrets.json.age         (re-included even though it's already encrypted — convenience)
Brewfile
mas-apps.txt
apt-packages.txt         (if present)
RESTORE.md
osaat-metadata.json
```

Files not in the above list are skipped by default. Use `--include-extras`
to include everything in the source directory.

## Flags

| Flag | Default | Effect |
|---|---|---|
| `--from <dir>` | _required for create_ | Source directory (typically the output of `osaat scan`). |
| `--age-recipient <key>` | _required for create_ | age recipient — `age1...` public key, or `@<file>` for a recipients file. |
| `--out <file>` | _required_ | Output path. |
| `--decrypt` | `false` | Switch to decrypt mode. |
| `--in <file>` | _required for decrypt_ | Encrypted archive path. |
| `--age-key <path>` | `~/.age/key.txt` if exists | age private key for decryption. |
| `--include-extras` | `false` | Include every file under `--from`, not just the known set. |

## Examples

Create:

```sh
osaat backup \
    --from ~/backup/osaat-2026-05-16/ \
    --age-recipient $(cat ~/.age/recipient.txt) \
    --out ~/backup/osaat-2026-05-16.tar.age
```

Round-trip:

```sh
osaat backup --decrypt \
    --in ~/backup/osaat-2026-05-16.tar.age \
    --age-key ~/.age/key.txt \
    --out /tmp/restore-check
```

## Why age and not gpg

age is a pure-Go library and produces a small, modern, audited format.
Adding `gpg` as a runtime dependency would mean either bundling it
(license-incompatible) or asking every user to install it. age is what
the binary already links against.

If you need `gpg` compatibility, decrypt the inner `secrets.json.age` with
the age `age-plugin-yubikey` / `age-plugin-gpg` plugins, then re-encrypt
the result with `gpg`.
