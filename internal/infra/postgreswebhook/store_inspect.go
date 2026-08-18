package postgreswebhook

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/example/go-service-template-rest/internal/infra/postgres/sqlcgen"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type InspectionCursor struct {
	CycleFrom       int64
	AttemptCycle    int64
	AttemptedAt     time.Time
	AttemptID       string
	ActionCreatedAt time.Time
	ActionID        string
}

type InspectionRequest struct {
	OwnerScope string
	DeliveryID string
	PageSize   int
	Cursor     InspectionCursor
}

type DeliveryInspection struct {
	AcceptanceID           string
	BusinessEventID        string
	FanoutSnapshotID       string
	EventType              string
	BusinessSchemaVersion  string
	ContentType            string
	DeliveryID             string
	DestinationID          string
	DestinationGeneration  int64
	State                  DeliveryState
	CurrentCycle           int64
	CumulativeSummary      OutcomeClass
	Sendable               bool
	NextDueAt              time.Time
	RedriveEligibleUntil   time.Time
	TerminalAt             time.Time
	CreatedAt              time.Time
	UpdatedAt              time.Time
	LegalHold              bool
	DestinationDisposition string
	ControlRevision        int64
	KeyStateRevision       int64
	RequiredSecretRevision int64
	Cycles                 []InspectionCycle
	Attempts               []InspectionAttempt
	Actions                []InspectionAction
	Next                   InspectionCursor
	MoreCycles             bool
	MoreAttempts           bool
	MoreActions            bool
}

type InspectionCycle struct {
	CycleNumber         int64
	Kind                string
	AuthorizingActionID string
	AcceptedAt          time.Time
	DeadlineAt          time.Time
	MaximumAttempts     int
	AttemptsUsed        int
	Disposition         string
	FinalizedAt         time.Time
}

type InspectionAttempt struct {
	CycleNumber         int64
	AttemptID           string
	Fence               int64
	AttemptedAt         time.Time
	LeaseExpiresAt      time.Time
	SendAuthorized      bool
	MayHaveSent         bool
	ResponseHeaderBytes *int
	ResponseBodyBytes   *int
	ResponseStatus      *int
	RetryAfter          time.Duration
	RetryDelay          time.Duration
	Outcome             OutcomeClass
	FinalizedAt         time.Time
}

type InspectionAction struct {
	ActionID                  string
	ActorReference            string
	Kind                      string
	TargetGeneration          int64
	ExpectedState             string
	Reason                    string
	DuplicateRiskAcknowledged bool
	State                     string
	Result                    string
	CreatedAt                 time.Time
	CompletedAt               time.Time
	ResultCycle               int64
}

func (s *Store) InspectDelivery(ctx context.Context, request InspectionRequest) (DeliveryInspection, error) {
	if !s.valid() || validateToken("owner_scope", request.OwnerScope) != nil || validateToken("delivery_id", request.DeliveryID) != nil || request.PageSize < 1 || request.PageSize > MaxClaimScanPage || request.Cursor.CycleFrom < 0 || request.Cursor.AttemptCycle < 0 {
		return DeliveryInspection{}, fmt.Errorf("%w: inspection request is invalid", ErrConfig)
	}
	opCtx, cancel := context.WithTimeout(ctx, s.options.OperationTimeout)
	defer cancel()
	tx, err := s.pool.BeginTx(opCtx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return DeliveryInspection{}, fmt.Errorf("begin webhook inspection: %w", err)
	}
	defer func() { _ = tx.Rollback(context.WithoutCancel(opCtx)) }()
	inspection, err := inspectDelivery(opCtx, sqlcgen.New(tx), request)
	if err != nil {
		return DeliveryInspection{}, err
	}
	if err := tx.Commit(opCtx); err != nil {
		return DeliveryInspection{}, fmt.Errorf("commit webhook inspection: %w", err)
	}
	return inspection, nil
}

