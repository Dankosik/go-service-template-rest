# Test Readability And Proof Shape

## When To Load
Load this when a diff changes tests by adding or removing table tests, subtests, assertion helpers, custom comparison helpers, fixture builders, terse failure messages, or broad "test cleanup."

This file is for simplification review of the proof's readability. Coverage completeness belongs to `go-qa-review` unless the readability issue itself creates merge risk.

## Review Lens
- Tests should make setup, trigger, and expected outcome obvious on first read.
- A table is clearer when entries share the same setup and assertion logic; split tests when each row needs mode-specific checks.
- Helper layers are useful for setup or stable comparisons, but harmful when they hide which branch or invariant is being proven.
- Failure messages should identify the function, inputs, got value, and wanted value when practical.

## Real Finding Examples
Finding example: a giant table hides different proof shapes.

```text
[medium] [go-language-simplifier-review] internal/app/orders/service_test.go:52
Issue: The new table drives success, validation, conflict, and cancellation cases through `wantErr`, `wantStatus`, and `wantAudit` mode fields.
Impact: Each row now needs custom decoding to understand what behavior is being proven, so adding a case can silently skip the relevant assertion.
Suggested fix: Split the table into success and failure groups, or use subtests with explicit assertion blocks for each behavior class.
Reference: references/test-readability-and-proof-shape.md
```

Finding example: an assertion helper hides diagnosis.

```text
[low] [go-language-simplifier-review] internal/infra/http/problem_test.go:39
Issue: `assertProblem(t, got, want)` hides the route, status, and body comparison behind one helper whose failure message only says "problem mismatch."
Impact: The test is shorter but a future failure will not identify which response field or input path broke.
Suggested fix: Either compare the stable response struct with a diff at the call site, or make the helper call `t.Helper()` and print the function/input plus `diff -want +got`.
Reference: references/test-readability-and-proof-shape.md
```

## Non-Findings To Avoid
- Do not reject table-driven tests when every row uses the same setup and assertions.
- Do not require verbose tests when a helper has a policy name, calls `t.Helper()`, and produces useful diagnostics.
- Do not flag `t.Fatal` for setup failures that prevent the test from continuing.
- Do not turn test readability into a demand for more coverage unless the changed behavior has a real proof gap.

## Bad And Good Simplifications
Bad: one table encodes several unrelated assertion modes.

```go
tests := []struct {
	name      string
	input     Input
	wantErr   bool
	wantCode  int
	wantAudit bool
}{/* success, conflict, canceled, malformed */}
```

Good: split by proof shape so each assertion reads directly.

```go
func TestCreateOrderSuccess(t *testing.T) {
	// success cases assert stored order and audit write
}

func TestCreateOrderRejectsInvalidInput(t *testing.T) {
	// validation cases assert ErrInvalidInput and no audit write
}
```

Bad: terse failure messages make diagnosis depend on reading the helper.

```go
if got != want {
	t.Fatal("bad result")
}
```

Good: name the function, input, got value, and want value.

```go
if got != want {
	t.Fatalf("Classify(%q) = %v, want %v", input, got, want)
}
```

## Escalation Guidance
- Hand off to `go-qa-review` when the issue is missing scenarios, weak assertion strength, nondeterminism, or validation readiness.
- Hand off to `go-domain-invariant-review` when the test hides the business invariant or transition being proven.
- Hand off to `go-concurrency-review` when test simplification hides synchronization, cancellation, race, or goroutine-lifecycle proof.
- Escalate to planning/spec skills only when the required proof obligation is not defined by existing task artifacts.

## Source Anchors
- [Go Test Comments](https://go.dev/wiki/TestComments): readable subtest names, got-before-want, input identification, helper guidance, and table-driven test boundaries.
- [Go Code Review Comments](https://go.dev/wiki/CodeReviewComments): useful test failures and table-driven test calibration.
- [testing package](https://pkg.go.dev/testing): `T.Helper`, `T.Run`, `T.Fatal`, `T.Error`, and cleanup behavior.
- Repository pattern: `go-qa-review/SKILL.md`.
