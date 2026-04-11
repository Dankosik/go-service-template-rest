# Determinism, Isolation, And Flake Risk

## Behavior Change Thesis
When loaded for tests that rely on timing, scheduler, randomness, or shared-state luck, this file makes the model identify the uncontrolled source and deterministic proof shape instead of reflexively saying "remove the sleep", "run with `-race`", or "increase `-count`".

## When To Load
Load this when tests involve sleeps, timers, randomness, environment variables, process-global state, `t.Parallel()`, goroutine coordination, leak checks, race evidence, or `testing/synctest` suitability.

## Decision Rubric
- Flag nondeterminism only when the uncontrolled source can make the test pass or fail without proving the behavior.
- Distinguish sleep as observation from sleep as synchronization; only the latter is usually merge-risk.
- Prefer channels, hooks, fake clocks, controlled readers/writers, `t.Setenv`, `t.TempDir`, and `t.Cleanup` when they prove the event directly.
- Use `go test -race` for shared-memory race evidence, not for liveness or ordering proof.
- Use high `-count` only as supporting flake evidence after the test controls the risky interleaving.
- Recommend `testing/synctest` only when all goroutines and timers under test can stay inside the test bubble.

## Imitate

```text
[high] [go-qa-review] cmd/service/internal/bootstrap/main_shutdown_test.go:123
Issue:
`TestDrainAndShutdownWaitsForPropagationDelay` uses a real sleep as the only proof that the drain marker is set before shutdown starts. If the goroutine reaches shutdown late or early due to scheduler timing, the test can pass without proving the required ordering.
Impact:
The service can regress to shutting down before readiness propagation while this test remains green on faster local runs.
Suggested fix:
Replace the sleep-only observation with a fake clock or explicit gate that records `StartDrain` before `Shutdown`, then assert the call order. If the code is suitable for `testing/synctest`, keep all goroutines and timers inside the bubble.
Reference:
Validate with `go test ./cmd/service/internal/bootstrap -run '^TestDrainAndShutdownWaitsForPropagationDelay$' -count=100`.
```

Copy this shape: identify the ordering claim, the scheduler hole, and the deterministic replacement.

```text
[high] [go-qa-review] internal/infra/http/server_test.go:211
Issue:
The new parallel subtests mutate the package-level logger while each server is still running. `t.Parallel()` can interleave those mutations, so a failure may report the wrong server state or pass with another case's logger.
Impact:
Shutdown diagnostics and race-sensitive behavior can regress while the test result depends on subtest scheduling.
Suggested fix:
Keep these cases serial or inject a per-server logger fixture so each parallel case owns its process-visible state.
Reference:
Validate with `go test -race ./internal/infra/http -run '^TestServerRunAndShutdown$' -count=50`.
```

Copy this shape: only object to `t.Parallel()` because a shared fixture is mutable and relevant.

## Reject

```text
[medium] [go-qa-review] cmd/service/internal/bootstrap/main_shutdown_test.go:123
Issue:
This test sleeps, so it is flaky.
Impact:
It might fail sometimes.
Suggested fix:
Remove the sleep.
Reference:
Run `go test -race`.
```

Reject this because it treats a symptom as the defect and recommends a command that does not prove ordering.

## Agent Traps
- Some sleeps intentionally wait for external propagation and may be acceptable if the assertion does not depend on precise timing.
- `t.Setenv`, `t.TempDir`, and `t.Cleanup` are usually evidence of isolation, not automatic risk; still flag `t.Setenv` in parallel tests or tests with parallel ancestors.
- Leak checks prove lifecycle cleanup, not ordering or protocol progress.
- A race-free run only covers executed paths and does not prove no goroutine is stuck or that shutdown ordering is correct.
- Do not suggest `synctest` for code that uses real network I/O, external processes, or goroutines outside the controlled bubble.

## Validation Shape
Pair deterministic coordination changes with the narrow test command, then add repetition, `-race`, or leak checks only when they match the risk: `-race` for shared memory, high `-count` for flake confidence, leak checks for goroutine lifecycle.
