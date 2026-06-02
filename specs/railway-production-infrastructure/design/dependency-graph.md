# Railway Full Production Infrastructure Dependency Graph

Status: review-ready
Date: 2026-06-02

## Runtime Dependency Graph

```text
gonka-proxy
  -> private billing-service app URL
     -> service-auth verifier
        -> gonka-proxy JWKS endpoint
     -> billing-service Postgres
     -> billing-service Kafka-compatible broker readiness

billing-service app
  -> dedicated billing-service-postgres
  -> private billing-service-kafka
  -> service-auth JWKS URL
  -> internal config/network policy

billing-worker
  -> dedicated billing-service-postgres
  -> private billing-service-kafka
     -> terminal/checkpoint/close consumers
     -> billing facts producer
  -> internal config/network policy

/migrate
  -> dedicated billing-service-postgres

backup/PITR/restore proof
  -> billing-service-postgres
  -> restored sibling Postgres
  -> broker/proxy evidence for semantic reconciliation
```

## Deployment Dependency Graph

```text
source-topology read-back
  -> Docker image repair
     -> app deploy
     -> billing-worker deploy

dedicated Postgres
  -> backup/PITR/restore proof
  -> /migrate schema proof
  -> app Postgres readiness
  -> worker Postgres readiness

private Kafka-compatible broker
  -> topic creation/read-back
  -> worker broker readiness
  -> app Redpanda readiness probe
  -> lag proof

service auth/JWKS
  -> protected app route readiness
  -> clean gonka-proxy caller proof

worker readiness + proxy contract + rollback proof
  -> authority mode enablement
```

## Required Dependencies By Surface

| Surface | Required dependencies | Failure semantics |
| --- | --- | --- |
| App default-closed baseline | Railway app service, image, `/health/ready`, network policy | Existing app may stay live without paid readiness. |
| App paid readiness | Postgres, service auth/JWKS, Kafka-compatible broker, microlease runtime, worker readiness, Redpanda readiness probe, private ingress | Fail readiness or keep authority closed when dependency is not ready. |
| `/migrate` | Dedicated Postgres and migrations in image | Dirty/missing migration blocks promotion/authority. |
| `billing-worker` | Image with `/billing-worker`, Postgres, broker, service auth config because validation requires it, all seven roles | Disabled/no-op worker is not readiness; migrated cohorts no-spend/read-only. |
| Broker topics | Private persistent broker, admin/read-back command path | Missing topic/retention/lag proof blocks worker/app Redpanda readiness and paid authority. |
| Service auth | Proxy JWKS, issuer, audience, `kid`, scopes, private URL | Missing JWKS/proxy proof blocks paid readiness; no shared-key fallback. |
| Restore cutover | PITR sibling restore, schema proof, semantic reconciliation, manual approval | Incomplete reconciliation keeps restored sibling proof-only. |
| Rollback | Authority close, worker drain, deployment IDs, data/broker evidence | Rollback fails closed and must not revive forbidden money writers. |

## Pre-Mutation Gates

The future ledger must stop before mutation if any gate fails:

- Railway project/environment mismatch;
- source repo/branch/root/config/Dockerfile read-back mismatch;
- selected Kafka template/service cannot prove private persistence and
  topic/lag read-back;
- app or worker would require public domain proof;
- variable evidence would require printing secret values;
- clean `gonka-proxy` provider contract is unavailable for paid-readiness
  phases.

## Coupling Controls

The design intentionally avoids:

- app code creating broker topics at startup;
- worker depending on public HTTP health;
- proxy reading billing-service database directly;
- billing-service trusting proxy-local reserve/balance state as money truth;
- public `/metrics` as a validation dependency;
- shared global secrets or unscoped bearer tokens for migrated authority.

## Scaling Guard

App replicas may follow the repo policy floor once Railway read-back proves
settings and dependency capacity. Worker replicas remain one for first paid
authority because the worker code sets each role to `MaxConcurrency: 1` and the
three consumers share one consumer group. Scaling above one worker replica
requires planning proof for partitions, assignment, idempotency, lag, outbox
relay concurrency, stale reconciliation, and admission-control renewal safety.
