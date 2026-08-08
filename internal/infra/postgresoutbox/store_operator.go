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
	Event                Event
	CreatedAt            time.Time
	AvailableAt          time.Time
	CycleAttemptCount    int
	TotalAttemptCount    int64
	LastAttemptAt        time.Time
	LeaseToken           string
	LeaseExpiresAt       time.Time
	PublishedAt          time.Time
	PoisonedAt           time.Time
	PublicationUncertain bool
	LastErrorClass       string
	RedriveCount         int
	LastRedriveID        string
	LastRedrivenAt       time.Time
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

const (
	actionRedrivePoison   = "redrive_poison"
	actionRedriveUnknown  = "redrive_unknown"
	actionConfirmAccepted = "confirm_accepted"
)

// Redrive releases one deterministic poison for another delivery cycle.
func (s *Store) Redrive(ctx context.Context, id, auditID string) error {
	return s.runOperatorAction(ctx, "redrive", actionRedrivePoison, id, auditID)
}

// RedriveUnknown releases an outcome-unknown event while preserving its sticky
// publication uncertainty.
func (s *Store) RedriveUnknown(ctx context.Context, id, auditID string) error {
	return s.runOperatorAction(ctx, "redrive_unknown", actionRedriveUnknown, id, auditID)
}

// ConfirmAccepted marks an outcome-unknown event as published without touching
// the broker. Ordered events advance their existing outbox head.
func (s *Store) ConfirmAccepted(ctx context.Context, id, auditID string) error {
	return s.runOperatorAction(ctx, "confirm_accepted", actionConfirmAccepted, id, auditID)
}

func (s *Store) runOperatorAction(ctx context.Context, operation, action, id, auditID string) (err error) {
	started := time.Now()
	defer func() { s.recordOperation(ctx, operation, started, err) }()
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
		return operatorActionInTx(ctx, sqlcgen.New(tx), action, id, auditID)
	})
	if err != nil {
		return fmt.Errorf("%s outbox event: %w", operation, err)
	}
	return nil
}

func operatorActionInTx(ctx context.Context, queries *sqlcgen.Queries, action, id, auditID string) error {
	replayed, err := operatorAuditSpent(ctx, queries, action, id, auditID)
	if err != nil || replayed {
		return err
	}
	row, err := queries.LockOutboxEventForAction(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		return fmt.Errorf("lock outbox event for operator action: %w", err)
	}
	replayed, err = operatorAuditSpent(ctx, queries, action, id, auditID)
	if err != nil || replayed {
		return err
	}

	cycle, hasCycle, err := operatorActionCycle(action, row)
	if err != nil {
		return err
	}
	var cycleNumber *int32
	if hasCycle {
		cycleNumber = &cycle
	}

	inserted, err := queries.InsertOutboxAction(ctx, sqlcgen.InsertOutboxActionParams{
		AuditID: auditID, EventID: id, ActionKind: action, CycleNumber: cycleNumber,
	})
	if err != nil {
		return fmt.Errorf("insert outbox operator audit: %w", err)
	}
	if inserted == 0 {
		replayed, err = operatorAuditSpent(ctx, queries, action, id, auditID)
		if err != nil || replayed {
			return err
		}
		return ErrOperatorAuditConflict
	}

	return applyOperatorAction(ctx, queries, action, id, auditID, row.OrderingKey != nil)
}

func operatorActionCycle(action string, row sqlcgen.OutboxEvent) (int32, bool, error) {
	switch action {
	case actionRedrivePoison:
		if !row.PoisonedAt.Valid || row.PublishedAt.Valid || row.PublicationUncertain == nil || *row.PublicationUncertain {
			return 0, false, ErrOperatorStateConflict
		}
	case actionRedriveUnknown, actionConfirmAccepted:
		if !outcomeUnknown(row) {
			return 0, false, ErrOperatorStateConflict
		}
	default:
		return 0, false, fmt.Errorf("%w: unknown operator action", ErrConfig)
	}
	if action == actionConfirmAccepted {
		return 0, false, nil
	}
	if row.RedriveCount == math.MaxInt32 {
		return 0, false, fmt.Errorf("%w: redrive count exhausted", ErrOperatorStateConflict)
	}
	return row.RedriveCount + 1, true, nil
}

func applyOperatorAction(
	ctx context.Context,
	queries *sqlcgen.Queries,
	action, id, auditID string,
	ordered bool,
) error {
	switch action {
	case actionRedrivePoison:
		rows, err := queries.RedrivePoisonedOutboxEvent(
			ctx, sqlcgen.RedrivePoisonedOutboxEventParams{AuditID: &auditID, ID: id},
		)
		return operatorRows("redrive poisoned outbox event", rows, err)
	case actionRedriveUnknown:
		rows, err := queries.RedriveUnknownOutboxEvent(
			ctx, sqlcgen.RedriveUnknownOutboxEventParams{AuditID: &auditID, ID: id},
		)
		return operatorRows("redrive unknown outbox event", rows, err)
	case actionConfirmAccepted:
		return confirmAccepted(ctx, queries, id, ordered)
	default:
		return fmt.Errorf("%w: unknown operator action", ErrConfig)
	}
}

func confirmAccepted(ctx context.Context, queries *sqlcgen.Queries, id string, ordered bool) error {
	if !ordered {
		rows, err := queries.ConfirmUnorderedOutboxAccepted(ctx, id)
		return operatorRows("confirm unordered outbox event", rows, err)
	}
	for range orderedPublishSnapshots {
		result, err := queries.ConfirmOrderedOutboxAccepted(ctx, id)
		if err != nil {
			return fmt.Errorf("confirm ordered outbox event: %w", err)
		}
		if result.Marked {
			return nil
		}
		if !result.SnapshotConflict {
			break
		}
	}
	return ErrOperatorStateConflict
}

func operatorAuditSpent(ctx context.Context, queries *sqlcgen.Queries, action, id, auditID string) (bool, error) {
	earlier, err := queries.FindOutboxAction(ctx, auditID)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return false, nil
	case err != nil:
		return false, fmt.Errorf("find outbox operator audit: %w", err)
	case earlier.EventID == id && earlier.ActionKind == action:
		return true, nil
	default:
		return false, fmt.Errorf("%w: audit id belongs to another event or action", ErrOperatorAuditConflict)
	}
}

func outcomeUnknown(row sqlcgen.OutboxEvent) bool {
	return !row.PublishedAt.Valid && row.PoisonedAt.Valid &&
		row.PublicationUncertain != nil && *row.PublicationUncertain &&
		row.LastErrorClass != nil && *row.LastErrorClass == classOutcomeUnknown
}

func operatorRows(operation string, rows int64, err error) error {
	if err != nil {
		return fmt.Errorf("%s: %w", operation, err)
	}
	if rows != 1 {
		return ErrOperatorStateConflict
	}
	return nil
}
