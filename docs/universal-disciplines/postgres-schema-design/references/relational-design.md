# Relational Design

Load when relation grain, identity, cardinality, normalization, or lifecycle can
change the model.

Start from candidate keys and functional dependencies. Each relation has one
row meaning; every non-key fact depends on that key. Decompose repeating,
partial-key, transitive, or non-key determinants only when the join is lossless
and the resulting invariant remains enforceable. Atomicity is domain-relative:
split a value when its parts have independent rules or operations, not because
it contains punctuation.

Many-to-many facts belong on an association relation with domain uniqueness.
Use a surrogate only when other records address the association, while retaining
its business key. One-to-one extensions need a distinct lifecycle, access
boundary, sparse large data, or subtype rule. Polymorphic `(type,id)` cannot
carry an ordinary foreign key; prefer a shared parent or separate associations
when referential integrity matters.

Choose hierarchy shape from operations: adjacency for ordinary parent changes,
closure for frequent ancestry, materialized path for subtree reads that can pay
reparenting cost. Distinguish effective and recorded time when history needs both
clocks. Snapshot mutable commercial facts that must not change with current
catalog state. Use range/exclusion constraints for non-overlap when supported.

Tenant ownership participates in keys and foreign keys whenever identity is
tenant-scoped. RLS may add authorization defense but does not replace integrity.
Choose cascade/restrict/null from ownership and residual row meaning. Soft
deletion is a product state with visibility, retention, restore, foreign-key,
and uniqueness rules.

Use typed relations for stable facts involved in joins, uniqueness,
authorization, lifecycle, or filtering. `jsonb` fits a bounded atomic document
with a schema/size/query owner; EAV fits only genuinely user-defined attributes
with a type catalog and query limits. Every denormalized copy names authority,
measured benefit, synchronization, staleness/failure, rebuild, and write/storage
cost.
