# 15 Domain Invariants And Acceptance

## Phase Target
- Current phase: `Phase 2`
- Target gate: `G2`
- Scope: deployment-policy behavior for Railway production runtime, without changing product API/domain logic.

## Domain Terms And Scope
1. **Stable revision**: currently serving healthy deployment revision.
2. **Candidate revision**: new revision being deployed and validated.
3. **Promotion**: candidate becomes active serving revision.
4. **Drain window**: overlap period where old revision remains available while candidate proves readiness.
5. **Policy snapshot**: repository-reviewed deployment settings (`railway.toml`) plus Railway-managed secrets.

Scope boundaries:
1. In scope: rollout policy, deployment admission, replacement safety, config reproducibility, networking exposure governance.
2. Out of scope: sanitization business rules, detector logic, schema/migration mechanics, multi-service workflow design.

## Invariant Register
| DOM ID | Invariant Class | Owner Role / Owner Service | Source Of Truth | Enforcement Point(s) | Verifiable Pass/Fail Rule |
|---|---|---|---|---|---|
| DOM-001 | `local_hard_invariant` | Platform / `privacy-sanitization-service` deploy policy | GitHub Actions status for target commit | Railway `Wait for CI` gate before deploy start | **Pass**: deploy starts only after successful CI for commit. **Fail**: deploy starts from failed/unknown CI state. |
| DOM-002 | `local_hard_invariant` | Reliability / service runtime + Railway deploy controller | Runtime readiness endpoint (`GET /health/ready`) | Railway `Healthcheck Path` admission gate | **Pass**: candidate is promoted only after readiness success. **Fail**: candidate is promoted while readiness fails/unknown. |
| DOM-003 | `local_hard_invariant` | Service owner / runtime lifecycle | Runtime drain state + shutdown timeout contract | Teardown overlap + app shutdown sequence | **Pass**: old revision stops receiving traffic before termination and exits after drain. **Fail**: old revision terminated while still traffic-eligible. |
| DOM-004 | `local_hard_invariant` | Platform / production environment policy | Production scale policy | Railway scale settings | **Pass**: production baseline runs with minimum `2` replicas in region. **Fail**: baseline falls to single replica without approved exception. |
| DOM-005 | `local_hard_invariant` | DevOps / deployment governance | `railway.toml` + PR history | Config-as-code registration and review flow | **Pass**: deploy policy changes are traceable in repository review history. **Fail**: UI-only policy drift bypasses repo review. |
| DOM-006 | `local_hard_invariant` | Security / platform networking | Networking policy decision | Railway networking exposure settings + egress policy allowlist | **Pass**: private networking is default; public ingress/egress only via explicit approved exception path. **Fail**: public exposure or outbound target allowance is enabled by implicit default. |

## Major Invariant Decisions

### DOM-101: Promotion Must Be Readiness-Gated
- Owner role/service: Reliability / `privacy-sanitization-service`.
- Context: rollout can promote unhealthy candidate if admission is not tied to readiness semantics.
- Options:
1. Promote on readiness success (`/health/ready`) only.
2. Promote after fixed timer regardless of readiness.
- Selected option: `1`.
- Rejected option and reason: `2` rejected due to non-deterministic outage risk.
- Transition constraints:
  - Allowed: `candidate_starting -> candidate_ready -> promoted`.
  - Forbidden: `candidate_starting -> promoted`.
- Violation semantics:
  - Sync/deploy-time: promotion rejected; deploy marked failed.
  - Async/runtime: incident signal + rollback to stable revision.
- Duplicate/replay handling:
  - Re-running same deployment attempt must not bypass readiness gate.
- Cross-domain impact:
  - API: no payload change; operational dependency on readiness path.
  - Data/distributed: no cross-service consistency impact.
  - Reliability/testing: rollout checks become mandatory.
- Rollout and rollback note:
  - Promotion remains blocked until readiness passes; rollback path must stay available.
