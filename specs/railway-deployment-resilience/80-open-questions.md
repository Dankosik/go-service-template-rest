# 80 Open Questions

## Phase Target
- Current phase: `Phase 2`
- Target gate: `G2`
- Purpose: closure log for previously open questions; no active blockers remain.

## Active Open Questions
None.

## Closure Register

### OQ-001: Deploy-Health SLO Target For Release Governance
- Status: `closed`
- Owner: Observability + Platform
- Decision:
1. Lock `deploy_health_admission_ratio` target to `99.5% / 28d`.
2. Keep budget states: `green <= 25%`, `yellow <= 50%`, `orange <= 100%`, `red > 100%` consumed budget.
3. Keep burn-rate routing/floors from `OBS-004` as normative release-governance policy.
- Why this closes the question:
1. SLO target and threshold policy are now explicit and testable in `70`.
2. No additional scope expansion is required.
- Synced artifacts:
1. `50-security-observability-devops.md` (`OBS-004` SLO/alert section).
2. `55-reliability-and-resilience.md` (reliability requirements and residual risks).
3. `70-test-plan.md` (`SCN-011`, `EVID-007`).
4. `90-signoff.md` (accepted/reopen criteria updates).

### OQ-002: Per-Replica Capacity Baseline
- Status: `closed`
- Owner: Platform + Reliability
- Decision:
1. Production baseline remains `2` replicas minimum.
2. Per-replica baseline caps are locked to `CPU 2 vCPU` and `Memory 2 GiB`.
3. Any increase above this baseline requires explicit spec reopen with updated evidence.
- Why this closes the question:
1. Capacity baseline is now deterministic and enforceable through config-as-code and tests.
2. Prevents uncontrolled cost/performance drift from previous max-plan defaults.
- Synced artifacts:
1. `55-reliability-and-resilience.md` (requirements/risk updates).
2. `60-implementation-plan.md` (`WP-2`, verification hooks).
3. `70-test-plan.md` (`SCN-012`, `EVID-004`, `EVID-007`).
4. `90-signoff.md` (reopen criteria updates).

### OQ-003: Rollout/Restart/Drain Numeric Policy
- Status: `closed`
- Owner: Reliability + Platform + Service Owner
- Decision:
1. Candidate readiness promotion timeout is locked to `180s`.
2. Drain window before old-revision termination is locked to `45s`.
3. Application graceful shutdown timeout baseline is locked to `30s` (drain window keeps safety buffer).
4. Restart policy remains `On Failure` with max retries locked to `5` before rollback-required handling.
- Why this closes the question:
1. Rollout, drain, and restart behavior now has explicit numeric contracts.
2. Thresholds are directly testable and tied to rollback evidence.
- Synced artifacts:
1. `15-domain-invariants-and-acceptance.md` (timeout handling and traceability notes).
2. `55-reliability-and-resilience.md` (numeric requirements).
3. `60-implementation-plan.md` (`WP-1`, `WP-5`).
4. `70-test-plan.md` (`SCN-013`, `EVID-003`, `EVID-007`).
5. `90-signoff.md` (reopen criteria updates).

### OQ-006: `openapi-breaking` Required-Context Policy
- Status: `reframed_closed`
- Owner: DevOps + Architecture
- Decision:
1. Required merge contexts stay as defined in branch protection (`openapi-contract` is required).
2. `openapi-breaking` remains PR-executed compatibility evidence, but is not promoted to global required context in this scope.
3. For this feature, API contract is unchanged, so this question is no longer a `G2` blocker.
- Why this closes/reframes the question:
1. No API surface change is in scope, so making `openapi-breaking` globally required now would add policy complexity without risk reduction.
2. Reopen trigger is explicit and bounded to API-scope changes.
- Synced artifacts:
1. `50-security-observability-devops.md` (gate policy wording).
2. `90-signoff.md` (reopen criteria updates).

## Gate Assessment Input
1. `80` has no active open questions.
2. Previously blocking OQs are either closed (`OQ-001..003`) or reframed and closed as non-blocking for current scope (`OQ-006`).
3. This satisfies `G2` criterion "open questions are closed" for current Phase 2 package.
