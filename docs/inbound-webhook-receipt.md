<!-- profile:inbound-webhooks-standard:start -->
# Inbound Standard Webhooks receipt

`INBOUND_WEBHOOKS=standard-webhooks` adds `POST /webhooks/{endpoint_id}`. The
sender authenticates the exact raw body with Standard Webhooks headers. One
PostgreSQL commit creates the receipt and its River job; `204` means only that
the signed delivery is durably owned for asynchronous processing.

## Configuration

The service process selects both leaves:

- `inbound_webhooks.endpoints`: non-secret endpoint IDs plus active and optional
  predecessor key references
- `inbound_webhooks.static_secrets`: environment-only `whsec_` secret material

The jobs worker selects only `inbound_webhooks.endpoints` and rejects
`inbound_webhooks.static_secrets` when present.

## Mixed-version rollout

Forward: additive migration; stop and drain every old jobs worker; start only
jobs workers that know `inbound_webhook_receipt` and have complete bindings;
then expose the new HTTP service.

Rollback: block new public ingress and drain the new HTTP service. While the new
worker remains the only jobs-worker version, run this read against the writable
primary:

```sql
SELECT
    NOT pg_is_in_recovery()
        AND current_setting('transaction_read_only') = 'off' AS writer_primary,
    count(*) FILTER (WHERE outcome = 'pending') AS pending_receipts
FROM inbound_webhook_receipts;
```

Only `writer_primary = true AND pending_receipts = 0`, observed after HTTP
drain, permits replacement by an old worker. Any nonzero or unknown result is
roll-forward-only.

A down migration may drop `inbound_webhook_receipts` only when the table is
empty.

Provider registration, ingress/TLS, secret rotation, capacity, and execution
remain external inputs. Local proof cannot establish those facts.
<!-- profile:inbound-webhooks-standard:end -->
