# Railway Deployment Profile

This document defines the repository-managed Railway deployment policy baseline for this service template.

## Source Of Truth

- Deployment policy source of truth: `railway.toml`
- Owners: Platform + Service Owner
- Secret boundary:
  - `railway.toml` contains only non-secret deployment policy fields.
  - Secrets remain in Railway environment variables.

## Canonical Build Path

- Production build path is Dockerfile-only.
- Canonical Dockerfile: `build/docker/Dockerfile`
- Railway compatibility rule:
  - do not use BuildKit cache mounts (`RUN --mount=type=cache,...`) in the canonical Dockerfile unless mount IDs are explicitly Railway-formatted and validated for the target service.
- CI/CD parity requirement:
  - `.github/workflows/cd.yml` must build images with `docker build -f build/docker/Dockerfile`.
  - `railway.toml` must keep `builder = "DOCKERFILE"` and `dockerfilePath = "build/docker/Dockerfile"`.

## Policy Baseline

`railway.toml` tracks the current hard baseline:

- `preDeployCommand = ["/migrate"]`
- `healthcheckPath = "/health/ready"`
- `healthcheckTimeout = 180`
- `restartPolicyType = "ON_FAILURE"`
- `restartPolicyMaxRetries = 5`
- `overlapSeconds = 45`
- `drainingSeconds = 30`

Migration baseline:

- Railway runs one pre-deploy migrator before promoting a new release.
- The runtime image must ship `/migrate` and `/env/migrations/`.
- Normal app startup must not own migrations.
- Same-deploy schema changes must remain mixed-version compatible while Railway overlap is enabled.
- Destructive or contract-only schema changes require a staged expand/migrate/verify/contract rollout.
- A failed pre-deploy migration blocks promotion; fix the migration or configuration and redeploy while the old release remains active. Do not move migration ownership into app startup as a workaround.

Release evidence baseline (tracked in comments and rollout evidence):

- production replica floor: `>=2`
- per-replica baseline: `2 vCPU / 2 GiB`

## GitHub Autodeploy Prerequisites

The template can encode the migration hook in `railway.toml`, but GitHub-triggered deploy wiring still has one operator-managed step in Railway:

- connect the service to the intended GitHub repository and branch;
- enable Railway `Wait for CI` when deploys should wait for push-triggered GitHub workflows.

The repository already provides the required push-triggered CI workflow in `.github/workflows/ci.yml`, so `Wait for CI` can be enabled without adding a second deploy pipeline.

## Governance And Drift Policy

- Policy changes must be PR-reviewed and traceable in git history.
- `railway.toml` owns non-secret Railway policy.
- The runtime Dockerfile and release workflow own `/migrate`, migration assets,
  and the canonical image build path.
- GitHub Rulesets or organization policy own required status contexts; review
  them when CI jobs change.
- UI-only Railway policy edits are non-compliant until reconciled back into `railway.toml` via PR.

## Change Procedure

1. Update `railway.toml` in a PR.
2. Run `make check-full` when the claim includes migration and runtime-image
   readiness, or run and report the narrower affected gates.
3. Include rollout evidence in PR/release packet:
   - `railway.toml` diff,
   - linked review trail,
   - active settings snapshot from Railway.
4. For numeric policy changes (`180/45/30/5`) or baseline floor/caps, reopen spec before merge.
