# Outbound Webhook Delivery

<!-- profile:webhooks-durable:start -->

The `WEBHOOKS=durable` profile is a default-off PostgreSQL acceptance and
delivery engine. It does not discover subscribers, expose an operator API,
provision a receiver, or certify a deployment. Select it only with
`DATABASE=postgres`.

## Accept in the feature transaction

Build the complete immutable fan-out before opening the transaction and call
`postgreswebhook.PrepareAcceptance`. Reuse that prepared value across any
transaction retry. In the caller-owned `pgx.Tx`, write the business mutation
and make `Store.Accept` the final SQL operation before commit. The store returns
the stable delivery IDs; after a lost commit result, use writer-only
`Store.ResolveAcceptance` with the same prepared value. Do not regenerate IDs,
read a replica, or perform network I/O inside this transaction.

## Receiver verification

Every attempt is one HTTPS `POST` with the retained body and content type. The
receiver verifies:

- `Webhook-Id`: the stable delivery ID;
- `Webhook-Timestamp`: the attempt's Unix seconds;
- `Webhook-Signature`: one or two space-separated `v1,<base64 HMAC-SHA256>` values.

The signed bytes are the exact raw sequence
`<Webhook-Id>.<Webhook-Timestamp>.<request-body>`. Do not parse, normalize, or
re-encode the body before verification. Base64-decode each supported `v1`
entry, compute HMAC-SHA256 with the active key and, during an authorized
overlap, its predecessor, then compare MACs in constant time. The wire format
does not carry an internal key reference: overlapping signatures are the
rotation mechanism. Reject timestamps outside a receiver-owned replay window
and deduplicate by `Webhook-Id`. Retries and redrives are intentionally
at-least-once and may arrive duplicated or out of order. An HTTP 2xx proves
receiver acceptance, not exactly-once business processing.

## Configuration and secrets

The worker requires all webhook bounds, PostgreSQL, and an environment-only
`APP__WEBHOOKS__STATIC_SECRETS` manifest. Its strict JSON shape is:

```json
{"revision":2,"entries":[{"owner_scope":"orders","destination_id":"partner-a","key_reference":"orders-v2","secret":"whsec_<base64-32-to-64-bytes>"}]}
```

Each tuple is owner and destination scoped. Stage rotation by adding the higher
manifest revision before changing durable key authority. Retain the predecessor
through every referenced delivery and overlap horizon, drain incompatible
replicas, and only then remove old bytes. Revisions only move forward. Never put
the manifest in YAML, a config file, logs, metrics, traces, or an action note.
Every accepted destination policy must fit the running worker's attempt,
header, response-body, drain, concurrency, delivery-age, and retention caps and
must admit the event's exact content type, schema version, and body size.
Automatic pause is deliberately unavailable: the store rejects the flag,
pause fields, and a pause operator action until a complete policy and engine
are implemented.

## Worker operation

Run `/webhook-worker` as a separate service from the API. `/health/live` is
process-only; `/health/ready` opens only after schema, writer clock, capacity,
secret coverage, reconciliation, and a complete observation succeed. Probes
never send a webhook. PostgreSQL or authority loss withdraws readiness while
the process continues bounded polling; recovery requires a new complete
observation. SIGTERM closes claim admission, drains authorized attempts within
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
redrive, retention, privacy erasure, and namespace-retirement operations. A
real operator adapter must add authenticated principal, owner scope, roles,
audit policy, and rate limits; opaque IDs never authorize access. Redrive opens
a new cycle without changing the accepted delivery ID or body. Retention and
privacy run in bounded, restartable batches. A privacy or namespace tombstone
is permanent authority against resurrection; external secret erasure remains a
separate controlled operation after no durable reference remains.
Migration `000006_postgres_webhook_retention.sql` is the additive upgrade from
the merged baseline. Cleanup independently expires payload, attempt, action,
destination, and key-reference evidence in dependency order; production
rollback is fix-and-roll-forward, never migration `Down` after erasure.

## Rollout limit

Local PostgreSQL, network, image, and process checks prove only the reusable
pack. They do not prove production capacity, egress, receiver behavior, secret
delivery, migration safety, Railway service creation, SLOs, or provider
configuration. Keep emission disabled until the target-specific authority,
migration, no-row worker, same-writer canary, recovery, and applicable
rotation/privacy checkpoints in the accepted rollout plan are complete.

<!-- profile:webhooks-durable:end -->
