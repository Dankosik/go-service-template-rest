# 65 Coder Detailed Plan: Railway Deployment Resilience

## Execution Context
Scope boundaries:
1. Implement Railway deployment-resilience hardening defined in frozen specs (`15/20/30/40/50/55/60/70/80/90`).
2. Preserve existing product behavior and endpoint contracts while hardening rollout, security posture, delivery gates, and evidence quality.
3. Keep implementation minimal and deterministic under `DES-001..DES-004`.

Non-goals:
1. No new product API endpoints or payload/status-contract expansion (`30` is no-change).
2. No datastore schema/cache contract changes (`40` is no-change).
3. No new external rollout/orchestration control planes, multi-region topology, or Terraform-first expansion.
4. No redesign of sanitization business logic/detector stack.

Critical invariants to preserve:
1. `DOM-001`: deploy admission remains CI-gated.
2. `DOM-002`: promotion remains readiness-gated (`/health/ready`).
3. `DOM-003`: drain-first replacement remains mandatory.
4. `DOM-004`: production baseline remains `>=2` replicas with `2 vCPU / 2 GiB` per-replica baseline.
5. `DOM-005`: deployment-policy changes remain repo-reviewable/config-as-code.
6. `DOM-006`: networking remains private-by-default with explicit exception governance.

Forbidden changes under current freeze:
1. Altering approved numeric policy (`180s`, `45s`, `30s`, `5 retries`) without `Spec Reopen`.
2. Altering deploy-health SLO policy (`99.5% / 28d`) or burn-rate routing without `Spec Reopen`.
3. Promoting `openapi-breaking` to globally required merge context in this no-API-change scope.
4. Logging secrets/tokens/raw payload fragments in deploy/rollback/network-policy diagnostics.

## Execution Mode
Mode: `batch` with atomic task execution (one task per run is still valid).

Checkpoint policy:
1. Mandatory checkpoint every `2` tasks (`CP-1..CP-4`).
2. Go/no-go decision is required before entering the next task group.
3. Any blocker ambiguity triggers the Clarification Contract and pauses only affected tasks.

Coder autonomy contract:
1. Task cards define outcomes, constraints, and evidence only.
2. Coder chooses low-level decomposition, exact file layout, and refactor shape inside approved intent.
3. Declared change surface is expected module/layer scope, not a hard file-path lock.

## Task Graph
Ordered dependency graph:
1. `RDR-T01 -> RDR-T02 -> RDR-T03 -> RDR-T04 -> RDR-T05 -> RDR-T06 -> RDR-T07 -> RDR-T08`

Dependency notes:
1. `RDR-T01` establishes policy source-of-truth baseline required by all downstream policy and evidence tasks.
2. `RDR-T02` and `RDR-T03` must land before capacity/networking hardening per `DES-003` sequencing.
3. `RDR-T07` starts only after implementation tasks (`T01..T06`) expose stable behavior contracts.
4. `RDR-T08` is final closure and reconciliation gate.

## Task Cards

### Task ID
`RDR-T01`

Objective:
1. Establish deployment-policy source-of-truth and canonical build-path parity for Railway governance.

Spec Traceability:
1. `ARCH-003`, `ARCH-004`, `DOM-005`, `DES-002`, `DOPS-002`, `DOPS-004`, `DOPS-005`.
2. Test obligations: `SCN-009`, `EVID-005`, `OQ-006` (reframed-closed constraint).

Change Surface:
1. Deployment-policy configuration layer (config-as-code artifact and related validation hooks).
2. Delivery-governance layer (workflow and branch-protection alignment surfaces).
3. Minimal docs/governance surfaces for source-of-truth clarity.

Task Sequence:
1. Add/align config-as-code deployment policy baseline with explicit ownership and non-secret policy fields.
2. Ensure canonical production build path remains repository Dockerfile-driven.
3. Align governance checks so policy drift is fail-closed and reviewable.

Verification Commands:
1. `rg -n "railway\\.toml|source of truth|policy|healthcheck|restart|replica|cpu|memory" .`
2. `rg -n "build/docker/Dockerfile|docker build" .github/workflows/cd.yml`
3. `make guardrails-check`

