<div align="center">
  <img src="docs/assets/logo.svg" width="104" alt="bwenv logo">
  <h1>bwenv</h1>
  <p><strong>One Bitwarden project. Many clean, app-scoped environments.</strong></p>
  <p>
    <a href="https://github.com/saiteja-madha/bwenv/actions/workflows/ci.yml"><img alt="CI" src="https://github.com/saiteja-madha/bwenv/actions/workflows/ci.yml/badge.svg"></a>
    <a href="https://github.com/saiteja-madha/bwenv/releases"><img alt="Release" src="https://img.shields.io/github/v/release/saiteja-madha/bwenv?display_name=tag"></a>
    <a href="LICENSE"><img alt="MIT License" src="https://img.shields.io/badge/license-MIT-22c55e"></a>
  </p>
</div>

![A homelab server routing app environments into a secure vault](docs/assets/bwenv-hero.png)

`bwenv` is a small wrapper around the official [Bitwarden Secrets Manager CLI](https://bitwarden.com/help/secrets-manager-cli/). It lets homelabbers and self-hosters keep many application environments inside one Bitwarden project without scattering `.env` files across servers.

```text
immich__DB_PASSWORD       → only Immich
paperless__DB_PASSWORD    → only Paperless-ngx
shared__TZ                → both, with --include-shared
```

## Why bwenv?

- Keep app secrets separated by a simple `<app>__KEY` convention.
- Import and export familiar dotenv data.
- Run Docker Compose, systemd helpers, scripts, and CLIs with secrets in memory.
- Share common values while allowing app-specific overrides.
- Retain official `bws` authentication, profiles, server routing, and output formats.

## Quick start

Install the official `bws` CLI, create a machine-account access token, and choose one project for your environment.

```bash
export BWS_ACCESS_TOKEN="your-machine-account-token"
export BWS_PROJECT_ID="your-project-uuid"

bwenv create immich DB_PASSWORD 'correct-horse-battery-staple'
bwenv create shared TZ 'America/Los_Angeles'

bwenv list immich --include-shared
bwenv run immich --include-shared -- docker compose up -d
```

App values override shared values with the same key. `bwenv run` removes `BWS_ACCESS_TOKEN` before starting the child process.

## Install

### Installer (macOS and Linux)

The installer downloads the matching release and requires a valid SHA-256 checksum.

```bash
curl -fsSL https://raw.githubusercontent.com/saiteja-madha/bwenv/main/install.sh | bash
```

Set `INSTALL_DIR` to choose another destination or `BWENV_VERSION` to pin a release:

```bash
curl -fsSL https://raw.githubusercontent.com/saiteja-madha/bwenv/main/install.sh |
  INSTALL_DIR="$HOME/bin" BWENV_VERSION=v1.0.0 bash
```

### Homebrew HEAD

Until the first stable formula is published, build the current main branch through the included formula:

```bash
brew tap saiteja-madha/bwenv https://github.com/saiteja-madha/bwenv
brew install --HEAD bwenv
```

### Build from source

```bash
git clone https://github.com/saiteja-madha/bwenv.git
cd bwenv
make build
sudo make install
```

## Commands

| Command | Purpose |
|---|---|
| `create <app> <key> <value>` | Create one variable; refuses to overwrite |
| `import <app> <file\|->` | Upsert variables from dotenv or stdin |
| `list <app>` | List normalized app secrets |
| `get <app> <key>` | Get one app key, optionally falling back to shared |
| `edit <app> <key>` | Change a key, value, or note |
| `delete <app> <key>...` | Delete one or more resolved keys |
| `run <app> -- <command>` | Run through a shell with the effective environment |
| `completion <shell>` | Generate shell completions with Cobra |
| `version` | Print build metadata |

Read the [complete CLI reference](docs/reference/cli.md) for flags, output formats, exit behavior, shared precedence, and examples.

## Security model

`bwenv` never stores your access token or fetched secret values. It invokes the official `bws` binary and keeps transformed environments in process memory. Every create or edit—including operations initiated by `import`—must ultimately pass the value to `bws` as a process argument, which can be visible to local process inspection tools. Restrict local host access and remove temporary source files promptly.

Only run trusted commands with `bwenv run`: the child receives the selected secrets and has your normal operating-system permissions.

## Project docs

- [Documentation index](docs/README.md)
- [Architecture and security boundaries](docs/architecture.md)
- [Contributing](CONTRIBUTING.md)
- [License](LICENSE)

The hero artwork is original to this project and generated for bwenv.
