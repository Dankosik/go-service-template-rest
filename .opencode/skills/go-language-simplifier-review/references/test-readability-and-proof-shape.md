# Test Readability And Proof Shape

Behavior Change Thesis: When loaded for test cleanup, this file makes the model protect the proof's readable setup, trigger, and assertion shape instead of likely mistake of approving giant tables or helpers that shorten tests while hiding what behavior is proven.

## When To Load
Load this when a diff changes tests by adding or removing table tests, subtests, assertion helpers, custom comparison helpers, fixture builders, terse failure messages, or broad "test cleanup."

Use this for simplification review of proof readability. Coverage completeness, nondeterminism, and validation readiness belong to `go-qa-review` unless the readability issue itself creates merge risk.

## Decision Rubric
- A table is clearer when entries share setup, trigger, and assertion shape.
- Split tests when rows require different setup modes, hidden assertion modes, or branch-specific proof.
- Keep helpers when they name stable setup/comparison policy, call `t.Helper()` where appropriate, and report useful diagnostics.
- Flag helpers that hide which branch, invariant, or side effect is being proven.
- Failure messages should identify the function or behavior, relevant input, got value, and wanted value when practical.

## Imitate
Finding shape to copy when a giant table hides different proof shapes:

```text
[medium] [go-language-simplifier-review] internal/app/orders/service_test.go:52
Issue: The new table drives success, validation, conflict, and cancellation cases through `wantErr`, `wantStatus`, and `wantAudit` mode fields.
Impact: Each row now needs custom decoding to understand what behavior is being proven, so adding a case can silently skip the relevant assertion.
Suggested fix: Split the table into success and failure groups, or use subtests with explicit assertion blocks for each behavior class.
Reference: references/test-readability-and-proof-shape.md
```

Copy the move: identify the mixed proof shapes and the assertion that can be skipped.

Finding shape to copy when an assertion helper hides diagnosis:

```text
[low] [go-language-simplifier-review] internal/infra/http/problem_test.go:39
Issue: `assertProblem(t, got, want)` hides the route, status, and body comparison behind one helper whose failure message only says "problem mismatch."
Impact: The test is shorter but a future failure will not identify which response field or input path broke.
Suggested fix: Either compare the stable response struct with a diff at the call site, or make the helper call `t.Helper()` and print the function/input plus `diff -want +got`.
Reference: references/test-readability-and-proof-shape.md
```

Copy the move: connect hidden helper output to slower diagnosis, not personal formatting taste.

## Reject
Reject one table that encodes several unrelated assertion modes:

```go
tests := []struct {
	name      string
	input     Input
	wantErr   bool
	wantCode  int
	wantAudit bool
}{/* success, conflict, canceled, malformed */}
```

Prefer grouping by proof shape:

```go
func TestCreateOrderSuccess(t *testing.T) {
	// success cases assert stored order and audit write
}

func TestCreateOrderRejectsInvalidInput(t *testing.T) {
	// validation cases assert ErrInvalidInput and no audit write
}
```

Reject failure messages that erase diagnosis:

```go
if got != want {
	t.Fatal("bad result")
}
```

Prefer messages that name the behavior and values:

```go
if got != want {
	t.Fatalf("Classify(%q) = %v, want %v", input, got, want)
}
```

## Agent Traps
- Do not reject table-driven tests when every row uses the same setup and assertions.
- Do not require verbose tests when a helper has a policy name and useful diagnostics.
- Do not flag `t.Fatal` for setup failures that prevent the test from continuing.
- Do not turn a readability finding into a demand for more coverage unless hidden assertions create a real proof gap.

## Validation Shape
Validation is the test itself: after the fix, each subtest or helper failure should identify which behavior broke and why. When the diff simplified production behavior too, pair this reference with the production-side reference only if the proof shape and production simplification are independent decision pressures.
