# bwenv

A tiny helper for **Bitwarden Secrets Manager** (`bws`) to manage app-specific env vars using a prefix convention.

- Store secrets in a single project (e.g. `dev-sandbox`) as `<app>__KEY`.
- Push (`add`/`load`) and pull (`pull`/`run`) without writing a `.env` file.

## Install

```bash
curl -fsSL -o /usr/local/bin/bwenv https://raw.githubusercontent.com/saiteja-madha/bwenv/main/bwenv
chmod +x /usr/local/bin/bwenv
```
