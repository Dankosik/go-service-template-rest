-- +goose Up

-- The claim update rewrites only lease and attempt columns, and no index
-- references any of them. PostgreSQL can therefore keep the new row version on
-- the same heap page and skip every index insert, but only while that page has
-- free space. Default fillfactor packs pages full at insert time, so the very
-- first update of a freshly appended row leaves its page and rewrites all of
-- its index entries. Publication still changes indexed columns and stays a
-- normal indexed update, so the claim is the transition this reserve buys.
--
-- A batch claim leases consecutive rows, which is most of one heap page at a
-- time, so the page must hold a second version of every live row it carries.
-- That makes just under half the page the useful reserve rather than the
-- conventional small one. Measured over insert, claim, and publish of a
-- 2,000-event backlog on the repository's PostgreSQL 17 fixture, claims that
-- stayed heap-only and the resulting cycle WAL were:
--
--   fillfactor  100     70     60     50     45     40     30
--   HOT claims  201    845   1168   1700   1996   2000   2000
--   cycle WAL  2.12M  1.53M  1.31M  1.06M  0.97M  0.94M  0.94M
--   heap       1.18M  1.07M  1.00M  0.94M  0.93M  0.95M  1.29M
--
-- 45 is the knee: every claim is heap-only and the heap is at its smallest,
-- because the version sprawl a packed page causes costs more than the reserve
-- does. Below it the unused reserve starts growing the table again. With a
-- 1 KiB payload the same setting moved cycle WAL from 6.28M to 1.00M and the
-- heap from 8.15M to 5.50M, so this is a ratio rather than a value tuned to one
-- payload size. Re-measure if the claim stops updating whole pages at a time.
ALTER TABLE outbox_events SET (fillfactor = 45);

-- Autovacuum's default scale factors are proportional to table size, so a
-- seven-day retention window makes maintenance wait longer the more rows are
-- retained even though the churn that produces dead tuples is unrelated to how
-- many published rows are being kept: at the 0.2 default a one-million-row
-- table tolerates 200,000 dead tuples before its first vacuum. These values
-- keep a floor that does not move with retention and a 1% slope that still
-- scales, which is a starting point rather than a measured optimum. The
-- analyze setting also keeps the planner's row estimate current, which the
-- published-retained observation reads directly.
--
-- Reopen from the deployment's own signals: raise the thresholds when
-- autovacuum runs back to back on this table, since each pass reads its
-- indexes, and lower them when pg_stat_user_tables.n_dead_tup or the relation
-- size keeps growing between passes.
ALTER TABLE outbox_events SET (
    autovacuum_vacuum_scale_factor = 0.01,
    autovacuum_vacuum_threshold = 10000,
    autovacuum_vacuum_insert_scale_factor = 0.01,
    autovacuum_vacuum_insert_threshold = 10000,
    autovacuum_analyze_scale_factor = 0.01,
    autovacuum_analyze_threshold = 5000
);

-- Every append and every publication for an ordering key updates that key's
-- single head row, and a batched ordered finalization advances one head per
-- event in the lease, so heads face the same whole-page update shape. Over
-- 2,000 ordered appends and publications across 200 keys, heap-only head
-- updates were 3064/3800 at fillfactor 100, 3652/3800 at 70, and 3800/3800 at
-- 45, where the head relation was also 43% smaller and its index 50% smaller.
-- Heads are never deleted, so a size-proportional scale factor would let the
-- dead versions of a hot key wait behind the whole retained key space.
ALTER TABLE outbox_ordering_heads SET (
    fillfactor = 45,
    autovacuum_vacuum_scale_factor = 0.01,
    autovacuum_vacuum_threshold = 2000,
    autovacuum_analyze_scale_factor = 0.01,
    autovacuum_analyze_threshold = 2000
);

-- outbox_ordering_heads.last_sequence is the retained, monotonic authority for
-- an ordering key: Append advances it under a row lock in the same transaction
-- as the insert and rejects any sequence at or below it, and cleanup never
-- deletes a head. A unique index spanning published rows therefore rejects
-- nothing the head already accepts, while every ordered append pays to
-- maintain it and every retained published row keeps an entry in it. The
-- pending lookup that claim and successor resolution already need carries the
-- uniqueness that still has work to do: at most one unpublished event per
-- (key, sequence).
--
-- Build this index CONCURRENTLY out of band before migrating a populated
-- deployment; the in-transaction statement below holds a SHARE lock that blocks
-- appends while it builds, and becomes a no-op once the index exists.
CREATE UNIQUE INDEX IF NOT EXISTS outbox_events_ordering_pending_key
    ON outbox_events (ordering_key, ordering_sequence)
    WHERE published_at IS NULL AND ordering_key IS NOT NULL;

DROP INDEX outbox_events_ordering_head_idx;
DROP INDEX outbox_events_ordering_sequence_key;

-- +goose Down

-- Restoring the full-table unique index fails closed when a published row and
-- an unpublished row share an ordering key and sequence. Only a writer that
-- bypassed Append can create that state; resolve it deliberately rather than
-- letting a rollback discard an event.
CREATE UNIQUE INDEX outbox_events_ordering_sequence_key
    ON outbox_events (ordering_key, ordering_sequence)
    WHERE ordering_key IS NOT NULL;
CREATE INDEX outbox_events_ordering_head_idx
    ON outbox_events (ordering_key, ordering_sequence)
    WHERE published_at IS NULL AND ordering_key IS NOT NULL;

DROP INDEX outbox_events_ordering_pending_key;

ALTER TABLE outbox_ordering_heads RESET (
    fillfactor,
    autovacuum_vacuum_scale_factor,
    autovacuum_vacuum_threshold,
    autovacuum_analyze_scale_factor,
    autovacuum_analyze_threshold
);

ALTER TABLE outbox_events RESET (
    fillfactor,
    autovacuum_vacuum_scale_factor,
    autovacuum_vacuum_threshold,
    autovacuum_vacuum_insert_scale_factor,
    autovacuum_vacuum_insert_threshold,
    autovacuum_analyze_scale_factor,
    autovacuum_analyze_threshold
);
