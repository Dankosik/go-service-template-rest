-- +goose Up
-- profile:inbound-webhooks-standard:start

CREATE TABLE inbound_webhook_receipts (
    receipt_id text COLLATE "C" NOT NULL UNIQUE,
    endpoint_id text COLLATE "C" NOT NULL,
    delivery_id text COLLATE "C" NOT NULL,
    body_sha256 bytea NOT NULL,
    signed_at timestamptz NOT NULL,
    received_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    payload bytea,
    outcome text COLLATE "C" NOT NULL DEFAULT 'pending',
    terminal_reason text COLLATE "C",
    terminal_at timestamptz,
    PRIMARY KEY (endpoint_id, delivery_id),
    CHECK (octet_length(receipt_id) BETWEEN 1 AND 128),
    CHECK (octet_length(endpoint_id) BETWEEN 1 AND 64),
    CHECK (octet_length(delivery_id) BETWEEN 1 AND 256),
    CHECK (octet_length(body_sha256) = 32),
    CHECK (outcome IN ('pending', 'handled', 'quarantined', 'failed')),
    CHECK (
        (outcome = 'pending' AND payload IS NOT NULL AND terminal_reason IS NULL AND terminal_at IS NULL)
        OR (outcome = 'handled' AND payload IS NULL AND terminal_reason IS NULL AND terminal_at IS NOT NULL)
        OR (outcome = 'quarantined' AND payload IS NOT NULL AND octet_length(terminal_reason) BETWEEN 1 AND 64 AND terminal_at IS NOT NULL)
        OR (outcome = 'failed' AND payload IS NOT NULL AND terminal_reason = 'attempts_exhausted' AND terminal_at IS NOT NULL)
    ),
    CHECK (isfinite(signed_at) AND isfinite(received_at)),
    CHECK (terminal_at IS NULL OR isfinite(terminal_at))
);

CREATE INDEX inbound_webhook_receipts_pending_idx
    ON inbound_webhook_receipts (received_at, receipt_id)
    WHERE outcome = 'pending';

-- profile:inbound-webhooks-standard:end

-- +goose Down
-- profile:inbound-webhooks-standard:start

-- +goose StatementBegin
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM inbound_webhook_receipts) THEN
        RAISE EXCEPTION 'inbound_webhook_receipts is not empty';
    END IF;
END;
$$;
-- +goose StatementEnd

DROP TABLE inbound_webhook_receipts;

-- profile:inbound-webhooks-standard:end
