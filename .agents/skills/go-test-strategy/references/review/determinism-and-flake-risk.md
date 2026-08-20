# Determinism And Flake Risk

## Load When
Load this when tests involve sleeps, timers, randomness, process-global state, `t.Parallel()`, goroutine coordination, leak checks, race evidence, or `testing/synctest` suitability.

## Decide
- Flag nondeterminism only when the uncontrolled source can decide the result without proving the behavior.
- Distinguish a sleep that observes from a sleep that synchronizes. Only the synchronizing one is a merge risk; a sleep whose assertion does not depend on precise timing can be correct.
- Replace an uncontrolled source with the thing that makes the event observable: a gate, a channel, a recorded call order, an injected clock, or a controlled reader.
- `testing/synctest` is this repository's form for time-driven behavior, and the bubble must contain every goroutine and timer under test. Real network I/O, external processes, and container-backed work disqualify it.
- Read `-race` as shared-memory evidence over the interleavings that actually executed. It is not liveness proof, not ordering proof, and not evidence that no goroutine is stuck.
- Reach for repetition only after the risky interleaving is controlled, and prefer the repository gate `make test-flake-smoke` (`-count=5 -shuffle=on`) over an ad-hoc high `-count`.

## Inspect
`cmd/service/internal/bootstrap/shutdown_test.go` is the house form: `t.Parallel()` outside, `synctest.Test(t, …)` inside, and an assertion that the propagation delay elapsed *exactly* — `time.Since(startedAt) != 20*time.Millisecond` fails the test. A fake clock inside a bubble turns a timing claim into an equality check, which is why the file needs no repetition flag at all.

## Reject
- "This test sleeps, so it is flaky." That names a symptom and leaves the ordering claim unproved; say which ordering the sleep is standing in for and what would record it directly.
- Objecting to `t.Parallel()` without a shared mutable fixture the subtests actually contend for.
- A high `-count` on a test that is already deterministic inside a bubble. Repetition of a controlled path buys nothing and costs the suite.

## Reopen
- The Go 1.26 API is `synctest.Test(t, func(t *testing.T))` with `synctest.Wait()`. `synctest.Run(func())` was the Go 1.24 `GOEXPERIMENT` form and no longer applies.
- Leak checks prove lifecycle cleanup, not ordering and not protocol progress.
- Isolation hygiene is already enforced mechanically and is not worth a finding: `t.Setenv` panics when the test or any ancestor is parallel, `usetesting` requires `t.Setenv`/`t.TempDir`/`t.Context`, and `make test-parallelism-check` runs `paralleltest,tparallel`.
- Real-time deadlines belong in a test as outer failure diagnostics, not as the mechanism that establishes the ordering.

## Prove
Pair the deterministic correction with the narrow named test, then add instrumentation only where it matches the risk: `-race` for shared memory, `make test-flake-smoke` for ordering confidence, leak checks for goroutine lifetime.
