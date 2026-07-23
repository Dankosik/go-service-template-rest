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
variable. A caller-provided `APP_VERSION` build argument takes precedence.

The GHCR image published by `.github/workflows/cd.yml` is a different build.
Its signature and attestations do not prove what a Railway source build runs.
Source-build release evidence therefore comes from the selected repository
commit plus Railway's build and deployment records.

### Published image

The CD workflow publishes a signed GHCR image, CycloneDX SBOM, build provenance,
and an immutable `image@sha256:...` reference. A derived repository may deploy
that digest to Railway or another platform after verifying it.

This is opt-in operator configuration. Neither `railway.toml` nor the workflow
contains a Railway project identifier or changes a Railway service.

See [CI/CD Production-Ready Checklist](ci-cd-production-ready.md) for consumer
verification commands.

## Repository-owned Railway policy

`railway.toml` is the source of truth only for non-secret settings that Railway
can read from the derived source repository:

- `builder = "DOCKERFILE"`
- `dockerfilePath = "build/docker/Dockerfile"`
- `preDeployCommand = ["/migrate"]`
- `healthcheckPath = "/health/ready"`
- `healthcheckTimeout = 180`
- `restartPolicyType = "ON_FAILURE"`
- `restartPolicyMaxRetries = 5`
- `overlapSeconds = 45`
- `drainingSeconds = 45`

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

## Migration contract

- Railway runs one `/migrate` pre-deploy command before promotion.
- The runtime image contains `/migrate` and `/migrations/`; the directory is
  empty until the first owned migration and the migrator then exits as an
  explicit successful no-op.
- Application startup does not run migrations.
- When migrations exist, CI rehearses `up -> down 1 -> up 1`. It always runs
  the image's migrator, starts the production image against the database,
  checks readiness, and sends SIGTERM.
- Overlapping releases require mixed-version-compatible schema changes.
- Destructive or forward-only changes require a staged
  expand/migrate/verify/contract plan and an explicit recovery method.
- A failed pre-deploy command blocks promotion. Fix the migration or
  configuration; do not move migration ownership into application startup.

## Operator-owned configuration

The template deliberately does not choose these values:

- Railway project, environment, service, branch, domain, or region;
- secrets and database connection references;
- replica count, CPU, memory, autoscaling, or spend;
- public/private networking and access control for `/metrics`;
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
2. run `make migration-validate` or `make check-full`;
3. confirm the runtime image becomes ready and exits cleanly on SIGTERM;
4. leave project-specific settings and live deployment evidence to the derived
   service's operator.