Expected Evidence:
1. Config-as-code policy artifact exists and is traceable in review flow.
2. Canonical build path is explicitly Dockerfile-based (`ARCH-003`).
3. `EVID-005` prerequisites are present (policy delta + governance trace).

Review Checklist:
1. Intended policy-governance behavior is implemented with fail-closed defaults.
2. No API/data contract drift is introduced.
3. Evidence hooks for drift/reconciliation are present.
4. Ambiguity state is explicit (`none` or `blocked`).

Ambiguity Triggers:
1. `contract`: if implementation requires API-surface expansion to enforce policy.
2. `other`: if Railway policy fields cannot represent frozen requirements without new control plane.
3. `devops`: if required context names cannot remain stable across CI and branch protection.

Change Reconciliation:
1. Record during execution:
   - `actual_touched_surface`
   - `deviation_from_expected` (`none` or short rationale)
   - `decision_ids_affected`

Progress Status:
`done`

### Task ID
`RDR-T02`

Objective:
1. Implement rollout-admission and drain-first replacement enforcement for deploy lifecycle safety.

Spec Traceability:
1. `DOM-001`, `DOM-002`, `DOM-003`, `ARCH-002`, `DES-003`.
2. Reliability requirements: promotion timeout/restart/drain/shutdown policy from `55`.
3. Test obligations: `SCN-001`, `SCN-002`, `SCN-003`, `SCN-004`, `SCN-005`, `SCN-006`, `SCN-013`; `EVID-002`, `EVID-003`, `EVID-008`.

Change Surface:
1. Runtime lifecycle/shutdown orchestration layer.
2. Health/readiness gating layer.
3. Deployment-policy integration points for admission and replacement sequencing.

Task Sequence:
1. Enforce readiness-gated promotion and reject unhealthy/unknown admission paths.
2. Enforce drain-before-termination ordering and graceful shutdown semantics.
3. Ensure retry/timeout policy interfaces align to frozen numeric policy.

Verification Commands:
1. `go test ./internal/app/health ./internal/infra/http`
2. `go test ./cmd/service`
3. `make test-integration`

Expected Evidence:
1. Promotion does not occur before readiness success (`DOM-002`).
2. Drain-first termination sequence is observable and deterministic (`DOM-003`).
3. Failed/unknown CI or admission path is rejected deterministically (`DOM-001`, `SCN-002`, `EVID-008`).

Review Checklist:
1. Behavior aligns with rollout safety invariants and no bypass transitions are introduced.
2. Shutdown/drain flow stays bounded and cancel-safe.
3. Test observables are sequence-aware, not final-state-only.
4. Ambiguity state is explicit.

Ambiguity Triggers:
1. `reliability`: if platform/runtime knobs cannot enforce `180s/45s/30s/5` policy as specified.
2. `invariant`: if any allowed/forbidden transition in `15` becomes unimplementable.
3. `test`: if deterministic sequence assertions cannot be produced with current harness.

Change Reconciliation:
1. Record during execution:
   - `actual_touched_surface`
   - `deviation_from_expected`
   - `sequence_assertions_added`

Progress Status:
`done`

### Task ID
`RDR-T03`

Objective:
1. Implement bounded deploy/rollback/drift observability contract and correlation continuity.

Spec Traceability:
1. `OBS-001`, `OBS-002`, `OBS-003`, `OBS-004`.
2. Linked domain/reliability decisions: `DOM-002`, `DOM-003`, `DOM-005`.
3. Test obligations: `SCN-014`, `SCN-015`, `SCN-016`, `SCN-017`; `EVID-009`, `EVID-010`, `EVID-011`, `EVID-012`.

Change Surface:
1. Telemetry instrumentation layer (logs/metrics/traces).
2. Deploy lifecycle emission points (admission, rollback, drift detect/reconcile).
3. Alert-policy evidence surfaces and correlation metadata pipeline.

Task Sequence:
1. Emit structured events and low-cardinality metrics for deploy/rollback/drift contracts.
2. Enforce correlation-key usage in logs/traces only; keep metric labels bounded.
3. Implement SLI/budget-state evidence extraction path for release-readiness packets.

Verification Commands:
1. `go test ./internal/infra/telemetry ./cmd/service`
2. `make test-integration`

