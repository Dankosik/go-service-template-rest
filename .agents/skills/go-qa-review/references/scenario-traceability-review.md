# Scenario Traceability Review Examples

## When To Load
Load this when a review needs to decide whether changed behavior, approved obligations, invariants, or API/data/reliability expectations are actually represented by named test scenarios.

## Review Lens
Traceability is about proof quality, not test volume. A useful scenario names the behavior it protects, exercises the changed path, and asserts the observable outcome that would regress. Line coverage, a large table, or a passing broad package run is not enough if the changed behavior can still leak through untested.

## Bad Finding Example
```text
[medium] [go-qa-review] internal/infra/http/router_test.go:79
Issue:
The route tests need more coverage.
Impact:
Coverage might go down.
Suggested fix:
Add more tests.
Reference:
Run go test.
```

Why it fails: it does not identify the changed behavior, the missing scenario, or how a regression would escape review.

## Good Finding Example
```text
[high] [go-qa-review] internal/infra/http/router_test.go:79
Issue:
`TestRouterHTTPPolicy` covers normal method handling but does not trace the new fail-closed CORS preflight rule for unknown origins. The changed policy can now regress without a scenario that sends an untrusted `Origin` and verifies the denied response headers.
Impact:
A future middleware change could reopen cross-origin access while the test suite still passes because only accepted-origin and no-origin cases are exercised.
Suggested fix:
Add one named subtest for an untrusted origin on the affected route, asserting status, absence of allow headers, and the problem response shape if the contract requires it.
Reference:
Validate with `go test ./internal/infra/http -run 'TestRouterHTTPPolicy/untrusted_origin' -count=1`.
```

## Non-Findings To Avoid
- Do not flag "new code has no unit test" without naming the behavior that lacks proof.
- Do not treat a coverage percentage drop as a QA finding unless it corresponds to a concrete untested behavior.
- Do not require every table input to map to a spec item when the changed risk is already covered by a smaller scenario.
- Do not demand integration tests for behavior fully proven at the unit or contract layer.

## Smallest Safe Correction
Prefer the narrowest scenario that proves the changed obligation:
- add one named `t.Run` case to an existing table when the setup is already correct;
- add a focused contract test when the behavior is API-visible;
- add a regression test for the exact invariant or fail path instead of expanding a broad matrix;
- use the existing package test helpers when they preserve scenario intent instead of hiding it.

## Validation Command Examples
```bash
go test ./internal/infra/http -run '^TestRouterHTTPPolicy$/untrusted_origin$' -count=1
go test ./internal/infra/http -run '^TestOpenAPIRuntimeContract' -count=1
make openapi-check
make test
```

## Source Links From Exa
- [testing package docs](https://pkg.go.dev/testing)
- [cmd/go test flags docs](https://pkg.go.dev/cmd/go/internal/test)

## Repo-Local Convention Links
- `docs/build-test-and-development-commands.md`
- `Makefile`
- `test/README.md`
