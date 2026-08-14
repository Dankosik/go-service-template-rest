package postgresidempotency

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

const maintenanceProbeName = "postgres_http_idempotency"

type maintenanceSnapshot struct {
	observedAt       time.Time
	writer           bool
	terminal         error
	rows             int64
	relationBytes    int64
	resultBytes      int64
	oldestExpiryUnix int64
}

func (s *Store) Name() string { return maintenanceProbeName }

// Check reads the last maintenance observation only. Readiness never consumes a
// pool connection or runs cleanup.
func (s *Store) Check(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("check HTTP idempotency readiness: %w", err)
	}
	if !s.valid() {
		return fmt.Errorf("%w: store is required", ErrUnavailable)
	}
	return s.snapshotError(s.safety.Load())
}

func (s *Store) snapshotError(snapshot *maintenanceSnapshot) error {
	if snapshot == nil || snapshot.observedAt.IsZero() {
		return fmt.Errorf("%w: maintenance has not observed writer safety", ErrUnavailable)
	}
	if snapshot.terminal != nil {
		return snapshot.terminal
	}
	if !snapshot.writer {
		return fmt.Errorf("%w: maintenance did not observe a writer", ErrUnavailable)
	}
	if time.Since(snapshot.observedAt) > s.options.MaxMaintenanceLag {
		return fmt.Errorf("%w: maintenance observation is stale", ErrUnavailable)
	}
	if snapshot.relationBytes >= s.options.MaxRelationBytes-s.options.AdmissionHeadroomBytes {
		return fmt.Errorf("%w: maintenance relation safety reserve is exhausted", ErrUnavailable)
	}
	return nil
}

func (s *Store) allowsFirstExecution() bool {
	return s != nil && s.snapshotError(s.safety.Load()) == nil
}

// Maintain performs one bounded writer cycle. The caller owns cadence and retry.
func (s *Store) Maintain(ctx context.Context) error {
	if !s.valid() {
		return fmt.Errorf("%w: store is required", ErrConfig)
	}
	if !s.maintenanceMu.TryLock() {
		return fmt.Errorf("%w: maintenance cycle already running", ErrUnavailable)
	}
	defer s.maintenanceMu.Unlock()
	if snapshot := s.safety.Load(); snapshot != nil && snapshot.terminal != nil {
		return snapshot.terminal
	}

	snapshot, err := s.maintain(ctx)
	if err != nil {
		return s.finishMaintenanceError(ctx, snapshot, err)
	}
	if terminal := s.publishSnapshot(snapshot); terminal != nil {
		return terminal
	}
	if s.maintenanceFailed {
		s.maintenanceFailed = false
		s.telemetry.recordMaintenance(ctx, transitionCleanupRecovered, nil)
	}
	return nil
}

func (s *Store) finishMaintenanceError(ctx context.Context, snapshot *maintenanceSnapshot, err error) error {
	if snapshot != nil {
		if terminal := s.publishSnapshot(snapshot); terminal != nil {
			s.telemetry.recordMaintenance(ctx, "", terminal)
			return terminal
		}
	}
	if errors.Is(err, ErrEpochLost) || errors.Is(err, ErrIntegrityConflict) {
		s.markTerminal(err)
	}
	cleanupFailed := snapshot == nil && errors.Is(err, ErrUnavailable)
	if cleanupFailed {
		s.maintenanceFailed = true
	}
	event := ""
	if cleanupFailed {
		event = transitionCleanupFailed
	}
	s.telemetry.recordMaintenance(ctx, event, err)
	return err
}

func (s *Store) maintain(ctx context.Context) (*maintenanceSnapshot, error) {
	conn, err := s.pool.Acquire(ctx)
	if err != nil {
		return nil, unavailable(ctx, "acquire maintenance connection")
	}
	defer conn.Release()
	tx, err := conn.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, unavailable(ctx, "begin maintenance")
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var writer, commitTimestamps bool
	if err := tx.QueryRow(ctx, `
		SELECT
			NOT pg_is_in_recovery() AND current_setting('transaction_read_only') = 'off',
			current_setting('track_commit_timestamp') = 'on'`).Scan(&writer, &commitTimestamps); err != nil {
		return nil, unavailable(ctx, "check maintenance authority")
	}
	if !writer {
		snapshot := &maintenanceSnapshot{observedAt: time.Now(), writer: false}
		return snapshot, fmt.Errorf("%w: maintenance requires the writer", ErrUnavailable)
	}
	if !commitTimestamps {
		return nil, fmt.Errorf("%w: track_commit_timestamp is disabled", ErrEpochLost)
	}

	remaining := s.options.CleanupBatchSize
	selected, materialized, err := materializeMaintenanceEpochs(ctx, tx, remaining)
	if err != nil {
		return nil, unavailable(ctx, "materialize maintenance epochs")
	}
	if selected != materialized {
		return nil, fmt.Errorf("%w: an exact commit epoch is unavailable", ErrEpochLost)
	}
	remaining -= selected
	if remaining > 0 {
		expired, err := expireMaintenanceResults(ctx, tx, remaining)
		if err != nil {
			return nil, unavailable(ctx, "expire maintenance results")
		}
		remaining -= expired
	}
	if remaining > 0 {
		if _, err := deleteMaintenanceGuards(ctx, tx, remaining); err != nil {
			return nil, unavailable(ctx, "delete maintenance guards")
		}
	}

	snapshot, err := observeMaintenance(ctx, tx)
	if err != nil {
		return nil, unavailable(ctx, "observe maintenance safety")
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, unavailable(ctx, "commit maintenance")
	}
	snapshot.observedAt = time.Now()
	return snapshot, nil
}

