# 60 Implementation Plan

Status: updated

Changed sections linked to devops decisions: `DOPS-001`, `DOPS-002`, `DOPS-003`, `DOPS-005`.
Changed sections linked to security decisions: `SEC-001`, `SEC-002`, `SEC-003`.
Changed sections linked to observability decisions: `OBS-001`, `OBS-002`, `OBS-003`, `OBS-004`.
Changed sections linked to domain decisions: `DOM-001`, `DOM-002`, `DOM-003`, `DOM-004`, `DOM-005`, `DOM-006`.
Changed sections linked to design decisions: `DES-001`, `DES-002`, `DES-003`, `DES-004`.

## WP-1 Safe Rollout Baseline (`DOM-001`, `DOM-002`, `DOM-003`)
1. Keep CI gate enabled for deploy trigger.
2. Configure healthcheck to `/health/ready`.
3. Enable teardown overlap/drain behavior.
4. Set candidate promotion timeout to `180s`.
5. Keep restart policy `On Failure` with max retries `5`.
6. Verify in logs that old revision terminates only after candidate promotion.

## WP-2 Availability Baseline (`DOM-004`)
1. Set production minimum to `2` replicas in one region.
2. Set per-replica baseline caps to `CPU 2 vCPU` and `Memory 2 GiB`.
3. Any cap increase requires explicit spec reopen with updated evidence packet.

## WP-3 Config Governance (`DOM-005`)
1. Add `railway.toml` as deployment policy source of truth.
2. Bind config-as-code path in Railway.
3. Ensure non-secret deploy settings are changed via PR flow only.

## WP-4 Networking Policy (`DOM-006`)
1. Keep private networking as default integration path.
2. Keep public ingress disabled unless approved exception is active (`SEC-002`).
3. Exception workflow requirements for public ingress or egress:
- approved owner + reason + scope + expiry + rollback plan;
- evidence packet attached to release-readiness record.
4. Keep egress allowlist-first baseline and reject unapproved public outbound targets (`SEC-003`).
5. Keep one exception-template format for both ingress and egress to avoid parallel policy branches.

## WP-5 Verification And Rollback (`DOM-002`, `DOM-003`)
1. Mandatory post-deploy checks:
- `GET /health/live`
- `GET /health/ready`
2. Mandatory numeric policy checks:
- promotion timeout observed `<= 180s` from candidate start to ready;
- drain completion observed `<= 45s` before old-revision termination;
- graceful shutdown timeout configured to `30s`;
- restart retries do not exceed `5` before rollback-required handling.
3. Rollback trigger:
- failed health-gated rollout, failed drain sequence, or unstable candidate.
4. Rollback action:
- redeploy previous known-good revision and re-run smoke checks.
5. Evidence capture obligations:
- collect `EVID-001..EVID-014` defined in `70` for each production rollout.
6. Numeric threshold assertions in `SCN-011..SCN-013` are mandatory and release-relevant.

## WP-6 Delivery Gates And Release Evidence (`DOPS-001`, `DOPS-002`, `DOPS-003`, `DOPS-005`)
1. Keep branch-protection required contexts aligned with active CI job names; treat name drift as merge blocker.
2. Keep `full` merge gate fail-closed on required contexts (`repo-integrity`, `lint`, `openapi-contract`, `test`, `test-race`, `test-coverage`, `test-integration`, `migration-validate`, `go-security`, `secret-scan`, `container-security`).
3. Keep release gate flow explicit:
- `release-preflight` must pass before tag publish;
- `publish-main`/`publish-release` must include passing Trivy, SBOM generation, keyless signature, and provenance attestation.
4. For Railway rollout/readiness claims, keep evidence bundle mandatory:
- `railway.toml` PR diff and review trace;
- `EVID-001..EVID-014` from `70`;
- rollback record with trigger, owner, previous digest/revision, and post-rollback `/health/live` + `/health/ready` checks.
5. Keep implementation scope minimal:
- no new external policy engine, no additional rollout control plane, no Terraform-first expansion in this pass.

## WP-7 Minimal Observability Signals For Deploy/Rollback/Drift (`OBS-001`, `OBS-002`, `OBS-003`, `OBS-004`)
1. Emit bounded deploy-health telemetry:
- log `deploy_health_check`;
- metrics `deploy_health_admission_total`, `deploy_health_admission_duration_seconds`, `deploy_health_probe_failures_total`;
- trace span `deploy.health.admission`.
2. Emit bounded rollback telemetry:
- log `rollback_execution`;
- metrics `rollback_execution_total`, `rollback_recovery_duration_seconds`, `rollback_postcheck_total`;
- trace span `deploy.rollback.execute` linked to failed rollout.
3. Emit bounded config-drift telemetry:
- logs `config_drift_detected` and `config_drift_reconciled`;
- metrics `config_drift_detected_total`, `config_drift_open`, `config_drift_reconcile_duration_seconds`;
- trace span `deploy.config_drift.check`.
4. Enforce correlation and cardinality rules:
- keep `rollout_id`, `deployment_id`, `rollback_id`, `drift_id`, `ci_run_id`, `commit_sha` in logs/traces only;
- do not use these fields as metric labels.
5. Attach alert-routing and budget-state evidence from `OBS-004` into rollout readiness packet (`EVID-012`).

## Complexity Guardrail
1. Do not parallelize high-risk rollout-safety changes with capacity-baseline changes or future networking-model changes unless explicitly reopened via `DES-003`.
