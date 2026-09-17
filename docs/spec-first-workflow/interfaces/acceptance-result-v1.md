# Acceptance Result V1

The unit Lead reports code production. Only the delivery owner reports final
acceptance; a ledger task has no acceptance verdict.

## Implementation Result

```text
unit: <T<ID> or fixed implementation scope>
verdict: Implemented | Blocked
candidate: <bounded diff, tree, commit, patch identity, or none if blocked>
provides: <implemented code or contract; unverified>
next_owner: <next implementation owner, or exact missing input and its owner>
```

Return Implemented when in-scope code, tests, and cleanup are written and writers
have joined. [Implementation](../phases/implementation.md#feedback-during-coding)
owns permitted coding feedback; do not attach a per-task proof receipt or
wait for final-validation resources. The result unlocks local dependent coding;
it makes no behavior, acceptance, or release claim. Blocked retains partial
work and names the missing implementation input. Neither verdict starts final
validation while another code task remains unfinished.

## Completion Result

Use only after every planned code task is Implemented and assembled. A
standalone single-task delivery also uses Completion; an individual task inside
a ledger cannot use this branch.

```text
unit: Completion
verdict: Accepted | Blocked
candidate: <fixed assembled candidate>
evidence: <local completion scope, required results, and material optional gaps; keep requested external outcomes distinct>
review: <final review result or not_required under shared Review>
invalidated_receipts: <evidence invalidated by repair; omit when none>
next_owner: <exact remaining action and owner, or none>
```

Accepted requires the ordinary local criterion in
[AGENTS.md](../../../AGENTS.md#validation-budget), explicit task additions,
resolved blocking findings, and any final review selected by
[Review](../shared/review.md). Load the
[Evidence Contract](../shared/evidence-contract.md) for explicit additions,
reuse, claim scope, or unavailable infrastructure. Missing optional
integration or runtime observations do not block local Accepted. Use existing
evidence and next_owner fields to state scope and material gaps; no new status
or proof artifact is needed.

Blocked means a required criterion or an explicitly requested CI, runtime,
release, or external action remains incomplete. Report a completed local portion
separately without relabeling the whole larger request as Accepted. Preserve
genuine external-effect gates. Do not convert an unfinished code subset into
Completion to run checks early.

During orchestrated execution these results are input to the Orchestrator,
which records canonical state without repeating implementation or validation.
A root-local Lead may record its own result. Result IDs and this interface's
filename do not create a per-unit acceptance stage.
