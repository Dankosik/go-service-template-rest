-- name: ListPostgresJobsSchemaColumns :many
SELECT
    table_name::text AS table_name,
    column_name::text AS column_name,
    data_type::text AS data_type,
    udt_name::text AS udt_name,
    is_nullable::text AS is_nullable,
    coalesce(column_default, '')::text AS column_default,
    coalesce(collation_name, '')::text AS collation_name
FROM information_schema.columns
WHERE table_schema = current_schema()
  AND table_name = ANY(sqlc.arg(table_names)::text[])
ORDER BY table_name, ordinal_position;

-- name: ListPostgresJobsSchemaConstraints :many
SELECT
    relation.relname::text AS table_name,
    constraint_record.conname::text AS constraint_name,
    constraint_record.contype::text AS constraint_type,
    md5(pg_get_constraintdef(constraint_record.oid, true))::text AS definition_hash
FROM pg_catalog.pg_constraint AS constraint_record
JOIN pg_catalog.pg_class AS relation ON relation.oid = constraint_record.conrelid
JOIN pg_catalog.pg_namespace AS namespace ON namespace.oid = relation.relnamespace
WHERE namespace.nspname = current_schema()
  AND relation.relname = ANY(sqlc.arg(table_names)::text[])
ORDER BY relation.relname, constraint_record.conname;

-- name: ListPostgresJobsSchemaIndexes :many
SELECT
    relation.relname::text AS table_name,
    index_relation.relname::text AS index_name,
    md5(pg_get_indexdef(index_record.indexrelid))::text AS definition_hash
FROM pg_catalog.pg_index AS index_record
JOIN pg_catalog.pg_class AS relation ON relation.oid = index_record.indrelid
JOIN pg_catalog.pg_class AS index_relation ON index_relation.oid = index_record.indexrelid
JOIN pg_catalog.pg_namespace AS namespace ON namespace.oid = relation.relnamespace
WHERE namespace.nspname = current_schema()
  AND relation.relname = ANY(sqlc.arg(table_names)::text[])
ORDER BY relation.relname, index_relation.relname;

-- name: GetPostgresJobsNeutralScope :one
SELECT work_class
FROM postgres_job_claim_scopes
WHERE work_class = 'neutral';

-- name: CheckPostgresJobsProducerAuthority :one
SELECT
    (NOT pg_is_in_recovery()
        AND current_setting('transaction_read_only') = 'off')::boolean AS writer_primary,
    (has_table_privilege(current_user, 'postgres_jobs', 'SELECT')
        AND has_table_privilege(current_user, 'postgres_jobs', 'INSERT'))::boolean AS producer_privileges;

-- name: InsertPostgresJobAcceptance :one
INSERT INTO postgres_jobs (
    logical_job_id,
    producer_scope,
    producer_key,
    occurrence_scope,
    occurrence_id,
    effect_scope,
    effect_key,
    intent_fingerprint,
    kind,
    args_version,
    policy_version,
    payload,
    work_class,
    state,
    available_at
)
VALUES (
    sqlc.arg(logical_job_id)::text,
    sqlc.arg(producer_scope)::text,
    sqlc.arg(producer_key)::text,
    sqlc.arg(occurrence_scope)::text,
    sqlc.arg(occurrence_id)::text,
    sqlc.arg(effect_scope)::text,
    sqlc.arg(effect_key)::text,
    sqlc.arg(intent_fingerprint)::bytea,
    sqlc.arg(kind)::text,
    sqlc.arg(args_version)::text,
    sqlc.arg(policy_version)::text,
    sqlc.arg(payload)::bytea,
    'neutral',
    CASE
        WHEN sqlc.arg(available_at)::timestamptz <= clock_timestamp() THEN 'ready'
        ELSE 'scheduled'
    END,
    sqlc.arg(available_at)::timestamptz
)
ON CONFLICT DO NOTHING
RETURNING logical_job_id;

-- name: ListPostgresJobsAcceptanceConflicts :many
SELECT
    logical_job_id,
    intent_fingerprint
