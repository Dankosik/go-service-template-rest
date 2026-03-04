# 55 Reliability And Resilience

Status: updated

Changed sections linked to domain decisions: `DOM-001`, `DOM-002`, `DOM-003`, `DOM-004`, `DOM-005`, `DOM-006`.
Changed sections linked to design decisions: `DES-001`, `DES-003`.
Changed sections linked to security decisions: `SEC-001`, `SEC-002`, `SEC-003`.
Changed sections linked to observability decisions: `OBS-001`, `OBS-002`, `OBS-003`, `OBS-004`.

## Baseline Reliability Model
| Component | Criticality | Failure Mode | Policy | Linked Decision |
|---|---|---|---|---|
| CI gate before deploy | `critical_fail_closed` | CI failed/unknown | do not deploy | `DOM-001` |
| Readiness healthcheck during deploy | `critical_fail_closed` | candidate not ready | block promotion, fail rollout | `DOM-002` |
| Replacement drain sequence | `critical_fail_degraded` | old revision removed too early | require overlap + drain before termination | `DOM-003` |
| Replica baseline | `critical_fail_degraded` | capacity collapse on single failure | keep minimum `2` replicas | `DOM-004` |
| Config drift control | `critical_fail_degraded` | hidden UI drift | block release-readiness until reconciled | `DOM-005` |
| Public exposure policy | `critical_fail_closed` | unapproved public ingress | block exposure change, trigger incident, and revert exposure | `DOM-006` |
| Public egress policy | `critical_fail_closed` | unapproved outbound destination | deny outbound call and raise policy-violation incident | `SEC-003` |

## Reliability Requirements
1. Railway `Healthcheck Path` must be `/health/ready` for current runtime (`DOM-002`).
2. Candidate promotion timeout must be `180s`; timeout requires rollback path activation (`DOM-002`).
3. Teardown overlap/drain must be enabled with drain window `45s` and app graceful shutdown timeout `30s` (`DOM-003`).
4. Restart policy remains `On Failure` with max retries `5` before rollback-required handling.
5. Production baseline must run with at least `2` replicas and per-replica caps `CPU 2 vCPU` / `Memory 2 GiB` (`DOM-004`).
6. Deploy policy must be managed via config-as-code and reviewed in PR (`DOM-005`).
7. Private networking remains default; public ingress only via approved and time-bounded exception (`DOM-006`, `SEC-002`).
8. Public egress is allowlist-first by default; new outbound targets require explicit exception workflow (`SEC-003`).

## Reliability Risks
1. **RISK-REL-001**: fixed rollout timeout (`180s`) may be tight under cold-start spikes; watch `deploy_health_admission_duration_seconds` tail.
2. **RISK-REL-002**: fixed per-replica caps (`2 vCPU` / `2 GiB`) may need tuning under sustained growth; tune only via controlled spec reopen.
3. **RISK-REL-003**: fixed drain window (`45s`) may be insufficient for rare long-running requests; monitor `shutdown_timeout_total`.

## Reliability-Observability Gate Hooks
1. Deploy admission gate (`DOM-002`) must emit:
- `deploy_health_admission_total` and `deploy_health_admission_duration_seconds`;
- structured event `deploy_health_check` with bounded `reason_class`.
2. Rollback gate (`DOM-003`) must emit:
- `rollback_execution_total`, `rollback_recovery_duration_seconds`;
- structured event `rollback_execution` correlated to failed rollout.
3. Config drift release-readiness gate (`DOM-005`) must emit:
- `config_drift_detected_total`, `config_drift_open`, `config_drift_reconcile_duration_seconds`;
- structured events `config_drift_detected` and `config_drift_reconciled`.
4. Rollout/release decisions must consume observability policy from `50`:
- SLI formulas and alert routing from `OBS-004`;
- evidence bundle continuity in `70`.
5. Networking policy controls (`SEC-001..SEC-003`) must emit security events:
- `network_ingress_policy_violation`, `network_egress_policy_violation`, `network_exception_state_change`;
- release-readiness remains blocked until policy violations are reconciled.