func inspectDelivery(ctx context.Context, queries *sqlcgen.Queries, request InspectionRequest) (DeliveryInspection, error) {
	row, err := queries.ReadWebhookDeliveryInspection(ctx, sqlcgen.ReadWebhookDeliveryInspectionParams{OwnerScope: request.OwnerScope, DeliveryID: request.DeliveryID})
	if errors.Is(err, pgx.ErrNoRows) {
		return DeliveryInspection{}, ErrNotFound
	}
	if err != nil {
		return DeliveryInspection{}, fmt.Errorf("read webhook delivery inspection: %w", err)
	}
	pageSize, err := int32Value(request.PageSize + 1)
	if err != nil {
		return DeliveryInspection{}, err
	}
	from := time.Unix(0, 0).UTC()
	if !request.Cursor.AttemptedAt.IsZero() {
		from = request.Cursor.AttemptedAt
	}
	actionFrom := time.Unix(0, 0).UTC()
	if !request.Cursor.ActionCreatedAt.IsZero() {
		actionFrom = request.Cursor.ActionCreatedAt
	}
	cycles, err := queries.ListWebhookInspectionCycles(ctx, sqlcgen.ListWebhookInspectionCyclesParams{OwnerScope: request.OwnerScope, DeliveryID: request.DeliveryID, CycleFrom: request.Cursor.CycleFrom, PageSize: pageSize})
	if err != nil {
		return DeliveryInspection{}, fmt.Errorf("list webhook inspection cycles: %w", err)
	}
	attempts, err := queries.ListWebhookInspectionAttempts(ctx, sqlcgen.ListWebhookInspectionAttemptsParams{OwnerScope: request.OwnerScope, DeliveryID: request.DeliveryID, AfterCycle: request.Cursor.AttemptCycle, AfterAttemptedAt: pgtime(from), AfterAttemptID: request.Cursor.AttemptID, PageSize: pageSize})
	if err != nil {
		return DeliveryInspection{}, fmt.Errorf("list webhook inspection attempts: %w", err)
	}
	actions, err := queries.ListWebhookInspectionActions(ctx, sqlcgen.ListWebhookInspectionActionsParams{OwnerScope: request.OwnerScope, DeliveryID: request.DeliveryID, AfterCreatedAt: pgtime(actionFrom), AfterActionID: request.Cursor.ActionID, PageSize: pageSize})
	if err != nil {
		return DeliveryInspection{}, fmt.Errorf("list webhook inspection actions: %w", err)
	}
	result := inspectionSummary(row)
	result.Next = request.Cursor
	result.MoreCycles = len(cycles) > request.PageSize
	result.MoreAttempts = len(attempts) > request.PageSize
	result.MoreActions = len(actions) > request.PageSize
	cycles = cycles[:min(len(cycles), request.PageSize)]
	attempts = attempts[:min(len(attempts), request.PageSize)]
	actions = actions[:min(len(actions), request.PageSize)]
	for _, cycle := range cycles {
		item := InspectionCycle{CycleNumber: cycle.CycleNumber, Kind: cycle.CycleKind, AcceptedAt: inspectedTime(cycle.AcceptedAt), DeadlineAt: inspectedTime(cycle.DeadlineAt), MaximumAttempts: int(cycle.MaximumAttempts), AttemptsUsed: int(cycle.AttemptsUsed), Disposition: cycle.Disposition, FinalizedAt: inspectedTime(cycle.FinalizedAt)}
		if cycle.AuthorizingActionID != nil {
			item.AuthorizingActionID = *cycle.AuthorizingActionID
		}
		result.Cycles = append(result.Cycles, item)
		result.Next.CycleFrom = cycle.CycleNumber + 1
	}
	for _, attempt := range attempts {
		item := InspectionAttempt{CycleNumber: attempt.CycleNumber, AttemptID: attempt.AttemptID, Fence: attempt.Fence, AttemptedAt: inspectedTime(attempt.AttemptedAt), LeaseExpiresAt: inspectedTime(attempt.LeaseExpiresAt), SendAuthorized: attempt.SendAuthorized, MayHaveSent: attempt.MayHaveSent, RetryAfter: nullableDuration(attempt.RetryAfterDelayNs), RetryDelay: nullableDuration(attempt.RetryDelayNs), FinalizedAt: inspectedTime(attempt.FinalizedAt)}
		item.ResponseHeaderBytes = nullableInt(attempt.ResponseHeaderBytes)
		item.ResponseBodyBytes = nullableInt(attempt.ResponseBodyBytes)
		item.ResponseStatus = nullableInt(attempt.ResponseStatus)
		if attempt.OutcomeClass != nil {
			item.Outcome = OutcomeClass(*attempt.OutcomeClass)
		}
		result.Attempts = append(result.Attempts, item)
		result.Next.AttemptCycle, result.Next.AttemptedAt, result.Next.AttemptID = attempt.CycleNumber, item.AttemptedAt, attempt.AttemptID
	}
	for _, action := range actions {
		result.Actions = append(result.Actions, InspectionAction{ActionID: action.ActionID, ActorReference: action.ActorReference, Kind: action.ActionKind, TargetGeneration: action.TargetGeneration, ExpectedState: action.ExpectedState, Reason: action.Reason, DuplicateRiskAcknowledged: action.DuplicateRiskAcknowledged, State: action.State, Result: action.Result, CreatedAt: inspectedTime(action.CreatedAt), CompletedAt: inspectedTime(action.CompletedAt), ResultCycle: action.ResultCycle})
		result.Next.ActionCreatedAt, result.Next.ActionID = inspectedTime(action.CreatedAt), action.ActionID
	}
	return result, nil
}

func inspectionSummary(row sqlcgen.ReadWebhookDeliveryInspectionRow) DeliveryInspection {
	return DeliveryInspection{AcceptanceID: row.AcceptanceID, BusinessEventID: row.BusinessEventID, FanoutSnapshotID: row.FanoutSnapshotID, EventType: row.EventType, BusinessSchemaVersion: row.BusinessSchemaVersion, ContentType: row.ContentType, DeliveryID: row.DeliveryID, DestinationID: row.DestinationID, DestinationGeneration: row.DestinationGeneration, State: DeliveryState(row.State), CurrentCycle: row.CurrentCycle, CumulativeSummary: OutcomeClass(row.CumulativeSummary), Sendable: row.Sendable, NextDueAt: inspectedTime(row.NextDueAt), RedriveEligibleUntil: inspectedTime(row.RedriveEligibleUntil), TerminalAt: inspectedTime(row.TerminalAt), CreatedAt: inspectedTime(row.CreatedAt), UpdatedAt: inspectedTime(row.UpdatedAt), LegalHold: row.LegalHold, DestinationDisposition: row.DestinationDisposition, ControlRevision: row.ControlRevision, KeyStateRevision: row.KeyStateRevision, RequiredSecretRevision: row.RequiredSecretRevision}
}

func inspectedTime(value pgtype.Timestamptz) time.Time {
	if !value.Valid {
		return time.Time{}
	}
	return value.Time.UTC()
}

func nullableDuration(value *int64) time.Duration {
	if value == nil {
		return 0
	}
	return time.Duration(*value)
}

func nullableInt(value *int32) *int {
	if value == nil {
		return nil
	}
	converted := int(*value)
	return &converted
}
