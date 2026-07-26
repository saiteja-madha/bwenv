class Bwenv < Formula
  desc "App-scoped environments backed by Bitwarden Secrets Manager"
  homepage "https://github.com/saiteja-madha/bwenv"
  version "1.0.0"
  license "MIT"
  head "https://github.com/saiteja-madha/bwenv.git", branch: "main" do
    depends_on "go" => :build
  end

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/saiteja-madha/bwenv/releases/download/v1.0.0/bwenv-darwin-arm64", using: :nounzip
      sha256 "e4e8566d953631b88c3a3d506471c0d5e93e9c5296140b246f223cbdebec0ff4"
    else
      url "https://github.com/saiteja-madha/bwenv/releases/download/v1.0.0/bwenv-darwin-amd64", using: :nounzip
      sha256 "4aa0b9ee4c4f99c0d2e70fe17ababa2984e52c00507861b5d9f5dc6d3d46795c"
    end
  end

  on_linux do
    if Hardware::CPU.arm? && Hardware::CPU.is_64_bit?
      url "https://github.com/saiteja-madha/bwenv/releases/download/v1.0.0/bwenv-linux-arm64", using: :nounzip
      sha256 "1ceb0d31a5c4f43afc5af88f3bf94a3b0d92c18d8b82a8f41d8c5b061aa0e182"
    else
      url "https://github.com/saiteja-madha/bwenv/releases/download/v1.0.0/bwenv-linux-amd64", using: :nounzip
      sha256 "c45fca0a306b38b9859c6bb2a158185d10458194e4da08a42f347c6210c6d2ab"
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
