# Startup, Readiness, Liveness, And Shutdown Contracts

## Behavior Change Thesis
When loaded for service lifecycle contracts, this file makes the model choose separate startup, liveness, readiness, diagnostics, and drain semantics instead of likely mistake "health check the database" or "wait for requests to finish" without bounds.

## When To Load
Load when the spec needs startup behavior, readiness/liveness semantics, health endpoints, traffic admission, restart policy, draining, graceful shutdown, telemetry flush, or long-lived connection behavior.

## Decision Rubric
- Startup gates slow mandatory initialization before liveness/readiness start making decisions; skip it when process launch is the only initialization.
- Liveness is a restart signal for local process inability to make progress. External dependency outages should not trigger restarts by default.
- Readiness is a traffic-admission signal. Fail it when the instance cannot serve core traffic or is draining; keep it true for optional dependency degradation when core traffic remains safe.
- Diagnostic health can report component detail, but it must not become the only traffic-admission or restart signal without clear semantics and cost/sensitivity boundaries.
- Shutdown/drain fails readiness, stops accepting new work, drains in-flight work within `<drain window>`, flushes or records required signals, and exits before hard kill.
- Long-lived, hijacked, streaming, and background operations need explicit close, signal, drain, or exclusion policy.

## Imitate
- "`/livez` checks only local process progress and does not fail because the database is down; `/readyz` fails when database access is required for core traffic."
- "During shutdown, readiness fails before the listener stops accepting new work; in-flight requests drain for `<drain window>`; long-lived streams follow a separate close policy."
- "The diagnostic health endpoint reports optional dependency degradation, but traffic admission remains tied to whether the core flow can serve."

## Reject
- "Liveness checks the database." A dependency outage can cause restart loops without fixing the dependency.
- "Readiness remains true while draining." Traffic may continue arriving after the instance has begun shutdown.
- "Shutdown relies only on process signal handling." This lacks a caller-visible drain rule, deadline, and long-lived connection policy.
- "Health endpoint performs expensive deep checks on every probe." The probe itself can overload the service.

## Agent Traps
- Do not collapse liveness and readiness into one "healthy" boolean unless the platform genuinely has no separate signals.
- Do not make optional dependency degradation remove all traffic if the core flow can safely serve.
- Do not assume Go `Server.Shutdown` handles hijacked or long-lived connections the way ordinary requests drain; require an explicit policy.

## Validation Shape
- Given optional dependency failure, diagnostics show degradation but readiness remains true if core traffic is still safe.
- Given core dependency failure, readiness becomes false after the specified threshold or hysteresis and returns true only after recovery criteria.
- Given liveness failure, the check points to local process inability to make progress rather than remote dependency state.
- Given shutdown, readiness changes before new traffic is admitted, in-flight requests are drained or timed out, and the process exits within the platform grace period.
- Given hijacked, streaming, or long-lived connections, the spec states whether they are closed, signaled, drained, or excluded from graceful shutdown.
