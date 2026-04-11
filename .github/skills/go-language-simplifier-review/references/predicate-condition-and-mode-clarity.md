# Predicate, Condition, And Mode Clarity

Behavior Change Thesis: When loaded for compound predicates, boolean clusters, or raw modes, this file makes the model preserve call-site decision clarity instead of likely mistake of accepting shorter conditions or extracted predicates that hide policy.

## When To Load
Load this when a diff adds or rewrites compound conditions, negative predicates, boolean clusters, same-typed positional arguments, string modes, raw status codes, option decoding, or extracted predicate helpers.

Use this when the review question is "can a reader tell which decision is being made here?" If the issue is helper economics beyond the predicate itself, use `helper-extraction-economics.md`; if the issue is domain state semantics, hand off to domain review.

## Decision Rubric
- Flag predicates that force mental De Morgan expansion or require the reader to inspect a helper to learn the policy terms.
- Flag boolean clusters, same-typed positional arguments, raw strings, raw status codes, and option bags when they encode modes that can form invalid combinations.
- Prefer names that reveal the policy decision, not just mechanical inputs.
- Keep a one-use condition inline when named subexpressions make the decision clearer than a generic helper.
- Do not reject every boolean. A call like `SetEnabled(true)` can read clearly when the method name supplies the predicate.

## Imitate
Finding shape to copy when a predicate helper hides the actual decision:

```text
[medium] [go-language-simplifier-review] internal/app/reports/filter.go:47
Issue: `shouldInclude` hides tenant visibility, archive state, and admin override policy behind one generic predicate name.
Impact: Callers cannot tell which report visibility decision is being made without opening the helper, so future changes can reuse it for the wrong policy.
Suggested fix: Rename and narrow the predicate to the policy it answers, such as `canIncludeArchivedReport`, or keep the condition local with named subexpressions.
Reference: references/predicate-condition-and-mode-clarity.md
```

Copy the move: identify the policy terms hidden behind the generic predicate name.

Finding shape to copy when an API exposes hidden modes:

```text
[high] [go-language-simplifier-review] internal/app/exports/service.go:29
Issue: `RunExport(ctx, id, true, false, "retry")` replaced a small options type with same-typed booleans and a raw mode string.
Impact: The call site is shorter but less safe to read, and invalid mode combinations can now reach runtime instead of being named by the API shape.
Suggested fix: Restore named options or split the modes into policy-named entry points; treat exported API changes as design-sensitive.
Reference: references/predicate-condition-and-mode-clarity.md
```

Copy the move: state the invalid-combination risk and whether the boundary needs design escalation.

## Reject
Reject a shorter condition when it hides the policy:

```go
if !user.Disabled && (user.Admin || !report.Archived) && mode != "private" {
	return true
}
```

Prefer named policy terms the reader must verify:

```go
canViewArchived := user.Admin || !report.Archived
allowedMode := mode != privateReportMode
if !user.Disabled && canViewArchived && allowedMode {
	return true
}
```

Reject positional flags that make callers decode modes:

```go
svc.Send(ctx, id, true, false)
```

Prefer policy-named entry points or a named option shape:

```go
svc.SendPreview(ctx, id)

svc.Send(ctx, id, SendOptions{
	PreviewOnly: true,
})
```

## Agent Traps
- Do not create a helper for a condition that is clearer inline and used once.
- Do not treat a private predicate rename as a public API redesign unless callers or contracts are affected.
- Do not miss negative predicates that become especially risky when combined, such as `!disabled && !archived && !skip`.
- Do not approve `map[string]any` or raw string option bags as "flexible" simplification without a named boundary and validation owner.

## Validation Shape
Ask for targeted tests when mode combinations, operator precedence, or option decoding changed. The proof should include at least one case for each behavior class, not only the happy path.
