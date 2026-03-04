# 50 Security Observability DevOps

Status: updated

Changed sections linked to devops decisions: `DOPS-001`, `DOPS-002`, `DOPS-003`, `DOPS-004`, `DOPS-005`.
Changed sections linked to security decisions: `SEC-001`, `SEC-002`, `SEC-003`.
Changed sections linked to observability decisions: `OBS-001`, `OBS-002`, `OBS-003`, `OBS-004`.
Changed sections linked to architecture/design decisions: `ARCH-003`, `ARCH-004`, `DES-001`, `DES-002`, `DES-003`, `DES-004`.
Changed sections linked to runtime and verification decisions: `DOM-001`, `DOM-002`, `DOM-003`, `DOM-005`, `DOM-006`, `TST-002`, `TST-004`, `TST-005`.

## Phase Target
- Current phase: `Phase 2`
- Target gate: `G2`
- Focus: formal merge/release gates, private-by-default networking security policy with explicit ingress/egress exception path, and minimally sufficient observability signals without adding extra platform layers.

## Decision Register

### DOPS-001: Four-Tier Delivery Gate Model With Fail-Closed Merge Policy
- Phase/Gate: `Phase 2`, target `G2`
- Owner: Platform + DevOps
- Context/risk: merge/release decisions were spread across docs and workflows without one gate contract.
- Options:
1. Keep ad-hoc checks per workflow file and rely on reviewer judgment.
2. Define explicit `fast-path`/`full`/`nightly`/`release` tiers mapped to existing repository workflows and required contexts.
- Selected: **Option 2**.
- Rejected: Option 1 rejected due to non-deterministic merge eligibility and audit drift.
- Gate-level impact:
1. `fast-path`: local feedback only (`make check`), non-blocking for merge.
2. `full`: merge gate on PR to protected branch, blocking.
3. `nightly`: release-readiness signal, blocks release promotion when red.
4. `release`: tag/main publish preflight and trust evidence gates, blocking.
- Enforcement points:
1. `.github/workflows/ci.yml`
2. `.github/workflows/nightly.yml`
3. `.github/workflows/cd.yml`
4. `scripts/dev/configure-branch-protection.sh`
- Required compliance evidence:
1. Green required contexts for merge.
2. Nightly run state for release-readiness.
3. CD run artifacts for publish jobs.
- Exception policy:
1. Temporary bypass requires owner, expiry (max 14 days), compensating control, and reopen condition in `90`.

### DOPS-002: Railway Config-As-Code Rollout And Rollback Evidence Contract
- Phase/Gate: `Phase 2`, target `G2`
- Owner: Platform + Service Owner
- Context/risk: UI-only deployment policy edits and missing rollback evidence create unreviewable rollout risk.
- Options:
1. Keep Railway UI as operational source of truth and collect rollback evidence manually.
2. Make `railway.toml` the deployment-policy source of truth and require evidence bundle for rollout/rollback.
- Selected: **Option 2**.
- Rejected: Option 1 rejected due to drift risk and weak post-incident reconstruction.
- Gate-level impact:
1. `full`: production-policy changes are blocking without config-as-code evidence.
2. `release`: rollout/rollback evidence bundle is blocking for release-readiness claim.
- Enforcement points:
1. `repo-integrity` docs/config drift checks in CI.
2. `Wait for CI` + readiness-gated rollout in Railway.
3. Implementation obligations in `60` and evidence matrix in `70`.
- Required compliance evidence:
1. `railway.toml` diff and PR review trail.
2. `EVID-001..EVID-014` from `70`.
3. Rollback record with trigger, owner, previous revision/digest, and post-rollback health checks.
- Exception policy:
1. Emergency UI change allowed only with incident ticket, owner, expiry, and mandatory same-day reconciliation PR.

