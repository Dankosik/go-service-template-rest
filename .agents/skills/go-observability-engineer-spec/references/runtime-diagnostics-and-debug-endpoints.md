# Runtime Diagnostics And Debug Endpoints

## When To Load This
Load this reference when the spec needs health probes, graceful shutdown telemetry, pprof, expvar, runtime diagnostics, admin/debug listener policy, incident-only debug controls, crash diagnostics, or telemetry flush behavior.

## Operational Questions
- Which endpoint tells the orchestrator to restart the process, and which endpoint removes it from traffic?
- How does startup readiness differ from steady-state dependency readiness?
- During shutdown, how does the operator see readiness fail, drain begin, in-flight work finish, telemetry flush, and bounded exit?
- Which runtime diagnostics are available during an incident, who can access them, and how do they expire?
- What sensitive data can pprof, expvar, traces, heap profiles, goroutine dumps, command lines, or debug variables reveal?

## Good Telemetry Examples
- `/livez` answers "should the process be restarted?" and avoids fragile downstream dependency checks unless the process cannot recover without restart.
- `/readyz` answers "should this instance receive traffic?" and can fail during dependency outage, overload, warmup, drain, or maintenance.
- `/startupz` answers "has initialization completed?" and protects slow startup from premature liveness restarts.
- Shutdown emits bounded events or metrics for readiness-fail time, drain duration, in-flight requests, worker stop, exporter flush success/failure, and final exit reason.
- pprof and expvar are on an internal-only or loopback/admin listener with auth/network controls, disabled by default or guarded by a time-bound incident switch.

## Bad Telemetry Examples
- Liveness and readiness both run the same DB ping, causing dependency outages to restart otherwise healthy pods.
- Public `/debug/pprof` or `/debug/vars` mounted on the main customer router.
- "Enable pprof during incidents" with no owner, access control, expiry, audit log, or privacy note.
- Graceful shutdown that stops accepting requests but drops traces/logs/metrics before exporter flush.
- Readiness always returns 200 even while workers are paused, backlog is not draining, or the instance is in shutdown.

## Cardinality Traps
- Exporting goroutine names, request IDs, job IDs, queue item IDs, or per-entity debug variables as labels.
- Using expvar for arbitrary maps keyed by tenant, account, request, message, or job IDs.
- Encoding incident IDs or timestamps as metric labels for debug activation state.
- Adding one metric per debug endpoint or profile name when a bounded `profile` label is enough.

## Selected And Rejected Options
- Select distinct `/livez`, `/readyz`, and `/startupz` semantics because they drive different orchestration decisions.
- Select an isolated admin/debug listener over mounting debug handlers on the public API router.
- Select time-bound incident activation with audit logging for expensive or sensitive diagnostics.
- Select runtime metrics such as goroutine count, heap, CPU, file descriptors, queue wait, and telemetry exporter failures when they support capacity or incident decisions.
- Reject dependency-heavy liveness checks unless a failed dependency makes the process unrecoverable.
- Reject public debug endpoints and always-on profiling when no access, retention, and privacy policy exists.

## Exa Source Links
- Kubernetes Liveness, Readiness, and Startup Probes: https://kubernetes.io/docs/concepts/configuration/liveness-readiness-startup-probes/
- Kubernetes Configure Liveness, Readiness and Startup Probes: https://kubernetes.io/docs/tasks/configure-pod-container/configure-liveness-readiness-startup-probes/
- Go `net/http/pprof`: https://pkg.go.dev/net/http/pprof
- Go `expvar`: https://pkg.go.dev/expvar
- Google SRE Workbook, Monitoring: https://sre.google/workbook/monitoring/
