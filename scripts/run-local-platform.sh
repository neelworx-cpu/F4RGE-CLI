#!/usr/bin/env bash
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
go_bin="${GO:-go}"

if ! command -v "$go_bin" >/dev/null 2>&1; then
  if [[ -x /tmp/go/bin/go ]]; then
    go_bin="/tmp/go/bin/go"
  else
    echo "Go toolchain not found. Set GO=/path/to/go or install Go." >&2
    exit 1
  fi
fi

export F4RGE_PLATFORM_URL="${F4RGE_PLATFORM_URL:-http://localhost:3007}"
export F4RGE_AUTH_URL="${F4RGE_AUTH_URL:-http://localhost:3003/cli}"
export F4RGED_GLOBAL_CONFIG="${F4RGED_GLOBAL_CONFIG:-/tmp/4rged-preview-config}"
export F4RGED_GLOBAL_DATA="${F4RGED_GLOBAL_DATA:-/tmp/4rged-preview-data}"
export F4RGED_CACHE_DIR="${F4RGED_CACHE_DIR:-/tmp/4rged-preview-cache}"
export FORCE_COLOR="${FORCE_COLOR:-1}"
export TERM="${TERM:-xterm-256color}"
export COLORTERM="${COLORTERM:-truecolor}"
unset NO_COLOR

cd "$root"
"$go_bin" build -o ./4rged .
exec ./4rged --data-dir /tmp/4rged-preview/.4rged "$@"
