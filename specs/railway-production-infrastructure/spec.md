# Billing Service Railway Full Production Infrastructure Rollout Specification

Mode: full orchestrated
Status: approved for technical design with named proof obligations
Date: 2026-06-02
Owner: orchestrator

## Context

This specification reopens
`specs/railway-production-infrastructure/` from the verified app-only Railway
deployment to the full production infrastructure rollout for `billing-service`.

Current live baseline, refreshed read-only on 2026-06-02:

- Railway project: `empathetic-clarity`
  (`58904982-91dd-4780-9b70-699dc9271e18`);
- Railway environment: `production`
  (`1c272106-e6be-4547-ae8d-2e7933eca6df`);
- app service: `billing-service`
  (`64592463-335e-4be0-a8d1-330438fd61d0`);
- latest app deployment timestamp: `2026-06-02 10:53:54.790 UTC`,
  `SUCCESS`;
- source repo read-back: `Dankosik/go-service-template-rest`;
- builder: `DOCKERFILE`;
- health check path: `/health/ready`;
- production service list has no `billing-service-postgres`, no
  `billing-worker`, and no billing-specific broker service.

The earlier app-only `tasks.md`, `design/`, `test-plan.md`, and `rollout.md`
remain historical proof for that narrowed deployment only. They do not
authorize creating databases, brokers, topics, worker services, service-auth
rollout, proxy cutover, paid traffic, public ingress, or live data migration.

This reopened scope is protected-domain work: data, money, service auth,
worker/concurrency, broker/event transport, service-to-service dependencies,
deployment, rollout, rollback, and validation proof all apply. No Railway
mutation is authorized until a future approved and reviewed implementation
ledger explicitly names the exact resource change.

## Scope / Non-goals

In scope for the full rollout specification:

- preserve the current app-only deployment as the safe live baseline until a
  future implementation ledger approves a transition;
- decide the production Railway topology for the HTTP app, dedicated
  `billing-service` Postgres, Kafka-compatible broker and topics, and
  `billing-worker`;
- decide migration ordering, schema-version proof, and mixed-version safety for
  `/migrate`, the app, and the worker;
- require backup, PITR, restore, and semantic reconciliation proof for the
  production billing Postgres service;
- require broker/topic/consumer-group/lag proof for terminal, checkpoint,
  close, inbox/outbox, and billing facts flows;
- require `billing-worker` readiness for terminal consumers, checkpoint
  consumers, close consumers, inbox retry, outbox relay, stale reconciliation,
  and admission-control renewal;
- require scoped service-auth/JWKS and an exact `gonka-proxy` provider-contract
  handoff before any paid cohort can use billing-service as money authority;
- require Railway variables, secret-source policy, private networking, no
  accidental public `/metrics`, and key-only evidence;
- require rollout, rollback, fail-closed, and validation proof before claiming
  production billing readiness.

Out of scope for this specification:

- changing customer-facing top-up, payment-provider, PSP webhook, refund, or
  deposit behavior unless a separate approved billing/payment spec adds it;
- using proxy-local balance rows, in-memory reservations, Redis, or process
  memory as customer-money authority for migrated cohorts;
- enabling public billing ingress while `/metrics` remains on the root router,
  unless a later approved design provides a safe private metrics path or
  explicit ingress exception;
- approving a dirty or draft `gonka-proxy` checkout as the provider contract;
- copying raw secrets, DSNs, bearer tokens, private keys, JWKS contents, event
  payloads, request bodies, prompts, completions, SSE chunks, provider payloads,
  or dynamic proof URLs into artifacts or evidence;
- deploying, configuring, deleting, or changing live Railway resources in this
  specification session.

## Constraints

- `specs/balance-usage-authority-cutover/spec.md` remains the current business
  authority context for migrated paid cohorts: billing-service PostgreSQL is
  the customer-money source of truth, billing-issued microleases are required,
  and direct per-request reserve fallback is rejected for migrated cohorts.
