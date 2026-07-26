#!/usr/bin/env bash
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT
checksums="$tmp/checksums.txt"
formula="$tmp/bwenv.rb"

for asset in \
  bwenv-darwin-amd64 \
  bwenv-darwin-arm64 \
  bwenv-linux-amd64 \
  bwenv-linux-arm64; do
  printf '%064d  %s\n' 1 "$asset" >>"$checksums"
done

"$root/scripts/render-homebrew-formula.sh" v9.8.7 "$checksums" "$formula"
ruby -c "$formula" >/dev/null
grep -Fq 'version "9.8.7"' "$formula"
grep -Fq '/v9.8.7/bwenv-darwin-arm64' "$formula"

if "$root/scripts/render-homebrew-formula.sh" v9.8.7 /dev/null "$formula" >/dev/null 2>&1; then
  printf 'formula renderer accepted missing checksums\n' >&2
  exit 1
fi
