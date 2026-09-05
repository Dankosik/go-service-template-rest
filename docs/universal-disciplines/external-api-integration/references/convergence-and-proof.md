# Provider Convergence And Proof

Load when an external side effect can complete ambiguously, through callback or
poll, across reconciliation, or during provider migration.

Persist local operation identity and canonical intent before I/O. Bind provider
idempotency/reference data to that identity and account/environment. Automatic
write retry after possible transmission is safe only under the pinned provider
contract or after authoritative resolution.

Classify outcomes as `succeeded`, `permanent`, `retryable`, `ambiguous`, or
`incompatible`. Status code alone is insufficient. A timeout, truncated
response, or connection loss after possible send stays ambiguous until a
documented lookup, authenticated callback, poll, or reconciliation establishes
the result. `not found` proves absence only after the provider's documented
visibility window and lookup completeness.

Reconciliation selects non-terminal, overdue, failed-callback, and drift
candidates with a declared lookback/checkpoint; it queries provider truth by a
stable identifier and applies only allowed transitions. Measure oldest
ambiguity, reconciliation lag, checkpoint age, and resolved/unresolved drift.

Proof injects failure before send, after possible send, after provider commit,
and around local persistence; it correlates the same operation through
callback/poll and reconciliation. Migration assigns one writer per operation,
preserves identities across routing, drains both providers, and keeps old
callbacks/reconciliation until in-flight ambiguity closes. Rollback stops new
routing but never erases unresolved effects.