- Redpanda/Kafka is transport, replay, quarantine, and outbox propagation. It
  is not the reserve or no-negative money gate.
- `env/config/default.yaml` and `internal/config/validate.go` make the
  dependency chain explicit: enabling microlease runtime or worker mode
  requires Postgres, service auth, and Redpanda config to be enabled; enabling
  balance/usage authority also requires microlease runtime, worker readiness,
  and Redpanda readiness.
- `railway.toml` is the app deployment-policy source of truth: Dockerfile build
  from `build/docker/Dockerfile`, `/migrate` pre-deploy, `/health/ready`, and
  `ON_FAILURE` restart policy.
- The current Dockerfile builds `/service` and `/migrate` and copies
  `/env/migrations`; it does not build or copy `/billing-worker`.
- Railway private networking supports service-to-service communication inside
  one project environment through internal DNS and without public exposure.
- Railway volume backups can be scheduled daily, weekly, or monthly and are
  restored through the service Backups tab. Railway PITR for Postgres archives
  WAL to a Railway bucket and restores to a new sibling Postgres service; the
  source service is not touched by restore.
- Evidence must stay secret-free. Variable evidence is limited to key names,
  non-secret booleans/modes, IDs, statuses, domains, health results, sanitized
  log summaries, and approved proof command names.

## Decisions

### D1. Current Live App Is Preserved Baseline, Not Full Readiness

The existing `billing-service` Railway app remains the live baseline.

Full infrastructure rollout must not reinterpret the app-only healthcheck as
paid billing readiness. The app may stay default-closed until the future ledger
proves the database, broker, worker, service auth, proxy dependency, rollback,
and validation gates needed for production billing.

### D2. Dedicated Billing Postgres Is Required

The production billing database target is a dedicated Railway Postgres service
named `billing-service-postgres`, or an implementation-time exact equivalent
that is single-tenant to billing-service and explicitly approved by technical
design review. Reusing the old shared `Postgres` service or an app-only no-op
`/migrate` posture is rejected for customer-money authority.

Required database policy:

- the DSN source is only `APP__POSTGRES__DSN`, preferably a Railway reference
  variable to `billing-service-postgres`;
- the DSN must satisfy the repository DSN contract: one TCP target, explicit
  host, port, database, user, non-empty password, and `sslmode`;
- `/migrate` remains the only schema-promotion command for the app service;
- app and worker authority modes cannot start until `/migrate` has run against
  the target and migration version plus dirty-state read-back proves the schema
  is current;
- Postgres must be private to the project environment and must not be exposed
  through public ingress.

Backup and restore policy:

- daily scheduled backups are the minimum baseline, with an additional manual
  pre-cutover backup before any paid authority enablement;
- PITR is mandatory before `internal_cohort`, `migrated`, or any external paid
  authority mode is enabled;
- the first PITR proof is valid only after Railway has taken a post-enable base
  backup;
- restore proof must restore to a new sibling Postgres service, verify schema
  version and dirty state, verify representative billing state, and keep cutover
  manual;
- restored sibling cutover is forbidden until billing reconciles active
  microleases, child debit lineage, terminal gaps, inbox/outbox state, and any
  broker/proxy evidence created after the restore point.

### D3. Kafka-Compatible Broker Strategy Is Selected

The production broker target is a private, persistent Kafka-compatible Railway
service for billing-service. Railway template research found Kafka templates
and no Redpanda template; the billing code uses Kafka protocol through
`segmentio/kafka-go`, so the deployed service may be Kafka-compatible while the
application config namespace remains `redpanda`.

The broker service name is `billing-service-kafka` unless technical design
finds an already-approved Railway naming convention that avoids collision while
preserving billing-service ownership.

The following topic and group names are approved as the billing-service event
contract defaults:

- terminal topic: `billing.microlease.terminal.v1`;
- checkpoint topic: `billing.microlease.checkpoint.v1`;
- close topic: `billing.microlease.close.v1`;
- billing facts topic: `billing.microlease.facts.v1`;
- consumer group: `billing-service-microleases`.

