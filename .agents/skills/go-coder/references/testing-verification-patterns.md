# Testing And Verification Patterns

## Behavior Change Thesis
When loaded for test or verification pressure, this file makes the model prove the changed behavior at the smallest reliable layer instead of adding broad, brittle, stale, or ceremonial tests and commands.

## When To Load
Load this when implementation work adds or changes tests, fuzz tests, benchmarks, deterministic seams, golden files, failure messages, concurrency proof, or final verification commands.

## Decision Rubric
- Prefer a regression test that would fail before the change over broad coverage churn.
- Test at the smallest layer where the changed behavior is observable.
- Use direct tests for one or two clear cases; use tables when cases are genuinely parallel.
- Make failure messages include the operation, input, got value, and expected value when practical.
- Control clocks, randomness, goroutine completion, temp files, external I/O, and generated IDs.
- For Go 1.25+ self-contained concurrent/time tests, consider `testing/synctest`; avoid it when correctness depends on goroutines outside the synctest bubble, real network, external process, or non-durably-blocking I/O/lock behavior.
- Use `t.TempDir` for ordinary scratch files; reserve Go 1.26+ `t.ArtifactDir` for outputs a human or CI needs to inspect, and remember they persist only when `-artifacts` is set.
- Match verification commands to the claim; do not claim repository-wide readiness from a stale or too-narrow command.

## Imitate
Make failures useful.

```go
if got != want {
	t.Fatalf("NormalizeEmail(%q) = %q, want %q", input, got, want)
}
```

Use a direct test when the setup is one obvious case.

```go
func TestNormalizeEmailTrimsSpace(t *testing.T) {
	got := NormalizeEmail(" ada@example.com ")
	if got != "ada@example.com" {
		t.Fatalf("NormalizeEmail() = %q, want %q", got, "ada@example.com")
	}
}
```

Use a table when the cases share the same shape and each case adds signal.

```go
func TestNormalizeEmailRejectsInvalidInput(t *testing.T) {
	tests := []struct {
		name string
		in   string
	}{
		{name: "empty", in: ""},
		{name: "missing at", in: "ada.example.com"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NormalizeEmail(tt.in)
			if !errors.Is(err, ErrInvalidEmail) {
				t.Fatalf("NormalizeEmail(%q) error = %v, want %v", tt.in, err, ErrInvalidEmail)
			}
		})
	}
}
```

Use fuzzing for cheap deterministic parser or validator invariants.

```go
func FuzzParseEmail(f *testing.F) {
	f.Add("ada@example.com")
	f.Fuzz(func(t *testing.T, in string) {
		email, err := ParseEmail(in)
		if err != nil {
			return
		}
		if _, err := ParseEmail(email.String()); err != nil {
			t.Fatalf("ParseEmail(%q).String() = %q, reparses with %v", in, email.String(), err)
		}
	})
}
```

## Reject
Reject vague failures.

```go
if got != want {
	t.Fatal("wrong result")
}
```

Reject table ceremony for one case.

```go
tests := []struct {
	name string
	in   string
	want string
}{
	{name: "trims", in: " ada@example.com ", want: "ada@example.com"},
}
```

Reject sleep-based async tests.

```go
go worker.Run()
time.Sleep(50 * time.Millisecond)
if !worker.Ready() {
	t.Fatal("worker not ready")
}
```

## Agent Traps
- Adding an assertion library for simple comparisons when plain Go gives clearer failures.
- Comparing exact error strings when the contract is `errors.Is`, `errors.AsType`/`errors.As`, status code, or exported type.
- Hiding important setup in one-off helpers that do not call `t.Helper`.
- Depending on map iteration order, log formatting, wall-clock sleeps, or exact generated IDs unless that is the contract.
- Using fuzzing for slow, stateful, networked, or nondeterministic behavior.
- Reporting "tested" from a command that did not cover the changed package.
- Using Go 1.26+ `t.ArtifactDir` for ordinary scratch files or assertions instead of persisted test artifacts.

## Validation Shape
- Start with the smallest proving command, such as `go test ./internal/orders -run TestCreateOrder`.
- Use `go test ./...` when the change touches shared packages, generated contracts, or cross-cutting behavior.
- Use `go test -race` for concurrency, shared state, background work, and resource-lifetime changes.
- Use repeated targeted runs for flakes: `go test ./pkg/foo -run TestName -count=100`.
- Use fuzzing for parser and validator invariants: `go test ./pkg/foo -fuzz=FuzzParseEmail -fuzztime=30s`.
- Use `t.TempDir`, `t.Cleanup`, `httptest`, fake clocks, and channels to keep tests deterministic.
