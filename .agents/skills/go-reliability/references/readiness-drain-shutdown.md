# Readiness, Drain, And Shutdown

## Load When

Load when health, readiness inputs, signals, drain, or teardown changes.

## Decide

`internal/health/service.go` refreshes readiness in the background;
`HealthReady` reads only the startup gate and cached verdict. A probe-time
dependency call consumes the capacity it reports on and can evict an overloaded
instance into a more overloaded fleet. Add a `health.Probe`, preserve
fail-closed `ErrNotEvaluated`/`ErrStale`, and keep liveness independent of
dependency health.

`drainAndShutdown` disables readiness, waits the configured propagation delay,
then shuts down API, diagnostics, background work, dependencies, and telemetry
in that order. Background cancellation follows HTTP drain because draining
requests may still depend on it; telemetry closes last to record all stages.

All stages spend one remaining `http.grace_period` budget through
`shutdownBudget`; a new teardown step updates the shared tail budget rather than
adding an independent timeout. A bare blocking `defer close()` outside that
budget can consume the time needed by later flushes. Readiness propagation is
part of `shutdown_timeout`, not extra platform time.

## Reject

Reject I/O in readiness handlers, readiness that stays true while draining,
listener shutdown before propagation, an unclamped teardown timeout, or a
dependency close that can block without joining the common budget.

## Prove

Run focused shutdown/readiness tests and assert ordering: readiness turns false
before listener stop, background joins before dependencies close, and telemetry
flush remains inside the common deadline.
