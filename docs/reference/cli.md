# CLI reference

## Model

`bwenv` stores an environment key as `<app>__<KEY>` in one Bitwarden Secrets Manager project. The project is selected by `--project-id` or `BWS_PROJECT_ID`.

App names use letters, numbers, `.`, `_`, or `-`, must start with a letter or number, and cannot contain `__`. Keys must match `[A-Za-z_][A-Za-z0-9_]*`.

`shared__KEY` is the reserved shared namespace. When `--include-shared` is used, shared keys load first and app-specific keys override them. Every effective key is emitted once.

## Global options

| Option | Environment fallback | Meaning |
|---|---|---|
| `--project-id` | `BWS_PROJECT_ID` | Project containing the app-prefixed secrets |
| `-o, --output` | — | `json`, `yaml`, `env`, `table`, `tsv`, or `none` |
| `-c, --color` | official `NO_COLOR` conventions | `yes`, `no`, or `auto`; auto detects terminals and honors `NO_COLOR`/`CLICOLOR` |
| `-t, --access-token` | `BWS_ACCESS_TOKEN` | Official `bws` machine-account token |
| `-f, --config-file` | `BWS_CONFIG_FILE` | Official `bws` config file |
| `-p, --profile` | `BWS_PROFILE` | Official `bws` profile |
| `-u, --server-url` | `BWS_SERVER_URL` | Override the configured Bitwarden server |
| `--verbose` | — | Print masked `bws` subprocess commands to stderr |
| `-h, --help` | — | Help |
| `-V, --version` | — | bwenv version |

Authentication and routing options are forwarded to official `bws`. An explicitly supplied access token is placed in the subprocess environment instead of duplicated in the `bws` argument list. Internally, bwenv requests JSON without color so it can safely filter and merge records, then renders the normalized result.

`run --uuids-as-keynames` also honors the official `BWS_UUIDS_AS_KEYNAMES` environment variable.

## Commands

### create

```text
bwenv create <app> <key> <value> [--note NOTE] [--dry-run]
```

Creates `<app>__<key>`. It refuses to overwrite or create a second secret with the same full key. Empty values are valid.

### import

```text
bwenv import <app> <file|-> [--dry-run]
```

Parses dotenv syntax and upserts every key. `-` reads stdin. The entire input, all key names, and every value are validated before the project is listed or writes begin. Operations are sorted by key and use one initial project listing. Existing secrets whose values are already equal are reported as unchanged without issuing an edit. Remote writes are not transactional: on failure, the error reports how many prior operations completed.

If a dotenv file defines the same key more than once, the final definition wins.

Import summaries support all global output formats. Table and TSV report one create, update, or unchanged action per key; env output reports `APP`, `CREATED`, `UPDATED`, `UNCHANGED`, and `DRY_RUN`.

### list

```text
bwenv list <app> [--include-shared]
```

Lists normalized secret records in deterministic key order. Shared/app collisions resolve to the app record.

### get

```text
bwenv get <app> <key> [--include-shared]
```

Returns exactly one normalized record. With shared enabled, an app record is preferred and shared is only a fallback. Missing and duplicate full keys are errors.

### edit

```text
bwenv edit <app> <key> [--key NEW_KEY] [--value VALUE] [--note NOTE] [--dry-run]
```

At least one editable field is required. `--value=` stores an empty value; `--note=` clears a note. Renames stay within the same app and refuse collisions.

### delete

```text
bwenv delete <app> <key>... [--dry-run]
```

Resolves every key before deleting anything, then uses official `bws` multi-delete. Shared fallback is intentionally not used for mutations.

### export

```text
bwenv export <app> [--include-shared]
```

Writes the effective environment to stdout. Its default format is lossless dotenv syntax suitable for a later `bwenv import`; an explicit global `--output` selects another supported representation.

### run

```text
bwenv run <app> [--include-shared] [--shell SHELL] [--no-inherit-env]
                [--uuids-as-keynames] -- <command>
```

Uses `sh` on macOS/Linux and PowerShell on Windows unless `--shell` is supplied. With no command arguments, it reads a command from piped stdin; interactive stdin without a command fails immediately. Command and shell validation happens before secrets are fetched. The child inherits the current environment by default, except every case variant of `BWS_ACCESS_TOKEN` is always removed, including a selected secret with that key. `--no-inherit-env` retains only `PATH` and required Windows shell variables before adding secrets.

Separately supplied command arguments are quoted for the selected shell so spaces and quotes retain their argument boundaries. A single command string is passed to the shell verbatim; use that form for pipelines, redirections, variable expansion, and compound commands:

```bash
bwenv run immich -- printf '<%s>\n' 'value with spaces'
bwenv run immich -- 'docker compose up -d && docker compose logs -f'
```

The child’s exit code becomes bwenv’s exit code. Only run trusted commands.

### completion and version

```text
bwenv completion <bash|zsh|fish|powershell>
bwenv version
```

Neither requires `bws`, authentication, or project configuration.

## Output record

JSON and YAML retain official secret fields such as `id`, `projectId`, `value`, `note`, and timestamps. `key` is prefix-stripped and `source` is `app` or `shared`. Table and TSV output contain ID, key, value, source, and creation date. `none` suppresses successful output.

## Failure and security behavior

- Official `bws` stderr is preserved on successful and failed calls, and nonzero exit codes are preserved.
- Access tokens, values, and notes are masked in verbose subprocess logs.
- `--access-token` remains visible in the original bwenv process arguments when used; prefer `BWS_ACCESS_TOKEN`. bwenv does not copy that token into the child `bws` argument list.
- Null bytes are rejected before remote work begins because operating-system process arguments and environments cannot carry them.
- Values passed to `create` and `edit` remain visible in local process listings while the official `bws` subprocess runs.
- Duplicate stored full keys are errors; bwenv never selects one arbitrarily.
