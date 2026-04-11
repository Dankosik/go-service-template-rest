# Predicate, Condition, And Mode Clarity

## When To Load
Load this when a diff adds or rewrites compound conditions, negative predicates, boolean clusters, string modes, raw status codes, option decoding, or extracted predicate helpers.

This file is about whether the decision reads clearly at the call site and in the branch, not whether the predicate is shorter.

## Review Lens
- Flag predicates that force mental De Morgan expansion or require hidden mode decoding.
- Treat clusters of booleans, same-typed positional arguments, and raw strings as likely hidden state.
- Prefer names that reveal the policy decision, not only the mechanical inputs.
- Be skeptical of extracted predicates that hide policy terms the reader must still inspect elsewhere.

## Real Finding Examples
Finding example: a predicate helper hides the actual decision.

```text
[medium] [go-language-simplifier-review] internal/app/reports/filter.go:47
Issue: `shouldInclude` hides tenant visibility, archive state, and admin override policy behind one generic predicate name.
Impact: Callers cannot tell which report visibility decision is being made without opening the helper, so future changes can reuse it for the wrong policy.
Suggested fix: Rename and narrow the predicate to the policy it answers, such as `canIncludeArchivedReport`, or keep the condition local with named subexpressions.
Reference: references/predicate-condition-and-mode-clarity.md
```

Finding example: a public signature exposes hidden modes.

```text
[high] [go-language-simplifier-review] internal/app/exports/service.go:29
Issue: `RunExport(ctx, id, true, false, "retry")` replaced a small options type with same-typed booleans and a raw mode string.
Impact: The call site is shorter but less safe to read, and invalid mode combinations can now reach runtime instead of being named by the API shape.
Suggested fix: Restore named options or split the modes into policy-named entry points; treat exported API changes as design-sensitive.
Reference: references/predicate-condition-and-mode-clarity.md
```

## Non-Findings To Avoid
- Do not flag common local checks such as `if err != nil` or one obvious guard.
- Do not require a helper for a condition that is clearer inline and used once.
- Do not reject every boolean. A call like `SetEnabled(true)` can read clearly when the method name supplies the predicate.
- Do not turn a private predicate rename into a public API redesign unless callers or contracts are affected.

## Bad And Good Simplifications
Bad: a shorter condition hides the policy.

```go
if !user.Disabled && (user.Admin || !report.Archived) && mode != "private" {
	return true
}
```

Good: name the policy terms the reader must verify.

```go
canViewArchived := user.Admin || !report.Archived
allowedMode := mode != privateReportMode
if !user.Disabled && canViewArchived && allowedMode {
	return true
}
```

Bad: positional flags make callers decode modes.

```go
svc.Send(ctx, id, true, false)
```

Good: either use policy-named entry points or a named option shape.

```go
svc.SendPreview(ctx, id)

svc.Send(ctx, id, SendOptions{
	PreviewOnly: true,
})
```

## Escalation Guidance
- Escalate to `go-design-review` when mode shape belongs to a public or cross-package boundary.
- Escalate to `api-contract-designer-spec` when raw modes or flags are part of client-visible REST behavior.
- Escalate to `go-domain-invariant-review` when the predicate encodes business eligibility, state transitions, or acceptance rules.
- Escalate to `go-idiomatic-review` when the issue depends on Go exported API, typed constants, zero values, or method-set behavior.

## Source Anchors
- [Go Code Review Comments](https://go.dev/wiki/CodeReviewComments): variable names, named result parameters, line length, interfaces, and in-band errors.
- [Effective Go](https://go.dev/doc/effective_go): names affect visibility and readability in Go.
- [Package names](https://go.dev/blog/package-names): names provide context and help maintainers decide what belongs together.
- Repository pattern: `go-language-simplifier-review/evals/evals.json` includes boolean mode and option-bag review cases.
