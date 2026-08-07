package postgresoutbox

import (
	"context"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/example/go-service-template-rest/internal/infra/postgres/sqlcgen"
	"github.com/jackc/pgx/v5"
)

// Record is one stored event with its whole delivery state. It is the operator
// view of a row, and Relay.reconcilePublished reads two of its fields —
// PublishedAt and LeaseToken — to resolve a finalization a batch statement did
// not report, so those two are relay behavior as well.
type Record struct {
	Event             Event
	CreatedAt         time.Time
	AvailableAt       time.Time
	CycleAttemptCount int
	TotalAttemptCount int64
	LastAttemptAt     time.Time
	LeaseToken        string
	LeaseExpiresAt    time.Time
	PublishedAt       time.Time
	PoisonedAt        time.Time
	LastErrorClass    string
	RedriveCount      int
	LastRedriveID     string
	LastRedrivenAt    time.Time
}

func (s *Store) Get(ctx context.Context, id string) (Record, error) {
	if !s.valid() {
		return Record{}, errStoreRequired()
	}
	if err := validateText(ErrConfig, "id", id, maxTextBytes); err != nil {
		return Record{}, err
	}
	row, err := s.queries.GetOutboxEvent(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Record{}, ErrNotFound
		}
		return Record{}, fmt.Errorf("get outbox event: %w", err)
	}
	return recordFromRow(row), nil
}

// Redrive releases one poisoned event for another delivery cycle, recording
// auditID so a repeated call for the same audit id is idempotent.
//
// Unlike Append it owns its transaction, so its locking and audit-conflict
// rules cannot be proven against a stubbed driver; TestPostgresOutboxRedrive in
// test/postgres_outbox_integration_test.go is where its contract is exercised.
func (s *Store) Redrive(ctx context.Context, id, auditID string) (err error) {
	started := time.Now()
	defer func() { s.recordOperation(ctx, "redrive", started, err) }()
	if !s.valid() {
		return errStoreRequired()
	}
	if err := validateText(ErrConfig, "id", id, maxTextBytes); err != nil {
		return err
	}
	if err := validateText(ErrConfig, "audit_id", auditID, maxTextBytes); err != nil {
		return err
	}

	err = s.pool.InTx(ctx, pgx.TxOptions{}, func(tx pgx.Tx) error {
		return redriveInTx(ctx, sqlcgen.New(tx), id, auditID)
	})
	if err != nil {
		return fmt.Errorf("redrive outbox event: %w", err)
	}
	return nil
}

// redriveInTx is the whole redrive under one transaction: take the row's lock
// first so a concurrent redrive of the same event serializes behind it, settle
// the audit id, then release the row for another cycle.
func redriveInTx(ctx context.Context, queries *sqlcgen.Queries, id, auditID string) error {
	row, err := queries.LockOutboxEventForRedrive(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		return fmt.Errorf("lock outbox event for redrive: %w", err)
	}

	replayed, err := auditIDAlreadySpent(ctx, queries, id, auditID)
	if err != nil || replayed {
		return err
	}

	if !row.PoisonedAt.Valid || row.PublishedAt.Valid {
		return ErrRedriveRejected
	}
	if row.RedriveCount == math.MaxInt32 {
		return fmt.Errorf("%w: redrive count exhausted", ErrRedriveRejected)
	}
	if err := queries.InsertOutboxRedrive(ctx, sqlcgen.InsertOutboxRedriveParams{
		AuditID:     auditID,
		EventID:     id,
		CycleNumber: row.RedriveCount + 1,
	}); err != nil {
		return fmt.Errorf("insert outbox redrive: %w", err)
	}
	rows, err := queries.RedriveOutboxEvent(ctx, sqlcgen.RedriveOutboxEventParams{AuditID: &auditID, ID: id})
	if err != nil {
		return fmt.Errorf("redrive outbox event: %w", err)
	}
	if rows != 1 {
		return ErrRedriveRejected
	}
	return nil
}

// auditIDAlreadySpent is what makes a repeated call idempotent. The audit id is
// the caller's own key for one redrive decision, so the three outcomes are
// distinct: this event already spent it, which is the retry the caller meant
// and does nothing; another event spent it, which is a caller fault rather than
// a race; or it is unspent and this redrive may claim it.
func auditIDAlreadySpent(ctx context.Context, queries *sqlcgen.Queries, id, auditID string) (bool, error) {
	earlierEventID, err := queries.FindOutboxRedrive(ctx, auditID)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return false, nil
	case err != nil:
		return false, fmt.Errorf("find outbox redrive: %w", err)
	case earlierEventID == id:
		return true, nil
	default:
		return false, fmt.Errorf("%w: audit id belongs to another event", ErrRedriveConflict)
	}
}
