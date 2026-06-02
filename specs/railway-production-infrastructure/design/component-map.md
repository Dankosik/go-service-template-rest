# Railway Full Production Infrastructure Component Map

Status: review-ready
Date: 2026-06-02

## Active Components

| Component | Role In Full Rollout | Change From App-Only Baseline |
| --- | --- | --- |
| Railway app service `billing-service` | Existing HTTP app runtime and protected billing API surface. | Preserved, but later paid readiness becomes dependency-gated instead of app-health-only. |
| `railway.toml` | App deployment policy: Dockerfile, `/migrate`, `/health/ready`, restart, overlap, and drain. | Remains app policy source of truth; branch/root/config-path read-back becomes a pre-mutation gate. |
| `build/docker/Dockerfile` | Canonical image for app, migrator, and worker. | Must be repaired to build/copy `/billing-worker` in addition to `/service`, `/migrate`, and migrations. |
| `cmd/service` | HTTP app process, readiness, network policy, protected routes, and `/metrics` root-router exception. | Runs with Postgres, service auth, Redpanda, microlease, and authority dependencies only after gated variables are ready. |
| `cmd/migrate` | Sole schema-promotion path for app deploys. | Runs against dedicated billing Postgres when enabled; no longer only no-ops. |
| `cmd/billing-worker` | Separate worker process for terminal, checkpoint, close, inbox retry, outbox relay, stale reconciliation, and admission-control renewal roles. | Becomes deployable only after image repair and private worker service creation. |
| Railway Postgres service `billing-service-postgres` | Dedicated private production database for billing-service money authority. | New required resource; old shared Postgres and app-only no-DB posture are rejected. |
| Railway backup/PITR resources | Daily backups, manual pre-cutover backup, PITR bucket/WAL archive, restore-to-sibling proof. | New required data-resilience surface. |
| Railway Kafka-compatible service `billing-service-kafka` | Private persistent broker for terminal/checkpoint/close inbound events and billing facts outbox. | New required resource; selected candidate template code is `kafka` pending pre-mutation read-back. |
| Broker topics and group | Four approved topics and consumer group `billing-service-microleases`. | New rollout-owned admin/read-back surface; app/worker runtime does not create topics. |
| `internal/config` | Validated runtime config snapshot and fail-fast dependency guard. | Existing guards become production gates for Postgres, service auth, Redpanda, microlease, worker readiness, and authority mode. |
| `env/config/default.yaml` | Default-disabled non-secret baseline. | Still default-closed; production variables must override coherently by key name. |
| `env/.env.example` | Key-name examples for app, worker, network, auth, broker, and authority modes. | Guides future Railway variable key families; no values copied into evidence. |
| `internal/infra/postgres` | Postgres pool, repositories, SQLC consumers, migration-backed schema access. | Becomes production authority adapter once dedicated Postgres and migrations are proven. |
| `env/migrations` | Schema source of truth. | Migration version and dirty-state proof become required production read-backs. |
| `internal/infra/redpanda` | Kafka protocol consumers, producers, broker probe, event parsing, quarantine, outbox relay. | Uses Kafka-compatible broker despite `redpanda` config namespace. |
| `internal/app/microleaseworker` | Required worker roles, probes, ready state, task loop, concurrency labels. | Must be wired with bounded secret-free readiness/task evidence through observer/log surface. |
| `internal/infra/http/service_auth.go` | RS256/JWKS verifier, issuer/audience checks, `kid` lookup, route-scope enforcement. | Becomes required for paid authority; shared bearer-key bridge stays rejected. |
| `api/openapi/service.yaml` | Protected route and scope contract authority. | Scope matrix remains runtime authority for service-auth planning and proxy handoff. |
| `gonka-proxy` sibling repo | Provider of service JWTs, JWKS publication, private URL caller, microlease HTTP calls, event producers, child-debit lineage. | Required clean provider contract; current dirty checkout is evidence of gaps, not approval. |
| `docs/build-test-and-development-commands.md` | Repository proof command authority. | Drives future repo, generated, migration, security, container, and integration proof selection. |

## Required Runtime Posture

The production paid-readiness posture must be coherent:

- `APP__POSTGRES__ENABLED=true`;
- `APP__POSTGRES__DSN` set from Railway secret/reference source only;
- `APP__SERVICE_AUTH__ENABLED=true`;
- `APP__SERVICE_AUTH__ISSUER`, `APP__SERVICE_AUTH__AUDIENCE`, and
  `APP__SERVICE_AUTH__JWKS_URL` set by key name only in evidence;
- `APP__REDPANDA__ENABLED=true`;
- `APP__REDPANDA__BROKERS` set to private Kafka-compatible endpoint(s);
- four `APP__REDPANDA__*TOPIC` keys and `APP__REDPANDA__CONSUMER_GROUP` match
  the approved names;
- `APP__FEATURE_FLAGS__POSTGRES_READINESS_PROBE=true`;
- `APP__FEATURE_FLAGS__REDPANDA_READINESS_PROBE=true` or a stricter reviewed
  broker readiness proof;
- `APP__MICROLEASE__ENABLED=true`;
- `APP__MICROLEASE__WORKER_ENABLED=true` only after worker evidence proves all
  required roles and dependencies;
- `APP__BALANCE_USAGE_AUTHORITY__ENABLED=true` only after rollout gate permits
  `internal_cohort`, `migrated`, or another approved authority mode;
- `NETWORK_PUBLIC_INGRESS_ENABLED=false` unless a later approved ingress design
  explicitly changes it.

## Worker Service Posture

`billing-worker` must be a separate private Railway service:

- same repo, branch, root, `railway.toml`/Dockerfile lineage as the app unless a
  later review approves an exact alternative;
- start command `/billing-worker`;
- no public domain;
- Railway HTTP health checks disabled unless technical design reopens and adds
  a private worker health surface;
- one replica for the first production authority rollout;
- non-zero exit on config/dependency/role wiring failure;
- role evidence for `terminal_consumer`, `checkpoint_consumer`,
  `close_consumer`, `inbox_retry`, `outbox_relay`,
  `stale_reconciliation`, and `admission_control_renewal`.

## Broker Service Posture

`billing-service-kafka` must be:

- private to the Railway project/environment;
- persistent with explicit volume/storage posture;
- Kafka protocol compatible with `segmentio/kafka-go`;
- configured or proven with internal endpoint(s) accepted by
  `redpanda.brokers` host:port validation;
- able to create or read back topic existence, partition count, retention, and
  consumer lag without exposing credentials.

No app or worker code currently owns topic creation. Topic administration stays
with the future rollout ledger.

## Intentional Non-Touches

The technical design does not introduce these changes:

- customer top-up, payment-provider, deposit, refund, or PSP webhook behavior;
- Redis, process memory, or proxy-local tables as money authority for migrated
  cohorts;
- public billing domain or public `/metrics`;
- direct cross-service database access from `gonka-proxy` into billing-service
  Postgres;
- generated OpenAPI/proto contract changes beyond consuming the current
  route-scope authority;
- `tasks.md` edits, code edits, Railway deployment, variables, services,
  domains, databases, brokers, or volumes in this phase.

## Planning Constraints

Planning must create implementation work from these components without
inventing new owners:

- image repair before worker service creation;
- dedicated Postgres before migrations/authority enablement;
- backup/PITR/restore proof before paid authority;
- broker and topic proof before worker/app Redpanda readiness;
- worker readiness before authority mode;
- clean proxy contract before paid traffic;
- rollback/fail-closed proof before paid readiness.
