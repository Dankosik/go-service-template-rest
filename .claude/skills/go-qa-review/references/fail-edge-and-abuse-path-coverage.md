# Fail, Edge, And Abuse Path Coverage

## Behavior Change Thesis
When loaded for changes touching failures, boundaries, malformed input, or abuse-path obligations, this file makes the model ask for the smallest representative negative proof instead of demanding exhaustive matrices, fuzzing everything, or taking over threat modeling.

## When To Load
Load this when changed behavior touches failure classes, invalid input, boundary values, malformed payloads, retries, conflict handling, parser behavior, fuzz/regression seeds, or abuse paths that are already in scope for the diff.

## Decision Rubric
- Flag a missing negative only when the changed branch, boundary, or failure class can regress while current tests pass.
- Ask for one representative boundary or failure class unless the code intentionally distinguishes multiple classes.
- Prefer a regression seed for a known fuzz failure; prefer table tests for simple enumerated validation.
- Do not demand fuzzing when the contract is clearer as deterministic examples.
- Hand off to security, reliability, domain, or DB/cache review when the hard question is what the failure semantics should be.

## Imitate

```text
[high] [go-qa-review] internal/config/config_test.go:1040
Issue:
The validator test covers accepted duration values but not the new lower-bound rejection path. A zero or negative duration can still be accepted without any test failing.
Impact:
An invalid production timeout can slip through config validation and turn into an unbounded wait at runtime.
Suggested fix:
Add one table case for the rejected boundary, asserting the stable validation error class rather than incidental error text.
Reference:
Validate with `go test ./internal/config -run '^TestValidate/.+duration' -count=1`.
```

Copy this shape: name the exact boundary, false pass, and deterministic negative case.

```text
[medium] [go-qa-review] internal/infra/http/router_test.go:188
Issue:
The body-size test covers oversized JSON but not the new malformed chunked-body path. That parser path now returns the same problem type and can drift without a regression seed or named malformed-input case.
Impact:
Malformed request handling could stop returning the contract error shape while oversized-body tests still pass.
Suggested fix:
Add one malformed-body case using the existing request helper, asserting status and problem type. Use fuzz only if this path was introduced by a fuzz-found crash or parser hardening work.
Reference:
Validate with `go test ./internal/infra/http -run '^TestRouterRejectsMalformedBody$' -count=1`.
```

Copy this shape: keep fuzz optional and tie it to a parser/fuzz-origin signal.

## Reject

```text
[medium] [go-qa-review] internal/config/config_test.go:1040
Issue:
Need more edge cases and fuzzing.
Impact:
There could be bugs.
Suggested fix:
Add fuzz tests for everything.
Reference:
Run the fuzz tests.
```

Reject this because it asks for volume and tooling without naming the changed failure class.

## Agent Traps
- "Edge cases" is too vague; use the changed boundary name.
- Fuzzing is not a replacement for a named regression expectation.
- Abuse-path proof belongs here only after the abuse class is already accepted; threat depth belongs to security review.
- Integration tests are not automatically stronger for pure validation logic.

## Validation Shape
Use the narrow package and named negative scenario when possible. Use `go test <package> -run '^$' -fuzz=<target> -fuzztime=<duration>` or repo fuzz smoke only when fuzz targets or fuzz-suitable parser/input hardening are actually in scope.
