# Migrations, schema evolution, and data reliability instructions for LLMs

## Load policy
- Load: Optional
- Use when:
  - Designing or reviewing schema migrations for production systems
  - Planning zero-downtime rollout across schema and application versions
  - Designing expand/contract changes, backfills, reindexing, and data verification steps
  - Defining rollback limits, recovery strategy, and migration risk controls
  - Defining backup/restore drills, retention, archival, PII deletion, and DR basics
- Do not load when: The task is a local code-only refactor with no schema/data-lifecycle impact

## Purpose
- This document defines operational defaults for schema evolution without downtime.
- Goal: keep service availability and data correctness during rolling deployments where old and new code coexist.
- Treat this as an LLM contract: produce phased rollout plans, enforce compatibility rules, and reject destructive-first changes.

## Baseline assumptions
- Default DB: PostgreSQL-compatible OLTP.
- Default deployment model: rolling or canary, with mixed application versions during rollout.
- Default ownership model: one service owns one schema/database boundary.
- Default consistency model across services: eventual consistency with explicit idempotency/outbox patterns.
- If DB engine, table size, traffic profile, or RTO/RPO are missing:
  - assume PostgreSQL,
  - assume no maintenance window,
  - assume zero-downtime is required,
  - assume migration must be backward compatible until contract phase.

## Required inputs before generating a migration plan
Resolve these first. If missing, apply defaults and state assumptions.

- DB engine and major version
- Table/index size and estimated row counts for touched objects
- Traffic profile (peak write/read periods)
- Allowed lock budget per statement
- Deployment model (rolling/canary/blue-green)
- Recovery objectives (RPO, RTO)
- Data lifecycle class (business critical, audit, PII)
- Event/publication dependencies (outbox/CDC/broker)

## Compatibility contract during rollout
Default rule: migrations and code must be version-compatible across mixed rollout.

- Expand phase schema must work with both old and new application versions.
- Application changes must be tolerant to old and new schema simultaneously.
- Contract phase starts only after data verification passes and old code is fully drained.
- If compatibility cannot be guaranteed, require explicit maintenance window and approval.

Compatibility rules:
- Add before remove.
- Read old+new during transition; write strategy must be explicitly defined.
- Never rename in place as one step on critical tables.
- Never require “all pods switch instantly” for correctness.

## Migration execution defaults
Default rule: migrations run by one controlled process, not by every app instance.

- Use versioned, immutable migration files in VCS.
- Use a dedicated migration job/stage in CI/CD.
- Keep migration user/role separate from runtime application role.
- Stop on dirty migration state and require manual operator decision.
- Record migration start/end, version, duration, and failure reason.

PostgreSQL safety defaults:
- Set session-level `lock_timeout` in migration session (default: 2s unless overridden by service SLO).
- Set session-level `statement_timeout` in migration session (default: 15m unless operation requires more and is approved).
- Set `idle_in_transaction_session_timeout` for migration session (default: 60s).
- Do not set global `lock_timeout` in DB config as a migration shortcut.

## Default rollout sequence (schema + application)
Use this sequence for behavior-changing schema updates.

1. Prepare.
- Define change class: additive, backfill-required, constraint-tightening, destructive.
- Define rollback mode: code rollback only, roll-forward only, or limited schema rollback.
- Define verification queries and success thresholds before touching production.

2. Expand schema.
- Add nullable/new columns, new tables, new indexes, or `NOT VALID` constraints.
- Avoid destructive operations in this phase.
- Keep old code fully functional.

3. Deploy compatibility code.
- New code must tolerate both old and new schema.
- Reads use fallback logic if backfill is incomplete.
- Writes must follow explicit transition policy:
  - same-DB dual-column writes in one transaction are allowed as temporary bridge,
  - cross-system dual writes (DB + broker/API) are forbidden; use outbox.

4. Backfill/rebuild.
- Run idempotent background jobs in small batches.
- Track progress with durable watermark/checkpoint.
- Throttle to protect p95/p99 latency and replication lag budgets.

5. Verify.
- Validate row-level and aggregate invariants.
- Confirm no incompatible app versions are still serving traffic.
- Confirm replication/CDC lag is inside threshold.

6. Switch reads/writes.
- Move read path to new fields/indexes under feature flag.
- Disable transitional write path after verification window.

7. Contract.
- Drop deprecated columns/indexes/paths only after verification gate passes.
- Keep rollback notes explicit: contract may be irreversible without restore.

## Expand -> Migrate/Backfill -> Contract playbook

### Expand
- Prefer additive DDL only:
  - `ADD COLUMN` nullable,
  - `ADD TABLE`,
  - `CREATE INDEX CONCURRENTLY` (PostgreSQL),
  - `ADD CONSTRAINT ... NOT VALID`.
