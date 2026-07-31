# Production-ready Goose migration runtime

status: ready

## Scope and non-goals

### In scope

- Replace `golang-migrate` with `github.com/pressly/goose/v3` as the sole SQL
  migration engine retained by the base template.
- Preserve a separate production `/migrate` binary invoked by Railway
  pre-deploy.
- Make migration source admission, PostgreSQL locking, time budgets,
  transactional policy, failure reporting, history integrity, integration
  rehearsal, image packaging, and operator documentation one coherent
  production path.
- Make the Goose source format and safety checks part of initialization,
  local development, CI, the production image, and derived-service guidance.
- Remove replaced `golang-migrate` dependencies, code, tests, commands, and
  documentation.

### Non-goals

- Running migrations during application startup.
- Adding a second migration engine or retaining a compatibility mode in the
  base template.
- Rewriting or converting migration history in an already deployed derived
  repository.
- Adding schema diff generation, ORM auto-migration, Atlas, or a general
  database deployment platform.
- Treating `down` as the ordinary production rollback strategy.
- Running large data backfills inside schema migration transactions.

## Behavior and contract delta

### MIG-1 — Engine and source authority

The base template uses the pinned Goose Provider API as its only migration
engine. Repository-owned `.sql` files in the configured migration directory
are the canonical schema-change source.

- A Goose migration has one numeric version and one file containing explicit
  `-- +goose Up` and `-- +goose Down` sections.
- Every file name matches
  `^[0-9]{6}_[a-z][a-z0-9]*(?:_[a-z0-9]+)*\.sql$`. The six-digit version is
  in the range `000001` through `999999`; versions are unique and strictly
  increasing when files are sorted by name, but need not be contiguous.
- `sqlc` consumes the migration directory as schema input and therefore sees
  only Goose Up sections in the same lexicographic order as the runtime.
- Go migrations are not accepted.
- Legacy `.up.sql` and `.down.sql` files are rejected by template structure
  checks.
- Duplicate versions, malformed names, missing direction annotations, and
  forbidden annotations are errors before opening PostgreSQL. Goose
  parser-invalid migration sources block the merge/release gate; if such a
  file reaches the production runner despite that gate, Goose rejects it
  before executing SQL from that file. PostgreSQL syntax and behavior are
  proved by the migration rehearsal.
- Out-of-order or missing migration application is disabled.
- Applied migration files are append-only: a base-to-HEAD change may add a new
  migration, but may not modify, rename, or delete a migration already present
  in the comparison base.

The repository omits `migrations/` before its first owned migration. The image
still creates `/migrations`, and an existing runtime directory containing no
`.sql` migration files is the valid empty source state. An explicitly
configured or image-required migration directory that does not exist is a
packaging or operator error and fails closed.

### MIG-2 — Single production owner

The production `/migrate` binary is the only template-owned production entry
point for schema migration.

- It accepts no command or direction argument and applies pending migrations
  upward only.
- It runs before application promotion through the existing Railway
  pre-deploy hook.
- Application startup, readiness, replicas, and background workers never run
  migrations opportunistically.
- Multiple concurrent invocations are serialized by one PostgreSQL
  session-level advisory lock.
- Failure to acquire or release the required lock is a migration failure.

### MIG-3 — Finite lifecycle

Every migration run is bounded and cancellation-aware.

- Connection establishment and ping observe the caller context and the
  configured PostgreSQL connect timeout.
- Waiting for the migration advisory lock cannot exceed the configured
  migration-lock budget.
- SQL execution observes both the caller context and the configured statement
  timeout.
- Waiting for PostgreSQL DDL/data locks cannot exceed the configured
  migration-lock budget.
- Unlock and connection cleanup use a detached cleanup context because the
  operation context may already be canceled, but that cleanup context has its
  own finite deadline and cannot extend the run indefinitely.
- The existing overall migration timeout remains the outer budget. A timeout,
  cancellation, lock failure, SQL failure, metadata failure, or cleanup
  failure is never reported as success.

The default budgets remain five minutes overall, two minutes per statement,
and fifteen seconds for migration/advisory and PostgreSQL lock acquisition.
The existing PostgreSQL connection timeout remains authoritative for
connection establishment.

### MIG-4 — Transaction and exceptional-operation policy

Each migration runs in one transaction by default. Its version is recorded in
the same transaction as its SQL effects.

The base template rejects every migration containing
`-- +goose NO TRANSACTION` during source admission and CI. Goose does not
persist an attempt or dirty marker before non-transactional SQL, so accepting
that annotation would permit a killed process to leave durable effects that
the next run cannot distinguish from an unstarted migration without
repository-owned recovery machinery.

