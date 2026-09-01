# Railway Deployment Profile

This repository is a service template. It ships a generic Railway source-build
profile and release-image tooling, but it never connects to a Railway project,
service, environment, domain, or secret.

## Independent delivery paths

Choose one path for each derived service and keep its evidence separate.

### Railway source build

Connect the derived GitHub repository and branch in Railway. Railway builds
`build/docker/Dockerfile` from that source and applies `railway.toml`.

For GitHub-triggered builds, Railway supplies `RAILWAY_GIT_COMMIT_SHA`.
The Dockerfile embeds its first 12 characters as the default application
version, so startup logs identify the source commit without a project-specific
variable. The OCI version and revision labels retain the full Railway commit;
a caller-provided `APP_VERSION` and `VCS_REF` remain authoritative outside
Railway.

The GHCR image published by `.github/workflows/cd.yml` is a different build.
Its signature and attestations do not prove what a Railway source build runs.
Source-build release evidence therefore comes from the selected repository
commit plus Railway's build and deployment records.

### Published image

When repository variable `ENABLE_GHCR_PUBLISH=true`, the CD workflow publishes
a signed GHCR image, CycloneDX SBOM, build provenance, and an immutable
`image@sha256:...` reference. A derived repository may deploy that digest to
Railway or another platform after verifying it. Publication is disabled by
default.

For a PostgreSQL service, first publication is fail-closed. The owner must
confirm that the repository's GHCR container package does not yet exist and set
`MIGRATION_HISTORY_BOOTSTRAP_SHA` to the exact candidate commit. After that
candidate publishes successfully, clear the variable. Every later main or
release publication verifies the signed and attested `migration-history` image,
requires its `/migrations` corpus to be a byte-identical prefix of the
candidate, advances the marker to the verified candidate digest, and only then
promotes public aliases. If the package already exists but the marker is
missing, publication stays blocked for an explicit history-recovery procedure.

This is opt-in operator configuration. Neither `railway.toml` nor the workflow
contains a Railway project identifier or changes a Railway service.

See [CI/CD Production-Ready Checklist](ci-cd-production-ready.md) for consumer
verification commands.

## Repository-owned Railway policy

`railway.toml` is the source of truth only for non-secret settings that Railway
can read from the derived source repository:

- `builder = "DOCKERFILE"`
- `dockerfilePath = "build/docker/Dockerfile"`
- `watchPatterns` covering runtime source, module manifests, migrations, the
  production Dockerfile, `.dockerignore`, and this policy file
- `preDeployCommand = ["/migrate"]` (PostgreSQL profile only)
- `healthcheckPath = "/health/ready"`
- `healthcheckTimeout = 180`
- `restartPolicyType = "ON_FAILURE"`
- `restartPolicyMaxRetries = 5`
- `overlapSeconds = 45`
- `drainingSeconds = 45`

The watch patterns prevent documentation, test, local Compose, and agent-only
changes from starting a Railway source deployment. Keep them conservative:
when a new path can affect the runtime image or deployment policy, add it in
the same pull request.

The application default worst-case shutdown sequence is 35 seconds: the
30-second HTTP shutdown budget (which includes the 15-second
readiness-propagation delay) plus the 5-second bootstrap telemetry flush. The
45-second draining window therefore leaves Railway roughly a 10-second margin
before SIGKILL. If a derived service changes the application shutdown budget,
keep the platform draining window greater than that full sequence and re-run
the runtime-image shutdown check. On platforms other than Railway, configure
the equivalent stop grace explicitly (for example `docker stop --time 45`,
Compose `stop_grace_period: 45s`, or Kubernetes
`terminationGracePeriodSeconds: 50`); default grace periods of 10-30 seconds
SIGKILL the service mid-drain.

Do not use project-specific BuildKit cache mount IDs in the template
Dockerfile. A derived service may add them after it owns and verifies its
Railway service ID.

<!-- profile:webhooks-durable:start -->
## Durable webhook jobs

The image contains `/jobs-worker`, but `railway.toml` does not create or
configure a second Railway service. An adopter must run that binary from the
exact service image, use the same write-home PostgreSQL writer, supply the
worker-only webhook secret manifest, expose only its private diagnostics port,
and preserve the jobs worker stop-grace contract. Freshly initialized services
omit the legacy webhook migrations. An existing deployment that applied them
keeps emission disabled and drains every legacy `/webhook-worker` before
migration 000007. Then require no-row jobs-worker, egress, receiver canary, and
secret-rotation checkpoints with authenticated evidence.
<!-- profile:webhooks-durable:end -->

<!-- profile:grpc:start -->
## Native gRPC networking

The gRPC profile adds a second application listener but does not change
Railway exposure automatically. `railway.toml` still targets the REST port and
uses `/health/ready`; enabling `APP__GRPC__SERVER__ENABLED` neither publishes
the gRPC port nor proves that a public Railway HTTP domain preserves native
gRPC HTTP/2 trailers.

