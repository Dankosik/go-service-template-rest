# Condition-Based Waiting For Go Tests

## Behavior Change Thesis
When loaded for sleep-based async tests, this file makes the model wait on an observable condition or event instead of inflating sleeps or hiding goroutine lifecycle bugs.

## When To Load
Load when a Go test uses `time.Sleep`, polls guessed timing, fails only on slower CI, or needs deterministic waiting for asynchronous work.

## Decision Rubric
- Name the behavior the test actually needs: worker drained, message committed, callback observed, server ready, goroutine exited, or clock advanced.
- Prefer an explicit event, channel, fake clock, or test hook when the production design already has a reliable signal.
- Use polling only when no event source exists, and make the condition read fresh state every loop.
- Keep timeout and interval small enough to localize failure, with a failure message that names the missing condition.
- Keep fixed delays only when timing itself is the behavior under test, such as debounce, throttle, lease expiry, or backoff.

## Imitate

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

```go
waitFor(t, 2*time.Second, 20*time.Millisecond, func() bool {
	return repo.PendingJobs() == 0
}, "job queue drain")
```

Copy the shape: the test names readiness in business or lifecycle terms, and the timeout failure points at the missing condition.

## Reject

```go
time.Sleep(2 * time.Second)
```

This makes the test slower without proving the worker is ready, drained, or stopped.

```go
waitFor(t, time.Minute, time.Second, func() bool {
	return len(debugGlobal) > 0
}, "thing")
```

This waits too long, relies on an implementation detail, and gives a vague failure message.

## Agent Traps
- Replacing a sleep with a larger sleep and calling it stabilization.
- Waiting on a private implementation detail that can change while the observable behavior is still wrong.
- Hiding a goroutine leak by extending the test timeout.
- Adding a helper without `t.Helper()`, making failures point inside the helper instead of the caller.
- Forgetting that package-order flakes still need the wider reproducer from `flaky-repro-controls-go.md`.

## Validation Shape
Show the old timing assumption with a repetition command when feasible, then rerun the same command after replacing the sleep. Use `-race` or `-cpu=1,4` only when the failure class points at shared state or scheduler sensitivity.