An operation that PostgreSQL forbids in a transaction, such as
`CREATE INDEX CONCURRENTLY`, is not smuggled into the ordinary migration path.
It requires a separately designed, restart-safe release operation with a
durable attempt record, deterministic absent/complete/partial observation,
bounded cleanup or forward repair, and dedicated integration proof. Adding
that release class is a reopen condition, not an undocumented exception in a
migration file.

Large backfills are separate restart-safe jobs with checkpoints, throttling,
abort criteria, and verification. A migration may perform only bounded data
changes whose worst-case statement duration fits its configured budget.

### MIG-5 — Version state and failure reporting

Goose's version table is authoritative for the set of successfully applied
migration versions. Repository Git history is authoritative for migration-file
identity and immutability.

Let `S` be source versions sorted ascending by the canonical six-digit file
name. Goose's PostgreSQL store returns version rows in reverse application
order (newest first). The runner requires one oldest bootstrap version `0`,
requires every non-bootstrap row to be applied, rejects duplicate or
non-ascending application history, and reverses those non-bootstrap rows to
construct canonical ascending applied versions `A`. When the version table is
absent, `A` is empty and there is no bootstrap row to validate.

Before new SQL runs, `A` must be exactly a prefix of `S`; the empty prefix and
the full prefix are valid. The run fails before applying SQL when:

- an applied database version is absent from the source;
- an earlier source version is missing from the applied set while a later
  source version is applied;
- applied versions are duplicated, unordered, or otherwise not the same
  prefix;
- the source is empty while the database has an applied version; or
- source admission found a duplicate, malformed, legacy, or forbidden
  migration.

The runner never repairs, deletes, renames, force-sets, or silently ignores a
conflicting version. Its error reports the current database version, source
target version, and recovery-document location without including SQL or
credentials. An absent Goose version table is the valid initial empty applied
set; any other metadata-read error fails closed.

Before applying work, `/migrate` reports the current and target versions.
For each applied migration it emits structured, secret-free fields for:

- version;
- base file name;
- direction;
- duration;
- result.

The terminal record includes current version before and after, target version,
applied count, total duration, and terminal result. A partial error names the
failed version and the versions completed during that invocation. Logs never
include the DSN, credentials, raw SQL, or environment contents.

Already-current schema and an existing empty source directory are successful
no-change outcomes. Missing source, inconsistent version state, forbidden
non-transactional source, connection failure, contention timeout,
cancellation, and cleanup failure are distinct nonzero outcomes.

### MIG-6 — Rehearsal and production parity

The repository migration gate uses disposable PostgreSQL and the same pinned
engine, source parser, configuration policy, and migration files as
production.

When an owned migration corpus exists, the gate proves:

1. all Up sections apply;
2. a second Up is a no-op;
3. all Down sections execute on the disposable database;
4. all Up sections reapply;
5. the production image `/migrate` reaches the same final version;
6. the application starts and becomes ready against that final schema;
7. cancellation, statement timeout, advisory-lock contention, DDL-lock
   timeout, and bounded cleanup fail without false success.

Before the first migration, the gate proves the existing-empty-directory
no-op and production image packaging behavior rather than inventing a schema.

### MIG-7 — Template and developer workflow

PostgreSQL-enabled initialized services retain Goose, the `/migrate` binary,
configuration, migration checks, Docker packaging, Railway pre-deploy wiring,
and documentation. PostgreSQL-disabled profiles remove those surfaces and the
direct Goose dependency.

Developer documentation provides:

- how to create a correctly named Goose SQL migration;
- annotation and transaction rules;
- how to run focused migration tests and full rehearsal;
- why production exposes only Up;
- append-only history rules;
- recovery for transactional and non-transactional failures;
- expand/backfill/verify/contract rollout guidance.

## Invariants and edge cases

- REST/OpenAPI behavior, application readiness semantics, sqlc authority, and
  PostgreSQL runtime-pool ownership remain unchanged.
- The same production image still contains `/service`, `/migrate`, and the
  migration directory.
- A no-change migration run does not claim that the directory was missing.
- A failed pre-deploy never promotes the new application image.
- Cancellation after one or more migrations reports partial completion rather
  than converting it to a global success.
- A cleanup error is retained even when the migration operation also failed.
- A migration file cannot enable environment substitution; deployment
  configuration is not SQL source authority.
- A migration file cannot opt out of transactions.
- A migration version is never reused.
- A successful Down rehearsal proves executable reversal on a disposable
  database, not that production rollback is safe after observed data changes.