For service-to-service traffic in one Railway environment, bind the gRPC
listener to `::` and call `<service>.railway.internal:<grpc-port>` through
Railway private networking. Choose application TLS when the service contract
requires endpoint identity beyond the private encrypted mesh; otherwise
plaintext remains an explicit deployment trust decision.

For public native gRPC, either prove the current public HTTP path end to end
with unary and streaming trailer checks or configure Railway TCP Proxy. A TCP
Proxy deployment requires application TLS and client hostname verification;
the proxy hostname and port are deployment-owned values and do not belong in
this template. See [Native gRPC](grpc.md) for runtime config and proof.
<!-- profile:grpc:end -->

## Migration contract

- Railway runs one `/migrate` pre-deploy command before promotion.
- The runtime image contains `/migrate` and `/migrations/`; the directory is
  empty until the first owned migration and the migrator then exits as an
  explicit successful no-op after connecting, acquiring the Goose lock, and
  confirming that database state is also empty.
- Application startup does not run migrations.
- CI validates canonical transactional Goose source and append-only review
  history. When migrations exist, it rehearses `up all -> down all -> up all`
  on a disposable Compose database. It always runs the image's migrator, starts
  the production image against the database, checks readiness, and sends
  SIGTERM.
- Overlapping releases require mixed-version-compatible schema changes.
- Destructive or forward-only changes require a staged
  expand/migrate/verify/contract plan and an explicit recovery method.
- A failed pre-deploy command blocks promotion. Fix the migration or
  configuration; do not move migration ownership into application startup.
- The migrator owns fixed `5m` overall, `2m` statement, and `15s` lock budgets.
  The lock budget is also the detached cleanup reserve. Railway
  documents that a failed pre-deploy command blocks deployment and is not
  retried, but does not publish a platform timeout for this command, so do not
  rely on an implicit provider bound.

### Failed migration recovery

A failed migration file is rolled back because every accepted file runs in a
transaction. Earlier files that completed in the same run remain committed and
recorded in `goose_db_version`. The runner has no dirty-bit escape hatch and
never forces a version. It also refuses to run if applied versions are not an
exact prefix of the image source.

1. Keep promotion blocked. Use the safe terminal fields
   `failure.stage`, `failure.context`, `failure.sqlstate`,
   `migration.before`, `migration.after`, and the failed filename to correlate
   the pre-deploy run with PostgreSQL logs. The runner deliberately does not log
   the DSN, SQL text, or wrapped driver error.
2. Inspect the actual schema and Goose state through an approved database
   console:

   ```sql
   SELECT id, version_id, is_applied, tstamp
   FROM goose_db_version
   ORDER BY id;
   ```

3. If the failed file is unchanged, correct the cause — for example a lock,
   statement budget, permissions, or conflicting data — and rerun the same
   image. If SQL must change, append a reviewed corrective migration; never edit
   an already published file.
4. If manual DDL or state-table edits created divergence, restore from the
   service's approved backup/recovery path or execute a separately reviewed
   corrective procedure under exclusive migration ownership. Do not edit
   `goose_db_version` merely to unblock deployment.
5. Rerun `/migrate`, then verify readiness and the service's material durable
   path.

Do not paste a production DSN into shell history. A state-table edit can hide
schema divergence; it cannot repair it.

## Operator-owned configuration

The template deliberately does not choose these values:

- Railway project, environment, service, branch, domain, or region;
- secrets and database connection references;
- replica count, CPU, memory, autoscaling, or spend;
- private reachability for the separate metrics listener when its default
  loopback bind is changed;
- alerting, continuous health monitoring, or an external uptime check;
- release approval, rollback authority, and retention policy.

For source builds, enable Railway `Wait for CI` when promotion must wait for the
repository's push checks. Treat Railway's deployment healthcheck as a
startup/promotion gate, not continuous monitoring.

Set `GOMEMLIMIT` only after choosing a real container memory limit. Leave
headroom for the binary, non-Go allocations, and the operating system; 90-95%
of the container limit is a starting range to validate, not a template default.
Leave `GOMAXPROCS` unset unless measurements justify an override because current
Go derives it from the container CPU limit.

## Rollback

- For a source build, select a previously successful Railway deployment and
  verify its source commit before rollback.
- For an image deployment, restore the previously accepted immutable digest,
  not an unverified mutable tag.
- Application rollback does not undo a database migration. Schema recovery
  follows the migration's declared reversible, forward-only, or destructive
  class.

After any rollback, verify readiness, application version, core dependency
health, and the external user path owned by the derived service.

## Change proof

For changes to this profile:

1. review the `railway.toml` and Dockerfile diff;
2. run `make migration-validate` and any other affected image leaf;
3. confirm the runtime image becomes ready and exits cleanly on SIGTERM;
4. leave project-specific settings and live deployment evidence to the derived
   service's operator.
