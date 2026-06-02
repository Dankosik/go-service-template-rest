# Deployment Surfaces And Sibling Examples

Status: targeted full-rollout research complete
Date: 2026-06-02

## Question

Which repository and Railway deployment surfaces are already usable, and which
ones block a full app+worker+database+broker rollout?

## App Surface

`railway.toml` is the repository deployment-policy source of truth. It selects
Dockerfile builds, `build/docker/Dockerfile`, `/migrate` as the pre-deploy
command, `/health/ready` as healthcheck, `ON_FAILURE` restart policy, and
overlap/drain settings (`railway.toml:10`). It also records a production
replica baseline comment of at least two replicas and 2 vCPU / 2 GiB per
replica (`railway.toml:6`).

`docs/railway-deployment-profile.md` says GitHub-triggered deploy wiring still
requires an operator-managed Railway step to connect the service to the
intended repo and branch, and optionally enable `Wait for CI`
(`docs/railway-deployment-profile.md:47`).

## Worker Surface

The current Dockerfile is not sufficient for a worker service because it does
not build or copy `/billing-worker`. Railway docs allow overriding a service
start command for Dockerfile/image deployments, but that only helps after the
image contains the target binary.

Sibling worker services in production are separate Railway services sourced
from their paired repos. This supports a future `billing-worker` service from
the same repo/image, with no public domain, once the image/start-command gap is
closed. It does not prove worker health, readiness, scaling, or broker lag.

## Dependency Config Surface

Default config disables the protected full-rollout dependencies:

- Postgres disabled;
- service auth disabled;
- Redpanda disabled;
- microlease disabled;
- microlease worker disabled;
- balance/usage authority disabled in `inert_expand`.

Enabling microlease runtime, worker mode, or authority is not an independent
variable flip. Validation requires the dependency set to be enabled coherently,
and Redis must remain disabled for the first microlease target
(`internal/config/validate.go:156`, `internal/config/validate.go:173`).

## Handoff Implications

Specification reopen must keep deployment strategy explicit:

- preserve the existing app service and `railway.toml` app profile;
- require an image change or equivalent approved image strategy before
  `billing-worker` can be deployed;
- decide whether app and worker use the same source/image with different start
  commands or distinct images;
- decide Railway source repo, branch, root/config path, and `Wait for CI`
  expectations before mutation;
- keep all variable proof key-only and source-policy compliant.
