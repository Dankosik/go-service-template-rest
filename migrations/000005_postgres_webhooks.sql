-- +goose Up

CREATE SEQUENCE webhook_fairness_sequence AS bigint;

CREATE TABLE webhook_clock (
    singleton boolean PRIMARY KEY DEFAULT true CHECK (singleton),
    high_water timestamptz NOT NULL,
    regression boolean NOT NULL DEFAULT false,
    observed_at timestamptz NOT NULL,
    CHECK (isfinite(high_water) AND isfinite(observed_at))
);

INSERT INTO webhook_clock (high_water, observed_at)
VALUES (clock_timestamp(), clock_timestamp());

CREATE TABLE webhook_destinations (
    owner_scope text COLLATE "C" NOT NULL,
    destination_id text COLLATE "C" NOT NULL,
    generation bigint NOT NULL,
    ownership_verification_receipt text COLLATE "C" NOT NULL,
    url text NOT NULL,
    selection_revision text COLLATE "C" NOT NULL,
    payload_version_preference text COLLATE "C" NOT NULL,
    signature_profile text COLLATE "C" NOT NULL,
    signing_authority_binding text COLLATE "C" NOT NULL,
    policy jsonb NOT NULL,
    policy_fingerprint bytea NOT NULL,
    destination_concurrency integer NOT NULL,
    global_concurrency integer NOT NULL,
    control_revision bigint NOT NULL DEFAULT 1,
    required_secret_revision bigint NOT NULL,
    key_state_revision bigint NOT NULL DEFAULT 1,
    active_key_reference text COLLATE "C" NOT NULL,
    predecessor_key_reference text COLLATE "C",
    predecessor_valid_until timestamptz,
    disposition text COLLATE "C" NOT NULL DEFAULT 'active',
    last_considered_sequence bigint NOT NULL DEFAULT nextval('webhook_fairness_sequence'),
    created_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL,
    PRIMARY KEY (owner_scope, destination_id, generation),
    CHECK (generation > 0 AND control_revision > 0 AND required_secret_revision > 0 AND key_state_revision > 0),
    CHECK (octet_length(owner_scope) BETWEEN 1 AND 256 AND owner_scope !~ '[[:space:][:cntrl:]]'),
    CHECK (octet_length(destination_id) BETWEEN 1 AND 256 AND destination_id !~ '[[:space:][:cntrl:]]'),
    CHECK (octet_length(ownership_verification_receipt) BETWEEN 1 AND 256 AND ownership_verification_receipt !~ '[[:space:][:cntrl:]]'),
    CHECK (octet_length(url) BETWEEN 1 AND 2048),
    CHECK (octet_length(selection_revision) BETWEEN 1 AND 256 AND selection_revision !~ '[[:space:][:cntrl:]]'),
    CHECK (octet_length(payload_version_preference) BETWEEN 1 AND 256 AND payload_version_preference !~ '[[:space:][:cntrl:]]'),
    CHECK (signature_profile = 'v1'),
    CHECK (octet_length(signing_authority_binding) BETWEEN 1 AND 256 AND signing_authority_binding !~ '[[:space:][:cntrl:]]'),
    CHECK (octet_length(active_key_reference) BETWEEN 1 AND 256 AND active_key_reference !~ '[[:space:][:cntrl:]]'),
    CHECK (predecessor_key_reference IS NULL OR (octet_length(predecessor_key_reference) BETWEEN 1 AND 256 AND predecessor_key_reference !~ '[[:space:][:cntrl:]]')),
    CHECK ((predecessor_key_reference IS NULL) = (predecessor_valid_until IS NULL)),
    CHECK (octet_length(policy_fingerprint) = 32 AND destination_concurrency BETWEEN 1 AND 256 AND global_concurrency BETWEEN destination_concurrency AND 256),
    CHECK (disposition IN ('active', 'automatically_paused', 'administratively_disabled', 'retired')),
    CHECK (isfinite(created_at) AND isfinite(updated_at) AND (predecessor_valid_until IS NULL OR isfinite(predecessor_valid_until)))
);