Expected Evidence:
1. Signal contract names and bounded labels match `50` policy.
2. Correlation chain exists for failed rollout -> rollback -> post-check.
3. Drift lifecycle emits detect and reconcile closure evidence.

Review Checklist:
1. No high-cardinality IDs leak into metric labels.
2. No secret/raw payload logging in telemetry events.
3. Observability outputs are sufficient for machine-checkable evidence packets.
4. Ambiguity state is explicit.

Ambiguity Triggers:
1. `observability`/`other`: if existing telemetry stack cannot represent required signal contract without scope expansion.
2. `security`: if required event fields risk sensitive-data exposure.
3. `test`: if evidence packets cannot be reproduced deterministically.

Change Reconciliation:
1. Record during execution:
   - `actual_touched_surface`
   - `label_allowlist_validation_result`
   - `deviation_from_expected`

Progress Status:
`done`

### Task ID
`RDR-T04`

Objective:
1. Enforce availability baseline and numeric policy assertions for capacity/restart/drain thresholds.

Spec Traceability:
1. `DOM-004`, `ARCH-001`, `DES-003`.
2. Closed-question obligations: `OQ-001`, `OQ-002`, `OQ-003`.
3. Test obligations: `SCN-007`, `SCN-011`, `SCN-012`, `SCN-013`; `EVID-004`, `EVID-007`.

Change Surface:
1. Deployment-capacity policy layer.
2. Threshold assertion/validation layer for readiness and release evidence.
3. Reliability gating surfaces consuming numeric thresholds.

Task Sequence:
1. Encode baseline floor/caps (`>=2` replicas, `2 vCPU/2 GiB` per replica) in policy-governance path.
2. Encode/validate numeric rollout/restart/drain policy (`180s/45s/30s/5`).
3. Ensure failure/degraded handling remains explicit when baseline is violated.

Verification Commands:
1. `make test`
2. `make test-integration`
3. `rg -n "99\\.5%|180s|45s|30s|2 vCPU|2 GiB|5" specs/railway-deployment-resilience`

Expected Evidence:
1. Capacity baseline and caps are enforced and auditable.
2. Numeric assertions are machine-checkable and included in release-readiness packet.
3. `SCN-011..013` threshold checks are mandatory, not advisory.

Review Checklist:
1. Capacity policy remains minimal and does not introduce multi-region/control-plane scope.
2. Numeric policy remains identical to frozen decisions.
3. Degraded/fail-closed semantics are explicit.
4. Ambiguity state is explicit.

Ambiguity Triggers:
1. `reliability`: if platform cannot express thresholds with required precision.
2. `invariant`: if enforcing caps conflicts with existing rollout policy.
3. `other`: if required evidence collection depends on unsupported runtime signals.

Change Reconciliation:
1. Record during execution:
   - `actual_touched_surface`
   - `policy_values_applied`
   - `deviation_from_expected`

Progress Status:
`done`

### Task ID
`RDR-T05`

Objective:
1. Implement private-by-default networking policy, ingress exception lifecycle, and egress deny-path controls.

Spec Traceability:
1. `DOM-006`, `SEC-001`, `SEC-002`, `SEC-003`.
2. Reliability/security hooks from `55` and `50` threat matrix.
3. Test obligations: `SCN-010`, `SCN-018`, `SCN-019`; `EVID-006`, `EVID-013`, `EVID-014`.

Change Surface:
1. Networking policy and exception-governance layer.
2. Outbound policy enforcement layer (allowlist/scheme/deny behavior).
3. Security audit-event instrumentation surface.

Task Sequence:
1. Enforce private-by-default ingress posture with fail-closed defaults.
2. Implement explicit ingress-exception lifecycle contract (owner/reason/scope/expiry/rollback).
3. Implement egress allowlist deny-path with non-bypassable fail behavior and audit signals.

Verification Commands:
1. `make test`
2. `make test-integration`
3. `rg -n "network_ingress_policy_violation|network_egress_policy_violation|network_exception_state_change" .`

Expected Evidence:
1. Unapproved ingress/egress changes are denied fail-closed.
2. Approved ingress exception lifecycle is auditable end-to-end.
3. Egress deny-path evidence includes violation event and no-bypass retry proof.

