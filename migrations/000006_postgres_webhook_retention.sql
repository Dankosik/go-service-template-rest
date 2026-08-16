-- +goose Up

ALTER TABLE webhook_destinations
    ADD COLUMN destination_retained_until timestamptz,
    ADD COLUMN key_references_retained_until timestamptz,
    ADD COLUMN key_references_erased_at timestamptz;

UPDATE webhook_destinations
SET destination_retained_until = created_at + ((policy->'horizons'->>5)::double precision / 1000000000) * interval '1 second',
    key_references_retained_until = created_at + ((policy->'horizons'->>6)::double precision / 1000000000) * interval '1 second';

ALTER TABLE webhook_destinations
    ALTER COLUMN active_key_reference DROP NOT NULL,
    ALTER COLUMN destination_retained_until SET NOT NULL,
    ALTER COLUMN key_references_retained_until SET NOT NULL,
    ADD CHECK (isfinite(destination_retained_until) AND isfinite(key_references_retained_until)),
    ADD CHECK ((active_key_reference IS NULL) = (key_references_erased_at IS NOT NULL)),
    ADD CHECK (key_references_erased_at IS NULL OR isfinite(key_references_erased_at));

ALTER TABLE webhook_events
    ADD COLUMN payload_retained_until timestamptz,
    ADD COLUMN payload_erased_at timestamptz;

UPDATE webhook_events e
SET payload_retained_until = e.accepted_at + COALESCE((
    SELECT max((d.policy_snapshot->'horizons'->>0)::double precision / 1000000000)
    FROM webhook_deliveries d
    WHERE d.owner_scope = e.owner_scope AND d.business_event_id = e.business_event_id
), 0) * interval '1 second';

ALTER TABLE webhook_events
    ALTER COLUMN body DROP NOT NULL,
    ALTER COLUMN payload_retained_until SET NOT NULL,
    ADD CHECK (isfinite(payload_retained_until) AND (payload_erased_at IS NULL OR isfinite(payload_erased_at))),
    ADD CHECK ((body IS NULL) = (payload_erased_at IS NOT NULL));

ALTER TABLE webhook_deliveries
    ADD COLUMN active_retained_until timestamptz,
    ADD COLUMN terminal_retained_until timestamptz,
    ADD COLUMN attempts_retained_until timestamptz,
    ADD COLUMN actions_retained_until timestamptz;

UPDATE webhook_deliveries
SET active_retained_until = created_at + ((policy_snapshot->'horizons'->>1)::double precision / 1000000000) * interval '1 second',
    terminal_retained_until = created_at + ((policy_snapshot->'horizons'->>2)::double precision / 1000000000) * interval '1 second',
    attempts_retained_until = created_at + ((policy_snapshot->'horizons'->>3)::double precision / 1000000000) * interval '1 second',
    actions_retained_until = created_at + ((policy_snapshot->'horizons'->>4)::double precision / 1000000000) * interval '1 second';

ALTER TABLE webhook_deliveries
    ALTER COLUMN active_retained_until SET NOT NULL,
    ALTER COLUMN terminal_retained_until SET NOT NULL,
    ALTER COLUMN attempts_retained_until SET NOT NULL,
    ALTER COLUMN actions_retained_until SET NOT NULL,
    ADD CHECK (isfinite(active_retained_until) AND isfinite(terminal_retained_until)
        AND isfinite(attempts_retained_until) AND isfinite(actions_retained_until));

ALTER TABLE webhook_attempts
    DROP COLUMN retry_after,
    ADD COLUMN retry_after_delay_ms bigint,
    ADD COLUMN retry_after_source text COLLATE "C",
    ADD COLUMN retained_until timestamptz;

UPDATE webhook_attempts a
SET retained_until = d.attempts_retained_until
FROM webhook_deliveries d
WHERE d.owner_scope = a.owner_scope AND d.delivery_id = a.delivery_id;

ALTER TABLE webhook_attempts
    ALTER COLUMN retained_until SET NOT NULL,
    ADD CHECK (retry_after_delay_ms IS NULL OR retry_after_delay_ms >= 0),
    ADD CHECK (retry_after_source IS NULL OR retry_after_source IN ('delay_seconds', 'http_date')),
    ADD CHECK ((retry_after_delay_ms IS NULL) = (retry_after_source IS NULL)),
    ADD CHECK (isfinite(retained_until));

ALTER TABLE webhook_operator_actions ADD COLUMN retained_until timestamptz;
UPDATE webhook_operator_actions SET retained_until = created_at + interval '365 days';
ALTER TABLE webhook_operator_actions
    ALTER COLUMN retained_until SET NOT NULL,
    ADD CHECK (isfinite(retained_until));

-- +goose Down

ALTER TABLE webhook_operator_actions DROP COLUMN retained_until;
ALTER TABLE webhook_attempts
    DROP COLUMN retained_until,
    DROP COLUMN retry_after_source,
    DROP COLUMN retry_after_delay_ms,
    ADD COLUMN retry_after text;
ALTER TABLE webhook_deliveries
    DROP COLUMN actions_retained_until,
    DROP COLUMN attempts_retained_until,
    DROP COLUMN terminal_retained_until,
    DROP COLUMN active_retained_until;
ALTER TABLE webhook_events
    DROP COLUMN payload_erased_at,
    DROP COLUMN payload_retained_until,
    ALTER COLUMN body SET NOT NULL;
ALTER TABLE webhook_destinations
    DROP COLUMN key_references_erased_at,
    DROP COLUMN key_references_retained_until,
    DROP COLUMN destination_retained_until,
    ALTER COLUMN active_key_reference SET NOT NULL;
