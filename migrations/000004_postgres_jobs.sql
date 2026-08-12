-- +goose Up

CREATE TABLE postgres_jobs (
    logical_job_id text COLLATE "C" PRIMARY KEY,
    producer_scope text COLLATE "C" NOT NULL,
    producer_key text COLLATE "C" NOT NULL,
    occurrence_scope text COLLATE "C" NOT NULL,
    occurrence_id text COLLATE "C" NOT NULL,
    effect_scope text COLLATE "C" NOT NULL,
    effect_key text COLLATE "C" NOT NULL,
    intent_fingerprint bytea NOT NULL,
    kind text COLLATE "C" NOT NULL,
    args_version text COLLATE "C" NOT NULL,
    policy_version text COLLATE "C" NOT NULL,
    payload bytea NOT NULL,
    work_class text COLLATE "C" NOT NULL,
    state text COLLATE "C" NOT NULL,
    available_at timestamptz NOT NULL,
    recovery_generation bigint NOT NULL DEFAULT 0,
    attempt_generation bigint NOT NULL DEFAULT 0,
    attempts_used integer NOT NULL DEFAULT 0,
    budget_started_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    current_worker_id text COLLATE "C",
    lease_expires_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    updated_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    terminal_at timestamptz,
    CONSTRAINT postgres_jobs_producer_key UNIQUE (producer_scope, producer_key),
    CONSTRAINT postgres_jobs_occurrence_key UNIQUE (occurrence_scope, occurrence_id),
    CONSTRAINT postgres_jobs_effect_key UNIQUE (effect_scope, effect_key),
    CONSTRAINT postgres_jobs_identity_check CHECK (
        octet_length(logical_job_id) BETWEEN 1 AND 256 AND logical_job_id !~ '[[:cntrl:]]'
        AND octet_length(producer_scope) BETWEEN 1 AND 256 AND producer_scope !~ '[[:cntrl:]]'
        AND octet_length(producer_key) BETWEEN 1 AND 256 AND producer_key !~ '[[:cntrl:]]'
        AND octet_length(occurrence_scope) BETWEEN 1 AND 256 AND occurrence_scope !~ '[[:cntrl:]]'
        AND octet_length(occurrence_id) BETWEEN 1 AND 256 AND occurrence_id !~ '[[:cntrl:]]'
        AND octet_length(effect_scope) BETWEEN 1 AND 256 AND effect_scope !~ '[[:cntrl:]]'
        AND octet_length(effect_key) BETWEEN 1 AND 256 AND effect_key !~ '[[:cntrl:]]'
    ),
    CONSTRAINT postgres_jobs_revision_check CHECK (
        octet_length(kind) BETWEEN 1 AND 256 AND kind !~ '[[:cntrl:]]'
        AND octet_length(args_version) BETWEEN 1 AND 256 AND args_version !~ '[[:cntrl:]]'
        AND octet_length(policy_version) BETWEEN 1 AND 256 AND policy_version !~ '[[:cntrl:]]'
    ),
    CONSTRAINT postgres_jobs_intent_check CHECK (
        octet_length(intent_fingerprint) = 32
        AND octet_length(payload) BETWEEN 1 AND 262144
        AND payload IS JSON
    ),
    CONSTRAINT postgres_jobs_work_class_check CHECK (work_class = 'neutral'),
    CONSTRAINT postgres_jobs_state_check CHECK (
        state IN (
            'ready', 'scheduled', 'retry_wait', 'running', 'cancel_requested',
            'succeeded', 'cancelled', 'exhausted', 'permanent', 'poison', 'outcome_unknown'
        )
    ),
    CONSTRAINT postgres_jobs_generation_check CHECK (
        recovery_generation >= 0 AND attempt_generation >= 0 AND attempts_used >= 0
    ),
    CONSTRAINT postgres_jobs_owner_check CHECK (
        (state IN ('running', 'cancel_requested')) = (current_worker_id IS NOT NULL)
        AND (current_worker_id IS NULL) = (lease_expires_at IS NULL)
        AND (
            current_worker_id IS NULL
            OR (
                octet_length(current_worker_id) BETWEEN 1 AND 256
                AND current_worker_id !~ '[[:cntrl:]]'
            )
        )
    ),
    CONSTRAINT postgres_jobs_terminal_check CHECK (
        (state IN ('succeeded', 'cancelled', 'exhausted', 'permanent', 'poison', 'outcome_unknown'))
        = (terminal_at IS NOT NULL)
    ),
    CONSTRAINT postgres_jobs_timestamp_check CHECK (
        isfinite(available_at)
        AND isfinite(budget_started_at)
        AND isfinite(created_at)
        AND isfinite(updated_at)
        AND (lease_expires_at IS NULL OR isfinite(lease_expires_at))
        AND (terminal_at IS NULL OR isfinite(terminal_at))
    )
);

