class Bwenv < Formula
  desc "Bitwarden Secrets Manager helper for local dev"
  homepage "https://github.com/saiteja-madha/bwenv"
  license "MIT"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/saiteja-madha/bwenv/releases/download/vVERSION/bwenv-darwin-arm64"
      sha256 "PLACEHOLDER_ARM64"
    else
      url "https://github.com/saiteja-madha/bwenv/releases/download/vVERSION/bwenv-darwin-amd64"
      sha256 "PLACEHOLDER_AMD64"
    end
  end

  on_linux do
    if Hardware::CPU.arm?
      url "https://github.com/saiteja-madha/bwenv/releases/download/vVERSION/bwenv-linux-arm64"
      sha256 "PLACEHOLDER_LINUX_ARM64"
    else
      url "https://github.com/saiteja-madha/bwenv/releases/download/vVERSION/bwenv-linux-amd64"
      sha256 "PLACEHOLDER_LINUX_AMD64"
    end
  end

  def install
    bin.install Dir["*"].first => "bwenv"
  end

  test do
    assert_match "bwenv", shell_output("#{bin}/bwenv --help")
  end
end
