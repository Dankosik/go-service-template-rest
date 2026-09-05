# Proof Obligations

## Load When
Load this when approved behavior must become proof obligations: which level proves a claim, which scenario rows discriminate, and what each row observes.

## Decide
- Escalate a level only because the lower level cannot observe the boundary. Importance and blast radius raise the proof bar, not the level.
- Unit proves deterministic local logic; contract proves client-visible boundary behavior; integration proves a real source-of-truth boundary; e2e smoke proves composed wiring after smaller tests already own the correctness claims.
- When the level choice is nontrivial, name the rejected level and the evidence gap it would leave. An asserted level and a chosen one read alike until the rejection is named.
- Give every row preconditions, input shape, level, and one observable that changes when the behavior is wrong.
- "Returns an error" is not an observable. Name the error identity, status, persisted state, emitted message, or the side effect whose absence proves the contract.
- Keep the matrix small enough that every row earns its place. Three rows that each discriminate prove more than a complete-looking matrix that cannot say what any row rules out.
- `-race`, repetition, and coverage are instrumentation over a scenario, never a scenario.
- Choose fuzz only for input-heavy logic with a cheap invariant and a seed corpus. Otherwise a named example is the stronger oracle.

## Inspect
`RollbackOnPartialFailure` selects integration because the claim is about durable state; rows are all-steps-succeed, mid-step failure, and cancellation before commit; the observable is that the rows are all present or all absent. The rejected level is unit-with-a-mock-repository, and the gap is decisive: it passes with no transaction at all.

## Reject
- "Add integration coverage for safety" when the changed behavior is observable lower down. It hides which invariant failed and spends the run time that would have paid for the discriminating case.
- "Happy path, invalid input, edge case" with no named data shape. Nobody can turn the row into a test without re-deciding what was meant.
- E2E smoke offered in place of API, data, reliability, or security proof. Smoke proves wiring and cannot localize a contract break.

## Reopen
- A row that cannot name an observable is underspecified behavior, not a future test idea. Route it back through the contract's escalation path rather than inventing the missing policy here.
- Branch coverage of an implementation is not proof of the invariant the branch was written for.

## Prove
Each obligation reads: risk → selected level → observable → reopen condition.
For a nontrivial level choice, also name the rejected level and its evidence
gap. Missing a required part prevents readiness; an obvious level choice needs
no invented alternative.
