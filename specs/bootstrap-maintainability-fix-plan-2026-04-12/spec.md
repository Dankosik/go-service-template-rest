# Bootstrap Maintainability Fixes

## Context

The read-only review of `cmd/service/internal/bootstrap` found no Go language-level correctness issues, but it did find maintainability and source-of-truth drift around startup telemetry, lifecycle argument passing, dependency-probe labels, readiness predicates, and two misleading helper names.

This task fixed the review findings without changing startup/shutdown semantics, network policy decisions, dependency probing behavior, HTTP contract behavior, or readiness/liveness semantics.

## Scope / Non-goals

In scope:

- Correct the startup rejection telemetry contract so non-config failures are not counted as config validation failures.
- Reduce lifecycle call-site risk in `serveHTTPRuntime`.
- Make dependency probe rejection labels derive from the existing `startupDependencyProbeLabels` source of truth.
- Add config-level readiness predicate helpers for Postgres and Mongo to match the existing Redis helper pattern.
- Rename misleading helper surfaces for explicit env declaration and egress exception validation.
- Update focused tests to prove the contracts above.

Out of scope:

- Changing dependency criticality or degraded-mode behavior.
- Changing startup retry budgets, timeout budgets, shutdown ordering, or readiness/liveness behavior.
- Changing network policy allowlist, ingress exception, egress exception, or host classification semantics.
- Adding API, database, migration, rollout, dashboard, alert, or runbook artifacts.
- Broad refactors of bootstrap into new packages or manager abstractions.

## Constraints

- Keep `cmd/service/internal/bootstrap` as the composition-root owner for lifecycle orchestration.
- Keep `internal/config` as the source of truth for reusable config predicates.
- Keep `internal/infra/telemetry` as the owner of shared Prometheus metric instruments.
- Metric labels must stay low-cardinality and allowlisted.
- The later implementation session must not create new workflow/process/design artifacts unless a recorded reopen condition fires.

## Decisions

1. Add a new metric instrument `startup_rejections_total{reason}` in `internal/infra/telemetry`.
   - Purpose: answer the operator question, "Why did startup reject readiness/process startup?"
   - Reasons must be bounded. Planned reasons: `config_load`, `config_parse`, `config_validate`, `config_strict_unknown_key`, `config_secret_policy`, `policy_violation`, `dependency_init`, `startup_error`, and `other`.
   - Add `Metrics.IncStartupRejection(reason string)` with normalization to the bounded set.
   - Config-stage failures should increment both the config-specific failure metric and the startup rejection metric.
   - Policy, dependency, and HTTP startup failures should increment only the startup rejection metric, not `config_validation_failures_total`.
   - Rejected alternative: broaden `config_validation_failures_total` to mean all startup failures. That preserves one metric but keeps the misleading name as the operator contract.
   - Rejected alternative: resurrect per-policy metrics such as `network_policy_violation_total`. That widens the telemetry surface without adding a better operator decision than one bounded rejection reason counter.

2. Keep `config_validation_failures_total{reason}` for config-layer failures only.
   - The existing metric name is imperfect even for parse/load failures, but this task should not rename or remove it because the review finding is about non-config startup failures being counted there.
   - The implementation may clarify tests and helper comments, but it should not perform a broad compatibility-breaking telemetry rename beyond adding `startup_rejections_total`.

3. Replace the 11-argument `serveHTTPRuntime` call shape with a small unexported struct.
   - The struct should use named fields for `signalCtx`, `bootstrapCtx`, `bootstrapSpan`, `cfg`, `log`, `metrics`, `healthSvc`, `srv`, `readinessCheck`, `admission`, and `shutdownDelay`.
   - Keep `serveHTTPRuntime` behavior unchanged.
   - Rejected alternative: split runtime serving into several new abstraction layers. The current risk is call-site ambiguity, not missing lifecycle ownership.

4. Change `recordDependencyProbeRejection` to consume `startupDependencyProbeLabels`.
   - Derive dependency, operation, and probe stage from the label struct.
   - Keep `mode` and `err` as explicit inputs.
   - Preserve the existing log event name, span attributes, and startup rejection behavior.

5. Add canonical config predicates for Postgres and Mongo readiness participation.
   - Planned API shape: `Config.PostgresReadinessProbeRequired()` and `Config.MongoReadinessProbeRequired()`.
   - Use them in `internal/config.validateReadinessProbeBudgets` and `cmd/service/internal/bootstrap.initStartupDependencies`.
   - Keep `Config.RedisReadinessProbeRequired()` semantics unchanged.

6. Rename `parseOptionalBoolEnvWithPresence` to explicit declaration terminology.
   - Preferred name: `parseOptionalBoolEnvWithExplicitDeclaration`.
   - Semantics must remain: absent or blank env value is not an explicit declaration; a valid non-empty bool token is an explicit declaration.
   - Rejected alternative: treat raw env presence with blank value as declared. That would weaken the public-ingress fail-closed declaration rule.

7. Rename `EmitEgressExceptionState` to validation terminology.
   - Preferred name: `ValidateEgressExceptionState`.
   - Do not add new log/metric emission in this task.
   - Rejected alternative: add actual emission because no operator decision or signal contract was requested beyond fixing the misleading helper surface.

## Open Questions / Assumptions

- Assumption: adding `startup_rejections_total` is acceptable even if local tests or dashboards expecting non-config reasons on `config_validation_failures_total` need adjustment.
- Assumption: this is a template repository; no external production dashboard compatibility requirement is recorded in the task artifacts.
- No user-only product decision is currently blocking implementation.

## Plan Summary / Link

Implementation should follow `plan.md` and `tasks.md`. The expected next phase is `workflow-plans/implementation-phase-1.md`.

## Validation

Required proof:

- `go test -count=1 ./cmd/service/internal/bootstrap`
- `go test -count=1 ./internal/config ./internal/infra/telemetry`
- Prefer `make test` or `go test ./...` if runtime budget allows.
- Targeted assertions must prove the new metric split, config readiness predicate reuse, unchanged startup/shutdown behavior, and unchanged network policy behavior.

Fresh validation on 2026-04-12:

- `gofmt -l` on touched Go files: no output.
- `git diff --check` on touched Go files: passed.
- `go test -count=1 ./cmd/service/internal/bootstrap`: 91 passed in 1 package.
- `go test -count=1 ./internal/config`: 119 passed in 1 package.
- `go test -count=1 ./internal/infra/telemetry -run 'TestNormalizeStartupRejectionReason|TestCoreMetricsHandlerExposesExpectedSeries|TestMetricsNilAndZeroValueMethodsAreNoops'`: 13 passed in 1 package.
- `go test -count=1 ./internal/config ./internal/infra/telemetry`: not fully verified in the current workspace because unrelated tracing work fails `TestSetupTracingUsesConfigResourceAttributesOnly`.
- `go test ./...`: not fully verified in the current workspace because unrelated Postgres repository test edits fail to build and the unrelated tracing test above fails.

## Outcome

Implemented in this session. All `tasks.md` items T001-T009 are complete for the bootstrap maintainability scope; no task-local reopen condition fired. Full package/repo validation is blocked by unrelated parallel workspace changes outside this task.
