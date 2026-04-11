# Deterministic Concurrency And Time Tests

## When To Load
Load this when tests touch goroutines, channels, worker pools, cancellation, shutdown, backpressure, timers, tickers, deadlines, race detection, or `testing/synctest`.

## Determinism Rules
- Do not use `time.Sleep` as a hope that another goroutine reached a state; use an explicit handshake.
- Use a bounded timeout only as a diagnostic guard around a deterministic condition.
- Prove blocking operations unblock on cancel or shutdown.
- Keep channel send/close ownership explicit.
- Use `testing/synctest` when code can run inside a self-contained bubble and fake time makes the test simpler.
- Avoid `testing/synctest` when the test depends on real network I/O, external processes, goroutines outside the bubble, or mutex blocking as the main synchronization point.
- Validate concurrency-sensitive changes with `go test -race` or the repo race target when supported.

## Good Example
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

Why it is good: cancellation happens only after the goroutine confirms it is at the publication boundary.

## Bad Example
```go
func TestPublisherBlockedSendUnblocksOnCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	go Publish(ctx, make(chan Event), Event{ID: "evt-1"}, nil)
	time.Sleep(10 * time.Millisecond)
	cancel()
	time.Sleep(10 * time.Millisecond)
}
```

Why it is bad: it proves no observable outcome and can pass even if the goroutine leaks.

## Synctest Example
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

Use this only when `testing/synctest` is available in the repo's Go version and the code under test stays inside the bubble.

## Assertion Patterns
- Assert bounded exit: receive from `errCh` or `doneCh` and inspect the error or final state.
- Assert max concurrency with an atomic in-flight counter and an immediate fail-on-overflow signal.
- Assert sibling shutdown by blocking one worker behind a channel, failing another, then requiring the blocked worker to observe cancellation.
- Assert no duplicate side effects under concurrency with a thread-safe fake that records call count and payload.
- Assert wrapped fatal causes with `errors.Is` or `errors.As`.

## Deterministic Coordination Patterns
- `ready := make(chan struct{})` closed just before a goroutine blocks or sends.
- `release := make(chan struct{})` to hold work until all goroutines are queued.
- `done := make(chan error, 1)` so goroutine completion cannot block.
- `sync.WaitGroup` only when every `Add` happens before the goroutine can call `Done`.
- `atomic.Int64` or a mutex-protected fake for call counts shared across goroutines.
- `testing/synctest.Wait()` to wait until goroutines in a bubble are durably blocked.

## Repository-Local Cues
- `internal/infra/http/goleak_test.go` installs goleak for the HTTP package.
- `cmd/service/internal/bootstrap/main_shutdown_test.go` verifies shutdown ordering and cancellation categories.
- Some existing server lifecycle tests use short sleeps while polling real network startup. Prefer explicit hooks for new pure concurrency tests.

## Exa Source Links
- [Go race detector article](https://go.dev/doc/articles/race_detector.html)
- [testing/synctest package](https://pkg.go.dev/testing/synctest)
- [Go testing package](https://pkg.go.dev/testing)
- [Go context package](https://pkg.go.dev/context)

