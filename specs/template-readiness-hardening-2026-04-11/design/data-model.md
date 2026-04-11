# Data And SQLC Design

## Decision

`ping_history` should not remain an unexplained production schema. It either becomes clearly documented sample material or is removed from the default runtime schema/query path.

Preferred implementation path:

1. Check whether sqlc generation/checks and the repository tests remain meaningful if `ping_history` migrations/queries/repository/generated artifacts are removed.
2. If yes, remove the runtime sample schema and replace it with docs that describe the persistence recipe.
3. If no, keep the sample temporarily but make it explicit in docs/tests as a template SQLC sample, not business state.

## Migration Rule

Default migrations should be deterministic:

- prefer `CREATE TABLE ...` over `CREATE TABLE IF NOT EXISTS ...`,
- prefer `DROP TABLE ...` over `DROP TABLE IF EXISTS ...`,
- document any intentional idempotent repair migration as an exception.

## Repository Rule

Hand-written repositories should:

- accept contexts from callers,
- wrap generated sqlc rows/types before exposing app-facing records,
- validate bounded list/page inputs before passing them to SQL,
- own transaction cleanup with bounded contexts,
- avoid leaking `sqlcgen` into `internal/app`.

## Test Rule

- Fast repository logic tests stay near `internal/infra/postgres`.
- Migration-backed read/write tests stay under `test/` with the `integration` tag.
- Integration helpers that apply schema directly must be named as schema bootstrap, not migration rehearsal, unless they use the actual migrate tool.

