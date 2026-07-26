#!/usr/bin/env bash
set -euo pipefail

version="${1:?usage: render-homebrew-formula.sh <version> <checksums> [output]}"
checksums="${2:?usage: render-homebrew-formula.sh <version> <checksums> [output]}"
output="${3:-Formula/bwenv.rb}"
tag="$version"
[[ "$tag" == v* ]] || tag="v${tag}"
formula_version="${tag#v}"

[[ "$formula_version" =~ ^[0-9]+\.[0-9]+\.[0-9]+([.-][0-9A-Za-z.-]+)?$ ]] || {
  printf 'invalid release version: %s\n' "$version" >&2
  exit 1
}
[[ -f "$checksums" ]] || {
  printf 'checksums file not found: %s\n' "$checksums" >&2
  exit 1
}

checksum() {
  local asset="$1"
  local value
  value="$(awk -v file="$asset" '$2 == file || $2 == "*" file { print $1; exit }' "$checksums")"
  [[ "$value" =~ ^[0-9a-fA-F]{64}$ ]] || {
    printf 'missing or invalid checksum for %s\n' "$asset" >&2
    exit 1
  }
  printf '%s' "$value" | tr '[:upper:]' '[:lower:]'
}

darwin_amd64="$(checksum bwenv-darwin-amd64)"
darwin_arm64="$(checksum bwenv-darwin-arm64)"
linux_amd64="$(checksum bwenv-linux-amd64)"
linux_arm64="$(checksum bwenv-linux-arm64)"

mkdir -p "$(dirname "$output")"
cat >"$output" <<EOF
class Bwenv < Formula
  desc "App-scoped environments backed by Bitwarden Secrets Manager"
  homepage "https://github.com/saiteja-madha/bwenv"
  version "$formula_version"
  license "MIT"
  head "https://github.com/saiteja-madha/bwenv.git", branch: "main" do
    depends_on "go" => :build
  end

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/saiteja-madha/bwenv/releases/download/$tag/bwenv-darwin-arm64", using: :nounzip
      sha256 "$darwin_arm64"
    else
      url "https://github.com/saiteja-madha/bwenv/releases/download/$tag/bwenv-darwin-amd64", using: :nounzip
      sha256 "$darwin_amd64"
    end
  end

  on_linux do
    if Hardware::CPU.arm? && Hardware::CPU.is_64_bit?
      url "https://github.com/saiteja-madha/bwenv/releases/download/$tag/bwenv-linux-arm64", using: :nounzip
      sha256 "$linux_arm64"
    else
      url "https://github.com/saiteja-madha/bwenv/releases/download/$tag/bwenv-linux-amd64", using: :nounzip
      sha256 "$linux_amd64"
    end
  end

  def install
    if build.head?
      ldflags = %W[
        -s -w
        -X bwenv/cmd.Version=#{version}
        -X bwenv/cmd.Commit=#{version}
        -X bwenv/cmd.Date=homebrew
      ]
      system "go", "build", *std_go_args(ldflags: ldflags), "."
    else
      binary = Dir["bwenv-*"].first
      odie "release binary not found" unless binary
      chmod 0755, binary
      bin.install binary => "bwenv"
    end
  end

  test do
    expected = build.head? ? version.to_s : "v#{version}"
    assert_match "bwenv #{expected}", shell_output("#{bin}/bwenv version")
    assert_match "create", shell_output("#{bin}/bwenv --help")
  end
end
EOF