FROM postgres_jobs
WHERE logical_job_id = sqlc.arg(logical_job_id)::text
   OR (producer_scope = sqlc.arg(producer_scope)::text
       AND producer_key = sqlc.arg(producer_key)::text)
   OR (occurrence_scope = sqlc.arg(occurrence_scope)::text
       AND occurrence_id = sqlc.arg(occurrence_id)::text)
   OR (effect_scope = sqlc.arg(effect_scope)::text
       AND effect_key = sqlc.arg(effect_key)::text)
ORDER BY logical_job_id;

-- name: ReadPostgresJobsAcceptance :one
WITH input AS (
    SELECT
        sqlc.arg(producer_scope)::text AS producer_scope,
        sqlc.arg(producer_key)::text AS producer_key
)
SELECT
    (job.logical_job_id IS NOT NULL)::boolean AS row_exists,
    job.logical_job_id,
    job.producer_scope,
    job.producer_key,
    job.occurrence_scope,
    job.occurrence_id,
    job.effect_scope,
    job.effect_key,
    job.intent_fingerprint,
    (NOT pg_is_in_recovery()
        AND current_setting('transaction_read_only') = 'off')::boolean AS writer_primary
FROM input
LEFT JOIN postgres_jobs AS job USING (producer_scope, producer_key);

-- name: ClaimPostgresJobs :many
WITH input_keys AS MATERIALIZED (
    SELECT kind.kind, args.args_version, policy.policy_version
    FROM unnest(sqlc.arg(kinds)::text[]) WITH ORDINALITY AS kind(kind, position)
    JOIN unnest(sqlc.arg(args_versions)::text[]) WITH ORDINALITY AS args(args_version, position) USING (position)
    JOIN unnest(sqlc.arg(policy_versions)::text[]) WITH ORDINALITY AS policy(policy_version, position) USING (position)
),
observed AS MATERIALIZED (
    SELECT clock_timestamp() AS observed_at
),
uncovered AS MATERIALIZED (
    SELECT DISTINCT job.kind, job.args_version, job.policy_version
    FROM postgres_jobs AS job
    WHERE NOT EXISTS (
        SELECT 1
        FROM input_keys AS key
        WHERE key.kind = job.kind
          AND key.args_version = job.args_version
          AND key.policy_version = job.policy_version
    )
),
missing AS MATERIALIZED (
    SELECT kind, args_version, policy_version
    FROM uncovered
    ORDER BY kind, args_version, policy_version
    LIMIT 16
),
scope AS MATERIALIZED (
    SELECT paused, scope_generation
    FROM postgres_job_claim_scopes
    WHERE work_class = 'neutral'
    FOR SHARE
),
eligible AS MATERIALIZED (
    SELECT job.logical_job_id
    FROM postgres_jobs AS job
    CROSS JOIN observed
    CROSS JOIN scope
    WHERE NOT EXISTS (SELECT 1 FROM uncovered)
      AND NOT scope.paused
      AND job.work_class = 'neutral'
      AND job.state IN ('ready', 'scheduled', 'retry_wait')
      AND job.available_at <= observed.observed_at
    ORDER BY job.available_at, job.logical_job_id
    LIMIT sqlc.arg(claim_limit)::integer
    FOR UPDATE OF job SKIP LOCKED
),
claimed AS (
    UPDATE postgres_jobs AS job
    SET state = 'running',
        attempt_generation = job.attempt_generation + 1,
        attempts_used = job.attempts_used + 1,
        current_worker_id = sqlc.arg(worker_id)::text,
        lease_expires_at = observed.observed_at
            + sqlc.arg(lease_microseconds)::bigint * interval '1 microsecond',
        updated_at = observed.observed_at,
        terminal_at = NULL
    FROM eligible, observed
    WHERE job.logical_job_id = eligible.logical_job_id
    RETURNING
        job.logical_job_id,
        job.producer_scope,
        job.producer_key,
        job.occurrence_scope,
        job.occurrence_id,
        job.effect_scope,
        job.effect_key,
        job.kind,
        job.args_version,
        job.policy_version,
        job.payload,
        job.recovery_generation,
        job.attempt_generation,
        job.attempts_used,
        job.budget_started_at,
        job.current_worker_id,
        job.lease_expires_at,
        observed.observed_at AS started_at
),
inserted_attempts AS (
    INSERT INTO postgres_job_attempts (
        logical_job_id,
        attempt_generation,
        recovery_generation,
        attempt_number,
        worker_id,
        started_at,
        lease_expires_at
    )
    SELECT
        claimed.logical_job_id,
        claimed.attempt_generation,
        claimed.recovery_generation,
        claimed.attempts_used,
        claimed.current_worker_id,
        claimed.started_at,
        claimed.lease_expires_at
    FROM claimed
    RETURNING logical_job_id, attempt_generation
)
SELECT
    NOT EXISTS (SELECT 1 FROM uncovered)::boolean AS compatible,
    scope.paused,
    scope.scope_generation,
    observed.observed_at::timestamptz AS observed_at,
    ARRAY(SELECT kind FROM missing ORDER BY kind, args_version, policy_version)::text[] AS missing_kinds,
    ARRAY(SELECT args_version FROM missing ORDER BY kind, args_version, policy_version)::text[] AS missing_args_versions,
    ARRAY(SELECT policy_version FROM missing ORDER BY kind, args_version, policy_version)::text[] AS missing_policy_versions,
    claimed.logical_job_id,
    claimed.producer_scope,
    claimed.producer_key,
    claimed.occurrence_scope,
    claimed.occurrence_id,
    claimed.effect_scope,
    claimed.effect_key,
    claimed.kind,
    claimed.args_version,
    claimed.policy_version,
    claimed.payload,
    claimed.recovery_generation,
    claimed.attempt_generation,
    claimed.attempts_used,
    claimed.budget_started_at,
    claimed.current_worker_id,
    claimed.started_at::timestamptz AS started_at,
    claimed.lease_expires_at
