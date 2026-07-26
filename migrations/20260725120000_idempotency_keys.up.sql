-- Idempotency keys: one row per unsafe request a client asked to be executed once.
--
-- The version is a timestamp rather than 0001 on purpose. golang-migrate parses the
-- leading digits as the version, so `0001` and `000001` are the same version — and
-- `000001_<name>` is exactly what `migrate create -seq` produces for a service's
-- first migration. A template that claimed version 1 would make that first
-- migration fail with "duplicate migration file" at deploy time. The timestamp
-- convention is golang-migrate's own default and cannot collide with a hand-picked
-- sequence number.
--
-- The primary key is what makes the claim atomic. Reserve is a single
-- INSERT ... ON CONFLICT DO NOTHING, so exactly one of two concurrent attempts
-- affects a row and the loser learns it lost; the SELECT-then-INSERT shape a
-- service reaches for instead has a window where both attempts see no row.
CREATE TABLE idempotency_keys (
    key text PRIMARY KEY,

    -- fingerprint identifies the intent the key stands for. The same key presented
    -- with a different request is a client defect and is refused, rather than
    -- answered with somebody else's result.
    fingerprint text NOT NULL,

    -- The response held for replay. NULL until the attempt completes, which is how
    -- an in-flight reservation is told apart from a finished one.
    status integer,
    headers jsonb,
    body bytea,

    -- replayable is false when the response was too large to hold, or was streamed
    -- in a way the recorder could not capture. The key is still spent: giving up on
    -- the exact bytes must not give up on the work running once.
    replayable boolean NOT NULL DEFAULT true,

    completed_at timestamptz,

    -- expires_at bounds the table. Without the sweep this index supports, the table
    -- keeps every key any client ever sent for the life of the service.
    expires_at timestamptz NOT NULL
);

CREATE INDEX idempotency_keys_expires_at_idx ON idempotency_keys (expires_at);
