# 70 Test Plan

Status: updated

Changed sections linked to testing decisions: `TST-001`, `TST-002`, `TST-003`, `TST-004`, `TST-005`.
Changed sections linked to domain decisions: `DOM-001`, `DOM-002`, `DOM-003`, `DOM-004`, `DOM-005`, `DOM-006`.
Changed sections linked to architecture decisions: `ARCH-001`, `ARCH-002`, `ARCH-004`.
Changed sections linked to design decisions: `DES-001`, `DES-002`, `DES-003`, `DES-004`.
Changed sections linked to security decisions: `SEC-001`, `SEC-002`, `SEC-003`.
Changed sections linked to observability decisions: `OBS-001`, `OBS-002`, `OBS-003`, `OBS-004`.
Changed sections linked to closed/reframed questions: `OQ-001`, `OQ-002`, `OQ-003`, `OQ-006`.

## Phase Target
- Current phase: `Phase 2`
- Target gate: `G2`
- Focus: finalize implementation-ready test strategy and evidence matrix for Railway deployment resilience.

## Scope And Test Levels
1. In scope:
- deploy admission, readiness-gated promotion, drain-first replacement, restart behavior, capacity baseline, policy drift control, networking ingress/egress policy with exception path;
- evidence traceability across `DOM/ARCH/DES/OQ`;
- command-level quality expectations for local and CI validation.
2. Out of scope:
- writing test code in this pass;
- API payload/schema expansion beyond existing healthcheck usage;
- DB/cache migration testing (no schema/cache scope in this feature).
3. Selected level mix:
- `integration`: primary proof level for deploy/runtime boundary behavior.
- `contract`: health endpoint semantics used by rollout admission.
- `e2e-smoke`: minimal end-to-end confidence for production rollout gate.
- `unit`: limited to future parser/validator helpers for deploy policy diff logic.

## Test-Level Selection Rationale

### TST-001: Rollout Safety Proof Level
- Phase/Gate: `Phase 2`, target `G2`
- Owner: QA + Platform
- Context/risk: rollout and replacement safety (`DOM-001/002/003`, `ARCH-002`).
- Options:
1. `e2e-smoke` only.
2. `integration + contract + e2e-smoke`.
- Selected: **Option 2**.
- Rejected: Option 1 rejected due to weak diagnosability when promotion/drain fails.
- Scenario classes required: `happy`, `fail`, `edge`, `abuse`.
- Pass/fail observable baseline:
1. candidate promotion occurs only after healthy `GET /health/ready`;
2. old revision termination occurs after drain sequence;
3. failed admission blocks promotion and triggers rollback path.
- Traceability: `DOM-001/002/003`, `ARCH-002`, `DES-003`.
- Governing competency: `Test-Level Selection Competency`, `Reliability And Failure-Mode Competency`.

### TST-002: Evidence Matrix As First-Class Artifact
- Phase/Gate: `Phase 2`, target `G2`
- Owner: QA + Architecture
- Context/risk: decision traceability drift between `15/20/60/70/80/90`.
- Options:
1. Free-text references per scenario.
2. Stable evidence IDs with explicit `DOM/ARCH/DES/OQ` mapping.
- Selected: **Option 2**.
- Rejected: Option 1 rejected due to low auditability and harder review diffs.
- Required outcome: every critical scenario produces an evidence artifact ID.
- Traceability: `DES-002`, `DES-003`.
- Governing competency: `Invariant And Acceptance Traceability Competency`, `Evidence Threshold Competency`.

