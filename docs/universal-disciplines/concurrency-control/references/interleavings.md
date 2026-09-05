# Breaking Interleavings

Load when the contested state, invariant, or writers are not yet explicit.

Name the smallest durable unit whose invariant can break: row, key, document,
or a cross-row fact such as a balance or at-most-one-active rule. Enumerate every
writer, including retries, sibling endpoints, workers, consumers, other service
replicas, operators, and external callbacks. Reads that choose a later write are
part of that write path.

Write the smallest two-actor schedule. Common shapes are lost update (`A read,
B read, A write, B write`), check-then-act (`A check, B check, A effect, B
effect`), write skew across disjoint rows, and a stale holder resuming after its
lease expires. If no schedule violates the invariant because one writer owns it
or operations commute and are idempotent, reject extra coordination.

A useful proof forces the schedule with barriers or injected pauses and fails
when the eventual arbitration mechanism is removed; worker-count stress alone
does not establish the schedule.