- For uniqueness on large tables (PostgreSQL):
  - build unique index concurrently,
  - attach constraint via `USING INDEX`.
- For `NOT NULL` transition (PostgreSQL):
  - add `CHECK (col IS NOT NULL) NOT VALID`,
  - validate constraint,
  - then set `NOT NULL`.

### Migrate/backfill
Default rule: backfills are resumable jobs, never one giant transaction.

- Must be idempotent (safe retry).
- Must persist progress (`watermark`, `last_processed_id`, or equivalent).
- Must commit per batch.
- Must include bounded retries with backoff for transient failures.
- Must include stop conditions:
  - error-rate threshold,
  - replication/CDC lag threshold,
  - lock wait budget threshold.

Suggested starting defaults (override per service profile):
- batch size: 500-2000 rows
- inter-batch sleep: 25-100 ms
- max retry attempts per batch: 3
- verification cadence: every 10k-50k migrated rows

### Contract
Default rule: contract is last and often rollback-limited.

Preconditions:
- 100% production traffic on compatible/new code path
- verification queries show no invariant violations
- backfill completion = 100%
- rollback strategy updated to restore-based if destructive

Contract actions:
- remove old write path
- remove old read fallback
- drop deprecated columns/indexes/tables
- tighten final constraints

## Reindexing and maintenance rules
- For PostgreSQL, prefer `CREATE INDEX CONCURRENTLY` / `REINDEX CONCURRENTLY` on hot tables.
- Do not use blocking index builds on write-critical tables without explicit downtime window.
- Treat invalid/failed concurrent index builds as incident-grade signals; repair before proceeding.
- Do not use `VACUUM FULL` as routine maintenance in production rollouts.

## Data verification defaults
Default rule: no contract migration without objective verification evidence.

Required verification layers:
- Row count parity where applicable (source vs target representation)
- Aggregate parity (sum/count/min/max checks by partition/tenant)
- Deterministic sample diff checks (stable sampling strategy)
- Domain invariant checks (uniqueness, referential assumptions, status transitions)

Acceptance gates (default):
- Critical invariants: 0 violations
- Aggregate delta tolerance: 0 unless explicitly documented
- Verification repeated at least twice after backfill completion before contract

## Rollback strategy and limitations
Default rule: prefer fast code rollback after expand; avoid relying on down migrations for destructive steps.

Rollback classes:
- Safe rollback:
  - code-only rollback while expanded schema remains
- Conditional rollback:
  - limited schema rollback for additive changes if no data loss risk
- Restore-based rollback:
  - destructive/data-rewrite phases requiring backup restore or roll-forward fix

Non-negotiable limitations:
- Dropped columns/tables are not safely recoverable without restore.
- Backfilled transformed data can be semantically irreversible.
- Once contract is applied, old binaries may no longer function.
- Down migrations are not a guaranteed emergency path in production.

## Event publication, dual writes, and reliability during evolution
Default rule: no cross-system dual writes.

- If state change must emit event/message, use transactional outbox (or equivalent atomic pattern).
- CDC consumers must be idempotent and replay-safe.
- Monitor WAL/binlog/oplog retention and consumer lag.
- Do not contract schema while downstream consumers still depend on old payload/field semantics.

## Backup, restore drills, retention, archival, and PII deletion
Default rule: backup strategy is valid only if restore is tested.

### Backup and restore drills
- Distinguish backup vs archive:
  - backup = operational recovery,
  - archive = long-term/compliance retention.
- Define explicit RPO and RTO per data class.
- PostgreSQL default for critical services:
  - periodic base backups + continuous WAL archiving (PITR-capable).
- Run restore drills on schedule:
  - at least monthly restore test in non-prod,
  - at least quarterly full runbook drill for critical services.
- Each drill must verify:
  - recovery time,
  - data correctness checks,
  - application startup against restored data.

### Retention and archival
- Define retention policy per dataset: hot, warm, archive, purge.
- Retention must include deletion trigger, storage class, and owner.
- Archival must be immutable, encrypted, and restore-tested.
- Do not keep undeclared “just in case” data indefinitely.

### PII deletion defaults
- Maintain explicit data map of PII fields and storage locations.
- Support deletion requests with traceable workflow and audit trail.
- Delete/anonymize in primary stores first, then propagate to derived stores.
- For immutable backups/archives:
  - do not perform unsafe in-place edits,
  - enforce expiration-based deletion windows,
  - document residual retention period and legal basis.
- Never claim “hard deleted” if retained in recoverable backup scope.

### Disaster recovery basics
- Maintain DR runbook for:
  - accidental destructive migration,
  - regional outage,
  - corruption/backfill bug.