### DOPS-003: Release Trust Evidence As Hard Release Gate
- Phase/Gate: `Phase 2`, target `G2`
- Owner: Platform + Security
- Context/risk: image publish without strong provenance/signature evidence weakens release trust.
- Options:
1. Require vulnerability scan only.
2. Require vulnerability scan + SBOM + keyless signature + provenance attestation.
- Selected: **Option 2**.
- Rejected: Option 1 rejected due to insufficient supply-chain integrity proof.
- Gate-level impact:
1. `release`: blocking for `publish-main` and `publish-release`.
- Enforcement points:
1. Trivy scan steps in `cd.yml`.
2. SBOM generation/upload in `cd.yml`.
3. `cosign sign` step in `cd.yml`.
4. provenance attestation step in `cd.yml`.
- Required compliance evidence:
1. Trivy pass result (`CRITICAL,HIGH` gate).
2. SBOM artifact.
3. Signature record.
4. Provenance attestation reference.
- Exception policy:
1. Trust-gate bypass is prohibited for production releases.

### DOPS-004: Container And Runtime Hardening Baseline Stays Minimal And Strict
- Phase/Gate: `Phase 2`, target `G2`
- Owner: Platform + Security
- Context/risk: convenience-driven runtime image expansion increases attack surface.
- Options:
1. Use broader runtime images/shell tooling for operational convenience.
2. Keep multi-stage, distroless static, non-root, reproducible build defaults with explicit exception path.
- Selected: **Option 2**.
- Rejected: Option 1 rejected due to security noise and baseline drift.
- Gate-level impact:
1. `full`: container-security gate blocks merge when baseline is violated.
2. `release`: Trivy gate blocks publish.
- Enforcement points:
1. `build/docker/Dockerfile` policy review.
2. CI `container-security` job.
3. CD Trivy steps.
- Required compliance evidence:
1. Dockerfile baseline fields (`CGO_ENABLED=0`, distroless non-root, exec-form `ENTRYPOINT`, `STOPSIGNAL`).
2. Passing container scan.
- Exception policy:
1. cgo/dynamic runtime exceptions require explicit rationale and rollback plan in spec/sign-off.

### DOPS-005: Branch Protection Governance Uses Existing Repository Controls
- Phase/Gate: `Phase 2`, target `G2`
- Owner: Platform + Repo Admins
- Context/risk: introducing a separate policy engine would add unnecessary complexity for current scope.
- Options:
1. Add external policy-control layer for merge governance now.
2. Use existing `gh`-managed branch protection and stable required check contexts from repository scripts/workflows.
- Selected: **Option 2**.
- Rejected: Option 1 rejected as overengineering for current single-service scope.
- Gate-level impact:
1. `full`: blocking via required contexts and PR protection settings.
- Enforcement points:
1. `make gh-protect BRANCH=main`
2. `scripts/dev/configure-branch-protection.sh`
- Required compliance evidence:
1. Required context list in branch protection matches policy.
2. No direct-push/force-push bypass for protected branch.
- Exception policy:
1. Any temporary branch-rule relaxation must be time-bounded and recorded in `90`.

## Security Decision Register

### SEC-001: Private-By-Default Networking Baseline Is Mandatory
- Phase/Gate: `Phase 2`, target `G2`
- Owner: Security + Platform
- Context and trust boundary:
1. Service is an internal sidecar with no default requirement for internet-facing exposure.
2. Boundary classes in scope: `internal platform`, `approved partner/internal upstream egress`, `public internet (untrusted)`.
- Threat scenario and impact:
1. accidental public ingress enables unauthorized traffic surface expansion;
2. unrestricted public egress enables SSRF-assisted exfiltration and uncontrolled dependency spread.
- Options:
1. Keep mixed posture and decide exposure case-by-case in operations chat.
2. Enforce private-by-default ingress and controlled egress with fail-closed policy.
- Selected: **Option 2**.
- Rejected: Option 1 rejected due to policy ambiguity and weak auditability.
- Control mapping and enforcement points:
1. `contract`: no public endpoint contract is introduced in this pass.
2. `infra`: Railway networking remains private by default; public domain/ingress is disabled unless approved exception is active.
3. `service`: outbound clients must use explicit allowlist and reject unapproved public targets.
4. `observability`: networking policy violations must emit security audit event with correlation identifiers.
- Fail behavior and audit obligations:
1. Missing approval for public ingress or egress is `fail_closed`: block release-readiness and revert exposure.
2. Client-visible behavior for blocked policy path: deny operation (`403`/policy violation for ingress governance path, deterministic outbound-policy failure for egress callers).
3. Emit audit event with `environment`, `deployment_id` (if present), `policy_class`, `decision`.
- Verification obligations:
1. `70`: keep `SCN-010` for unapproved ingress fail path.
2. `70`: add egress-policy deny-path scenario and evidence packet.
- Cross-domain impact:
1. reliability: networking posture becomes safety gate input, not advisory note.
2. delivery: release-readiness evidence includes networking-policy state.
3. observability: policy decisions produce bounded-cardinality telemetry.
- Residual risk and reopen criteria:
1. `[assumption]` service remains internal-only by default in this phase.
2. Reopen if product requires persistent public ingress as baseline behavior.