Review Checklist:
1. No default-public exposure paths are introduced.
2. Exception metadata completeness and expiry enforcement are explicit.
3. Security events remain redacted and bounded.
4. Ambiguity state is explicit.

Ambiguity Triggers:
1. `security`: if exception workflow requires broader authz redesign.
2. `contract`: if policy enforcement requires public API behavior changes.
3. `reliability`: if deny-path handling causes unacceptable retry/availability regressions.

Change Reconciliation:
1. Record during execution:
   - `actual_touched_surface`
   - `policy_fail_closed_checks`
   - `deviation_from_expected`

Progress Status:
`done`

### Task ID
`RDR-T06`

Objective:
1. Align CI/CD merge/release gates and branch-protection governance with approved hard-stop policy.

Spec Traceability:
1. `DOPS-001`, `DOPS-002`, `DOPS-003`, `DOPS-004`, `DOPS-005`.
2. Linked constraints: `DOM-001`, `DOM-005`, `OQ-006`.
3. Test/process obligations: `TST-004`, `EVID-001..EVID-014` gate-readiness continuity.

Change Surface:
1. CI/CD workflow policy layer (full/nightly/release tiers).
2. Branch-protection governance script/policy surfaces.
3. Release-trust evidence generation and blocking semantics.

Task Sequence:
1. Align required merge contexts and fail-closed behavior across CI and branch protection.
2. Ensure release trust steps (Trivy, SBOM, signature, provenance) remain mandatory and blocking.
3. Keep `openapi-breaking` as PR evidence-only signal for this scope.

Verification Commands:
1. `rg -n "repo-integrity|lint|openapi-contract|test|test-race|test-coverage|test-integration|migration-validate|go-security|secret-scan|container-security" .github/workflows/ci.yml scripts/dev/configure-branch-protection.sh`
2. `rg -n "release-preflight|Trivy|SBOM|cosign|attest-build-provenance" .github/workflows/cd.yml`
3. `make guardrails-check`

Expected Evidence:
1. Required contexts and enforcement points remain consistent and blocking.
2. Release trust evidence chain is complete and non-bypassable.
3. Policy for `openapi-breaking` matches frozen scope decision.

Review Checklist:
1. No silent downgrade of blocking gates to informational mode.
2. No contradictory required-context definitions across governance surfaces.
3. Release trust flow is intact for both main and tag publish paths.
4. Ambiguity state is explicit.

Ambiguity Triggers:
1. `devops`: if workflow job names cannot be stabilized without broader pipeline redesign.
2. `other`: if repository governance tooling constraints prevent required checks parity.
3. `security`: if trust-evidence steps conflict with current signing/provenance infrastructure.

Change Reconciliation:
1. Record during execution:
   - `actual_touched_surface`
   - `required_context_diff`
   - `deviation_from_expected`

Progress Status:
`done`

### Task ID
`RDR-T07`

Objective:
1. Implement/align scenario and evidence tests so frozen decision coverage is executable (`SCN-001..SCN-019`, `EVID-001..EVID-014`).

Spec Traceability:
1. `TST-001`, `TST-002`, `TST-003`, `TST-004`, `TST-005`.
2. Decision/invariant coverage: `DOM-*`, `ARCH-001/002/004`, `DES-001..004`, `SEC-001..003`, `OBS-001..004`, `OQ-001/002/003/006`.

Change Surface:
1. Integration/contract/e2e-smoke test suites for deployment-policy behavior.
2. Evidence capture and traceability mapping layer.
3. Minimal helper/test-fixture surfaces needed for deterministic scenario execution.

Task Sequence:
1. Encode scenario matrix assertions for happy/fail/edge/abuse classes.
2. Implement evidence IDs and mapping outputs expected by release-readiness packet.
3. Ensure no-op coverage declarations for unchanged API/data surfaces remain explicit.

Verification Commands:
1. `make test`
2. `make test-integration`
3. `make test-race`

Expected Evidence:
1. Scenario matrix is executable with deterministic pass/fail semantics.
2. Evidence artifacts `EVID-001..EVID-014` are produced or derivable without manual reinterpretation.
3. Traceability closure from decisions/invariants to tests is explicit.

