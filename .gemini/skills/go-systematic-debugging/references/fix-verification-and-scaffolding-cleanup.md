# Fix Verification And Scaffolding Cleanup

## Behavior Change Thesis
When loaded before a completion claim, this file makes the model match RED/GREEN proof to the original defect and remove temporary diagnostics instead of overclaiming from a narrow pass.

## When To Load
Load after the likely root cause is known and before reporting success. Use it for RED/GREEN proof, regression command selection, temporary diagnostic cleanup, and residual-risk wording.

## Decision Rubric
- Capture RED before editing when feasible; if not feasible, say why.
- Rerun the exact reproducer after the fix before claiming the original bug is fixed.
- Match extra proof to the defect class: repetition for flakes, `-race` for races, dump/profile comparison for leaks, build command for build failures.
- Run broader package or repo commands only when they support the claim being made.
- Remove temporary debug logs, sleeps, panics, pprof endpoints, env toggles, dumps, profiles, and traces unless the task explicitly asks to preserve evidence.
- State residual risk directly when proof is narrower than the requested claim.

## Imitate

```bash
go test ./internal/orders -run '^TestCheckout$' -count=100 -v
go test ./internal/orders -run '^TestCheckout$' -race -count=50 -v
git diff --check
rg -n 'DEBUG|TEMP|TODO\(|fmt\.Print|spew|panic\(|time\.Sleep|t\.Skip' .agents internal cmd
```

Copy the shape: the verification commands match the flake/race failure mode, and cleanup checks look for temporary scaffolding before reporting success.

## Reject

```text
Verified: go test ./internal/orders -run '^TestCheckout$' passed once.
```

This is too weak for a flake fixed under repetition or `-race`.

```text
All good: go test ./... passed, but the exact failing seed was not rerun.
```

This broad pass does not prove the known reproducer is fixed.

## Agent Traps
- Changing the reproducer after the fix and claiming the original bug is gone.
- Deleting a failing test instead of proving why it was invalid.
- Leaving temporary diagnostics in code or source-controlled artifacts.
- Claiming repository-wide safety from a narrow package command.
- Omitting the RED gap when the old failure could not be reproduced.

## Validation Shape

| Defect class | Minimum proof | Extra proof when warranted |
|---|---|---|
| deterministic panic | exact failing test now passes | package tests |
| race | `-race` repro no longer reports | repeated `-race` and package scope |
| flake | failing seed or repeated command now passes | wider package `-shuffle` |
| hang | prior hang command completes | goroutine/profile no longer shows blocked owner |
| timeout | budget evidence no longer expires at same boundary | trace or stats confirm wait source is gone |
| build failure | same build command passes | `go test -run '^$'` for test compile |

Always report exact command, working directory or package scope, key outcome, cleanup result, and residual risk if proof is incomplete.
