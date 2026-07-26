# Contributing to bwenv

Thanks for helping make secret-backed homelabs easier to operate. Small, focused changes with tests and clear behavior are especially welcome.

## Development setup

You need Go as declared in `go.mod`. The real `bws` binary and a Bitwarden token are not needed for unit tests.

```bash
git clone https://github.com/saiteja-madha/bwenv.git
cd bwenv
go mod download
make verify
make test-race
```

For a manual integration test, install the official `bws` CLI and follow the
[live E2E instructions](docs/testing.md) using a disposable project. Never
commit access tokens, fixture secrets, exported environments, or local config
files.

## Making a change

1. Open an issue for substantial behavioral or command-interface changes.
2. Create a feature branch from `main` (e.g., `fix/description` or `feat/description`).
3. Keep Bitwarden subprocess logic in `internal/bws` and environment naming/merge behavior in `internal/environment`.
4. Add or update tests for successful behavior, validation failures, subprocess failures, and secret-redaction boundaries.
5. Update the canonical documents under `docs/` when behavior changes.
6. Run `make verify`, `make test-race`, and `make build-all` before opening a pull request.

Use `gofmt`; do not hand-format Go code. Error messages must not include secret values or access tokens. Verbose subprocess logs must mask both.

## Tests

Unit tests use injected clients and local child processes. They must be deterministic, cross-platform where practical, and must never contact Bitwarden. Tests that require a real service belong in a separately documented, opt-in integration suite.

```bash
make test       # unit and command tests
make test-race  # race detector
make vet        # Go static analysis
make lint       # golangci-lint, when installed
make shellcheck # all installer and maintenance shell scripts
make compat     # compare an installed bws with the wrapped CLI contract
```

`make compat` requires an installed official `bws` binary but does not require a token, project, or network call to Bitwarden. Offline compatibility-script fixtures run as part of `make verify`.

## Documentation

`docs/` is also the source of truth for coding agents. Prefer precise behavior, commands, precedence rules, and failure modes over marketing language. Update `docs/README.md` whenever adding or removing a document, then run `scripts/check-doc-links.sh`.

## Pull requests

Push your feature branch to origin and open a pull request against `main`. Never push directly to `main`.

Explain the user-visible outcome and why the chosen approach fits bwenv’s narrow scope. Include test evidence. Keep unrelated refactors out of the same pull request, and note any security or compatibility implications explicitly.

By contributing, you agree that your contribution is licensed under the project’s [MIT License](LICENSE).
