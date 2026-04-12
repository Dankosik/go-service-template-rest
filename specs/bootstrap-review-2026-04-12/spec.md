# Bootstrap Review Fixes Spec

Status: implemented; validation passed.

## Context

The completed review of `cmd/service/internal/bootstrap` found four issues:

- OTLP tracing exporter egress is configured before bootstrap network policy can admit or deny the outbound target.
- Config load stage durations are mapped twice: once for metrics and once for spans.
- The Postgres probe-address parse error drops the raw parser cause; after rechecking secret policy, this needs a secret-safe diagnostic decision rather than blindly wrapping the raw DSN parser error.
- Startup dependency probe labels use `probeStage` and `probeName` in a way that obscures which value owns trace/rejection stage versus low-budget error text.

The repository baseline says bootstrap owns startup/shutdown flow, dependency admission, and runtime policy, while telemetry owns OTel SDK setup and config parsing details. `NETWORK_*` variables are a bootstrap-owned operator policy channel that constrains outbound targets. Tracing remains optional and fail-open.

## Scope / Non-goals

In scope:
- Gate configured OTLP tracing exporter egress through bootstrap network policy before enabling the exporter.
- Keep tracing optional: policy-denied telemetry must disable tracing rather than make the service fail startup, unless the broader network policy configuration is itself invalid and would already block startup.
- Remove the duplicated config-stage duration mapping used for metrics and spans.
- Preserve or improve Postgres probe-address diagnostics without leaking raw DSN content.
- Clarify startup dependency probe label ownership and tests.

Non-goals:
- No API, schema, migration, generated-code, or rollout changes.
- No change to HTTP routing, app services, dependency probe criticality, or readiness semantics.
- No change to OTel resource identity behavior except as needed to prevent hidden exporter target selection.
- No broad telemetry/config redesign beyond the trace exporter egress target seam.

## Constraints

- `cmd/service/internal/bootstrap` remains the network-policy owner.
- `internal/infra/telemetry` remains the owner of OTLP endpoint precedence/parsing and OTel SDK option construction.
- Do not duplicate OTLP endpoint parsing in bootstrap.
- Do not allow `OTLPHeaders` alone, SDK defaults, or `OTEL_EXPORTER_OTLP_*` environment fallback to enable an exporter target that bootstrap cannot inspect.
- `NETWORK_EGRESS_ALLOWED_SCHEMES` currently denies disallowed schemes before host private/public classification; do not change that policy while fixing OTLP egress.
- DSN-related errors must respect `docs/configuration-source-policy.md`: secret-like values such as DSNs must not leak through errors or logs.
- Existing package-level checks must remain green: `go test ./cmd/service/internal/bootstrap`, `go test -race ./cmd/service/internal/bootstrap`, and `go vet ./cmd/service/internal/bootstrap`.

## Decisions

1. Telemetry exporter egress must be policy-admitted before `telemetry.SetupTracing` creates an OTLP exporter.

2. Tracing remains fail-open. If a configured telemetry exporter target is denied by network policy, bootstrap should record telemetry as `feature_off`/degraded and continue startup; the exporter must not be created. If `NETWORK_*` configuration is invalid, the normal network-policy startup rejection remains in force.

3. Telemetry should expose one pure target-description seam for bootstrap to inspect. Preferred shape: an exported telemetry helper that uses the same endpoint-selection rules as `buildTraceExporterOptions` and returns the effective endpoint target, scheme, configured flag, and parse error. Bootstrap calls the helper and then calls `networkPolicy.EnforceEgressTarget`.

4. `OTLPTracesEndpoint` keeps precedence over `OTLPEndpoint`. Scheme-less OTLP endpoint strings continue to mean insecure HTTP for telemetry parsing; for network policy they should be checked as `http`. `http://...` checks as `http`, and `https://...` checks as `https`.

5. Headers alone must not silently create an exporter that uses SDK default or SDK environment-derived endpoint. The implementation should align `buildTraceExporterOptions` with the target-description helper so exporter creation requires an explicit application-config endpoint, or otherwise pass an explicit inspected default target. Prefer explicit endpoint requirement unless implementation proves a compatible local-default behavior is already intentionally tested.

6. Config stage durations should have one package-local source of truth, such as `configLoadStageDurations(report config.LoadReport)`, consumed by both metrics and span recording.

