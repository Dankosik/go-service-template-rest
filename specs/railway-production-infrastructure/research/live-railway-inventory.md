# Live Railway Inventory And Source Topology

Status: targeted full-rollout research complete
Date: 2026-06-02

## Question

What live Railway topology facts and source assumptions can be used by
specification reopen without mutating resources or exposing secrets?

## Read-Only Inventory

Project and environment:

- project `empathetic-clarity`
  (`58904982-91dd-4780-9b70-699dc9271e18`);
- environment `production`
  (`1c272106-e6be-4547-ae8d-2e7933eca6df`).

Billing app service:

- service `billing-service`
  (`64592463-335e-4be0-a8d1-330438fd61d0`);
- latest deployment `8c61aefe-2f72-411d-995d-f1ac5accf259`;
- deployment status `SUCCESS`;
- latest deployment timestamp `2026-06-02 10:53:54.790 UTC`;
- source repo `Dankosik/go-service-template-rest`;
- builder `DOCKERFILE`;
- health check path `/health/ready`;
- variable count 11, values not read.

Current local git remote also points to
`https://github.com/Dankosik/go-service-template-rest.git`, branch `main`.

Production service list has no `billing-service-postgres`, no
`billing-worker`, and no billing-specific broker service. The only existing
billing-specific service is the HTTP app service.

Sibling production pattern evidence:

- service-specific Postgres services exist for other services and mostly use
  `ghcr.io/railwayapp-templates/postgres-ssl:18`;
- existing worker services such as `pricing-worker`, `notification-worker`,
  and `document-processing-worker` are separate Railway services sourced from
  the paired service repository;
- `gonka-proxy` is in the same production project and has source repo
  `Dankosik/gonka-proxy`.

## Source-Topology Limits

Railway MCP `get_service_config` exposes repo, builder, healthcheck path, and
variable count, but not branch, root directory, or complete service source
settings. CLI read-back confirmed the repo, but not the branch/root-directory
decision. Therefore the branch/root/config-path assumption remains a
specification input requiring explicit Railway UI/API read-back before any
future mutation.

## Handoff Implications

Specification reopen can rely on:

- the existing app service is live and must be preserved unless a reviewed
  implementation ledger authorizes mutation;
- full rollout is additive/reconciliatory relative to the app service because
  dedicated DB, worker, and broker are absent;
- the intended source repo appears aligned between local git and live Railway,
  but branch/root-directory still need explicit approval-grade read-back.

Do not use app-only `tasks.md` as authorization for full rollout mutations.