### TST-003: Restart/Drain/Capacity Fail-Path Coverage
- Phase/Gate: `Phase 2`, target `G2`
- Owner: QA + Reliability
- Context/risk: restart storms, premature termination, and capacity baseline regression (`DOM-003/004`, `ARCH-001/002`).
- Options:
1. Validate only final steady state.
2. Validate event sequence with timestamped observables.
- Selected: **Option 2**.
- Rejected: Option 1 rejected because final-state-only checks can hide transient unsafe transitions.
- Required coverage:
1. restart during rollout never bypasses readiness gate;
2. drain timeout handling and termination ordering;
3. replica floor (`>=2`) and resource-cap evidence snapshots.
- Traceability: `DOM-003`, `DOM-004`, `ARCH-001`, `ARCH-002`, `DES-001`.
- Governing competency: `Scenario Matrix Completeness Competency`, `Reliability And Failure-Mode Competency`.

### TST-004: Execution And Gate Evidence Policy
- Phase/Gate: `Phase 2`, target `G2`
- Owner: QA + DevOps
- Context/risk: tests declared but not executable in repo/CI path.
- Options:
1. Manual smoke checklist only.
2. Command-backed baseline + targeted operational smoke evidence.
- Selected: **Option 2**.
- Rejected: Option 1 rejected due to non-deterministic merge/release quality.
- Mandatory command baseline:
1. `make fmt`
2. `make test`
3. `go vet ./...`
4. `make lint`
5. `make test-race` for concurrency/restart-sensitive changes
6. `make test-integration` for rollout/deploy-boundary checks
- Traceability: `DES-002`, `50-security-observability-devops.md`.
- Governing competency: `Quality Gates And Execution Competency`.

### TST-005: Numeric Policy Assertions After OQ Closure
- Phase/Gate: `Phase 2`, target `G2`
- Owner: QA + Platform
- Context/risk: rollout/capacity/drain thresholds are now fixed and must be enforced, not deferred.
- Options:
1. Keep threshold checks as advisory while relying only on invariant ordering checks.
2. Make threshold checks mandatory for readiness claims using fixed numeric policy.
- Selected: **Option 2**.
- Rejected: Option 1 rejected due to weak regression detection and ambiguous release readiness.
- Rule:
1. invariant checks for gate presence/order remain mandatory;
2. numeric threshold checks are mandatory for `SCN-011..SCN-013`.
- Traceability: closed in `80` (`OQ-001`, `OQ-002`, `OQ-003`).
- Governing competency: `Reliability And Failure-Mode Competency`.

## Traceability To Invariants And Decisions
| Key | Covered by scenarios | Evidence IDs | Notes |
|---|---|---|---|
| `DOM-001` | `SCN-001`, `SCN-002` | `EVID-001`, `EVID-008` | CI gate strictness |
| `DOM-002` | `SCN-001`, `SCN-003`, `SCN-006` | `EVID-002` | readiness admission |
| `DOM-003` | `SCN-004`, `SCN-005`, `SCN-013` | `EVID-003` | drain-first replacement |
| `DOM-004` | `SCN-007`, `SCN-012` | `EVID-004` | replica/capacity baseline |
| `DOM-005` | `SCN-009` | `EVID-005` | config drift governance |
| `DOM-006` | `SCN-010` | `EVID-006` | networking exposure policy |
| `ARCH-001` | `SCN-007`, `SCN-012` | `EVID-004` | topology/capacity baseline |
| `ARCH-002` | `SCN-001`, `SCN-003`, `SCN-004`, `SCN-005`, `SCN-013` | `EVID-002`, `EVID-003` | rollout admission and replacement safety |
| `ARCH-004` | `SCN-009` | `EVID-005` | config-as-code source of truth |
| `DES-001` | `SCN-004`, `SCN-007` | `EVID-003`, `EVID-004` | minimal control surface |
| `DES-002` | `SCN-009` | `EVID-005` | single ownership chain |
| `DES-003` | `SCN-001`, `SCN-004`, `SCN-007` | `EVID-002`, `EVID-003`, `EVID-004` | complexity-safe sequencing |
| `SEC-001` | `SCN-010`, `SCN-019` | `EVID-006`, `EVID-014` | private-by-default ingress and controlled egress baseline |
| `SEC-002` | `SCN-010`, `SCN-018` | `EVID-006`, `EVID-013` | public ingress exception lifecycle |
| `SEC-003` | `SCN-019` | `EVID-014` | egress allowlist and deny-path policy |
| `OBS-001` | `SCN-014` | `EVID-009` | deploy-health signal contract |
| `OBS-002` | `SCN-015` | `EVID-010` | rollback signal and correlation continuity |
| `OBS-003` | `SCN-016` | `EVID-011` | config-drift detect/reconcile signal pair |
| `OBS-004` | `SCN-017` | `EVID-012` | SLI/alert routing and budget-state signal evidence |
| `OQ-001` | `SCN-011` | `EVID-007` | closed in `80`; SLO threshold tests are active |
| `OQ-002` | `SCN-012` | `EVID-007`, `EVID-004` | closed in `80`; cap baseline assertions are active |
| `OQ-003` | `SCN-013` | `EVID-007`, `EVID-003` | closed in `80`; drain/restart threshold assertions are active |
| `OQ-006` | `SCN-009` | `EVID-005` | reframed-closed in `80`; no API-scope blocker in current pass |