7. Postgres probe-address parse diagnostics must stay secret-safe. Do not wrap or log the raw `pgxpool.ParseConfig` error unless tests prove the error cannot include DSN-sensitive content. The safer default is to keep a redacted error message and add tests/comments that make the redaction intentional, not accidental.

8. Probe label fields should expose their role. Prefer collapsing the budget-check stage and span/rejection stage to one canonical `startup.probe.<dep>` value when that preserves observability semantics; otherwise rename the fields to role-specific names and add tests that pin the distinction.

## Open Questions / Assumptions

- Assumption: It is acceptable for a policy-denied OTLP exporter to degrade tracing instead of failing startup because tracing is already modeled as optional fail-open.
- Assumption: Operators who configure public OTLP endpoints should add `http` or `https` to `NETWORK_EGRESS_ALLOWED_SCHEMES` plus the host to the egress allowlist or active exception.
- Reopen condition: If security or reliability review says telemetry egress policy violations must fail startup, reopen this spec before implementation.
- Reopen condition: If OTel SDK v1.19.0 environment fallback cannot be neutralized without a broader config-source decision, reopen technical design.

## Review Coverage Audit

All final review/research outputs are accounted for:

| Source | Point | Coverage |
| --- | --- | --- |
| Workflow adequacy challenge | The review-only stop rule needed clarification so review did not drift into implementation planning. | Resolved in `workflow-plan.md` and `workflow-plans/review-phase-1.md`; no code task. |
| `go-idiomatic-review` lane | No findings. | No implementation task. |
| `go-language-simplifier-review` lane | Config stage duration mapping is duplicated between metrics and spans. | Covered by Decision 6, `design/*`, `plan.md`, and `tasks.md` T006. |
| `go-language-simplifier-review` lane | `probeStage` / `probeName` naming hides stage/label intent. | Covered by Decision 8, `design/*`, `plan.md`, and `tasks.md` T008. The optional observability handoff is represented as a proof/reopen concern if label semantics are external dashboard contracts. |
| `go-design-review` lane | OTLP exporter target bypasses bootstrap egress policy. | Covered by Decisions 1-5, `design/*`, `plan.md`, and `tasks.md` T001-T005. Security/reliability policy choice is recorded as fail-open with a reopen condition. |
| Orchestrator local review | Postgres probe-address parse diagnostics dropped parser cause. | Covered by Decision 7, `design/*`, `plan.md`, and `tasks.md` T007, with the corrected secret-safe framing. |
| Review validation / residual risk | Prior package checks were green but implementation still needs fresh proof. | Covered by `workflow-plan.md`, Validation below, `plan.md`, and `tasks.md` T009. |

Superseded notes are not implementation targets. In particular, any earlier draft references to `overlayPathsFlag`, `probeWithRetry`, or `serveHTTPRuntime` are not part of the final reconciled review findings and must not drive this implementation session.

## Plan Summary / Link

Implementation strategy and task order live in `plan.md` and `tasks.md`.

## Validation

Required proof after implementation:
- Unit tests for telemetry exporter target extraction and headers-without-endpoint behavior.
- Bootstrap tests proving denied OTLP egress disables telemetry without creating an exporter and without blocking startup for policy-denied optional telemetry.
- Existing network-policy tests remain unchanged unless the test update explicitly pins OTLP scheme behavior.
- Tests proving config load stage metrics and spans consume the same full ordered stage set.
- Tests or comments proving Postgres DSN parse diagnostics remain secret-safe.
- Tests proving dependency probe label stage semantics are clear and stable.
- `go test ./internal/infra/telemetry ./cmd/service/internal/bootstrap`
- `go test -race ./cmd/service/internal/bootstrap`
- `go vet ./internal/infra/telemetry ./cmd/service/internal/bootstrap`

Fresh proof after implementation:
- `go test ./internal/infra/telemetry`
- `go test ./cmd/service/internal/bootstrap`
- `go test ./internal/infra/telemetry ./cmd/service/internal/bootstrap`
- `go test -race ./cmd/service/internal/bootstrap`
- `go vet ./internal/infra/telemetry ./cmd/service/internal/bootstrap`

## Outcome

Implemented in this session. `tasks.md` T001-T009 are complete.
