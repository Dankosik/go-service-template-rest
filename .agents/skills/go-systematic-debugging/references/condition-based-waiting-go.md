# Condition-Based Waiting For Go Tests

## When To Load
Load this reference when a Go test uses `time.Sleep`, polls guessed timing, fails only on slower CI, or needs a deterministic way to wait for asynchronous work.

Use it to prove readiness through observable state instead of waiting for wall-clock luck.

## Commands
Use repetition to prove the old timing assumption and the replacement:

```bash
go test ./path/to/pkg -run '^TestName$' -count=100 -v
go test ./path/to/pkg -run '^TestName$' -race -count=50 -v
go test ./path/to/pkg -run '^TestName$' -cpu=1,4 -count=50 -v
go test ./path/to/pkg -run '^TestName$' -count=1 -timeout=30s -v
```

If the sleep is hiding package-order leakage, combine this with `references/flaky-repro-controls-go.md` and use a wider `-run` or package-level `-shuffle` command.

## Evidence To Capture
- exact test command and failure frequency before the change
- the condition the test really needs, such as "worker drained" or "message committed"
- timeout and polling interval, with a behavioral reason for both
- failure message that reports the missing condition
- whether `-race`, `-cpu`, or CI speed changes the signal

## Recommended Pattern

```go
func waitFor(t *testing.T, timeout time.Duration, interval time.Duration, cond func() bool, what string) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(interval)
	}
	t.Fatalf("timeout waiting for %s after %s", what, timeout)
}
```

Usage:

```go
waitFor(t, 2*time.Second, 20*time.Millisecond, func() bool {
	return repo.PendingJobs() == 0
}, "job queue drain")
```

Prefer an explicit event, channel, fake clock, or test hook over polling when the production design already has a reliable signal.

## Bad Debugging Moves
- replacing `time.Sleep(100 * time.Millisecond)` with a larger sleep
- waiting for an implementation detail unrelated to the behavior under test
- hiding a goroutine leak by making the test wait longer
- using a timeout so large that failure no longer localizes the missing condition
- adding a helper whose failure message says only "timed out"

## Good Debugging Moves
- wait on the narrowest behavior that proves readiness
- keep timeout and interval explicit
- make the condition read fresh state on each loop
- use `t.Helper()` so failure points at the test call site
- keep fixed delays only when timing itself is the behavior under test, such as debounce, throttle, lease expiry, or backoff

## Source Links
- [testing package](https://pkg.go.dev/testing)
- [context package](https://pkg.go.dev/context)
- [Go blog: pipelines and cancellation](https://go.dev/blog/pipelines)
- [Go data race detector](https://go.dev/doc/articles/race_detector)