Topic administration is owned by the future rollout ledger, not by current app
runtime code. The ledger must create or verify topics, read back topic
existence and policy, and record lag without exposing broker credentials.
Minimum topic policy for technical design:

- terminal, checkpoint, and close topics must retain at least 7 days of data;
- billing facts must retain at least 30 days of data;
- partitions must be selected with keyed ordering for microlease, child-debit,
  and account-scope flows;
- worker replicas stay at one until partitioning, assignment, idempotency,
  lag, outbox, and reconciliation proof make higher replica counts safe.

Broker absence or degraded lag does not release existing exposure. It closes
new paid admission and new microlease issuance until recovery proof is green.

### D4. `billing-worker` Is A Separate Private Railway Service

The full rollout requires a separate Railway worker service named
`billing-worker` before paid authority can be enabled.

The worker must use the same repository source and canonical Dockerfile lineage
as the app, but the image must be repaired to ship `/service`, `/migrate`, and
`/billing-worker` before any worker service start command can be approved. The
worker service start command is `/billing-worker`.

The worker runtime has seven required roles:

- terminal consumer;
- checkpoint consumer;
- close consumer;
- inbox retry;
- outbox relay;
- stale reconciliation;
- admission-control renewal.

Worker readiness must prove Postgres and broker probes, role presence, bounded
concurrency, signal-aware shutdown, lag/backlog gates, and admission-control
freshness. The worker must not have a public domain. Railway HTTP health checks
for the worker remain disabled unless technical design adds a private health
surface; otherwise readiness proof comes from secret-free logs, metrics, and
read-back evidence that the worker passed its internal probes and started all
required roles.

If `billing-worker` is disabled, exits as no-op, is stale, or cannot prove
admission-control freshness, migrated paid cohorts are no-spend/read-only:
new microlease capacity must not be issued or replenished.

### D5. HTTP App Must Be Private And Dependency-Gated Before Paid Traffic

The HTTP app may move from default-closed to production billing mode only after
the full dependency chain is ready.

Required posture for paid billing readiness includes:

- `APP__POSTGRES__ENABLED=true`;
- `APP__SERVICE_AUTH__ENABLED=true`;
- `APP__REDPANDA__ENABLED=true`;
- `APP__FEATURE_FLAGS__REDPANDA_READINESS_PROBE=true` or an approved stricter
  broker readiness proof;
- `APP__MICROLEASE__ENABLED=true`;
- `APP__MICROLEASE__WORKER_ENABLED=true` where current config validation uses
  that flag as the authority worker-readiness assertion;
- `APP__BALANCE_USAGE_AUTHORITY__ENABLED=true` only after the rollout gate
  permits the selected authority mode;
- `NETWORK_PUBLIC_INGRESS_ENABLED=false` unless a later approved ingress design
  explicitly changes it.

The app must not expose `/metrics` publicly by accident. If external health
proof or proxy reachability requires a public domain, reopen specification or
technical design for metrics isolation, auth, or an explicit ingress exception.

The app replica target remains the `railway.toml` policy baseline of at least
two production replicas. Live Railway settings must be read back before any
production-readiness claim because the current deployment status alone does not
prove replica count.

### D6. Service Auth Is Scoped RS256/JWKS

Production billing authority requires scoped service JWTs verified by
billing-service against a JWKS URL. The legacy proxy
`BILLING_SERVICE_AUTH_KEY` bearer-key bridge is rejected for migrated money
authority.

Ownership:

- billing-service owns JWT verification, issuer/audience/scope enforcement, and
  `APP__SERVICE_AUTH__JWKS_URL` configuration;
- `gonka-proxy` owns signing private-key custody, JWT issuance for its calls,
  JWKS publication for its public keys, and key rotation evidence;
- no shared signing key, static bearer key, or unscoped token is accepted for
  migrated paid authority.

Key rotation must publish a new public key before signing with it, keep the old
key published through at least token max TTL plus cache/skew allowance, and
prove `kid` rollover without exposing private keys, bearer tokens, or JWKS
contents.