### SEC-002: Public Ingress Requires Explicit, Time-Bounded Exception Workflow
- Phase/Gate: `Phase 2`, target `G2`
- Owner: Security + Platform + Service Owner
- Context and trust boundary:
1. public ingress crosses from `untrusted internet` to service boundary and increases abuse surface.
- Threat scenario and impact:
1. unreviewed ingress exposure can bypass intended internal-only trust model and enable abuse/DoS probing.
- Options:
1. Prohibit all public ingress with no exception path.
2. Allow only explicit exception workflow with strict preconditions, expiry, and rollback plan.
- Selected: **Option 2**.
- Rejected: Option 1 rejected because emergency integration/testing needs can exist and must stay governed.
- Control mapping and enforcement points:
1. `contract`: no new externally advertised API behavior in this pass.
2. `infra`: enable public ingress only after approved exception record with owner, scope, expiry, and rollback trigger.
3. `service`: exception requires active AuthN/AuthZ controls at edge path (`401/403` fail semantics stay mandatory).
4. `delivery`: merge/release evidence must include exception approval packet when ingress is enabled.
- Fail behavior and audit obligations:
1. If any exception field is missing (`owner`, `reason`, `scope`, `expiry`, `rollback plan`), exposure change is denied.
2. Exception expiry is fail-closed: on expiry, ingress must be disabled or exception renewed by explicit approval.
3. Every exception create/update/close emits security audit event.
- Verification obligations:
1. `70`: add approved-exception lifecycle scenario (create -> active -> expiry/close).
2. `70`: evidence must include approval record and post-change exposure snapshot.
- Cross-domain impact:
1. API/data: no contract/schema expansion required.
2. reliability: ingress exception cannot bypass rollout/readiness gates.
3. devops: exception evidence is part of release-readiness bundle.
- Residual risk and reopen criteria:
1. residual operational risk is bounded by time-limited approvals.
2. Reopen if repeated emergency exceptions indicate baseline model mismatch.

### SEC-003: Public Egress Policy Uses Allowlist-First Default With Exception Path
- Phase/Gate: `Phase 2`, target `G2`
- Owner: Security + Platform
- Context and trust boundary:
1. outbound calls from service to external networks are untrusted integration boundaries.
- Threat scenario and impact:
1. broad egress enables SSRF pivot, data exfiltration, and shadow dependency growth.
- Options:
1. Allow unrestricted outbound internet and rely on application checks only.
2. Restrict egress to approved destinations/schemes by default; require exception workflow for new public egress paths.
- Selected: **Option 2**.
- Rejected: Option 1 rejected due to weak least-privilege posture.
- Control mapping and enforcement points:
1. `service`: outbound HTTP clients enforce scheme/host allowlist, bounded redirect policy, and timeout budget.
2. `infra`: networking policy records approved egress classes and tracks exception lifecycle.
3. `observability`: emit `network_egress_policy_violation` for denied targets with bounded taxonomy (`policy_class`, `reason_class`).
4. `delivery`: release-readiness requires reconciled egress policy snapshot.
- Fail behavior and audit obligations:
1. Unapproved egress target is denied (`fail_closed`) and must not be retried with policy-bypass fallback.
2. Policy violations open security incident/ticket and block release-readiness until reconciled.
- Verification obligations:
1. `70`: add deny-path scenario for disallowed host/scheme and audit-evidence requirements.
2. `70`: include evidence packet for egress allowlist snapshot + violation trail.
- Cross-domain impact:
1. reliability: outbound dependency failures cannot force policy bypass.
2. observability: adds bounded security event taxonomy without broad telemetry expansion.
3. platform: no additional control plane; uses existing config-as-code + approval workflow.
- Residual risk and reopen criteria:
1. `[assumption]` static outbound IP is not baseline requirement in this phase.
2. Reopen if upstream dependency requires static egress identity as hard contract.

