-- +goose Up

-- +goose StatementBegin
DO $$
DECLARE
    body_constraint text;
BEGIN
    SELECT conname INTO STRICT body_constraint
    FROM pg_constraint
    WHERE conrelid = 'webhook_events'::regclass
      AND contype = 'c'
      AND pg_get_constraintdef(oid) LIKE '%octet_length(body)%';
    EXECUTE format('ALTER TABLE webhook_events DROP CONSTRAINT %I', body_constraint);
END $$;
-- +goose StatementEnd

ALTER TABLE webhook_events
    ALTER COLUMN body DROP NOT NULL,
    ADD CONSTRAINT webhook_events_retained_body_check CHECK (
        (body IS NULL OR octet_length(body) BETWEEN 1 AND 262144)
        AND octet_length(intent_fingerprint) = 32
    );

ALTER TABLE webhook_deliveries
    ADD COLUMN payload_retained_until timestamptz NOT NULL DEFAULT 'infinity',
    ADD COLUMN active_retained_until timestamptz NOT NULL DEFAULT 'infinity',
    ADD COLUMN terminal_summary_retained_until timestamptz NOT NULL DEFAULT 'infinity',
    ADD COLUMN attempt_retained_until timestamptz NOT NULL DEFAULT 'infinity',
    ADD COLUMN action_retained_until timestamptz NOT NULL DEFAULT 'infinity',
    ADD COLUMN destination_generation_retained_until timestamptz NOT NULL DEFAULT 'infinity',
    ADD COLUMN receiver_dedup_retained_until timestamptz NOT NULL DEFAULT 'infinity',
    ADD COLUMN legal_hold boolean NOT NULL DEFAULT false;

ALTER TABLE webhook_deliveries
    ADD CONSTRAINT webhook_deliveries_retention_finite CHECK (
        (isfinite(payload_retained_until) OR payload_retained_until = 'infinity')
        AND (isfinite(active_retained_until) OR active_retained_until = 'infinity')
        AND (isfinite(terminal_summary_retained_until) OR terminal_summary_retained_until = 'infinity')
        AND (isfinite(attempt_retained_until) OR attempt_retained_until = 'infinity')
        AND (isfinite(action_retained_until) OR action_retained_until = 'infinity')
        AND (isfinite(destination_generation_retained_until) OR destination_generation_retained_until = 'infinity')
        AND (isfinite(receiver_dedup_retained_until) OR receiver_dedup_retained_until = 'infinity')
    );

ALTER TABLE webhook_deliveries
    DROP CONSTRAINT webhook_deliveries_cumulative_summary_check,
    ADD CONSTRAINT webhook_deliveries_cumulative_summary_check CHECK (
        cumulative_summary IN ('none', 'http_accepted', 'http_rejected', 'locally_denied', 'attempts_exhausted', 'outcome_unknown', 'closed_unknown', 'retained')
    );

ALTER TABLE webhook_attempts
    ADD COLUMN key_references text[] NOT NULL DEFAULT ARRAY[]::text[],
    ADD COLUMN retry_after_delay_ns bigint,
    ADD COLUMN retry_delay_ns bigint,
    ADD CONSTRAINT webhook_attempts_key_references_bound CHECK (cardinality(key_references) BETWEEN 0 AND 2),
    ADD CONSTRAINT webhook_attempts_retry_delays_bound CHECK (
        (retry_after_delay_ns IS NULL OR retry_after_delay_ns >= 0)
        AND (retry_delay_ns IS NULL OR retry_delay_ns >= 0)
    );

ALTER TABLE webhook_operator_actions
    ADD COLUMN retain_until timestamptz NOT NULL DEFAULT 'infinity',
    ADD COLUMN request_payload jsonb,
    ADD COLUMN result_cycle bigint NOT NULL DEFAULT 0;

ALTER TABLE webhook_operator_actions
    ADD CONSTRAINT webhook_operator_actions_retention_finite CHECK (isfinite(retain_until) OR retain_until = 'infinity'),
    ADD CONSTRAINT webhook_operator_actions_payload_shape CHECK (request_payload IS NULL OR jsonb_typeof(request_payload) = 'object'),
    ADD CONSTRAINT webhook_operator_actions_result_cycle_check CHECK (result_cycle >= 0);