## Scenario Matrix (Happy/Fail/Edge/Abuse)
| Scenario ID | Class | Level | Preconditions | Expected Observable |
|---|---|---|---|---|
| `SCN-001` | happy | integration + e2e-smoke | CI success, candidate starts | promotion occurs only after `GET /health/ready` healthy |
| `SCN-002` | fail | integration | CI failed/unknown | deployment is not admitted |
| `SCN-003` | fail | integration + contract | candidate readiness failing | candidate not promoted; rollout marked failed |
| `SCN-004` | happy | integration | replacement rollout with overlap enabled | old revision serves until drain starts, then terminates |
| `SCN-005` | edge | integration | candidate restarts during rollout | no bypass of readiness gate; promotion waits for healthy state |
| `SCN-006` | abuse | contract + integration | wrong healthcheck path configured | rollout admission fails deterministically |
| `SCN-007` | fail | integration | replica floor reduced below `2` | environment marked degraded and rollback/scale action triggered |
| `SCN-008` | edge/concurrency | integration | two close deploy triggers | deterministic active revision (latest CI-passed commit) |
| `SCN-009` | abuse | integration | UI-only deploy policy change | drift detected against `railway.toml`, release readiness blocked |
| `SCN-010` | abuse/security | integration + e2e-smoke | public ingress enabled without approval | policy violation raised and revert path triggered |
| `SCN-011` | edge | integration + policy check | deploy-health SLO policy active (`99.5% / 28d`) | SLO packet confirms target and budget-state routing thresholds |
| `SCN-012` | edge | integration | capacity baseline policy active | baseline remains `>=2` replicas with per-replica caps `2 vCPU` / `2 GiB` |
| `SCN-013` | edge | integration | drain/restart numeric policy active | promotion timeout `<=180s`, drain `<=45s`, graceful shutdown timeout `30s`, restart retries `<=5` before rollback-required handling |
| `SCN-014` | happy | integration | successful rollout attempt in production-like env | `deploy_health_check` log emitted; `deploy_health_admission_total{result="success"}` increments; `deploy.health.admission` span present |
| `SCN-015` | fail | integration | forced rollback path after failed admission | `rollback_execution` event is correlated by `rollout_id`; rollback metrics and span emitted |
| `SCN-016` | abuse | integration | drift introduced via out-of-band config change | `config_drift_detected` event emitted, `config_drift_open=1`, then reconcile emits close event and gauge resets |
| `SCN-017` | edge | integration + policy check | low-volume deployment period | alert routing follows `OBS-004` floors (no noisy page below floor, ticket/page escalation works as specified) |
| `SCN-018` | happy/security | integration + policy check | approved temporary public ingress exception is active | exception metadata (`owner`, `scope`, `expiry`, `rollback plan`) exists, exposure snapshot matches approved scope, expiry enforcement is auditable |
| `SCN-019` | abuse/security | integration | outbound call attempts disallowed public host/scheme | call denied fail-closed, `network_egress_policy_violation` event emitted, no policy-bypass retry |

