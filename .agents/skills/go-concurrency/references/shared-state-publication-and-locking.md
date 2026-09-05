# Shared State Publication And Locking

## Load When

Load when shared maps/slices/pointers/interfaces cross goroutines, atomics or
`sync.Map` appear, or lock/condition scope changes.

## Decide

Name the happens-before edge and the state span it covers. Unlock/lock,
send/receive, close/observing receive, `Once`, completed goroutine/`Wait`, and an
observed atomic store/load are edges; startup order and elapsed time are not.

An atomic store publishes only preceding writes. Build the value completely and
keep it immutable after publication, or synchronize later mutations separately.
A readiness flag does not protect a mutable payload. `sync.Map.Range` is not a
snapshot; a multi-key invariant needs one lock or immutable snapshot. Prefer one
owner per invariant—mutex, owning goroutine, or swapped snapshot—because
per-field atomics can expose combinations no writer created.

Lock scope follows the invariant. Never hold it through user callbacks, channel
sends, HTTP/database calls, or other reentrant/blocking work without an explicit
reason. `sync.Cond` protects one predicate and rechecks it in a loop. `RWMutex`
is a measured throughput choice, not a default.

## Prove

Force writer/reader overlap at the publication edge and assert the invariant.
Race tests cover only executed schedules; a clean run never proves absence.
Record any residual immutability contract that runtime tooling cannot enforce.
