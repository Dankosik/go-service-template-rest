# Startup, Readiness, Liveness, And Shutdown

## Behavior Change Thesis
When loaded for symptom `startup, health probe, readiness, liveness, signal handling, HTTP drain, or shutdown sequencing changed`, this file makes the model distinguish process liveness, traffic readiness, and drain completion instead of likely mistake `approve one health endpoint or fire-and-forget shutdown as operationally sufficient`.

## When To Load
Load when a diff touches service bootstrap, health endpoints, Kubernetes probes, readiness gates, liveness checks, startup warmup, signal handling, `http.Server.Shutdown`, drain behavior, `RegisterOnShutdown`, or shutdown sequencing.

Keep findings local: ask for startup, health, and drain semantics in the changed service path. Hand off detailed goroutine shutdown ownership to `go-concurrency-review`, deployment probe policy to `go-devops-spec`, and global lifecycle design to `go-reliability-spec`.

## Decision Rubric
- Readiness returns success before the server has loaded required config, warmed mandatory state, or connected to critical dependencies.
- Liveness checks optional dependencies and can restart healthy processes during a dependency outage.
- Startup and liveness share a short timeout, so slow initialization can trigger crash loops.
- Readiness and liveness endpoints are identical without justification.
- Shutdown calls `Server.Shutdown` from a goroutine, but `main` exits when `ListenAndServe` returns `http.ErrServerClosed`.
- Shutdown tears down dependencies before the server stops accepting new requests.
- Long-lived or hijacked connections are ignored when the code depends on them draining.
- The service does not mark itself not-ready before teardown.
- Health checks perform expensive work on every probe and can create their own overload.

## Imitate

Bad finding shape to copy: conflating probes can turn a dependency outage into restart-driven capacity loss.

```go
func (h *Health) healthz(w http.ResponseWriter, r *http.Request) {
	if err := h.db.PingContext(r.Context()); err != nil {
		http.Error(w, "down", http.StatusServiceUnavailable)
		return
	}
	w.WriteHeader(http.StatusOK)
}
```

```text
[medium] [go-reliability-review] internal/http/health.go:18
Issue: The changed liveness/readiness endpoint depends on the DB for both probe types.
Impact: A database outage can cause the orchestrator to restart otherwise live instances, reducing capacity while the dependency is already unhealthy.
Suggested fix: Split liveness from readiness: liveness should prove the process can make progress; readiness can fail when critical dependencies or warmup are not ready for traffic.
Reference: Kubernetes probe semantics and Google SRE health-check death-spiral guidance.
```

Good correction shape: split process liveness from traffic readiness.

```go
func (h *Health) livez(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func (h *Health) readyz(w http.ResponseWriter, r *http.Request) {
	if !h.ready.Load() {
		http.Error(w, "not ready", http.StatusServiceUnavailable)
		return
	}
	if err := h.db.PingContext(r.Context()); err != nil {
		http.Error(w, "dependency not ready", http.StatusServiceUnavailable)
		return
	}
	w.WriteHeader(http.StatusOK)
}
```

Bad finding shape to copy: shutdown started in a goroutine is not graceful if `main` can exit first.

```go
go func() {
	<-signals
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = srv.Shutdown(ctx)
}()

if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
	return err
}
return nil
```

Good correction shape: mark not-ready, call bounded `Shutdown`, and wait for completion.

```go
shutdownDone := make(chan error, 1)
go func() {
	<-signals
	ready.Store(false)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	shutdownDone <- srv.Shutdown(ctx)
}()

if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
	return err
}
return <-shutdownDone
```

If background goroutine ownership, channel closure, or signal coordination gets complex, escalate to `go-concurrency-review`.

## Reject

```go
func healthz(w http.ResponseWriter, r *http.Request) {
	if err := expensiveDeepCheck(r.Context()); err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}
	w.WriteHeader(http.StatusOK)
}
```

Reject when this endpoint is used as liveness; it can restart a live process or overload the dependency with probes.

```go
go func() { _ = srv.Shutdown(ctx) }()
return srv.ListenAndServe()
```

Reject when the caller can return after `http.ErrServerClosed` before drain finishes.

```go
db.Close()
_ = srv.Shutdown(ctx)
```

Reject because in-flight handlers may still need the dependency being torn down.

## Agent Traps
- Do not assume `healthz` means both readiness and liveness; ask which probe calls it.
- Do not require liveness to check every dependency. A dependency outage should usually affect readiness, not kill healthy capacity.
- Do not weaken liveness to cover slow startup; use startup gating when the platform supports it.
- Do not treat `Shutdown` as covering hijacked or long-lived connections unless the code handles them separately.
- Do not escalate all probe changes to delivery policy; keep local when the code clearly wires the wrong endpoint or drain order.

## Validation Shape
- `go test ./... -run 'Test.*(Health|Ready|Readiness|Live|Liveness|Startup)'`
- `go test ./... -run 'Test.*(Shutdown|Signal|Drain|ServerClosed)'`
- `go test -race ./...` when readiness flags, shutdown channels, or goroutine lifecycle changed.
- `kubectl describe pod <pod>` and `kubectl get endpoints <service>` only when validating a live Kubernetes deployment.

Prefer unit or integration tests that assert readiness flips to false before shutdown closes dependencies.
