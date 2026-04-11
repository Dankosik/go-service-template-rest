# Deterministic Concurrency And Time Tests

## Behavior Change Thesis
When loaded for goroutines, channels, shutdown, timers, deadlines, or race-sensitive behavior, this file makes the model use deterministic handshakes, bounded exit checks, fake time where suitable, and race evidence instead of likely mistake: `time.Sleep`, pass-by-no-panic tests, or unobserved goroutine leaks.

## When To Load
Load this when tests touch goroutines, channels, worker pools, shutdown, backpressure, timers, tickers, deadlines, race detection, or `testing/synctest`. If the primary issue is error category or context propagation without goroutine scheduling, load `error-context-and-cancellation-tests.md` instead.

## Decision Rubric
- For "wait until goroutine reaches state", add an explicit `ready` handshake at the boundary being tested.
- For cancellation or shutdown, prove the blocked operation returns and inspect the final error or state.
- For max concurrency or duplicate side effects, use an atomic or mutex-protected fake and fail at the first overflow or duplicate.
- For timer and backoff behavior, prefer injected clocks or `testing/synctest` when the code can stay inside the fake-time bubble.
- Use a real timeout only as a diagnostic guard around a deterministic condition.
- Use `go test -race` or the repository race target for concurrency-sensitive changes when the platform/toolchain supports it.
- Verify `testing/synctest` in the active Go toolchain before relying on it. In this repo, `go.mod` currently targets Go 1.26.1, so `synctest` is expected to be available.

## Imitate
```go
func TestPublisherBlockedSendUnblocksOnCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	readyToSend := make(chan struct{})
	errCh := make(chan error, 1)
	out := make(chan Event)

	go func() {
		errCh <- Publish(ctx, out, Event{ID: "evt-1"}, readyToSend)
	}()

	<-readyToSend
	cancel()

	select {
	case err := <-errCh:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("Publish() error = %v, want context.Canceled", err)
		}
	case <-time.After(time.Second):
		t.Fatal("Publish() did not return after cancellation")
	}
}
```

Copy the shape: cancellation happens only after the goroutine confirms it reached the blocked send boundary, and the test observes exit.

```go
func TestRetryTimerFiresAfterBackoff(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		retried := false
		StartRetryLoop(t.Context(), time.Second, func() {
			retried = true
		})

		synctest.Wait()
		if retried {
			t.Fatal("retried before backoff elapsed")
		}

		time.Sleep(time.Second)
		synctest.Wait()
		if !retried {
			t.Fatal("retry did not run after backoff")
		}
	})
}
```

Copy the shape only when the code under test stays inside the `synctest` bubble and does not depend on real network I/O, external processes, or goroutines outside the bubble.

## Reject
```go
func TestPublisherBlockedSendUnblocksOnCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	go Publish(ctx, make(chan Event), Event{ID: "evt-1"}, nil)
	time.Sleep(10 * time.Millisecond)
	cancel()
	time.Sleep(10 * time.Millisecond)
}
```

Reject because it proves no observable outcome and can pass even if the goroutine leaks.

## Agent Traps
- Using a timeout as synchronization rather than as a guard around deterministic coordination.
- Forgetting to buffer `errCh` or `doneCh`, causing the goroutine under test to block on reporting.
- Calling `WaitGroup.Add` after a goroutine can call `Done`.
- Closing a channel from the receiver side without owning the close.
- Treating a clean race run as proof of liveness, ordering, or cancellation semantics.
- Using `synctest` for real network startup, process lifecycle, or code that spawns work outside the bubble.

## Validation Shape
- Run the focused test with `-count=1`.
- Add `-race` or `make test-race` when the changed path includes shared state, goroutines, worker pools, channel handoff, or cancellation races.
- Use repeated `-count=N` only after deterministic coordination exists; repetition is not a substitute for a proper handshake.