FROM observed
CROSS JOIN scope
LEFT JOIN claimed ON EXISTS (
    SELECT 1
    FROM inserted_attempts
    WHERE inserted_attempts.logical_job_id = claimed.logical_job_id
      AND inserted_attempts.attempt_generation = claimed.attempt_generation
)
ORDER BY claimed.logical_job_id;

-- name: ResolvePostgresJobsClaims :many
WITH input_attempts AS (
    SELECT job.logical_job_id, attempt.attempt_generation, recovery.recovery_generation, worker.worker_id
    FROM unnest(sqlc.arg(logical_job_ids)::text[]) WITH ORDINALITY AS job(logical_job_id, position)
    JOIN unnest(sqlc.arg(attempt_generations)::bigint[]) WITH ORDINALITY AS attempt(attempt_generation, position) USING (position)
    JOIN unnest(sqlc.arg(recovery_generations)::bigint[]) WITH ORDINALITY AS recovery(recovery_generation, position) USING (position)
    JOIN unnest(sqlc.arg(worker_ids)::text[]) WITH ORDINALITY AS worker(worker_id, position) USING (position)
)
SELECT
    input_attempts.logical_job_id::text AS logical_job_id,
    input_attempts.attempt_generation::bigint AS attempt_generation,
    input_attempts.recovery_generation::bigint AS recovery_generation,
    input_attempts.worker_id::text AS worker_id,
    (
        job.logical_job_id IS NOT NULL
        AND attempt.logical_job_id IS NOT NULL
    )::boolean AS committed
FROM input_attempts
LEFT JOIN postgres_jobs AS job
    ON job.logical_job_id = input_attempts.logical_job_id
   AND job.attempt_generation = input_attempts.attempt_generation
   AND job.recovery_generation = input_attempts.recovery_generation
   AND job.current_worker_id = input_attempts.worker_id
   AND job.state IN ('running', 'cancel_requested')