ALTER TABLE webhook_operator_actions
    DROP CONSTRAINT webhook_operator_actions_action_kind_check,
    ADD CONSTRAINT webhook_operator_actions_action_kind_check CHECK (
        action_kind IN ('destination_state', 'key_rotation', 'redrive', 'close_unknown', 'retention_hold', 'privacy_delete', 'namespace_retire')
    );

CREATE TABLE webhook_destination_tombstones (
    owner_scope text COLLATE "C" NOT NULL,
    destination_id text COLLATE "C" NOT NULL,
    generation bigint NOT NULL,
    retired_at timestamptz NOT NULL,
    PRIMARY KEY (owner_scope, destination_id, generation),
    CHECK (generation > 0),
    CHECK (octet_length(owner_scope) BETWEEN 1 AND 256 AND owner_scope !~ '[[:space:][:cntrl:]]'),
    CHECK (octet_length(destination_id) BETWEEN 1 AND 256 AND destination_id !~ '[[:space:][:cntrl:]]'),
    CHECK (isfinite(retired_at))
);

CREATE INDEX webhook_deliveries_event_idx
    ON webhook_deliveries (owner_scope, business_event_id, delivery_id);

CREATE INDEX webhook_operator_actions_target_idx
    ON webhook_operator_actions (owner_scope, target_kind, target_id, action_id);

