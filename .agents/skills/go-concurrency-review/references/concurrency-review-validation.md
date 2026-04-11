# Concurrency Review Validation

Behavior Change Thesis: When loaded for evidence-quality symptoms, this file makes the model match proof to the concurrency failure mode instead of treating "tests passed" or `go test -race` as blanket validation.

## When To Load
Symptom: the review needs to judge whether concurrency evidence is enough, suggest validation commands, distinguish race proof from liveness proof, or write residual risk for missing tests.

## Decision Rubric
- Match the command to the defect: race detector for executed shared-memory races, liveness tests for deadlocks and shutdown hangs, deterministic gates for interleavings, fake clocks or `testing/synctest` for time-driven logic.
- `go test -race` is strong evidence for executed data races; it does not prove a protocol cannot deadlock or that every goroutine exits.
- A sleep-only test is weak evidence unless it merely supplements a stronger gate or completion assertion.
- Repeated runs with `-count=100` help only after the test controls the risky interleaving; do not turn scheduler luck into a proof strategy.
- If the reviewer cannot run or infer the needed command, state the residual risk precisely rather than saying "looks fine."

## Imitate
```text
[medium] [go-concurrency-review] broker/broker_test.go:52
Issue:
Axis: Tests And Validation Evidence; `TestUnsubscribe` sleeps for 10ms and then assumes the publisher goroutine has reached the send path. That timing does not prove the send/close interleaving that production relies on, and it can pass while `Publish` still races with `Unsubscribe`.
Impact:
The PR can merge a send-on-closed panic or blocked publisher path with tests that pass because the scheduler did not choose the bad interleaving.
Suggested fix:
Replace the sleep with a gate channel that parks `Publish` immediately before the send, then unsubscribe and assert the publisher exits without panic or block. Run it under the race detector.
Reference:
Validate with `go test -race ./internal/broker -run TestPublishUnsubscribeRace -count=100`.
```

Copy the shape: it ties the evidence gap to a production interleaving, not to abstract test coverage.

## Reject
```text
No findings. They ran the tests.
```

Reject this shape: it does not say which concurrent path was exercised, whether the race detector was used, or which liveness/shutdown behavior remains unproven.

```text
The race detector passed, so shutdown is safe.
```

Reject this shape: race-clean code can still deadlock, leak goroutines, or wait forever on shutdown.

## Agent Traps
- Do not demand every validation style for every finding; choose the proof that matches the failure.
- Do not turn missing tests into a finding when the code path is obviously safe and the risk is low; record residual risk instead.
- Do not use `time.Sleep` as the recommended primary coordination mechanism.
- Do not mention `testing/synctest` as a magic fix when the test crosses external I/O, network, process, or goroutine boundaries outside the controlled bubble.

## Validation Shape
- Race publication path: `go test -race ./internal/cache -run TestConfigReload -count=100`.
- Send/close interleaving: gate the sender before the send, then run `go test -race ./internal/broker -run TestPublishUnsubscribeRace -count=100`.
- Shutdown liveness: assert completion and run `go test ./internal/poller -run TestStopReturnsImmediately -count=100 -timeout=5s`.
- Residual risk wording: "Race evidence covers the shared map path, but not the shutdown wait." "No concurrency findings; residual risk is limited to unexercised production workload interleavings because only unit-level race evidence is present."
