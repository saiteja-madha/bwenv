# ADR 0003: Explicit application discovery

Status: accepted
Date: 2026-10-29

## Decision

Use `bwenv apps` for namespace discovery, including shared by default. Keep `list <app>` for secret records. Namespace discovery and secret listing return different record types and deserve separate commands.

The namespace-summary behavior of `list --all` shipped in v1.2.0. Preserve its output and shared filtering while warning on stderr. Do not silently change a summary command into one that discloses secret values. A future major release may redefine the flag to list secrets across namespaces, with explicit namespace identity and separate shared records.

## Consequences

Existing scripts remain compatible. New callers use `apps`. Namespace validation and sorting belong to the environment package; rendering belongs to the output package.
