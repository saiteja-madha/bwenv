# bwenv documentation

This directory is the canonical product and engineering reference for humans and coding agents. The root [README](../README.md) is the concise user entry point.

## Start here

| Document | Audience | Authority |
|---|---|---|
| [CLI reference](reference/cli.md) | Users, operators, support | Commands, flags, outputs, validation, and exit behavior |
| [Architecture](architecture.md) | Maintainers, reviewers, coding agents | Boundaries, data flow, security invariants, and test seams |
| [Command-surface decision](decisions/0002-bws-aligned-cli.md) | Maintainers | Why bwenv uses flat environment commands and excludes project/config wrappers |
| [Contributing](../CONTRIBUTING.md) | Contributors | Local workflow and pull-request expectations |
| [Fixed issues](decisions/0001-fixed-issues.md) | Maintainers | Historical record of the initial audit; superseded where noted |

## Documentation rules

- Update these references in the same change as user-visible behavior.
- Describe observable behavior and invariants; avoid copying implementation details that can drift.
- Keep secrets and real access tokens out of examples and fixtures.
- Add new documents to this index and run `scripts/check-doc-links.sh`.
