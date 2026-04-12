# Implementation Phase 1

## Scope

Implement the approved maintainability fixes for bootstrap/config/telemetry seams.

## Inputs

- `spec.md`
- `design/overview.md`
- `design/component-map.md`
- `design/sequence.md`
- `design/ownership-map.md`
- `plan.md`
- `tasks.md`

## Allowed Writes

- `internal/infra/telemetry/metrics.go`
- `internal/infra/telemetry/metrics_test.go`
- `internal/config/*` files needed for readiness predicates and tests
- `cmd/service/internal/bootstrap/*` files needed for startup metric calls, HTTP runtime argument struct, dependency rejection labels, env-declaration helper naming, egress exception validation naming, and focused tests
- Existing task/progress artifacts in this task folder

## Not In Scope

- API/OpenAPI changes
- data migrations
- behavior changes to dependency probing, shutdown, network policy decisions, ingress/egress allowlist semantics, retry budgets, or readiness/liveness contract
- new dashboards, alerts, runbooks, or external deployment rollout files

## Execution Notes

- Start with `tasks.md` T001 and continue in dependency order.
- Keep changes local and behavior-preserving except for the intended metric contract correction.
- Do not create new generic helper packages or broad abstractions.
- Update tests alongside each changed seam.

## Stop Rule

Stop and reopen technical design if implementation reveals that the metric contract must preserve old non-config samples on `config_validation_failures_total`, or that readiness predicate semantics require a product/operator policy decision not recorded in `spec.md`.

## Status

Complete. Implementation followed `tasks.md` T001-T008 and no reopen condition fired. Task-local validation evidence and unrelated full-validation blockers are recorded in `workflow-plans/validation-phase-1.md` and `spec.md`.
