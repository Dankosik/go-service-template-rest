# Railway Full Production Infrastructure Technical Design

Status: review-ready
Phase: technical-design
Date: 2026-06-02
Owner: orchestrator

## Purpose

This design replaces the stale app-only Railway deployment context with the
full production infrastructure design approved in `spec.md`.

It does not authorize Railway mutation, production paid traffic, planning, or
implementation. It prepares the review packet for the mandatory technical
design review gate.

## Chosen Approach

The rollout remains additive and fail-closed around the live app service:

- preserve the current Railway HTTP app service `billing-service` as the safe
  baseline until a future approved ledger authorizes changes;
- add a dedicated private Railway Postgres service
  `billing-service-postgres` for billing-service money authority;
- use a private persistent Kafka-compatible broker service
  `billing-service-kafka`; Railway template code `kafka` is the selected
  candidate from current read-only template search, with a hard pre-mutation
  read-back gate for private persistence, internal endpoint, topic policy, and
  lag proof;
- repair the canonical Docker image so the same repo/image lineage ships
  `/service`, `/migrate`, `/billing-worker`, and `/env/migrations`;
- run `billing-worker` as a separate private Railway service with start command
  `/billing-worker` and no public domain;
- enable RS256/JWKS service auth for billing-service protected routes, with
  `gonka-proxy` owning JWT signing, JWKS publication, and key rotation;
- keep public billing ingress disabled by default because `/metrics` remains on
  the root router and is not private or protected;
- require dependency-gated rollout, restore drill, broker/topic/lag proof,
  worker readiness proof, clean proxy contract proof, rollback/fail-closed
  proof, and secret-free validation before any paid readiness claim.

## Artifact Index

| Artifact | Status | Trigger / rationale |
| --- | --- | --- |
| `design/overview.md` | review-ready | Entry point and handoff for the full infrastructure design. |
| `design/component-map.md` | review-ready | Required because app, database, broker, worker, auth, proxy, image, and Railway service surfaces change. |
| `design/sequence.md` | review-ready | Required because migration, PITR/restore, broker, worker, proxy, authority, rollback, and fail-closed ordering are planning-critical. |
| `design/ownership-map.md` | review-ready | Required because source-of-truth, provider-contract, generated-code, Railway, and money-authority ownership must stay explicit. |
| `design/data-model.md` | review-ready | Triggered by dedicated production Postgres, migrations, PITR, restore, reconciliation, inbox/outbox, and money-state rollback. |
| `design/dependency-graph.md` | review-ready | Triggered by app/worker/Postgres/broker/proxy dependency shape and source-topology gates. |
| `design/contracts/service-auth-and-broker.md` | review-ready | Triggered by RS256/JWKS, route scopes, proxy handoff, Kafka topic contracts, producer identity, and lag proof. |
| `test-plan.md` | review-ready | Triggered because validation spans repo checks, image contents, Railway read-backs, database restore, broker lag, worker readiness, service auth, proxy proof, rollback, and privacy controls. |
| `rollout.md` | review-ready | Triggered because deploy order, backup/PITR, restore, mixed-version safety, worker drain, authority cutover, and failback are planning-critical. |

The existing `tasks.md` remains the verified app-only implementation ledger
only. It is historical baseline and is not a current implementation handoff for
this full rollout.

## Selected Design Decisions

### Application Service

The app service keeps `railway.toml` as its deployment-policy source of truth:
Dockerfile build, `/migrate` pre-deploy, `/health/ready`, restart policy,
overlap, and drain. Paid readiness later requires read-back of the actual
branch, root directory, Dockerfile path, `railway.toml` path, `Wait for CI`
posture when GitHub autodeploy is used, replica count, and no public domain.

The app is not considered paid-ready only because Railway healthcheck is green.
Paid readiness requires Postgres, broker, worker, service auth, proxy, rollback,
and validation gates to pass.

### Database

Billing-service Postgres authority moves to a dedicated private Railway
Postgres service named `billing-service-postgres`. The future ledger must set
`APP__POSTGRES__DSN` from Railway's secret/reference source, never YAML or raw
artifact text, and must prove the strict DSN contract without printing the DSN.

Daily backups are the baseline. A manual pre-cutover backup, PITR enablement,
first post-enable base backup, restore to a sibling Postgres service, schema
version/dirty-state proof, representative billing-state checks, and semantic
reconciliation are required before restored cutover or paid authority.

### Broker And Topics

The broker target is Kafka-compatible, private, persistent, and owned by
billing-service infrastructure. The application config namespace remains
`redpanda` because the code uses Kafka protocol through `segmentio/kafka-go`.

Selected service name: `billing-service-kafka`.

Selected Railway template candidate: `kafka` from current read-only Railway
template search. Because no verified Kafka template was found, future
implementation must read back the selected template/service before mutation and
reopen specification or technical design if it cannot prove private persistent
KRaft-compatible operation, internal endpoint posture, retention controls, and
lag/read-back commands.

Topic defaults remain:

- `billing.microlease.terminal.v1`;
- `billing.microlease.checkpoint.v1`;
- `billing.microlease.close.v1`;
- `billing.microlease.facts.v1`;
- consumer group `billing-service-microleases`.

