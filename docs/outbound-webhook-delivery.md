# Outbound Webhook Delivery

<!-- profile:webhooks-durable:start -->

The `WEBHOOKS=durable` profile is a default-off PostgreSQL acceptance and
delivery engine. It does not discover subscribers, expose a public operator
transport,
provision a receiver, or certify a deployment. Select it only with
`DATABASE=postgres`.

## Accept in the feature transaction

Build the complete immutable fan-out before opening the transaction and call
`postgreswebhook.PrepareAcceptance`. Reuse that prepared value across any
transaction retry. Prefer `Store.AcceptAtomic`: its callback writes the feature
mutation, then the store accepts the complete fan-out as the final SQL operation
and owns commit/rollback plus unknown-commit classification. Existing repository
code may instead call `Store.Accept` in a caller-owned `pgx.Tx`, but it must be
the final SQL operation before immediate commit. The store returns the stable
delivery IDs. Reconstructing the same complete acceptance intent
produces the same IDs, so after a lost commit result use writer-only
`Store.ResolveAcceptance` with that exact intent. Do not change the intent,
read a replica, or perform network I/O inside this transaction.

## Receiver verification

Every attempt is one HTTPS `POST` with the retained body and content type. The
receiver verifies:

- `Webhook-Id`: the stable delivery ID;
- `Webhook-Timestamp`: the attempt's Unix seconds;
- `Webhook-Signature`: one or two space-separated
  `v1,<padded-base64-HMAC-SHA256>` entries, newest key first.

The signed bytes are the exact concatenation
`<delivery-id>.<timestamp>.<body>`. Verify each entry against the active key
and, during an authorized overlap, its predecessor; compare MACs in constant
time. The key reference is deliberately not sent on the wire: the receiver's
out-of-band destination configuration owns the authorized key set. Reject a
timestamp outside the receiver's replay window and deduplicate the stable
`Webhook-Id` for at least the declared delivery-plus-redrive dedup horizon.
Retries and redrives are intentionally at-least-once and may arrive duplicated
or out of order. An HTTP 2xx proves receiver HTTP acceptance, not exactly-once
business processing.

## HTTP and retry contract

| Evidence | Durable classification and next step |
| --- | --- |
| `200`-`299` | `http_accepted`; terminal. |
| `408`, `425`, `429`, and `500`-`599` except `501`/`505` | Ambiguous retryable response; retry only while attempts and the fixed delivery deadline remain. |
| Other HTTP status, including every `3xx`, `501`, and `505` | `http_rejected`; terminal. Redirects are never followed. |
| Failure proved before a request write | Definitely-not-sent; retry within budget. |
| Failure after a possible write | `outcome_unknown`; retry within budget, then terminal/suspended for explicit close or redrive. |
| URL/policy denial, non-public DNS/address evidence, or permanent TLS validation denial | `locally_denied`; terminal until an operator remediates and explicitly redrives. Transient DNS/connect failures remain definitely-not-sent retries. |

Backoff uses decorrelated jitter in `[base, min(cap, 3*previous-delay)]`; the
chosen delay is durable and feeds the next attempt. A valid `Retry-After`
delta-seconds or HTTP-date is normalized to a capped duration; malformed or
past dates are ignored and raw header text is not retained. The later of local
backoff and the normalized hint wins, but a retry at or after the immutable
cycle deadline is terminal exhaustion. DNS, signing, authorization, connect,
TLS, request write, response headers, and bounded response reading share one
total attempt deadline. Response header and non-2xx body evidence are capped;
response content is never stored.

## Configuration and secrets

The worker requires all webhook bounds, PostgreSQL, and an environment-only
`APP__WEBHOOKS__STATIC_SECRETS` manifest. Its strict JSON shape is:

```json
{"revision":2,"entries":[{"owner_scope":"orders","destination_id":"partner-a","key_reference":"orders-v2","secret":"whsec_<base64-32-to-64-bytes>"}]}
```

The document is limited to 1 MiB and 4096 entries; duplicates, unknown fields,
cross-destination key reuse, and trailing JSON are rejected. Each tuple is
owner and destination scoped. Stage rotation by adding the higher
manifest revision before changing durable key authority. Retain the predecessor
through every referenced delivery and overlap horizon, drain incompatible
replicas, and only then remove old bytes. Revisions only move forward. Never put
the manifest in YAML, a config file, logs, metrics, traces, or an action note.
Every accepted destination policy must fit the worker's configured attempt,
header, body, drain, and concurrency ceilings. The engine rejects unsupported
automatic-pause policy rather than accepting a control it does not execute.

## Worker operation

Run `/webhook-worker` as a separate service from the API. `/health/live` is
process-only; `/health/ready` opens only after schema, writer clock, capacity,
secret coverage, reconciliation, and a complete observation succeed. Probes
never send a webhook. The binary loads only its app/HTTP/observability,
PostgreSQL, and webhook configuration sections, so unrelated selected profiles
cannot become hidden startup dependencies. PostgreSQL or authority loss withdraws readiness while
the process continues bounded polling; recovery requires both a successful
maintenance cycle and a new complete observation. SIGTERM closes claim
admission, drains authorized attempts within
the configured bound, then joins diagnostics, telemetry, and PostgreSQL cleanup.

