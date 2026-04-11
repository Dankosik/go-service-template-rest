# Concurrency Review Validation Examples

## When To Load
Load this when a review needs to judge whether concurrency evidence is enough, suggest validation commands, distinguish race proof from liveness proof, or write residual risk for missing tests.

## Review Lens
Validation should match the failure mode. `go test -race` finds runtime-executed data races; it does not prove a protocol cannot deadlock, that shutdown joins all goroutines, or that a sleep-based test is deterministic.

## Bad Review Example
```text
No findings. They ran the tests.
```

Why it fails: it does not name which concurrent path was tested, whether the race detector was used, or which liveness/shutdown behavior remains unproven.

## Good Review Example
```text
[medium] [go-concurrency-review] broker/broker_test.go:52
Issue:
Axis: Tests And Validation Evidence; `TestUnsubscribe` sleeps for 10ms and then assumes the publisher goroutine has reached the send path. That timing does not prove the send/close interleaving that the production code relies on, and it can pass while `Publish` still races with `Unsubscribe`.
Impact:
The PR can merge a send-on-closed panic or blocked publisher path with tests that pass because the scheduler did not choose the bad interleaving.
Suggested fix:
Replace the sleep with a gate channel that parks `Publish` immediately before the send, then unsubscribe and assert the publisher exits without panic or block. Run it under the race detector.
Reference:
Go race detector runtime-coverage limits and `testing/synctest` guidance for deterministic concurrent tests; validate with `go test -race ./internal/broker -run TestPublishUnsubscribeRace -count=100`.
```

## Failure Mode
Write an evidence-gap finding or residual risk when:
- tests use `time.Sleep` as the only coordination for a concurrent interleaving;
- the PR changes shared state but no `-race` command is provided or obvious;
- `-race` passes but the defect is a pure protocol deadlock, blocked send, or shutdown hang;
- a context cancellation path has no test proving workers observe the canceled context;
- a timer or ticker loop has no test proving `Stop` or `Close` returns promptly;
- the changed code depends on `testing/synctest` but the test crosses external I/O, network, process, or goroutine boundaries outside the bubble.

## Smallest Safe Correction
Prefer validation fixes like:
- add gate channels to place goroutines at the risky interleaving;
- use `go test -race` for shared-memory or send/close races;
- add a liveness test with a completion channel and a short test timeout;
- use fake clocks or `testing/synctest` for time-driven concurrent code when available;
- loop a narrow deterministic test with `-count=100` only after removing scheduler sleeps;
- record residual risk if the reviewer cannot run or infer the needed command.

## Validation Evidence
Use commands like:
```bash
go test -race ./internal/broker -run TestPublishUnsubscribeRace -count=100
go test ./internal/poller -run TestStopReturnsImmediately -count=100 -timeout=5s
go test ./internal/worker -run TestCancelUnblocksWorkers -count=100
```

Phrase evidence precisely:
- "Race evidence covers the shared map path, but not the shutdown wait."
- "This liveness test proves `Stop` returns after cancel, but it does not prove all worker fan-out is bounded."
- "No concurrency findings; residual risk is limited to unexercised production workload interleavings because only unit-level race evidence is present."

## Source Links From Exa
- [Data Race Detector](https://go.dev/doc/articles/race_detector.html)
- [testing/synctest package docs](https://pkg.go.dev/testing/synctest)
- [Testing concurrent code with testing/synctest](https://go.dev/blog/synctest)
- [The Go Memory Model](https://go.dev/ref/mem)