Terminal, checkpoint, and close topics retain at least 7 days. Billing facts
retains at least 30 days. First production worker scale is one replica until
partitioning, assignment, idempotency, lag, outbox, and reconciliation proof
allow a higher count.

### Worker

`billing-worker` is a separate Railway service from the same repo and repaired
image lineage. It starts `/billing-worker`, has no public domain, and does not
use a public HTTP health surface.

Worker readiness is proven by a designed secret-free runtime evidence surface:

- process exits non-zero when config loading, Postgres open/probe, broker probe,
  consumer construction, producer construction, or required role validation
  fails;
- startup emits bounded structured logs for dependency probes and required role
  set without payloads or secrets;
- task execution emits low-cardinality role/result/reason evidence through the
  existing `microleaseworker.Observer` seam or equivalent bounded worker log
  surface;
- Railway read-back proves service status, start command, no public domain, and
  replica count;
- broker lag, inbox/outbox backlog, stale reconciliation, and admission-control
  freshness are read back from broker/database evidence, not inferred from
  process liveness.

If a later implementation cannot produce this worker evidence without adding an
HTTP health endpoint, technical design must reopen before a worker domain is
attached.

### Service Auth And Proxy Handoff

Billing-service verifies RS256 JWTs from `APP__SERVICE_AUTH__JWKS_URL` and
enforces route scopes from `api/openapi/service.yaml`. `gonka-proxy` owns
private-key custody, token signing, JWKS publication, key rotation, and exact
Railway variable key names. Shared bearer-key fallback is rejected for migrated
money authority.

Current sibling `gonka-proxy` evidence is dirty and incomplete: it has draft
RS256 signing and microlease allocator files, but no clean JWKS publication
contract, no committed Kafka/Redpanda producer dependency, and the shared
balance client still has a legacy bearer-key path for unscoped operations. The
design therefore treats `gonka-proxy` as a required clean provider-contract
handoff, not as approved current provider evidence.

### Network And Metrics

Default posture is private service-to-service traffic only. The app and worker
must not gain public billing domains by default. `NETWORK_PUBLIC_INGRESS_ENABLED`
must be explicitly false unless a later approved design isolates metrics or
records an explicit public-ingress exception. Public `/metrics` exposure remains
forbidden.

## Rejected Options

| Option | Rejection reason |
| --- | --- |
| Reuse the app-only design as a planning input | It explicitly excluded Postgres, broker, worker, service auth, proxy, paid traffic, and full rollout proof. |
| Reuse the old shared Railway `Postgres` service | It weakens billing-service money ownership and backup/restore evidence for customer-money authority. |
| Treat Kafka absence as an app readiness issue only | Broker absence must close new paid admission and microlease issuance; it is not a harmless app health warning. |
| Deploy worker by start-command override only | The current image lacks `/billing-worker`; the image lineage must be repaired before service deployment. |
| Add public app/worker domains for easy proof | `/metrics` is not private/protected and public ingress requires a separate approved exception. |
| Accept the current dirty `gonka-proxy` checkout as provider contract | It is not clean, lacks JWKS publication and committed event producer proof, and still contains legacy fallback risk. |
| Allow direct reserve or proxy-local money fallback on rollback | The approved billing authority model rejects those paths for migrated scopes. |

## Review Readiness

Technical design review can start from this packet. Planning must not start
until the mandatory review records `PASS` or eligible `CONCERNS`.

Review should explicitly check:

- source-of-truth ownership across `spec.md`, `railway.toml`, `env/migrations`,
  `api/openapi/service.yaml`, `env/config/default.yaml`, Railway read-backs, and
  sibling `gonka-proxy` contract evidence;
- whether the selected broker candidate and worker readiness evidence surface
  are concrete enough for planning without hiding a new design choice;
- whether backup/PITR/restore, semantic reconciliation, and rollback gates
  preserve money authority;
- whether private networking and no-public-metrics policy remain enforceable;
- whether `test-plan.md` and `rollout.md` carry all triggered proof and
  choreography obligations without becoming task ledgers.

## Reopen Conditions

Reopen specification if:

- Railway cannot provide a private persistent Kafka-compatible broker that
  satisfies retention, topic read-back, and lag proof requirements;
- a clean `gonka-proxy` contract cannot satisfy RS256/JWKS, scope, private URL,
  event producer, child-debit lineage, and no-fallback requirements;
- public billing ingress or public `/metrics` exposure is required;
- source-topology read-back shows a different repo, branch, root, Dockerfile,
  or deployment-policy owner than the approved target.

Reopen technical design if:

- the selected Railway Kafka template/service can satisfy the spec but needs a
  different concrete template/image or topic administration path;
- worker readiness cannot be proven through the selected non-HTTP evidence
  surface;
- app or worker replica/resource baselines must change from the recorded
  Railway policy before mutation;
- restore, semantic reconciliation, broker lag, or proxy proof needs a different
  design sequence while preserving the approved spec.

## Boundary

This session stops at technical design. Do not create or approve `tasks.md`, do
not start technical design review in this session, do not implement code, and do
not mutate Railway resources.
