# Domain Language And Meaning Drift

## Behavior Change Thesis
When loaded for symptom "a rename or vocabulary change touches business meaning", this file makes the model distinguish behavior-changing semantic drift from pure naming taste instead of likely mistake "treat domain-term changes as cosmetic or over-report ordinary readability issues."

## When To Load
Load this when a review touches domain states, lifecycle terms, event names, domain errors, eligibility vocabulary, ownership terms, obligation names, amount meanings, timestamps with business meaning, or caller-facing comments that explain a business rule.

## Decision Rubric
- Report a finding only when changed vocabulary collapses, broadens, narrows, or swaps a locally meaningful business concept.
- Ask what a caller, audit reader, support workflow, or downstream consumer would now believe differently.
- Preserve established vocabulary when it distinguishes business outcomes, such as `cancelled` vs `expired`, `owner` vs `creator`, `authorized` vs `captured`, or `available` vs `reserved`.
- Hand off readability-only naming issues to `go-language-simplifier-review` and exported Go API naming concerns to `go-idiomatic-review`.

## Imitate
```text
[medium] [go-domain-invariant-review] internal/billing/credit.go:88
Issue:
The diff renames `ReservedCredit` to `AvailableCredit` in the allocation check, but local billing tests treat reserved credit as already committed and unavailable for new purchases.
Impact:
Callers can read the branch as allowing reallocation of committed credit, and a future cleanup can preserve the new name while accidentally double-spending the same reserved amount.
Suggested fix:
Keep the reserved-credit vocabulary in the allocation guard, or update the approved billing rule and tests if the product now intentionally treats reserved credit as available.
Reference:
Local billing credit tests and allocation rule names.
```

Copy the shape: changed term, local distinction, plausible business misread, smallest vocabulary or contract fix.

## Reject
```text
[low] internal/billing/credit.go:88
`ReservedCredit` would be a clearer name than `AvailableCredit`.
```

Failure: this is taste-only unless it proves a local rule uses the terms differently.

## Agent Traps
- Do not infer a business rule from a single renamed variable when surrounding code and tests treat the terms as synonyms.
- Do not use this reference for ordinary helper naming, initialisms, or receiver names.
- Do not accept "just a rename" when domain errors, event names, audit labels, or state names are part of the accepted behavior.
- Do not require a terminology redesign in review; restore local meaning or escalate if the product language is genuinely changing.
