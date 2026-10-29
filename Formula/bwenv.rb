class Bwenv < Formula
  desc "App-scoped environments backed by Bitwarden Secrets Manager"
  homepage "https://github.com/saiteja-madha/bwenv"
  version "1.3.0"
  license "MIT"
  head "https://github.com/saiteja-madha/bwenv.git", branch: "main" do
    depends_on "go" => :build
  end

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/saiteja-madha/bwenv/releases/download/v1.3.0/bwenv-darwin-arm64", using: :nounzip
      sha256 "36bb2e0edfa0a95c47ef821860041359d1c41452c47995eb3a11b81e7c0c9aa7"
    else
      url "https://github.com/saiteja-madha/bwenv/releases/download/v1.3.0/bwenv-darwin-amd64", using: :nounzip
      sha256 "2c205de8cf1fa1d14ab3f7f66a2e409f4aad1ad4b11ea85b7d2257ff78873fa9"
    end
  end

  on_linux do
    if Hardware::CPU.arm? && Hardware::CPU.is_64_bit?
      url "https://github.com/saiteja-madha/bwenv/releases/download/v1.3.0/bwenv-linux-arm64", using: :nounzip
      sha256 "aca384623aa9df6a89e2b20fb3637235d64e23b53eb90979f375d17ec86ba6c5"
    else
      url "https://github.com/saiteja-madha/bwenv/releases/download/v1.3.0/bwenv-linux-amd64", using: :nounzip
      sha256 "117c2c3508ed2ec756ac2e079d9558d64a2422fb0bbd60e8c956b958d0bda45b"
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