## Reliability And Failure-Mode Coverage
1. Admission reliability:
- readiness gate enforced (`SCN-001/003/006`).
2. Replacement reliability:
- drain-first termination ordering (`SCN-004/005/013`).
3. Capacity reliability:
- replica floor and cap-policy evidence (`SCN-007/012`).
4. Governance reliability:
- drift detection and rollback readiness (`SCN-009`).
5. Networking reliability/security posture:
- private-by-default ingress plus controlled egress enforcement (`SCN-010`, `SCN-018`, `SCN-019`).

## Contract/API Coverage
1. No new product API contract in scope (`30` remains no-change).
2. Contract checks required for existing endpoints used by rollout control:
- `GET /health/live` reachable during normal/degraded rollout.
- `GET /health/ready` strictly gates promotion.
3. Negative contract behavior:
- unhealthy readiness state must not be interpreted as promotable.

## Data/Cache Consistency And Migration Coverage
Status: no changes required.

Justification: current scope has no data model/cache/migration change; retain explicit no-op coverage reference to `40` under `TST-004`.

## Security/Observability Verification Obligations
1. Security:
- verify unapproved public ingress is detected and treated as policy violation (`SCN-010`).
- verify ingress exception lifecycle is evidence-backed and expiry-enforced (`SCN-018`).
- verify disallowed outbound target is blocked fail-closed with audit evidence (`SCN-019`).
2. Observability:
- deployment evidence must include readiness transitions, restart events, and drain ordering.
- deploy health, rollback, and drift signals must satisfy `OBS-001..OBS-004` contracts with bounded labels.
3. Required evidence artifacts:
- `EVID-001`: CI status bound to deployed commit.
- `EVID-002`: readiness gate and promotion timeline.
- `EVID-003`: drain sequence and termination ordering timeline.
- `EVID-004`: replica floor and resource-cap snapshot.
- `EVID-005`: `railway.toml` diff + PR review trail + active settings snapshot.
- `EVID-006`: networking exposure snapshot + exception approval record (if any).
- `EVID-007`: numeric-threshold assertion packet (`SCN-011..SCN-013`).
- `EVID-008`: deploy rejection evidence for failed/unknown CI state.
- `EVID-009`: deploy-health signal packet (`deploy_health_check` log + admission metric sample + trace/span reference).
- `EVID-010`: rollback correlation packet (`rollback_execution` log + rollback metrics + linked span reference).
- `EVID-011`: drift lifecycle packet (detected/reconciled logs + `config_drift_open` transition + reconcile duration metric).
- `EVID-012`: SLI/alert routing packet (deploy/rollback/drift SLI values, budget state, and alert routing outcome for the window).
- `EVID-013`: networking exception packet (approval record + scope + expiry + exposure snapshot + close/revert evidence).
- `EVID-014`: egress deny packet (blocked target evidence + `network_egress_policy_violation` event + no-bypass retry proof).

## Quality Checks And Execution Expectations
1. Mandatory command path before readiness claim:
- `make fmt`
- `make test`
- `go vet ./...`
- `make lint`
2. Conditional command path:
- `make test-race` when rollout/restart orchestration concurrency is changed.
- `make test-integration` for deployment-boundary scenario execution.
3. If API contract/migrations change in later passes:
- `make openapi-check`
- `make migration-validate` (or repository equivalent once added).

## Residual Risks And Reopen Criteria
1. Reopen if deploy-health SLO target/budget thresholds are changed from current `OBS-004` policy.
2. Reopen if replica baseline or per-replica caps change from `2` replicas / `2 vCPU` / `2 GiB`.
3. Reopen if rollout/restart/drain numeric policy changes from `180s` / `45s` / `30s` / `5` retries.
4. Reopen this plan if `ARCH-001/002/004` or `DES-001/002/003/004` materially change.