## Trust Boundaries And Threat Assumptions
1. Boundary map:
- `internal`: GitHub CI/CD, Railway private networking, service runtime.
- `public internet`: denied by default for ingress, constrained by allowlist for egress.
- `approved exception`: temporary, explicitly approved public ingress/egress scope.
2. Threat assumptions:
- accidental exposure/misconfiguration is more likely than targeted zero-day in this phase;
- policy drift and ad-hoc exceptions are primary control risks.
3. Active assumptions:
- `[assumption]` no persistent public ingress is required for normal sidecar operation.
- `[assumption]` required egress targets are finite and allowlist-manageable.

## Identity, AuthN/AuthZ, And Tenant Isolation Requirements
1. Identity baseline for networking exceptions:
- no trust is granted from source IP or unsigned forwarding headers;
- exposed ingress path (if exception active) must enforce existing AuthN/AuthZ boundary semantics and explicit `401/403` behavior.
2. Caller/subject model:
- service keeps internal service identity model; no public anonymous access mode is introduced.
3. Tenant-isolation implication:
- exception paths must not bypass tenant scoping or object-level authorization checks in service layer.

## Threat-Class Control Matrix
| Threat Class | Boundary | Control | Enforcement Points | Fail Behavior | Linked Decision |
|---|---|---|---|---|---|
| Unauthorized public ingress | public -> service | private-by-default + approved exception only | `infra`, `delivery` | deny exposure change, revert if drift detected | `SEC-001`, `SEC-002` |
| SSRF / unapproved outbound | service -> public | egress allowlist, scheme restrictions, redirect checks | `service`, `infra` | deny request and emit violation event | `SEC-001`, `SEC-003` |
| Policy drift / shadow changes | config runtime drift | config-as-code reconciliation + evidence gate | `infra`, `delivery`, `observability` | block release-readiness | `SEC-001`, `SEC-002`, `SEC-003` |
| Sensitive data leakage in diagnostics | all boundaries | redacted security events only, no token/payload logging | `service`, `observability` | sanitize output, incident if leak detected | `SEC-001`, `SEC-003` |

## Secrets, Sensitive Data, And Redaction Rules
1. No tokens/secrets/raw payload fragments in ingress/egress policy events.
2. Security events may include bounded identifiers only (`environment`, `deployment_id`, `policy_class`, `reason_class`).
3. Exception records store approval metadata, not runtime secret values.

## Abuse-Resistance And Fail-Closed Policies
1. Networking policy controls are deny-by-default and `fail_closed` on missing approval.
2. Public ingress exception must include expiry and rollback trigger; expired exception is invalid by default.
3. Egress policy violations must not trigger auto-retry to alternate unapproved targets.
4. No bypass path may be introduced through debug/admin endpoints in this pass.

## Security Verification And Negative Test Obligations
1. Mandatory negative paths in `70`:
- unapproved public ingress enablement is denied and reversible (`SCN-010`);
- disallowed outbound target is denied with audit evidence (`SCN-019`).
2. Mandatory lifecycle path in `70`:
- approved temporary ingress exception has explicit open/close evidence and expiry enforcement (`SCN-018`).
3. Evidence obligations:
- networking exception packet (`EVID-013`);
- egress policy deny packet (`EVID-014`).

