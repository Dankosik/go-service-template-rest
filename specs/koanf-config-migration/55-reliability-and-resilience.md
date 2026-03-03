# 55 Reliability And Resilience

## Startup Reliability Model
| Component | Criticality | Budget | Failure Mode | Policy |
|---|---|---|---|---|
| config defaults/file/env load | fail-closed | startup load budget | parse/load error | abort startup |
| config parse/validate | fail-closed | validation budget | semantic/strict error | abort startup |
| postgres probe (if enabled) | fail-closed | probe budget | dependency init failure | abort startup |
| redis probe (cache mode) | fail-open (`feature_off`) | probe budget | dependency init failure | continue degraded |
| redis probe (store mode) | fail-closed | probe budget | dependency init failure | abort startup |
| mongo probe (if enabled) | degraded | probe budget | dependency init failure | continue degraded |
| telemetry setup | fail-open | telemetry budget | setup failure | continue with local observability |

## Reliability Requirements
1. Context cancellation is honored in config load and validation.
2. No retries for config parse/validation failures.
3. Startup errors are typed and mapped to deterministic reasons.
4. Shutdown path emits explicit milestones and flushes telemetry best-effort.