LEFT JOIN postgres_job_attempts AS attempt
    ON attempt.logical_job_id = input_attempts.logical_job_id
   AND attempt.attempt_generation = input_attempts.attempt_generation
   AND attempt.recovery_generation = input_attempts.recovery_generation
   AND attempt.worker_id = input_attempts.worker_id
ORDER BY input_attempts.logical_job_id, input_attempts.attempt_generation;

-- name: RenewPostgresJobAttempts :many
WITH input_attempts AS MATERIALIZED (
    SELECT job.logical_job_id, attempt.attempt_generation, recovery.recovery_generation, worker.worker_id
    FROM unnest(sqlc.arg(logical_job_ids)::text[]) WITH ORDINALITY AS job(logical_job_id, position)
    JOIN unnest(sqlc.arg(attempt_generations)::bigint[]) WITH ORDINALITY AS attempt(attempt_generation, position) USING (position)
    JOIN unnest(sqlc.arg(recovery_generations)::bigint[]) WITH ORDINALITY AS recovery(recovery_generation, position) USING (position)
    JOIN unnest(sqlc.arg(worker_ids)::text[]) WITH ORDINALITY AS worker(worker_id, position) USING (position)
),
observed AS MATERIALIZED (
    SELECT clock_timestamp() AS observed_at
),
renewed_jobs AS (
    UPDATE postgres_jobs AS job
    SET lease_expires_at = observed.observed_at
            + sqlc.arg(lease_microseconds)::bigint * interval '1 microsecond',
        updated_at = observed.observed_at
    FROM input_attempts, observed
    WHERE job.logical_job_id = input_attempts.logical_job_id
      AND job.attempt_generation = input_attempts.attempt_generation
      AND job.recovery_generation = input_attempts.recovery_generation
      AND job.current_worker_id = input_attempts.worker_id
      AND job.state IN ('running', 'cancel_requested')
    RETURNING
        job.logical_job_id,
        job.attempt_generation,
        job.recovery_generation,
        job.current_worker_id,
        job.state,
        job.lease_expires_at,
        observed.observed_at
),
renewed_attempts AS (
    UPDATE postgres_job_attempts AS attempt
    SET lease_expires_at = renewed_jobs.lease_expires_at
    FROM renewed_jobs
    WHERE attempt.logical_job_id = renewed_jobs.logical_job_id
      AND attempt.attempt_generation = renewed_jobs.attempt_generation
      AND attempt.recovery_generation = renewed_jobs.recovery_generation
      AND attempt.worker_id = renewed_jobs.current_worker_id
      AND attempt.finalized_at IS NULL
    RETURNING attempt.logical_job_id, attempt.attempt_generation
)
SELECT
    renewed_jobs.logical_job_id,
    renewed_jobs.attempt_generation,
    renewed_jobs.recovery_generation,
    renewed_jobs.current_worker_id AS worker_id,
    renewed_jobs.observed_at::timestamptz AS observed_at,
    renewed_jobs.lease_expires_at,
    (renewed_jobs.state = 'cancel_requested')::boolean AS cancel_requested
FROM renewed_jobs
JOIN renewed_attempts USING (logical_job_id, attempt_generation)
ORDER BY renewed_jobs.logical_job_id, renewed_jobs.attempt_generation;