## Residual Risk, Compensating Controls, And Reopen Criteria
1. Residual risk: manual exception approvals can still create operational burden.
2. Compensating controls: time-bounded approval metadata, mandatory evidence packet, drift detection, and release-readiness blocking.
3. Reopen criteria:
- repeated exception usage that effectively becomes permanent behavior;
- new external integration requiring broader/public egress baseline;
- requirement to expose public ingress as default service mode.

## Observability Decision Register

### OBS-001: Minimal Deploy-Health Signal Contract
- Phase/Gate: `Phase 2`, target `G2`
- Owner: Platform + Service Owner
- Operational question: can rollout promotion health be evaluated and audited without manual log forensics?
- Options:
1. Rely only on raw Railway/GitHub logs.
2. Define a bounded deploy-health signal contract with structured events, low-cardinality metrics, and trace correlation IDs.
- Selected: **Option 2**.
- Rejected: Option 1 rejected due to slow diagnosis and weak machine-checkability.
- Signal contract delta:
1. structured event `deploy_health_check` with `environment`, `rollout_id`, `deployment_id`, `ci_run_id`, `result`, `reason_class`, `duration_ms`;
2. metrics:
- `deploy_health_admission_total{environment,result,reason_class}`
- `deploy_health_admission_duration_seconds{environment,result}`
- `deploy_health_probe_failures_total{environment,probe_type}`;
3. trace span `deploy.health.admission` (low-cardinality name).
- Cardinality/cost controls:
1. `rollout_id`, `deployment_id`, `ci_run_id`, `commit_sha` are logs/traces only, never metric labels.
- Cross-domain impact:
1. reliability rollout gates (`DOM-002`, `DOM-003`) consume explicit admission signals.
- Verification obligations:
1. add signal presence/shape checks in `70` (`EVID-009`).
- Reopen conditions:
1. repeated false-positive admission failures with no signal-level root cause.

### OBS-002: Rollback Signal And Correlation Continuity Contract
- Phase/Gate: `Phase 2`, target `G2`
- Owner: Platform + Reliability
- Operational question: can failed rollout and rollback lifecycle be correlated end-to-end within minutes?
- Options:
1. Keep rollback proof only as free-form incident note.
2. Require explicit rollback event/metric/span contract linked to failed rollout correlation IDs.
- Selected: **Option 2**.
- Rejected: Option 1 rejected due to unverifiable rollback effectiveness.
- Signal contract delta:
1. structured event `rollback_execution` with `rollback_id`, `rollout_id`, `trigger`, `result`, `previous_revision`, `duration_ms`;
2. metrics:
- `rollback_execution_total{environment,trigger,result}`
- `rollback_recovery_duration_seconds{environment,result}`
- `rollback_postcheck_total{environment,endpoint,result}`;
3. trace span `deploy.rollback.execute` linked to failed admission span.
- Cardinality/cost controls:
1. `rollback_id` and revisions are logs/traces only.
- Cross-domain impact:
1. aligns with rollback evidence contract in `DOPS-002`.
- Verification obligations:
1. add correlation-chain verification in `70` (`EVID-010`).
- Reopen conditions:
1. rollback can complete without correlated telemetry chain.

### OBS-003: Config-Drift Detection And Reconciliation Signal Contract
- Phase/Gate: `Phase 2`, target `G2`
- Owner: DevOps + Platform
- Operational question: is policy drift between Railway active settings and `railway.toml` detectable and closure-tracked?
- Options:
1. periodic manual drift audit only.
2. explicit drift detection and reconciliation telemetry tied to release-readiness.
- Selected: **Option 2**.
- Rejected: Option 1 rejected due to silent compliance drift risk.
- Signal contract delta:
1. structured events `config_drift_detected` and `config_drift_reconciled`;
2. metrics:
- `config_drift_detected_total{environment,source}`
- `config_drift_open{environment}`
- `config_drift_reconcile_duration_seconds{environment,result}`;
3. trace span `deploy.config_drift.check` for check/reconcile runs.
- Cardinality/cost controls:
1. `source` bounded to `ci` or `runtime`.
- Cross-domain impact:
1. directly supports `DOM-005` and `DOPS-002`.
- Verification obligations:
1. add drift-open/drift-close evidence checks in `70` (`EVID-011`).
- Reopen conditions:
1. drift incidents are detected but not observable to closure.