- Run failover exercise at least semiannually for critical services.
- Define primary/replica read policy during incidents (avoid stale read assumptions).
- Keep schema migration tooling and restore tooling version-compatible and tested together.

## Decision rules (if/then)
1. If schema change can be additive, use expand/contract and avoid maintenance window.
2. If change is destructive by nature, postpone destruction until compatibility window closes and backups are verified.
3. If touched table is large or write-critical, use online/concurrent index and constraint strategies.
4. If backfill affects user-facing latency or replication lag, throttle and extend rollout timeline.
5. If event emission is part of state change, require outbox/atomic publish pattern.
6. If rollback needs dropped data, treat rollback as restore-based and plan drill before production.
7. If verification fails, stop contract phase and keep expanded schema while fixing forward.
8. If PII is involved, require explicit retention/deletion workflow before merge.
9. If DB engine semantics differ (MySQL metadata locks, NoSQL lazy delete, OLAP mutations), adjust plan and document engine-specific risks.

## Anti-patterns to reject
- Destructive migration first (`DROP/RENAME` before compatibility rollout).
- Schema/code incompatibility during rolling deploy (new code requires schema not yet deployed, or old code breaks on new schema).
- Cross-system dual writes without outbox.
- Big-bang backfill in one long transaction.
- Running migrations from every app pod at startup.
- Assuming down migration always provides safe production rollback.
- Contracting schema before downstream consumers/read models are updated.
- Using stale read replicas as source of truth for migration verification.
- Treating backup existence as proof of recoverability without restore drills.
- Claiming PII deletion while archived/backed-up copies remain recoverable beyond policy.

## MUST / SHOULD / NEVER

### MUST
- MUST deliver migration plans as phased rollout (`Expand -> Migrate/Backfill -> Contract`).
- MUST enforce schema/application compatibility during mixed-version deployments.
- MUST run migrations via one controlled migrator process.
- MUST use session-level lock/time budgets for migration safety.
- MUST design backfills as idempotent, resumable, throttled jobs.
- MUST define objective data verification gates before contract.
- MUST document rollback class and limitations explicitly.
- MUST require restore-tested backup strategy (not backup-only claims).
- MUST define retention/archival/PII deletion semantics for affected data.

### SHOULD
- SHOULD keep production migrations forward-compatible and preferably forward-only.
- SHOULD prefer concurrent/online index and constraint patterns on hot tables.
- SHOULD gate read/write path switches behind feature flags.
- SHOULD monitor replication/CDC lag during migration windows.
- SHOULD include migration observability in PR: lock risk, duration estimate, and dashboards/alerts.
- SHOULD schedule regular DR/restore drills and treat failures as release blockers.

### NEVER
- NEVER start with destructive schema changes on active production paths.
- NEVER rely on synchronized instant deploy of all app instances for correctness.
- NEVER perform DB+broker/API dual writes as a consistency strategy.
- NEVER execute unbounded backfills without checkpoints and kill criteria.
- NEVER assume rollback is safe after data-destructive contract steps.
- NEVER drop old schema paths before verification and traffic cutover are complete.
- NEVER treat untested backups as acceptable reliability posture.

## Review checklist
Before approving migration/schema-evolution changes, verify:

- Rollout and compatibility:
  - Phases are explicit (`expand`, `backfill`, `contract`).
  - Old and new app versions are both compatible during rollout.
  - Feature-flag or equivalent switch strategy is defined.
- DDL safety:
  - Lock/statement timeout policy is defined for migration sessions.
  - Large-table index/constraint operations use online-safe patterns.
  - Migration execution ownership is single-runner and auditable.
- Backfill and verification:
  - Backfill is idempotent and resumable with checkpoints.
  - Throttling and abort thresholds are documented.
  - Verification queries and acceptance thresholds are explicit.
- Rollback and recovery:
  - Rollback class is identified (safe/conditional/restore-based).
  - Irreversible steps are highlighted.
  - Restore plan exists for destructive outcomes.
- Reliability and compliance:
  - Backup vs archive is distinguished.
  - Restore drill schedule exists and is current.
  - Retention/archival policy is explicit.
  - PII deletion flow includes backups/derived stores handling.
- Integration consistency:
  - Outbox or equivalent is used when events/messages are emitted.
  - CDC/replication lag monitoring is present.
  - Downstream contract/version impact is assessed.

## What good output looks like
- Migration plan is staged, measurable, and rollback-aware.
- No step depends on instant fleet-wide deployment.
- Backfill and verification are treated as first-class phases, not footnotes.
- Destructive actions happen last, with explicit recovery limits.
- Backup/restore, retention, and PII deletion are operationally actionable and testable.
- LLM-generated proposals remain conservative, explicit, and safe under production rollout realities.
