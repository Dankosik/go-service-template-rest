# Goroutine Leak Profiles

## Load When

Load this reference when:

- a change creates or owns goroutines, channels, locks, wait groups, or shutdown paths;
- goroutine count grows, shutdown stalls, or production liveness is investigated;
- pprof diagnostics are added or changed.

## Contract

`goroutineleak` is production evidence for a high-confidence subset of
permanent blocking. It does not prevent leaks, and an empty profile is not
proof that no goroutine is leaking.

Before implementation, record the goroutine owner, maximum population, stop
signal, every blocking site and its unblock event, join point, and whether any
abandonment is valid only because the process exits next.

## Coverage

The profile can report permanent waits on channel send and receive (including
nil channels), blocking `select`, `sync.Mutex`, `sync.RWMutex`,
`sync.WaitGroup`, and `sync.Cond`.

It does not cover file or network I/O, syscalls, runnable busy loops, or custom
synchronization outside supported primitives. A synchronization object that
remains reachable from a global, registry, or runnable goroutine may hide a
real leak. Remove stale lifecycle references after the owner has stopped and
joined.

## Design Rules

- Prefer cancellation-aware sends and receives.
- Give every background activity an owner-controlled stop and join.
- Treat a result-channel buffer as a liveness proof only when its capacity
  covers the maximum sends possible after the receiver stops; state that bound.
- Do not retain channels, locks, wait groups, or stopped runtime instances past
  their owner lifetime.
- A shutdown deadline may abandon work only at an executable process boundary
  that proceeds directly to process exit. Restartable components remain fully
  joinable.
- Treat a non-empty profile as strong evidence of irreversible blocking under
  the supported model. The reported stack is the blocking site, not necessarily
  the root cause.

## Evidence Selection

- Use `testing/synctest` for deterministic time and scheduling.
- Use package `goleak` for unexpected test survivors, including unsupported
  blocking classes.
- Use `-race` for conflicting shared-memory access.
- Use the ordinary `goroutine` profile for broad inventory, including I/O waits.
- Use `goroutineleak` for high-confidence production partial deadlocks.

No one of these replaces the others.

## Reject

Reject treating an empty profile as proof of no leaks; replacing `goleak`,
`synctest`, or `-race`; enabling pprof by default or exposing it publicly;
scraping `goroutineleak` at metrics frequency; hiding an unbounded protocol
behind an unexplained buffer; intentionally leaking in a package test process;
or returning from a reusable component while owned goroutines still run.

## Prove

For a changed concurrent path, deterministically reach each relevant blocking
site, stop its owner, assert the join, run the package's existing `goleak`
gate, use `-race` when shared state overlaps, and test the pprof gate when
diagnostics routing changes.

During an incident, capture build identity, `goroutineleak`, the ordinary
goroutine profile, timestamp and workload, and whether pprof exposure changed.