### OBS-004: Minimal SLI/SLO And Alert Routing For Deploy/Rollback/Drift
- Phase/Gate: `Phase 2`, target `G2`
- Owner: Observability + Platform On-call
- Operational question: what minimum SLO/alert policy is enough for release gating without over-instrumentation?
- Options:
1. full broad service-wide observability stack in this pass.
2. focused operational SLI/SLO set for deploy health, rollback, and config drift only.
- Selected: **Option 2**.
- Rejected: Option 1 rejected due to scope expansion and unnecessary telemetry cost.
- Policy delta:
1. 28-day rolling window for deployment-governance objectives;
2. budget states `green/yellow/orange/red` with default thresholds from operability baseline;
3. burn-rate paging for deploy-health plus immediate rollback-failure page and drift ticketing.
- Cross-domain impact:
1. release gating in `DOPS-001..003` now consumes explicit observability policy.
- Verification obligations:
1. add SLI formula and alert-route evidence checks in `70` (`EVID-012`).
- Reopen conditions:
1. approved objective threshold change for deploy-health SLO materially changes alert/gate behavior.

## CI Gate Matrix And Blocking Policy
| Tier | Trigger | Blocking Semantics | Gate Set | Enforcement |
|---|---|---|---|---|
| `fast-path` | local developer loop | non-blocking for merge | `make check` | local workflow |
| `full` | PR to protected branch | hard stop for merge | required contexts: `repo-integrity`, `lint`, `openapi-contract`, `test`, `test-race`, `test-coverage`, `test-integration`, `migration-validate`, `go-security`, `secret-scan`, `container-security` | `ci.yml` + branch protection |
| `nightly` | schedule/manual nightly | hard stop for release promotion when red | `nightly.yml` `reliability` job (flake/fuzz/race/integration/openapi/security/container) | nightly workflow |
| `release` | `main` publish and tag release | hard stop for publish | `release-preflight` and publish trust steps (`Trivy`, SBOM, signature, provenance) | `cd.yml` |

## Merge And Release Hard-Stop Criteria
1. Merge is blocked when any required context is `failed`, `timed_out`, or `cancelled`.
2. Merge is blocked when docs/config drift checks fail for behavior/CI/deploy-policy changes.
3. Merge is blocked for Railway policy changes not represented in `railway.toml` + PR review flow (`DOM-005`).
4. Release publish is blocked when `release-preflight` fails.
5. Release publish is blocked when Trivy fails at `HIGH/CRITICAL`.
6. Release publish is blocked when SBOM, signature, or provenance evidence is missing.
7. Release-readiness claim is blocked when rollout/rollback evidence bundle is incomplete.

## Drift, Compatibility, And Migration Validation Policy
1. Docs drift is mandatory for behavior/contract/CI-sensitive paths.
2. OpenAPI generate/validate/lint/drift checks are mandatory in `full` and `release-preflight`.
3. `openapi-breaking` runs on PR for compatibility evidence; required merge contexts remain branch-protection stable in this phase (`openapi-contract` required, `openapi-breaking` not globally required).
4. Migration validation is conditional and mandatory when `env/migrations/**` changes.
5. Deployment-policy drift is fail-closed: UI-only Railway setting changes are non-compliant until reconciled into `railway.toml`.

## Containerization And Runtime Hardening Baseline
1. Build remains canonical via `build/docker/Dockerfile` (`ARCH-003`), no extra build/control layers.
2. Runtime baseline: distroless static, non-root user, exec-form `ENTRYPOINT`, `STOPSIGNAL SIGTERM`.
3. Build reproducibility defaults remain mandatory: pinned base images by digest, `-trimpath`, `-mod=readonly`, `-buildvcs=false`.
4. Secrets are not embedded in images or repository configs.
5. Runtime-hardening exceptions require explicit owner, expiry, and rollback plan.