Review Checklist:
1. Tests assert sequence/timing where required, not only final state.
2. Security and observability negative paths are covered.
3. Evidence IDs are stable and cross-artifact consistent.
4. Ambiguity state is explicit.

Ambiguity Triggers:
1. `test`: if environment constraints make key scenarios non-deterministic.
2. `reliability`: if timing-sensitive assertions are flaky under current harness.
3. `security`: if negative-path security tests require infrastructure not available in current environment.

Change Reconciliation:
1. Record during execution:
   - `actual_touched_surface`
   - `scenario_ids_implemented`
   - `deviation_from_expected`

Progress Status:
`done`

### Task ID
`RDR-T08`

Objective:
1. Execute final verification baseline, reconcile touched surface, and close `G2.5` evidence package.

Spec Traceability:
1. Gate objective: `G2.5` completion requirements.
2. Consolidates all previous decision/test obligations and freeze constraints.

Change Surface:
1. Repository-wide validation and final reconciliation records.
2. Optional docs alignment updates when behavior/CI-sensitive surfaces changed.

Task Sequence:
1. Run mandatory baseline quality commands and collect outputs.
2. Run conditional checks based on actual touched surfaces (race/integration/openapi/sqlc/stringer).
3. Complete per-task reconciliation records and final coverage closure summary.

Verification Commands:
1. `make fmt`
2. `make test`
3. `go vet ./...`
4. `make lint`
5. Conditional:
   - `make test-race`
   - `make test-integration`
   - `make openapi-check`
   - `make sqlc-check`
   - `make stringer-drift-check`

Expected Evidence:
1. Mandatory checks pass and conditional checks pass where applicable.
2. No unresolved ambiguity records remain for completed tasks.
3. Coverage matrix rows are closed with command output + evidence packet references.

Review Checklist:
1. Final implementation stays inside frozen scope and preserves no-change API/data boundaries.
2. All evidence claims are command-backed.
3. Reconciliation logs explain any surface deviation from planned change areas.
4. Ambiguity state is explicit (`none` required for closure).

Ambiguity Triggers:
1. `other`: any verification command fails with behavior suggesting spec mismatch.
2. `contract`/`security`/`reliability`/`test`: unresolved contradiction discovered during final gate run.

Change Reconciliation:
1. `actual_touched_surface`:
   - verification-only execution for repository-wide quality gates;
   - reconciliation update in this artifact (`65-coder-detailed-plan.md`).
2. `deviation_from_expected`:
   - `none`; expected `RDR-T08` surface was validation + reconciliation and remained inside frozen scope.
3. `final_gate_command_results`:
   - mandatory baseline:
     - `make fmt` -> `pass`
     - `make test` -> `pass`
     - `go vet ./...` -> `pass`
     - `make lint` -> `pass`
   - conditional checks (executed for closure):
     - `make test-race` -> `pass`
     - `make test-integration` -> `pass`
     - `make openapi-check` -> `pass`
     - `make sqlc-check` -> `pass`
     - `make stringer-drift-check` -> `pass`
   - coverage-matrix supporting command:
     - `make guardrails-check` -> `pass`
4. Coverage closure summary:
   - rollout/capacity/networking/observability rows closed by `make test-integration` and `RDR-T07` matrix execution (`SCN-001..019`, `EVID-001..014`);
   - delivery-gate row closed by `make guardrails-check` + required-context parity from hardened CI/CD workflows;
   - no-change API/data row preserved: no `30` or `40` contract expansion introduced.
5. Ambiguity state:
   - `none`.

Progress Status:
`done`

## Checkpoint Plan
Cadence: every 2 tasks.

### CP-1 (after `RDR-T01`, `RDR-T02`)
Go/no-go criteria:
1. Required checks for `T01..T02` completed and evidence captured.
2. No drift from `DOM-001..003`, `ARCH-002`, `DES-003`.
3. Reconciliation entries exist for both tasks; deviations are justified.
4. No unresolved blocker ambiguity on rollout lifecycle contracts.

### CP-2 (after `RDR-T03`, `RDR-T04`)
Go/no-go criteria:
1. Signal contract (`OBS-001..004`) and numeric policy (`OQ-001..003`) evidence is present.
2. Capacity baseline and threshold assertions are deterministic.
3. Reconciliation entries for `T03..T04` are complete.
4. No unresolved blocker ambiguity on observability cardinality or threshold enforceability.