CREATE INDEX webhook_destinations_fair_claim_idx
    ON webhook_destinations (last_considered_sequence, owner_scope, destination_id, generation)
    WHERE disposition IN ('active', 'automatically_paused');

CREATE INDEX webhook_destinations_secret_revision_idx
    ON webhook_destinations (required_secret_revision DESC)
    WHERE disposition IN ('active', 'automatically_paused');

CREATE TABLE webhook_events (
    owner_scope text COLLATE "C" NOT NULL,
    business_event_id text COLLATE "C" NOT NULL,
    acceptance_id text COLLATE "C" NOT NULL,
    fanout_snapshot_id text COLLATE "C" NOT NULL,
    event_type text COLLATE "C" NOT NULL,
    business_schema_version text COLLATE "C" NOT NULL,
    content_type text COLLATE "C" NOT NULL,
    body bytea NOT NULL,
    delivery_envelope_version text COLLATE "C" NOT NULL,
    subscriber_policy_revision text COLLATE "C" NOT NULL,
    origin_trace_link text,
    intent_fingerprint bytea NOT NULL,
    retention_policy_identity text COLLATE "C" NOT NULL,
    control_revision bigint NOT NULL DEFAULT 1,
    accepted_at timestamptz NOT NULL,
    PRIMARY KEY (owner_scope, business_event_id),
    UNIQUE (owner_scope, acceptance_id),
    UNIQUE (owner_scope, fanout_snapshot_id),
    CHECK (octet_length(owner_scope) BETWEEN 1 AND 256 AND owner_scope !~ '[[:space:][:cntrl:]]'),
    CHECK (octet_length(business_event_id) BETWEEN 1 AND 256 AND business_event_id !~ '[[:space:][:cntrl:]]'),
    CHECK (octet_length(acceptance_id) BETWEEN 1 AND 256 AND acceptance_id !~ '[[:space:][:cntrl:]]'),
    CHECK (octet_length(fanout_snapshot_id) BETWEEN 1 AND 256 AND fanout_snapshot_id !~ '[[:space:][:cntrl:]]'),
    CHECK (octet_length(event_type) BETWEEN 1 AND 256 AND event_type !~ '[[:space:][:cntrl:]]'),
    CHECK (octet_length(business_schema_version) BETWEEN 1 AND 256 AND business_schema_version !~ '[[:space:][:cntrl:]]'),
    CHECK (octet_length(content_type) BETWEEN 1 AND 256 AND content_type !~ '[[:cntrl:]]'),
    CHECK (octet_length(body) BETWEEN 1 AND 262144 AND octet_length(intent_fingerprint) = 32),
    CHECK (octet_length(delivery_envelope_version) BETWEEN 1 AND 256 AND delivery_envelope_version !~ '[[:space:][:cntrl:]]'),
    CHECK (octet_length(subscriber_policy_revision) BETWEEN 1 AND 256 AND subscriber_policy_revision !~ '[[:space:][:cntrl:]]'),
    CHECK (octet_length(retention_policy_identity) BETWEEN 1 AND 256 AND retention_policy_identity !~ '[[:space:][:cntrl:]]'),
    CHECK (control_revision > 0 AND isfinite(accepted_at))
);

CREATE TABLE webhook_fanouts (
    owner_scope text COLLATE "C" NOT NULL,
    fanout_snapshot_id text COLLATE "C" NOT NULL,
    business_event_id text COLLATE "C" NOT NULL,
    member_count integer NOT NULL,
    member_fingerprint bytea NOT NULL,
    accepted_at timestamptz NOT NULL,
    PRIMARY KEY (owner_scope, fanout_snapshot_id),
    FOREIGN KEY (owner_scope, business_event_id) REFERENCES webhook_events (owner_scope, business_event_id) ON DELETE CASCADE,
    CHECK (member_count BETWEEN 1 AND 1000 AND octet_length(member_fingerprint) = 32 AND isfinite(accepted_at))
);