`gonka-proxy` default scopes for migrated paid authority are limited to:

- `billing.accounts.resolve`;
- `billing.balances.read`;
- `billing.usage.read`;
- `billing.usage.write`;
- `billing.microleases.read`;
- `billing.microleases.write`;
- `billing.operations.read`.

Proxy admin, reconciliation, or billing admin scopes are out of scope unless a
separate approved spec adds them.

### D7. `gonka-proxy` Provider Contract Blocks Paid Readiness

This specification approves the billing-service infrastructure target and the
exact proxy handoff requirement. It does not approve the current dirty
`gonka-proxy` checkout as a provider contract.

Before paid authority can be enabled, a clean `gonka-proxy` contract or sibling
implementation ledger must prove:

- JWKS publication, key rotation, issuer, audience, subject, token TTL, `kid`,
  and exact Railway variable key names;
- private proxy-to-billing URL using Railway internal networking;
- route scopes matching D6;
- microlease issue, readback, and close HTTP calls where required;
- terminal, checkpoint, and close event production to the approved topics with
  producer identity `gonka-proxy`;
- durable proxy child-debit lineage before external execution;
- no legacy `BILLING_SERVICE_AUTH_KEY` fallback for migrated money authority;
- no operator-adjustment path, proxy-local money writer, Redis spend authority,
  process-local reserve, or direct per-request reserve fallback for migrated
  cohorts.

If the clean provider contract cannot meet these requirements, reopen
specification before technical design or planning claims production billing
readiness.

### D8. Railway Variables And Evidence Are Key-Only

The future ledger must set and read back Railway variables by key name and
non-secret posture only.

Required key families include:

- app/process identity: `APP__APP__ENV`, `APP__APP__VERSION`,
  `APP__HTTP__ADDR`, `APP__LOG__LEVEL`;
- Postgres: `APP__POSTGRES__ENABLED`, `APP__POSTGRES__DSN`;
- service auth: `APP__SERVICE_AUTH__ENABLED`,
  `APP__SERVICE_AUTH__ISSUER`, `APP__SERVICE_AUTH__AUDIENCE`,
  `APP__SERVICE_AUTH__JWKS_URL`;
- Redpanda/Kafka: `APP__REDPANDA__ENABLED`, `APP__REDPANDA__BROKERS`,
  topic keys, consumer group, and healthcheck timeout;
- readiness flag: `APP__FEATURE_FLAGS__REDPANDA_READINESS_PROBE`;
- microlease and authority mode keys from `env/.env.example`;
- `NETWORK_PUBLIC_INGRESS_ENABLED` and any approved `NETWORK_*` egress or
  ingress exception keys;
- observability exporter keys only when the endpoint and headers policy is
  approved.

Secret values, DSNs, bearer tokens, private keys, JWKS documents, OTLP headers,
and dynamic proof URLs must not be copied into artifacts.

### D9. Railway Source Topology Is A Pre-Mutation Proof Gate

The target source topology is:

- existing app service source repo: `Dankosik/go-service-template-rest`;
- target branch: `main`;
- target root directory: repository root;
- deployment policy file: `railway.toml`;
- canonical Dockerfile: `build/docker/Dockerfile`;
- app pre-deploy command: `/migrate`;
- app health path: `/health/ready`;
- worker service source: same repo, branch, root, and image lineage, with
  start command `/billing-worker`;
- Railway `Wait for CI` enabled when GitHub-triggered deploys are used.

Read-only Railway service config currently proves repo, builder, health path,
variable count, environment, and status, but not branch or root directory.
Therefore source topology is not an approval blocker for this spec, but it is a
mandatory pre-mutation proof gate for the future implementation ledger. If
read-back shows a different branch, root, Dockerfile, config path, or deploy
policy owner, stop and reopen specification or technical design before
mutation.

### D10. Rollout Must Be Dependency-Gated And Fail Closed