CREATE FUNCTION webhook_delivery_policy_valid(candidate jsonb)
RETURNS boolean
LANGUAGE sql
IMMUTABLE
PARALLEL SAFE
AS $$
SELECT COALESCE(jsonb_typeof(candidate) = 'object'
    AND candidate ?& ARRAY[
        'maximum_payload_bytes', 'accepted_content_types', 'accepted_business_schemas',
        'maximum_attempts', 'maximum_delivery_age', 'backoff_base', 'backoff_cap',
        'retry_after_cap', 'attempt_timeout', 'response_header_timeout',
        'response_header_bytes', 'response_body_bytes', 'destination_concurrency',
        'global_concurrency', 'drain_timeout', 'redrive_attempts', 'redrive_age',
        'horizons', 'automatic_pause', 'automatic_pause_classes', 'pause_window',
        'pause_duration', 'pause_threshold', 'pause_minimum_traffic',
        'pause_manual_only', 'pause_retention_effect', 'pause_alert_policy'
    ]
    AND candidate - ARRAY[
        'maximum_payload_bytes', 'accepted_content_types', 'accepted_business_schemas',
        'maximum_attempts', 'maximum_delivery_age', 'backoff_base', 'backoff_cap',
        'retry_after_cap', 'attempt_timeout', 'response_header_timeout',
        'response_header_bytes', 'response_body_bytes', 'destination_concurrency',
        'global_concurrency', 'drain_timeout', 'redrive_attempts', 'redrive_age',
        'horizons', 'automatic_pause', 'automatic_pause_classes', 'pause_window',
        'pause_duration', 'pause_threshold', 'pause_minimum_traffic',
        'pause_manual_only', 'pause_retention_effect', 'pause_alert_policy',
        'minimum_tls_version'
    ] = '{}'::jsonb
    AND jsonb_typeof(candidate->'accepted_content_types') = 'array'
    AND jsonb_array_length(candidate->'accepted_content_types') BETWEEN 1 AND 64
    AND NOT EXISTS (
        SELECT 1 FROM jsonb_array_elements(candidate->'accepted_content_types') item
        WHERE jsonb_typeof(item) <> 'string'
           OR octet_length(item #>> '{}') NOT BETWEEN 1 AND 256
           OR (item #>> '{}') ~ '[[:cntrl:]]'
    )
    AND jsonb_array_length(candidate->'accepted_content_types') = (
        SELECT count(DISTINCT lower(btrim(item #>> '{}')))
        FROM jsonb_array_elements(candidate->'accepted_content_types') item
    )
    AND jsonb_typeof(candidate->'accepted_business_schemas') = 'array'
    AND jsonb_array_length(candidate->'accepted_business_schemas') BETWEEN 1 AND 64
    AND NOT EXISTS (
        SELECT 1 FROM jsonb_array_elements(candidate->'accepted_business_schemas') item
        WHERE jsonb_typeof(item) <> 'string'
           OR octet_length(item #>> '{}') NOT BETWEEN 1 AND 256
           OR (item #>> '{}') ~ '[[:space:][:cntrl:]]'
    )
    AND jsonb_array_length(candidate->'accepted_business_schemas') = (
        SELECT count(DISTINCT item #>> '{}')
        FROM jsonb_array_elements(candidate->'accepted_business_schemas') item
    )
    AND candidate->'automatic_pause' = 'false'::jsonb
    AND (candidate->'automatic_pause_classes' = 'null'::jsonb OR (
        jsonb_typeof(candidate->'automatic_pause_classes') = 'array'
        AND jsonb_array_length(candidate->'automatic_pause_classes') = 0
    ))
    AND (candidate->>'pause_window')::numeric = 0
    AND (candidate->>'pause_duration')::numeric = 0
    AND (candidate->>'pause_threshold')::bigint = 0
    AND (candidate->>'pause_minimum_traffic')::bigint = 0
    AND candidate->'pause_manual_only' = 'false'::jsonb
    AND candidate->>'pause_retention_effect' = ''
    AND candidate->>'pause_alert_policy' = ''
    AND jsonb_typeof(candidate->'horizons') = 'array'
    AND jsonb_array_length(candidate->'horizons') = 8
    AND NOT EXISTS (
        SELECT 1 FROM jsonb_array_elements_text(candidate->'horizons') horizon
        WHERE horizon::numeric <= 0 OR horizon::numeric > 315360000000000000
    )
    AND (candidate->'horizons'->>0)::numeric >= GREATEST((candidate->>'maximum_delivery_age')::numeric, (candidate->'horizons'->>6)::numeric)
    AND (candidate->'horizons'->>1)::numeric >= GREATEST((candidate->>'maximum_delivery_age')::numeric, (candidate->'horizons'->>6)::numeric)
    AND (candidate->'horizons'->>2)::numeric >= GREATEST((candidate->>'maximum_delivery_age')::numeric, (candidate->'horizons'->>6)::numeric)
    AND (candidate->'horizons'->>3)::numeric >= (candidate->>'maximum_delivery_age')::numeric
    AND (candidate->'horizons'->>4)::numeric >= (candidate->'horizons'->>6)::numeric
    AND (candidate->'horizons'->>5)::numeric >= GREATEST((candidate->>'maximum_delivery_age')::numeric, (candidate->'horizons'->>6)::numeric)
    AND (candidate->'horizons'->>7)::numeric >= GREATEST((candidate->>'maximum_delivery_age')::numeric, (candidate->'horizons'->>6)::numeric)
    AND (candidate->>'maximum_payload_bytes')::bigint BETWEEN 1 AND 262144
    AND (candidate->>'maximum_attempts')::bigint BETWEEN 1 AND 100
    AND (candidate->>'maximum_delivery_age')::numeric BETWEEN 1 AND 31536000000000000
    AND (candidate->>'backoff_base')::numeric BETWEEN 1 AND (candidate->>'backoff_cap')::numeric
    AND (candidate->>'backoff_cap')::numeric <= (candidate->>'maximum_delivery_age')::numeric
    AND (candidate->>'retry_after_cap')::numeric BETWEEN 1 AND (candidate->>'maximum_delivery_age')::numeric
    AND (candidate->>'attempt_timeout')::numeric BETWEEN 1 AND 600000000000
    AND (candidate->>'response_header_timeout')::numeric BETWEEN 1 AND (candidate->>'attempt_timeout')::numeric
    AND (candidate->>'response_header_bytes')::bigint BETWEEN 1 AND 1048576
    AND (candidate->>'response_body_bytes')::bigint BETWEEN 1 AND 1048576
    AND (candidate->>'destination_concurrency')::bigint BETWEEN 1 AND (candidate->>'global_concurrency')::bigint
    AND (candidate->>'global_concurrency')::bigint BETWEEN 1 AND 256
    AND (candidate->>'drain_timeout')::numeric > (candidate->>'attempt_timeout')::numeric
    AND (candidate->>'drain_timeout')::numeric <= 1800000000000
    AND (candidate->>'redrive_attempts')::bigint BETWEEN 1 AND 100
    AND (candidate->>'redrive_age')::numeric BETWEEN 1 AND 31536000000000000
    AND (NOT candidate ? 'minimum_tls_version' OR candidate->>'minimum_tls_version' IN ('1.2', '1.3')), false)
$$;

ALTER TABLE webhook_destinations
    ADD CONSTRAINT webhook_destinations_policy_valid CHECK (
        webhook_delivery_policy_valid(policy)
        AND (policy->>'destination_concurrency')::integer = destination_concurrency
        AND (policy->>'global_concurrency')::integer = global_concurrency
    );

ALTER TABLE webhook_deliveries
    ADD CONSTRAINT webhook_deliveries_policy_valid CHECK (webhook_delivery_policy_valid(policy_snapshot));

ALTER TABLE webhook_attempts
    ADD CONSTRAINT webhook_attempts_authorization_evidence CHECK (
        NOT send_authorized OR (
            may_have_sent
            AND key_reference IS NOT NULL
            AND signature_header_digest IS NOT NULL
            AND dns_set_digest IS NOT NULL
            AND selected_address IS NOT NULL
        )
    );

-- +goose Down

DROP INDEX webhook_operator_actions_target_idx;
DROP INDEX webhook_deliveries_event_idx;

DROP TABLE webhook_destination_tombstones;

ALTER TABLE webhook_attempts
    DROP CONSTRAINT webhook_attempts_authorization_evidence;

ALTER TABLE webhook_deliveries
    DROP CONSTRAINT webhook_deliveries_policy_valid;

ALTER TABLE webhook_destinations
    DROP CONSTRAINT webhook_destinations_policy_valid;

DROP FUNCTION webhook_delivery_policy_valid(jsonb);

ALTER TABLE webhook_operator_actions
    DROP CONSTRAINT webhook_operator_actions_action_kind_check,
    ADD CONSTRAINT webhook_operator_actions_action_kind_check CHECK (
        action_kind IN ('destination_state', 'key_rotation', 'redrive', 'close_unknown', 'privacy_delete', 'namespace_retire')
    ),
    DROP CONSTRAINT webhook_operator_actions_result_cycle_check,
    DROP CONSTRAINT webhook_operator_actions_payload_shape,
    DROP CONSTRAINT webhook_operator_actions_retention_finite,
    DROP COLUMN result_cycle,
    DROP COLUMN request_payload,
    DROP COLUMN retain_until;

ALTER TABLE webhook_attempts
    DROP CONSTRAINT webhook_attempts_retry_delays_bound,
    DROP CONSTRAINT webhook_attempts_key_references_bound,
    DROP COLUMN retry_delay_ns,
    DROP COLUMN retry_after_delay_ns,
    DROP COLUMN key_references;

ALTER TABLE webhook_deliveries
    DROP CONSTRAINT webhook_deliveries_cumulative_summary_check;

UPDATE webhook_deliveries
SET cumulative_summary = 'attempts_exhausted'
WHERE cumulative_summary = 'retained';

ALTER TABLE webhook_deliveries
    ADD CONSTRAINT webhook_deliveries_cumulative_summary_check CHECK (
        cumulative_summary IN ('none', 'http_accepted', 'http_rejected', 'locally_denied', 'attempts_exhausted', 'outcome_unknown', 'closed_unknown')
    ),
    DROP CONSTRAINT webhook_deliveries_retention_finite,
    DROP COLUMN legal_hold,
    DROP COLUMN receiver_dedup_retained_until,
    DROP COLUMN destination_generation_retained_until,
    DROP COLUMN action_retained_until,
    DROP COLUMN attempt_retained_until,
    DROP COLUMN terminal_summary_retained_until,
    DROP COLUMN active_retained_until,
    DROP COLUMN payload_retained_until;

ALTER TABLE webhook_events
    DROP CONSTRAINT webhook_events_retained_body_check,
    ALTER COLUMN body SET NOT NULL,
    ADD CONSTRAINT webhook_events_body_intent_fingerprint_check CHECK (
        octet_length(body) BETWEEN 1 AND 262144 AND octet_length(intent_fingerprint) = 32
    );
