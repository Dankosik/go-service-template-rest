# Scenario Traceability Review

## Behavior Change Thesis
When loaded for a review that risks saying "add more tests", this file makes the model require one named scenario tied to changed behavior and regression leakage instead of treating test count, broad tables, or coverage as proof.

## When To Load
Load this when changed behavior, approved obligations, invariants, or API/data/reliability expectations may not be represented by named test scenarios.

## Decision Rubric
- Flag a gap only when you can name the changed behavior, the missing scenario, and the regression that would still pass.
- Prefer one focused scenario over expanding a broad table when the table would dilute the obligation.
- Treat coverage percentage and package-level success as supporting context, never as the finding.
- Do not require integration coverage when a unit or contract scenario fully proves the changed behavior.
- Do not require every table input to trace to a spec item when a smaller set already proves the changed risk.

## Imitate

```text
[high] [go-qa-review] internal/infra/http/router_test.go:79
Issue:
`TestRouterHTTPPolicy` covers normal method handling but does not trace the new fail-closed CORS preflight rule for unknown origins. The changed policy can now regress without a scenario that sends an untrusted `Origin` and verifies the denied response headers.
Impact:
A future middleware change could reopen cross-origin access while the test suite still passes because only accepted-origin and no-origin cases are exercised.
Suggested fix:
Add one named subtest for an untrusted origin on the affected route, asserting status, absence of allow headers, and the problem response shape if the contract requires it.
Reference:
Validate with `go test ./internal/infra/http -run '^TestRouterHTTPPolicy$/untrusted_origin$' -count=1`.
```

Copy this shape: exact scenario missing, why existing tests do not cover it, smallest local scenario, targeted validation.

```text
[medium] [go-qa-review] internal/config/config_test.go:412
Issue:
The new `TrimSpace` normalization path is only exercised through the accepted default config case. There is no named scenario proving that user-supplied whitespace around an endpoint is normalized before validation.
Impact:
A future validation refactor could reject otherwise valid config or preserve whitespace in the runtime endpoint while the broad default-config test still passes.
Suggested fix:
Add a named table row for a whitespace-padded endpoint and assert the normalized endpoint value after validation.
Reference:
Validate with `go test ./internal/config -run '^TestValidate/.+endpoint' -count=1`.
```

Copy this shape: bounded traceability finding for a behavior that already has nearby setup.

## Reject

```text
[medium] [go-qa-review] internal/infra/http/router_test.go:79
Issue:
The route tests need more coverage.
Impact:
Coverage might go down.
Suggested fix:
Add more tests.
Reference:
Run `go test ./...`.
```

Reject this because it does not identify the behavior, missing scenario, or regression path. It also substitutes a broad command for proof.

## Agent Traps
- "New code has no unit test" is not a finding until you name the unproved behavior.
- A large table can still miss the only scenario that matters.
- A public contract change usually needs a contract-visible scenario; a pure helper change usually does not.
- Existing package helpers are fine when their names and failure output preserve the scenario intent.

## Validation Shape
Prefer the narrowest command that would fail if the missing scenario regressed, usually `go test <package> -run '<test>/<scenario>' -count=1`. Use contract or aggregate commands only when the missing scenario depends on generated/API-visible proof.
