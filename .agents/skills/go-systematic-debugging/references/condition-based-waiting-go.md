# Condition-Based Waiting For Go Tests

## Overview
Sleep-based synchronization (`time.Sleep`) is a common source of flakes.
Wait for a condition, not for guessed timing.

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

## Guidelines
- timeout must be explicit and tied to expected behavior
- polling interval should be bounded, often in the `10-50ms` range
- failure message must describe the missing condition
- the condition function must read fresh state on each loop
- prefer the narrowest condition that proves the behavior under test

## Fixed-Delay Exception
Use a fixed delay only when timing itself is the behavior under test, such as debounce, throttle, or lease expiry.
Document why a fixed delay is required and how its value was derived.
