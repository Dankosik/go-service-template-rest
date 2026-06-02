# Data, Postgres, Backups, PITR, And Restore

Status: targeted full-rollout research complete
Date: 2026-06-02

## Question

What production Postgres, migration, backup, PITR, restore, and evidence policy
must specification reopen decide before full Railway rollout can be approved?

## Findings

Live Railway read-only inventory on 2026-06-02 found no
`billing-service-postgres` service in project `empathetic-clarity`
production. Existing sibling Postgres services use Railway's
`ghcr.io/railwayapp-templates/postgres-ssl:18` image, while the older shared
`Postgres` service uses `:17`. This is pattern evidence only; no database was
created or changed.

Repository config keeps Postgres disabled by default in
`env/config/default.yaml:20`. Enabling microlease runtime or worker mode
requires Postgres, service auth, and Redpanda to be enabled together
(`internal/config/validate.go:156`), and enabling balance/usage authority also
requires Postgres, service auth, microlease, and worker readiness when that
guard is active (`internal/config/validate.go:184`).

The production DSN must come from `APP__POSTGRES__DSN`, not YAML or implicit
PG environment variables. `docs/configuration-source-policy.md:63` and
`docs/configuration-source-policy.md:65` require an explicit TCP DSN with host,
port, database, user, non-empty password, and `sslmode`; secrets must remain
outside YAML.

The app deploy profile already owns migration ordering for the HTTP service:
`railway.toml:15` runs `/migrate` before deploy, and the Docker image currently
copies `/migrate` plus `/env/migrations` (`build/docker/Dockerfile:35`). Current
migration files cover versions `000001` through `000004`. When Postgres is
disabled, `cmd/migrate/main.go` prints a no-op message; when enabled, it runs
the repository migrator against `env/migrations`.

Railway backup docs show volume backups can be manual or scheduled daily,
weekly, and monthly. Daily backups are retained for 6 days, weekly backups for
1 month, and monthly backups for 3 months. Restoring a volume backup stages a
new volume mounted at the same path, keeps the previous volume unmounted, and
requires a deploy to apply; restoring also removes newer backups after the
chosen restore point.

Railway PITR docs show Postgres PITR uses pgBackRest WAL archiving to a Railway
bucket. Enabling PITR creates a `Postgres-PITR` bucket, sets `WAL_ARCHIVE_*`
variables, and redeploys Postgres. The restore window starts only after the
first post-enable base backup; it is not retroactive. PITR restore creates a
new sibling Postgres service and never touches the source. Cutover from the
restored fork is manual by swapping connection strings, copying rows, or
replacing the source.

## Evidence Limits

Live variable values, DSNs, bucket credentials, backup inventories, and restore
URLs were intentionally not read or printed. Railway's read-only service config
does not expose enough branch/root-directory detail to approve source topology.

## Handoff Implications

Specification reopen must decide:

- whether the production database is a dedicated service named
  `billing-service-postgres`;
- Postgres image/version policy and volume size baseline;
- exact `APP__POSTGRES__DSN` reference shape with `sslmode`, without exposing
  values;
- migration order and proof, including migration version and dirty-state
  evidence after `/migrate`;
- backup schedule baseline;
- whether PITR is mandatory before enabling money authority;
- restore drill shape: restore to sibling service, verify migration/data sanity,
  and keep cutover manual unless a later approved rollout explicitly authorizes
  a switch.