-- name: FinalizePostgresJobAttempt :one
WITH observed AS MATERIALIZED (
    SELECT clock_timestamp() AS observed_at
),
locked_job AS MATERIALIZED (
    SELECT *
    FROM postgres_jobs
    WHERE logical_job_id = sqlc.arg(logical_job_id)::text
    FOR UPDATE
),
locked_attempt AS MATERIALIZED (
    SELECT attempt.*
    FROM postgres_job_attempts AS attempt
    JOIN locked_job ON locked_job.logical_job_id = attempt.logical_job_id
    WHERE attempt.attempt_generation = sqlc.arg(attempt_generation)::bigint
      AND attempt.recovery_generation = sqlc.arg(recovery_generation)::bigint
      AND attempt.worker_id = sqlc.arg(worker_id)::text
    FOR UPDATE OF attempt
),
current_attempt AS MATERIALIZED (
    SELECT locked_job.logical_job_id
    FROM locked_job
    JOIN locked_attempt USING (logical_job_id)
    WHERE locked_attempt.finalized_at IS NULL
      AND locked_job.attempt_generation = sqlc.arg(attempt_generation)::bigint
      AND locked_job.recovery_generation = sqlc.arg(recovery_generation)::bigint
      AND locked_job.current_worker_id = sqlc.arg(worker_id)::text
      AND locked_job.state IN ('running', 'cancel_requested')
),
updated_job AS (
    UPDATE postgres_jobs AS job
    SET state = sqlc.arg(final_state)::text,
        available_at = CASE
            WHEN sqlc.arg(final_state)::text = 'retry_wait'
            THEN observed.observed_at
                + sqlc.arg(delay_microseconds)::bigint * interval '1 microsecond'
            ELSE job.available_at
        END,
        attempts_used = sqlc.arg(attempts_used)::integer,
        current_worker_id = NULL,
        lease_expires_at = NULL,
        updated_at = observed.observed_at,
        terminal_at = CASE
            WHEN sqlc.arg(final_state)::text = 'retry_wait' THEN NULL
            ELSE observed.observed_at
        END
    FROM current_attempt, observed
    WHERE job.logical_job_id = current_attempt.logical_job_id
    RETURNING job.logical_job_id
),
updated_attempt AS (
    UPDATE postgres_job_attempts AS attempt
    SET finalized_at = observed.observed_at,
        final_state = sqlc.arg(final_state)::text,
        outcome = sqlc.arg(outcome)::text,
        effect_status = sqlc.arg(effect_status)::text,
        failure_code = NULLIF(sqlc.arg(failure_code)::text, ''),
        retry_at = CASE
            WHEN sqlc.arg(final_state)::text = 'retry_wait'
            THEN observed.observed_at
                + sqlc.arg(delay_microseconds)::bigint * interval '1 microsecond'
            ELSE NULL
        END,
        attempts_used = sqlc.arg(attempts_used)::integer,
        elapsed_used_milliseconds = sqlc.arg(elapsed_used_milliseconds)::bigint
    FROM updated_job, observed
    WHERE attempt.logical_job_id = updated_job.logical_job_id
      AND attempt.attempt_generation = sqlc.arg(attempt_generation)::bigint
    RETURNING attempt.*
),
result_attempt AS MATERIALIZED (
    SELECT * FROM updated_attempt
    UNION ALL
    SELECT * FROM locked_attempt
    WHERE locked_attempt.finalized_at IS NOT NULL
    LIMIT 1
)
SELECT
    CASE
        WHEN EXISTS (SELECT 1 FROM updated_attempt) THEN 'applied'
        WHEN EXISTS (SELECT 1 FROM locked_attempt WHERE finalized_at IS NOT NULL) THEN 'repeated'
        WHEN NOT EXISTS (SELECT 1 FROM locked_job) THEN 'not_found'
        ELSE 'stale'
    END::text AS status,
    EXISTS (SELECT 1 FROM result_attempt)::boolean AS has_result,
    coalesce((SELECT final_state FROM result_attempt), '')::text AS final_state,
    coalesce((SELECT outcome FROM result_attempt), '')::text AS outcome,
    coalesce((SELECT effect_status FROM result_attempt), '')::text AS effect_status,
    coalesce((SELECT failure_code FROM result_attempt), '')::text AS failure_code,
    (SELECT retry_at FROM result_attempt)::timestamptz AS retry_at,
    coalesce((SELECT attempts_used FROM result_attempt), 0)::integer AS attempts_used,
    coalesce((SELECT elapsed_used_milliseconds FROM result_attempt), 0)::bigint AS elapsed_used_milliseconds,
    (SELECT finalized_at FROM result_attempt)::timestamptz AS finalized_at;

