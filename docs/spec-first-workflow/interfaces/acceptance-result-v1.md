# Acceptance Result V1

Every Acceptance-Unit Lead returns:

```text
unit: <T<ID> or acceptance-unit identity>
verdict: Accepted | Blocked
candidate: <tree, commit, patch, or bounded diff identity>
evidence: <claim-matched focused proof; integrated proof when already available>
review: <fixed candidate identity and verdict>
provides: <stable accepted output>
invalidated_receipts: <previous evidence invalidated by repair; omit when none>
next_owner: <exact owner and reopen condition; Blocked only>
```

The result is the Lead-owned acceptance decision. During orchestrated
execution it is input to the Orchestrator, not a canonical ledger write.
When no Orchestrator exists, a root-local Lead may record its own fixed
unit result.
