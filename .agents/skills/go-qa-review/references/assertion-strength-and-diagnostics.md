# Assertion Strength And Diagnostics Examples

## When To Load
Load this when a review needs to judge whether assertions prove observable behavior, side effects, state transitions, error shape, or failure diagnostics strongly enough for the changed risk.

## Review Lens
Assertions should fail for the right reason and tell the maintainer what broke. A test that checks only `err == nil`, only "no panic", or only that a helper returned something can pass while the changed contract is wrong. Stronger assertions should stay tied to stable outputs; avoid overfitting to private formatting or incidental implementation detail.

## Bad Finding Example
```text
[medium] [go-qa-review] internal/app/health/service_test.go:32
Issue:
The assertion is weak.
Impact:
The test is not ideal.
Suggested fix:
Use testify require.
Reference:
Run go test.
```

Why it fails: it turns the finding into a library preference and never states which observable behavior can regress.

## Good Finding Example
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

## Non-Findings To Avoid
- Do not require `testify` or another assertion package when standard `testing` assertions are clear.
- Do not demand exact string matching for unstable error text when `errors.Is`, `errors.As`, status codes, or structured error fields are the stable contract.
- Do not flag a helper as opaque if the helper name, call site, and failure message preserve the scenario intent.
- Do not require assertions on every internal field when the public behavior is already proven.

## Smallest Safe Correction
Prefer local assertion upgrades:
- replace "no error" with got/want checks for the observable result that matters;
- assert stable error identity or class with `errors.Is`/`errors.As` when available;
- include the actual value, expected value, and scenario name in the failure message;
- call `t.Helper()` in local assertion helpers so diagnostics point to the scenario;
- keep implementation-detail assertions out unless the changed contract is internal by design.

## Validation Command Examples
```bash
go test ./internal/app/health -run '^TestService' -count=1
go test ./internal/infra/http -run '^TestOpenAPIRuntimeContractReadinessUnavailable$' -count=1
go test ./internal/config -run '^TestValidate' -count=1
```

## Source Links From Exa
- [testing package docs](https://pkg.go.dev/testing)
- [testing package examples and subtests](https://go.dev/pkg/testing/?m=old)

## Repo-Local Convention Links
- `internal/infra/http/openapi_contract_test.go`
- `docs/build-test-and-development-commands.md`