### CP-3 (after `RDR-T05`, `RDR-T06`)
Go/no-go criteria:
1. Security networking controls (`SEC-001..003`) and delivery gates (`DOPS-001..005`) are aligned.
2. Fail-closed semantics for ingress/egress and merge/release controls are preserved.
3. Reconciliation entries for `T05..T06` are complete.
4. No unresolved blocker ambiguity on governance/security exception policy.

### CP-4 (after `RDR-T07`, `RDR-T08`)
Go/no-go criteria:
1. Scenario/evidence matrix closure is complete (`SCN-001..019`, `EVID-001..014`).
2. Mandatory command baseline is green; conditional checks executed where applicable.
3. Full coverage matrix is closed with explicit evidence links.
4. Final reconciliation package is complete and no task remains `blocked`.

## Clarification Contract
Required fields:
1. `request_id`
2. `blocked_task_id`
3. `ambiguity_type` (`contract`, `invariant`, `security`, `reliability`, `test`, `other`)
4. `conflicting_sources`
5. `decision_impact`
6. `proposed_options`
7. `owner`
8. `resume_condition`

Resolution policy:
1. Block only affected task(s); unrelated tasks may continue if dependency-safe.
2. No blocked task may resume until `resume_condition` is explicitly satisfied.
3. If ambiguity implies frozen-decision contradiction, escalate to `Spec Clarification Request` or `Spec Reopen`.
4. Every clarification outcome must update reconciliation notes for impacted task cards.

## Coverage Matrix
| Obligation Cluster | Source IDs | Task Coverage | Evidence/Checks |
|---|---|---|---|
| Rollout admission and replacement safety | `DOM-001`, `DOM-002`, `DOM-003`, `ARCH-002`, `DES-003` | `RDR-T02`, `RDR-T07`, `RDR-T08` | `SCN-001..006`, `SCN-013`; `EVID-002`, `EVID-003`, `EVID-008`; `make test-integration` |
| Capacity baseline and numeric thresholds | `DOM-004`, `ARCH-001`, `OQ-001`, `OQ-002`, `OQ-003`, `TST-005` | `RDR-T04`, `RDR-T07`, `RDR-T08` | `SCN-007`, `SCN-011..013`; `EVID-004`, `EVID-007`; threshold checks |
| Config-as-code governance and drift closure | `DOM-005`, `ARCH-004`, `DES-002`, `DOPS-002`, `OBS-003`, `OQ-006` | `RDR-T01`, `RDR-T03`, `RDR-T06`, `RDR-T07` | `SCN-009`, `SCN-016`; `EVID-005`, `EVID-011`; guardrails checks |
| Networking security posture and exception lifecycle | `DOM-006`, `SEC-001`, `SEC-002`, `SEC-003` | `RDR-T05`, `RDR-T07`, `RDR-T08` | `SCN-010`, `SCN-018`, `SCN-019`; `EVID-006`, `EVID-013`, `EVID-014`; integration/security assertions |
| Delivery gate and release trust hard-stops | `DOPS-001`, `DOPS-003`, `DOPS-004`, `DOPS-005`, `TST-004` | `RDR-T01`, `RDR-T06`, `RDR-T08` | required-context parity, release-preflight trust steps, `make guardrails-check` |
| Deploy/rollback observability and SLO routing | `OBS-001`, `OBS-002`, `OBS-004` | `RDR-T03`, `RDR-T07`, `RDR-T08` | `SCN-014`, `SCN-015`, `SCN-017`; `EVID-009`, `EVID-010`, `EVID-012` |
| Simplicity/no-drift constraints | `DES-001`, `DES-004`, `30` no-change, `40` no-change | `RDR-T01`, `RDR-T07`, `RDR-T08` | no API/data expansion; traceability vocabulary remains `SCN-*`/`EVID-*` |

## Execution Notes
1. Default execution unit: one task per run; checkpoint must be updated after each 2-task group.
2. If actual touched modules differ from planned change surface, record rationale in the corresponding Task Card before proceeding.
3. Do not mark any task `done` without command-backed evidence and reconciliation notes.