func materializeMaintenanceEpochs(ctx context.Context, tx pgx.Tx, limit int) (selected, updated int, err error) {
	err = tx.QueryRow(ctx, `
		WITH candidates AS (
			SELECT ctid, xmin
			FROM postgres_http_idempotency
			WHERE phase = 'completed' AND committed_at IS NULL
			ORDER BY identity_token
			FOR UPDATE SKIP LOCKED
			LIMIT $1
		), updated AS (
			UPDATE postgres_http_idempotency AS target
			SET committed_at = pg_xact_commit_timestamp(candidates.xmin)
			FROM candidates
			WHERE target.ctid = candidates.ctid
			  AND pg_xact_commit_timestamp(candidates.xmin) IS NOT NULL
			RETURNING 1
		)
		SELECT (SELECT count(*) FROM candidates), count(*) FROM updated`, limit).Scan(&selected, &updated)
	if err != nil {
		return 0, 0, fmt.Errorf("materialize maintenance epochs: %w", err)
	}
	return selected, updated, nil
}

func expireMaintenanceResults(ctx context.Context, tx pgx.Tx, limit int) (int, error) {
	command, err := tx.Exec(ctx, `
		WITH candidates AS (
			SELECT ctid
			FROM postgres_http_idempotency
			WHERE phase = 'completed'
			  AND committed_at IS NOT NULL
			  AND octet_length(result) > 0
			  AND committed_at + ((replay_nanos - 1) / 1000 + 1) * interval '1 microsecond' <= clock_timestamp()
			ORDER BY identity_token
			FOR UPDATE SKIP LOCKED
			LIMIT $1
		)
			UPDATE postgres_http_idempotency AS target
			SET result = ''::bytea
			FROM candidates
			WHERE target.ctid = candidates.ctid
		`, limit)
	if err != nil {
		return 0, fmt.Errorf("expire maintenance results: %w", err)
	}
	return int(command.RowsAffected()), nil
}

func deleteMaintenanceGuards(ctx context.Context, tx pgx.Tx, limit int) (int, error) {
	command, err := tx.Exec(ctx, `
		WITH candidates AS (
			SELECT ctid
			FROM postgres_http_idempotency
			WHERE phase = 'completed'
			  AND committed_at IS NOT NULL
			  AND duplicate_risk_permanent = false
			  AND committed_at + ((duplicate_risk_nanos - 1) / 1000 + 1) * interval '1 microsecond' <= clock_timestamp()
			ORDER BY identity_token
			FOR UPDATE SKIP LOCKED
			LIMIT $1
		)
			DELETE FROM postgres_http_idempotency AS target
			USING candidates
			WHERE target.ctid = candidates.ctid
		`, limit)
	if err != nil {
		return 0, fmt.Errorf("delete maintenance guards: %w", err)
	}
	return int(command.RowsAffected()), nil
}

func observeMaintenance(ctx context.Context, tx pgx.Tx) (*maintenanceSnapshot, error) {
	snapshot := &maintenanceSnapshot{writer: true}
	var oldestExpiry *time.Time
	err := tx.QueryRow(ctx, `
		SELECT
			count(*),
			coalesce(sum(octet_length(result)), 0),
			pg_total_relation_size('postgres_http_idempotency'::regclass),
			min(committed_at + ((replay_nanos - 1) / 1000 + 1) * interval '1 microsecond')
		FROM postgres_http_idempotency`).Scan(
		&snapshot.rows,
		&snapshot.resultBytes,
		&snapshot.relationBytes,
		&oldestExpiry,
	)
	if oldestExpiry != nil {
		snapshot.oldestExpiryUnix = oldestExpiry.UTC().Unix()
	}
	if err != nil {
		return nil, fmt.Errorf("observe maintenance safety: %w", err)
	}
	return snapshot, nil
}

func (s *Store) publishSnapshot(snapshot *maintenanceSnapshot) error {
	for {
		previous := s.safety.Load()
		if previous != nil && previous.terminal != nil {
			return previous.terminal
		}
		if s.safety.CompareAndSwap(previous, snapshot) {
			return nil
		}
	}
}

func (s *Store) markTerminal(err error) {
	for {
		previous := s.safety.Load()
		if previous != nil && previous.terminal != nil {
			return
		}
		next := maintenanceSnapshot{terminal: err}
		if previous != nil {
			next = *previous
			next.terminal = err
		}
		if s.safety.CompareAndSwap(previous, &next) {
			if s.terminalErrors != nil {
				select {
				case s.terminalErrors <- err:
				default:
				}
			}
			return
		}
	}
}
