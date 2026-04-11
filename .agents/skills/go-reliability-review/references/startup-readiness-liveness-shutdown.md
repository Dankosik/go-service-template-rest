# Startup, Readiness, Liveness, And Shutdown

## When To Load
Load this reference when a diff touches service bootstrap, health endpoints, Kubernetes probes, readiness gates, liveness checks, startup warmup, signal handling, `http.Server.Shutdown`, drain behavior, `RegisterOnShutdown`, or shutdown sequencing.

Keep findings local: ask for startup, health, and drain semantics in the changed service path. Hand off detailed goroutine shutdown ownership to `go-concurrency-review`, deployment probe policy to `go-devops-spec`, and global lifecycle design to `go-reliability-spec`.

## Review Smells
- Readiness returns success before the server has loaded required config, warmed mandatory state, or connected to critical dependencies.
- Liveness checks optional dependencies and can restart healthy processes during a dependency outage.
- Startup and liveness share a short timeout, so slow initialization can trigger crash loops.
- Readiness and liveness endpoints are identical without justification.
- Shutdown calls `Server.Shutdown` from a goroutine, but `main` exits when `ListenAndServe` returns `http.ErrServerClosed`.
- Shutdown tears down dependencies before the server stops accepting new requests.
- Long-lived or hijacked connections are ignored when the code depends on them draining.
- The service does not mark itself not-ready before teardown.
- Health checks perform expensive work on every probe and can create their own overload.

## Failure Modes
- Pods admit traffic before they can serve it, causing cold-start errors.
- A dependency outage becomes a restart storm because liveness checks are dependency checks.
- Shutdown drops in-flight requests or exits before cleanup completes.
- Health-checking consumes enough resources to worsen an overload.
- Load balancers keep sending traffic to an instance that is already draining.

## Review Examples

Bad: one health endpoint handles both readiness and liveness and depends on the DB.

```go
func (h *Health) healthz(w http.ResponseWriter, r *http.Request) {
	if err := h.db.PingContext(r.Context()); err != nil {
		http.Error(w, "down", http.StatusServiceUnavailable)
		return
	}
	w.WriteHeader(http.StatusOK)
}
```

Review finding shape:

```text
[medium] [go-reliability-review] internal/http/health.go:18
Issue: The changed liveness/readiness endpoint depends on the DB for both probe types.
Impact: A database outage can cause the orchestrator to restart otherwise live instances, reducing capacity while the dependency is already unhealthy.
Suggested fix: Split liveness from readiness: liveness should prove the process can make progress; readiness can fail when critical dependencies or warmup are not ready for traffic.
Reference: Kubernetes probe semantics and Google SRE health-check death-spiral guidance.
```

Good: split process liveness from traffic readiness.

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

Bad: shutdown can return before graceful drain finishes.

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

Good: wait for shutdown completion.

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

## Smallest Safe Fix
- Split liveness and readiness when one endpoint conflates process health with dependency readiness.
- Add a startup probe or startup gate for slow initialization instead of weakening liveness globally.
- Mark the instance not-ready before shutdown drains.
- Call `http.Server.Shutdown` with a bounded context and wait for it before process exit.
- Stop accepting traffic before closing shared dependencies used by in-flight requests.
- Keep probe handlers cheap, bounded, and explicit about critical versus optional dependencies.
- Register or separately drain long-lived hijacked connections when the service uses them.

## Validation Commands
- `go test ./... -run 'Test.*(Health|Ready|Readiness|Live|Liveness|Startup)'`
- `go test ./... -run 'Test.*(Shutdown|Signal|Drain|ServerClosed)'`
- `go test -race ./...` when readiness flags, shutdown channels, or goroutine lifecycle changed.
- `kubectl describe pod <pod>` and `kubectl get endpoints <service>` only when validating a live Kubernetes deployment.

Prefer unit or integration tests that assert readiness flips to false before shutdown closes dependencies.

## Exa Source Links
- Go `net/http` `Server.Shutdown` documentation: https://pkg.go.dev/net/http#Server.Shutdown
- Kubernetes liveness, readiness, and startup probes concept: https://kubernetes.io/docs/concepts/configuration/liveness-readiness-startup-probes/
- Kubernetes probe configuration guide: https://kubernetes.io/docs/tasks/configure-pod-container/configure-liveness-readiness-startup-probes/
- Azure Health Endpoint Monitoring pattern: https://learn.microsoft.com/en-us/azure/architecture/patterns/health-endpoint-monitoring
- Google SRE Addressing Cascading Failures, health-check death spirals: https://sre.google/sre-book/addressing-cascading-failures/

