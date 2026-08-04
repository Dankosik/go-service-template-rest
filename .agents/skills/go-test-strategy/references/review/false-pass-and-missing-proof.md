# False Pass And Missing Proof

## Behavior Change Thesis
When loaded for a review of changed tests, this file makes the model state the regression that would still pass instead of likely mistake "report that coverage is low, that the assertion is weak, or that the change needs more edge cases and a fuzz target."

## When To Load
Load this when changed behavior may have no named scenario, when an assertion may survive a plausible regression, or when a failure class, boundary, or malformed-input path went unproved.

## Decision Rubric
- A finding names three things: the changed behavior, the missing scenario or weak assertion, and the regression that stays green. Without the third, there is no finding yet.
- Treat coverage percentage, test count, and package-level success as context. They are never the finding.
- Judge an assertion by whether a plausible regression passes through it. Upgrade toward error identity via `errors.Is`/`errors.As`, status, persisted state, side effect, or emitted event.
- Match exact text only where the text is the contract the caller or operator depends on; where phrasing is incidental, assert identity instead.
- Ask for one representative failure class unless the code deliberately distinguishes several, and prefer a regression seed over new fuzzing when a crash is already known.
- Prefer one focused scenario over widening a table when the extra rows would dilute the obligation rather than discriminate.

## Imitate
`internal/health/service_test.go` holds both halves of the distinction. `TestServiceRefreshFail` asserts `errors.Is(err, downErr)`, because the caller's contract is the wrapped identity. `TestServiceRefreshNamesTheFailingProbe` asserts the message text itself, because *which* dependency answered is the operator-facing contract. Same file, opposite oracles, and each is chosen from what would leak if the other were used.

## Reject
- "The route tests need more coverage. Add more tests." It identifies no behavior, no missing scenario, and no regression path, and it substitutes a broad command for proof.
- "The assertion is weak; use testify." Library preference is taste, not a leak. This repository asserts with the standard `testing` package and carries `testify` only as an indirect dependency.
- "Add fuzz tests for everything." Volume and tooling stand in for the changed failure class that a named case would pin down.
- "New code has no unit test" before the unproved behavior has a name.

## Agent Traps
- "Only checks that no error occurred" can be right when success is the entire contract. Find the missing observable before flagging it.
- `t.Helper()` placement is enforced by `thelper` in the mandatory lint run and does not deserve a review finding.
- A public contract change usually needs a contract-visible scenario; a pure helper change usually does not.
- Existing package helpers are fine as long as their names and failure output still identify the scenario and the value that broke.

## Validation Shape
Name the narrowest command that would fail if the missing scenario regressed, usually `go test <package> -run '<Test>/<subtest>' -count=1`. Validation should demonstrate that the strengthened proof rejects the previously possible false pass, not merely that the suite is still green. [validation-commands.md](../decision/validation-commands.md) owns the wider gate map.
