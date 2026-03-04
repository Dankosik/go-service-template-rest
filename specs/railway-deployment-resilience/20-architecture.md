# 20 Architecture

## Phase Target
- Current phase: `Phase 2`
- Target gate: `G2`
- Focus: integrated design sanity pass for Railway hardening package, preserving simplicity-first baseline and removing cross-artifact drift before coding.

## Context And Constraints
1. Service is an internal sidecar and must remain private-by-default.
2. Deployment path already exists: GitHub `main` -> Railway with `Wait for CI`.
3. Current gaps are operational (health-gated rollout, scale baseline, config drift control), not business-logic gaps.
4. Existing runtime supports `GET /health/live` and `GET /health/ready`.

## Boundaries And Ownership
1. **Service boundary**
- Keep one deployable unit: `privacy-sanitization-service` in Railway.
- No extraction to additional services/modules for this scope.

2. **Ownership boundary**
- Service team owns runtime behavior and graceful shutdown semantics.
- Platform/DevOps owns Railway environment policy and deployment controls.
- Security owns ingress/egress exposure policy and exception governance.

3. **Source-of-truth boundary**
- Runtime app behavior: repository code + config contract.
- Deployment settings: `railway.toml` (once adopted) + Railway secret variables.

## Interaction Style And Consistency Frame
1. **Interaction style**
- No new sync hops are introduced in this feature.
- No async/event workflow is introduced for deployment control.

2. **Consistency model**
- No cross-service transaction design is needed.
- Deployment policy consistency is configuration consistency (repo-reviewed desired state), not distributed ACID.

## Architectural Decisions

### ARCH-001: Topology Baseline For Availability
- Owner: Platform + Service owner
- Context: current state is single region, one replica, which is fragile during restart/deploy windows.
- Options:
1. Single region, fixed minimum `2` replicas.
2. Single region, one replica.
3. Multi-region active-active from start.
- Selected: **Option 1**.
- Rejected:
1. Option 2 rejected due to avoidable availability risk.
2. Option 3 rejected as premature complexity/cost without SLO evidence.
- Trade-offs:
1. Gain: simpler HA improvement with low operational overhead.
2. Loss: no regional disaster tolerance yet.
- Cross-domain impact:
1. API: no contract change.
2. Data: no schema/cache change.
3. Security: unchanged trust boundary.
4. Operability: lower restart/deploy outage probability.
- Risks and controls:
1. Risk: under/over-provisioning.
2. Control: set conservative resource caps, then tune from metrics.
- Reopen conditions:
1. Repeated region-level incidents.
2. Approved SLO requires regional redundancy.
3. Capacity profile exceeds single-region envelope.

### ARCH-002: Rollout Admission And Replacement Safety
- Owner: Platform + Reliability
- Context: current deployment has no explicit healthcheck path and teardown overlap is disabled.
- Options:
1. Health-gated rolling replacement with teardown overlap/drain.
2. Immediate replacement without health gate.
3. External progressive-delivery control plane (custom canary orchestration).
- Selected: **Option 1**.
- Rejected:
1. Option 2 rejected due to rollout risk and request-loss windows.
2. Option 3 rejected due to overengineering for current service size.
- Trade-offs:
1. Gain: safer rollouts with minimal mechanism count.
2. Loss: slightly slower deployment completion.
- Cross-domain impact:
1. API: no payload changes; readiness path usage only.
2. Data: none.
3. Security: unchanged.
4. Operability: deterministic rollout admission criteria.
- Risks and controls:
1. Risk: wrong health endpoint semantics.
2. Control: pin healthcheck to `/health/ready`, validate in smoke checks.
- Reopen conditions:
1. Frequent false-positive/false-negative readiness behavior.
2. Rollout time becomes unacceptable for release frequency.

