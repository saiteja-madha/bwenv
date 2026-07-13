# bwenv — AGENTS.md

## Project

Go CLI wrapper around Bitwarden Secrets Manager CLI (`bws`). Stores secrets as `<app>__KEY` in a single Bitwarden project, lets you pull/run per-app env vars without `.env` files.

- **Module:** `bwenv`, Go 1.24.3, dep: `github.com/spf13/cobra`
- **Entry:** `main.go` → `cmd.Execute()` → cobra commands in `cmd/`
- **Bitwarden layer:** `internal/bws/client.go` — shells out to `bws` binary via `exec.Command`
- **Runtime deps (host):** `bws` CLI, `jq`, `BWS_ACCESS_TOKEN` env var

## Commands

| Command | Purpose |
|---------|---------|
| `add <app> KEY VALUE` or `KEY=VALUE` | Upsert secret |
| `load <app> <file>` | Bulk upsert from .env file |
| `list <app>` | List key names (prefix stripped) |
| `pull <app>` | Print `KEY=VALUE` lines |
| `run <app> <cmd> [args...]` | Exec command with secrets as env vars |
| `completion bash\|zsh\|fish` | Generate shell completions |

Global flags: `--project-id`, `--dry-run`, `--include-shared`, `--verbose`/`-v`

## Architecture notes

- All Bitwarden I/O goes through `internal/bws/client.go` which calls `bws` CLI subprocesses.
- `ListSecrets()` fetches all project secrets; filtering by app prefix happens in-memory via `FilterEnvLines`/`FilterAppKeys`.
- `UpdateSecret` uses `--value=<val>` syntax (`--value=%s`) — not `--value <val>` — to prevent clap flag misinterpretation when values start with `-`.
- `validateValue()` rejects null bytes (`\x00`) which would truncate C strings in `execve`.
- Empty secret values are valid (fixed in `cmd/add.go`).
- Values are always passed as `bws` CLI args — visible in process lists (`ps`). This is an inherent bws limitation (no stdin support for values).

## Testing

```sh
go test ./...           # all tests
go test -v ./internal/...  # bws client tests only
```

Tests exist only in `internal/bws/client_test.go` (unit tests for `validateValue`, `FilterEnvLines`, `FilterAppKeys`). `cmd/` package has no tests — cobra commands depend on the global `bwsClient` and require integration-style mocking.

## Build & CI

```sh
make build        # go build -o bwenv
make build-all    # cross-compile all platforms to dist/
make test         # go test ./...
make clean        # rm dist/ bwenv
```

CI (`.github/workflows/ci.yml`): build + test on ubuntu/macos/windows for push/PR to `main`.
Release (`.github/workflows/release.yml`): tag `v*` triggers multi-arch build + GitHub release.
Lint (`.golangci.yml`): gofmt, govet, errcheck, staticcheck, gosimple, ineffassign, unused, revive.

## Secret naming convention

```
<app>__KEY           → myapp__DATABASE_URL
shared__KEY          → shared__LOG_LEVEL
```

`--include-shared` injects both `<app>__*` and `shared__*` secrets.

## Dev commands

```sh
make run ARGS="list myapp"   # build + run with args
go build -o bwenv && ./bwenv --help
```

## Documentation

Keep docs in `docs/`. Structure:

```
docs/
├── README.md          # index, points to entry docs
├── decisions/         # ADR-style design records
└── guides/            # step-by-step how-tos (when needed)
```

Rules:
- Prefer small modular files over one large document
- Update existing docs before creating new ones
- Remove stale or redundant documentation
- Keep docs close to the code they describe when practical
- Verify internal links are valid after changes
- docs/README.md is the index — explain what exists, where, and who it is for
