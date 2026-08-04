# Reference Selector

State the decision pressure and behavior-change thesis before loading.

| Symptom or decision pressure | Load | Behavior change |
| --- | --- | --- |
| A migration must add a column, constraint, index, enum value, or backfill against a table that already holds rows. | [migrations-in-one-transaction.md](migrations-in-one-transaction.md) | Split the change into additive DDL plus an application-owned backfill instead of the concurrent index and in-migration backfill this repository's runner rejects. |
| A change publishes state downstream, adds a projection, export, or search surface, or sets a retention rule. | [authority-and-derived-surfaces.md](authority-and-derived-surfaces.md) | Append to the outbox this repository already owns and keep retention off `outbox_ordering_heads`, instead of building a second delivery path. |

Modeling pressure with no repository-specific answer — normalization, keys and identity, tenant scoping, money and time types, constraint choice, JSONB versus relational, soft deletion, row-level security — belongs to [postgres-schema-design](../../../../docs/universal-disciplines/postgres-schema-design/SKILL.md), which also owns the expand-and-contract sequence. Concurrency mechanism selection belongs to [concurrency-control](../../../../docs/universal-disciplines/concurrency-control/SKILL.md), index and query cost to [postgres-performance](../../../../docs/universal-disciplines/postgres-performance/SKILL.md).
