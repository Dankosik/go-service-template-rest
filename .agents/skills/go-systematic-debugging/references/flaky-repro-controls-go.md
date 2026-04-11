# Flaky Reproduction Controls For Go

## When To Load
Load this reference when a Go test fails only under repetition, CI, `-race`, `-shuffle`, a specific CPU count, slower machines, different environment variables, or wider package scope.

Use it to turn "sometimes fails" into a measured failure class before changing code.

## Commands
Start with one variable at a time:

```bash
go test ./path/to/pkg -run '^TestName$' -count=100 -v
go test ./path/to/pkg -run '^TestName$' -race -count=50 -v
go test ./path/to/pkg -run '^TestName$' -cpu=1,4 -count=50 -v
go test ./path/to/pkg -run '^(TestA|TestB|TestC)$' -shuffle=on -count=50 -v
go test ./path/to/pkg -shuffle=on -count=20 -v
go test ./path/to/pkg -run '^TestName$' -count=1 -json
```

When `-shuffle=on` exposes a failing seed, replay it:

```bash
go test ./path/to/pkg -shuffle=123456789 -count=1 -v
```

Use a wider `-run` or full package scope when checking package-order leakage. Use a narrow single-test command for lifecycle, local race, or scheduler-sensitive proof.

## Evidence To Capture
- exact command, package, test name, and working directory
- `-count`, `-shuffle` value or seed, `-race`, `-cpu`, timeout, and relevant env
- first distinct failure symptom and stack, not every repeated duplicate
- failure frequency, for example `7/100`
- whether a wider scope is required for the failure to appear
- cleanup, temp path, port, env var, clock, random seed, and global state conditions

## Common Failure Classes
- order dependence from shared package state, global registries, caches, env vars, temp paths, or leaked ports
- race exposed by `-race`, repeated execution, or altered `GOMAXPROCS`
- goroutine lifecycle leak from workers that outlive the test or miss shutdown
- sleep-based readiness where CI speed changes the result
- real network, DB, cache, or wall-clock dependency that a unit test treated as stable
- `t.Parallel` overlap with mutable fixtures or cleanup

## Bad Debugging Moves
- inflating `time.Sleep` without proving readiness semantics
- pairing `-shuffle` with an over-narrow `-run` and claiming package-order coverage
- mixing `-shuffle`, `-race`, `-cpu`, and env changes in one first experiment
- skipping the failing seed, command, or failure frequency in the report
- "fixing" by disabling the test without proving whether production behavior is also racy

## Good Debugging Moves
- separate order, race, CPU, and environment experiments so each result falsifies one hypothesis
- replay the `-shuffle` seed before editing
- isolate state with `t.Setenv`, `t.TempDir`, explicit fixture ownership, and `t.Cleanup`
- replace readiness sleeps with condition-based waits, explicit channels, fake clocks, or deterministic hooks
- keep the narrow reproducer for local proof and the wider reproducer for package-order proof

## Example Debugging Flow
For a CI-only flake:

```bash
go test ./internal/orders -run '^TestCheckout$' -count=100 -v
go test ./internal/orders -run '^TestCheckout$' -race -count=50 -v
go test ./internal/orders -run '^(TestCheckout|TestCacheRefresh)$' -shuffle=on -count=50 -v
go test ./internal/orders -run '^(TestCheckout|TestCacheRefresh)$' -shuffle=1700000000000000000 -count=1 -v
```

Interpretation:
- narrow `-count` fails: local lifecycle, timing, or shared state inside the test
- only `-race` fails: unsynchronized shared access on an executed path
- only wider `-shuffle` fails: order or cleanup leakage
- only `-cpu` varies: scheduler sensitivity or accidental parallelism

## Source Links
- [testing package](https://pkg.go.dev/testing)
- [go command test packages](https://pkg.go.dev/cmd/go#hdr-Test_packages)
- [go command testing flags](https://pkg.go.dev/cmd/go#hdr-Testing_flags)
- [Go data race detector](https://go.dev/doc/articles/race_detector)
