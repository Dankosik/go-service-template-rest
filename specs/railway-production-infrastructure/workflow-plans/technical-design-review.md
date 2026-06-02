# Railway Full Production Infrastructure Technical Design Review

Phase: technical-design-review
Status: complete
Verdict: CONCERNS eligible for planning with named proof obligations
Review owner: orchestrator
Review type: distinct read-only technical design review
Date: 2026-06-02

## Reviewed Packet

This review covers the full Railway production infrastructure packet:

- `AGENTS.md`
- `docs/spec-first-workflow.md`
- `workflow-plan.md`
- `workflow-plans/technical-design.md`
- `workflow-plans/specification.md`
- `spec.md`
- `design/overview.md`
- `design/component-map.md`
- `design/sequence.md`
- `design/ownership-map.md`
- `design/data-model.md`
- `design/dependency-graph.md`
- `design/contracts/service-auth-and-broker.md`
- `test-plan.md`
- `rollout.md`
- `research/*.md`
- `docs/repo-architecture.md`
- `railway.toml`
- `docs/railway-deployment-profile.md`
- `env/config/default.yaml`
- `env/.env.example`
- `docs/configuration-source-policy.md`
- `build/docker/Dockerfile`
- `docs/build-test-and-development-commands.md`
- current `tasks.md` as historical app-only baseline only

The review also checked current code/config surfaces for config validation,
service auth, worker runtime, OpenAPI route scopes, network policy, Docker image
contents, repository command targets, and read-only sibling `gonka-proxy`
provider-contract evidence.

## Review Scope

This review judges whether planning can create implementation-ready tasks for
the full Railway production infrastructure packet without inventing
architecture, ownership, contract, sequencing, rollout, or validation policy.

This review does not approve `tasks.md`, authorize Railway mutation, deploy,
change variables, create databases, create brokers, create worker services,
enable paid traffic, attach public domains, or approve the current dirty
`gonka-proxy` checkout as provider proof.

## Review Method

The review tried to falsify the packet against:

- canonical scope and proof obligations in `spec.md`;
- source-of-truth ownership across `railway.toml`, `env/migrations`,
  `api/openapi/service.yaml`, config defaults, Dockerfile, and Railway
  read-back obligations;
- dependency direction and runtime boundaries from `docs/repo-architecture.md`;
- service-auth, worker, broker, data, network, rollout, rollback, and
  validation proof surfaces;
- stale app-only artifact isolation.

Local-review rationale: this session is the distinct read-only technical design
review gate requested by the user. No additional subagent was spawned because
the current packet, repo sources, research notes, and sibling read-only evidence
closed the review questions without an unresolved independent lane.

## Findings

### TDR-C01 Broker Candidate, Topic Admin, And Lag Proof

Classification: `proof_obligation`

Evidence:

- `spec.md` selects a private Kafka-compatible broker and approved topic/group
  names, while requiring technical design to choose the template/image, topic
  administration path, partition keys, lag thresholds, and read-back commands.
- `design/overview.md` selects `billing-service-kafka` and Railway template
  candidate `kafka`, with a pre-mutation read-back/reopen gate because no
  verified Kafka template was found.
- `design/contracts/service-auth-and-broker.md` defines topic contracts,
  producer identities, and the green/warning/critical lag classification model,
  but leaves exact numeric thresholds as task-ledger proof values derived from
  runtime capacity, partitioning, and observed baseline.

Resolution:

Planning may proceed only if it front-loads a broker proof checkpoint before any
broker mutation or paid-readiness task. The ledger must require key-only proof of
private persistence, internal endpoint posture, topic admin/read-back command
path, topic existence, retention, partition count/keying policy, consumer group,
and lag/backlog buckets. If the selected candidate cannot prove those properties,
the ledger must stop and reopen specification or technical design as already
recorded in the design.

### TDR-C02 Worker Readiness Is Non-HTTP And Must Be Made Observable

Classification: `proof_obligation`

Evidence:

- `build/docker/Dockerfile` currently builds and copies `/service` and
  `/migrate`, but not `/billing-worker`.
- `cmd/billing-worker` exists and the worker runtime validates seven required
  roles, probes Postgres and broker before ready, and bounds role labels.
- `cmd/billing-worker/internal/bootstrap/run.go` exits successfully when
  `microlease.worker_enabled` is false, so a disabled/no-op worker must never be
  accepted as paid-readiness proof.
- `design/overview.md`, `component-map.md`, `sequence.md`, and `test-plan.md`
  choose no public worker HTTP health surface and require secret-free logs,
  metrics, read-backs, lag/backlog, admission-control freshness, and
  shutdown/drain proof.

Resolution:

Planning may proceed only if worker tasks make image repair precede worker
service creation and carry explicit proof for `/billing-worker`, start command,
no public domain, HTTP healthcheck disabled, all seven roles, dependency probes,
bounded task evidence, admission-control freshness, and shutdown/drain. If the
non-HTTP evidence surface cannot be produced without adding a worker health
endpoint, planning or implementation must stop and reopen technical design
before attaching any worker domain.

### TDR-C03 Source Topology And Railway Resource Settings Are Pre-Mutation Gates

