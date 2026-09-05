---
name: postgres-schema-design
description: "Relational identity, cardinality, invariant, constraint, lifecycle, and migration depth reference."
---

# PostgreSQL Schema Design

Use only after the active phase identifies a durable relational-model decision.
Inherit its authority, artifact, review, proof, output, and completion contract;
do not select a work mode or authorize production DDL here.

Core invariant: every table has one row meaning and stable identity; every
business invariant is enforced by the database or assigned to one explicit
transactional owner with a falsifier.

Load one branch:

- relation grain, identity, cardinality, normalization, or lifecycle ->
  [relational-design.md](references/relational-design.md);
- PostgreSQL type, constraint, index, generated value, RLS, or migration shape
  -> [postgresql-ddl.md](references/postgresql-ddl.md).
