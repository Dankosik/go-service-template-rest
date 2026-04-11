# Testing And Verification Patterns

## When To Load
Load this when implementation work adds or changes tests, fuzz tests, benchmarks, verification commands, deterministic seams, golden files, failure messages, or concurrency/lifecycle proof.

## Good/Bad Examples

Bad: failure output loses the input, got value, and expected value.

```go
if got != want {
	t.Fatal("wrong result")
}
```

Good: make the failure useful to a future reader.

```go
if got != want {
	t.Fatalf("NormalizeEmail(%q) = %q, want %q", input, got, want)
}
```

Bad: table ceremony for one obvious case.

```go
tests := []struct {
	name string
	in   string
	want string
}{
	{name: "trims", in: " ada@example.com ", want: "ada@example.com"},
}
for _, tt := range tests {
	t.Run(tt.name, func(t *testing.T) {
		got := NormalizeEmail(tt.in)
		if got != tt.want {
			t.Fatalf("got %q, want %q", got, tt.want)
		}
	})
}
```

Good: use the direct test when it is clearer.

```go
func TestNormalizeEmailTrimsSpace(t *testing.T) {
	got := NormalizeEmail(" ada@example.com ")
	if got != "ada@example.com" {
		t.Fatalf("NormalizeEmail() = %q, want %q", got, "ada@example.com")
	}
}
```

Good: use a table when cases are genuinely parallel.

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

Bad: sleeping to test asynchronous behavior.

```go
go worker.Run()
time.Sleep(50 * time.Millisecond)
if !worker.Ready() {
	t.Fatal("worker not ready")
}
```

Good: test a signal or controlled seam.

```go
ready := make(chan struct{})
go worker.Run(ready)

select {
case <-ready:
case <-time.After(time.Second):
	t.Fatal("worker did not become ready")
}
```

Good: use fuzzing for parsers or validators with cheap, deterministic invariants.

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

## Common False Simplifications
- Adding an assertion library for simple comparisons when Go failure messages would be clearer.
- Comparing exact error strings when the contract is `errors.Is`, `errors.As`, status code, or exported type.
- Turning every test into a table; a single direct test can be better.
- Hiding important setup in one-off helpers that do not call `t.Helper`.
- Depending on wall-clock sleeps, map iteration order, log formatting, or exact generated IDs unless those are the contract.
- Using fuzzing for slow, stateful, networked, or nondeterministic behavior.
- Claiming verification from a stale command or from a command that did not cover the changed package.

## Validation Or Test Patterns
- Start with the smallest proving command, such as `go test ./internal/orders -run TestCreateOrder`.
- Use `go test ./...` when the change touches shared packages, generated contracts, or cross-cutting behavior.
- Use `go test -race` for concurrency, shared state, background work, and resource lifetime changes.
- Use repeated targeted runs for flakes: `go test ./pkg/foo -run TestName -count=100`.
- Use fuzzing for input parsers and validators: `go test ./pkg/foo -fuzz=FuzzParseEmail -fuzztime=30s`.
- Use `t.TempDir`, `t.Cleanup`, `httptest`, and explicit fake clocks or channels to keep tests deterministic.
- In Go 1.26+, use `t.ArtifactDir` only for artifacts a human or CI needs to inspect; do not turn tests into file-output workflows without a reason.

## Source Links Gathered Through Exa
- [testing package](https://pkg.go.dev/testing)
- [Go Fuzzing](https://go.dev/doc/fuzz)
- [Go Test Comments](https://go.dev/wiki/TestComments)
- [Go Code Review Comments](https://go.dev/wiki/CodeReviewComments)
- [Data Race Detector](https://go.dev/doc/articles/race_detector)
- [Go 1.26 release notes](https://go.dev/doc/go1.26)
