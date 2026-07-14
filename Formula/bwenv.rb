class Bwenv < Formula
  desc "App-scoped environments backed by Bitwarden Secrets Manager"
  homepage "https://github.com/saiteja-madha/bwenv"
  license "MIT"
  head "https://github.com/saiteja-madha/bwenv.git", branch: "main"

  depends_on "go" => :build

  def install
    ldflags = %W[
      -s -w
      -X bwenv/cmd.Version=#{version}
      -X bwenv/cmd.Commit=#{version}
      -X bwenv/cmd.Date=homebrew
    ]
    system "go", "build", *std_go_args(ldflags: ldflags), "."
  end

  test do
    assert_match "bwenv #{version}", shell_output("#{bin}/bwenv version")
    assert_match "create", shell_output("#{bin}/bwenv --help")
  end
end