Build and reuse one exact local image:

```bash
make runtime-image-build RUNTIME_IMAGE=service:webhook-test
make migration-validate RUNTIME_IMAGE=service:webhook-test
WEBHOOK_RUNTIME_IMAGE=service:webhook-test REQUIRE_DOCKER=1 \
  go test -p=1 -count=1 -tags=integration ./test \
  -run '^TestWebhookWorkerProcessLifecycle$'
```

## Control, retention, and privacy boundaries

The store exposes transport-neutral destination control, key rotation,
redrive, retention hold/release, privacy erasure, and namespace-retirement
operations. `InspectDelivery` returns one owner-scoped, paginated, redacted
delivery/cycle/attempt/action chain; it exposes no payload, URL, DNS address,
key reference, signature, or request payload. Every `ApplyAction` call requires
the worker's current immutable secret-manifest revision so stale operator
replicas cannot mutate key-dependent state. A
real operator adapter must add authenticated principal, owner scope, roles,
audit policy, and rate limits; opaque IDs never authorize access. Redrive opens
a new cycle without changing the accepted delivery ID or body. Retention and
privacy run in bounded, restartable batches across separately materialized
payload, active-delivery, terminal-summary, attempt, action,
destination-generation, redrive, and receiver-dedup horizons. Legal hold blocks
ordinary cleanup. Worker maintenance automatically resumes pending namespace
retirements; an operator retry is only an idempotent readback, not the recovery
mechanism. Retired destination generations become minimal identity
tombstones after dependent facts expire. Event, namespace, and destination
tombstones are permanent authority against resurrection; external secret
erasure remains a separate controlled operation after no durable reference
remains. Privacy erasure does not free an occupied capacity slot early: its
lease remains a possible-send fence until bounded orphan reconciliation proves
that no unfinalized attempt still owns it.

The reusable action record accepts only bounded identifiers and an empty
protected-note field; an adapter that needs free-form case notes keeps them in
its own access-controlled audit system and passes only a bounded reason and
authority receipt. Mutation requests for a missing target are retained under
the engine's finite maximum action horizon so identical retries return the
original `not_found` or `state_conflict` receipt. An authorized privacy deletion
of an absent event instead records a permanent minimal identity tombstone; it
stores no invented event content and prevents later resurrection.

Operator requests use kind-specific typed payloads (`DestinationStateAction`,
`KeyRotationAction`, `RedriveAction`, `CloseUnknownAction`,
`RetentionHoldAction`, `PrivacyDeletionAction`, and
`NamespaceRetirementAction`), not positional values. The durable action record
stores that non-secret payload and the resulting redrive cycle so an idempotent
replay returns the original receipt.

The transport accepts only public `https` URLs on port `443`, rejects the whole
bounded DNS answer when any address is non-public, rechecks and pins each
candidate at dial, and tries the next authorized answer only when the prior
connection definitely wrote no request. It retains the original hostname for
`Host` and SNI, disables proxies and redirects, and uses the system trust store.
TLS 1.3 is materialized into every newly accepted immutable snapshot when no
minimum is supplied; a destination must set its immutable
`minimum_tls_version` policy to `1.2` for an explicit compatibility exception.
The public-address corpus is pinned to the
[IANA IPv4](https://www.iana.org/assignments/iana-ipv4-special-registry/iana-ipv4-special-registry.xhtml)
and [IANA IPv6](https://www.iana.org/assignments/iana-ipv6-special-registry/iana-ipv6-special-registry.xhtml)
Special-Purpose Address Space registry revision `2025-10-09`; update the
revision and corpus together after registry review.

Required telemetry covers ready/scheduled/in-flight/suspended/terminal/
quarantined/disabled and pending-privacy depth, oldest due age,
claim/maintenance/observation freshness, capacity, clock regression,
HTTP/locally-denied/unknown outcomes, automatic/redrive exhaustion,
reconciliation/privacy/cleanup progress and duration, and bounded error classes. It
contains no payload, URL, address, secret, key reference, signature, response
content, or arbitrary remote error.
The closed `error_class` values are `none`, `ssrf_denied`,
`secret_rotation_failure`, `response_bound`, `reconciliation_conflict`,
`tls_denied`, `timeout`, `canceled`, and `other`.

The additive reference-repair migration keeps new retention/action columns at
`infinity` for rows written by an overlapping old runtime. Restartable bounded
maintenance pages replace those sentinels from each immutable delivery policy;
readiness stays closed while any live row still needs that backfill. Before a
new producer accepts v2/TLS-policy snapshots, drain incompatible old workers
and move capacity to one higher monotone revision so a v1 worker cannot claim
new work.

## Rollout limit

Local PostgreSQL, network, image, and process checks prove only the reusable
pack. They do not prove production capacity, egress, receiver behavior, secret
delivery, migration safety, Railway service creation, SLOs, or provider
configuration. Keep emission disabled until the target-specific authority,
migration, no-row worker, same-writer canary, recovery, and applicable
rotation/privacy checkpoints in the accepted rollout plan are complete.

<!-- profile:webhooks-durable:end -->
