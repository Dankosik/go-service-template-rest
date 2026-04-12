# Design Overview

## Chosen Approach

The implementation should be a small compatibility-preserving repair:
- Bind `postgres.healthcheck_timeout` to runtime Postgres readiness in bootstrap.
- Clarify Redis/Mongo reserved extension keys in docs instead of deleting existing config fields.
- Keep config readability cleanups local to `internal/config`.

The main design choice is where to enforce the Postgres runtime readiness cap. The selected option is bootstrap-level wrapping because bootstrap owns runtime policy binding and can cap the registered readiness probe without changing global `postgres.Pool.Check(ctx)` semantics.

Rejected option:
- Store `HealthcheckTimeout` inside `postgres.Pool` and apply it in every `Pool.Check`. This is workable, but it broadens adapter semantics and makes direct `Pool.Check` callers inherit a config-derived cap even when a caller already supplies a deadline. Use this only if a future task wants the Postgres adapter to own all healthcheck timeout policy.

## Artifact Index

- `component-map.md`: affected packages and stable surfaces.
- `sequence.md`: runtime readiness flow before and after the change.
- `ownership-map.md`: source-of-truth and dependency-direction rules for implementation.

No conditional data-model, dependency-graph, API contract, test-plan, or rollout artifact is required.

## Readiness Summary

Design status: approved for planning.

Planning can proceed because:
- The compatibility stance for Redis/Mongo keys is explicit.
- The Postgres runtime readiness budget owner is explicit.
- No API, schema, migration, adapter, or generated-code changes are required.

Reopen design if implementation shows bootstrap cannot wrap the Postgres probe without changing `internal/app/health.Service` or `postgres.Pool.Check` semantics.
