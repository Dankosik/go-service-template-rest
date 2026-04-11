# Naming And Intent Exposure

## When To Load
Load this when a diff changes helper names, receiver names, package-facing names, booleans, modes, exported identifiers, comments, or feature vocabulary in a way that affects reviewability.

This file is not for taste-only renaming. Use it when names hide role, phase, ownership, policy, or call-site meaning.

## Review Lens
- Good names reduce the context a reader must load before changing code.
- Prefer names that reveal role, phase, or policy over generic mechanism words such as `data`, `process`, `result`, `do`, `handle`, or `helper`.
- Keep vocabulary stable inside one feature area. Drift between `archive`, `disable`, `deactivate`, and `delete` can hide behavior differences.
- Boolean names should read clearly at the call site, usually as facts or options.

## Real Finding Examples
Finding example: a helper name hides a policy distinction.

```text
[medium] [go-language-simplifier-review] internal/app/accounts/lifecycle.go:63
Issue: `processAccount` now handles both deactivation and deletion policy, but the name exposes neither lifecycle state.
Impact: Future callers can reuse it without noticing that one path preserves audit history while the other removes credentials.
Suggested fix: Split or rename the helper around the lifecycle policy it owns, such as `deactivateAccount` and `deleteAccountCredentials`.
Reference: references/naming-and-intent-exposure.md
```

Finding example: vocabulary drift makes a cleanup risky.

```text
[low] [go-language-simplifier-review] internal/app/reports/archive.go:28
Issue: The new cleanup renames `archived` to `disabled` only in one predicate while the rest of the package still uses archive vocabulary.
Impact: A reader has to decide whether "disabled" is a new state or a synonym, which raises future branch-misread risk.
Suggested fix: Keep the existing archive vocabulary, or rename the whole local policy only if the behavior actually changed and the design approves it.
Reference: references/naming-and-intent-exposure.md
```

## Non-Findings To Avoid
- Do not flag idiomatic short names like `ctx`, `err`, `w`, `r`, `tx`, or a consistent one-letter receiver when scope is small.
- Do not demand long names for locals whose meaning is obvious from type and nearby use.
- Do not block on package stutter or initialism style unless it affects exported API clarity or future maintenance.
- Do not rewrite comments that already explain constraints, invariants, or why a surprising step exists.

## Bad And Good Simplifications
Bad: names reduce characters but hide meaning.

```go
func process(data Account, flag bool) error {
	if flag {
		return deleteData(data)
	}
	return updateData(data)
}
```

Good: names expose the lifecycle decision.

```go
func applyAccountLifecycle(account Account, removeCredentials bool) error {
	if removeCredentials {
		return deleteAccountCredentials(account)
	}
	return deactivateAccount(account)
}
```

Bad: a comment narrates the code.

```go
// Set archived to true.
report.Archived = true
```

Good: keep comments for non-obvious constraints.

```go
// Preserve the original owner so audit exports can still group archived reports.
report.Archived = true
```

## Escalation Guidance
- Escalate to `go-idiomatic-review` when naming affects exported docs, package names, initialisms, receiver choices, or public Go API shape.
- Escalate to `go-design-review` when vocabulary drift reflects unclear ownership or a cross-package concept split.
- Escalate to `go-domain-invariant-review` when a rename may collapse distinct domain states.
- Keep purely local taste suggestions out of findings unless they reduce real branch-misread or maintenance risk.

## Source Anchors
- [Effective Go](https://go.dev/doc/effective_go): names are important in Go and affect exported visibility.
- [Go Code Review Comments](https://go.dev/wiki/CodeReviewComments): package names, receiver names, variable names, initialisms, and doc comments.
- [Package names](https://go.dev/blog/package-names): names provide context and help maintain package focus.
- Repository pattern: `go-idiomatic-review/SKILL.md` naming and exported-surface guidance.
