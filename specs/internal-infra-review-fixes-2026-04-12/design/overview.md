# Design Overview

## Chosen Approach

Use small ownership-preserving fixes instead of broad refactors:

- OTel env behavior: remove the hidden env reader.
- OTel vocabulary: add one narrowly named neutral vocabulary package, `internal/observability/otelconfig`.
- OTel SDK setup: keep SDK-specific behavior inside `internal/infra/telemetry`.
- HTTP server: add a local receiver guard and inspectable error.
- Metrics: replace status booleans with intent-named telemetry methods.
- Postgres fixture: delete unused transaction sample behavior.
- Route metadata: consolidate the local manual route table and reason metadata.

## Artifact Index

- `component-map.md`: affected packages and stable surfaces.
- `ownership-map.md`: source-of-truth and dependency ownership.
- `sequence.md`: planned implementation and validation sequence.
- `dependency-graph.md`: dependency direction after adding `internal/observability/otelconfig`.

No data model, API contract, migration, or rollout artifact is triggered.

## Design Rationale

The main risk is not algorithmic complexity; it is source-of-truth drift. The design therefore avoids a generic helper package and avoids making `internal/config` depend on `internal/infra/telemetry`.

The new `internal/observability/otelconfig` package is allowed only because the OTel vocabulary is shared by config validation/defaulting and telemetry runtime setup. It should contain vocabulary and pure validation/normalization only. It must not own SDK construction, exporter parsing, metric emission, config loading, or bootstrap flow.

`SetupTracing` should no longer apply its own resource identity defaults (`service`, `dev`, `unknown`). Resource identity values should come from the config snapshot passed by bootstrap; `internal/config` owns defaulting and validation.

## Rejected Options

- Keep `resource.WithFromEnv()` and document OTEL env later: rejected because it preserves the hidden config channel that caused the finding.
- Put OTel constants in `internal/infra/telemetry` and import them from `internal/config`: rejected because it reverses the intended config-to-adapter boundary.
- Put OTel constants in `internal/config` and import them from telemetry: rejected because telemetry should not depend on the config package to build SDK adapters.
- Create `internal/common` or `internal/shared`: rejected because the vocabulary is narrow and does not justify a bucket package.
- Keep the Postgres transaction fixture as an example: rejected because docs say `ping_history` is only a replaceable SQLC fixture, not production transaction policy.

## Readiness

Design is stable enough for implementation planning. If implementation exposes an import cycle from `internal/observability/otelconfig`, reopen design rather than moving constants into `common`.
