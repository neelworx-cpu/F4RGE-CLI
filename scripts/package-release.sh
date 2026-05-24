#!/usr/bin/env bash
set -euo pipefail

version="${1:-latest}"
root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
dist="$root/dist/cli"
go_bin="${GO:-go}"

mkdir -p "$dist"

build_archive() {
  local goos="$1"
  local goarch="$2"
  local ext=""
  local archive_ext="tar.gz"
  if [[ "$goos" == "windows" ]]; then
    ext=".exe"
    archive_ext="zip"
  fi

  local work
  work="$(mktemp -d)"
  trap 'rm -rf "$work"' RETURN

  echo "building $goos/$goarch"
  (cd "$root" && GOOS="$goos" GOARCH="$goarch" CGO_ENABLED=0 "$go_bin" build -o "$work/4rged$ext" .)

  local artifact="4rged-${version}-${goos}-${goarch}.${archive_ext}"
  if [[ "$goos" == "windows" ]]; then
    (cd "$work" && zip -q "$dist/$artifact" "4rged$ext")
  else
    (cd "$work" && tar -czf "$dist/$artifact" "4rged$ext")
  fi
  (cd "$dist" && shasum -a 256 "$artifact" > "$artifact.sha256")
}

build_archive darwin arm64
build_archive darwin amd64
build_archive linux arm64
build_archive linux amd64
build_archive windows amd64
build_archive windows arm64

echo "release artifacts written to $dist"
