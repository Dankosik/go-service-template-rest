# Service Auth, Broker, And Proxy Handoff Contracts

Status: review-ready
Date: 2026-06-02

## Scope

This design-only contract artifact records the service-auth, broker/topic, and
`gonka-proxy` provider handoff constraints needed for the full Railway
infrastructure rollout. Runtime contract authorities remain
`api/openapi/service.yaml`, repository code, future event schema authorities,
and live Railway/broker read-backs.

## Service Auth Contract

Billing-service verifier:

- enabled by `APP__SERVICE_AUTH__ENABLED=true`;
- configured by key names `APP__SERVICE_AUTH__ISSUER`,
  `APP__SERVICE_AUTH__AUDIENCE`, and `APP__SERVICE_AUTH__JWKS_URL`;
- accepts Bearer JWTs signed with RS256 only;
- requires non-empty `kid`;
- verifies issuer and audience;
- fetches JWKS from the configured URL;
- enforces route scopes declared in `api/openapi/service.yaml`.

`gonka-proxy` provider contract:

- owns private-key custody and JWT signing;
- owns JWKS publication and public-key rotation;
- signs with subject `svc:gonka-proxy`;
- uses token TTL no greater than the provider contract maximum;
- publishes a new key before signing with it;
- keeps old keys published through token TTL plus cache/skew allowance;
- proves `kid` rollover without printing private keys, bearer tokens, JWTs, or
  JWKS contents.

Rejected:

- shared static bearer key for migrated money authority;
- unscoped tokens;
- accepting current dirty proxy draft files as production provider proof.

## Scope Matrix

Default proxy scopes for migrated paid authority:

| Capability | Scope |
| --- | --- |
| Account resolve | `billing.accounts.resolve` |
| Balance read | `billing.balances.read` |
| Usage read | `billing.usage.read` |
| Usage reserve/finalize/write | `billing.usage.write` |
| Microlease issue/close | `billing.microleases.write` |
| Microlease readback | `billing.microleases.read` |
| Operation readback/support-safe status | `billing.operations.read` |

`billing.reconciliation.read`, `billing.admin.read`, proxy admin scopes, and
operator-adjustment capabilities are out of scope for migrated paid authority
unless a separate approved specification adds them.

## Private URL Contract

The provider handoff must use Railway private networking in the same project
and environment. The design expects an internal service-to-service URL for the
app service. Evidence records only key names, endpoint posture, and reachability
class; it must not copy dynamic proof URLs, bearer tokens, request bodies, or
payloads.

If a public billing domain is required for proxy reachability or validation,
reopen specification because `/metrics` is not isolated or protected.

## Broker Contract

Broker service:

- service name `billing-service-kafka`;
- Kafka-compatible, private, persistent;
- selected candidate Railway template code `kafka`;
- no public UI/domain by default;
- internal endpoint(s) accepted by `redpanda.brokers` host:port validation;
- credentials stay in Railway secret/reference variables only.

Topics:

| Topic | Producer | Consumer | Minimum retention | Keying expectation |
| --- | --- | --- | --- | --- |
| `billing.microlease.terminal.v1` | `gonka-proxy` | `billing-worker` terminal consumer | 7 days | microlease/child-debit/account-scope ordering |
| `billing.microlease.checkpoint.v1` | `gonka-proxy` | `billing-worker` checkpoint consumer | 7 days | microlease/account-scope ordering |
| `billing.microlease.close.v1` | `gonka-proxy` | `billing-worker` close consumer | 7 days | microlease/account-scope ordering |
| `billing.microlease.facts.v1` | `billing-service` worker outbox relay | downstream approved consumers | 30 days | aggregate/account-scope ordering |

Consumer group:

- `billing-service-microleases`.

Topic administration:

- owned by future rollout ledger;
- app/worker runtime must not silently create topics as a substitute for
  retention, partition, and lag read-back;
- read-back must prove topic existence, retention, partition count, consumer
  group, and lag summary without printing broker credentials.

## Producer Identity Contract

Inbound events accepted by the worker must use producer identity `gonka-proxy`.
Worker outbox events use producer identity `billing-service`.

Missing, malformed, or unapproved producer identity is not a retry-to-success
case. It is quarantine/reconciliation evidence and must not release exposure or
enable new admission.

## Proxy Provider Handoff

Before paid readiness, a clean `gonka-proxy` contract or sibling ledger must
prove:

- JWKS publication endpoint and key rotation policy;
- exact Railway variable key names for signing and private billing URL;
- issuer, audience, subject, `kid`, TTL, and scope behavior;
- private proxy-to-billing URL and no public billing fallback;
- microlease issue, readback, and close HTTP calls where required;
- terminal, checkpoint, and close event production to the approved topics;
- producer identity `gonka-proxy`;
- durable child-debit allocation before external execution;
- terminal obligation durability before external execution returns;
- no legacy `BILLING_SERVICE_AUTH_KEY` fallback for migrated authority;
- no operator-adjustment path, proxy-local money writer, Redis spend authority,
  process-local reserve, or direct per-request reserve fallback for migrated
  cohorts.

Current read-only sibling evidence remains a blocker for paid readiness:

- the checkout is dirty;
- RS256 signing is present only as draft provider evidence;
- `.env.example` still documents the older billing shared-key section;
- focused searches found no committed Kafka/Redpanda producer dependency;
- no clean JWKS publication route was found;
- current shared-balance paths only cover a subset of scopes and retain legacy
  fallback risk.

## Lag And Readiness Contract

Minimum secret-free readiness evidence:

- broker service status and private endpoint posture;
- topic existence, retention, partition count, and consumer group read-back;
- consumer lag buckets for terminal/checkpoint/close topics;
- outbox relay backlog bucket;
- inbox retry backlog bucket;
- stale reconciliation backlog bucket;
- admission-control freshness timestamp/bucket;
- worker role set and dependency probe result labels.

Lag thresholds are planning proof obligations. Technical design selects the
classification model:

- `green`: no critical lag/backlog, admission control fresh;
- `warning`: bounded lag/backlog that does not open new risk;
- `critical`: close new paid admission and microlease issuance until recovery
  proof is green.

The exact numeric thresholds must be task-ledger proof values derived from
runtime capacity, topic partitioning, and observed baseline. If planning cannot
name them without a new architecture decision, reopen technical design.

## Contract Evidence Boundary

Allowed:

- key names;
- scope names;
- `kid` identifiers only when not secret and not copied as token/JWKS content;
- TTL numbers;
- service IDs and statuses;
- topic names, retention, partitions, group names, and lag buckets;
- sanitized error classes.

Forbidden:

- private keys, JWKS contents, JWTs, bearer tokens, DSNs, broker credentials,
  event payloads, request bodies, dynamic proof URLs, raw prompts, completions,
  SSE chunks, provider payloads, and raw Railway variable values.
