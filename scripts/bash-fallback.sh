#!/usr/bin/env bash
# bash-fallback.sh — minimal macOS application audit, no Go required.
#
# Produces a report.json that conforms to the osaat.report/v1 schema,
# so the Go binary (and `osaat diff`) can consume the output. Use this
# when you need a record of what's installed on a Mac *before* you
# bootstrap Homebrew / Go / osaat itself — for example, on a fresh
# machine you're about to migrate to.
#
# Limitations vs. the Go binary:
#   - macOS only.
#   - No license extraction (use the Go binary for that).
#   - No age encryption.
#   - No PDF / Markdown / TXT / CSV / HTML reporters.
#   - No `mas` / `brew` cross-reference; source is filesystem-derived only.
#   - No SHA256SUMS file.
#
# Compatibility: Bash 3.2 (the default `/bin/bash` on macOS), POSIX
# tools (awk, sed, grep, stat, du), and macOS-specific `mdls`,
# `codesign`, `hostname`, `uname`, `xattr`.
#
# Usage:
#   ./bash-fallback.sh                       # write to stdout
#   ./bash-fallback.sh --out <dir>           # write <dir>/report.json
#   ./bash-fallback.sh --pretty              # pretty-print (default)
#   ./bash-fallback.sh --help

set -euo pipefail
LC_ALL=C
export LC_ALL

readonly VERSION="1.0"
readonly SCHEMA="osaat.report/v1"

usage() {
  cat <<'EOF'
osaat bash-fallback — minimal macOS application audit.

Usage:
  bash-fallback.sh [--out <dir>] [--help]

Flags:
  --out <dir>   Write <dir>/report.json instead of stdout.
  --help, -h    Show this message.

The output is osaat.report/v1 compatible and can be consumed by:

  osaat diff <old.json> bash-fallback-report.json
EOF
}

OUT_DIR=""

while [[ $# -gt 0 ]]; do
  case "$1" in
    --out)
      if [[ $# -lt 2 ]]; then echo "--out requires an argument" >&2; exit 2; fi
      OUT_DIR="$2"; shift 2 ;;
    --help|-h)
      usage; exit 0 ;;
    --)
      shift; break ;;
    -*)
      echo "unknown flag: $1" >&2
      usage >&2
      exit 2 ;;
    *)
      echo "unexpected positional argument: $1" >&2
      exit 2 ;;
  esac
done

if [[ "$(uname -s)" != "Darwin" ]]; then
  echo "bash-fallback.sh only runs on macOS. Use the Go binary on other platforms." >&2
  exit 1
fi

# json_escape stdin → stdout. Escapes \, ", control chars, and newlines.
# Intentionally simple — relies on the limited character set of plist
# values (mostly ASCII strings).
json_escape() {
  local s="$1"
  s="${s//\\/\\\\}"
  s="${s//\"/\\\"}"
  s="${s//	/\\t}"
  s="${s//$'\n'/\\n}"
  s="${s//$'\r'/\\r}"
  printf '%s' "$s"
}

# read_mdls <path> <attribute>  →  echoes value or empty string for "(null)".
read_mdls() {
  local val
  val=$(mdls -name "$2" -raw "$1" 2>/dev/null || true)
  if [[ "$val" == "(null)" ]]; then
    printf ''
  else
    printf '%s' "$val"
  fi
}

# epoch_to_rfc3339 <epoch>  →  echoes UTC RFC 3339 timestamp.
epoch_to_rfc3339() {
  local epoch="$1"
  if [[ -z "$epoch" ]]; then return; fi
  # macOS date(1) uses -r for "from epoch".
  date -u -r "$epoch" "+%Y-%m-%dT%H:%M:%SZ" 2>/dev/null || true
}

# signing_status <app_path>  →  echoes signed | ad_hoc | unsigned | unknown.
signing_status() {
  local path="$1"
  local out
  out=$(codesign -dv --verbose=4 "$path" 2>&1 || true)
  case "$out" in
    *"not signed at all"*) printf 'unsigned' ;;
    *"Signature=adhoc"*)   printf 'ad_hoc' ;;
    *"Authority="*)        printf 'signed' ;;
    *)                     printf 'unknown' ;;
  esac
}

# whereFroms_url <app_path>  →  echoes the first URL from kMDItemWhereFroms.
whereFroms_url() {
  local path="$1"
  local raw
  raw=$(mdls -name kMDItemWhereFroms "$path" 2>/dev/null || true)
  printf '%s\n' "$raw" | grep -Eo 'https?://[^"]+' | head -n 1 || true
}

