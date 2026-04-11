# False Simplification Patterns

Behavior Change Thesis: When loaded for a broad cleanup or DRY claim with no narrower dominant symptom, this file makes the model challenge whether the diff lowers reader state instead of likely mistake of treating shorter code as simpler code.

## When To Load
Load this as a challenge/smell triage reference when a cleanup spans several simplification axes, or when the prompt only says "simplify", "dedupe", "readability", or "cleanup" and no narrower reference clearly owns the risk.

Do not load this by default when a narrower reference already matches the primary symptom, such as helper extraction, error mapping, control-flow ordering, predicate modes, test proof shape, naming, source-of-truth drift, or Go-semantic stop signs.

## Decision Rubric
- Finding-worthy: the cleanup reduces visible structure while making readers remember hidden branch state, hidden modes, side-effect order, ownership, cleanup, error identity, or caller-visible behavior.
- Finding-worthy: similar-looking branches were merged even though they intentionally own different statuses, retries, notifications, audit writes, cleanup, or error classes.
- Not a finding: a longer rewrite makes each behavior name, phase, or side effect easier to audit.
- Not a finding: repetition remains local because each repeated branch carries a distinct contract.
- Smallest safe fix: usually restore the semantic boundary, or extract only the stable presentation/policy part after each branch has chosen its outcome.

## Imitate
Finding shape to copy when a dedupe helper merges failure semantics:

```text
[high] [go-language-simplifier-review] internal/app/orders/complete.go:88
Issue: The new `finishOrderFailure` helper merges validation, inventory conflict, and payment retry failures behind one status and notification path.
Impact: Reviewers now have to inspect helper inputs to recover which branch owns customer-visible status and retry behavior, and future changes can silently route a conflict like a validation error.
Suggested fix: Keep the repeated response shape local, or extract only the shared formatting after each branch has selected its stable failure class.
Reference: references/false-simplification-patterns.md
```

Copy the move: name the collapsed semantic classes, explain the concrete future misroute, and suggest a smaller extraction boundary.

Finding shape to copy when a loop hides operation order:

```text
[medium] [go-language-simplifier-review] internal/infra/http/export.go:57
Issue: The new `runExportStep` loop hides which steps have already written the audit event before returning an error.
Impact: The shorter loop makes branch order less visible, so adding a new step can accidentally emit audit state before the export is durable.
Suggested fix: Keep the step order explicit or name each step result so the audit boundary remains readable at the call site.
Reference: references/false-simplification-patterns.md
```

Copy the move: make "why order matters" the defect, not "loops are bad."

## Reject
Reject this kind of finding:

```text
Issue: This code is longer than necessary and should use a helper.
```

It is taste-only unless it identifies the policy a helper would own and the merge-risk of leaving it local.

Reject this refactor as "simpler":

```go
func finish(w http.ResponseWriter, err error, status int, retry bool) {
	if retry {
		w.Header().Set("Retry-After", "1")
	}
	http.Error(w, err.Error(), status)
}
```

The flags make the caller decode failure policy. Prefer names that expose stable outcomes:

```go
func writeRetryableConflict(w http.ResponseWriter, err error) {
	w.Header().Set("Retry-After", "1")
	writeProblem(w, http.StatusConflict, err)
}

func writeValidationProblem(w http.ResponseWriter, err error) {
	writeProblem(w, http.StatusBadRequest, err)
}
```

Reject this refactor when order is the contract:

```go
for _, step := range []func(context.Context) error{s.reserve, s.charge, s.audit} {
	if err := step(ctx); err != nil {
		return err
	}
}
```

Prefer explicit phases when future edits must see the sequence:

```go
if err := s.reserve(ctx); err != nil {
	return fmt.Errorf("reserve order: %w", err)
}
if err := s.charge(ctx); err != nil {
	return fmt.Errorf("charge order: %w", err)
}
return s.audit(ctx)
```

## Agent Traps
- Do not use this file as a generic checklist after a narrower reference already explains the defect.
- Do not ask for extraction just because two branches have matching shape.
- Do not praise a guard-clause or loop cleanup until cleanup order, side-effect order, and which error wins remain obvious.
- Do not call protective code ceremony when it preserves ownership, lifetime, or caller-visible contracts.

## Validation Shape
Ask for focused proof only when the finding depends on subtle behavior preservation: branch-specific status/error mapping, side-effect order, cleanup precedence, or inspectable error identity. Green broad tests are not enough if they never distinguish the collapsed branches.
