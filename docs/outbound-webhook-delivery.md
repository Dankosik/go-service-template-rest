# Outbound Webhook Delivery

<!-- profile:webhooks-durable:start -->

`WEBHOOKS=durable` requires `DATABASE=postgres JOBS=postgres`. It contributes
one `outbound_webhook` definition to `/jobs-worker`; the PostgreSQL jobs pack
owns durable jobs, claims, retries, recovery, concurrency, telemetry, and
shutdown. There is no webhook-specific worker or active delivery schema.

## Feature API

Construct a `postgreswebhook.Dispatcher` from the immutable endpoint manifest.
Before opening the feature transaction, call `Prepare` with one semantic JSON
event and its receiver IDs. Inside the transaction, call `Prepared.Stage` as
the final operation and return its error so the transaction rolls back on a
conflict. Its boolean reports whether the fan-out was inserted or already
existed.

Each receiver becomes one job with a stable `whd_...` identity. Every job also
contains the fingerprint of the complete sorted fan-out, so adding, removing,
or changing a receiver under the same event identity conflicts rather than
partially replaying. Reuse the same `Prepared` value across transaction retries.
After an unknown commit result, `Prepared.ResolveCurrent` reads the writable
primary and reports whether the complete fan-out still exists. This is an
immediate reconciliation aid, not a permanent acceptance ledger: the feature's
business transaction must reject replay after River removes terminal jobs.

The business input is intentionally small:

- owner scope, event ID, event type, occurrence time, and JSON data;
- one or more receiver IDs, up to 1000.

`Data` is limited to 128 KiB. The module emits a canonical JSON body containing
`type`, `timestamp`, and `data`; it does not preserve caller whitespace, accept
non-JSON content, transform receiver-specific schemas, or expose delivery
policy knobs to feature code.

## Endpoints and secrets

`webhooks.endpoints` is a non-secret immutable JSON snapshot:

```json
{"endpoints":[{"owner_scope":"orders","receiver_id":"partner-a","generation":1,"url":"https://partner.example/hooks","active_key_reference":"orders-v1"}]}
```

The jobs worker separately requires the environment-only
`APP__WEBHOOKS__STATIC_SECRETS` value:

```json
{"entries":[{"owner_scope":"orders","receiver_id":"partner-a","key_reference":"orders-v1","secret":"whsec_<padded-base64-32-to-64-bytes>"}]}
```

Both documents reject unknown and duplicate fields, duplicate bindings, invalid
identifiers, and more than 4096 entries. Secrets are receiver-scoped and the
same bytes cannot be reused across receivers. A predecessor key reference in
the endpoint snapshot produces a second signature during a controlled rotation.
Keep referenced keys available until every job using that endpoint generation
is terminal.

## Wire and retry behavior

The handler uses the official Standard Webhooks Go implementation. Every
attempt is an HTTPS `POST` with:

- `Webhook-Id`: the stable job/delivery ID;
- `Webhook-Timestamp`: attempt Unix seconds;
- `Webhook-Signature`: one or two space-separated `v1,<base64-HMAC-SHA256>` values.

The signed and sent bytes are identical. A `2xx` response completes the job.
`408`, `425`, `429`, and retryable `5xx` responses retry; other HTTP responses
are permanent. A valid `Retry-After` becomes a floor under the deterministic
job backoff, capped at 24 hours. A failure after a possible request write is an
ambiguous effect and retries with the same `Webhook-Id`. Receivers therefore
remain responsible for deduplicating that ID.

The fixed job policy allows at most 20 attempts before a four-day deadline,
with exponential backoff from five seconds, stable bounded jitter, a 24-hour
cap, and a 30-second attempt timeout. Process shutdown uses the separate
`http.shutdown_timeout` and `http.grace_period` budgets.

## Endpoint safety

Only public HTTPS destinations on port 443 are accepted. The transport rejects
credentials, queries, fragments, redirects, non-public DNS answers, and private
addresses rechecked at dial. It pins each authorized resolved address while
retaining the original hostname for Host and SNI, disables environment proxies,
compression, keep-alive, and HTTP/2, requires TLS 1.3, bounds response headers,
and closes ignored response bodies without draining them. Deployment network
policy remains defense in depth; application validation is not a firewall.

## Deliberate removals

The former destination/cycle/attempt/capacity/operator ledger, eight retention
horizons, legal hold, privacy and namespace tombstones, automatic pause fields,
public inspection model, and dedicated webhook telemetry are not part of this
minimal product. No production caller consumed those APIs. The business store
owns source-data erasure; deployment owns endpoint and secret lifecycle; the
jobs pack owns retained execution evidence.

Reopen those decisions only for a named legal retention, erasure, authenticated
operator API, customer portal, or per-receiver rate-limit requirement.

Migration `000007_postgres_webhooks_retire.sql` renames the published legacy
relations instead of dropping them, preserving rollback for the previous
worker. It is not mixed-version compatible: stop emission and drain every
dedicated webhook worker before applying it, then start the jobs worker. A
runtime rollback first applies this migration's Down section. The deprecated
relations can be physically removed only after the previous runtime leaves the
rollback window and retained data has an explicit disposition.

Local focused checks:

```sh
go test -vet=off ./internal/infra/postgreswebhook ./internal/config ./cmd/jobs-worker/...
make test-webhook-race
make sqlc-check migration-check
```

These prove the reusable module only. Target endpoint ownership, secret
delivery, egress, capacity, migration execution, and receiver processing remain
deployment evidence.

<!-- profile:webhooks-durable:end -->
