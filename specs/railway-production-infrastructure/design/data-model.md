# Railway Full Production Infrastructure Data Model

Status: review-ready
Date: 2026-06-02

## Scope

This artifact defines the production data and recovery posture for the approved
Railway full infrastructure rollout. It does not introduce new schema decisions
beyond the existing migration-backed billing model; it defines how that model
is hosted, migrated, backed up, restored, reconciled, and proven before paid
authority.

## Production Database Target

| Field | Design |
| --- | --- |
| Service name | `billing-service-postgres` |
| Scope | Dedicated private Railway Postgres for billing-service only |
| Public exposure | None |
| DSN source | `APP__POSTGRES__DSN` from Railway secret/reference source only |
| Runtime consumers | App `/migrate`, app Postgres repositories, `billing-worker` Postgres repositories |
| Schema source | `env/migrations` |
| Query source | `internal/infra/postgres/queries` and generated SQLC output |
| Cutover posture | Manual, gated, fail-closed |

The future implementation ledger must prove the repository DSN contract without
printing the DSN: one TCP target, explicit host, port, database, user,
non-empty password class, and `sslmode`.

## Migration Model

`/migrate` remains the only schema-promotion command for the app deploy path.
Normal app startup and worker startup must not run migrations.

Required production proof:

- canonical image contains `/migrate` and `/env/migrations`;
- Railway app pre-deploy command is `/migrate`;
- migration version read-back matches the latest repository migration;
- dirty state is false;
- migration rehearsal is run through the repository-owned command path selected
  by `docs/build-test-and-development-commands.md`;
- same-deploy schema changes remain mixed-version compatible with
  `railway.toml` overlap/drain.

Current migration-backed model includes billing money, ledger, reconciliation,
microlease, child-debit, checkpoint, inbox, outbox, and admission-control
surfaces. Production readiness depends on proving that existing model against
the target database, not on creating a new schema inside this technical design
phase.

## Backup, PITR, And Restore

Minimum baseline:

- daily scheduled backups;
- manual pre-cutover backup before any authority enablement;
- PITR enabled before `internal_cohort`, `migrated`, or any external paid
  authority mode;
- first PITR proof only after Railway has taken a post-enable base backup;
- restore proof to a new sibling Postgres service, not in-place overwrite of
  the source.

Restore proof must verify:

- restored service identity and private posture;
- restored migration version and dirty state;
- representative account, ledger, spending microlease, child debit, checkpoint,
  inbox, outbox, reconciliation, and admission-control rows by support-safe
  summaries only;
- no copied DSN, credentials, request bodies, payloads, or raw customer data in
  artifacts.

## Semantic Reconciliation Before Restored Cutover

Restored sibling cutover is forbidden until billing reconciles all evidence
created after the restore point:

- active microleases and available child cap;
- allocated child debits and terminal obligations;
- terminal, checkpoint, and close gaps;
- inbox rows and retry ownership;
- outbox rows and published/retry state;
- broker offsets and consumer lag;
- proxy child-debit lineage and terminal publication evidence;
- admission-control freshness;
- reconciliation cases and manual-review state.

If semantic reconciliation is incomplete, the restored database remains a proof
artifact only. Paid authority stays closed.

## Failure And Rollback Data Semantics

| Condition | Data posture |
| --- | --- |
| Broker degraded or lag critical | Close new paid admission and microlease issuance; keep existing exposure reconciled. |
| Worker disabled/no-op/stale | Migrated cohorts are no-spend/read-only; do not infer readiness from service liveness. |
| PITR restore selected | Restore to sibling, verify schema/data, reconcile semantics, then require manual cutover approval. |
| App rollback | Do not roll back database schema or revive legacy money writers without reviewed data plan. |
| Authority rollback | No direct reserve fallback, no proxy-local money writes, and no new child debits after admission closes. |
| Existing child debits | Must settle, write off, or reconcile through billing-service authority. |

## Data Ownership

Billing-service Postgres is the source of truth for migrated money state.
Redpanda/Kafka is transport/replay/quarantine/outbox propagation, not the
reserve or no-negative gate. `gonka-proxy` may hold durable child-debit lineage
needed for external execution, but it does not own customer-money balances for
migrated scopes.

## Planning Proof Obligations

Planning must carry tasks for:

- dedicated Postgres read-back and DSN key-only posture;
- migration version and dirty-state proof;
- backup schedule and manual pre-cutover backup proof;
- PITR enablement and first base-backup proof;
- restore-to-sibling drill;
- semantic reconciliation queries/checks;
- rollback data posture proof;
- secret-free evidence capture.