-- name: ListExpiredPostgresJobAttempts :many
WITH observed AS MATERIALIZED (
    SELECT clock_timestamp() AS observed_at
)
SELECT
    job.logical_job_id,
    job.kind,
    job.args_version,
    job.policy_version,
    job.state,
    job.recovery_generation,
    job.attempt_generation,
    job.attempts_used,
    job.current_worker_id,
    job.budget_started_at,
    attempt.started_at,
    attempt.lease_expires_at,
    observed.observed_at::timestamptz AS observed_at,
    greatest(
        0,
        floor(extract(epoch FROM observed.observed_at - job.budget_started_at) * 1000)
    )::bigint AS elapsed_milliseconds
FROM postgres_jobs AS job
JOIN postgres_job_attempts AS attempt
  ON attempt.logical_job_id = job.logical_job_id
 AND attempt.attempt_generation = job.attempt_generation
CROSS JOIN observed
WHERE job.state IN ('running', 'cancel_requested')
  AND job.current_worker_id IS NOT NULL
  AND job.lease_expires_at < observed.observed_at
  AND attempt.finalized_at IS NULL
ORDER BY job.lease_expires_at, job.logical_job_id
LIMIT sqlc.arg(candidate_limit)::integer;

-- name: RescuePostgresJobAttempt :one
WITH observed AS MATERIALIZED (
    SELECT clock_timestamp() AS observed_at
),
locked_job AS MATERIALIZED (
    SELECT *
    FROM postgres_jobs
    WHERE logical_job_id = sqlc.arg(logical_job_id)::text
    FOR UPDATE
),
locked_attempt AS MATERIALIZED (
    SELECT attempt.*
    FROM postgres_job_attempts AS attempt
    JOIN locked_job ON locked_job.logical_job_id = attempt.logical_job_id
    WHERE attempt.attempt_generation = sqlc.arg(attempt_generation)::bigint
      AND attempt.recovery_generation = sqlc.arg(recovery_generation)::bigint
      AND attempt.worker_id = sqlc.arg(worker_id)::text
    FOR UPDATE OF attempt
),
current_expired AS MATERIALIZED (
    SELECT locked_job.logical_job_id
    FROM locked_job
    JOIN locked_attempt USING (logical_job_id)
    CROSS JOIN observed
    WHERE locked_attempt.finalized_at IS NULL
      AND locked_job.attempt_generation = sqlc.arg(attempt_generation)::bigint
      AND locked_job.recovery_generation = sqlc.arg(recovery_generation)::bigint
      AND locked_job.current_worker_id = sqlc.arg(worker_id)::text
      AND locked_job.state IN ('running', 'cancel_requested')
      AND locked_job.lease_expires_at < observed.observed_at
),
updated_job AS (
    UPDATE postgres_jobs AS job
    SET state = sqlc.arg(final_state)::text,
        available_at = CASE
            WHEN sqlc.arg(final_state)::text = 'retry_wait'
            THEN observed.observed_at
                + sqlc.arg(delay_microseconds)::bigint * interval '1 microsecond'
            ELSE job.available_at
        END,
        attempts_used = sqlc.arg(attempts_used)::integer,
        current_worker_id = NULL,
        lease_expires_at = NULL,
        updated_at = observed.observed_at,
        terminal_at = CASE
            WHEN sqlc.arg(final_state)::text = 'retry_wait' THEN NULL
            ELSE observed.observed_at
        END
    FROM current_expired, observed
    WHERE job.logical_job_id = current_expired.logical_job_id
    RETURNING job.logical_job_id
),
updated_attempt AS (
    UPDATE postgres_job_attempts AS attempt
    SET finalized_at = observed.observed_at,
        final_state = sqlc.arg(final_state)::text,
        outcome = sqlc.arg(outcome)::text,
        effect_status = sqlc.arg(effect_status)::text,
        failure_code = NULLIF(sqlc.arg(failure_code)::text, ''),
        retry_at = CASE
            WHEN sqlc.arg(final_state)::text = 'retry_wait'
            THEN observed.observed_at
                + sqlc.arg(delay_microseconds)::bigint * interval '1 microsecond'
            ELSE NULL
        END,
        attempts_used = sqlc.arg(attempts_used)::integer,
        elapsed_used_milliseconds = sqlc.arg(elapsed_used_milliseconds)::bigint
    FROM updated_job, observed
    WHERE attempt.logical_job_id = updated_job.logical_job_id
      AND attempt.attempt_generation = sqlc.arg(attempt_generation)::bigint
    RETURNING attempt.*
),
result_attempt AS MATERIALIZED (
    SELECT * FROM updated_attempt
    UNION ALL
    SELECT * FROM locked_attempt
    WHERE locked_attempt.finalized_at IS NOT NULL
    LIMIT 1
)
SELECT
    CASE
        WHEN EXISTS (SELECT 1 FROM updated_attempt) THEN 'applied'
        WHEN EXISTS (SELECT 1 FROM locked_attempt WHERE finalized_at IS NOT NULL) THEN 'repeated'
        WHEN NOT EXISTS (SELECT 1 FROM locked_job) THEN 'not_found'
        ELSE 'stale'
    END::text AS status,
    EXISTS (SELECT 1 FROM result_attempt)::boolean AS has_result,
    coalesce((SELECT final_state FROM result_attempt), '')::text AS final_state,
    coalesce((SELECT outcome FROM result_attempt), '')::text AS outcome,
    coalesce((SELECT effect_status FROM result_attempt), '')::text AS effect_status,
    coalesce((SELECT failure_code FROM result_attempt), '')::text AS failure_code,
    (SELECT retry_at FROM result_attempt)::timestamptz AS retry_at,
    coalesce((SELECT attempts_used FROM result_attempt), 0)::integer AS attempts_used,
    coalesce((SELECT elapsed_used_milliseconds FROM result_attempt), 0)::bigint AS elapsed_used_milliseconds,
    (SELECT finalized_at FROM result_attempt)::timestamptz AS finalized_at;

