# Fixed Issues and Design Decisions

> Historical record. The command and client architecture described here was superseded by [ADR 0002](0002-bws-aligned-cli.md). Consult the current [CLI reference](../reference/cli.md) and [architecture](../architecture.md) for authoritative behavior.

Summary of fixes applied during the initial codebase audit (July 2026).

## Issue \#2 — Special characters, null bytes, and empty values

**Problem:** Secret values with null bytes silently truncated via `execve`. Values starting with `-` misinterpreted by `bws`'s clap parser. Empty values mis-rejected.

**Fix:**
- Added `validateValue()` in `internal/bws/client.go` — rejects null bytes with a clear error
- Changed `UpdateSecret` from `--value <val>` to `--value=<val>` — prevents clap flag misinterpretation
- Removed `value == ""` guard in `cmd/add.go` — empty env vars are valid

**Trade-off:** Values remain visible in process lists (`ps`). `bws` v2.x has no stdin path for values.

## Issue \#12 — Security hardening

### Audit findings (July 2026)

**HIGH: install.sh had no integrity verification.** Binary downloaded and installed to `/usr/local/bin` without checksum check. Fix: download `checksums.txt` from the release, verify `sha256sum` before install, abort on mismatch.

**MEDIUM: install.sh curl without `--fail`.** HTTP error pages (429, 404) would be written as the binary. Fix: added `-f` flag to all curl calls.

**MEDIUM: install.sh parsed GitHub API with `grep | cut`.** Fragile JSON parsing. Fix: use `jq` (already a runtime dependency of bwenv).

**MEDIUM: install.sh no sudo fallback.** Writing to `/usr/local/bin` silently failed without root. Fix: check write permission, offer `sudo` escalation.

**LOW: BWS_ACCESS_TOKEN format not validated.** Only checked emptiness. Fix: validate `<client_id>.<client_secret>` format on load.

### Previously addressed

Error messages use secret name (`app__KEY`) not value. No value leaks exist. The process-list visibility is an inherent `bws` limitation documented in AGENTS.md.

## Issue \#1 — GitHub Actions CI/CD

- CI builds + tests on ubuntu/macos/windows for push/PR to `main`
- Release builds all platforms + `sha256sum` on `v*` tag push
- Linting via golangci-lint (gofmt, govet, staticcheck, revive, etc.)
- CodeQL security analysis weekly

## Issue \#5 — Tests

Extracted `FilterEnvLines`/`FilterAppKeys` as standalone functions for unit testing without a `bws` binary. Tests cover validation, filtering, and key extraction.

## Issue \#8 — Shell completion

`bwenv completion [bash|zsh|fish]` using cobra's built-in generators.

## Issue \#6 — Verbose mode

`--verbose`/`-v` flag logs `bws` commands with secret values masked (`***`).

## Design Decisions

### Value masking in verbose logging

`logCmd()` accepts arg indices to mask. Create masks index 4 (value), Edit masks index 5 (`--value=...`). The arg is replaced with `***` before printing.

### Testability

`FilterEnvLines` and `FilterAppKeys` are exported package-level functions operating on `[]Secret`. Tests construct mock secret lists directly — no `bws` binary needed.