### ARCH-003: Canonical Build Strategy
- Owner: Platform + Service owner
- Context: Railway uses Railpack by default, while CI/CD pipeline already builds and signs Docker image from `build/docker/Dockerfile`.
- Options:
1. Keep Railpack as canonical production build path.
2. Use repository Dockerfile as canonical production build path.
3. Introduce custom external build system.
- Selected: **Option 2**.
- Rejected:
1. Option 1 rejected due to potential drift from hardened CI image path.
2. Option 3 rejected as unnecessary complexity.
- Trade-offs:
1. Gain: reproducibility and parity with CI artifact policy.
2. Loss: tighter dependency on Dockerfile maintenance discipline.
- Cross-domain impact:
1. API: none.
2. Data: none.
3. Security: better runtime hardening consistency (non-root, distroless, STOPSIGNAL).
4. Operability: fewer "works in CI, differs in Railway build" cases.
- Risks and controls:
1. Risk: misconfigured Railway build source during migration.
2. Control: staged switch in non-prod environment first, then prod.
- Reopen conditions:
1. Railway-native builder gains required hardening parity with lower operational cost.

### ARCH-004: Deployment Configuration Source Of Truth
- Owner: Platform + DevOps
- Context: current deployment settings are primarily UI-managed and can drift silently.
- Options:
1. Keep UI-managed settings as operational source of truth.
2. Adopt `railway.toml` as config-as-code for deploy settings, keep secrets in Railway variables.
3. Adopt full Terraform stack immediately for all Railway resources.
- Selected: **Option 2**.
- Rejected:
1. Option 1 rejected due to weak reviewability and drift risk.
2. Option 3 rejected as unnecessary scope expansion now.
- Trade-offs:
1. Gain: reviewable, reproducible deployment controls with minimal process overhead.
2. Loss: requires discipline to keep file and environment aligned during transition.
- Cross-domain impact:
1. API: none.
2. Data: none.
3. Security: clearer secret/non-secret separation.
4. Operability: lower config drift and faster auditability.
- Risks and controls:
1. Risk: partial migration leaves split authority.
2. Control: define one-time cutover checklist and ownership.
- Reopen conditions:
1. Multi-service fleet scale requires centralized IaC beyond `railway.toml`.

## Architecture Risks And Trade-Off Summary
1. Single-region baseline is intentionally simple and fast to adopt, but not DR-grade.
2. Dockerfile canonical build improves determinism but adds upkeep requirements.
3. Health-gated overlap rollout reduces incident risk at cost of slower deploy completion.

## Design Integrity Pass (Go-Design-Spec)

### DES-001: Keep Deployment Control Surface Minimal
- Owner: Architecture + Platform
- Context and complexity symptom: multiple optional platform features can create configuration sprawl and unclear ownership.
- Options:
1. Minimal control set now: CI gate + readiness-gated rollout + drain overlap + config-as-code.
2. Extended control stack now: external progressive delivery controller, multi-region failover, additional IaC layer.
- Selected: **Option 1**.
- Rejected: Option 2 rejected as overengineering for current service scale and current single-service risk profile.
- Trade-offs:
1. Simplicity gain: fewer moving parts and faster operational adoption.
2. Flexibility loss: fewer advanced controls available immediately.
- Acceptance boundaries:
1. All four minimal controls are configured and reviewable.
2. No additional control plane/tooling introduced in this phase.
- Affected artifacts:
1. `20`: updated
2. `60`: updated
3. `80`: updated
4. `90`: updated
- Reopen conditions:
1. Current controls fail to meet approved SLO.
2. Recurrent rollout incidents require stronger mechanism.

### DES-002: Preserve Single Ownership Chain For Policy Changes
- Owner: Architecture + DevOps
- Context and complexity symptom: split authority between UI edits and repo docs can create hidden drift and high cognitive load.
- Options:
1. Single ownership chain: `ARCH -> DOM -> WP/TEST/OQ` with `railway.toml` as deploy-policy source of truth.
2. Dual ownership chain: UI policy as runtime truth + repo docs as advisory.
- Selected: **Option 1**.
- Rejected: Option 2 rejected due to maintainability and auditability drift risk.
- Trade-offs:
1. Simplicity gain: predictable locality of change and review path.
2. Cost: stricter discipline required for policy updates.
- Acceptance boundaries:
1. Any deploy-policy change has repository delta and linked decision IDs.
2. Drift is treated as blocker, not normal state.
- Affected artifacts:
1. `20`: updated
2. `15`: updated
3. `60`: updated
4. `70`: updated
5. `90`: updated
- Reopen conditions:
1. Organizational policy mandates additional centralized IaC governance.

