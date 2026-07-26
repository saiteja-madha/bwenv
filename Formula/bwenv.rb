class Bwenv < Formula
  desc "App-scoped environments backed by Bitwarden Secrets Manager"
  homepage "https://github.com/saiteja-madha/bwenv"
  version "1.1.0"
  license "MIT"
  head "https://github.com/saiteja-madha/bwenv.git", branch: "main" do
    depends_on "go" => :build
  end

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/saiteja-madha/bwenv/releases/download/v1.1.0/bwenv-darwin-arm64", using: :nounzip
      sha256 "7a128804c3dae257e5256f157ed1d41125f9145bd82ccade3f6cf294aefb582c"
    else
      url "https://github.com/saiteja-madha/bwenv/releases/download/v1.1.0/bwenv-darwin-amd64", using: :nounzip
      sha256 "50fc072a6e6f5f36c407e894704e21e9aa67fcde9fb03ec72475153c68045072"
    end
  end

  on_linux do
    if Hardware::CPU.arm? && Hardware::CPU.is_64_bit?
      url "https://github.com/saiteja-madha/bwenv/releases/download/v1.1.0/bwenv-linux-arm64", using: :nounzip
      sha256 "7bab71f9745533f28c4d8ecf260efd298f4dd928ad7f10c2d050d3771ed5c0d9"
    else
      url "https://github.com/saiteja-madha/bwenv/releases/download/v1.1.0/bwenv-linux-amd64", using: :nounzip
      sha256 "92dc1ada5144884db6a032f8fec1b411044159ab42ca844e668c05aee3427719"
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
