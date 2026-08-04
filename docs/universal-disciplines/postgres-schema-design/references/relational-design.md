# Relational design reference

Use the branches that match the model. The target is a lossless model in which each mutable fact has one authoritative owner and each business invariant has an enforcement path.

## Normalization test

Start from candidate keys and functional dependencies rather than column appearance.

| Form | Test | Typical repair |
| --- | --- | --- |
| 1NF | Each attribute holds one domain value at the relation's grain; repeating groups are relations. | Move repeated values into a child or association relation. |
| 2NF | Every non-key fact depends on the whole of each composite candidate key. | Split facts determined by only part of a composite key. |
| 3NF | Non-key facts depend on candidate keys rather than other non-key facts. | Move the transitive determinant and its facts to their own relation. |
| BCNF | Every non-trivial determinant is a candidate key. | Decompose when a non-key determinant creates update anomalies; record any dependency that becomes non-local. |

A useful decomposition preserves all original rows through a lossless join. Prefer dependency preservation as well: constraints spanning several relations cost more to enforce and are easier to race.

Atomicity is domain-relative. An address can be one immutable display value in one context and several independently queried facts in another. Split it according to invariants and operations, not punctuation.

## Association relations

A many-to-many relationship becomes a relation with foreign keys to both sides. Put relationship facts such as role, quantity, position, validity, or provenance there. Choose its identity from the domain:

- `UNIQUE (left_id, right_id)` when one active association may exist;
- a wider business key when context or version distinguishes associations;
- a surrogate key when other records reference the association, while retaining the business uniqueness constraint.

An optional one-to-one extension earns a separate table when it has a distinct lifecycle, access boundary, sparse large attributes, or subtype-specific invariants. Enforce one-to-one with a unique foreign key or a shared primary key.

## Subtypes and polymorphism

Choose explicitly among:

- single-table subtypes when most fields and constraints are shared and nullable subtype fields remain bounded;
- class-table subtypes with one base identity and one-to-one subtype tables when common references need a real shared parent;
- concrete tables when subtypes have independent identities and relationships.

A `(target_type, target_id)` pair cannot carry an ordinary foreign key to several target tables. Prefer a genuine shared parent identity or separate association tables when referential integrity matters. PostgreSQL table inheritance also does not extend unique, primary-key, or foreign-key constraints across the hierarchy, so treat it as a specialized storage feature rather than a default domain-polymorphism model.

## Hierarchies

- Adjacency list (`parent_id`) is the default for trees with ordinary parent/child changes; prevent illegal cycles in a transaction or dedicated validation path.
- Closure tables fit frequent ancestor/descendant queries at the cost of maintaining transitive rows.
- Materialized paths fit subtree reads and prefix operations when reparenting cost and path repair are acceptable.

Declare whether multiple parents are valid, whether ordering among siblings is business data, and what deletion or reparenting does to descendants.

## Temporal facts and history

Separate current state, immutable events, and historical snapshots according to the product contract. Name both clocks when needed:

- effective time: when the fact is true in the business domain;
- recorded time: when the system learned or stored it.

Historical documents such as order lines usually snapshot mutable commercial facts that must not change with the current catalog. For non-overlapping validity or booking periods, prefer PostgreSQL range types plus an exclusion constraint where the target version and operator classes support the rule. When overlap is forbidden only for active rows, make the exclusion constraint partial with a predicate that matches the lifecycle rule.

## Multitenancy

Make tenant ownership part of the relational invariant. When identifiers are only tenant-unique, use matching composite unique keys and foreign keys such as `(tenant_id, entity_id)` so a child cannot reference another tenant's row. Keep tenant identity in association relations and uniqueness rules where it determines scope.

Row-level security can add a database authorization boundary, but integrity constraints remain necessary. Test privileged roles, background jobs, backups, and referential-integrity behavior separately.

## Deletion and history

Choose `ON DELETE` behavior from ownership:

- `CASCADE` for dependent data whose lifecycle is wholly owned by the parent;
- `RESTRICT` or `NO ACTION` when deletion must wait for an explicit business transition;
- `SET NULL` only when the remaining row has a valid independent meaning.

Soft deletion is a product state, not a universal substitute for deletion. Define visibility, retention, restoration, foreign-key behavior, and whether uniqueness applies across deleted rows. Partial unique indexes can express active-row uniqueness when that is the actual rule.

## JSON, arrays, and EAV

Use typed columns and relations for stable facts that participate in joins, uniqueness, references, authorization, lifecycle, or frequent filtering. `jsonb` is appropriate for bounded documents whose substructure is genuinely flexible and updated as one atomic datum; document its schema, size, query paths, and validation owner.

EAV trades DDL evolution for weak types, weak constraints, complex queries, and hidden dependencies. Reserve it for genuinely user-defined attributes with a declared type/validation catalog and explicit query limits.

## Denormalization ledger

Every deliberate duplicate or precomputed aggregate records:

| Decision | Required evidence |
| --- | --- |
| Source of truth | Exact normalized relation or event |
| Benefit | Named query or write path and measured target |
| Synchronization | Transaction, generated value, worker, or refresh process |
| Failure semantics | Staleness bound and behavior during lag/failure |
| Repair | Rebuild/backfill procedure and verification query |
| Cost | Write amplification, locking, storage, and operational ownership |

## References

- [CMU: functional dependencies and schema decomposition](https://15445.courses.cs.cmu.edu/fall2017/notes/04-notes-functionaldependencies.pdf)
- [Microsoft: database normalization basics](https://learn.microsoft.com/en-us/previous-versions/troubleshoot/microsoft-365/microsoft-365-apps/access/database-normalization-description)
- [PostgreSQL: table inheritance caveats](https://www.postgresql.org/docs/current/ddl-inherit.html)
- [PostgreSQL: range constraints](https://www.postgresql.org/docs/current/rangetypes.html#RANGETYPES-CONSTRAINT)
- [PostgreSQL: JSON document design](https://www.postgresql.org/docs/current/datatype-json.html#JSON-DOC-DESIGN)
- [GitLab: polymorphic associations](https://docs.gitlab.com/development/database/polymorphic_associations/)
