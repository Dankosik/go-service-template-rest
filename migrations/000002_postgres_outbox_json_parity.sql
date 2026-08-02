-- +goose Up

-- +goose StatementBegin
DO $$
BEGIN
    IF current_setting('server_encoding') IS DISTINCT FROM 'UTF8' THEN
        RAISE EXCEPTION 'PostgreSQL outbox requires server_encoding=UTF8, got %',
            current_setting('server_encoding')
            USING ERRCODE = '22023';
    END IF;
END
$$;
-- +goose StatementEnd

ALTER TABLE outbox_events
    DROP CONSTRAINT outbox_events_payload_check,
    ADD CONSTRAINT outbox_events_payload_check CHECK (
        octet_length(payload) BETWEEN 1 AND 262144
        AND convert_from(payload, 'UTF8')::json IS NOT NULL
    ),
    DROP CONSTRAINT outbox_events_metadata_check,
    ADD CONSTRAINT outbox_events_metadata_check CHECK (
        octet_length(metadata) BETWEEN 2 AND 32768
        AND json_typeof(convert_from(metadata, 'UTF8')::json) = 'object'
    );

-- +goose Down

ALTER TABLE outbox_events
    DROP CONSTRAINT outbox_events_payload_check,
    ADD CONSTRAINT outbox_events_payload_check CHECK (
        octet_length(payload) BETWEEN 1 AND 262144
        AND convert_from(payload, 'UTF8')::jsonb IS NOT NULL
    ),
    DROP CONSTRAINT outbox_events_metadata_check,
    ADD CONSTRAINT outbox_events_metadata_check CHECK (
        octet_length(metadata) BETWEEN 2 AND 32768
        AND jsonb_typeof(convert_from(metadata, 'UTF8')::jsonb) = 'object'
    );