CREATE TABLE webhook_deliveries (
    owner_scope text COLLATE "C" NOT NULL,
    delivery_id text COLLATE "C" NOT NULL,
    business_event_id text COLLATE "C" NOT NULL,
    fanout_snapshot_id text COLLATE "C" NOT NULL,
    destination_id text COLLATE "C" NOT NULL,
    destination_generation bigint NOT NULL,
    url_snapshot text NOT NULL,
    policy_snapshot jsonb NOT NULL,
    state text COLLATE "C" NOT NULL DEFAULT 'ready',
    current_cycle bigint NOT NULL DEFAULT 0,
    next_due_at timestamptz NOT NULL,
    lease_owner text COLLATE "C",
    lease_expires_at timestamptz,
    fence bigint NOT NULL DEFAULT 0,
    cumulative_summary text COLLATE "C" NOT NULL DEFAULT 'none',
    sendable boolean NOT NULL DEFAULT true,
    redrive_eligible_until timestamptz NOT NULL,
    terminal_at timestamptz,
    created_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL,
    PRIMARY KEY (owner_scope, delivery_id),
    UNIQUE (owner_scope, fanout_snapshot_id, destination_id, destination_generation),
    FOREIGN KEY (owner_scope, business_event_id) REFERENCES webhook_events (owner_scope, business_event_id) ON DELETE CASCADE,
    FOREIGN KEY (owner_scope, fanout_snapshot_id) REFERENCES webhook_fanouts (owner_scope, fanout_snapshot_id) ON DELETE CASCADE,
    FOREIGN KEY (owner_scope, destination_id, destination_generation) REFERENCES webhook_destinations (owner_scope, destination_id, generation),
    CHECK (octet_length(delivery_id) BETWEEN 1 AND 256 AND delivery_id !~ '[[:space:][:cntrl:]]'),
    CHECK (state IN ('ready', 'scheduled', 'in_flight', 'suspended', 'terminal', 'quarantined')),
    CHECK (current_cycle >= 0 AND fence >= 0),
    CHECK (cumulative_summary IN ('none', 'http_accepted', 'http_rejected', 'locally_denied', 'attempts_exhausted', 'outcome_unknown', 'closed_unknown')),
    CHECK ((lease_owner IS NULL) = (lease_expires_at IS NULL)),
    CHECK (isfinite(next_due_at) AND isfinite(redrive_eligible_until) AND isfinite(created_at) AND isfinite(updated_at)),
    CHECK (lease_expires_at IS NULL OR isfinite(lease_expires_at)),
    CHECK (terminal_at IS NULL OR isfinite(terminal_at))
);

CREATE INDEX webhook_deliveries_due_idx
    ON webhook_deliveries (owner_scope, destination_id, destination_generation, next_due_at, delivery_id)
    WHERE state IN ('ready', 'scheduled') AND sendable;

CREATE TABLE webhook_cycles (
    owner_scope text COLLATE "C" NOT NULL,
    delivery_id text COLLATE "C" NOT NULL,
    cycle_number bigint NOT NULL,
    cycle_kind text COLLATE "C" NOT NULL,
    authorizing_action_id text COLLATE "C",
    accepted_at timestamptz NOT NULL,
    deadline_at timestamptz NOT NULL,
    maximum_attempts integer NOT NULL,
    attempts_used integer NOT NULL DEFAULT 0,
    disposition text COLLATE "C" NOT NULL DEFAULT 'active',
    finalized_at timestamptz,
    PRIMARY KEY (owner_scope, delivery_id, cycle_number),
    FOREIGN KEY (owner_scope, delivery_id) REFERENCES webhook_deliveries (owner_scope, delivery_id) ON DELETE CASCADE,
    CHECK (cycle_number >= 0 AND maximum_attempts BETWEEN 1 AND 100 AND attempts_used BETWEEN 0 AND maximum_attempts),
    CHECK (cycle_kind IN ('automatic', 'redrive')),
    CHECK (disposition IN ('active', 'http_accepted', 'http_rejected', 'locally_denied', 'attempts_exhausted', 'outcome_unknown', 'closed_unknown', 'privacy_deleted')),
    CHECK (isfinite(accepted_at) AND isfinite(deadline_at) AND deadline_at > accepted_at),
    CHECK ((disposition = 'active') = (finalized_at IS NULL)),
    CHECK (finalized_at IS NULL OR isfinite(finalized_at))
);

