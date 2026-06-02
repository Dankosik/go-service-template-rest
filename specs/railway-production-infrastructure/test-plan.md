# Railway Full Production Infrastructure Test Plan

Status: review-ready planning input
Trigger: validation spans repository checks, image contents, Railway read-backs,
dedicated Postgres/PITR/restore, Kafka-compatible broker/topics/lag,
`billing-worker` readiness, RS256/JWKS service auth, `gonka-proxy` provider
contract, private networking/no-public-metrics, rollback, and secret-free
evidence.

This is not an implementation task ledger. A future reviewed `tasks.md` owns
execution order, task IDs, and final proof.

No validation was run while repairing this plan because this technical-design
session changed only workflow/design documents and did not mutate code,
Railway resources, variables, databases, brokers, domains, or deployments.

## Exit Criteria

The future implementation closeout may claim production billing readiness only
after secret-free proof shows:

- live app baseline was preserved until approved mutation;
- source topology read-back matches approved repo, branch, root, Dockerfile,
  `railway.toml`, and `Wait for CI` posture when GitHub autodeploy is used;
- canonical image contains `/service`, `/migrate`, `/billing-worker`, and
  `/env/migrations`;
- dedicated private Postgres exists and is referenced by key-only
  `APP__POSTGRES__DSN` posture;
- migrations are current and not dirty on the target database;
- backup schedule, manual pre-cutover backup, PITR enablement, first base
  backup, and restore-to-sibling drill are proven;
- restored sibling semantic reconciliation is complete before any cutover;
- private Kafka-compatible broker exists and topic/retention/partition/group
  and lag proof is green;
- `billing-worker` is a private service, uses `/billing-worker`, starts all
  seven roles, passes Postgres/broker probes, drains/shuts down safely, and has
  fresh admission control;
- app `/health/ready` participates in required Postgres and broker readiness;
- service auth verifies RS256/JWKS and rejects missing/invalid scope classes;
- clean `gonka-proxy` provider contract proves JWT/JWKS, scopes, private URL,
  microlease calls, event producers, child-debit lineage, and no legacy
  fallback;
- app and worker have no public domains by default and `/metrics` is not
  exposed publicly;
- rollback proof closes new admission and does not revive forbidden reserve or
  proxy-local money paths;
- all evidence avoids secrets, DSNs, bearer tokens, private keys, JWKS content,
  event payloads, request bodies, dynamic proof URLs, Railway variable values,
  and raw customer/request payloads.

## Repository Proof

Required proof families, selected by changed surface in the future ledger:

- `make guardrails-check`;
- `make fmt-check`, `make lint`, and `make test`;
- `make test-race` for worker/concurrency changes;
- `make test-integration` or Docker equivalent for Postgres/broker/runtime
  behavior when Docker is available and required;
- `make migration-validate` or `make docker-migration-validate` when migration
  or schema proof is in scope;
- `make sqlc-check` when migrations or query sources change;
- `make openapi-check` when service-auth route or contract generation changes;
- `make go-security` and `make secret-scan` for security-sensitive code/config;
- `make docker-build` plus image inspection proving `/service`, `/migrate`,
  `/billing-worker`, and `/env/migrations`;
- `make docker-container-security` for the production image when Docker is
  available;
- `rtk git diff --check` for edited artifacts.

Stronger full parity when time and tooling allow:

- `BASE_REF=origin/main HEAD_REF=HEAD make check-full`;
- `BASE_REF=origin/main HEAD_REF=HEAD make pr-check` when Docker and refs are
  available.

Skipped Docker or GitHub-only checks must be named as limits, not counted as
proof.

## Railway Read-Back Proof

Required app/source read-back:

- project ID and environment ID;
- app service ID and latest deployment ID/status;
- source repo, branch, root directory, config path, Dockerfile path, builder,
  pre-deploy command, healthcheck path/timeout, restart policy, overlap, drain,
  replica count, and resource settings;
- no public domain unless separately approved;
- variable key presence and non-secret mode posture only.

Required worker read-back:

- service ID;
- source repo/branch/root/config path;
- start command `/billing-worker`;
- latest deployment ID/status;
- no public domain;
- replica count and resource settings;
- sanitized logs or bounded metrics proving dependency probes and all seven
  required roles.

