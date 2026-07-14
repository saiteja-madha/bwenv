# ADR 0002: Flat, bws-aligned environment commands

Status: accepted  
Date: 2026-07-14

## Context

Bitwarden’s free Secrets Manager limits projects, while self-hosters often operate many applications within the same development or homelab environment. Official `bws` injects an entire project and does not strip application prefixes.

An intermediate design proposed `bwenv env create`, but `env` repeats the product name and adds no resource disambiguation. There are no compatibility obligations for the earlier prototype command surface.

## Decision

Use flat commands: `create`, `import`, `list`, `get`, `edit`, `delete`, `export`, and `run`. Their naming follows official `bws secret` operations where the semantics match. `import` and `export` are bwenv-specific dotenv workflows.

Do not wrap `bws project` or `bws config`. Users manage those directly with official `bws`; bwenv accepts and forwards the official global authentication and routing options.

Store secrets as `<app>__KEY`, reserve `shared__KEY`, and make app-specific values override shared values. Mutations never fall back to shared. Duplicate stored keys are errors.

## Consequences

The CLI stays small and reads naturally: `bwenv create immich TOKEN ...`. Filtering requires bwenv to request JSON from `bws` and render normalized output itself. This introduces a small presentation layer but avoids reimplementing authentication or Secrets Manager APIs.
