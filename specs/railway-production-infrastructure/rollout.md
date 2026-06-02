# Railway Full Production Infrastructure Rollout

Status: review-ready planning input
Trigger: deployment order, dedicated Postgres, backup/PITR/restore, migration
proof, broker/topics/lag, worker service, service-auth/JWKS, proxy handoff,
private networking, authority enablement, rollback, and failback are
planning-critical.

This artifact is choreography context only. It is not an implementation task
ledger and does not authorize Railway mutation.

## Rollout Principles

- Preserve the current app-only `billing-service` deployment until a future
  approved and reviewed ledger authorizes mutation.
- Use additive infrastructure first; do not switch paid authority until
  dependency, restore, proxy, rollback, and validation gates pass.
- Keep services private by default.
- Keep evidence key-only and secret-free.
- Fail closed on missing dependency, stale worker, broker lag, proxy contract
  gap, restore uncertainty, or public-ingress ambiguity.
- Do not reintroduce direct per-request reserve fallback, proxy-local money
  writes, Redis spend authority, shared-key auth, or public `/metrics`.

## R0. Read-Only Preflight

Read back without mutation:

- Railway account, project, environment, and service inventory;
- existing app service ID, deployment ID/status, domains, source repo, builder,
  healthcheck, and variable key families;
- app source branch, root directory, config path, Dockerfile path, and `Wait for
  CI` posture when available;
- current absence or presence of `billing-service-postgres`, `billing-worker`,
  and billing-specific broker service;
- local git state and intended branch.

Stop if source topology or live resource shape differs from the approved target.

## R1. Repository And Image Readiness

Before Railway mutation:

- run guardrails and changed-surface repo checks;
- repair Dockerfile to include `/billing-worker`;
- build the canonical Docker image;
- inspect the image for `/service`, `/migrate`, `/billing-worker`, and
  `/env/migrations`;
- prove canonical Dockerfile path remains `build/docker/Dockerfile`.

Stop before worker service creation if `/billing-worker` is absent.

## R2. Dedicated Postgres And Recovery Setup

Create or verify private `billing-service-postgres`.

Required gates:

- key-only DSN reference posture for app and worker;
- sanitized DSN contract proof;
- daily backup schedule;
- manual pre-cutover backup;
- PITR enablement;
- first post-enable base backup proof;
- restore to sibling Postgres service;
- restored schema and dirty-state proof;
- representative state and semantic reconciliation summary.

Do not enable paid authority until restore proof and reconciliation pass.

## R3. Migration Promotion

Use the app pre-deploy `/migrate` path:

- set Postgres variables by key name only;
- deploy/migrate against the dedicated Postgres target;
- verify migration version and dirty state;
- verify app/worker code remains mixed-version safe under overlap/drain.

Rollback does not blindly roll back schema. Any schema downgrade or data
rollback requires reviewed data plan and reconciliation.

## R4. Broker And Topic Setup

Create or verify private persistent `billing-service-kafka`.

Selected candidate template code: `kafka`.

Required gates:

- internal endpoint posture;
- persistence/storage read-back;
- no public UI/domain unless separately approved;
- topic creation/read-back for terminal, checkpoint, close, and billing facts;
- retention read-back: inbound topics at least 7 days, facts at least 30 days;
- partition count and keying policy;
- consumer group `billing-service-microleases`;
- lag/backlog read-back.

If the candidate cannot prove private persistent Kafka-compatible operation and
lag/read-back requirements, stop and reopen specification or technical design.

## R5. Worker Service Deployment

Create or verify private `billing-worker`:

- same repo/source/image lineage as the app;
- start command `/billing-worker`;
- no public domain;
- one replica for first rollout;
- coherent variables for Postgres, service auth, Redpanda, microlease worker,
  and default fail-closed authority posture;
- no Railway HTTP healthcheck unless technical design is reopened to add a
  private worker health surface.

Required gates:

- process fails on missing dependency or role wiring;
- Postgres and broker probes pass;
- all seven roles are present;
- terminal/checkpoint/close consumers and outbox producer construct;
- worker task evidence is bounded and secret-free;
- shutdown/drain proof passes;
- admission-control freshness is current.

Disabled/no-op worker state is not production readiness.

## R6. App Dependency Readiness

Configure the app in dependency-gated paid-readiness posture:

- Postgres enabled;
- service auth enabled;
- Redpanda enabled;
- Redpanda readiness probe enabled or stricter reviewed proof;
- microlease runtime enabled;
- worker readiness assertion enabled only after R5 passes;
- balance/usage authority remains disabled or inert until R8;
- public ingress disabled.

Required gates:

- `/migrate` has run successfully;
- `/health/ready` participates in required dependencies;
- app replica/resource read-back matches approved baseline;
- no public domain and no public `/metrics`;
- logs show no startup/config/readiness/network-policy errors.

## R7. Proxy Provider Contract

Before paid traffic:

- verify clean `gonka-proxy` provider contract or sibling ledger;
- prove RS256 signing and JWKS publication/rotation;
- prove private billing URL;
- prove route scopes;
- prove microlease issue/readback/close HTTP calls where required;
- prove terminal/checkpoint/close event producers;
- prove durable child-debit lineage and terminal obligation before external
  execution;
- prove no legacy shared-key fallback or proxy-local money writer for migrated
  cohorts.

Current dirty proxy evidence is not sufficient for this gate.

## R8. Authority Enablement

Only after R0-R7 pass:

- move selected cohort from inert/default-closed state to the approved authority
  mode;
- verify no Redis or proxy-local money authority;
- verify admission-control freshness;
- verify broker lag green;
- verify worker readiness and reconciliation backlog green;
- run rollback/fail-closed proof before paid readiness claim.

Any critical dependency degradation closes new paid admission and microlease
issuance.

## R9. Closeout Evidence

Record secret-free evidence:

- service IDs and deployment IDs;
- source topology read-backs;
- image contents proof;
- migration version and dirty state;
- backup/PITR/restore status;
- topic/retention/partition/group/lag summaries;
- worker roles/readiness/drain proof;
- service-auth/proxy proof summaries;
- no-public-domain/no-public-metrics posture;
- rollback proof.

Closeout updates are owned by the future approved ledger, not this design
session.

## Rollback

Rollback is fail-closed:

1. close new paid admission and microlease issuance;
2. record current app and worker deployment IDs;
3. drain worker or stop it only after in-flight processing and offset/commit
   proof is safe;
4. redeploy prior app/worker deployment or restore variables by key name only;
5. do not delete broker topics or shrink retention without reviewed approval;
6. do not switch to restored Postgres until semantic reconciliation passes and
   manual cutover is approved;
7. already allocated child debits must settle, write off, or reconcile through
   billing;
8. direct reserve fallback, proxy-local money writes, Redis spend authority,
   and shared-key auth remain disabled for migrated scopes.

## Reopen Targets

Reopen specification if:

- private persistent Kafka-compatible broker is unavailable;
- clean `gonka-proxy` provider contract cannot meet service-auth/event/lineage
  requirements;
- public billing ingress or public `/metrics` is required;
- source topology differs in a way that changes the approved deployment owner.

Reopen technical design if:

- exact Kafka template/image/path must change while preserving spec decisions;
- worker readiness needs an HTTP/private health surface;
- app/worker resource/replica sizing changes materially;
- restore/reconciliation or rollback sequencing must change while preserving
  fail-closed money authority.
