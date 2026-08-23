-- name: ClaimInboundWebhookReceipt :execrows
INSERT INTO inbound_webhook_receipts (
    receipt_id,
    endpoint_id,
    delivery_id,
    body_sha256,
    signed_at,
    payload
) VALUES (
    sqlc.arg(receipt_id),
    sqlc.arg(endpoint_id),
    sqlc.arg(delivery_id),
    sqlc.arg(body_sha256),
    sqlc.arg(signed_at),
    sqlc.arg(payload)
)
ON CONFLICT (endpoint_id, delivery_id) DO NOTHING;

-- name: GetInboundWebhookReceiptByIdentity :one
SELECT
    receipt_id,
    endpoint_id,
    delivery_id,
    body_sha256,
    signed_at,
    received_at,
    payload,
    outcome,
    terminal_reason,
    terminal_at
FROM inbound_webhook_receipts
WHERE endpoint_id = sqlc.arg(endpoint_id)
  AND delivery_id = sqlc.arg(delivery_id);

-- name: GetInboundWebhookReceiptByID :one
SELECT
    receipt_id,
    endpoint_id,
    delivery_id,
    body_sha256,
    signed_at,
    received_at,
    payload,
    outcome,
    terminal_reason,
    terminal_at
FROM inbound_webhook_receipts
WHERE receipt_id = sqlc.arg(receipt_id);

-- name: MarkInboundWebhookHandled :execrows
UPDATE inbound_webhook_receipts
SET
    outcome = 'handled',
    payload = NULL,
    terminal_reason = NULL,
    terminal_at = clock_timestamp()
WHERE receipt_id = sqlc.arg(receipt_id)
  AND outcome = 'pending';

-- name: MarkInboundWebhookQuarantined :execrows
UPDATE inbound_webhook_receipts
SET
    outcome = 'quarantined',
    terminal_reason = sqlc.arg(terminal_reason),
    terminal_at = clock_timestamp()
WHERE receipt_id = sqlc.arg(receipt_id)
  AND outcome = 'pending';

-- name: MarkInboundWebhookFailed :execrows
UPDATE inbound_webhook_receipts
SET
    outcome = 'failed',
    terminal_reason = 'attempts_exhausted',
    terminal_at = clock_timestamp()
WHERE receipt_id = sqlc.arg(receipt_id)
  AND outcome = 'pending';
