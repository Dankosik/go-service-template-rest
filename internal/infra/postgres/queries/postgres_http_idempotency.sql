-- name: ClaimHTTPIdempotency :one
INSERT INTO postgres_http_idempotency (
    identity_token,
    fingerprint_version,
    fingerprint
)
SELECT
    sqlc.arg(identity_token)::bytea,
    sqlc.arg(fingerprint_version)::smallint,
    sqlc.arg(fingerprint)::bytea
WHERE NOT pg_is_in_recovery()
  AND current_setting('transaction_read_only') = 'off'
ON CONFLICT (identity_token) DO UPDATE
SET
    fingerprint_version = EXCLUDED.fingerprint_version,
    fingerprint = EXCLUDED.fingerprint,
    result = NULL,
    expires_at = NULL
WHERE postgres_http_idempotency.expires_at <= clock_timestamp()
RETURNING identity_token;

-- name: ReadHTTPIdempotency :one
WITH input AS (
    SELECT sqlc.arg(identity_token)::bytea AS identity_token
)
SELECT
    (entry.identity_token IS NOT NULL)::boolean AS row_exists,
    (NOT pg_is_in_recovery()
        AND current_setting('transaction_read_only') = 'off')::boolean AS writer_primary,
    coalesce(entry.expires_at > clock_timestamp(), FALSE)::boolean AS live,
    entry.fingerprint_version,
    entry.fingerprint,
    entry.result
FROM input
LEFT JOIN postgres_http_idempotency AS entry USING (identity_token);

-- name: CompleteHTTPIdempotency :one
UPDATE postgres_http_idempotency
SET
    result = sqlc.arg(result)::bytea,
    expires_at = clock_timestamp()
        + sqlc.arg(retention_micros)::bigint * interval '1 microsecond'
WHERE identity_token = sqlc.arg(identity_token)::bytea
  AND result IS NULL
RETURNING result;

-- name: CleanupHTTPIdempotency :execrows
WITH expired AS (
    SELECT identity_token
    FROM postgres_http_idempotency
    WHERE expires_at <= clock_timestamp()
    ORDER BY expires_at, identity_token
    FOR UPDATE SKIP LOCKED
    LIMIT sqlc.arg(batch_size)
)
DELETE FROM postgres_http_idempotency AS target
USING expired
WHERE target.identity_token = expired.identity_token;
