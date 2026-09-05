# Invariant Register Patterns

## Decide
- One row per condition that must stay true. Split a bundled `and` rule when its halves fail differently.
- Each row states: the rule in accepted business terms; the **false case** — the input, sequence, or replay that would violate it; the owner with authority to keep it true; the source of truth; the enforcement point; the violation outcome; and the proof that fails when the rule breaks. A row without a false case is a wish, not an invariant.
- The owner is the actor or policy with authority over the rule, which is usually not the component that first detects the violation. `docs/repo-architecture.md#source-of-truth` records accepted authority; `#domain-vocabulary` records a term whose two readings would change an outcome. Define a term only then.
- Choose the enforcement point by what must still hold under a concurrent writer and a mixed-version deploy. A rule a second transaction can break — uniqueness, terminal state, monotonic sequence, non-negative balance — is a database constraint; application ordering is a convention two callers race. `migrations/000001_postgres_outbox.sql` is the repository's precedent: terminal exclusivity is `outbox_events_terminal_check`, at-most-one-claimable-per-key is a partial unique index, and `outbox_ordering_heads.last_sequence` is retained past cleanup so the rejection outlives the rows it was derived from.
- Reserve the register for rules whose violation changes correctness, authority, or accepted behavior. Ordinary field validation stays out.
- `invariant-violation-semantics.md` owns the outcome vocabulary. `go-data-architecture` owns the constraint's shape and migration safety; this file owns which rule must survive.

## Reject
```text
INV-001: The API validates the request and writes a row to the database.
```
Failure: transport and storage mechanics replaced the rule. Nothing here can be false, so nothing can be proven.

```text
Eventual consistency makes this hold.
```
Failure: an unowned repair path. Either one boundary prevents the violation synchronously, or the row names the reconciliation that restores the rule and how long it may be false.

## Prove
Every row should imply a positive proof, a proof that its false case is rejected, and — when replay or concurrency can reach it — a proof that the enforcement point still holds with two writers.
