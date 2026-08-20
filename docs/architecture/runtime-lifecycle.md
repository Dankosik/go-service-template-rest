# Runtime Lifecycle

Load for startup, readiness, drain, shutdown, or process-resource ownership.

1. `cmd/service/main.go` delegates to bootstrap.
2. Bootstrap creates the signal-aware root context, baseline metrics, and the
   immutable config snapshot.
3. It configures logging and telemetry, validates ingress admission, and probes
   enabled dependencies.
4. HTTP may serve while startup admission runs, but readiness stays false.
5. `internal/health.Service` runs enabled dependency probes under one readiness
   timeout; liveness remains process-only.
6. Shutdown marks draining, flips readiness off, waits the propagation delay,
   drains application transports, and flushes telemetry inside one process
   grace budget.

Config and dependency admission precede traffic acceptance. Bootstrap, not
handlers or feature services, owns process lifecycle and partial-startup
cleanup.