CREATE TABLE webhook_attempts (
    owner_scope text COLLATE "C" NOT NULL,
    delivery_id text COLLATE "C" NOT NULL,
    cycle_number bigint NOT NULL,
    attempt_id text COLLATE "C" NOT NULL,
    fence bigint NOT NULL,
    capacity_slot integer NOT NULL,
    attempted_at timestamptz NOT NULL,
    lease_expires_at timestamptz NOT NULL,
    key_reference text COLLATE "C",
    signature_header_digest bytea,
    payload_digest bytea NOT NULL,
    payload_bytes integer NOT NULL,
    dns_set_digest bytea,
    selected_address bytea,
    send_authorized boolean NOT NULL DEFAULT false,
    may_have_sent boolean NOT NULL DEFAULT false,
    response_header_bytes integer,
    response_body_bytes integer,
    response_status integer,
    retry_after text,
    outcome_class text COLLATE "C",
    finalized_at timestamptz,
    PRIMARY KEY (owner_scope, delivery_id, cycle_number, attempt_id),
    UNIQUE (owner_scope, delivery_id, cycle_number, fence),
    FOREIGN KEY (owner_scope, delivery_id, cycle_number) REFERENCES webhook_cycles (owner_scope, delivery_id, cycle_number) ON DELETE CASCADE,
    CHECK (octet_length(attempt_id) BETWEEN 1 AND 256 AND attempt_id !~ '[[:space:][:cntrl:]]'),
    CHECK (fence > 0 AND capacity_slot > 0 AND payload_bytes BETWEEN 1 AND 262144),
    CHECK (octet_length(payload_digest) = 32),
    CHECK (signature_header_digest IS NULL OR octet_length(signature_header_digest) = 32),
    CHECK (dns_set_digest IS NULL OR octet_length(dns_set_digest) = 32),
    CHECK (selected_address IS NULL OR octet_length(selected_address) IN (4, 16)),
    CHECK (response_header_bytes IS NULL OR response_header_bytes >= 0),
    CHECK (response_body_bytes IS NULL OR response_body_bytes >= 0),
    CHECK (response_status IS NULL OR response_status BETWEEN 100 AND 599),
    CHECK (outcome_class IS NULL OR outcome_class IN ('http_accepted', 'definitely_not_sent_retryable', 'retryable_http_ambiguous', 'transport_ambiguous', 'http_rejected', 'locally_denied', 'attempts_exhausted', 'outcome_unknown', 'closed_unknown')),
    CHECK (isfinite(attempted_at) AND isfinite(lease_expires_at)),
    CHECK ((outcome_class IS NULL) = (finalized_at IS NULL)),
    CHECK (finalized_at IS NULL OR isfinite(finalized_at))
);

CREATE INDEX webhook_attempts_expired_idx
    ON webhook_attempts (lease_expires_at, owner_scope, delivery_id, cycle_number, attempt_id)
    WHERE finalized_at IS NULL;

CREATE TABLE webhook_capacity_slots (
    slot_number integer PRIMARY KEY,
    capacity_revision bigint NOT NULL,
    owner_scope text COLLATE "C",
    delivery_id text COLLATE "C",
    cycle_number bigint,
    attempt_id text COLLATE "C",
    lease_expires_at timestamptz,
    fence bigint,
    CHECK (slot_number > 0 AND capacity_revision > 0),
    CHECK ((owner_scope IS NULL) = (delivery_id IS NULL) AND (delivery_id IS NULL) = (cycle_number IS NULL) AND (cycle_number IS NULL) = (attempt_id IS NULL) AND (attempt_id IS NULL) = (lease_expires_at IS NULL) AND (lease_expires_at IS NULL) = (fence IS NULL)),
    CHECK (lease_expires_at IS NULL OR isfinite(lease_expires_at))
);

