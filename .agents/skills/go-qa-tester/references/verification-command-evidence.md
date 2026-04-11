# Verification Command Evidence

## When To Load
Load this before final handoff, when selecting validation commands, narrowing failing tests, adding race/fuzz/integration evidence, or reporting what was actually run.

## Command Selection
- Start with the narrowest command that proves the changed package or test name.
- Add broader repository commands when the change touches shared behavior or build/test infrastructure.
- Prefer repository `make` targets as the normal interface when they exist.
- Use raw `go test` commands to focus diagnosis or accelerate a package loop; do not silently let them replace the repo's required quality gate.
- Use `-count=1` when you need fresh execution rather than cached package results.
- Use `-run` to pin the relevant suite or subtest.
- Use `-race` for concurrency-sensitive changes when the platform/toolchain supports it.
- Use `-fuzz` plus bounded `-fuzztime` only for fuzzing work, not as a substitute for seed-corpus regression execution.
- Use integration targets or build tags only when the test depends on external services and the repo already supports that mode.

## Common Commands
```bash
go test ./internal/infra/http -run 'TestRouterHTTPPolicy' -count=1
go test ./internal/infra/postgres -run 'TestPingHistoryRepository' -count=1
go test -race ./cmd/service/internal/bootstrap -run 'Test.*Shutdown' -count=1
go test ./... -count=1
go test ./internal/parser -run '^$' -fuzz=FuzzParse -fuzztime=30s
make test
make test-race
make test-integration
make check
make check-full
```

Adapt package paths and test names to the actual changed surface.

## Good Test Example
```go
func TestWorkerStopsOnCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := runWorker(ctx, fakeSource{})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("runWorker() error = %v, want context.Canceled", err)
	}
}
```

Verification should include a focused fresh run such as:

```bash
go test ./internal/worker -run '^TestWorkerStopsOnCancellation$' -count=1
```

## Bad Test Example
```go
func TestWorkerStops(t *testing.T) {
	go runWorker(context.Background(), fakeSource{})
	time.Sleep(10 * time.Millisecond)
}
```

Why it is bad: it has no observable assertion, can leak the goroutine, and gives validation commands nothing meaningful to prove.

## Good Report Example
```text
Implemented repository tests for query error wrapping, null timestamp rejection, and list ordering.

Validation:
- go test ./internal/infra/postgres -run 'TestPingHistoryRepository' -count=1: passed
- make test: passed
```

## Bad Report Example
```text
Tests should pass. I added coverage.
```

Why it is bad: it gives no command evidence and claims coverage without naming the proof.

## Assertion Patterns For Validation
- A focused command proves the exact changed test names.
- A package command proves compile and interaction with nearby tests.
- A broader make target proves repository quality expectations.
- A race command proves executed concurrency paths; remember the race detector only reports races in code paths that ran.
- A fuzz command proves fuzz exploration for a bounded time; plain `go test` proves seed corpus regressions.

## Deterministic Coordination Patterns
- Re-run with `-count=1` after changing tests so cached success does not masquerade as validation.
- When a concurrency test is meant to flush scheduling bugs, use repeated `-count=N` only after the test has deterministic coordination and a bounded timeout.
- If a command cannot run because Docker, cgo, a race-supported platform, or another dependency is missing, report the blocker and the narrower evidence you did obtain.
- If validation fails, report the failing command and the key failure line; do not claim completion.

## Repository-Local Cues
- `docs/build-test-and-development-commands.md` lists `make check`, `make test`, `make vet`, `make test-race`, `make test-integration`, and `make check-full`.
- Integration tests in `test/` use `//go:build integration` and Docker/testcontainers.
- The repo uses local Go from `go.mod` and make targets as the default validation interface.

## Exa Source Links
- [Go command documentation](https://pkg.go.dev/cmd/go)
- [Go testing package](https://pkg.go.dev/testing)
- [Go fuzzing documentation](https://go.dev/doc/fuzz/)
- [Go race detector article](https://go.dev/doc/articles/race_detector.html)
