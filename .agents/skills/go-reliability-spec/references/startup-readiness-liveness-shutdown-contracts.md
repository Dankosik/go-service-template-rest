# Startup, Readiness, Liveness, And Shutdown Contracts

## When To Load This
Load this file when the spec needs startup behavior, readiness/liveness semantics, health endpoints, traffic admission, restart policy, draining, graceful shutdown, telemetry flush, or long-lived connection behavior.

## Contract Questions
- Which checks decide restart, which decide traffic admission, and which only report diagnostics?
- Which dependencies are required for startup or readiness, and which are allowed to be degraded?
- How does the service stop new work before draining old work?
- Which long-lived, hijacked, streaming, or background operations need separate shutdown policy?

## Option Comparisons
| Option | Use when | Contract shape | Reject when |
| --- | --- | --- | --- |
| Startup probe/check | Slow initialization could make liveness or readiness fire too early. | Startup succeeds only after mandatory initialization; liveness/readiness do not gate until startup succeeds. | Startup has no meaningful initialization beyond process launch. |
| Liveness check | The platform needs a restart signal for a stuck process. | Check local progress only; external dependency outages should not trigger restarts by default. | It includes slow or unavailable dependencies and causes restart loops. |
| Readiness check | Load balancers/orchestrators need a traffic-admission signal. | Fail readiness when the instance cannot serve core traffic or is draining. | It fails because an optional dependency is degraded but core traffic can continue. |
| Diagnostic health endpoint | Operators need dependency detail. | Report component status with careful cost, security, and sensitivity boundaries. | It becomes the only traffic-admission or restart signal without clear semantics. |
| Graceful shutdown/drain | Deployments or scale-down need safe stop behavior. | Fail readiness, stop accepting new work, drain in-flight work within `<drain window>`, flush/record required signals, and exit before hard kill. | The platform grace period is shorter than drain plus pre-stop work. |

## Accepted Examples
- "`/livez` checks only process progress and does not fail because the database is down; `/readyz` fails when database access is required for core traffic."
- "During shutdown, readiness fails before the listener stops accepting new work; in-flight requests drain for `<drain window>`; long-lived streams follow a separate close policy."
- "The diagnostic health endpoint can report optional dependency degradation, but traffic admission remains tied to whether core flow can serve."

## Rejected Examples
- "Liveness checks the database." Rejected because a dependency outage can cause restart loops without fixing the dependency.
- "Readiness remains true while draining." Rejected because traffic may continue arriving after the instance has begun shutdown.
- "Shutdown relies only on process signal handling." Rejected without a caller-visible drain rule, deadline, and long-lived connection policy.
- "Health endpoint performs expensive deep checks on every probe." Rejected because the probe itself can overload the service.

## Testable Failure Contracts
- Given optional dependency failure, diagnostics show degradation but readiness remains true if core traffic is still safe.
- Given core dependency failure, readiness becomes false after the specified threshold/hysteresis and returns true only after recovery criteria.
- Given liveness failure, the check points to local process inability to make progress rather than remote dependency state.
- Given shutdown, readiness changes before new traffic is admitted, in-flight requests are drained or timed out, and the process exits within the platform grace period.
- Given hijacked, streaming, or long-lived connections, the spec states whether they are closed, signaled, drained, or excluded from graceful shutdown.

## Exa Source Links
- Go `net/http` package and server controls: https://pkg.go.dev/net/http
- Go `Server.Shutdown`: https://pkg.go.dev/net/http#Server.Shutdown
- Microsoft Azure, Health Endpoint Monitoring pattern: https://learn.microsoft.com/en-us/azure/architecture/patterns/health-endpoint-monitoring
- Kubernetes, Liveness, Readiness, and Startup Probes: https://kubernetes.io/docs/concepts/configuration/liveness-readiness-startup-probes/
- Kubernetes, Configure Liveness, Readiness and Startup Probes: https://kubernetes.io/docs/tasks/configure-pod-container/configure-liveness-readiness-startup-probes/
- Kubernetes, Container Lifecycle Hooks: https://kubernetes.io/docs/concepts/containers/container-lifecycle-hooks