### DES-003: Enforce Complexity-Safe Sequencing
- Owner: Architecture + Reliability
- Context and complexity symptom: parallel changes to rollout, capacity, and exposure settings increase rollback ambiguity.
- Options:
1. Sequence by risk: rollout safety controls first, then replica/capacity tuning, then optional networking exceptions.
2. Execute all tracks in parallel.
- Selected: **Option 1**.
- Rejected: Option 2 rejected due to larger blast radius and harder root-cause analysis.
- Trade-offs:
1. Simplicity gain: clearer failure attribution and rollback control.
2. Cost: potentially slower full completion timeline.
- Acceptance boundaries:
1. Rollout gate hardening completed before capacity/networking exceptions.
2. Each step has explicit verification and rollback trigger.
- Affected artifacts:
1. `60`: updated
2. `70`: updated
3. `80`: updated
4. `90`: updated
- Reopen conditions:
1. Delivery pressure requires safe parallelization with proven automation.

### DES-004: Keep One Canonical Traceability Vocabulary Across `15/60/70/80/90`
- Owner: Architecture + QA
- Context and complexity symptom: mixed legacy test identifiers (`TEST-DOM-*`) and stale OQ links increase audit noise and make pre-G2 validation harder.
- Options:
1. Keep mixed identifier styles and reconcile manually during implementation.
2. Normalize traceability to active scenario/evidence IDs (`SCN-*`, `EVID-*`) and current open-question set only.
- Selected: **Option 2**.
- Rejected: Option 1 rejected due to higher review cost and hidden mismatch risk.
- Trade-offs:
1. Simplicity gain: one traceability language for specs, gates, and evidence.
2. Cost: small one-time edit overhead in domain/design docs.
- Acceptance boundaries:
1. `15` references active `SCN-*`/`EVID-*` IDs from `70`.
2. Closed questions are not treated as active blockers in invariant traceability.
- Affected artifacts:
1. `15`: updated
2. `70`: updated
3. `80`: updated
4. `90`: updated
- Reopen conditions:
1. test strategy format changes and requires replacing `SCN/EVID` indexing.

## Artifact Update Matrix
| Artifact | Status | Linked Decisions | Note |
|---|---|---|---|
| `15-domain-invariants-and-acceptance.md` | updated | `DES-002`, `DES-004` | ownership and traceability chain are explicit and aligned to active `SCN/EVID` IDs |
| `20-architecture.md` | updated | `DES-001`, `DES-002`, `DES-003`, `DES-004` | simplicity and consistency decisions recorded |
| `30-api-contract.md` | no changes required | `DES-001`, `DES-004` | no API behavior expansion in this scope and traceability alignment does not alter contract semantics |
| `40-data-consistency-cache.md` | no changes required | `DES-001`, `DES-004` | no datastore/cache contract drift and no traceability-driven data-model impact |
| `50-security-observability-devops.md` | updated | `DES-001`, `DES-002`, `DES-004` | policy control surface and evidence vocabulary are governance-aligned |
| `55-reliability-and-resilience.md` | updated | `DES-001`, `DES-003` | rollout/failure controls aligned to simplicity-first baseline |
| `60-implementation-plan.md` | updated | `DES-001`, `DES-002`, `DES-003`, `DES-004` | complexity-safe sequence and canonical traceability enforced |
| `70-test-plan.md` | updated | `DES-002`, `DES-003`, `DES-004` | traceability and stage-wise verification aligned to canonical IDs |
| `80-open-questions.md` | updated | `DES-001`, `DES-003`, `DES-004` | active blockers are closed/reframed; file keeps closure audit log only |
| `90-signoff.md` | updated | `DES-001`, `DES-002`, `DES-003`, `DES-004` | accepted design decisions and reopen criteria include traceability-integrity guardrail |
