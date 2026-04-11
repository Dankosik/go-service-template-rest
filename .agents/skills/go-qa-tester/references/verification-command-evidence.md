# Verification Command Evidence

## Behavior Change Thesis
When loaded before final handoff or validation selection, this file makes the model report fresh, risk-matched command evidence instead of likely mistake: vague "tests should pass" claims, cached success, blanket `go test ./...` as proof of every risk, or unreported failures.

## When To Load
Load this before final handoff, when selecting validation commands, narrowing failing tests, adding race/fuzz/integration evidence, or wording the validation report.

## Decision Rubric
- Start with the narrowest fresh command that proves the changed test name or package.
- Add package-level validation when helpers, fixtures, generated test assets, or nearby tests could be affected.
- Add broader repository commands when the change touches shared behavior, build/test infrastructure, generated artifacts, or cross-package contracts.
- Prefer repository `make` targets as the normal interface when they exist.
- Use raw `go test` commands for focused diagnosis or fast loops; do not silently let them replace a required repository gate.
- Use `-count=1` when you need fresh execution rather than cached results.
- Use `-run` for named suites or subtests.
- Use `-race` or `make test-race` for concurrency-sensitive changes when supported.
- Use fuzz execution only for fuzz targets; plain `go test` proves seed corpus regressions but not exploration.
- Use integration targets or build tags only when the test depends on external services and the repo already supports that mode.

## Imitate
Test:

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

Validation command shape:

```bash
go test ./internal/worker -run '^TestWorkerStopsOnCancellation$' -count=1
```

Report shape:

```text
Implemented repository tests for query error wrapping, null timestamp rejection, and list ordering.

Validation:
- go test ./internal/infra/postgres -run 'TestPingHistoryRepository' -count=1: passed
- make test: passed
```

Copy the shape: scenario coverage is named by behavior and every command includes its observed result.

## Reject
```go
func TestWorkerStops(t *testing.T) {
	go runWorker(context.Background(), fakeSource{})
	time.Sleep(10 * time.Millisecond)
}
```

Reject because it has no observable assertion, can leak the goroutine, and gives validation commands nothing meaningful to prove.

```text
Tests should pass. I added coverage.
```

Reject because it gives no command evidence and claims coverage without naming the proof.

## Command Examples
```bash
go test ./internal/infra/http -run 'TestRouterHTTPPolicy' -count=1
go test ./internal/infra/postgres -run 'TestPingHistoryRepository' -count=1
go test -race ./cmd/service/internal/bootstrap -run 'Test.*Shutdown' -count=1
go test ./... -count=1
go test ./internal/parser -run '^$' -fuzz=FuzzParse -fuzztime=30s
make test
make test-race
make test-fuzz-smoke
make test-integration
make check
make check-full
```

Adapt package paths and test names to the actual changed surface.

## Agent Traps
- Reporting "passed" from a cached run after editing tests.
- Treating a race-clean run as proof that cancellation or liveness semantics are correct.
- Requiring full CI for every local test change when a focused and package-level proof is enough.
- Skipping integration proof silently because Docker is unavailable.
- Hiding a failed command behind a broad summary.
- Claiming fuzz coverage when only seed corpus execution ran.

## Validation Shape
- Focused command proves the exact changed test names.
- Package command proves compilation and interaction with nearby tests.
- Broader make target proves repository quality expectations.
- Race command proves the executed concurrency paths are race-clean, not that unexecuted paths are safe.
- Fuzz command proves bounded exploration; plain `go test` proves seeded regressions.
- If validation cannot run because Docker, cgo, race support, toolchain, or another dependency is missing, report the blocker and the narrower evidence obtained.
