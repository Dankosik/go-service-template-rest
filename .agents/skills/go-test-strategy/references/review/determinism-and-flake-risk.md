# Determinism And Flake Risk

## Load When
Load this when tests involve sleeps, timers, randomness, process-global state, `t.Parallel()`, goroutine coordination, leak checks, race evidence, or `testing/synctest` suitability.

## Decide
- Flag nondeterminism only when the uncontrolled source can decide the result without proving the behavior.
- Distinguish a sleep that observes from a sleep that synchronizes. Only the synchronizing one is a merge risk; a sleep whose assertion does not depend on precise timing can be correct.
- Replace an uncontrolled source with the thing that makes the event observable: a gate, a channel, a recorded call order, an injected clock, or a controlled reader.
- Use `testing/synctest` only when the target module supports it and the bubble
  can contain the relevant goroutines and timers. Real network I/O, external
  processes, and container-backed work need explicit external synchronization.
- Read `-race` as shared-memory evidence over the interleavings that actually executed. It is not liveness proof, not ordering proof, and not evidence that no goroutine is stuck.
- Reach for repetition only after the risky interleaving is controlled; use a bounded focused `go test -count=5 -shuffle=on` run.

## Inspect
Inspect the target's current test fixtures and accepted timing contract. A fake
clock can make an exact interval directly observable when that equality is the
contract; a path or duration from another checkout is not its oracle.

## Reject
- "This test sleeps, so it is flaky." That names a symptom and leaves the ordering claim unproved; say which ordering the sleep is standing in for and what would record it directly.
- Objecting to `t.Parallel()` without a shared mutable fixture the subtests actually contend for.
- A high `-count` on a test that is already deterministic inside a bubble. Repetition of a controlled path buys nothing and costs the suite.

## Reopen
- Resolve API availability through [Go Modern
  Version](../../../go-modern-version/SKILL.md) before prescribing a test helper.
- Leak checks prove lifecycle cleanup, not ordering and not protocol progress.
- Check the target's `.golangci.yml`, applicable test APIs, and validation plan
  before claiming isolation hygiene is mechanically enforced. Report only the
  remaining shared-state risk that can change the test outcome.
- Real-time deadlines belong in a test as outer failure diagnostics, not as the mechanism that establishes the ordering.

## Prove
Pair the deterministic correction with the narrow named test, then add instrumentation only where it matches the risk: `-race` for shared memory, bounded `-count`/`-shuffle` for ordering confidence, leak checks for goroutine lifetime.