Do not call variable-listing tools or print variable values unless a tool can
guarantee key-only output.

## Postgres, Backup, PITR, And Restore Proof

Required:

- service ID/name `billing-service-postgres`;
- private connectivity posture;
- key-only `APP__POSTGRES__DSN` variable reference posture for app and worker;
- sanitized DSN contract proof without the DSN value;
- migration version and dirty-state read-back;
- daily backup schedule read-back;
- manual pre-cutover backup evidence;
- PITR enabled state and first post-enable base backup evidence;
- restore to sibling Postgres service;
- restored migration version and dirty state;
- semantic reconciliation summary covering active microleases, child debits,
  terminal gaps, inbox/outbox, broker offsets, proxy evidence, reconciliation
  cases, and admission-control freshness.

## Broker, Topics, And Lag Proof

Required:

- selected Kafka-compatible service ID/name `billing-service-kafka`;
- private endpoint posture and persistence/storage read-back;
- selected template/service evidence for candidate `kafka`, or reopened design
  if that candidate cannot meet the spec;
- topic names, existence, retention, partition count, and keying policy;
- consumer group `billing-service-microleases`;
- producer identities `gonka-proxy` inbound and `billing-service` outbox;
- lag/backlog buckets for terminal, checkpoint, close, outbox, inbox retry,
  stale reconciliation, and admission-control freshness;
- no credentials, broker URLs with secrets, event payloads, or request bodies in
  artifacts.

## Worker Readiness Proof

Required:

- image contains `/billing-worker`;
- config validation enables coherent Postgres, service auth, Redpanda,
  microlease worker, and default fail-closed authority posture;
- process exits non-zero on dependency or role wiring failure;
- startup evidence shows Postgres probe, broker probe, and required role set;
- task result evidence uses bounded labels for role, result, and reason class;
- shutdown/drain proof shows in-flight task loops stop within configured
  timeout;
- admission-control renewal is fresh and not expired;
- worker disabled/no-op state is never accepted as paid-readiness proof.

## Service Auth And Proxy Proof

Required billing-service proof:

- service auth enabled by key name;
- issuer/audience/JWKS URL key posture;
- RS256-only verifier behavior;
- missing token, invalid `kid`, wrong issuer, wrong audience, and missing scope
  failures return the expected protected-route behavior;
- allowed scopes match `api/openapi/service.yaml`.

Required proxy proof:

- clean checkout or sibling ledger, not dirty draft state;
- JWKS publication route/source and key rotation overlap;
- issuer, audience, subject, `kid`, TTL, and scope behavior by key name and
  sanitized summaries only;
- private Railway internal URL;
- microlease issue/readback/close calls;
- terminal/checkpoint/close producer implementation;
- durable child-debit lineage before external execution;
- no legacy `BILLING_SERVICE_AUTH_KEY` fallback for migrated money authority;
- no operator-adjustment, proxy-local money, Redis spend, process-local reserve,
  or direct per-request reserve fallback for migrated cohorts.

## Network And Metrics Proof

Required:

- `NETWORK_PUBLIC_INGRESS_ENABLED=false` by key-only posture for app and worker
  unless a later approved ingress exception exists;
- no service/custom public domains attached to app or worker by default;
- private service-to-service reachability from proxy to billing-service;
- `/metrics` not publicly exposed;
- any egress allowlist or exception keys are recorded by key name and approved
  metadata only.

## Rollback Proof

Required:

- pre-change app/worker deployment IDs and resource IDs;
- authority/admission close proof before app/worker rollback;
- no new microlease issuance after rollback close;
- worker drain/commit proof;
- broker topics/retention not destructively changed during rollback;
- restored DB cutover remains manual and blocked until semantic reconciliation;
- direct reserve fallback, proxy-local money writer, Redis spend authority, and
  shared-key fallback remain disabled for migrated scopes.

## Privacy Guard

Evidence may include IDs, statuses, key names, non-secret booleans/modes,
health results, migration versions, dirty-state booleans, topic names, retention
summaries, lag buckets, role labels, and sanitized logs.

Evidence must not include secrets, DSNs, bearer tokens, private keys, JWKS
contents, JWTs, event payloads, request bodies, prompts, completions, SSE
chunks, provider payloads, OTLP headers, dynamic proof URLs, Railway variable
values, or raw customer data.
