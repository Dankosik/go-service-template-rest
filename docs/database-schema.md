# Database Schema

Read this owner when a task reads, changes, or analyses a relational database
schema, or writes a migration.

The database catalog is the current schema. Migration files are its change
history. Do not reconstruct the current schema by reading the migration chain.
Read a specific migration, or its Git history, only to learn when, why, or how
something changed.

## Local Instance

`make db-up` starts a throwaway PostgreSQL from `env/docker-compose.yml`. It
applies the configured migration chain with its engine and prints the `psql`
command for that instance. Ask the catalog through that command:

| Question | `psql` input |
| --- | --- |
| One table with its columns, defaults, constraints, indexes, foreign keys, referencing tables, and triggers | `\d+ <table>` |
| One function or procedure body | `\sf+ <function>` |
| Tables, views, functions, types | `\dt`, `\dv`, `\df`, `\dT` |
| Privileges | `\dp <table>` |

To search the whole schema, run `pg_dump --schema-only --no-owner` through the
same container into a temporary file, then search that file.

`make db-down` removes the instance. Recreate it after switching to a branch
with different migrations, or after changing a migration that it has already
applied. A new migration applies on the next `make db-up`, and the same
commands show its effect.

`make db-up` stops at the first migration that cannot apply to an empty
database, for example one that checks production data or needs roles that the
platform provisions. Report that migration, then use the partly migrated
instance or an authorized live database. Where `make db-up` does not exist,
apply the chain to any throwaway PostgreSQL with the repository's migration
tool, then query its catalog the same way.

## Live Database

When analysing a running database, its catalog is the truth. Out-of-band DDL,
invalid indexes, and unapplied or newer migrations exist only there. Compare
the applied migration version with the repository head before comparing
shapes, and report differences as findings. Access stays read-only within the
authority that [AGENTS.md](../AGENTS.md#authority) grants.

Other stores follow the same rule; for ClickHouse, use `SHOW CREATE TABLE`.
Squashing or rewriting migrations does not make the schema visible; it only
compresses history.