CREATE INDEX webhook_capacity_slots_lease_idx
    ON webhook_capacity_slots (lease_expires_at, slot_number)
    WHERE attempt_id IS NOT NULL;

CREATE TABLE webhook_operator_actions (
    owner_scope text COLLATE "C" NOT NULL,
    action_id text COLLATE "C" NOT NULL,
    encoding_version text COLLATE "C" NOT NULL,
    request_fingerprint bytea NOT NULL,
    actor_reference text COLLATE "C" NOT NULL,
    action_kind text COLLATE "C" NOT NULL,
    target_kind text COLLATE "C" NOT NULL,
    target_id text COLLATE "C" NOT NULL,
    target_generation bigint NOT NULL,
    expected_state text COLLATE "C" NOT NULL,
    reason text COLLATE "C" NOT NULL,
    duplicate_risk_acknowledged boolean NOT NULL DEFAULT false,
    state text COLLATE "C" NOT NULL DEFAULT 'completed',
    result text COLLATE "C" NOT NULL,
    created_at timestamptz NOT NULL,
    completed_at timestamptz,
    PRIMARY KEY (owner_scope, action_id),
    CHECK (octet_length(action_id) BETWEEN 1 AND 256 AND action_id !~ '[[:space:][:cntrl:]]'),
    CHECK (encoding_version = 'webhook-operator-action-v1' AND octet_length(request_fingerprint) = 32),
    CHECK (action_kind IN ('destination_state', 'key_rotation', 'redrive', 'close_unknown', 'privacy_delete', 'namespace_retire')),
    CHECK (target_kind IN ('destination', 'delivery', 'event', 'namespace')),
    CHECK (target_generation >= 0 AND state IN ('pending', 'completed')),
    CHECK (result IN ('applied', 'not_found', 'state_conflict', 'audit_conflict', 'unauthorized', 'rejected', 'unknown')),
    CHECK (isfinite(created_at) AND (completed_at IS NULL OR isfinite(completed_at))),
    CHECK ((state = 'completed') = (completed_at IS NOT NULL))
);

CREATE TABLE webhook_tombstones (
    owner_scope text COLLATE "C" NOT NULL,
    target_kind text COLLATE "C" NOT NULL,
    target_id text COLLATE "C" NOT NULL,
    acceptance_id text COLLATE "C",
    fanout_snapshot_id text COLLATE "C",
    delivery_identities jsonb NOT NULL DEFAULT '[]',
    destination_identities jsonb NOT NULL DEFAULT '[]',
    last_semantic_class text COLLATE "C" NOT NULL,
    action_id text COLLATE "C" NOT NULL,
    action_encoding_version text COLLATE "C" NOT NULL,
    request_fingerprint bytea NOT NULL,
    first_disposition text COLLATE "C" NOT NULL,
    deletion_authority text COLLATE "C" NOT NULL,
    created_at timestamptz NOT NULL,
    PRIMARY KEY (owner_scope, target_kind, target_id),
    UNIQUE (owner_scope, action_id),
    CHECK (target_kind IN ('event', 'namespace')),
    CHECK (octet_length(request_fingerprint) = 32 AND action_encoding_version = 'webhook-operator-action-v1'),
    CHECK (last_semantic_class IN ('none', 'http_accepted', 'http_rejected', 'locally_denied', 'attempts_exhausted', 'outcome_unknown', 'closed_unknown', 'privacy_deleted')),
    CHECK (first_disposition IN ('applied', 'pending', 'completed', 'not_found', 'state_conflict', 'audit_conflict', 'rejected', 'unknown')),
    CHECK (isfinite(created_at))
);

-- +goose Down

DROP TABLE webhook_tombstones;
DROP TABLE webhook_operator_actions;
DROP TABLE webhook_capacity_slots;
DROP TABLE webhook_attempts;
DROP TABLE webhook_cycles;
DROP TABLE webhook_deliveries;
DROP TABLE webhook_fanouts;
DROP TABLE webhook_events;
DROP TABLE webhook_destinations;
DROP TABLE webhook_clock;
DROP SEQUENCE webhook_fairness_sequence;
