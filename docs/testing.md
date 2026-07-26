# Testing

The default verification suite is offline and does not require a Bitwarden
account, access token, project, or installed `bws` binary:

```bash
make verify
make test-race
```

## Live end-to-end smoke test

`scripts/e2e.sh` exercises the installed `bwenv` and official `bws` binaries
against a real Bitwarden Secrets Manager project. It is intentionally opt-in
and is not run by pull-request CI.

Use a disposable, otherwise empty project. The script creates and deletes
secrets and therefore requires read/write access:

```bash
export BWS_PROJECT_ID="disposable-project-id"
export BWS_ACCESS_TOKEN="..."
make build
PATH="$PWD:$PATH" ./scripts/e2e.sh
```

Never paste or commit the access token. The script uses a randomized app
namespace and randomized shared key names. Cleanup selects only that namespace
and those exact shared keys; it does not delete generic shared variables.

If cleanup reports a warning, inspect the disposable project and remove only
the printed run namespace before reusing it.