- Schema contraction remains blocked until mixed-version compatibility and
  restore or forward-repair evidence exist.

## Decisions, constraints, and authorities

- Goose v3.27.3 Provider is the selected engine; its pinned source and
  documentation own engine behavior.
- PostgreSQL is authoritative for schema and migration state.
- Repository Git history owns review-time migration identity. The signed
  production image addressed by the dedicated `migration-history` GHCR marker
  owns the last published corpus because Goose does not store file checksums
  and a branch ref can be deleted or rewritten.
- Local history checks without a comparison ref prove only tracked worktree
  changes. Pull-request CI compares from the merge base with its base SHA;
  push CI compares the exact previous event tree to the candidate tree without
  merge-base normalization. An unavailable nonzero previous SHA fails closed.
  Main-image and tagged-release publication both require the exact successful
  `push` CI history job for the candidate SHA, then compare the candidate
  image's complete migration corpus with the corpus extracted from the
  immutable digest currently named by `migration-history`. Every prior path
  and byte must remain; additions alone are accepted. The marker advances to
  the verified candidate digest before any public alias advances. The first
  publication is allowed only when a complete successful GitHub Packages owner
  listing proves that the GHCR package does not exist and the repository variable
  `MIGRATION_HISTORY_BOOTSTRAP_SHA` equals the candidate SHA. A set bootstrap
  variable is rejected after the marker exists, forcing the one-time authority
  to be cleared before a later publication.
  An explicitly supplied unreadable local/PR base ref, unavailable required
  push-CI result, missing post-bootstrap marker, unreadable marker image,
  ambiguous/unauthorized package API result, an existing package without a
  marker, or failed corpus comparison fails closed.
- `internal/config` owns validated time budgets.
- `internal/infra/postgresmigrate` owns migration construction and execution
  policy.
- `cmd/migrate` owns production source admission and operator-facing terminal
  outcome.
- Railway pre-deploy owns promotion ordering, not migration correctness.
- CI and integration tests own rehearsal; production does not expose Down.

## Success criteria and proof expectations

1. No production or test path imports `golang-migrate`, and module drift proves
   the dependency was removed.
2. The production binary applies Goose SQL migrations, reports no change
   correctly, and rejects arguments, missing sources, legacy source format,
   malformed sources, lock contention, cancellation, and invalid budgets.
3. Current and target versions plus per-migration and terminal structured
   results are observable without secret or SQL disclosure.
4. Real PostgreSQL proof covers transactional rollback, Up/no-op/Down/Up,
   statement timeout, DDL lock timeout, concurrent migrator serialization,
   cancellation, and bounded cleanup.
5. `migration-validate` exercises the actual repository source when it exists
   and the exact production image migration path before application readiness.
6. Project structure and base-reference CI checks reject changed/deleted
   history, duplicate versions, noncanonical names, legacy pairs, invalid
   annotations, disallowed environment substitution, and every
   `NO TRANSACTION` migration.
7. Both image publication channels serialize around one `migration-history`
   marker, reject any prior published-file change even after branch
   deletion/recreation, and advance a public alias only after the candidate
   image, signature/attestation, and marker comparison succeed.
8. PostgreSQL-disabled and enabled initialization profiles retain exactly
   their owned dependency, source, commands, tests, and documentation.
9. Docker, Railway, command, configuration, architecture, and recovery
   documentation describe the implemented behavior without claiming live
   deployment proof.
10. Focused Go tests, PostgreSQL integration tests, race proof for affected
   lifecycle code, module checks, template initialization, and the matching
   repository aggregate pass on the final candidate.

## Risks, assumptions, and reopen conditions

- Assumption: the base template has no released migration history, so replacing
  its source convention is safe. Reopen rollout design if an owned migration
  appears before implementation lands.
- Assumption: initialized services use one PostgreSQL schema-history owner per
  database. Reopen lock identity if multiple independent migration owners must
  share one database without serializing.
- Goose has no persisted dirty marker for `NO TRANSACTION`. Reopen MIG-4 only
  when an accepted PostgreSQL operation requires non-transactional execution;
  that change needs its own restart-safe release-operation design and proof,
  not a relaxation of source admission.
- Goose and `golang-migrate` both lack file checksums. Repository review checks
  and the published OCI corpus therefore remain separate authorities. Reopen
  history integrity if GHCR cannot retain the `migration-history` manifest and
  immutable digest or if another publication path can bypass the shared
  comparison and promotion sequence.
- Reopen dependency behavior and rerun lifecycle proof on any Goose or pgx
  version change.
- Existing derived repositories require a separate, repository-specific
  cutover design; this task neither modifies nor certifies them.
