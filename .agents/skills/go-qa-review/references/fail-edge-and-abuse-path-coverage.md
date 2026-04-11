# Fail, Edge, And Abuse Path Coverage Examples

## When To Load
Load this when changed behavior touches failure classes, invalid input, boundary values, malformed payloads, retries, conflict handling, parser behavior, fuzz/regression seeds, or abuse paths that are in scope for the diff.

## Review Lens
Fail-path coverage protects the behavior most likely to be skipped by happy-path tests. The review finding should name the specific failure class or boundary that changed, not ask for exhaustive matrices. Abuse-path depth belongs to security review when threat semantics are the hard part; QA owns whether the accepted obligation has executable proof.

## Bad Finding Example
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

Why it fails: it asks for broad test volume and fuzzing without tying the request to a changed failure mode.

## Good Finding Example
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

## Non-Findings To Avoid
- Do not require every invalid input permutation when one representative boundary proves the changed branch.
- Do not demand fuzzing for simple enumerated validation where table tests are clearer and deterministic.
- Do not treat fuzzing as a replacement for asserting stable contract behavior on known regressions.
- Do not take ownership of threat modeling; hand off to security review when the abuse class itself is uncertain.
- Do not require integration tests for pure validation unless the integration boundary changes the failure semantics.

## Smallest Safe Correction
Prefer the smallest proof of the risky failure class:
- add one negative table row for the boundary that changed;
- add a regression seed for a previously failing fuzz input;
- add malformed or oversized input only when the parser, transport, or abuse surface changed;
- assert stable error identity, status, or problem type instead of fragile text;
- hand off when the missing case depends on security, reliability, domain, or DB semantics rather than QA proof mechanics.

## Validation Command Examples
```bash
go test ./internal/config -run '^TestValidate' -count=1
go test ./internal/infra/http -run '^TestRouterRejectsRequestBodyTooLarge$' -count=1
go test <package> -run '^$' -fuzz=FuzzChangedInput -fuzztime=30s
make test-fuzz-smoke FUZZ_TIME=60s
```

## Source Links From Exa
- [testing package fuzzing docs](https://pkg.go.dev/testing)
- [Go Fuzzing](https://go.dev/doc/fuzz/)
- [Tutorial: Getting started with fuzzing](https://go.dev/doc/tutorial/fuzz.html)

## Repo-Local Convention Links
- `docs/build-test-and-development-commands.md`
- `Makefile`