## Release Trust Evidence Requirements
| Evidence | Source | Blocking Tier |
|---|---|---|
| Trivy image scan pass (`HIGH/CRITICAL`) | `ci.yml` `container-security`, `cd.yml` scan steps | `full`, `release` |
| SBOM artifact (`CycloneDX`) | `cd.yml` upload artifact step | `release` |
| Keyless image signature | `cd.yml` cosign step | `release` |
| Provenance attestation | `cd.yml` attest-build-provenance step | `release` |
| Rollout/rollback evidence bundle (`EVID-001..EVID-014`) | `70` + Railway deploy logs + rollback record | `release-readiness` |

## Telemetry Signal Contract
| Component | Structured Logs | Metrics | Traces | Correlation Keys | Owner |
|---|---|---|---|---|---|
| Deploy health admission | `deploy_health_check` (`result`, `reason_class`, `duration_ms`) | `deploy_health_admission_total`, `deploy_health_admission_duration_seconds`, `deploy_health_probe_failures_total` | `deploy.health.admission` | `rollout_id`, `deployment_id`, `ci_run_id`, `commit_sha` | Platform |
| Rollback execution | `rollback_execution` (`trigger`, `result`, `duration_ms`) | `rollback_execution_total`, `rollback_recovery_duration_seconds`, `rollback_postcheck_total` | `deploy.rollback.execute` | `rollback_id`, `rollout_id`, `deployment_id` | Platform + Reliability |
| Config drift lifecycle | `config_drift_detected`, `config_drift_reconciled` | `config_drift_detected_total`, `config_drift_open`, `config_drift_reconcile_duration_seconds` | `deploy.config_drift.check` | `drift_id`, `config_revision`, `ci_run_id` | DevOps |
| Networking policy security events | `network_ingress_policy_violation`, `network_egress_policy_violation`, `network_exception_state_change` | `network_policy_violation_total`, `network_exception_active` | `security.network.policy` | `deployment_id`, `policy_class`, `exception_id` | Security + Platform |
| Runtime shutdown/drain signal | `shutdown_started`, `readiness_disabled`, `drain_completed`, `shutdown_timeout` | `shutdown_drain_duration_seconds`, `shutdown_timeout_total` | `runtime.shutdown.sequence` | `deployment_id`, `rollout_id` | Service Owner |

## SLI/SLO, Error Budget, And Burn-Rate Policy
1. SLI `deploy_health_admission_ratio`:
- `good_events`: rollout attempts with candidate readiness success and no rollback-required transition.
- `total_events`: rollout attempts that passed CI gate and entered candidate startup.
- exclusions: manually cancelled runs before candidate startup.
- target: `99.5% / 28d`.
2. SLI `rollback_recovery_ratio`:
- `good_events`: rollback attempts restoring stable ready state within `5m`.
- `total_events`: rollback attempts triggered by rollout failure.
- target: `99.0% / 28d`.
3. SLI `config_drift_reconcile_ratio`:
- `good_events`: detected drift reconciled to `railway.toml` within `24h`.
- `total_events`: detected drift events.
- target: `100% / 28d`.
4. Budget states:
- `green <= 25%`, `yellow <= 50%`, `orange <= 100%`, `red > 100%` budget consumed.
5. Burn-rate and low-traffic floors:
- deploy-health page: burn-rate `>= 6` on `30m/6h` with floors `short >= 2 events`, `long >= 5 events`.
- rollback-failure page: immediate when `rollback_execution_total{result="failure"}` increments in production.
- config-drift ticket: open drift `> 4h`; escalate to page if `> 24h`.

