# Fix Verification And Scaffolding Cleanup

## When To Load
Load this reference after the likely root cause is known and before reporting success. It is for RED/GREEN proof, selecting regression commands, removing temporary diagnostics, and stating residual risk honestly.

Use it to avoid "works on my narrow command" overclaims.

## Commands
Capture RED before editing when possible:

```bash
go test ./path/to/pkg -run '^TestName$' -count=1 -v
go test ./path/to/pkg -run '^TestName$' -race -count=1 -v
go build ./path/to/pkg
```

Rerun the exact reproducer after the fix:

```bash
go test ./path/to/pkg -run '^TestName$' -count=1 -v
```

Use defect-shaped proof for broader confidence:

```bash
go test ./path/to/pkg -run '^TestName$' -count=100 -v
go test ./path/to/pkg -run '^TestName$' -race -count=50 -v
go test ./path/to/pkg -shuffle=123456789 -count=1 -v
go test ./path/to/pkg -run '^$' -count=1
go test ./path/to/pkg/...
go build ./...
```

Clean short-lived diagnostics:

```bash
git diff --check
rg -n 'DEBUG|TEMP|TODO\(|fmt\.Print|spew|panic\(|time\.Sleep|t\.Skip' .
git diff --name-only
```

Only keep artifacts such as `*.pprof`, `trace.out`, dumps, or captured logs when the task explicitly asks for preserved evidence or the repository already has an evidence directory for that purpose.

## Evidence To Capture
- RED command and failing signal, or why RED could not be captured
- minimal fix scope and first broken invariant fixed
- exact GREEN command that matches the original failure
- additional race, repetition, build, or package-level commands and outcomes
- temporary diagnostics removed or intentionally retained
- residual risk: what was not proven and why

## Bad Debugging Moves
- claiming repository-wide safety after one narrow package command
- changing the reproducer and then claiming the original bug is fixed
- leaving debug logs, sleeps, panics, temporary env toggles, or pprof endpoints behind
- deleting a failing test instead of proving why it was invalid
- keeping generated profiles, dumps, or trace files in source by accident

## Good Debugging Moves
- rerun the exact failing command before and after the fix when feasible
- match verification to defect class: repetition for flakes, `-race` for races, dump/profile comparison for leaks, build command for build failures
- add a regression test at the smallest layer that would have caught the broken invariant
- run a broader smoke only when it adds meaningful confidence
- report scope honestly: "fixed and verified for package X" is better than invented completeness

## Example Verification Matrix

| Defect class | Minimum proof | Extra proof when warranted |
|---|---|---|
| deterministic panic | exact failing test now passes | package tests |
| race | `-race` repro no longer reports | repeated `-race` and package scope |
| flake | failing seed or repeated command now passes | wider package `-shuffle` |
| hang | prior hang command completes | goroutine/profile no longer shows blocked owner |
| timeout | budget evidence no longer expires at same boundary | trace or stats confirm wait source is gone |
| build failure | same build command passes | `go test -run '^$'` for test compile |

## Source Links
- [testing package](https://pkg.go.dev/testing)
- [go command test packages](https://pkg.go.dev/cmd/go#hdr-Test_packages)
- [Go data race detector](https://go.dev/doc/articles/race_detector)
- [Go diagnostics](https://go.dev/doc/diagnostics)
- [cmd/go package](https://pkg.go.dev/cmd/go)
