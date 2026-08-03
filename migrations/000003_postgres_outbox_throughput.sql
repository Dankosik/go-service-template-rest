-- +goose Up

-- Wake relays on commit instead of only on the poll timer. A statement-level
-- trigger costs the appending transaction no extra client round trip, and
-- PostgreSQL collapses duplicate (channel, payload) notifications inside one
-- transaction. The signal is an optimization only: a relay that misses it
-- still claims the row on its next poll.
-- +goose StatementBegin
CREATE FUNCTION outbox_notify_appended() RETURNS trigger
LANGUAGE plpgsql
SET search_path = pg_catalog
AS $$
BEGIN
    PERFORM pg_notify('outbox_appended', '');
    RETURN NULL;
END
$$;
-- +goose StatementEnd

CREATE TRIGGER outbox_events_notify_appended
    AFTER INSERT ON outbox_events
    FOR EACH STATEMENT
    EXECUTE FUNCTION outbox_notify_appended();

-- State observation reads the pending set instead of every retained published
-- row, so its cost tracks backlog rather than retention volume. Build this
-- index CONCURRENTLY out of band before migrating a populated deployment: the
-- in-transaction statement below holds a SHARE lock that blocks appends for its
-- duration, and becomes a no-op once the index exists.
CREATE INDEX IF NOT EXISTS outbox_events_pending_idx
    ON outbox_events (created_at)
    WHERE published_at IS NULL;

-- +goose Down

DROP INDEX IF EXISTS outbox_events_pending_idx;
DROP TRIGGER outbox_events_notify_appended ON outbox_events;
DROP FUNCTION outbox_notify_appended();
