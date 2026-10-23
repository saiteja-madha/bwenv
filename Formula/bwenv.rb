class Bwenv < Formula
  desc "App-scoped environments backed by Bitwarden Secrets Manager"
  homepage "https://github.com/saiteja-madha/bwenv"
  version "1.2.0"
  license "MIT"
  head "https://github.com/saiteja-madha/bwenv.git", branch: "main" do
    depends_on "go" => :build
  end

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/saiteja-madha/bwenv/releases/download/v1.2.0/bwenv-darwin-arm64", using: :nounzip
      sha256 "2821d05a5298033b144da74d498fede0f76bf50f2f5cf2dc046146b7e24c3c98"
    else
      url "https://github.com/saiteja-madha/bwenv/releases/download/v1.2.0/bwenv-darwin-amd64", using: :nounzip
      sha256 "c92831e725db27e323f289040b0512004849b13bd914ee0a8b91ced2cd6cf2cb"
    end
  end

  on_linux do
    if Hardware::CPU.arm? && Hardware::CPU.is_64_bit?
      url "https://github.com/saiteja-madha/bwenv/releases/download/v1.2.0/bwenv-linux-arm64", using: :nounzip
      sha256 "bd78c4269007ce8d25aa114e1d6a7e50c2d70d09746211efc69a5a3ae86de475"
    else
      url "https://github.com/saiteja-madha/bwenv/releases/download/v1.2.0/bwenv-linux-amd64", using: :nounzip
      sha256 "d52877263eb60dbbe15493861f37745af608b41052ecf5152fc3a50a26829faf"
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