- Test obligations:
  - `SCN-001`, `SCN-003`, `SCN-006` with evidence `EVID-002` in `70`.
- Reopen conditions:
  - readiness endpoint semantics change or become non-representative.
- Linked open questions:
  - closed in `80` (`OQ-003`).

### DOM-102: Replacement Must Preserve Drain-First Termination
- Owner role/service: Service owner / runtime lifecycle.
- Context: without overlap/drain, deploy may drop in-flight traffic.
- Options:
1. Enable teardown overlap with drain-first shutdown.
2. Immediate old-revision termination on candidate start.
- Selected option: `1`.
- Rejected option and reason: `2` rejected due to request-loss risk during replacement.
- Transition constraints:
  - Allowed: `promoted(new) + draining(old) -> old_terminated`.
  - Forbidden: `old_serving -> old_terminated` without drain step.
- Violation semantics:
  - Sync/deploy-time: replacement policy violation fails rollout gate.
  - Async/runtime: trigger rollback and operational incident review.
- Duplicate/replay handling:
  - Repeated restart cycles must still execute drain-before-stop sequence.
- Cross-domain impact:
  - API: no contract change.
  - Reliability: depends on shutdown timeout alignment.
  - Testing: requires replacement-sequence verification.
- Rollout and rollback note:
  - drain window value must exceed app-level graceful shutdown budget.
- Test obligations:
  - `SCN-004`, `SCN-005`, `SCN-013` with evidence `EVID-003` in `70`.
- Reopen conditions:
  - measured rollout time exceeds acceptable release cadence.
- Linked open questions:
  - closed in `80` (`OQ-003`).

### DOM-103: Deployment Policy Must Be Repo-Reviewable
- Owner role/service: DevOps / deployment governance.
- Context: UI-only settings can drift and break repeatability.
- Options:
1. Config-as-code (`railway.toml`) for deploy policy + Railway vars for secrets.
2. UI-managed settings as primary source of truth.
- Selected option: `1`.
- Rejected option and reason: `2` rejected due to weak auditability and drift control.
- Transition constraints:
  - Allowed: policy change through repository PR and synchronized apply.
  - Forbidden: permanent production policy change without repo artifact update.
- Violation semantics:
  - Sync/governance: release readiness blocked until policy drift reconciled.
  - Async/runtime: if drift detected, mark config incident and reconcile to source.
- Duplicate/replay handling:
  - Re-applying same config should be idempotent (same desired state, no semantic drift).
- Cross-domain impact:
  - Security: secret/non-secret boundary explicit.
  - Operability: improved reproducibility and rollback clarity.
- Rollout and rollback note:
  - rollback must include both revision rollback and policy snapshot rollback when applicable.
- Test obligations:
  - `SCN-009` with evidence `EVID-005` in `70`.
- Reopen conditions:
  - fleet governance needs exceed `railway.toml` capability.
- Linked open questions:
  - closed in `80` (`OQ-001`, `OQ-002`).

## State Transition Rules

### State Model
1. `stable`
2. `candidate_starting`
3. `candidate_ready`
4. `promoted`
5. `rollback_required`
6. `rolled_back`

### Allowed Transitions
| Transition | Trigger | Preconditions | Postconditions |
|---|---|---|---|
| `stable -> candidate_starting` | deploy initiated | `DOM-001` satisfied (CI success) | candidate revision created |
| `candidate_starting -> candidate_ready` | readiness check success | healthcheck configured and passing (`DOM-002`) | candidate eligible for promotion |
| `candidate_ready -> promoted` | rollout promotion | minimum replica baseline maintained (`DOM-004`) | candidate serves traffic |
| `promoted -> stable` | deployment finalized | old revision drained (`DOM-003`) | new stable revision established |
| `candidate_starting -> rollback_required` | readiness timeout/failure | retries exhausted / admission failed | promotion blocked |
| `rollback_required -> rolled_back` | rollback executed | previous stable revision available | service restored to known-good revision |