The rollout order must be dependency-gated:

1. verify live Railway baseline, source topology, and no unintended public
   ingress;
2. create or verify dedicated Postgres;
3. enable backups and PITR, then perform restore proof to a sibling service;
4. run migrations and verify schema version plus dirty state;
5. create or verify the Kafka-compatible broker and topics;
6. repair the image to include `/billing-worker`;
7. deploy the worker service private and disabled or fail-closed until
   dependencies pass;
8. deploy app dependency variables and private service auth;
9. prove protected app, worker, broker, and database readiness;
10. prove `gonka-proxy` provider behavior and no legacy fallback;
11. enable authority mode only for approved cohort state;
12. run rollback/fail-closed proof before any paid readiness claim.

Rollback must fail closed:

- no direct per-request reserve fallback;
- no proxy-local balance mutation for migrated account scopes;
- no new microleases after rollback admission is closed;
- already minted valid microleases may be used only within remaining durable
  child cap until debit cutoff, and only while proxy durable terminal
  submission plus billing reconciliation remain healthy;
- already allocated child debits must settle, write off, or reconcile through
  billing;
- expired, revoked, stale, or over-cap authority cannot authorize new child
  debits;
- restored database cutover remains closed until semantic reconciliation proves
  active exposure, terminal gaps, inbox/outbox, and broker/proxy evidence are
  safe.

## Open Questions / Assumptions

- [defer_to_design] Technical design must choose the exact Railway
  Kafka-compatible service template/image, topic administration command path,
  partition keys, lag thresholds, and broker backup/retention read-back
  commands while preserving D3.
- [defer_to_design] Technical design must choose the exact worker readiness
  evidence surface. It may keep worker HTTP health disabled only if logs,
  metrics, and read-back proof are strong enough for Railway production
  operations.
- [defer_to_design] Technical design must decide exact app and worker Railway
  service settings, resource sizes, region/replica settings, and shutdown/drain
  proof while preserving the baseline constraints above.
- [reopen_spec_if_false] If Railway cannot provide a private persistent
  Kafka-compatible broker that meets D3 retention, topic read-back, and lag
  proof requirements, reopen specification.
- [reopen_spec_if_false] If the clean `gonka-proxy` provider contract cannot
  satisfy D6 and D7, production paid readiness is blocked and specification must
  reopen or route to a separate proxy contract specification.
- [reopen_spec_if_false] If future Railway read-back shows the app or worker
  source topology does not match D9, stop before mutation and reopen
  specification or technical design.
- [requires_user_decision] Any public billing domain or public `/metrics`
  exposure requires explicit approval unless a later approved design isolates
  metrics privately first.

## Clarification Gate

Formal `spec-clarification-challenge` was required because this is full
orchestrated protected-domain work.

Gate status: complete; reconciled with concerns.

Lanes: five read-only `challenger-agent` lanes using
`spec-clarification-challenge`.

Lenses:

- scope and spec coherence;
- domain invariants and edge cases;
- architecture ownership and dependency boundaries;
- API, data, compatibility, and source-of-truth consequences;
- security, reliability, delivery, and validation proof.

Scoped-down rationale: none for the initial challenge; the broad protected
domain lens set was used.

Fan-in resolution:

| Lens | Strongest finding | Resolution |
| --- | --- | --- |
| Scope/spec coherence | On-disk spec still described the pre-research blocked state. | Repaired status, handoff, and historical app-only artifact boundary. |
| Domain invariants | Restore, worker-disabled, broker-degraded, rollback cutoff, and proxy cutover edges needed explicit money-safety semantics. | Added D2, D3, D4, D7, and D10 constraints. |
| Architecture ownership | DB owner, broker strategy, worker image topology, JWKS owner, and source topology were under-decided. | Added D2, D3, D4, D6, D8, and D9. |
| API/data/compatibility | Proxy contract, scope matrix, topic defaults, Postgres authority, and source topology needed approval-safe classification. | Added D6, D7, D3, D2, and D9. |
| Security/reliability/delivery/validation | Backup/PITR/restore, readiness, auth proof, rollback, scaling, and validation evidence needed concrete policy. | Added D2, D4, D5, D8, D10, and the validation matrix. |

