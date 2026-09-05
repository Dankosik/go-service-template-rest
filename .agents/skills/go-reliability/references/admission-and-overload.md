# Admission And Overload

## Load When

Load when concurrency, fan-out, queued work, a pooled dependency, rejection, or
a new limiter/breaker changes.

## Decide

Reuse the three existing admission owners. `boundedAPIListener` caps accepted
connections before process allocation. `MaxInFlight` uses `TryAcquire` to shed
handlers promptly at `http.max_in_flight`; blocking acquire would recreate the
queue. Optional `RateLimit` applies caller policy after the global capacity
gate. Preserve 503 for capacity/not-ready, 429 for caller budget, and 504 for
spent request time. Health probes remain outside shedding.

Rate-limit key selection is service policy: proxy addresses may collapse the
fleet and forwarded headers are untrusted without an accepted edge. A new
pooled dependency gets its own concurrency/acquire budget when it must not
consume unrelated handler capacity. Non-HTTP process work uses the existing
background supervisor; durable work routes to the job owner.

There is no default breaker or retry policy because each adds a state machine
and load to a dependency-specific contract. Add one only when current failure
evidence and a useful degraded response justify it. Bound all queues, channels,
memory, retries, and tenant consumption; a buffer without an admission ceiling
only delays overload.

## Prove

Fill the selected limiter/pool and observe immediate documented rejection,
bounded in-flight work, distinct telemetry/status, unaffected health probes,
and recovery after capacity returns.