## Alert Routing, Dashboards, And Runbooks
1. Paging alerts:
- `ALERT-DEPLOY-HEALTH-BURN`: Platform on-call.
- `ALERT-ROLLBACK-FAILED`: Platform + Service Owner on-call.
2. Ticket alerts:
- `ALERT-CONFIG-DRIFT-OPEN`: DevOps backlog owner.
3. Minimum dashboard links (operational consumers):
- CI/CD run timeline (`ci.yml`, `cd.yml`);
- Railway deployment timeline and healthcheck events;
- release-readiness evidence board (`EVID-001..EVID-014`).
4. Runbook linkage:
- rollback and post-check procedure: `60` `WP-5` and `WP-6`;
- testable evidence procedure: `70` evidence sections.
5. Routing contract:
- every page/ticket must include `environment`, `rollout_id` (if present), `deployment_id` (if present), and first-action hint.

## Diagnostics And Debuggability Contract
1. Probe semantics (current runtime contract):
- `GET /health/live`: restart decision only.
- `GET /health/ready`: traffic admission only.
2. Startup signal policy:
- no new `/startupz` endpoint in this scope (to avoid API expansion); startup completion is observed as first successful readiness transition in deployment telemetry (`OBS-001`).
3. Shutdown signal policy:
- must emit ordered events: readiness disabled, drain complete or timeout, process exit status.
4. Debug endpoint exposure:
- no public debug endpoint exposure in this pass; keep existing private/internal posture.

## Telemetry Cost, Cardinality, And Retention Controls
1. Metric-label allowlist for this scope:
- `environment`, `result`, `reason_class`, `trigger`, `probe_type`, `source`, `endpoint`, `policy_class`.
2. Never as metric labels:
- `request_id`, `trace_id`, `span_id`, `rollout_id`, `deployment_id`, `rollback_id`, `drift_id`, `exception_id`, `commit_sha`.
3. Trace/log sampling:
- deployment-governance spans are always sampled (low-volume control path);
- no broad increase for request-path sampling.
4. Retention baseline for deployment-governance telemetry:
- metrics/logs/traces retained for at least `28d` to match budget window.
5. Redaction:
- no secrets/tokens/raw payloads in deploy/rollback/drift logs.

## Async Correlation, Retry/DLQ, And Reconciliation Observability
1. Status: no queue/DLQ async pipeline in current feature scope.
2. Minimal async-like requirement in scope:
- config-drift check/reconcile is treated as scheduled control flow and must keep stable `drift_id` across detect/reconcile events.
3. Reopen condition:
- if queue-based rollout/rollback orchestration is introduced, add full async retry/DLQ contract before `G2`.

## Exception And Risk-Acceptance Policy
1. Any bypass requires owner, reason, compensating controls, and expiry date.
2. Blocking gates cannot be silently downgraded to informational.
3. All active bypasses and reopen conditions must be recorded in `90`.
4. Critical unresolved devops/security/observability unknowns must be tracked in `80`.
5. Default posture is fail-closed for merge/release eligibility.

## Conditional Artifact Alignment
| Artifact | Status | Linked Decisions | Note |
|---|---|---|---|
| `55-reliability-and-resilience.md` | updated | `SEC-001`, `SEC-002`, `SEC-003`, `OBS-001`, `OBS-002`, `OBS-003` | reliability model now treats networking posture as fail-closed and keeps deploy/rollback/drift hooks |
| `60-implementation-plan.md` | updated | `DOPS-001`, `DOPS-002`, `DOPS-003`, `DOPS-005`, `SEC-001`, `SEC-002`, `SEC-003`, `OBS-001`, `OBS-002`, `OBS-003` | adds networking exception workflow obligations plus delivery and observability implementation contracts |
| `70-test-plan.md` | updated | `SEC-001`, `SEC-002`, `SEC-003`, `OBS-001`, `OBS-002`, `OBS-003`, `OBS-004` | adds security networking negative-path/lifecycle evidence obligations |
| `20-architecture.md` | no changes required | `SEC-001`, `OBS-004` | existing internal-sidecar topology already enforces private-by-default boundary without new control planes |
| `30-api-contract.md` | no changes required | `SEC-002`, `OBS-003` | no new public API surface is introduced; networking security policy is enforced out-of-band |
| `40-data-consistency-cache.md` | no changes required | `SEC-003`, `OBS-003` | networking policy pass does not alter datastore/cache contracts |