CREATE INDEX postgres_jobs_claim_idx
    ON postgres_jobs (work_class, available_at, logical_job_id)
    WHERE state IN ('ready', 'scheduled', 'retry_wait');

CREATE INDEX postgres_jobs_revision_idx
    ON postgres_jobs (kind, args_version, policy_version);

CREATE INDEX postgres_jobs_lease_idx
    ON postgres_jobs (lease_expires_at, logical_job_id)
    WHERE state IN ('running', 'cancel_requested');

CREATE INDEX postgres_jobs_observation_idx
    ON postgres_jobs (state, work_class, available_at, logical_job_id);

CREATE TABLE postgres_job_attempts (
    logical_job_id text COLLATE "C" NOT NULL REFERENCES postgres_jobs (logical_job_id),
    attempt_generation bigint NOT NULL,
    recovery_generation bigint NOT NULL,
    attempt_number integer NOT NULL,
    worker_id text COLLATE "C" NOT NULL,
    started_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    lease_expires_at timestamptz NOT NULL,
    finalized_at timestamptz,
    final_state text COLLATE "C",
    outcome text COLLATE "C",
    effect_status text COLLATE "C",
    failure_code text COLLATE "C",
    retry_at timestamptz,
    attempts_used integer,
    elapsed_used_milliseconds bigint,
    PRIMARY KEY (logical_job_id, attempt_generation),
    CONSTRAINT postgres_job_attempts_generation_check CHECK (
        attempt_generation > 0
        AND recovery_generation >= 0
        AND attempt_number > 0
        AND (attempts_used IS NULL OR attempts_used > 0)
        AND (elapsed_used_milliseconds IS NULL OR elapsed_used_milliseconds >= 0)
    ),
    CONSTRAINT postgres_job_attempts_worker_check CHECK (
        octet_length(worker_id) BETWEEN 1 AND 256 AND worker_id !~ '[[:cntrl:]]'
    ),
    CONSTRAINT postgres_job_attempts_state_check CHECK (
        final_state IS NULL
        OR final_state IN ('retry_wait', 'succeeded', 'cancelled', 'exhausted', 'permanent', 'poison', 'outcome_unknown')
    ),
    CONSTRAINT postgres_job_attempts_outcome_check CHECK (
        outcome IS NULL
        OR outcome IN ('success', 'retryable', 'permanent', 'poison', 'timeout', 'cancelled', 'panic', 'lost', 'unknown')
    ),
    CONSTRAINT postgres_job_attempts_effect_check CHECK (
        effect_status IS NULL OR effect_status IN ('none', 'completed', 'partial', 'unknown')
    ),
    CONSTRAINT postgres_job_attempts_failure_check CHECK (
        failure_code IS NULL
        OR (
            octet_length(failure_code) BETWEEN 1 AND 256
            AND failure_code !~ '[[:cntrl:]]'
        )
    ),
    CONSTRAINT postgres_job_attempts_final_check CHECK (
        (
            finalized_at IS NULL
            AND final_state IS NULL
            AND outcome IS NULL
            AND effect_status IS NULL
            AND failure_code IS NULL
            AND retry_at IS NULL
            AND attempts_used IS NULL
            AND elapsed_used_milliseconds IS NULL
        ) OR (
            finalized_at IS NOT NULL
            AND final_state IS NOT NULL
            AND outcome IS NOT NULL
            AND effect_status IS NOT NULL
            AND attempts_used IS NOT NULL
            AND elapsed_used_milliseconds IS NOT NULL
            AND (final_state = 'retry_wait') = (retry_at IS NOT NULL)
        )
    ),
    CONSTRAINT postgres_job_attempts_timestamp_check CHECK (
        isfinite(started_at)
        AND isfinite(lease_expires_at)
        AND (finalized_at IS NULL OR isfinite(finalized_at))
        AND (retry_at IS NULL OR isfinite(retry_at))
    )
);

