# Assertion Strength And Diagnostics

## Behavior Change Thesis
When loaded for a test that exists but can pass while the contract is wrong, this file makes the model request stable observable assertions and useful diagnostics instead of giving library-preference advice or vague "assert more" findings.

## When To Load
Load this when assertions may be too weak, helpers too opaque, error-shape checks missing, or failure output unable to localize the broken behavior.

## Decision Rubric
- Flag weak assertions only when the current assertion can pass through a plausible regression.
- Ask for stable contract checks: status, body shape, state transition, side effect, error identity/type/class, or emitted event.
- Prefer `errors.Is`, `errors.As`, status codes, or structured problem fields over fragile error-string equality when text is incidental.
- Do not require `testify`, `cmp`, or any helper library when standard `testing` assertions are clear.
- Treat a helper as risky only when its call site or failure output hides the scenario or the value that broke.

## Imitate

```text
[high] [go-qa-review] internal/infra/http/openapi_contract_test.go:86
Issue:
The readiness failure test only checks that the handler returns an error status, but the changed runtime contract also requires the body to stay `not ready` and the response to remain machine-classifiable. A handler that returns a generic 500 or an HTML body would still satisfy the current assertion.
Impact:
Deployment admission and monitoring clients could lose the expected readiness signal while the contract test still passes.
Suggested fix:
Assert the exact expected status and stable body or content type in this test, using got/want diagnostics that include the observed response body on failure.
Reference:
Validate with `go test ./internal/infra/http -run '^TestOpenAPIRuntimeContractReadinessUnavailable$' -count=1`.
```

Copy this shape: name the false pass, the contract consequence, and the stable assertion target.

```text
[medium] [go-qa-review] internal/app/health/service_test.go:32
Issue:
The degraded-health case asserts only that an error was returned. It never checks that `errors.Is(err, ErrDependencyUnavailable)` remains true after the new dependency classification path.
Impact:
Callers could lose the retryable error classification while this test still passes on any non-nil error.
Suggested fix:
Assert the stable error identity with `errors.Is` and keep the current message check out unless the message is a documented contract.
Reference:
Validate with `go test ./internal/app/health -run '^TestService/.+degraded' -count=1`.
```

Copy this shape: upgrade a weak error assertion without overfitting to incidental text.

## Reject

```text
[medium] [go-qa-review] internal/app/health/service_test.go:32
Issue:
The assertion is weak.
Impact:
The test is not ideal.
Suggested fix:
Use testify require.
Reference:
Run `go test`.
```

Reject this because it reports style taste, not a regression that can leak.

## Agent Traps
- "Only checks no error" can be acceptable when success itself is the full contract; find the missing observable before flagging it.
- Exact string matching is strong only when the string is stable contract, not private phrasing.
- `t.Helper()` is useful for local helpers, but it is not a substitute for asserting the right behavior.
- Do not ask for internal-field assertions when public behavior already fails on the same regression.

## Validation Shape
Use the package or named test that owns the assertion upgrade. Add `-count=1` when freshness matters. Validation should prove the strengthened assertion fails for the previously possible false pass, not merely that the suite still passes.
