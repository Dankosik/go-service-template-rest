# Condition-Based Waiting For Go Tests

## Overview
Sleep-based test synchronization (`time.Sleep`) is a common source of flakes.
Wait for a condition, not for guessed timing.

## Use When
- async behavior is under test (worker completion, queue drain, eventually consistent read)
- CI shows intermittent failures
- tests become unstable under load or parallel execution

## Prefer This Pattern

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

## Guidelines
- timeout must be explicit and tied to expected behavior
- polling interval should be bounded (for example 10-50ms)
- failure message must describe missing condition
- condition function must read fresh state each loop

## When Fixed Delay Is Acceptable
Only when timing itself is the behavior under test (debounce/throttle/lease expiry).
Document why fixed delay is required and how value is derived.