-- name: ObservePostgresJobs :one
WITH input_keys AS MATERIALIZED (
    SELECT kind.kind, args.args_version, policy.policy_version
    FROM unnest(sqlc.arg(kinds)::text[]) WITH ORDINALITY AS kind(kind, position)
    JOIN unnest(sqlc.arg(args_versions)::text[]) WITH ORDINALITY AS args(args_version, position) USING (position)
    JOIN unnest(sqlc.arg(policy_versions)::text[]) WITH ORDINALITY AS policy(policy_version, position) USING (position)
),
observed AS MATERIALIZED (
    SELECT clock_timestamp() AS observed_at
),
inventory AS MATERIALIZED (
    SELECT DISTINCT kind, args_version, policy_version
    FROM postgres_jobs
),
state_counts AS MATERIALIZED (
    SELECT state, count(*)::bigint AS job_count, min(available_at) AS oldest_available_at
    FROM postgres_jobs
    GROUP BY state
)
SELECT
    observed.observed_at::timestamptz AS observed_at,
    NOT EXISTS (
        SELECT 1
        FROM inventory
        WHERE NOT EXISTS (
            SELECT 1
            FROM input_keys
            WHERE input_keys.kind = inventory.kind
              AND input_keys.args_version = inventory.args_version
              AND input_keys.policy_version = inventory.policy_version
        )
    )::boolean AS compatible,
    ARRAY(SELECT kind FROM inventory ORDER BY kind, args_version, policy_version)::text[] AS retained_kinds,
    ARRAY(SELECT args_version FROM inventory ORDER BY kind, args_version, policy_version)::text[] AS retained_args_versions,
    ARRAY(SELECT policy_version FROM inventory ORDER BY kind, args_version, policy_version)::text[] AS retained_policy_versions,
    ARRAY(SELECT state FROM state_counts ORDER BY state)::text[] AS states,
    ARRAY(SELECT job_count FROM state_counts ORDER BY state)::bigint[] AS state_counts,
    ARRAY(SELECT oldest_available_at FROM state_counts ORDER BY state)::timestamptz[] AS oldest_available_at
FROM observed;
