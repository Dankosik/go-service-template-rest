# Runtime Diagnostics And Debug Endpoints

## Behavior Change Thesis
When loaded for health probes, shutdown, pprof, expvar, debug endpoints, runtime diagnostics, or telemetry flush symptoms, this file makes the model separate orchestration and incident-debug decisions with access, expiry, and privacy controls instead of likely mistake shared liveness/readiness checks, public debug handlers, or shutdown with no flush proof.

## When To Load
Load this when the spec needs `/livez`, `/readyz`, `/startupz`, graceful shutdown telemetry, pprof, expvar, admin/debug listener policy, incident-only debug controls, crash diagnostics, runtime metrics, or exporter flush behavior.

## Decision Rubric
- `/livez` answers "should this process be restarted?" Avoid downstream dependency checks unless the process cannot recover without restart.
- `/readyz` answers "should this instance receive traffic?" It may fail for warmup, dependency outage, overload, maintenance, paused workers, drain, or shutdown.
- `/startupz` answers "has initialization completed?" It protects slow startup from premature liveness restarts.
- Shutdown must expose readiness-fail time, drain begin/end, in-flight work, worker stop, exporter flush success/failure, and bounded exit reason.
- Put pprof, expvar, and expensive diagnostics on an isolated internal/admin listener or loopback-only surface with auth/network controls, audit, owner, and time-bound activation.
- Treat profiles, heap dumps, goroutine dumps, command lines, environment, and debug vars as potentially sensitive data.

## Imitate
- `/livez` avoids DB ping; `/readyz` includes dependency or overload state when traffic should stop; `/startupz` guards slow initialization.
  Copy the orchestration-decision split.
- Shutdown emits events or metrics for readiness fail, drain duration, in-flight requests, worker stop, exporter flush result, and final exit reason.
  Copy the proof that telemetry survived the exit path.
- pprof and expvar run on an internal-only/admin listener, disabled by default or guarded by time-bound incident activation with audit logging.
  Copy isolation plus expiry.

## Reject
- Liveness and readiness both run the same DB ping, causing dependency outages to restart healthy pods.
- Public `/debug/pprof` or `/debug/vars` on the main customer router.
- "Enable pprof during incidents" with no owner, access control, expiry, audit log, or privacy note.
- Graceful shutdown that stops accepting requests but drops traces/logs/metrics before exporter flush.
- Readiness always returns 200 while workers are paused, backlog is not draining, or shutdown is in progress.

## Agent Traps
- Treating health probes as monitoring dashboards. Probes drive orchestration decisions and should stay narrow.
- Exporting expvar maps keyed by tenant, account, request, message, or job IDs.
- Adding incident IDs or timestamps as metric labels for debug activation state.
- Forgetting that profiles and goroutine dumps can reveal request data, env values, command-line flags, and secrets.
- Making debug endpoints incident-only but never specifying how they expire or who can activate them.

## Validation Shape
- Verify liveness, readiness, and startup checks have distinct failure semantics and orchestration outcomes.
- Verify debug surfaces are absent from public/customer routers and protected by network/auth/audit/expiry controls.
- Verify shutdown proof includes readiness transition, drain, worker stop, telemetry exporter flush, and bounded exit.

## Canonical Verification Pointer
Use current Kubernetes probe docs and Go `net/http/pprof` or `expvar` docs when endpoint behavior or exposure defaults affect the spec.