### Forbidden Transitions
1. `candidate_starting -> promoted` without `candidate_ready`.
2. `stable -> promoted` without candidate deployment lifecycle.
3. `promoted -> old_revision_terminated` without drain/overlap sequence.

### Timeout And Stuck-State Handling
1. If candidate does not become ready in configured rollout window, transition to `rollback_required`.
2. If rollback cannot restore stable state, escalate to manual incident handling path.
3. Numeric policy baseline: promotion timeout `180s`, drain window `45s`, app graceful shutdown timeout `30s`, restart max retries `5`.

## Acceptance Criteria
1. **Happy path**
- Deploy from `main` starts only after successful CI.
- Candidate passes `/health/ready`, is promoted, old revision drains, then termination completes safely.

2. **Forbidden path**
- Promotion is blocked when readiness fails or remains unknown.
- Policy change without repository traceability is treated as non-compliant.

3. **Fail path**
- On rollout admission failure, system transitions to rollback-required and restores last stable revision.
- On replacement safety failure, deployment is marked failed and incident path is triggered.

4. **Compatibility**
- Policy hardening does not change application business endpoints or request/response payload behavior.

## Corner Cases And Edge Conditions
1. Repeated deploy trigger for same commit must preserve gating sequence and not bypass invariants.
2. Candidate may be healthy briefly then fail; promotion must not proceed if health gate fails at admission point.
3. Partial config-as-code adoption (file exists but not bound in Railway) is treated as drift-risk condition.
4. Temporary single-replica state during operational incident is allowed only as explicit exception with owner acknowledgement.

## Invariant Violation Semantics
| Violation | Deterministic Outcome | Recovery Path |
|---|---|---|
| `DOM-001` broken (CI gate bypass) | deployment rejected; no promotion | fix CI gate and redeploy from verified commit |
| `DOM-002` broken (unhealthy promotion risk) | candidate promotion blocked or deploy failed | rollback to stable revision; investigate readiness failure |
| `DOM-003` broken (drain skipped) | rollout marked failed | revert to stable revision; enforce teardown/drain settings |
| `DOM-004` broken (replica baseline violated) | environment marked degraded | restore baseline replica count or approve temporary exception |
| `DOM-005` broken (policy drift) | release readiness blocked | reconcile Railway settings to repo source of truth |
| `DOM-006` broken (unapproved public exposure or outbound target allowance) | security non-compliance incident | disable public ingress/egress exception or formalize approved exception with expiry and rollback plan |

## Traceability To Related Artifacts
| DOM ID | 30 API | 40 Data | 55 Reliability | 60 Plan | 70 Tests | 80 Questions | 90 Signoff |
|---|---|---|---|---|---|---|---|
| DOM-001 | no change | no change | CI gate criticality | WP-1, WP-5 | `SCN-001`, `SCN-002` | closed in `80` (`OQ-001`) | invariant acceptance |
| DOM-002 | healthcheck usage note | no change | rollout admission policy | WP-1, WP-5 | `SCN-001`, `SCN-003`, `SCN-006` | closed in `80` (`OQ-003`) | invariant acceptance |
| DOM-003 | no change | no change | drain/teardown policy | WP-1, WP-5 | `SCN-004`, `SCN-005`, `SCN-013` | closed in `80` (`OQ-003`) | invariant acceptance |
| DOM-004 | no change | no change | capacity baseline risk | WP-2 | `SCN-007`, `SCN-012` | closed in `80` (`OQ-002`) | invariant acceptance |
| DOM-005 | no change | no change | governance drift control | WP-3, WP-6 | `SCN-009`, `SCN-016` | closed/reframed in `80` (`OQ-001`, `OQ-002`, `OQ-006`) | invariant acceptance |
| DOM-006 | no change | no change | exposure/fallback policy | WP-4 | `SCN-010`, `SCN-018`, `SCN-019` | closed in `80` (items 9 and 10) | invariant acceptance |
