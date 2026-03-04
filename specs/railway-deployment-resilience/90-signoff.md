# 90 Signoff

## Status
Phase 2 test-strategy, devops-gate, observability, security, and architecture consolidation passes are completed (`go-qa-tester-spec`, `go-devops-spec`, `go-observability-engineer-spec`, `go-security-spec`, `go-architect-spec`).

Design sanity pass (`go-design-spec`) applied for simplicity and cross-artifact coherence; integrated pre-G2 consistency sweep completed.
`80-open-questions.md` has no active blockers; this package is `G2`-ready for transition to `Phase 2.5`.

## Accepted Design Decisions
1. `DES-001`: keep minimal deployment control surface; do not introduce extra control planes in this phase.
2. `DES-002`: preserve single ownership chain (`ARCH -> DOM -> WP/TEST/OQ`) for change locality and auditability.
3. `DES-003`: enforce complexity-safe sequencing (rollout safety first, then capacity/networking decisions).
4. `DES-004`: keep one canonical traceability vocabulary (`SCN-*`/`EVID-*`) and synchronized OQ-closure log across artifacts.

## Accepted Invariant Decisions
1. `DOM-101`: promotion is readiness-gated (`/health/ready`) and cannot bypass admission checks.
2. `DOM-102`: replacement must preserve drain-first termination semantics.
3. `DOM-103`: deployment policy changes must be repo-reviewable via config-as-code.

## Accepted Invariant Register Baseline
1. `DOM-001`: CI gate is mandatory before deploy.
2. `DOM-002`: unhealthy candidate must not be promoted.
3. `DOM-003`: old revision must drain before termination.
4. `DOM-004`: production minimum replica baseline is `2`.
5. `DOM-005`: policy drift outside repo review is non-compliant.
6. `DOM-006`: private networking default with explicit exception path for public ingress/egress.

## Accepted Testing Decisions
1. `TST-001`: rollout safety is proven by `integration + contract + e2e-smoke`, not e2e-only.
2. `TST-002`: evidence matrix with stable evidence IDs is mandatory for `DOM/ARCH/DES/OQ` traceability.
3. `TST-003`: restart/drain/capacity reliability requires event-sequence assertions, not final-state-only checks.
4. `TST-004`: command-backed quality path is mandatory (`make fmt`, `make test`, `go vet ./...`, `make lint`, plus conditional race/integration checks).
5. `TST-005`: numeric policy assertions for SLO/capacity/drain-restart thresholds are mandatory (`SCN-011..SCN-013`), not blocked.

## Accepted DevOps Decisions
1. `DOPS-001`: formal four-tier delivery policy (`fast-path`, `full`, `nightly`, `release`) is accepted with fail-closed merge/release semantics.
2. `DOPS-002`: Railway deployment policy is config-as-code-first (`railway.toml`) with mandatory rollout/rollback evidence bundle.
3. `DOPS-003`: release trust requires Trivy pass, SBOM, keyless signature, and provenance attestation.
4. `DOPS-004`: container/runtime baseline remains minimal and hardened (multi-stage build, distroless non-root runtime, reproducible build defaults).
5. `DOPS-005`: branch-protection governance remains repository-native (no external control plane introduced).

## Accepted Security Decisions
1. `SEC-001`: networking baseline is private-by-default with fail-closed posture for unapproved ingress/egress exposure.
2. `SEC-002`: public ingress is allowed only through explicit, time-bounded exception workflow with mandatory approval and rollback metadata.
3. `SEC-003`: public egress uses allowlist-first baseline; disallowed targets are denied fail-closed with auditable security events.

## Accepted Observability Decisions
1. `OBS-001`: deploy-health admission is observable via bounded logs/metrics/traces with explicit correlation keys.
2. `OBS-002`: rollback lifecycle requires correlated telemetry chain (failed rollout -> rollback execution -> post-check outcome).
3. `OBS-003`: config-drift detect/reconcile lifecycle is observable and tied to release-readiness policy.
4. `OBS-004`: minimal SLI/SLO and alert-routing policy is defined for deploy-health, rollback, and drift without expanding observability scope.

## Closed/Reframed Open Questions
1. `OQ-001` closed: deploy-health SLO target fixed at `99.5% / 28d`.
2. `OQ-002` closed: baseline fixed at `2` replicas with per-replica caps `2 vCPU` / `2 GiB`.
3. `OQ-003` closed: rollout/restart/drain numeric policy fixed (`180s` promotion timeout, `45s` drain, `30s` shutdown timeout, `5` restart retries).
4. `OQ-006` reframed-closed: `openapi-breaking` remains PR compatibility evidence, not a global required context in current no-API-change scope.

## Reopen Criteria
1. Deploy-health SLO target or budget thresholds change from `99.5% / 28d` and current burn-rate routing policy.
2. Capacity baseline changes from `2` replicas and `2 vCPU` / `2 GiB` per replica.
3. Rollout/restart/drain numeric policy changes from `180s` / `45s` / `30s` / `5 retries`.
4. Any change to private-by-default networking baseline, ingress exception contract, or egress allowlist/exception policy requires reopening `SEC-001..SEC-003`.
5. Any change to `ARCH-001/002/004` or `DES-001/002/003/004` invalidates `TST-001..TST-005` traceability and requires `70` refresh.
6. Change to required CI check set, branch protection model, or release trust evidence policy requires reopening `DOPS-001..DOPS-005`.
7. Any API-scope expansion that requires `openapi-breaking` as a global required context requires devops-gate reopen.
8. Any change to deploy-health/rollback/drift signal contract or metric label policy requires reopening `OBS-001..OBS-004`.

## Explicit Deviation Log
No deviation from simplicity-first posture: no multi-region topology, no custom rollout control plane, no extra orchestrators introduced.

## Blocker Check
1. No unresolved cross-artifact contradictions detected.
2. No hidden "decide in coding" system-level design gaps detected.
3. No active open-question blockers remain in `80`; threshold scenarios `SCN-011..SCN-013` are now mandatory checks.
