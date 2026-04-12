# Design Overview

## Chosen Approach

Use small source-of-truth repairs instead of broad bootstrap restructuring:

- Add one bounded startup rejection metric in `internal/infra/telemetry`.
- Route startup rejection reasons from bootstrap to that metric.
- Keep config-specific failure accounting on the existing config metric.
- Replace risky positional lifecycle parameters with named struct fields.
- Reuse existing label and config predicate seams instead of introducing new helper buckets.
- Rename two misleading helpers without behavior changes.

## Artifact Index

- `component-map.md`: affected packages and code surfaces.
- `sequence.md`: implementation and runtime order.
- `ownership-map.md`: source-of-truth and dependency boundaries.

## Cross-Domain Notes

- Observability: `startup_rejections_total{reason}` is the only new metric. The `reason` label is bounded and normalized.
- Config: readiness participation predicates belong in `internal/config` because both config validation and bootstrap consume them.
- Bootstrap: startup orchestration remains in `cmd/service/internal/bootstrap`; no app/infra ownership changes are needed.
- Testing: tests should move non-config failure assertions from `config_validation_failures_total` to `startup_rejections_total`.

## Readiness

Design is stable enough for planning and implementation with `CONCERNS` because the metric contract changes. The accepted risk and proof obligations are recorded in `workflow-plan.md` and `plan.md`.

## Reopen Conditions

Reopen design if implementation needs to:

- preserve non-config samples on `config_validation_failures_total`;
- add dashboards, alerts, or runbooks;
- change readiness participation semantics instead of only centralizing predicates;
- change network policy behavior instead of renaming helpers.
