# Complete Codebase Audit

Audit date: July 2026
Scope: All 17 source files across cmd/, internal/, root config, CI, install script

---

## Bugs

### B1. `load` silently drops non-matching lines from .env

**File:** `cmd/load.go:42-49`

Lines that don't match `^[A-Za-z_][A-Za-z0-9_]*=` are silently skipped. Common .env patterns that get dropped without warning:
- `export KEY=VALUE` (export prefix)
- `KEY="quoted value"` (matched but quotes become part of the value)
- `KEY=value # inline comment`
- Multiline quoted values (`KEY="line1\nline2"` — continuation line silently lost)

**Impact:** Users may miss secrets they intended to load. No error or warning.

### B2. `load` regex compiled on every invocation

**File:** `cmd/load.go:33`

```go
envRegex := regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*=`)
```

Inside the RunE closure, so it compiles the regex on every `bwenv load` call. Should be a package-level init-time compile.

### B3. `add` KEY=VALUE detection ambiguous with non-standard args

**File:** `cmd/add.go:24`

```go
if len(args) == 2 && strings.Contains(args[1], "=") {
```

If someone runs `bwenv add app` (only 1 arg), cobra's `RangeArgs(2, 3)` catches it before RunE. But `bwenv add app =value` (where key is empty) produces `fullName = "app__=value"` — confusing but not harmful.

### B4. `run` uses `syscall.Exec` which is Unix-only

**File:** `cmd/run.go:51`

```go
return syscall.Exec(binary, command, env)
```

`syscall.Exec` is not supported on Windows. The release workflow builds `bwenv-windows-amd64.exe`, but `bwenv run <app> <cmd>` will fail on Windows. `os/exec.Command` with `Stdin/Stdout/Stderr` passthrough would be cross-platform if needed.

### B5. `filterEnvLines` double-iterates secrets

**File:** `internal/bws/client.go:162-185`

When `includeShared` is true, the function iterates the full secret list twice — once for app prefix, once for shared prefix. Could merge into a single pass.

### B6. `upsertSecret` is O(n²) for bulk operations

**File:** `internal/bws/client.go:62-91, 138-149`

Every `UpsertSecret` calls `ListSecrets()` (full API fetch), then `GetSecretID` iterates the full list to find a match. For `bwenv load` with 100 secrets, this means 100 API calls × 1 full list fetch each = 100 fetches of the same data. No caching.

---

## Code Quality

### Q1. No interface for bws client

`cmd/` package depends directly on the concrete `*bws.Client` global. Cannot unit-test any command without mocking the `bws` binary. Tests for the `cmd` package remain impossible without a refactor toward dependency injection.

### Q2. Global mutable state

**File:** `cmd/root.go:12-17`

```go
var (
    projectID     string
    dryRun        bool
    includeShared bool
    verbose       bool
    bwsClient     *bws.Client
)
```

All commands share mutable package-level globals. Not thread-safe. The global `bwsClient` prevents parallel command execution and makes test setup tedious.

### Q3. No context propagation

No `context.Context` parameter on any `ListSecrets`, `CreateSecret`, `UpdateSecret`, etc. Long-running `load` operations can't be cancelled with Ctrl+C mid-way (though Go runtime handles SIGINT -> process kill, the bws subprocess may be orphaned).

### Q4. `NewClient` doesn't accept Verbose as parameter

**File:** `internal/bws/client.go:25-27`

```go
func NewClient(projectID string) *Client {
    return &Client{ProjectID: projectID}
}
// Callers must remember to set Verbose separately:
bwsClient = bws.NewClient(projectID)
bwsClient.Verbose = verbose
```

Easy to forget. Should be a functional option or constructor parameter.

### Q5. `Execute` calls `os.Exit` directly

**File:** `cmd/root.go:54-56`

```go
func Execute() {
    if err := rootCmd.Execute(); err != nil {
        os.Exit(1)
    }
}
```

`os.Exit` skips deferred cleanup. Cobra's `Execute()` already calls `os.Exit` internally when commands return errors. The deferred `rootCmd.Execute()` result handles this. Redundant wrapper.

### Q6. Error wrapping inconsistencies

Some errors use `%w` (wrapping, allows `errors.Is`/`errors.As`), some don't:
- `client.go:67,72` — uses `%w` ✓
- `client.go:113,132` — uses `%w` with name ✓
- `cmd/root.go:42` — uses string formatting, no wrap
- `cmd/run.go:48` — uses `%s`, no wrap

### Q7. Makefile convenience gaps

**File:** `Makefile`

- No `make fmt` or `make vet` targets
- `make install` copies to `/usr/local/bin` without sudo (fails for non-root)
- No `make run` integration test target

### Q8. `install.sh` binary name always `bwenv` on install path

**File:** `install.sh:33`

```bash
INSTALL_PATH="${INSTALL_DIR}/bwenv"
```

Regardless of the downloaded binary name (`bwenv-darwin-amd64`), the installed file is always `bwenv`. If someone installs from a different source (e.g., building from source), a stale binary could persist. Low risk.

### Q9. Version string absent

There is no `--version` flag, no `version` command, and no version constant in the code. The binary's version is whatever was last built. The release workflow tags `v*` but the binary has no embedded version.

---

## Performance

### P1. Full secret list fetched on every operation

Every command lists ALL project secrets via the `bws` API. For projects with hundreds or thousands of secrets, every `list`, `pull`, `run`, and element within `load` incurs a full fetch. A cache with TTL would dramatically improve bulk operations.

### P2. No pagination

`bws` API may paginate results for large projects. The current code assumes all secrets fit in a single response (`bws secret list <projectId> -o json`). If pagination kicks in, only the first page is returned.

---

## Enhancements

### E1. Embedded version string

Add `--version` / `version` subcommand using `ldflags`:
```sh
go build -ldflags="-X main.Version=$(git describe --tags)" -o bwenv
```

### E2. `--env` flag for `add` command

Read value from an environment variable instead of CLI arg:
```sh
bwenv add app KEY --env MY_SECRET_ENV_VAR
```

### E3. `run` should pass through exit code

If the child process exits non-zero, `bwenv run` should propagate the exit code instead of always exiting 0 or 1.

### E4. Progress output for `load`

When loading a large `.env` file, a progress counter or periodic status line would help:
```
loaded 50/200 secrets...
```

### E5. Overwrite confirmation or `--force` flag

`add` silently overwrites existing secrets. A confirmation prompt or `--force` flag would prevent accidental overwrites.

### E6. `pull` format options

`bwenv pull app --format json` or `--format yaml` for integration with tools that consume structured config.

### E7. `--shared` prefix as a configurable option

The `shared__` prefix is hardcoded. A config option or flag to set a custom shared prefix would increase flexibility.

### E8. macOS default install path

`install.sh` always uses `/usr/local/bin/bwenv`. On Apple Silicon Macs, the convention is `/opt/homebrew/bin/`. The script could detect ARM macOS and default to the Homebrew path.

### E9. Makefile `install` should try sudo

```makefile
install: build
	cp bwenv /usr/local/bin/  # fails for non-root
```

Could be:
```makefile
install: build
	install -d $(DESTDIR)/usr/local/bin
	install -m 755 bwenv $(DESTDIR)/usr/local/bin/
```

---

## Just My Opinion (Lowest Priority)

### O1. `.env` file format support

If the project aims to replace `.env` files entirely, supporting the full dotenv spec (quoted values, multiline, inline comments, export) would be necessary.

### O2. `Go 1.26.5` bundled tarball

A 40MB Go tarball sits in the repo root. It's gitignored but still pollutes the working tree. If it's for CI, it should be in a CI-specific setup script, not the repo root.

### O3. Homebrew formula placeholders

`Formula/bwenv.rb` contains hardcoded placeholder strings (`VERSION`, `PLACEHOLDER_*`). It requires manual editing per release.

---

## Summary by Priority

| Priority | Item | Type | Effort |
|----------|------|------|--------|
| High | B1 — .env loading silently drops common patterns | Bug | Small |
| High | B4 — `syscall.Exec` is Unix-only, Windows build broken at runtime | Bug | Medium |
| High | P2 — No pagination support for large projects | Bug | Medium |
| Medium | B2 — Regex compiled on every `load` call | Perf | Trivial |
| Medium | B5 — FilterEnvLines double-iteration | Perf | Trivial |
| Medium | B6 — UpsertSecret O(n²), no cache | Perf | Medium |
| Medium | Q1 — No client interface, cmd/ untestable | Quality | Medium |
| Medium | Q3 — No context propagation | Quality | Medium |
| Medium | E1 — No embedded version | Enhancement | Trivial |
| Medium | E3 — Child exit code not propagated | Bug | Trivial |
| Low | Q4 — Verbose flag easy to forget | Quality | Trivial |
| Low | Q6 — Error wrapping inconsistencies | Quality | Trivial |
| Low | Q7 — Makefile missing fmt/vet targets | Quality | Trivial |
| Low | E2 — `--env` flag for `add` | Enhancement | Small |
| Low | E4 — Progress output for `load` | Enhancement | Small |
| Low | E5 — Overwrite confirmation | Enhancement | Small |