Follow-up clarification: complete after reconciliation. The scoped follow-up
found no surviving approval-changing question and confirmed that the spec can
hand off to technical design while Railway mutation and production paid
readiness remain blocked behind later approved design, ledger, and proof gates.
Rerun clarification only if technical design changes an approval-level decision,
weakens a proof obligation, or pre-mutation/source/provider evidence falsifies a
recorded reopen condition.

## Task Breakdown / Handoff Link

No implementation ledger is approved for the full infrastructure rollout.

Existing `tasks.md` is the verified app-only ledger only and must not be used
to deploy or mutate full billing infrastructure.

Next phase: technical design reopen for the full production infrastructure
scope. The design bundle must replace or repair the stale app-only `design/`,
`test-plan.md`, and `rollout.md` context and must receive a fresh technical
design review before planning.

## Validation

Specification evidence used in this reopen:

- read repository workflow rules and specification skills;
- read the targeted research fan-in and all six research notes under
  `specs/railway-production-infrastructure/research/`;
- read `railway.toml`, `docs/railway-deployment-profile.md`,
  `env/config/default.yaml`, `env/.env.example`,
  `docs/configuration-source-policy.md`, `build/docker/Dockerfile`, and
  `docs/build-test-and-development-commands.md`;
- ran read-only Railway sanity checks for the current app service config,
  service list, and environment status;
- ran five read-only formal clarification challenge lanes and reconciled their
  findings into this specification.

Forward proof obligations for later phases:

| Claim | Minimum secret-free proof |
| --- | --- |
| Live baseline preserved | Service IDs, environment ID, deployment status, source repo, builder, health path, and no billing DB/worker/broker read-back. |
| Source topology safe | Branch, root directory, Dockerfile path, `railway.toml` policy, and `Wait for CI` read-back before mutation. |
| Dedicated Postgres ready | Service ID/name, private connectivity, key-only DSN variable posture, migration version and dirty state, backup schedule, PITR state, and restore drill to sibling. |
| Restore safe for money | Restored schema state plus active exposure, terminal gap, inbox/outbox, broker/proxy evidence, and manual cutover approval. |
| Broker ready | Service ID/name, private endpoint posture, topic names, topic policy read-back, consumer group, lag summary, and credential-free broker proof. |
| Worker ready | Service ID/name, image contains `/billing-worker`, start command, no public domain, roles started, dependency probes passed, shutdown/drain proof, and admission-control freshness. |
| App ready | `/health/ready`, Postgres readiness, broker readiness or stricter proof, private ingress posture, replica read-back, and no public `/metrics`. |
| Service auth ready | Key names only, issuer/audience/JWKS URL posture, `kid` rotation proof, scope matrix, and no token/key/JWKS value capture. |
| Proxy ready | Clean provider contract or sibling ledger, private URL, scopes, JWKS publication, event producers, child-debit lineage, no shared-key fallback, and no proxy-local money writer for migrated scopes. |
| Rollback ready | Fail-closed mode proof, no direct reserve fallback, no new microlease issuance after rollback close, cutoff/cap behavior, and reconciliation of allocated child debits. |
| Repo checks | Scope-matched commands from `docs/build-test-and-development-commands.md`, including guardrails, migration rehearsal, generated drift, OpenAPI, integration, security, and container checks as triggered by design/tasks. |

No code, config, generated artifacts, Railway services, Railway variables,
databases, brokers, domains, volumes, or deployments were mutated in this
specification session.

## Outcome

Specification approved for the next technical design phase.

This approval is not production readiness and not mutation authorization. Paid
billing readiness remains blocked until a future approved design, fresh
technical design review, approved task ledger, and secret-free validation prove
the database, broker, worker, service auth, proxy contract, private networking,
rollback, and validation gates above.
