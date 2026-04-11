# Determinism, Isolation, And Flake Risk Examples

## When To Load
Load this when tests involve sleeps, timers, randomness, environment variables, process-global state, `t.Parallel()`, goroutine coordination, leak checks, race evidence, or `testing/synctest`.

## Review Lens
Deterministic tests control the event they are proving. Timing luck, scheduler luck, leaked globals, shared temp paths, and parallel tests that mutate process-wide state weaken review evidence even when the test usually passes. The finding should name the uncontrolled source and the behavior that can escape.

## Bad Finding Example
```text
[medium] [go-qa-review] cmd/service/internal/bootstrap/main_shutdown_test.go:123
Issue:
This test sleeps, so it is flaky.
Impact:
It might fail sometimes.
Suggested fix:
Remove the sleep.
Reference:
Run go test -race.
```

Why it fails: some sleeps are bounded observations, not synchronization defects. The finding must explain what the sleep is trying to coordinate and why it cannot prove that event.

## Good Finding Example
```text
[high] [go-qa-review] cmd/service/internal/bootstrap/main_shutdown_test.go:123
Issue:
`TestDrainAndShutdownWaitsForPropagationDelay` uses a real sleep as the only proof that the drain marker is set before shutdown starts. If the goroutine reaches shutdown late or early due to scheduler timing, the test can pass without proving the required ordering.
Impact:
The service can regress to shutting down before readiness propagation while this test remains green on faster local runs.
Suggested fix:
Replace the sleep-only observation with a fake clock or explicit gate that records `StartDrain` before `Shutdown`, then assert the call order. If the code is suitable for Go 1.26's `testing/synctest`, keep all goroutines and timers inside the bubble.
Reference:
Validate with `go test ./cmd/service/internal/bootstrap -run '^TestDrainAndShutdownWaitsForPropagationDelay$' -count=100`.
```

## Non-Findings To Avoid
- Do not flag `t.Parallel()` when each case owns isolated fixtures and does not mutate process-wide state.
- Do not flag `t.Setenv`, `t.TempDir`, or `t.Cleanup` as global-state risk when they are used according to `testing` constraints.
- Do not treat `go test -race` as proof of liveness, shutdown ordering, or protocol progress.
- Do not recommend `testing/synctest` for tests that depend on real network I/O, external processes, or goroutines outside the test bubble.
- Do not demand `-count=100` as a substitute for deterministic coordination; use it only after the test controls the risky interleaving.

## Smallest Safe Correction
Prefer deterministic isolation:
- use channels or hooks to gate goroutines at the relevant interleaving;
- use fake clocks or `testing/synctest` for time-driven code when the whole test can stay inside the bubble;
- use `t.Setenv`, `t.TempDir`, and `t.Cleanup` for process or filesystem state;
- avoid `t.Parallel()` around `os.Chdir`, global telemetry providers, shared listeners, or package-level mutable state unless the test isolates them;
- run race evidence when shared-memory behavior is touched, and leak checks when goroutine lifecycle is the proof target.

## Validation Command Examples
```bash
go test -race ./cmd/service/internal/bootstrap -run '^TestDrainAndShutdown' -count=50
go test ./internal/infra/http -run '^TestServerRunAndShutdown$' -count=100 -timeout=10s
go test ./internal/infra/http -count=1
make test-race
```

## Source Links From Exa
- [testing package docs](https://pkg.go.dev/testing)
- [Data Race Detector](https://go.dev/doc/articles/race_detector.html)
- [testing/synctest package docs](https://pkg.go.dev/testing/synctest)

## Repo-Local Convention Links
- `internal/infra/http/goleak_test.go`
- `docs/build-test-and-development-commands.md`
- `Makefile`
