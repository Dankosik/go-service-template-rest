-- name: ReadHTTPIdempotency :one
WITH input AS (
    SELECT sqlc.arg(identity_token)::bytea AS identity_token
)
SELECT
    (entry.identity_token IS NOT NULL)::boolean AS row_exists,
    entry.generation,
    entry.phase,
    entry.provisional_fingerprint_version,
    entry.provisional_fingerprint,
    entry.fingerprint_version,
    entry.fingerprint,
    entry.result,
    entry.result_max_bytes,
    entry.replay_nanos,
    entry.duplicate_risk_nanos,
    entry.duplicate_risk_permanent,
    entry.recover_after,
    entry.committed_at,
    coalesce(entry.recover_after <= clock_timestamp(), FALSE)::boolean AS recovery_due,
    (NOT pg_is_in_recovery()
        AND current_setting('transaction_read_only') = 'off')::boolean AS writer_primary
FROM input
LEFT JOIN postgres_http_idempotency AS entry USING (identity_token);

-- name: CheckHTTPIdempotencyWriter :one
SELECT
    (NOT pg_is_in_recovery()
        AND current_setting('transaction_read_only') = 'off')::boolean AS writer_primary;

-- name: InsertHTTPIdempotencyReservation :one
INSERT INTO postgres_http_idempotency (
    identity_token,
    generation,
    phase,
    provisional_fingerprint_version,
    provisional_fingerprint,
    recover_after
)
SELECT
    sqlc.arg(identity_token)::bytea,
    nextval('postgres_http_idempotency_generation_seq'),
    'reserved',
    sqlc.arg(fingerprint_version)::text,
    sqlc.arg(fingerprint)::bytea,
    clock_timestamp() + sqlc.arg(recovery_micros)::bigint * interval '1 microsecond'
WHERE NOT pg_is_in_recovery()
  AND current_setting('transaction_read_only') = 'off'
ON CONFLICT (identity_token) DO NOTHING
RETURNING generation;

-- name: LockHTTPIdempotencyReservation :one
SELECT
    generation,
    phase,
    provisional_fingerprint_version,
    provisional_fingerprint,
    fingerprint_version,
    fingerprint,
    result,
    result_max_bytes,
    replay_nanos,
    duplicate_risk_nanos,
    duplicate_risk_permanent,
    recover_after,
    committed_at,
    (recover_after <= clock_timestamp())::boolean AS recovery_due
FROM postgres_http_idempotency
WHERE identity_token = sqlc.arg(identity_token)::bytea
FOR UPDATE NOWAIT;

-- name: AdvanceHTTPIdempotencyReservation :one
UPDATE postgres_http_idempotency
SET
    generation = nextval('postgres_http_idempotency_generation_seq'),
    provisional_fingerprint_version = sqlc.arg(fingerprint_version)::text,
    provisional_fingerprint = sqlc.arg(fingerprint)::bytea,
    recover_after = clock_timestamp() + sqlc.arg(recovery_micros)::bigint * interval '1 microsecond'
WHERE identity_token = sqlc.arg(identity_token)::bytea
  AND generation = sqlc.arg(generation)::bigint
  AND phase = 'reserved'
RETURNING generation;

-- name: CompleteHTTPIdempotencyReservation :one
UPDATE postgres_http_idempotency
SET
    phase = 'completed',
    provisional_fingerprint_version = NULL,
    provisional_fingerprint = NULL,
    fingerprint_version = sqlc.arg(fingerprint_version)::text,
    fingerprint = sqlc.arg(fingerprint)::bytea,
    result = sqlc.arg(result)::bytea,
    result_max_bytes = sqlc.arg(result_max_bytes)::bigint,
    replay_nanos = sqlc.arg(replay_nanos)::bigint,
    duplicate_risk_nanos = sqlc.narg(duplicate_risk_nanos)::bigint,
    duplicate_risk_permanent = sqlc.arg(duplicate_risk_permanent)::boolean
WHERE identity_token = sqlc.arg(identity_token)::bytea
  AND generation = sqlc.arg(generation)::bigint
  AND phase = 'reserved'
RETURNING generation;

-- name: ReleaseHTTPIdempotencyReservation :execrows
DELETE FROM postgres_http_idempotency
WHERE identity_token = sqlc.arg(identity_token)::bytea
  AND generation = sqlc.arg(generation)::bigint
  AND phase = 'reserved';

-- name: MaterializeHTTPIdempotencyCommitEpoch :one
UPDATE postgres_http_idempotency
SET committed_at = CASE
    WHEN current_setting('track_commit_timestamp') = 'on'
    THEN pg_xact_commit_timestamp(xmin)
    ELSE NULL
END
WHERE identity_token = sqlc.arg(identity_token)::bytea
  AND phase = 'completed'
  AND committed_at IS NULL
  AND CASE
      WHEN current_setting('track_commit_timestamp') = 'on'
      THEN pg_xact_commit_timestamp(xmin) IS NOT NULL
      ELSE FALSE
  END
RETURNING committed_at;
