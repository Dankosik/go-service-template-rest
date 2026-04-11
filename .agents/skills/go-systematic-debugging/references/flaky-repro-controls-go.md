# Flaky Reproduction Controls For Go

## Behavior Change Thesis
When loaded for intermittent Go test failures, this file makes the model isolate repetition, order, race, CPU, and environment variables instead of mixing knobs or claiming a flake is fixed from one lucky pass.

## When To Load
Load when a Go test fails only under repetition, CI, `-race`, `-shuffle`, a specific CPU count, slower machines, environment differences, or wider package scope.

## Decision Rubric
- Change one diagnostic variable at a time until the failure class is named.
- Use narrow single-test repetition for local lifecycle, timing, and shared-state suspicion.
- Use wider `-run` or package scope only when order dependence or leaked package state is the hypothesis.
- Treat `-shuffle` as meaningful only when multiple tests or subtests remain in scope; replay the failing seed before editing.
- Keep `-race`, `-cpu`, env, and shuffle experiments separate unless a later hypothesis requires a combined stress shape.
- Record frequency as data, for example `7/100`, not as "sometimes".

## Imitate

```bash
go test ./internal/orders -run '^TestCheckout$' -count=100 -v
go test ./internal/orders -run '^TestCheckout$' -race -count=50 -v
go test ./internal/orders -run '^TestCheckout$' -cpu=1,4 -count=50 -v
go test ./internal/orders -run '^(TestCheckout|TestCacheRefresh)$' -shuffle=on -count=50 -v
go test ./internal/orders -run '^(TestCheckout|TestCacheRefresh)$' -shuffle=1700000000000000000 -count=1 -v
```

Copy the separation: each command tests a different failure class, and the failing shuffle seed gets replayed as its own reproducer.

## Reject

```bash
go test ./internal/orders -run '^TestCheckout$' -shuffle=on -race -cpu=1,4 -count=100 -v
```

This mixes order, race, and scheduler variables while `-run` may be too narrow for shuffle to prove package-order leakage.

```text
Fixed: CI passed once after increasing the sleep.
```

This does not name the failure class, preserve the reproducer, or prove the old race/timing/order condition is gone.

## Agent Traps
- Treating a single local pass as evidence against a CI flake.
- Forgetting to capture or replay the `-shuffle` seed.
- Running a package-wide command first and losing the smallest failing scope.
- Reporting only the final failure summary instead of the first distinct failing stack or assertion.
- Disabling the test before proving whether the behavior is test-only or production-relevant.

## Validation Shape
Capture the exact command, working directory, package/test selector, relevant env, `-count`, `-shuffle` seed, `-race`, `-cpu`, timeout, first failure signal, and failure frequency. After the fix, rerun the same defect-shaped command, then add only the broader package or CI-shaped smoke that increases confidence for the named failure class.
