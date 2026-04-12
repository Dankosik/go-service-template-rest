# Internal Infra Review Fixes Spec

## Context

A read-only review of `internal/infra` found several maintainability and correctness risks:

- OTel resource attributes are read directly from environment via `resource.WithFromEnv()`, bypassing the repository config snapshot.
- Trace sampler ratio normalization accepts `NaN` and can pass it into the OTel SDK.
- Exported HTTP `Server` methods panic on nil receiver or zero-value use.
- Startup dependency metric call sites pass raw boolean status values through branching bootstrap code.
- The replaceable `ping_history` SQLC fixture contains an unused transactional workflow.
- Telemetry init failure reason strings are duplicated between bootstrap and telemetry normalization.
- Manual root route exception metadata is split between route declarations and a reason map; this is low-risk and test-guarded.

The user asked for the complete pre-implementation context to be written down, with implementation deferred to a later session.

## Scope / Non-goals

In scope:

- Fix config ownership drift for OTel resource attributes.
- Consolidate OTel sampler/protocol vocabulary without introducing a generic helper bucket or reversing dependency direction.
- Harden trace sampler runtime validation against non-finite ratios.
- Make exported HTTP server misuse return inspectable errors instead of panics.
- Replace raw startup dependency metric booleans with intent-named calls.
- Remove the unused Postgres transaction fixture and related test-only fake transaction scaffolding.
- Consolidate telemetry init failure reason literals.
- Update repository architecture/structure docs for the new `internal/observability/otelconfig` package.
- Consolidate manual root route exception reason metadata into one local source of truth.

Out of scope:

- Adding a new OTEL standard environment-variable configuration channel.
- Changing public HTTP API behavior, OpenAPI contracts, migrations, or generated code.
- Adding new Postgres transaction abstractions or generic transaction helpers.
- Refactoring router middleware order, route labels, or request/response problem semantics beyond the manual root route metadata cleanup.
- Changing startup dependency semantics or metric label names.
- Rewriting repository documentation beyond the minimal ownership note for `internal/observability/otelconfig`.

## Constraints

- `internal/config` owns typed runtime config defaults, snapshot construction, validation, and `APP__...` precedence.
- `internal/infra/telemetry` owns OTel SDK setup and Prometheus metric adapters.
- `cmd/service/internal/bootstrap` owns service composition and may depend on both config and telemetry.
- Do not make `internal/config` import `internal/infra/telemetry`.
- Do not create a generic `common`, `shared`, or `util` package.
- Keep generated `internal/infra/postgres/sqlcgen/*` untouched.
- Preserve low-cardinality metric label values.
- Preserve `/metrics` as the documented manual root-router exception.

## Decisions

1. Remove `resource.WithFromEnv()` from telemetry resource construction. The approved behavior is explicit config-only resource attributes for `service.name`, `service.version`, and `deployment.environment.name`. A future OTEL env channel would require a separate config/source-policy decision.
2. Introduce `internal/observability/otelconfig` as a tiny neutral owner for OTel config vocabulary. It may contain sampler/protocol constants, OTel-specific config defaults, normalization/validation helpers that do not import the OTel SDK, and tests for that vocabulary. It must not become a general observability or config helper bucket.
3. Keep SDK-specific construction in `internal/infra/telemetry`: `buildTraceSampler`, endpoint parsing, header parsing, tracer provider setup, and exporter options remain there.
4. Stop applying config-owned resource identity defaults inside `internal/infra/telemetry.SetupTracing`. `SetupTracing` should trim and consume the values it is passed, while `internal/config` owns defaults and any validation needed for required resource identity fields.
5. `buildTraceSampler` must reject `NaN` and `+/-Inf` before clamping or building SDK samplers. Existing finite clamping behavior for out-of-range direct calls may remain unless tests prove it conflicts with the approved config contract.
6. Add an exported `ErrUninitializedServer` or equivalent inspectable error for nil/zero-value HTTP `Server` use. Validate the receiver before dereferencing `s.srv`; keep `ErrNilListener` for initialized `Serve(nil)`.
7. Replace `SetStartupDependencyStatus(dep, mode, bool)` call sites with intent-named methods such as `MarkStartupDependencyReady(dep, mode)` and `MarkStartupDependencyBlocked(dep, mode)`. Keep the numeric gauge encoding private inside telemetry.
8. Move telemetry init failure reason literals into telemetry-owned constants and use them from bootstrap. Keep `normalizeTelemetryFailureReason` as the low-cardinality safety net.
9. Delete `createAndListRecentInTx`, `txRollbackTimeout`, the `db` repository field if it becomes unused, and related fake transaction tests from the `ping_history` fixture. Do not replace them with a generic transaction helper.
10. Consolidate manual root route metadata by making one route table own method, path, handler, and reason, and derive test lookup data from it. This should not change routing behavior.
11. Update `docs/repo-architecture.md` and `docs/project-structure-and-module-organization.md` enough to show the new `internal/observability/otelconfig` ownership boundary.

## Open Questions / Assumptions

- Assumption: no downstream code relies on OTEL standard resource env variables because current repository policy only documents `APP__...` plus `NETWORK_*` direct env exceptions.
- Assumption: `createAndListRecentInTx` is test-only fixture code; current evidence shows no production or integration use outside `internal/infra/postgres/ping_history_repository_test.go`.
- Assumption: introducing `internal/observability/otelconfig` is acceptable because it is narrowly named and avoids both `common` and config-to-infra dependency direction.

## Plan Summary / Link

Implementation plan: `plan.md`.

Task ledger: `tasks.md`.

Technical design entrypoint: `design/overview.md`.

## Validation

Validation completed:

- `go test ./internal/observability/... ./internal/config ./internal/infra/telemetry`
- `go test ./internal/infra/telemetry ./cmd/service/internal/bootstrap`
- `go test ./internal/infra/http -run 'TestServer'`
- `go test ./internal/infra/postgres`
- `rg createAndListRecentInTx internal/infra/postgres`
- `go test ./internal/infra/http -run 'ManualRootRoute|RootRouter|OpenAPIRuntimeContract'`
- `go test ./internal/infra/...`
- `go test ./internal/observability/... ./internal/config ./cmd/service/internal/bootstrap`
- `git diff --check`
- `go test -tags=integration ./test -run TestPingHistoryRepositorySQLCReadWrite`

## Outcome

Implemented and validated. The implementation keeps generated SQLC output untouched, adds `internal/observability/otelconfig` as the narrow OTel vocabulary owner, removes the hidden OTel resource env ingestion path, hardens trace sampler inputs, adds HTTP server receiver guards, replaces startup dependency status booleans with intent-named telemetry calls, removes the Postgres transaction-only fixture path, and consolidates manual root route metadata.
