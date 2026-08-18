-- +goose Up

CREATE TABLE postgres_http_idempotency (
    identity_token bytea PRIMARY KEY,
    fingerprint_version smallint NOT NULL,
    fingerprint bytea NOT NULL,
    result bytea,
    expires_at timestamptz,
    CONSTRAINT postgres_http_idempotency_identity_check CHECK (
        octet_length(identity_token) = 32
    ),
    CONSTRAINT postgres_http_idempotency_fingerprint_check CHECK (
        fingerprint_version > 0 AND octet_length(fingerprint) = 32
    ),
    CONSTRAINT postgres_http_idempotency_result_check CHECK (
        (result IS NULL AND expires_at IS NULL)
        OR (result IS NOT NULL AND octet_length(result) BETWEEN 1 AND 1048576 AND expires_at IS NOT NULL)
    )
);

-- +goose Down

DROP TABLE postgres_http_idempotency;
