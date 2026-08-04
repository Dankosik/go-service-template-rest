# Health Probes And Diagnostics

## Behavior Change Thesis
When loaded for readiness, probes, or debug surfaces, this file makes the model extend the cached readiness verdict instead of the likely mistake: adding a per-request dependency check to the probe route, which makes the probe consume the capacity it reports on and turns one slow dependency into a fleet-wide eviction.

## When To Load
Load this when a change adds a readiness probe or dependency check, alters drain or shutdown behavior, or exposes a diagnostic surface.

## Decision Rubric

**Readiness is a cached background verdict.** The routes are `/health/live` and `/health/ready`; there is no startup probe. `internal/health` refreshes on an interval through `Watch` and serves the last result from `Cached`. Per-request evaluation is the design this package exists to prevent: a pooled database ping needs a pool connection, so a saturated pool fails readiness, the orchestrator evicts the instance, its traffic lands on instances already saturated. A new dependency check becomes a `Probe` on that refresher, not a call in the handler.

Three properties of that cache carry the correctness and are easy to drop when extending it. A verdict older than `probeBudget + 3*max(interval, probeBudget)` is refused rather than served, because a refresher that died leaves the reassuring answer standing forever — the last thing a healthy service writes is "healthy". `FailureThreshold` holds the previous verdict through a blip, but only for an instance that was already healthy; one that never came up fails immediately. `probeBudget` is separate from `interval` so a configured probe timeout is not silently clamped to the refresh period.

**Probe routes are exempt from admission control.** The in-flight limiter passes `/health/live` and `/health/ready` through. Shedding a probe would report the instance unhealthy at exactly the moment it is merely busy, which is the eviction-under-load failure again by another route.

**Diagnostics are a separate listener, and pprof is off.** `newDiagnosticsServer` serves `/metrics` and `/debug/buildinfo` on the private listener, adding pprof only under `observability.pprof.enabled`. Two details are load-bearing. Importing `net/http/pprof` registers every profile on `http.DefaultServeMux` from its `init` and cannot be opted out of; what makes that harmless is that both servers are constructed with an explicit `Handler`, so nothing serves the default mux — a change that serves `DefaultServeMux` anywhere publishes the profiles. And the listener takes its own `pprofWriteTimeout` of 65s, because `pprof.Profile` and `pprof.Trace` hold the response open for their `?seconds=` argument and an API write timeout truncates every CPU profile at the moment it becomes useful.

`/debug/buildinfo` is deliberately not behind the pprof gate: a build identifier is not the disclosure a heap dump is, and the moment an operator needs it most is when profiling is off, which is the shipped default.

**Telemetry can be degraded while the service is healthy.** A service exporting no traces still answers every request and reports ready. `RecordTraceExporterState` publishes that state as a gauge precisely so it is alertable; metrics setup succeeds independently of tracing setup so the gauge survives the failure it reports.

## Reject
- A dependency call in the probe handler, because the probe then fails for the same saturation it is meant to report and the orchestrator amplifies it.
- Serving `http.DefaultServeMux` anywhere in the process, because the pprof `init` has already registered every profile on it.

## Validation Shape
- For a new probe: it is registered on the refresher, and its timeout fits inside `probeBudget`.
- For shutdown: readiness fails before drain begins, and exporter flush is proven rather than assumed.
- For a new diagnostic surface: it is on the diagnostics listener, and its gate and write timeout are stated.
