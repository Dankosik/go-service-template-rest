# Naming And Intent Exposure

Behavior Change Thesis: When loaded for renaming or vocabulary drift, this file makes the model require names to expose policy, role, phase, or ownership instead of likely mistake of treating naming as taste or blindly preferring shorter names.

## When To Load
Load this when a diff changes helper names, receiver names, package-facing names, booleans, modes, exported identifiers, comments, or feature vocabulary in a way that affects reviewability.

Do not load this for ordinary local names unless the rename changes branch-misread risk. If the issue is a boolean cluster or mode API, prefer `predicate-condition-and-mode-clarity.md`.

## Decision Rubric
- Finding-worthy: a name hides lifecycle state, policy, role, phase, ownership, or call-site meaning that a future change depends on.
- Finding-worthy: feature vocabulary drifts inside one package or workflow and makes synonyms look like new behavior.
- Not a finding: idiomatic short names such as `ctx`, `err`, `w`, `r`, `tx`, or a consistent one-letter receiver in small scope.
- Not a finding: a local short name is clear from type and nearby use.
- Comments should explain constraints, invariants, or surprising choices; do not ask for comments that only narrate code.

## Imitate
Finding shape to copy when a helper name hides a policy distinction:

```text
[medium] [go-language-simplifier-review] internal/app/accounts/lifecycle.go:63
Issue: `processAccount` now handles both deactivation and deletion policy, but the name exposes neither lifecycle state.
Impact: Future callers can reuse it without noticing that one path preserves audit history while the other removes credentials.
Suggested fix: Split or rename the helper around the lifecycle policy it owns, such as `deactivateAccount` and `deleteAccountCredentials`.
Reference: references/naming-and-intent-exposure.md
```

Copy the move: connect the generic name to a specific wrong reuse risk.

Finding shape to copy when vocabulary drift makes cleanup risky:

```text
[low] [go-language-simplifier-review] internal/app/reports/archive.go:28
Issue: The new cleanup renames `archived` to `disabled` only in one predicate while the rest of the package still uses archive vocabulary.
Impact: A reader has to decide whether "disabled" is a new state or a synonym, which raises future branch-misread risk.
Suggested fix: Keep the existing archive vocabulary, or rename the whole local policy only if the behavior actually changed and the design approves it.
Reference: references/naming-and-intent-exposure.md
```

Copy the move: explain the semantic ambiguity created by vocabulary drift.

## Reject
Reject names that reduce characters but hide meaning:

```go
func process(data Account, flag bool) error {
	if flag {
		return deleteData(data)
	}
	return updateData(data)
}
```

Prefer names that expose the lifecycle decision:

```go
func applyAccountLifecycle(account Account, removeCredentials bool) error {
	if removeCredentials {
		return deleteAccountCredentials(account)
	}
	return deactivateAccount(account)
}
```

Reject comments that narrate the next line:

```go
// Set archived to true.
report.Archived = true
```

Keep comments for constraints a reader cannot infer:

```go
// Preserve the original owner so audit exports can still group archived reports.
report.Archived = true
```

## Agent Traps
- Do not block on package stutter, initialism style, or receiver naming unless it affects exported API clarity or future maintenance.
- Do not ask for long names where type and local scope already supply meaning.
- Do not treat a rename as harmless if it collapses domain vocabulary such as archive, disable, deactivate, and delete.
- Do not rewrite comments that already record invariants or surprising constraints.

## Validation Shape
Naming findings are usually validated by reviewer inspection plus compile/test after rename. When vocabulary drift may signal behavior change, ask for proof that public docs, tests, and domain terms remain aligned or escalate to domain/design review.
