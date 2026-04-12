# Review Findings Traceability

Status: complete

This file maps every material point from the `internal/infra` review fan-out to an implementation disposition. "Covered" means the point is either scheduled for implementation, explicitly deferred as a separate decision, or recorded as conditional/non-goal so it is not lost.

## Accepted Implementation Items

| ID | Review point | Disposition | Owning artifact / task |
| --- | --- | --- | --- |
| R01 | `MaxIdleConns` is not a reliable strict cap when implemented with `pgxpool.AfterRelease` plus `pool.Stat().IdleConns()` because `AfterRelease` runs asynchronously before return-to-idle. | Implement stateful limiter. | `spec.md` Decisions 1; `design/sequence.md` Postgres pool release flow; `tasks.md` T001 |
| R02 | `NewPingHistoryRepository` can accept a typed nil DB and fail later inside SQLC-generated code. | Reject invalid DB dependency at exported boundary and add repository readiness guards. | `spec.md` Decisions 3; `design/component-map.md` Postgres repository; `tasks.md` T003 |
| R03 | `(*Pool).DB` has inconsistent nil receiver behavior compared with `Close` and `Check`. | Make `DB` nil-safe. | `spec.md` Decisions 2; `tasks.md` T002 |
| R04 | `Metrics` methods are nil-receiver tolerant but zero-value `Metrics` panics through nil collector fields. | Make methods nil/zero-value tolerant; handler returns not-found when registry is missing. | `spec.md` Decisions 8; `tasks.md` T004 |
| R05 | `NewRouter` duplicates the config-owned `MaxBodyBytes` fallback. | Require explicit validated `MaxBodyBytes`; update tests to pass config-owned default. | `spec.md` Decisions 4; `design/ownership-map.md` Config ownership; `tasks.md` T005 |
| R06 | HTTP OpenTelemetry middleware only wraps generated handlers and manual `/metrics`, missing 404/405, body/framing rejection, and recovery surfaces. | Move to one edge-wide server span wrapper without nested normal-route spans. | `spec.md` Decisions 5; `design/sequence.md` HTTP request flow; `tasks.md` T006 |
| R07 | Route capture renames spans but does not set route attributes or `otelhttp` labeler route tags. | Set bounded span name plus `http.route` span/labeler attribute after chi route resolution. | `spec.md` Decisions 6; `tasks.md` T006 |
| R08 | Strict generated `Metrics` handler buffers through `httptest.NewRecorder`, creating a second functional `/metrics` path behind the root-route exception. | Make strict `Metrics` explicitly non-runtime-owned and remove production recorder use. | `spec.md` Decisions 7; `tasks.md` T007 |
| R09 | `isSafeOTLPHeaderKey` name implies general validation though it is only an error-redaction predicate. | Rename to a redaction-specific name if touching the file. | `spec.md` Decisions 9; `tasks.md` T008 |
| R10 | `fakePingHistoryQuerier` callbacks are mandatory even when a test does not exercise that path, and unexpected calls panic unclearly. | Make fake callbacks optional and return targeted unexpected-call errors. | `spec.md` Decisions 9; `tasks.md` T008 |
| R11 | `createAndListRecentInTx` has rollback cleanup coverage, but lacks success-path and commit-error unit coverage. | Add coverage while updating Postgres repository tests. | `spec.md` Validation; `tasks.md` T003 |

## Explicit Deferrals And Non-goals

| ID | Review point | Disposition | Where recorded |
| --- | --- | --- | --- |
| D01 | `SetupTracing` duplicates some config-default fallback behavior. | Deferred and explicitly out of scope for this batch; reopen config/telemetry ownership separately if desired. | `spec.md` Decisions 10 and Reopen conditions; `plan.md` Accepted risks; `workflow-plan.md` Blockers and risks |
| D02 | `/metrics` security/auth remains operational-private-required but no runtime auth/internal listener is added. | Non-goal for this infra cleanup; not a routing/code correctness fix. | `spec.md` Non-goals; OpenAPI remains the source of truth |
| D03 | Current OpenAPI routes are static, so parameterized route-label collapse is not directly proved. | Conditional future obligation: add parameter-collapse tests when a parameterized route exists. | `design/sequence.md` fallback notes; validation phase targeted evidence focuses current static routes |
| D04 | SQLC generation drift was not freshly proved in review. | Conditional validation only if generated inputs or outputs change unexpectedly. | `spec.md` Validation; `plan.md` Cross-Phase Validation; `workflow-plans/validation-phase-1.md` Optional Validation |
| D05 | Full migration up/down rehearsal was not run in review. | Conditional validation only if migrations change; this task has no migration changes. | `spec.md` Non-goals and Validation |
| D06 | Docker-backed Postgres integration proof may be unavailable locally. | Optional validation, reported honestly if unavailable. | `spec.md` Validation; `plan.md` Planned Verification; `tasks.md` T009 |

## Coverage Conclusion

All material review fan-out points are covered by one of:
- implementation task T001-T008,
- validation task T009,
- explicit deferral/non-goal D01-D06.

No review point is left as an implicit chat-only note.
