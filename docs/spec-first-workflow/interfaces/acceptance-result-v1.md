# Acceptance Result V1

Every Acceptance-Unit Lead returns:

```text
unit: <T<ID> or acceptance-unit identity>
verdict: Accepted | Blocked
candidate: <tree, commit, patch, bounded diff identity, or none>
evidence: <claim-matched focused and required integrated proof; exact gaps for Blocked>
review: <fixed candidate identity and verdict, not_required with reason, or not_run with gap>
provides: <stable accepted output or none>
invalidated_receipts: <previous evidence invalidated by repair; omit when none>
next_owner: <exact owner and reopen condition; Blocked only>
```

Absent candidate or output and not_run review are allowed only for Blocked.
Use not_required only when shared [Review](../shared/review.md)'s Implementation
trigger does not apply; the reason is its brief disposition, not a separate
assessment. Accepted requires Lead self-review, resolution of blocking findings,
any triggered independent review, and every unit-required check to pass on the
candidate, including its Integrated check when specified. Later cross-unit or target-only proof
belongs to its named task or global Completion and does not extend this verdict.

The result is the Lead-owned acceptance decision. During orchestrated
execution it is input to the Orchestrator, not a canonical ledger write.
When no Orchestrator exists, a root-local Lead may record its own fixed
unit result.
