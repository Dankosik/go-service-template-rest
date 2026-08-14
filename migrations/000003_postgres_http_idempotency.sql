-- +goose Up

CREATE SEQUENCE postgres_http_idempotency_generation_seq AS bigint
    MINVALUE 1
    START WITH 1
    NO CYCLE;

CREATE TABLE postgres_http_idempotency (
    identity_token bytea PRIMARY KEY,
    generation bigint NOT NULL,
    phase text COLLATE "C" NOT NULL,
    provisional_fingerprint_version text COLLATE "C",
    provisional_fingerprint bytea,
    fingerprint_version text COLLATE "C",
    fingerprint bytea,
    result bytea,
    result_max_bytes bigint,
    replay_nanos bigint,
    duplicate_risk_nanos bigint,
    duplicate_risk_permanent boolean,
    recover_after timestamptz NOT NULL,
    committed_at timestamptz,
    CONSTRAINT postgres_http_idempotency_identity_token_check CHECK (
        octet_length(identity_token) = 32
    ),
    CONSTRAINT postgres_http_idempotency_generation_check CHECK (
        generation > 0
    ),
    CONSTRAINT postgres_http_idempotency_phase_check CHECK (
        phase IN ('reserved', 'completed')
    ),
    CONSTRAINT postgres_http_idempotency_state_check CHECK (
        (
            phase = 'reserved'
            AND provisional_fingerprint_version IS NOT NULL
            AND octet_length(provisional_fingerprint_version) > 0
            AND provisional_fingerprint IS NOT NULL
            AND octet_length(provisional_fingerprint) = 32
            AND fingerprint_version IS NULL
            AND fingerprint IS NULL
            AND result IS NULL
            AND result_max_bytes IS NULL
            AND replay_nanos IS NULL
            AND duplicate_risk_nanos IS NULL
            AND duplicate_risk_permanent IS NULL
            AND committed_at IS NULL
        )
        OR (
            phase = 'completed'
            AND provisional_fingerprint_version IS NULL
            AND provisional_fingerprint IS NULL
            AND fingerprint_version IS NOT NULL
            AND octet_length(fingerprint_version) > 0
            AND fingerprint IS NOT NULL
            AND octet_length(fingerprint) = 32
            AND result IS NOT NULL
            AND result_max_bytes IS NOT NULL
            AND result_max_bytes > 0
            AND octet_length(result) <= result_max_bytes
            AND replay_nanos IS NOT NULL
            AND replay_nanos > 0
            AND duplicate_risk_permanent IS NOT NULL
            AND (
                (duplicate_risk_permanent AND duplicate_risk_nanos IS NULL)
                OR (
                    NOT duplicate_risk_permanent
                    AND duplicate_risk_nanos IS NOT NULL
                    AND duplicate_risk_nanos >= replay_nanos
                )
            )
        )
    )
);

-- +goose Down

DROP TABLE postgres_http_idempotency;
DROP SEQUENCE postgres_http_idempotency_generation_seq;
