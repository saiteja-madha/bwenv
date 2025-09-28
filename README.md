# bwenv

A tiny helper for **Bitwarden Secrets Manager** (`bws`) to manage app-specific env vars using a prefix convention.

- Store secrets in a single project (e.g. `dev-sandbox`) as `<app>__KEY`.
- Push (`add`/`load`) and pull (`pull`/`run`) without writing a `.env` file.

## Install

```bash
curl -fsSL -o /usr/local/bin/bwenv https://raw.githubusercontent.com/saiteja-madha/bwenv/main/bwenv
chmod +x /usr/local/bin/bwenv
```

## Requirements

- A Bitwarden account with [Secrets Manager](https://bitwarden.com/help/article/secrets-manager/) enabled
- A Bitwarden organization with a project to store your secrets
- The [Secrets Manager CLI](https://bitwarden.com/help/article/secrets-manager-cli/) (`bws`) installed and configured
- Set your env vars (e.g. in ~/.zshrc):

```bash
export BWS_ACCESS_TOKEN="your_machine_access_token"
export BWS_PROJECT_ID="your_dev_project_uuid"
```

## Usage

```bash
bwenv add  <app> KEY VALUE
bwenv add  <app> KEY=VALUE
bwenv load <app> path/to/.env [--dry-run]
bwenv list <app>
bwenv pull <app> [--include-shared]
bwenv run  <app> [--include-shared] <command> [args...]
```

## Examples

Seed from an existing .env:

```bash
bwenv load notes-api .env
```

Run Flask without creating a .env:

```bash
bwenv run notes-api flask run
```

Include shared variables (from shared\_\_KEY secrets):

```bash
bwenv run notes-api --include-shared flask run
```

List keys for one app:

```bash
bwenv list notes-api
```

Print env lines that would be injected:

```bash
bwenv pull notes-api
```

## Why not just bws run?

- bws run injects all secrets in a project.
- bwenv lets you filter by app using a simple prefix, so you can keep everything in one project on the Free plan and still get per-app envs.