CREATE INDEX postgres_job_attempts_lease_idx
    ON postgres_job_attempts (lease_expires_at, logical_job_id, attempt_generation)
    WHERE finalized_at IS NULL;

CREATE TABLE postgres_job_claim_scopes (
    work_class text COLLATE "C" PRIMARY KEY,
    paused boolean NOT NULL DEFAULT false,
    scope_generation bigint NOT NULL DEFAULT 0,
    updated_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    CONSTRAINT postgres_job_claim_scopes_class_check CHECK (work_class = 'neutral'),
    CONSTRAINT postgres_job_claim_scopes_generation_check CHECK (scope_generation >= 0),
    CONSTRAINT postgres_job_claim_scopes_timestamp_check CHECK (isfinite(updated_at))
);

INSERT INTO postgres_job_claim_scopes (work_class) VALUES ('neutral');

CREATE TABLE postgres_job_actions (
    action_id text COLLATE "C" PRIMARY KEY,
    request_fingerprint bytea NOT NULL,
    actor_id text COLLATE "C" NOT NULL,
    action_kind text COLLATE "C" NOT NULL,
    target_scope text COLLATE "C",
    logical_job_id text COLLATE "C",
    expected_state text COLLATE "C",
    expected_generation bigint,
    reason text COLLATE "C" NOT NULL,
    result text COLLATE "C" NOT NULL,
    created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    completed_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    CONSTRAINT postgres_job_actions_identity_check CHECK (
        octet_length(action_id) BETWEEN 1 AND 256 AND action_id !~ '[[:cntrl:]]'
        AND octet_length(actor_id) BETWEEN 1 AND 256 AND actor_id !~ '[[:cntrl:]]'
        AND octet_length(request_fingerprint) = 32
    ),
    CONSTRAINT postgres_job_actions_kind_check CHECK (
        action_kind IN ('pause', 'resume', 'cancel', 'redrive', 'delete')
    ),
    CONSTRAINT postgres_job_actions_target_check CHECK (
        (
            action_kind IN ('pause', 'resume')
            AND target_scope = 'neutral'
            AND logical_job_id IS NULL
        ) OR (
            action_kind IN ('cancel', 'redrive', 'delete')
            AND target_scope IS NULL
            AND logical_job_id IS NOT NULL
        )
    ),
    CONSTRAINT postgres_job_actions_target_value_check CHECK (
        (target_scope IS NULL OR (
            octet_length(target_scope) BETWEEN 1 AND 256 AND target_scope !~ '[[:cntrl:]]'
        ))
        AND (logical_job_id IS NULL OR (
            octet_length(logical_job_id) BETWEEN 1 AND 256 AND logical_job_id !~ '[[:cntrl:]]'
        ))
    ),
    CONSTRAINT postgres_job_actions_expectation_check CHECK (
        (expected_state IS NULL OR expected_state IN (
            'ready', 'scheduled', 'retry_wait', 'running', 'cancel_requested',
            'succeeded', 'cancelled', 'exhausted', 'permanent', 'poison', 'outcome_unknown'
        ))
        AND (expected_generation IS NULL OR expected_generation >= 0)
        AND (expected_state IS NOT NULL OR expected_generation IS NOT NULL)
    ),
    CONSTRAINT postgres_job_actions_reason_check CHECK (
        octet_length(reason) BETWEEN 1 AND 256 AND reason !~ '[[:cntrl:]]'
    ),
    CONSTRAINT postgres_job_actions_result_check CHECK (
        result IN ('applied', 'not_found', 'state_conflict', 'audit_conflict', 'unauthorized', 'rejected', 'unknown')
    ),
    CONSTRAINT postgres_job_actions_timestamp_check CHECK (
        isfinite(created_at) AND isfinite(completed_at) AND completed_at >= created_at
    )
);

CREATE INDEX postgres_job_actions_job_idx
    ON postgres_job_actions (logical_job_id, created_at)
    WHERE logical_job_id IS NOT NULL;

-- +goose Down

DROP TABLE postgres_job_actions;
DROP TABLE postgres_job_claim_scopes;
DROP TABLE postgres_job_attempts;
DROP TABLE postgres_jobs;
