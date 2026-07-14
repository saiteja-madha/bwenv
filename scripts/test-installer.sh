#!/usr/bin/env bash
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT
mkdir -p "$tmp/fake-bin" "$tmp/release" "$tmp/install"

case "$(uname -s)" in
  Darwin) os="darwin" ;;
  Linux) os="linux" ;;
  *) printf 'installer test skipped on unsupported OS\n'; exit 0 ;;
esac
case "$(uname -m)" in
  x86_64|amd64) arch="amd64" ;;
  arm64|aarch64) arch="arm64" ;;
  *) printf 'installer test skipped on unsupported architecture\n'; exit 0 ;;
esac
asset="bwenv-${os}-${arch}"

# The generated fixture must evaluate its arguments when it runs.
# shellcheck disable=SC2016
printf '#!/usr/bin/env bash\n[[ "${1:-}" == version ]] || exit 2\nprintf "bwenv test\\n"\n' >"$tmp/release/$asset"
chmod +x "$tmp/release/$asset"
if command -v sha256sum >/dev/null 2>&1; then
  hash="$(sha256sum "$tmp/release/$asset" | awk '{print $1}')"
else
  hash="$(shasum -a 256 "$tmp/release/$asset" | awk '{print $1}')"
fi
printf '%s  %s\n' "$hash" "$asset" >"$tmp/release/checksums.txt"

# These single-quoted lines intentionally form a standalone fake curl script.
# shellcheck disable=SC2016
printf '%s\n' '#!/usr/bin/env bash' 'set -euo pipefail' \
  'url=""; output=""' \
  'while (($#)); do' \
  '  case "$1" in' \
  '    http*) url="$1"; shift ;;' \
  '    --output|-o) output="$2"; shift 2 ;;' \
  '    *) shift ;;' \
  '  esac' \
  'done' \
  'cp "${FIXTURE_DIR}/$(basename "$url")" "$output"' >"$tmp/fake-bin/curl"
chmod +x "$tmp/fake-bin/curl"

PATH="$tmp/fake-bin:$PATH" FIXTURE_DIR="$tmp/release" INSTALL_DIR="$tmp/install" \
  bash "$root/install.sh" >/dev/null
"$tmp/install/bwenv" version >/dev/null

printf '0%.0s' {1..64} >"$tmp/release/checksums.txt"
printf '  %s\n' "$asset" >>"$tmp/release/checksums.txt"
if PATH="$tmp/fake-bin:$PATH" FIXTURE_DIR="$tmp/release" INSTALL_DIR="$tmp/install" \
  bash "$root/install.sh" >/dev/null 2>&1; then
  printf 'installer accepted an invalid checksum\n' >&2
  exit 1
fi
