# Bootstrap Review Fixes Plan

## Execution Context

The next session should implement the review fixes from the approved `spec.md` and `design/` bundle. No API, schema, migration, generated-code, or rollout work is expected.

## Phase Plan

Phase: implementation-phase-1.
Objective: close the bootstrap review findings while preserving tracing fail-open behavior and secret-safe diagnostics.
Depends on: approved `spec.md` and `design/`.
Task ledger: `tasks.md` T001-T009.
Change surface:
- `internal/infra/telemetry/tracing.go`
- `internal/infra/telemetry/tracing_test.go`
- `cmd/service/internal/bootstrap/startup_bootstrap.go`
- `cmd/service/internal/bootstrap/startup_common.go`
- `cmd/service/internal/bootstrap/startup_common_additional_test.go`
- `cmd/service/internal/bootstrap/startup_probe_addresses.go`
- `cmd/service/internal/bootstrap/startup_probe_addresses_test.go`
- `cmd/service/internal/bootstrap/startup_dependency_labels.go`
- `cmd/service/internal/bootstrap/startup_dependencies.go`
- `cmd/service/internal/bootstrap/startup_dependencies_additional_test.go`

Acceptance criteria:
- OTLP exporter target is admitted by bootstrap network policy before exporter creation.
- Headers alone cannot create an exporter with an uninspected SDK default/env endpoint.
- Policy-denied telemetry disables tracing and does not block startup unless global `NETWORK_*` config is invalid.
- Metrics and spans share one config-stage duration mapping.
- Postgres probe-address parse diagnostics are explicitly secret-safe.
- Dependency probe stage labels are self-explanatory and covered by tests.

Planned verification:
- `go test ./internal/infra/telemetry ./cmd/service/internal/bootstrap`
- `go test -race ./cmd/service/internal/bootstrap`
- `go vet ./internal/infra/telemetry ./cmd/service/internal/bootstrap`

Review/checkpoint:
- After implementation, review the telemetry/network-policy path first because it carries the main security/reliability tradeoff.
- Then review the local maintainability changes for unnecessary abstraction or behavior drift.

## Implementation Readiness

Status: PASS.

The accepted risks are:
- Telemetry egress denial is treated as optional tracing degradation, not startup rejection.
- Postgres parse cause remains redacted unless a safe wrapping path is proven during implementation.

Proof obligations:
- Tests must prove the selected telemetry denial behavior.
- Tests or comments must prove DSN parse diagnostics are intentionally redacted.

## Blockers / Assumptions

Blockers: none.

Assumptions:
- `internal/infra/telemetry` can expose a pure target-description helper without widening its ownership.
- `OTLPEndpoint`/`OTLPTracesEndpoint` are the only supported application-config endpoint sources; SDK env endpoint fallback should not silently influence exporter target selection.

## Handoffs / Reopen Conditions

Start implementation from `tasks.md` T001.

Reopen technical design before coding further if:
- endpoint target extraction cannot be made to match exporter construction;
- the fix requires changing config-source policy for `OTEL_EXPORTER_OTLP_*`;
- security/reliability review rejects tracing fail-open on policy-denied telemetry egress.
