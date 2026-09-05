# PostgreSQL DDL

Load before PostgreSQL DDL or migration shape is decided; verify version and
provider behavior against current primary authority.

Give entities a primary key and retain every business duplicate rule as a unique
constraint with deliberate null semantics. Identity/UUID choice follows creation
topology, exposure, locality, and repository convention; neither supplies domain
uniqueness or gapless numbering. Use exact integer/numeric plus currency for
money, floating point only for approximate measures, `date` for civil date,
`timestamptz` for an instant, and a separate zone when future civil recurrence
depends on it.

Map required facts to `NOT NULL`, scoped identity to matching composite
unique/foreign keys, row-local rules to `CHECK`, candidate keys to unique,
referential ownership to foreign keys and explicit delete/update action, and
non-overlap to exclusion constraints. PostgreSQL does not automatically index
referencing foreign keys. Cross-row rules need unique/exclusion/FK or one
transactional concurrency owner; volatile cross-row lookup in `CHECK` is not
one. Generated values, enum evolution, `NULLS NOT DISTINCT`, and RLS behavior are
version-specific.

For populated tables, separate the logical end state from rollout: inspect
violations; make writers dual-compatible; add the least-locking supported shape
(`NOT VALID` plus later validation where valid); backfill in bounded resumable
batches with checkpoints and a verification query; switch reads/writes; remove
compatibility only after rollback closes. Concurrent index creation, validation,
type/default change, and rewrite differ in locks, transactions, WAL, disk,
failure remnants, and replica cost.

Proof applies migrations to empty and representative previous schemas, runs the
repository drift check, inserts one valid graph, attempts each constraint
violation and duplicate/concurrent conflict, verifies delete/update and tenant
scope, and exercises the promised rollback or roll-forward boundary. Local
migration success is not production evidence.
