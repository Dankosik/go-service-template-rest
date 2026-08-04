# PostgreSQL DDL reference

Read this reference before emitting PostgreSQL DDL or editing migrations. Verify syntax and operational behavior against the target PostgreSQL version and provider.

## Identity and keys

- Give every entity relation a primary key. Use identity columns or UUIDs according to creation topology, external exposure, index locality, and existing conventions; neither replaces domain uniqueness.
- Preserve each business identifier that defines duplicates with `UNIQUE` plus the null semantics the rule requires.
- Use composite primary keys when the relationship itself is the stable identity and references remain manageable. A surrogate key can coexist with a composite unique constraint.
- Sequences and identity columns allocate identifiers; they do not promise gapless business numbering. Model legally gapless numbering as a separate serialized business process.

## Types from semantics

- Use integer or `numeric(p,s)` for exact quantities. Store money as an exact amount plus currency; choose integer minor units only when every supported currency and rounding rule fits that representation. PostgreSQL `money` depends on locale and has a fixed fractional precision.
- Use `double precision` for approximate measurements, not exact commercial values.
- Use `date` for civil dates, `time` only for a time-of-day rule, `timestamptz` for an instant, and `timestamp` for a local wall-clock value whose zone is supplied by the domain. Store a zone identifier separately when future civil-time recurrence depends on it.
- Prefer `text` plus a constraint when length is a business rule; `varchar(n)` adds value only when that maximum is itself meaningful.
- Choose PostgreSQL enums for small, stable value sets whose deployment coupling is acceptable; use `CHECK` for table-local closed sets and a referenced relation for values with metadata or independent lifecycle.
- Use arrays or `jsonb` for one bounded atomic value. Promote stable, independently constrained or related members into typed columns or child relations.

## Constraint mapping

- Mark required facts `NOT NULL`; optionality is a business rule rather than a migration convenience.
- Use primary and unique constraints for candidate keys. PostgreSQL unique constraints treat nulls as distinct by default; use `NULLS NOT DISTINCT` on supported versions when one null must participate in uniqueness.
- Use foreign keys for referential integrity. Match referencing and referenced types, choose `ON DELETE`/`ON UPDATE` from lifecycle semantics, and use composite foreign keys for scoped identity such as tenant ownership.
- Add an index on referencing foreign-key columns when parent changes, deletes, or joins make that access path material; PostgreSQL does not create it automatically.
- Keep `CHECK` expressions within the current row and based on effectively immutable behavior. Cross-row rules belong in `UNIQUE`, exclusion constraints, foreign keys, or a concurrency-safe enforcement path.
- Use exclusion constraints for rules such as non-overlapping ranges. Name boundary semantics explicitly; half-open intervals often make adjacent periods unambiguous.
- Use generated columns for row-local derived values supported by the target version. Their expression restrictions and replication behavior are version-specific.

## Security and ownership

- Put tenant ownership and privacy boundaries in the schema model before adding row-level security.
- Treat row-level security as authorization defense in depth. Verify table owners, bypass roles, policy combinations, background workers, and backup behavior; referential-integrity checks have special visibility behavior.
- Separate secrets and highly restricted attributes when that creates a useful privilege, retention, or audit boundary. Table separation without different controls is only cosmetic.

## Existing data and migrations

For a new empty table, emit the complete invariant set together. For a populated table, separate logical end state from rollout mechanics:

1. inspect existing nulls, duplicates, orphans, ranges, and type violations;
2. make new application writes compatible with both schemas when required;
3. add constraints in the least-locking supported form, using `NOT VALID` and later validation where PostgreSQL permits it;
4. backfill in bounded, restartable batches with a verification query;
5. switch reads and writes;
6. remove compatibility columns or code only after the rollback window closes.

Concurrent index creation, constraint validation, type changes, defaults, and table rewrites have different transaction and lock behavior. Use the repository's migration framework and production runbook rather than copying a universal sequence.

## Validation

Prefer the smallest executable proof available:

- apply migrations to an empty database and to a representative previous schema;
- run schema-diff or generated-schema checks owned by the repository;
- insert one valid row graph;
- attempt each expected constraint violation;
- test duplicate/retry and concurrent-conflict paths;
- verify delete/update actions and tenant isolation;
- verify roll-forward or rollback behavior promised by the migration plan.

A successful local migration is not production proof.

## References

- [PostgreSQL: data definition](https://www.postgresql.org/docs/current/ddl.html)
- [PostgreSQL: constraints](https://www.postgresql.org/docs/current/ddl-constraints.html)
- [PostgreSQL: numeric types](https://www.postgresql.org/docs/current/datatype-numeric.html)
- [PostgreSQL: date/time types](https://www.postgresql.org/docs/current/datatype-datetime.html)
- [PostgreSQL: identity columns](https://www.postgresql.org/docs/current/ddl-identity-columns.html)
- [PostgreSQL: generated columns](https://www.postgresql.org/docs/current/ddl-generated-columns.html)
- [PostgreSQL: row security](https://www.postgresql.org/docs/current/ddl-rowsecurity.html)
- [GitLab: foreign keys and associations](https://docs.gitlab.com/development/database/foreign_keys/)
- [GitLab: migration style guide](https://docs.gitlab.com/development/migration_style_guide/)
