// Package waittest owns the bounded waits an out-of-process test performs and
// the loopback address it reserves to perform them against.
//
// Three packages had already written the same two waits — poll-until-true and
// receive-or-fail — and the copies differed only in their names and their tick
// interval. What a copy costs is not lines: a wait that times out without saying
// what it was waiting for turns one flaky integration test into an unreadable
// one, and every copy has to remember to say it. Here the description is a
// parameter, so no call site can omit it.
//
// Every function fails the test instead of returning an error, so each must be
// called from the goroutine running the test.
package waittest

import (
	"context"
	"net"
	"testing"
	"time"
)

// tick is how often [Until] re-evaluates its predicate.
//
// It is a constant rather than a parameter because no caller has a reason to
// choose: the timeout is what a test asserts on, and the interval only trades
// idle CPU against how quickly a satisfied condition is noticed. It stays far
// below every timeout this repository waits with.
const tick = 20 * time.Millisecond

// Until blocks until predicate reports true, and fails the test when timeout
// expires first.
//
// The predicate is polled rather than awaited because these conditions are
// observed from outside the process that establishes them — an HTTP probe, a
// consumer's stream state, a row count — and none of them offers a channel to
// select on. Prefer [Receive] wherever the code under test does offer one: a
// poll can only report that a condition was not reached in time, never when it
// was reached.
func Until(tb testing.TB, timeout time.Duration, predicate func() bool, description string) {
	tb.Helper()

	UntilFunc(tb, timeout, predicate, func() string { return description })
}

// UntilFunc is [Until] with the description computed at the timeout instead of
// before the wait.
//
// A poll that gives up is worth more when it can say what it last saw — "outbox
// count for %q = 0, want 3" rather than "outbox count" — and the last-seen value
// only exists once the wait has failed. Prefer [Until] wherever the description
// is a constant; reach for this one only when the message reads the same state
// the predicate does.
//
// describe runs once, after the deadline, so it may be as expensive as the
// message needs and may read whatever the final poll left behind.
func UntilFunc(tb testing.TB, timeout time.Duration, predicate func() bool, describe func() string) {
	tb.Helper()

	ctx, cancel := context.WithTimeout(tb.Context(), timeout)
	defer cancel()
	ticker := time.NewTicker(tick)
	defer ticker.Stop()
	for {
		if predicate() {
			return
		}
		select {
		case <-ticker.C:
		case <-ctx.Done():
			tb.Fatalf("timed out waiting for %s", describe())
		}
	}
}

// Receive returns the next value sent on values, and fails the test when
// timeout expires first.
func Receive[T any](tb testing.TB, values <-chan T, timeout time.Duration, description string) T {
	tb.Helper()

	var value T
	received := false
	select {
	case value = <-values:
		received = true
	case <-time.After(timeout):
	}
	// The failure is reported outside the select, and behind a condition, so
	// that the function keeps one reachable exit. Fatalf ends the goroutine, so
	// any statement written after it — in the timeout case or after the select —
	// is dead code that go/unreachable-statement reports.
	if !received {
		tb.Fatalf("timed out waiting for %s", description)
	}
	return value
}

// ReceiveSignal waits for one send on a close-only channel. It exists so a
// caller that wants the event rather than a value does not have to discard one.
func ReceiveSignal(tb testing.TB, signal <-chan struct{}, timeout time.Duration, description string) {
	tb.Helper()

	_ = Receive(tb, signal, timeout, description)
}

// FreeTCPAddr reserves a loopback address, releases it, and returns it for a
// listener the test is about to place there.
//
// The gap between that release and the real bind is inherent: the kernel can
// only hand out a port nothing currently holds, and only a bind holds one. A
// fixed port is the alternative, and it fails whenever two packages test at
// once — which is the normal case here.
//
// purpose names what the address is for, so a failure says which listener could
// not be placed rather than only that some port was unavailable.
func FreeTCPAddr(tb testing.TB, purpose string) string {
	tb.Helper()

	listener, err := (&net.ListenConfig{}).Listen(tb.Context(), "tcp", "127.0.0.1:0")
	if err != nil {
		tb.Fatalf("reserve %s address: %v", purpose, err)
	}
	address := listener.Addr().String()
	if err := listener.Close(); err != nil {
		tb.Fatalf("release %s address: %v", purpose, err)
	}
	return address
}
