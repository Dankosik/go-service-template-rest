# Design Overview: internal/infra Review Fixes

Status: approved

## Chosen Approach

Use small, local repairs in handwritten infra packages. Do not change contracts, generated files, migrations, or app-layer behavior.

The design keeps the review findings grouped by runtime surface:
- Postgres lifecycle and SQLC repository safety.
- HTTP edge tracing and route ownership.
- Telemetry metrics zero-value safety.
- Low-risk readability cleanup in touched files.

The implementation should avoid a broad abstraction pass. Every change should either remove a concrete panic/contract mismatch or make an existing policy explicit.

## Artifact Index

- `component-map.md`: affected files and stable surfaces.
- `sequence.md`: runtime flow before and after the changes.
- `ownership-map.md`: source-of-truth and dependency rules.

No conditional design artifacts are required:
- No `data-model.md`: no schema, migration, or persisted data shape changes.
- No `dependency-graph.md`: no package dependency inversion or new runtime dependency is planned.
- No `design/contracts/`: no OpenAPI, generated, event, or material external contract change.

## Key Design Choices

1. Preserve `MaxIdleConns` as a strict adapter contract by replacing the stat-based hook with stateful limiter accounting.
2. Prefer constructor-time repository dependency validation over allowing typed nils to fail later.
3. Move HTTP tracing to the edge so traces cover the same transport policy surface as logs and request metrics.
4. Keep route-template capture route-local so matched routes still get bounded names and attributes after chi resolution.
5. Make the strict generated `Metrics` method explicitly non-runtime-owned to protect the root `/metrics` exception.
6. Treat zero-value `Metrics` as no-op/not-found rather than a panic surface.

## Readiness Summary

Design is ready for planning and implementation under the assumptions in `spec.md`.

Reopen design if the implementation discovers that pgxpool hook composition needs to preserve existing user-supplied hooks, or if edge-wide tracing creates nested spans or loses route attributes in tests.
