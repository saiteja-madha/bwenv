# Releasing

Release tags matching `v*` drive the release workflow. The tagged commit is verified, cross-compiled for every supported target, smoke-tested, and published with a SHA-256 manifest.

## Supply-chain verification

The workflow creates signed GitHub build-provenance attestations for every released binary. Users can verify a downloaded artifact with:

```bash
gh attestation verify ./bwenv-darwin-arm64 --repo saiteja-madha/bwenv
```

The installer continues to verify the release checksum before installing. Checksums detect corruption; provenance additionally connects an artifact to its repository, commit, and GitHub Actions workflow.

## Homebrew

`Formula/bwenv.rb` installs stable release binaries and retains `--HEAD` support for source builds. After a release is published, the workflow regenerates the formula from that release’s checksum manifest and opens a pull request against `main`.

Repository Actions settings must permit GitHub Actions to create pull requests. An optional `BWENV_RELEASE_PAT` repository secret can be supplied so formula pull requests trigger the normal pull-request workflows; otherwise the workflow uses `GITHUB_TOKEN`.

## Official bws compatibility

The weekly compatibility workflow installs the latest published `bws` crate and checks every official global option and wrapped `secret`/`run` command used by bwenv. It requires no Bitwarden token or project. Run the same contract locally against any installed binary with:

```bash
make compat
```

Compatibility failures indicate that the official CLI surface changed and should be reviewed before the next bwenv release.
