# Review Findings Trace

## Source

Read-only review of `cmd/service/internal/bootstrap` on 2026-04-12.

## Accepted Findings

1. `config_validation_failures_total` is used for non-config startup failures in bootstrap (`policy_violation`, `dependency_init`, `startup_error`), which makes config-specific telemetry own broader startup rejection semantics.
2. `serveHTTPRuntime` has an 11-argument positional signature with two `context.Context` values whose ordering compiles when swapped but changes lifecycle behavior.
3. `recordDependencyProbeRejection` accepts dependency label strings separately even though `startupDependencyProbeLabels` already owns the dependency, operation, and probe stage values.
4. Postgres and Mongo readiness participation predicates are repeated in bootstrap and config validation, while Redis already uses `Config.RedisReadinessProbeRequired()`.
5. `parseOptionalBoolEnvWithPresence` actually reports non-empty declaration semantics, not raw environment presence.
6. `EmitEgressExceptionState` only validates active egress exception expiry; it does not emit state.

## Reconciled Review Notes

- The idiomatic Go lane found no language or standard-library contract issue with real merge-risk impact.
- The naming-only issues are included only where the current name can mislead future behavior changes.
- The metric issue is treated as an observability contract correction, not a stylistic rename.

## Coverage Matrix

| Finding | Decision coverage | Design coverage | Task coverage | Status |
| --- | --- | --- | --- | --- |
| 1. `config_validation_failures_total` is used for non-config startup failures. | `spec.md` Decisions 1-2 choose `startup_rejections_total{reason}`, keep config metric for config-layer failures, and reject broadening the old metric. | `design/overview.md`, `design/component-map.md`, and `design/sequence.md` define the telemetry split, bounded reasons, runtime sequence, and failure checks. | `tasks.md` T001-T003 and T009; validation phase assertion for non-config startup rejection reasons. | Covered |
| 2. `serveHTTPRuntime` has an 11-argument positional signature with two contexts. | `spec.md` Decision 3 requires a small unexported named-field struct and unchanged behavior. | `design/component-map.md` and `design/sequence.md` identify the bootstrap call-site refactor; `design/ownership-map.md` keeps lifecycle ownership in bootstrap. | `tasks.md` T005; implementation phase allowed writes include bootstrap HTTP runtime surfaces. | Covered |
| 3. `recordDependencyProbeRejection` accepts dependency label strings separately. | `spec.md` Decision 4 requires consuming `startupDependencyProbeLabels` and preserving logs/span attributes/startup rejection behavior. | `design/component-map.md`, `design/sequence.md`, and `design/ownership-map.md` assign dependency label source-of-truth to bootstrap labels. | `tasks.md` T006; implementation phase allowed writes include dependency rejection labels. | Covered |
| 4. Postgres and Mongo readiness predicates are repeated while Redis has a config helper. | `spec.md` Decision 5 requires `Config.PostgresReadinessProbeRequired()` and `Config.MongoReadinessProbeRequired()` and unchanged Redis semantics. | `design/component-map.md`, `design/sequence.md`, and `design/ownership-map.md` place readiness predicates in `internal/config` and consume them from validation and bootstrap. | `tasks.md` T004; validation phase requires predicate/probe alignment. | Covered |
| 5. `parseOptionalBoolEnvWithPresence` actually means non-empty explicit declaration. | `spec.md` Decision 6 selects explicit declaration terminology and preserves blank-env-is-not-declared semantics. | `design/component-map.md` and `design/sequence.md` identify the rename-only network policy parser change; `design/overview.md` says network policy behavior must not change. | `tasks.md` T007; validation phase requires unchanged network policy behavior. | Covered |
| 6. `EmitEgressExceptionState` validates but does not emit state. | `spec.md` Decision 7 selects `ValidateEgressExceptionState` and rejects adding new emission. | `design/component-map.md` and `design/sequence.md` identify the rename-only egress helper change; `design/overview.md` says no new emission behavior. | `tasks.md` T008; validation phase requires egress exception tests still pass. | Covered |

## Coverage Verdict

All accepted review findings are represented in the decision record, technical design bundle, implementation plan, task ledger, and validation phase expectations.
