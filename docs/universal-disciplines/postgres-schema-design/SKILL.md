---
name: postgres-schema-design
description: "Invariant-first PostgreSQL schema design. Use when translating business entities and rules into normalized tables, keys, relationships, constraints, ERDs, or safe migrations; or reviewing a schema for integrity gaps, update anomalies, temporal or tenant rules, polymorphic/EAV/JSON misuse, and intentional denormalization. Use postgres-performance instead when the primary outcome is query, load, or index tuning."
---

# Invariant-First PostgreSQL Schema Design

A schema is an executable model of business invariants:

`language -> invariants -> dependencies -> relations -> constraints -> scenarios`

Model correctness before physical optimization. Start normalized; make every denormalization name its source of truth, synchronization mechanism, repair path, and measured reason.

## Request and authority contract

Before modeling, establish the requested outcome, schema boundary, work mode (design, review, or implementation), target PostgreSQL version when relevant, available evidence, and success criteria. Distinguish confirmed rules from inferences and assumptions; surface the distinctions that affect the design.

- For design, review, or planning requests, inspect available product contracts, code, schemas, migrations, queries, and tests; preserve database and repository state.
- For build or change requests, edit canonical migrations, schema definitions, models, and tests within the supplied repository, then run non-destructive validation.
- Treat production DDL, backfills, data repair, constraint validation, and migration execution as separate production actions requiring explicit authority.

Ask a question when two plausible business meanings produce materially different identities, ownership, cardinalities, retention, money, or time semantics. Otherwise proceed with explicit assumptions.

Finish when the applicable completion criteria below are satisfied and the requested artifact is delivered at the stated schema boundary.

## 1. Extract the invariants

Read product language before drawing tables. Capture:

- business operations and states;
- identity and duplicate rules;
- cardinality and optionality;
- ownership, lifecycle, deletion, and retention;
- mutable facts versus historical snapshots or events;
- tenant, authorization, privacy, and audit boundaries;
- expected reads, writes, imports, retries, and concurrent conflicts.

Treat existing APIs and schemas as evidence. When they conflict with an explicit product rule, surface the conflict instead of silently preserving it.

**Completion criterion:** every supplied business rule is represented as a testable invariant or an explicit assumption, including create, update, delete, duplicate/retry, and history scenarios that matter to the feature.

## 2. Give each fact an identity and lifecycle

Classify each concept as an entity, value owned by another entity, association, immutable event, or historical snapshot. For every independently stored concept, state:

- the grain: exactly what one row represents;
- stable identity and candidate keys;
- owner and lifecycle;
- mutable and immutable attributes;
- effective time versus recorded time, when relevant.

Use a surrogate key when it improves references or lifecycle independence; retain business identifiers as `UNIQUE` constraints when the domain says they identify duplicates.

**Completion criterion:** every proposed relation has one declared grain, identity, owner, duplicate rule, and lifecycle; ambiguous concepts remain listed as decisions rather than hidden in columns.

## 3. Make relationships explicit

For each relationship, state both directions, cardinality, optionality, ownership, foreign-key direction, delete behavior, and whether the relationship has its own attributes or lifecycle. Model a many-to-many relationship as an association relation; its business identity determines its primary or unique key.

Write the functional dependencies that affect decomposition: `determinant -> dependent facts`. Prefer one authoritative storage location for each mutable fact.

**Completion criterion:** every relationship required by an invariant has recorded cardinality, optionality, ownership, foreign-key direction, delete behavior, and business uniqueness.

## 4. Normalize, then justify exceptions

Read [references/relational-design.md](references/relational-design.md) when the request includes a normalization review or the model includes composite keys, subtypes, hierarchies, temporal data, multitenancy, soft deletion, polymorphism, EAV/JSON, or denormalization.

Check 1NF, 2NF, 3NF, and BCNF against candidate keys and functional dependencies. Each decomposition must be lossless. Preserve dependencies in local constraints where practical; name any invariant that now requires a transaction, trigger, or cross-relation test.

Use normal forms to remove named update, insert, or delete anomalies; let the dependencies determine the relation count.

For duplicated or derived facts, record:

- authoritative source;
- freshness contract and update mechanism;
- failure and repair behavior;
- workload evidence that pays for the added consistency cost.

**Completion criterion:** every stored fact depends on the key of its relation; decompositions are lossless; every remaining dependency and intentional duplicate has one enforcement owner.

## 5. Map invariants to PostgreSQL

Read [references/postgresql-ddl.md](references/postgresql-ddl.md) when the requested artifact includes concrete column types, constraints, PostgreSQL DDL, or a migration plan.

Choose types from domain semantics, then map invariants to `NOT NULL`, primary keys, `UNIQUE`, foreign keys, `CHECK`, and exclusion constraints. Use transaction or application enforcement only for rules PostgreSQL cannot express declaratively, and pair each such rule with a concurrency-aware test.

**Completion criterion:** every invariant maps to a database constraint or to one explicitly owned enforcement path with its race model and test; version-specific syntax is verified against the target PostgreSQL version.

## 6. Walk adversarial scenarios

Attempt representative valid and invalid transitions:

- duplicate creation and idempotent retry;
- missing or deleted parent;
- concurrent inserts or updates;
- cross-tenant reference;
- mutable reference data after historical records exist;
- optional values participating in uniqueness;
- time-range overlap and boundary instants;
- migration from existing dirty or partially populated data.

For repository changes, run the smallest available migration/schema checks and constraint-focused tests. For design-only work, specify executable examples that should succeed or fail.

**Completion criterion:** every material invariant has at least one accepted scenario and one rejected or conflict scenario, with the responsible constraint or enforcement path named.

## 7. Report the model

Use the ordered template below as a menu, keeping only sections that carry information.

Include a Mermaid ERD when three or more relations interact or the hierarchy is otherwise hard to read.

```markdown
## Verdict
[Model boundary, major decisions, readiness, and artifact status: proposed, implemented locally, migrated, or verified live]

## Assumptions and decisions
- [Confirmed rule, explicit assumption, or blocking question]

## Relations
| Relation | One row represents | Identity | Owner/lifecycle |
| --- | --- | --- | --- |

## Relationships
| From -> to | Cardinality/optionality | Enforcement | Delete behavior |
| --- | --- | --- | --- |

## Invariant coverage
| Invariant | Constraint or owner | Validation scenario |
| --- | --- | --- |

## DDL or migration
[Proposed or changed artifacts, compatibility, rollout, rollback/roll-forward]

## Tradeoffs and gaps
[Intentional denormalization, unenforced rules, workload handoff, open evidence]
```

**Completion criterion:** every material relation, relationship, invariant, assumption, and remaining gap appears in the report, and the verdict makes readiness unambiguous.
