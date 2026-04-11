# False Simplification Patterns

## When To Load
Load this when a diff claims cleanup, deduplication, readability, DRY, or shorter code and the change might have hidden policy, ownership, cleanup, error contracts, or public behavior.

Use this as the first calibration file when you are unsure whether a change is actually simpler. Official Go guidance treats formatting and line count as secondary to clear program construction, names, explicit errors, and readable control flow.

## Review Lens
- Ask what facts the reader must now carry across the function, package, or call chain.
- Treat helper extraction, deduplication, and flattening as neutral until they reduce tracked state or exposed call-site burden.
- Separate structural duplication from semantic duplication. Similar-looking code can still own different behavior.
- Prefer local, behavior-preserving corrections over broad rewrites.

## Real Finding Examples
Finding example: a "dedupe" helper merged distinct branches.

```text
[high] [go-language-simplifier-review] internal/app/orders/complete.go:88
Issue: The new `finishOrderFailure` helper merges validation, inventory conflict, and payment retry failures behind one status and notification path.
Impact: Reviewers now have to inspect the helper inputs to recover which branch owns customer-visible status and retry behavior, and future changes can silently route a conflict like a validation error.
Suggested fix: Keep the repeated response shape local, or extract only the shared formatting after each branch has selected its stable failure class.
Reference: references/false-simplification-patterns.md
```

Finding example: a cleanup made side effects harder to trace.

```text
[medium] [go-language-simplifier-review] internal/infra/http/export.go:57
Issue: The new `runExportStep` loop hides which steps have already written the audit event before returning an error.
Impact: The shorter loop makes branch order less visible, so adding a new step can accidentally emit audit state before the export is durable.
Suggested fix: Keep the step order explicit or name each step result so the audit boundary remains readable at the call site.
Reference: references/false-simplification-patterns.md
```

## Non-Findings To Avoid
- Do not flag a longer rewrite when each line now carries a distinct, named behavior and lowers reader burden.
- Do not demand helper extraction for one-off local logic with no stable second use.
- Do not penalize repetition when each branch must preserve a different error, status, cleanup, or ownership contract.
- Do not call a guard clause false simplification when it preserves side-effect order and removes real nesting.

## Bad And Good Simplifications
Bad: the helper reduces lines but creates hidden modes.

```go
func finish(w http.ResponseWriter, err error, status int, retry bool) {
	if retry {
		w.Header().Set("Retry-After", "1")
	}
	http.Error(w, err.Error(), status)
}
```

Good: keep branch meaning explicit, and extract only stable presentation policy.

```go
func writeRetryableConflict(w http.ResponseWriter, err error) {
	w.Header().Set("Retry-After", "1")
	writeProblem(w, http.StatusConflict, err)
}

func writeValidationProblem(w http.ResponseWriter, err error) {
	writeProblem(w, http.StatusBadRequest, err)
}
```

Bad: the loop hides operation order that was previously the point.

```go
for _, step := range []func(context.Context) error{s.reserve, s.charge, s.audit} {
	if err := step(ctx); err != nil {
		return err
	}
}
```

Good: preserve phase order when the sequence is part of the contract.

```go
if err := s.reserve(ctx); err != nil {
	return fmt.Errorf("reserve order: %w", err)
}
if err := s.charge(ctx); err != nil {
	return fmt.Errorf("charge order: %w", err)
}
return s.audit(ctx)
```

## Escalation Guidance
- Escalate to `go-design-review` when the simplification crosses package ownership, public API shape, or approved design boundaries.
- Escalate to `go-domain-invariant-review` when separate business states or transitions were merged.
- Escalate to `go-idiomatic-review` when the risk depends on Go semantics such as nil behavior, mutable aliasing, method sets, or error wrapping.
- Ask for targeted tests or hand off to `go-qa-review` when the simplification relies on subtle branch or precedence preservation.

## Source Anchors
- [Go Code Review Comments](https://go.dev/wiki/CodeReviewComments): supplement to Effective Go; useful for explicit errors, indentation of error flow, package names, interfaces, and useful test failures.
- [Effective Go](https://go.dev/doc/effective_go): clear Go construction and naming, with the official note that it is not actively updated.
- Repository pattern: `go-design-review/references/accidental-complexity-and-helper-buckets.md`.
- Repository pattern: `go-coder/references/helper-extraction-and-package-ownership.md`.
