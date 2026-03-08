## Context

The repository currently ships a core observability baseline plus a thicker deployment-oriented observability layer. The user wants the core template narrowed to:
- structured logs,
- `/metrics`,
- optional tracing.

This is a subtractive template-baseline change, not a redesign of the service runtime model.

## Scope / Non-goals

In scope:
- remove Grafana-specific config surface from the core template
- remove deployment/rollback/config-drift/network-policy telemetry that goes beyond the core observability baseline
- keep the service runtime behavior intact where possible while removing the extra observability layer
- keep `/metrics`, structured logs, and optional tracing working
- update affected tests and template-facing docs/config defaults

Non-goals:
- redesigning startup, health, or network-policy behavior
- introducing a local Prometheus or Grafana bundle
- rewriting general observability guidance docs that are not part of the runtime template contract
- broad refactors outside the observability-simplification surface

## Constraints

- Execution shape: `lightweight local`
- Research mode: `local`
- Pre-spec challenge: `waived`
  Rationale: the target state is already user-approved, the change is subtractive, and the main risk is repository change-surface coverage rather than unresolved design ambiguity.
- The template must still expose a usable `/metrics` endpoint, structured logs, and optional OTLP tracing after simplification.
- The change should preserve runtime correctness and keep validation targeted to the touched packages.

## Decisions

1. Keep the core observability substrate to three capabilities only: logs, `/metrics`, and optional tracing.
2. Remove template config knobs that the simplified runtime does not honor:
   - `observability.grafana.*`
   - `observability.metrics.*`
3. Remove deployment-oriented telemetry from the core template:
   - deploy health admission metrics/logs/traces
   - rollback metrics/logs/traces
   - config drift metrics/logs/traces
   - network policy telemetry metrics/logs/traces
   - release-readiness policy threshold helpers tied to deployment telemetry
4. Keep startup, readiness, dependency probing, and network-policy enforcement behavior, but strip the extra telemetry wiring around those paths.
5. Keep core metrics that still fit the template baseline:
   - Go/process collectors
   - HTTP request counters and latency
   - config/bootstrap and tracing-init metrics that remain directly tied to app startup rather than platform rollout policy

## Open Questions / Assumptions

- Assumption: removing deployment telemetry from the core template is more valuable than preserving that specialized surface for future platform-specific deployments.
- Assumption: config/bootstrap metrics remain acceptable as part of the core `/metrics` baseline because they do not introduce external platform coupling.

## Implementation Plan

1. Trim config surface.
   Completion criteria:
   - dead observability config knobs are removed from runtime snapshot/defaults
   - strict config tests reject removed observability keys

2. Trim telemetry package to the core metric set.
   Completion criteria:
   - deploy/rollback/drift/network metric definitions and helpers are removed
   - release-readiness and policy-threshold helpers/tests are removed
   - `/metrics` handler and core HTTP/config/tracing metrics still work

3. Remove deployment telemetry wiring from bootstrap/runtime code while preserving behavior.
   Completion criteria:
   - `deployTelemetryRecorder` is removed
   - startup/network policy/bootstrap code no longer depends on deployment telemetry objects
   - startup/readiness/network policy behavior still compiles and tests cleanly

4. Update affected tests and narrow template-facing docs/config wording where needed.
   Completion criteria:
   - bootstrap and telemetry tests assert the remaining core behavior only
   - config defaults/docs no longer mention Grafana runtime config

## Validation

- `go test ./internal/config ./internal/infra/telemetry ./internal/infra/http ./cmd/service/internal/bootstrap`
- `git diff --check -- cmd/service/internal/bootstrap internal/config internal/infra/telemetry env/config/default.yaml specs/core-template-observability/spec.md test`
- targeted repository search confirmed removal of runtime Grafana config and deployment-telemetry symbols outside this spec artifact

## Outcome

Completed:
- core template observability is now limited to structured logs, `/metrics`, and optional OTLP tracing
- Grafana runtime config, dead metrics config knobs, deployment/rollback/drift/network telemetry helpers, and release-readiness policy-threshold helpers were removed
- bootstrap and network-policy behavior remain, but no longer carry deployment-telemetry wiring
