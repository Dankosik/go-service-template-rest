# Invariant And Acceptance Traceability

## Behavior Change Thesis
When loaded for symptom "approved invariants or acceptance criteria are not traceable to proof", this file makes the model map each claim to a falsifiable obligation instead of likely mistake "write a broad test suite outline and hope coverage implies correctness."

## When To Load
Load this when domain invariants, acceptance criteria, state transitions, or reopen conditions must become explicit QA obligations before coding.

## Decision Rubric
- Map each critical invariant to owner, violation path, selected proof level, scenario rows, observable, and reopen trigger.
- Map each acceptance criterion to what a client, durable store, emitted message, cache, or state transition can observe.
- Use unit proof for pure local invariants, contract proof for client-visible acceptance, integration proof for durable state, and process/reconciliation proof for cross-service convergence.
- If owner, violation outcome, terminal state, idempotency policy, or external behavior is missing, route back to the owning spec instead of inventing it in QA strategy.
- Keep traceability compact: prove the high-risk claims, do not create a spreadsheet for low-value implementation details.

## Imitate
| Claim | Selected Proof | Required Rows | Reopen Trigger |
| --- | --- | --- | --- |
| `OneActiveExportPerTenant` | Contract plus integration if durable lock/idempotency is storage-backed | first export accepted; duplicate same request equivalent; different duplicate conflicts; concurrent duplicate suppressed | conflict semantics or idempotency retention not approved |
| `NoCrossTenantRead` | Contract or integration | own object; other tenant object; unauthenticated; admin/internal actor if specified | tenant source or concealment policy unresolved |
| `RollbackOnPartialFailure` | Integration | all steps succeed; middle step fails; context cancels before commit; retry after rollback | transaction owner or retry class not approved |
| `AsyncEventuallyTerminal` | Integration or process proof | accepted; retryable failure; non-retryable failure; poison message; replay after restart | terminal states or poison policy missing |

## Reject
- "Acceptance criteria covered by unit tests" without naming which criterion maps to which observable.
- "Test state transitions" without forbidden transition, terminal state, stale version, or concurrency rows when those risks exist.
- "Cross-service invariant covered locally" when the approved behavior depends on replay, dedup, compensation, or reconciliation.
- "Residual risk: edge cases" without naming the missing upstream decision that prevents proof.

## Agent Traps
- QA strategy proves approved behavior; it does not decide product status codes, domain conflict semantics, retry policy, or tenant visibility.
- Do not confuse implementation branch coverage with invariant proof.
- Do not force every acceptance criterion into e2e. Pick the smallest level that can observe the claim.
- Do not hide blocked traceability. A blocked proof obligation is more useful than a fake scenario.

## Validation Shape
Use the shape: claim -> owner/source -> selected proof -> scenario rows -> observable -> rejected weaker proof or reopen trigger.