# emit_record_json — print one JSON object for a record, with field
# omission when a value is empty. Trailing comma handling is left to
# the caller (we just print the object).
emit_record_json() {
  local name="$1" bundle_id="$2" version="$3" source="$4" path="$5"
  local size_bytes="$6" installed_at="$7" signing="$8" download_url="$9"

  printf '    {\n'
  printf '      "name": "%s"' "$(json_escape "$name")"
  if [[ -n "$bundle_id" ]]; then
    printf ',\n      "bundle_id": "%s"' "$(json_escape "$bundle_id")"
  fi
  if [[ -n "$version" ]]; then
    printf ',\n      "version": "%s"' "$(json_escape "$version")"
  fi
  printf ',\n      "source": "%s"' "$(json_escape "$source")"
  if [[ -n "$download_url" ]]; then
    printf ',\n      "download_url": "%s"' "$(json_escape "$download_url")"
  fi
  if [[ -n "$path" ]]; then
    printf ',\n      "path": "%s"' "$(json_escape "$path")"
  fi
  if [[ -n "$size_bytes" && "$size_bytes" != "0" ]]; then
    printf ',\n      "size_bytes": %s' "$size_bytes"
  fi
  if [[ -n "$installed_at" ]]; then
    printf ',\n      "installed_at": "%s"' "$installed_at"
  fi
  if [[ -n "$signing" && "$signing" != "unknown" ]]; then
    printf ',\n      "signing_status": "%s"' "$signing"
  fi
  printf '\n    }'
}

# process_app — emit one record JSON object to fd 3 (the buffered
# records list). Called per .app bundle.
process_app() {
  local app_path="$1" default_source="$2"

  local name bundle_id version
  name=$(read_mdls "$app_path" kMDItemDisplayName)
  if [[ -z "$name" ]]; then
    name="$(basename "$app_path" .app)"
  fi
  # Some apps set kMDItemDisplayName to include ".app"; strip it.
  name="${name%.app}"
  bundle_id=$(read_mdls "$app_path" kMDItemCFBundleIdentifier)
  version=$(read_mdls "$app_path" kMDItemVersion)

  local size_kb size_bytes=0
  size_kb=$(du -sk "$app_path" 2>/dev/null | awk '{print $1}')
  if [[ -n "$size_kb" ]]; then
    size_bytes=$(( size_kb * 1024 ))
  fi

  local epoch installed_at
  epoch=$(stat -f %m "$app_path" 2>/dev/null || true)
  installed_at=$(epoch_to_rfc3339 "$epoch")

  local signing
  signing=$(signing_status "$app_path")

  local download_url
  download_url=$(whereFroms_url "$app_path")

  emit_record_json "$name" "$bundle_id" "$version" \
    "$default_source" "$app_path" "$size_bytes" "$installed_at" \
    "$signing" "$download_url"
}

# scan_root walks one application root and emits each .app inside it.
# COUNT and FIRST track JSON comma placement.
COUNT=0
FIRST=1

scan_root() {
  local root="$1" default_source="$2"
  [[ -d "$root" ]] || return 0
  for app in "$root"/*.app; do
    [[ -d "$app" ]] || continue
    if [[ $FIRST -eq 1 ]]; then
      FIRST=0
    else
      printf ',\n'
    fi
    process_app "$app" "$default_source"
    COUNT=$(( COUNT + 1 ))
  done
}

# Header → stdout (or the file via redirection at the end).
emit_header() {
  local arch
  arch=$(uname -m)
  case "$arch" in
    x86_64) arch=amd64 ;;
    aarch64) arch=arm64 ;;
  esac
  local hn
  hn=$(hostname -s)
  printf '{\n'
  printf '  "schema": "%s",\n' "$SCHEMA"
  printf '  "generated_at": "%s",\n' "$(date -u +"%Y-%m-%dT%H:%M:%SZ")"
  printf '  "host": {\n'
  printf '    "hostname": "%s",\n' "$(json_escape "$hn")"
  printf '    "os": "darwin",\n'
  printf '    "arch": "%s"\n' "$arch"
  printf '  },\n'
  printf '  "tool": {\n'
  printf '    "name": "osaat-bash-fallback",\n'
  printf '    "version": "%s",\n' "$VERSION"
  printf '    "commit": ""\n'
  printf '  },\n'
  printf '  "records": [\n'
}

emit_footer() {
  if [[ $COUNT -gt 0 ]]; then
    printf '\n'
  fi
  printf '  ]\n'
  printf '}\n'
}

# Build the full document.
build_report() {
  emit_header
  scan_root "/Applications" "unknown"
  scan_root "/Applications/Utilities" "unknown"
  scan_root "$HOME/Applications" "unknown"
  scan_root "/System/Applications" "system"
  scan_root "/System/Applications/Utilities" "system"
  emit_footer
  printf 'osaat bash-fallback: %d apps audited.\n' "$COUNT" >&2
}

if [[ -n "$OUT_DIR" ]]; then
  mkdir -p "$OUT_DIR"
  out_path="$OUT_DIR/report.json"
  build_report > "$out_path"
  printf 'wrote %s\n' "$out_path"
else
  build_report
fi
