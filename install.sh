#!/usr/bin/env bash
set -euo pipefail

REPO="${BWENV_REPO:-saiteja-madha/bwenv}"
VERSION="${BWENV_VERSION:-latest}"

fail() {
  printf 'bwenv installer: %s\n' "$*" >&2
  exit 1
}

command -v curl >/dev/null 2>&1 || fail "curl is required"

case "$(uname -s)" in
  Darwin) os="darwin" ;;
  Linux) os="linux" ;;
  *) fail "unsupported operating system: $(uname -s)" ;;
esac

case "$(uname -m)" in
  x86_64|amd64) arch="amd64" ;;
  arm64|aarch64) arch="arm64" ;;
  *) fail "unsupported architecture: $(uname -m)" ;;
esac

if [[ -n "${INSTALL_DIR:-}" ]]; then
  install_dir="$INSTALL_DIR"
elif [[ -d /usr/local/bin && -w /usr/local/bin ]]; then
  install_dir="/usr/local/bin"
else
  install_dir="${HOME}/.local/bin"
fi

asset="bwenv-${os}-${arch}"
if [[ "$VERSION" == "latest" ]]; then
  release_base="https://github.com/${REPO}/releases/latest/download"
else
  tag="$VERSION"
  [[ "$tag" == v* ]] || tag="v${tag}"
  release_base="https://github.com/${REPO}/releases/download/${tag}"
fi

tmp_dir="$(mktemp -d)"
trap 'rm -rf "$tmp_dir"' EXIT
binary_path="${tmp_dir}/${asset}"
checksums_path="${tmp_dir}/checksums.txt"

printf 'Installing bwenv for %s/%s...\n' "$os" "$arch"
curl --fail --silent --show-error --location --proto '=https' --tlsv1.2 \
  "${release_base}/${asset}" --output "$binary_path"
curl --fail --silent --show-error --location --proto '=https' --tlsv1.2 \
  "${release_base}/checksums.txt" --output "$checksums_path"

expected_hash="$(awk -v file="$asset" '$2 == file || $2 == "*" file { print $1; exit }' "$checksums_path")"
[[ -n "$expected_hash" ]] || fail "release checksum does not contain ${asset}"

if command -v sha256sum >/dev/null 2>&1; then
  actual_hash="$(sha256sum "$binary_path" | awk '{print $1}')"
elif command -v shasum >/dev/null 2>&1; then
  actual_hash="$(shasum -a 256 "$binary_path" | awk '{print $1}')"
else
  fail "sha256sum or shasum is required to verify the download"
fi
[[ "$actual_hash" == "$expected_hash" ]] || fail "checksum mismatch for ${asset}"

mkdir -p "$install_dir"
install -m 0755 "$binary_path" "${install_dir}/bwenv"
"${install_dir}/bwenv" version >/dev/null

printf 'Installed bwenv to %s/bwenv\n' "$install_dir"
case ":${PATH}:" in
  *":${install_dir}:"*) ;;
  *) printf 'Add %s to PATH before running bwenv.\n' "$install_dir" ;;
esac
