# Review Findings Coverage

This file verifies that every finding from the `internal/infra` read-only review fan-in has an explicit decision, design note, task, and validation hook before implementation starts.

## Coverage Matrix

| Review finding | Decision / design coverage | Task coverage | Validation coverage | Status |
| --- | --- | --- | --- | --- |
| `resource.WithFromEnv()` reads OTEL environment variables outside the repository config snapshot. | `spec.md` Decisions 1 and 11; `design/component-map.md` `internal/infra/telemetry`; `design/sequence.md` step 3. | `T003`, `T004`. | `test-plan.md` focused Phase 1 checks and manual diff check for `resource.WithFromEnv()`. | Covered. |
| `SetupTracing` applies resource identity fallback defaults (`service`, `dev`, `unknown`) inside the telemetry adapter. | `spec.md` Decision 4; `design/overview.md` rationale; `design/component-map.md` `internal/config` and `internal/infra/telemetry`; `design/sequence.md` steps 2 and 3. | `T002`, `T003`, `T004`. | `test-plan.md` manual diff check for no telemetry-owned resource identity fallback defaults. | Covered. |
| OTel sampler/protocol vocabulary is repeated between config validation/defaults and telemetry runtime setup. | `spec.md` Decisions 2 and 3; `design/dependency-graph.md`; `design/ownership-map.md`. | `T001`, `T002`, `T003`. | `test-plan.md` Phase 1 focused checks. | Covered. |
| `clampRatio` lets `NaN` pass into `sdktrace.TraceIDRatioBased`; infinities also need explicit runtime-boundary handling. | `spec.md` Decision 5; `design/component-map.md` `internal/infra/telemetry`; `design/sequence.md` step 3. | `T003`, `T004`. | `test-plan.md` Phase 1 focused checks for `TestBuildTraceSampler`. | Covered. |
| Exported HTTP `Server` methods panic on nil receiver or zero-value use. | `spec.md` Decision 6; `design/component-map.md` `internal/infra/http`; `plan.md` Phase 3. | `T007`. | `test-plan.md` Phase 3 focused check. | Covered. |
| `SetStartupDependencyStatus(dep, mode, bool)` hides ready/blocked intent behind raw booleans. | `spec.md` Decision 7; `design/component-map.md` `internal/infra/telemetry` and bootstrap; `plan.md` Phase 2. | `T005`. | `test-plan.md` Phase 2 focused checks. | Covered. |
| Telemetry init failure reason strings are split between bootstrap classification and telemetry normalization. | `spec.md` Decision 8; `design/ownership-map.md` label ownership; `design/sequence.md` step 4. | `T006`. | `test-plan.md` Phase 2 focused checks. | Covered. |
| The replaceable `ping_history` fixture contains unused transaction lifecycle policy and fake transaction scaffolding. | `spec.md` Decision 9; `design/component-map.md` `internal/infra/postgres`; `plan.md` Phase 4. | `T008`. | `test-plan.md` Phase 4 focused check and no-match `rg createAndListRecentInTx`. | Covered. |
| Manual root route declarations and documented exception reasons are maintained in parallel structures. | `spec.md` Decision 10; `design/component-map.md` `internal/infra/http`; `design/sequence.md` step 7; `plan.md` Phase 5. | `T009`. | `test-plan.md` Phase 5 focused check. | Covered. |

## Self-Check Result

All review findings are represented. The only additional planning-only item is updating repository architecture/structure docs for the new `internal/observability/otelconfig` boundary; that is included in `T001` and `spec.md` Decision 11 because it prevents the fix from creating a new undocumented package boundary.
