# Readiness, Drain, And Shutdown

## Behavior Change Thesis
When loaded for symptom `a health probe, readiness input, signal handler, or teardown step changed`, this file makes the model extend the cached-readiness and single-deadline teardown that already exist instead of likely mistake `check the dependency inside the probe handler and give the new teardown step a timeout of its own`.

## When To Load
Load when a change touches health endpoints, what readiness depends on, signal handling, drain sequencing, or anything added to process teardown.

## Readiness Is Cached And Does No I/O
`internal/health/service.go` refreshes readiness on a background loop; `HealthReady` in `internal/infra/http/health_handlers.go` reads an atomic startup gate and `health.Cached()`. A probe that checked a pooled dependency would need that dependency's capacity in order to report on it: a saturated pool fails readiness, the orchestrator evicts the instance, and its traffic moves to instances that are already saturated. Adding a `PingContext` to the probe handler restores exactly that loop.

What keeps the cache honest is the rest of the package: `ErrNotEvaluated` fails closed until the first refresh completes, `ErrStale` refuses a verdict older than `probeBudget + 3*max(interval, probeBudget)` so a dead refresher cannot leave one standing, and `health.failure_threshold` (3) means one slow round-trip does not evict a serving instance. A new dependency becomes a `health.Probe`, not a new endpoint.

`internal/config/validate.go` deliberately states no rule tying `health.refresh_interval` to `http.readiness_timeout`: the handler does no I/O, so that budget bounds nothing the refresher does. The interval only has to be small against the orchestrator's probe period, which this service cannot see.

Liveness is unconditional. `HealthLive` returns ok without consulting anything, because a dependency outage that restarts healthy processes removes capacity without fixing the dependency.

## Drain Is A Sequence With A Signal At Each Step
`drainAndShutdown` in `cmd/service/internal/bootstrap/shutdown.go`, in order:

1. `StartDrain()` — readiness begins answering 503, logged as `readiness_disabled`.
2. Sleep `http.readiness_propagation_delay` (15s) so a load balancer notices before connections stop being accepted. It is spent **out of** `http.shutdown_timeout`, not added to it: `effectiveDrainBudget = shutdown_timeout - readiness_propagation_delay`.
3. `Shutdown` the API server, then diagnostics (2s), background cancel and join (5s), dependency close (5s), telemetry flush (5s).

Telemetry flushes last so it records what the earlier stages did. Background work is cancelled after the HTTP drain rather than on the signal, because the drain above it is still serving requests that depend on it. Dependency close is a bounded stage because `pgxpool.Close` blocks until every connection is returned and takes no context of its own.

## One Deadline, Not Four
`shutdownBudget` in `cmd/service/internal/bootstrap/run.go` clamps every stage to what remains of `http.grace_period`, with a 100ms floor so a stage cut short can still report that it was. As four independent ceilings the worst case was their sum — 25s of drain plus 17s of teardown against the 30s Kubernetes grants by default — and the stage that lost was always the flush that records the drain.

`validateShutdownGraceBudget` enforces `grace_period >= shutdown_timeout + shutdownTailBudget`. A new teardown step raises that constant, and that constant is the only thing relating this process's structure to the one number the platform gave the operator.

## Reject
- A teardown step with its own unclamped timeout, or a bare `defer close()` outside the budget: both can spend time the flush needs and get discarded by SIGKILL.
- Readiness that stays true while draining, or a drain that stops accepting before readiness has had time to propagate. The delay is the whole mechanism.

## Proof
`go test ./cmd/service/... -run 'Shutdown|Drain|Readiness|Grace'` and `go test ./internal/health/...`. Assert ordering — readiness false before the listener stops, dependencies closed after the drain — not that shutdown returned nil.