Classification: `proof_obligation`

Evidence:

- `research/live-railway-inventory.md` records current app service, project,
  environment, source repo, builder, health path, and absent billing DB/worker/
  broker, but also records that branch/root/config-path evidence is not
  approval-grade.
- `railway.toml` owns the app Dockerfile, `/migrate`, `/health/ready`,
  restart, overlap, and drain policy, while comments record the app production
  replica/resource baseline.
- The design sets first worker rollout to one replica and requires app/worker
  resource/readiness read-backs before production-readiness claims.

Resolution:

Planning may proceed only if the first executable checkpoint is read-only
preflight: project/environment/service IDs, source repo, branch, root,
`railway.toml` path, Dockerfile path, `Wait for CI` posture when relevant,
domains, replica/resource settings, and variable key families. Mutation must
stop if the read-back differs from the approved target or if app/worker
resource or replica settings require a material design change.

### TDR-C04 Clean `gonka-proxy` Provider Contract Blocks Paid Readiness

Classification: `proof_obligation`

Evidence:

- `spec.md` rejects the current dirty `gonka-proxy` checkout as provider proof
  and requires RS256/JWKS, scopes, private URL, event producers,
  child-debit lineage, and no legacy fallback before paid authority.
- A fresh read-only sibling check found `gonka-proxy` still dirty, with draft
  service-auth/microlease work and no clean committed provider contract evidence
  for the full billing handoff.
- `design/contracts/service-auth-and-broker.md` names the proxy proof boundary
  and keeps legacy `BILLING_SERVICE_AUTH_KEY`, unscoped tokens, proxy-local money
  writers, Redis spend authority, and direct reserve fallback rejected.

Resolution:

Planning may proceed only if paid-readiness and authority-enablement tasks are
gated behind a clean `gonka-proxy` provider contract or sibling implementation
ledger. The billing-service ledger must not treat dirty sibling evidence as
success and must stop before paid authority if proxy JWKS publication, route
scopes, event production, durable child-debit lineage, private URL, or
no-fallback proof is missing.

## Gate Summary

Verdict: CONCERNS eligible for planning.

No `blocks_planning`, `reopens_spec`, or `reopens_design` finding remains in this
review. The design packet is coherent enough for planning because it selects one
target topology, owners, dependency order, failure semantics, rollout order,
rollback posture, and validation surfaces. The concerns above are accepted only
as named proof obligations for planning; they must not become open questions or
implementation-time architecture choices.

## Planning Input Obligations

Planning must carry these obligations into the new full-infrastructure
`tasks.md` and planning/readiness gate:

- `TDR-PO1`: front-load source-topology and live Railway read-back before any
  mutation.
- `TDR-PO2`: repair and inspect the Docker image for `/billing-worker` before
  creating or starting `billing-worker`.
- `TDR-PO3`: prove the selected Kafka-compatible broker candidate, topic admin
  path, topic policy, and lag/backlog evidence before worker/app Redpanda
  readiness or paid authority.
- `TDR-PO4`: prove worker non-HTTP readiness, role set, dependency probes,
  admission-control freshness, and shutdown/drain, or reopen technical design
  before adding a health surface.
- `TDR-PO5`: keep paid authority blocked until a clean `gonka-proxy` provider
  contract or sibling ledger proves RS256/JWKS, scopes, private URL, event
  production, child-debit lineage, and no legacy fallback.
- `TDR-PO6`: preserve secret-free evidence boundaries for Railway variables,
  DSNs, broker credentials, JWTs, JWKS, event payloads, request bodies, dynamic
  proof URLs, and raw customer/request data.
- `TDR-PO7`: preserve fail-closed rollback: no direct reserve fallback, no
  proxy-local money writer, no Redis spend authority, no new microleases after
  admission close, and no restored-DB cutover before semantic reconciliation.

## Reopen Targets

Reopen specification if:

- Railway cannot provide a private persistent Kafka-compatible broker satisfying
  the approved topic, retention, lag, and read-back requirements;
- a clean `gonka-proxy` provider contract cannot satisfy the approved service
  auth, event, lineage, private URL, and no-fallback requirements;
- public billing ingress or public `/metrics` exposure becomes required;
- source-topology read-back changes the approved deployment owner or target
  source assumptions.

Reopen technical design if:

- the Kafka candidate can satisfy the spec only through a different concrete
  template/image/topic administration path;
- worker readiness requires a private HTTP health surface instead of the chosen
  non-HTTP evidence surface;
- app or worker resource/replica settings materially change the design;
- restore, semantic reconciliation, broker lag, proxy proof, or rollback
  sequence needs a different design while preserving the approved spec.

## Next Action

Next phase: planning.

Planning must create a new full-infrastructure `tasks.md` and run the
post-ledger task-review/readiness gate against `spec.md`, the full design
bundle, this technical design review, `test-plan.md`, `rollout.md`, research
notes, and repo-local deployment/config sources. Planning must not approve the
historical app-only `tasks.md`, must not mutate Railway, and must stop before
implementation.

## Session Boundary

Session boundary reached: yes.

Ready for next session: yes.

Next session starts with: planning.
