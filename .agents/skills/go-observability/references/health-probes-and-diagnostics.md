# Health Probes And Diagnostics

## Load When
Load this when a change adds a readiness probe or dependency check, alters drain or shutdown behavior, or exposes a diagnostic surface.

## Decide

**Readiness is a cached background verdict.** The routes are `/health/live` and `/health/ready`; there is no startup probe. [Readiness, drain, and shutdown](../../go-reliability/references/readiness-drain-shutdown.md) owns the mechanism — the background refresher, the staleness bound, and the eviction cascade a per-request dependency check recreates. From this side, a new dependency check becomes a `Probe` on that refresher, not a call in the handler.

**Probe routes are exempt from admission control.** The in-flight limiter passes `/health/live` and `/health/ready` through. Shedding a probe would report the instance unhealthy at exactly the moment it is merely busy, which is the eviction-under-load failure again by another route.

**Diagnostics are a separate listener, and pprof is off.** `newDiagnosticsServer` serves `/metrics` and `/debug/buildinfo` on the private listener, adding pprof only under `observability.pprof.enabled`. Two details are load-bearing. Importing `net/http/pprof` registers every profile on `http.DefaultServeMux` from its `init` and cannot be opted out of; what makes that harmless is that both servers are constructed with an explicit `Handler`, so nothing serves the default mux — a change that serves `DefaultServeMux` anywhere publishes the profiles. And the listener takes its own `pprofWriteTimeout` of 65s, because `pprof.Profile` and `pprof.Trace` hold the response open for their `?seconds=` argument and an API write timeout truncates every CPU profile at the moment it becomes useful.

`/debug/buildinfo` is deliberately not behind the pprof gate: a build identifier is not the disclosure a heap dump is, and the moment an operator needs it most is when profiling is off, which is the shipped default.

**Telemetry can be degraded while the service is healthy.** A service exporting no traces still answers every request and reports ready. `RecordTraceExporterState` publishes that state as a gauge precisely so it is alertable; metrics setup succeeds independently of tracing setup so the gauge survives the failure it reports.

## Reject
- A dependency call in the probe handler — the probe then consumes the capacity it reports on.
- Serving `http.DefaultServeMux` anywhere in the process, because the pprof `init` has already registered every profile on it.

## Prove
- For a new probe: it is registered on the refresher, and its timeout fits inside `probeBudget`.
- For shutdown: readiness fails before drain begins, and exporter flush is proven rather than assumed.
- For a new diagnostic surface: it is on the diagnostics listener, and its gate and write timeout are stated.
