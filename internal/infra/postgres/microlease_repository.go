package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/Dankosik/billing-service/internal/infra/postgres/sqlcgen"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrMicroleaseRepository = errors.New("microlease repository")

type MicroleaseRepository struct {
	pool    *pgxpool.Pool
	queries *sqlcgen.Queries
}

type IssueMicroleaseCommand struct {
	MicroleaseID               string
	AccountID                  string
	AccountScopeKey            string
	ProxyAllocatorOwnerID      string
	MicroleaseGeneration       int64
	LeaseFence                 string
	IssuedCapUSDAtoms          int64
	PricingSnapshotID          string
	PricingSnapshotFingerprint string
	PricingPolicyVersion       string
	PricingDecisionAt          time.Time
	PricingSelectorKey         string
	PricingContractVersion     string
	FeePolicyVersion           string
	MicroleasePolicyVersion    string
	IssuedAt                   time.Time
	DebitCutoffAt              time.Time
	ExpiresAt                  time.Time
	IdempotencyRecordID        string
	IdempotencyKey             string
	RequestFingerprint         string
	StoredOutcomeID            string
	LedgerEntryID              string
	OutboxID                   string
	EventFingerprint           string
	SafeMetadata               map[string]string
}

type CloseMicroleaseCommand struct {
	MicroleaseID            string
	AccountID               string
	IdempotencyRecordID     string
	IdempotencyKey          string
	RequestFingerprint      string
	StoredOutcomeID         string
	LedgerEntryID           string
	OutboxID                string
	EventFingerprint        string
	ReleasedUSDAtoms        int64
	UnresolvedReservedAtoms int64
	CloseState              string
	ClosedAt                time.Time
	Now                     time.Time
	SafeMetadata            map[string]string
}

type TerminalSettlementCommand struct {
	InboxID                    string
	Topic                      string
	PartitionID                int32
	OffsetValue                int64
	EventID                    string
	ProducerIdentity           string
	EventFingerprint           string
	MicroleaseChildDebitID     string
	MicroleaseID               string
	DebitAuthorizationID       string
	AccountID                  string
	AccountScopeKey            string
	ProxyAllocatorOwnerID      string
	MicroleaseGeneration       int64
	ChildSequence              int64
	ChildCapUSDAtoms           int64
	ChargedUSDAtoms            int64
	ReleasedUSDAtoms           int64
	WriteOffUSDAtoms           int64
	RequestBasisFingerprint    string
	TerminalBasisFingerprint   string
	PricingSnapshotID          string
	PricingSnapshotFingerprint string
	TerminalKind               string
	TerminalState              string
	LedgerEntryID              string
	SettlementEffectID         string
	OutboxID                   string
	OutboxEventFingerprint     string
	TerminalAt                 time.Time
	SettledAt                  time.Time
	SafeMetadata               map[string]string
}

type CheckpointCommand struct {
	InboxID                    string
	Topic                      string
	PartitionID                int32
	OffsetValue                int64
	EventID                    string
	ProducerIdentity           string
	EventFingerprint           string
	CheckpointID               string
	MicroleaseID               string
	AccountID                  string
	AccountScopeKey            string
	ProxyAllocatorOwnerID      string
	MicroleaseGeneration       int64
	CheckpointSequence         int64
	CheckpointKind             string
	AllocatedChildHighWater    int64
	AllocatedChildCount        int64
	AllocatedChildCapUSDAtoms  int64
	TerminalSubmittedCount     int64
	TerminalPublishedCount     int64
	TerminalAcceptedCount      int64
	UnresolvedChildCount       int64
	UnresolvedChildCapUSDAtoms int64
	LocalRemainingUSDAtoms     int64
	CheckpointFingerprint      string
	CreatedAt                  time.Time
	AppliedAt                  time.Time
	SafeMetadata               map[string]string
}

type AdmissionControlCommand struct {
	AdmissionControlID          string
	ScopeKind                   string
	ScopeKey                    string
	UseClass                    string
	State                       string
	ReasonCode                  string
	TerminalLagBucket           string
	StaleAgeBucket              string
	ReconciliationBacklogBucket string
	AuditedActorKind            string
	AuditedActorID              string
	ExpiresAt                   time.Time
	RenewedAt                   time.Time
	CreatedAt                   time.Time
	SafeMetadata                map[string]string
}

type QuarantineCommand struct {
	InboxID          string
	Topic            string
	PartitionID      int32
	OffsetValue      int64
	EventID          string
	ProducerIdentity string
	BusinessIdentity string
	EventFingerprint string
	ReasonClass      string
	QuarantinedAt    time.Time
	SafeMetadata     map[string]string
}

type MicroleaseRecord struct {
	MicroleaseID            string
	AccountID               string
	AccountScopeKey         string
	State                   string
	IssuedCapUSDAtoms       int64
	AvailableChildUSDAtoms  int64
	TerminalChargedUSDAtoms int64
	TerminalReleaseUSDAtoms int64
	WriteOffUSDAtoms        int64
	ExpiresAt               time.Time
}

func NewMicroleaseRepository(pool *Pool) (*MicroleaseRepository, error) {
	if pool == nil || pool.pool == nil {
		return nil, fmt.Errorf("%w: postgres pool is required", ErrMicroleaseRepository)
	}
	return NewMicroleaseRepositoryFromPGXPool(pool.pool)
}

func NewMicroleaseRepositoryFromPGXPool(pool *pgxpool.Pool) (*MicroleaseRepository, error) {
	if pool == nil {
		return nil, fmt.Errorf("%w: pgx pool is required", ErrMicroleaseRepository)
	}
	return &MicroleaseRepository{
		pool:    pool,
		queries: sqlcgen.New(pool),
	}, nil
}

func (r *MicroleaseRepository) Issue(ctx context.Context, cmd IssueMicroleaseCommand) (MicroleaseRecord, error) {
	if err := r.require(); err != nil {
		return MicroleaseRecord{}, err
	}
	if cmd.IssuedCapUSDAtoms <= 0 {
		return MicroleaseRecord{}, fmt.Errorf("%w: issued cap must be positive", ErrMicroleaseRepository)
	}
	now := nonZeroTime(cmd.IssuedAt)
	safeMetadata, err := marshalSafeObject(cmd.SafeMetadata)
	if err != nil {
		return MicroleaseRecord{}, err
	}

	microleaseID, accountID, idempotencyID, outcomeID, ledgerID, outboxID, err := issueCommandUUIDs(cmd)
	if err != nil {
		return MicroleaseRecord{}, err
	}

	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return MicroleaseRecord{}, fmt.Errorf("%w: begin issue transaction: %w", ErrMicroleaseRepository, err)
	}
	defer rollbackUnlessCommitted(ctx, tx)
	q := r.queries.WithTx(tx)

	balance, err := q.LockAccountBalanceByAccountID(ctx, accountID)
	if err != nil {
		return MicroleaseRecord{}, fmt.Errorf("%w: lock account balance: %w", ErrMicroleaseRepository, err)
	}
	if balance.AvailableUsdAtoms < cmd.IssuedCapUSDAtoms {
		return MicroleaseRecord{}, fmt.Errorf("%w: insufficient available balance", ErrMicroleaseRepository)
	}

	if _, err = q.CreateIdempotencyRecord(ctx, sqlcgen.CreateIdempotencyRecordParams{
		IdempotencyRecordID: idempotencyID,
		AccountID:           accountID,
		OperationKind:       "microlease_issue",
		IdempotencyKey:      cmd.IdempotencyKey,
		RequestFingerprint:  cmd.RequestFingerprint,
		RetentionClass:      "hot_replay",
		FirstSeenAt:         timestamptzValue(now),
	}); err != nil {
		return MicroleaseRecord{}, fmt.Errorf("%w: create issue idempotency: %w", ErrMicroleaseRepository, err)
	}

	settledAfter := balance.SettledUsdAtoms
	reservedAfter := balance.ReservedUsdAtoms + cmd.IssuedCapUSDAtoms
	availableAfter := settledAfter - reservedAfter
	ledger, err := q.CreateLedgerEntry(ctx, sqlcgen.CreateLedgerEntryParams{
		LedgerEntryID:          ledgerID,
		AccountID:              accountID,
		AccountScopeKey:        cmd.AccountScopeKey,
		EffectType:             "microlease_reserve",
		AmountUsdAtoms:         cmd.IssuedCapUSDAtoms,
		SettledDeltaUsdAtoms:   0,
		ReservedDeltaUsdAtoms:  cmd.IssuedCapUSDAtoms,
		PendingDeltaUsdAtoms:   0,
		SettledAfterUsdAtoms:   settledAfter,
		ReservedAfterUsdAtoms:  reservedAfter,
		AvailableAfterUsdAtoms: availableAfter,
		PendingAfterUsdAtoms:   balance.PendingUsdAtoms,
		BalanceVersionAfter:    balance.Version + 1,
		IdempotencyRecordID:    idempotencyID,
		EffectiveAt:            timestamptzValue(now),
		CreatedAt:              timestamptzValue(now),
		CreatedByKind:          "service",
		ReasonCode:             "microlease_issue",
		SafeMetadata:           safeMetadata,
	})
	if err != nil {
		return MicroleaseRecord{}, fmt.Errorf("%w: create issue ledger entry: %w", ErrMicroleaseRepository, err)
	}
	if _, err = q.UpdateAccountBalanceAfterLedger(ctx, sqlcgen.UpdateAccountBalanceAfterLedgerParams{
		AccountID:         accountID,
		SettledUsdAtoms:   settledAfter,
		ReservedUsdAtoms:  reservedAfter,
		AvailableUsdAtoms: availableAfter,
		PendingUsdAtoms:   balance.PendingUsdAtoms,
		LastLedgerEntryID: ledger.LedgerEntryID,
		UpdatedAt:         timestamptzValue(now),
	}); err != nil {
		return MicroleaseRecord{}, fmt.Errorf("%w: update balance after issue: %w", ErrMicroleaseRepository, err)
	}

	if _, err = q.CreateOperationOutcome(ctx, sqlcgen.CreateOperationOutcomeParams{
		StoredOutcomeID:     outcomeID,
		IdempotencyRecordID: idempotencyID,
		AccountID:           accountID,
		OperationKind:       "microlease_issue",
		OutcomeStatus:       "success",
		PrimaryResourceType: "spending_microlease",
		PrimaryResourceID:   cmd.MicroleaseID,
		LedgerEntryID:       ledgerID,
		SafeOutcome:         safeMetadata,
		CreatedAt:           timestamptzValue(now),
	}); err != nil {
		return MicroleaseRecord{}, fmt.Errorf("%w: create issue outcome: %w", ErrMicroleaseRepository, err)
	}
	if _, err = q.MarkIdempotencyCommitted(ctx, sqlcgen.MarkIdempotencyCommittedParams{
		IdempotencyRecordID: idempotencyID,
		State:               "committed",
		StoredOutcomeID:     outcomeID,
		LastSeenAt:          timestamptzValue(now),
	}); err != nil {
		return MicroleaseRecord{}, fmt.Errorf("%w: commit issue idempotency: %w", ErrMicroleaseRepository, err)
	}

	row, err := q.CreateSpendingMicrolease(ctx, sqlcgen.CreateSpendingMicroleaseParams{
		MicroleaseID:               microleaseID,
		AccountID:                  accountID,
		AccountScopeKey:            cmd.AccountScopeKey,
		ProxyAllocatorOwnerID:      cmd.ProxyAllocatorOwnerID,
		MicroleaseGeneration:       cmd.MicroleaseGeneration,
		LeaseFence:                 cmd.LeaseFence,
		State:                      "active",
		IssuedCapUsdAtoms:          cmd.IssuedCapUSDAtoms,
		AvailableChildCapUsdAtoms:  cmd.IssuedCapUSDAtoms,
		PricingSnapshotID:          cmd.PricingSnapshotID,
		PricingSnapshotFingerprint: cmd.PricingSnapshotFingerprint,
		PricingPolicyVersion:       cmd.PricingPolicyVersion,
		PricingDecisionAt:          timestamptzValue(cmd.PricingDecisionAt),
		PricingSelectorKey:         cmd.PricingSelectorKey,
		PricingContractVersion:     cmd.PricingContractVersion,
		FeePolicyVersion:           cmd.FeePolicyVersion,
		MicroleasePolicyVersion:    cmd.MicroleasePolicyVersion,
		IssuedAt:                   timestamptzValue(cmd.IssuedAt),
		DebitCutoffAt:              timestamptzValue(cmd.DebitCutoffAt),
		ExpiresAt:                  timestamptzValue(cmd.ExpiresAt),
		IdempotencyRecordID:        idempotencyID,
		StoredOutcomeID:            outcomeID,
		SafeMetadata:               safeMetadata,
		CreatedAt:                  timestamptzValue(now),
	})
	if err != nil {
		return MicroleaseRecord{}, fmt.Errorf("%w: create microlease: %w", ErrMicroleaseRepository, err)
	}

	if _, err = q.CreateBillingOutbox(ctx, sqlcgen.CreateBillingOutboxParams{
		OutboxID:         outboxID,
		EventType:        "MicroleaseIssued",
		AggregateType:    "spending_microlease",
		AggregateID:      cmd.MicroleaseID,
		EventFingerprint: cmd.EventFingerprint,
		SafePayload:      safePayload(map[string]any{"microlease_id": cmd.MicroleaseID, "result": "issued"}),
		NextAttemptAt:    timestamptzValue(now),
		CreatedAt:        timestamptzValue(now),
	}); err != nil {
		return MicroleaseRecord{}, fmt.Errorf("%w: create issue outbox: %w", ErrMicroleaseRepository, err)
	}

	if err = tx.Commit(ctx); err != nil {
		return MicroleaseRecord{}, fmt.Errorf("%w: commit issue transaction: %w", ErrMicroleaseRepository, err)
	}
	return mapMicroleaseRecord(row), nil
}

func (r *MicroleaseRepository) ReadMicrolease(ctx context.Context, microleaseID string) (MicroleaseRecord, error) {
	if err := r.require(); err != nil {
		return MicroleaseRecord{}, err
	}
	id, err := uuidValue(microleaseID)
	if err != nil {
		return MicroleaseRecord{}, err
	}
	row, err := r.queries.GetSpendingMicrolease(ctx, id)
	if err != nil {
		return MicroleaseRecord{}, fmt.Errorf("%w: read microlease: %w", ErrMicroleaseRepository, err)
	}
	return mapMicroleaseRecord(row), nil
}

func (r *MicroleaseRepository) Close(ctx context.Context, cmd CloseMicroleaseCommand) (MicroleaseRecord, error) {
	if err := r.require(); err != nil {
		return MicroleaseRecord{}, err
	}
	now := nonZeroTime(cmd.Now)
	microleaseID, accountID, idempotencyID, outcomeID, ledgerID, outboxID, err := closeCommandUUIDs(cmd)
	if err != nil {
		return MicroleaseRecord{}, err
	}
	safeMetadata, err := marshalSafeObject(cmd.SafeMetadata)
	if err != nil {
		return MicroleaseRecord{}, err
	}

	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return MicroleaseRecord{}, fmt.Errorf("%w: begin close transaction: %w", ErrMicroleaseRepository, err)
	}
	defer rollbackUnlessCommitted(ctx, tx)
	q := r.queries.WithTx(tx)

	lease, err := q.LockSpendingMicrolease(ctx, microleaseID)
	if err != nil {
		return MicroleaseRecord{}, fmt.Errorf("%w: lock microlease: %w", ErrMicroleaseRepository, err)
	}
	balance, err := q.LockAccountBalanceByAccountID(ctx, accountID)
	if err != nil {
		return MicroleaseRecord{}, fmt.Errorf("%w: lock balance for close: %w", ErrMicroleaseRepository, err)
	}
	if _, err = q.CreateIdempotencyRecord(ctx, sqlcgen.CreateIdempotencyRecordParams{
		IdempotencyRecordID: idempotencyID,
		AccountID:           accountID,
		OperationKind:       "microlease_close",
		IdempotencyKey:      cmd.IdempotencyKey,
		RequestFingerprint:  cmd.RequestFingerprint,
		RetentionClass:      "hot_replay",
		FirstSeenAt:         timestamptzValue(now),
	}); err != nil {
		return MicroleaseRecord{}, fmt.Errorf("%w: create close idempotency: %w", ErrMicroleaseRepository, err)
	}

	var ledgerRef pgtype.UUID
	if cmd.ReleasedUSDAtoms > 0 {
		settledAfter := balance.SettledUsdAtoms
		reservedAfter := balance.ReservedUsdAtoms - cmd.ReleasedUSDAtoms
		availableAfter := settledAfter - reservedAfter
		ledger, err := q.CreateLedgerEntry(ctx, sqlcgen.CreateLedgerEntryParams{
			LedgerEntryID:          ledgerID,
			AccountID:              accountID,
			AccountScopeKey:        lease.AccountScopeKey,
			EffectType:             "microlease_close_release",
			AmountUsdAtoms:         -cmd.ReleasedUSDAtoms,
			SettledDeltaUsdAtoms:   0,
			ReservedDeltaUsdAtoms:  -cmd.ReleasedUSDAtoms,
			PendingDeltaUsdAtoms:   0,
			SettledAfterUsdAtoms:   settledAfter,
			ReservedAfterUsdAtoms:  reservedAfter,
			AvailableAfterUsdAtoms: availableAfter,
			PendingAfterUsdAtoms:   balance.PendingUsdAtoms,
			BalanceVersionAfter:    balance.Version + 1,
			IdempotencyRecordID:    idempotencyID,
			EffectiveAt:            timestamptzValue(now),
			CreatedAt:              timestamptzValue(now),
			CreatedByKind:          "worker",
			ReasonCode:             "microlease_close_release",
			SafeMetadata:           safeMetadata,
		})
		if err != nil {
			return MicroleaseRecord{}, fmt.Errorf("%w: create close ledger entry: %w", ErrMicroleaseRepository, err)
		}
		ledgerRef = ledger.LedgerEntryID
		if _, err = q.UpdateAccountBalanceAfterLedger(ctx, sqlcgen.UpdateAccountBalanceAfterLedgerParams{
			AccountID:         accountID,
			SettledUsdAtoms:   settledAfter,
			ReservedUsdAtoms:  reservedAfter,
			AvailableUsdAtoms: availableAfter,
			PendingUsdAtoms:   balance.PendingUsdAtoms,
			LastLedgerEntryID: ledgerRef,
			UpdatedAt:         timestamptzValue(now),
		}); err != nil {
			return MicroleaseRecord{}, fmt.Errorf("%w: update balance after close: %w", ErrMicroleaseRepository, err)
		}
	} else {
		ledgerRef = sqlcgen.OperationOutcome{}.LedgerEntryID
	}

	if _, err = q.CreateOperationOutcome(ctx, sqlcgen.CreateOperationOutcomeParams{
		StoredOutcomeID:     outcomeID,
		IdempotencyRecordID: idempotencyID,
		AccountID:           accountID,
		OperationKind:       "microlease_close",
		OutcomeStatus:       "success",
		PrimaryResourceType: "spending_microlease",
		PrimaryResourceID:   cmd.MicroleaseID,
		LedgerEntryID:       ledgerRef,
		SafeOutcome:         safeMetadata,
		CreatedAt:           timestamptzValue(now),
	}); err != nil {
		return MicroleaseRecord{}, fmt.Errorf("%w: create close outcome: %w", ErrMicroleaseRepository, err)
	}
	if _, err = q.MarkIdempotencyCommitted(ctx, sqlcgen.MarkIdempotencyCommittedParams{
		IdempotencyRecordID: idempotencyID,
		State:               "committed",
		StoredOutcomeID:     outcomeID,
		LastSeenAt:          timestamptzValue(now),
	}); err != nil {
		return MicroleaseRecord{}, fmt.Errorf("%w: commit close idempotency: %w", ErrMicroleaseRepository, err)
	}
	state := cmd.CloseState
	if state == "" {
		state = "closed"
	}
	row, err := q.UpdateSpendingMicroleaseSettlementTotals(ctx, sqlcgen.UpdateSpendingMicroleaseSettlementTotalsParams{
		MicroleaseID:                      microleaseID,
		State:                             state,
		AvailableChildCapUsdAtoms:         0,
		AllocatedChildCapReportedUsdAtoms: lease.AllocatedChildCapReportedUsdAtoms,
		TerminalChargedUsdAtoms:           lease.TerminalChargedUsdAtoms,
		TerminalReleasedUsdAtoms:          lease.TerminalReleasedUsdAtoms + cmd.ReleasedUSDAtoms,
		WriteOffUsdAtoms:                  lease.WriteOffUsdAtoms,
		LastCheckpointSequence:            lease.LastCheckpointSequence,
		LastCheckpointFingerprint:         lease.LastCheckpointFingerprint,
		ClosedAt:                          timestamptzValue(nonZeroTime(cmd.ClosedAt)),
		UpdatedAt:                         timestamptzValue(now),
	})
	if err != nil {
		return MicroleaseRecord{}, fmt.Errorf("%w: update close totals: %w", ErrMicroleaseRepository, err)
	}
	if _, err = q.CreateBillingOutbox(ctx, sqlcgen.CreateBillingOutboxParams{
		OutboxID:         outboxID,
		EventType:        "MicroleaseClosed",
		AggregateType:    "spending_microlease",
		AggregateID:      cmd.MicroleaseID,
		EventFingerprint: cmd.EventFingerprint,
		SafePayload:      safePayload(map[string]any{"microlease_id": cmd.MicroleaseID, "result": state}),
		NextAttemptAt:    timestamptzValue(now),
		CreatedAt:        timestamptzValue(now),
	}); err != nil {
		return MicroleaseRecord{}, fmt.Errorf("%w: create close outbox: %w", ErrMicroleaseRepository, err)
	}

	if err = tx.Commit(ctx); err != nil {
		return MicroleaseRecord{}, fmt.Errorf("%w: commit close transaction: %w", ErrMicroleaseRepository, err)
	}
	return mapMicroleaseRecord(row), nil
}

func (r *MicroleaseRepository) ApplyTerminalSettlement(ctx context.Context, cmd TerminalSettlementCommand) error {
	if err := r.require(); err != nil {
		return err
	}
	now := nonZeroTime(cmd.SettledAt)
	ids, err := terminalCommandUUIDs(cmd)
	if err != nil {
		return err
	}
	safeMetadata, err := marshalSafeObject(cmd.SafeMetadata)
	if err != nil {
		return err
	}

	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("%w: begin terminal transaction: %w", ErrMicroleaseRepository, err)
	}
	defer rollbackUnlessCommitted(ctx, tx)
	q := r.queries.WithTx(tx)

	if _, err = q.CreateBillingEventInboxReceipt(ctx, sqlcgen.CreateBillingEventInboxReceiptParams{
		InboxID:               ids.inboxID,
		Topic:                 cmd.Topic,
		PartitionID:           cmd.PartitionID,
		OffsetValue:           cmd.OffsetValue,
		EventID:               cmd.EventID,
		ProducerIdentity:      cmd.ProducerIdentity,
		BusinessIdentityType:  "microlease_child_debit",
		BusinessIdentityValue: cmd.DebitAuthorizationID,
		EventFingerprint:      cmd.EventFingerprint,
		SafeMetadata:          safeMetadata,
		ReceivedAt:            timestamptzValue(now),
	}); err != nil {
		return fmt.Errorf("%w: create terminal inbox: %w", ErrMicroleaseRepository, err)
	}
	lease, err := q.LockSpendingMicrolease(ctx, ids.microleaseID)
	if err != nil {
		return fmt.Errorf("%w: lock terminal microlease: %w", ErrMicroleaseRepository, err)
	}
	balance, err := q.LockAccountBalanceByAccountID(ctx, ids.accountID)
	if err != nil {
		return fmt.Errorf("%w: lock terminal balance: %w", ErrMicroleaseRepository, err)
	}
	ledgerParams := terminalLedgerParams(cmd, balance, ids, safeMetadata, now)
	ledger, err := q.CreateLedgerEntry(ctx, ledgerParams)
	if err != nil {
		return fmt.Errorf("%w: create terminal ledger: %w", ErrMicroleaseRepository, err)
	}
	settledAfter := ledger.SettledAfterUsdAtoms
	reservedAfter := ledger.ReservedAfterUsdAtoms
	if _, err = q.UpdateAccountBalanceAfterLedger(ctx, sqlcgen.UpdateAccountBalanceAfterLedgerParams{
		AccountID:         ids.accountID,
		SettledUsdAtoms:   settledAfter,
		ReservedUsdAtoms:  reservedAfter,
		AvailableUsdAtoms: settledAfter - reservedAfter,
		PendingUsdAtoms:   balance.PendingUsdAtoms,
		LastLedgerEntryID: ledger.LedgerEntryID,
		UpdatedAt:         timestamptzValue(now),
	}); err != nil {
		return fmt.Errorf("%w: update terminal balance: %w", ErrMicroleaseRepository, err)
	}
	if _, err = q.CreateMicroleaseChildDebit(ctx, sqlcgen.CreateMicroleaseChildDebitParams{
		MicroleaseChildDebitID:     ids.childID,
		MicroleaseID:               ids.microleaseID,
		DebitAuthorizationID:       cmd.DebitAuthorizationID,
		AccountID:                  ids.accountID,
		AccountScopeKey:            cmd.AccountScopeKey,
		ProxyAllocatorOwnerID:      cmd.ProxyAllocatorOwnerID,
		MicroleaseGeneration:       cmd.MicroleaseGeneration,
		ChildSequence:              cmd.ChildSequence,
		ChildCapUsdAtoms:           cmd.ChildCapUSDAtoms,
		ChargedUsdAtoms:            cmd.ChargedUSDAtoms,
		ReleasedUsdAtoms:           cmd.ReleasedUSDAtoms,
		WriteOffUsdAtoms:           cmd.WriteOffUSDAtoms,
		RequestBasisFingerprint:    cmd.RequestBasisFingerprint,
		TerminalBasisFingerprint:   &cmd.TerminalBasisFingerprint,
		PricingSnapshotID:          cmd.PricingSnapshotID,
		PricingSnapshotFingerprint: cmd.PricingSnapshotFingerprint,
		TerminalKind:               cmd.TerminalKind,
		State:                      cmd.TerminalState,
		TerminalEventID:            &cmd.EventID,
		TerminalInboxID:            ids.inboxID,
		LedgerEntryID:              ledger.LedgerEntryID,
		SettlementEffectID:         ids.settlementEffectID,
		SafeMetadata:               safeMetadata,
		CreatedAt:                  timestamptzValue(now),
		TerminalAt:                 timestamptzValue(cmd.TerminalAt),
		SettledAt:                  timestamptzValue(now),
	}); err != nil {
		return fmt.Errorf("%w: create terminal child projection: %w", ErrMicroleaseRepository, err)
	}
	if _, err = q.UpdateSpendingMicroleaseSettlementTotals(ctx, sqlcgen.UpdateSpendingMicroleaseSettlementTotalsParams{
		MicroleaseID:                      ids.microleaseID,
		State:                             lease.State,
		AvailableChildCapUsdAtoms:         maxInt64(0, lease.AvailableChildCapUsdAtoms-cmd.ChildCapUSDAtoms),
		AllocatedChildCapReportedUsdAtoms: lease.AllocatedChildCapReportedUsdAtoms + cmd.ChildCapUSDAtoms,
		TerminalChargedUsdAtoms:           lease.TerminalChargedUsdAtoms + cmd.ChargedUSDAtoms,
		TerminalReleasedUsdAtoms:          lease.TerminalReleasedUsdAtoms + cmd.ReleasedUSDAtoms,
		WriteOffUsdAtoms:                  lease.WriteOffUsdAtoms + cmd.WriteOffUSDAtoms,
		LastCheckpointSequence:            lease.LastCheckpointSequence,
		LastCheckpointFingerprint:         lease.LastCheckpointFingerprint,
		ClosedAt:                          lease.ClosedAt,
		UpdatedAt:                         timestamptzValue(now),
	}); err != nil {
		return fmt.Errorf("%w: update terminal totals: %w", ErrMicroleaseRepository, err)
	}
	if _, err = q.MarkBillingEventInboxApplied(ctx, sqlcgen.MarkBillingEventInboxAppliedParams{
		InboxID:   ids.inboxID,
		AppliedAt: timestamptzValue(now),
	}); err != nil {
		return fmt.Errorf("%w: mark terminal inbox applied: %w", ErrMicroleaseRepository, err)
	}
	if _, err = q.CreateBillingOutbox(ctx, sqlcgen.CreateBillingOutboxParams{
		OutboxID:         ids.outboxID,
		EventType:        "MicroleaseTerminalApplied",
		AggregateType:    "microlease_child_debit",
		AggregateID:      cmd.DebitAuthorizationID,
		EventFingerprint: cmd.OutboxEventFingerprint,
		SafePayload:      safePayload(map[string]any{"microlease_id": cmd.MicroleaseID, "debit_authorization_id": cmd.DebitAuthorizationID, "result": cmd.TerminalState}),
		NextAttemptAt:    timestamptzValue(now),
		CreatedAt:        timestamptzValue(now),
	}); err != nil {
		return fmt.Errorf("%w: create terminal outbox: %w", ErrMicroleaseRepository, err)
	}
	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("%w: commit terminal transaction: %w", ErrMicroleaseRepository, err)
	}
	return nil
}

func (r *MicroleaseRepository) RecordCheckpoint(ctx context.Context, cmd CheckpointCommand) error {
	if err := r.require(); err != nil {
		return err
	}
	now := nonZeroTime(cmd.CreatedAt)
	ids, err := checkpointCommandUUIDs(cmd)
	if err != nil {
		return err
	}
	safeMetadata, err := marshalSafeObject(cmd.SafeMetadata)
	if err != nil {
		return err
	}

	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("%w: begin checkpoint transaction: %w", ErrMicroleaseRepository, err)
	}
	defer rollbackUnlessCommitted(ctx, tx)
	q := r.queries.WithTx(tx)

	if _, err = q.CreateBillingEventInboxReceipt(ctx, sqlcgen.CreateBillingEventInboxReceiptParams{
		InboxID:               ids.inboxID,
		Topic:                 cmd.Topic,
		PartitionID:           cmd.PartitionID,
		OffsetValue:           cmd.OffsetValue,
		EventID:               cmd.EventID,
		ProducerIdentity:      cmd.ProducerIdentity,
		BusinessIdentityType:  "microlease_checkpoint",
		BusinessIdentityValue: cmd.MicroleaseID,
		EventFingerprint:      cmd.EventFingerprint,
		SafeMetadata:          safeMetadata,
		ReceivedAt:            timestamptzValue(now),
	}); err != nil {
		return fmt.Errorf("%w: create checkpoint inbox: %w", ErrMicroleaseRepository, err)
	}
	if _, err = q.LockSpendingMicrolease(ctx, ids.microleaseID); err != nil {
		return fmt.Errorf("%w: lock checkpoint microlease: %w", ErrMicroleaseRepository, err)
	}
	if _, err = q.CreateMicroleaseCheckpoint(ctx, sqlcgen.CreateMicroleaseCheckpointParams{
		CheckpointID:                  ids.checkpointID,
		MicroleaseID:                  ids.microleaseID,
		AccountID:                     ids.accountID,
		AccountScopeKey:               cmd.AccountScopeKey,
		ProxyAllocatorOwnerID:         cmd.ProxyAllocatorOwnerID,
		MicroleaseGeneration:          cmd.MicroleaseGeneration,
		CheckpointSequence:            cmd.CheckpointSequence,
		CheckpointKind:                cmd.CheckpointKind,
		AllocatedChildHighWater:       cmd.AllocatedChildHighWater,
		AllocatedChildCount:           cmd.AllocatedChildCount,
		AllocatedChildCapSumUsdAtoms:  cmd.AllocatedChildCapUSDAtoms,
		TerminalSubmittedCount:        cmd.TerminalSubmittedCount,
		TerminalPublishedCount:        cmd.TerminalPublishedCount,
		TerminalAcceptedCount:         cmd.TerminalAcceptedCount,
		UnresolvedChildCount:          cmd.UnresolvedChildCount,
		UnresolvedChildCapSumUsdAtoms: cmd.UnresolvedChildCapUSDAtoms,
		LocalRemainingUsdAtoms:        cmd.LocalRemainingUSDAtoms,
		CheckpointFingerprint:         cmd.CheckpointFingerprint,
		InboxID:                       ids.inboxID,
		CreatedAt:                     timestamptzValue(now),
		AppliedAt:                     timestamptzValue(nonZeroTime(cmd.AppliedAt)),
	}); err != nil {
		return fmt.Errorf("%w: create checkpoint: %w", ErrMicroleaseRepository, err)
	}
	if _, err = q.MarkBillingEventInboxApplied(ctx, sqlcgen.MarkBillingEventInboxAppliedParams{
		InboxID:   ids.inboxID,
		AppliedAt: timestamptzValue(now),
	}); err != nil {
		return fmt.Errorf("%w: mark checkpoint inbox applied: %w", ErrMicroleaseRepository, err)
	}
	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("%w: commit checkpoint transaction: %w", ErrMicroleaseRepository, err)
	}
	return nil
}

func (r *MicroleaseRepository) UpsertAdmissionControl(ctx context.Context, cmd AdmissionControlCommand) error {
	if err := r.require(); err != nil {
		return err
	}
	id, err := uuidValue(cmd.AdmissionControlID)
	if err != nil {
		return err
	}
	safeMetadata, err := marshalSafeObject(cmd.SafeMetadata)
	if err != nil {
		return err
	}
	_, err = r.queries.UpsertBillingAdmissionControl(ctx, sqlcgen.UpsertBillingAdmissionControlParams{
		AdmissionControlID:          id,
		ScopeKind:                   cmd.ScopeKind,
		ScopeKey:                    cmd.ScopeKey,
		UseClass:                    cmd.UseClass,
		State:                       cmd.State,
		ReasonCode:                  cmd.ReasonCode,
		TerminalLagBucket:           cmd.TerminalLagBucket,
		StaleAgeBucket:              cmd.StaleAgeBucket,
		ReconciliationBacklogBucket: cmd.ReconciliationBacklogBucket,
		AuditedActorKind:            cmd.AuditedActorKind,
		AuditedActorID:              cmd.AuditedActorID,
		SafeMetadata:                safeMetadata,
		ExpiresAt:                   timestamptzValue(cmd.ExpiresAt),
		RenewedAt:                   timestamptzValue(cmd.RenewedAt),
		CreatedAt:                   timestamptzValue(cmd.CreatedAt),
	})
	if err != nil {
		return fmt.Errorf("%w: upsert admission control: %w", ErrMicroleaseRepository, err)
	}
	return nil
}

func (r *MicroleaseRepository) RecordQuarantine(ctx context.Context, cmd QuarantineCommand) error {
	if err := r.require(); err != nil {
		return err
	}
	inboxID, err := uuidValue(cmd.InboxID)
	if err != nil {
		return err
	}
	safeMetadata, err := marshalSafeObject(cmd.SafeMetadata)
	if err != nil {
		return err
	}
	now := nonZeroTime(cmd.QuarantinedAt)
	identity := cmd.BusinessIdentity
	if identity == "" {
		identity = fmt.Sprintf("%s:%d:%d", cmd.Topic, cmd.PartitionID, cmd.OffsetValue)
	}
	eventID := cmd.EventID
	if eventID == "" {
		eventID = identity
	}
	producer := cmd.ProducerIdentity
	if producer == "" {
		producer = "unknown"
	}
	fingerprint := cmd.EventFingerprint
	if fingerprint == "" {
		fingerprint = "unknown"
	}
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("%w: begin quarantine transaction: %w", ErrMicroleaseRepository, err)
	}
	defer rollbackUnlessCommitted(ctx, tx)
	q := r.queries.WithTx(tx)
	receipt, err := q.CreateBillingEventInboxReceipt(ctx, sqlcgen.CreateBillingEventInboxReceiptParams{
		InboxID:               inboxID,
		Topic:                 cmd.Topic,
		PartitionID:           cmd.PartitionID,
		OffsetValue:           cmd.OffsetValue,
		EventID:               eventID,
		ProducerIdentity:      producer,
		BusinessIdentityType:  "redpanda_event",
		BusinessIdentityValue: identity,
		EventFingerprint:      fingerprint,
		SafeMetadata:          safeMetadata,
		ReceivedAt:            timestamptzValue(now),
	})
	if err != nil {
		return fmt.Errorf("%w: create quarantine inbox: %w", ErrMicroleaseRepository, err)
	}
	if _, err = q.MarkBillingEventInboxConflict(ctx, sqlcgen.MarkBillingEventInboxConflictParams{
		InboxID:      receipt.InboxID,
		FailureClass: optionalString(cmd.ReasonClass),
		SafeMetadata: safeMetadata,
		UpdatedAt:    timestamptzValue(now),
	}); err != nil {
		return fmt.Errorf("%w: mark quarantine inbox: %w", ErrMicroleaseRepository, err)
	}
	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("%w: commit quarantine transaction: %w", ErrMicroleaseRepository, err)
	}
	return nil
}

func (r *MicroleaseRepository) RetryEligibleInbox(ctx context.Context, now time.Time, limit int32) (int64, error) {
	if err := r.require(); err != nil {
		return 0, err
	}
	if limit <= 0 {
		limit = 100
	}
	tag, err := r.pool.Exec(ctx, `
		UPDATE billing_event_inbox
		SET state = 'received',
		    failure_class = NULL,
		    updated_at = $1
		WHERE inbox_id IN (
		    SELECT candidate.inbox_id
		    FROM billing_event_inbox AS candidate
		    WHERE candidate.state IN ('conflict', 'quarantined', 'reconcile_required')
		    ORDER BY candidate.updated_at, candidate.inbox_id
		    FOR UPDATE SKIP LOCKED
		    LIMIT $2
		)
	`, timestamptzValue(now), limit)
	if err != nil {
		return 0, fmt.Errorf("%w: retry eligible inbox: %w", ErrMicroleaseRepository, err)
	}
	return tag.RowsAffected(), nil
}

func (r *MicroleaseRepository) ScanStaleMicroleases(ctx context.Context, now time.Time, limit int32) ([]MicroleaseRecord, error) {
	if err := r.require(); err != nil {
		return nil, err
	}
	if limit <= 0 {
		limit = 100
	}
	rows, err := r.queries.ListStaleSpendingMicroleases(ctx, sqlcgen.ListStaleSpendingMicroleasesParams{
		ExpiresAt: timestamptzValue(now),
		Limit:     limit,
	})
	if err != nil {
		return nil, fmt.Errorf("%w: scan stale microleases: %w", ErrMicroleaseRepository, err)
	}
	records := make([]MicroleaseRecord, 0, len(rows))
	for _, row := range rows {
		records = append(records, mapMicroleaseRecord(row))
	}
	return records, nil
}

func (r *MicroleaseRepository) ClaimOutbox(ctx context.Context, now time.Time, limit int32) ([]sqlcgen.BillingOutbox, error) {
	if err := r.require(); err != nil {
		return nil, err
	}
	rows, err := r.queries.ClaimBillingOutbox(ctx, sqlcgen.ClaimBillingOutboxParams{
		NextAttemptAt: timestamptzValue(now),
		UpdatedAt:     timestamptzValue(now),
		Limit:         limit,
	})
	if err != nil {
		return nil, fmt.Errorf("%w: claim outbox: %w", ErrMicroleaseRepository, err)
	}
	return rows, nil
}

func (r *MicroleaseRepository) MarkOutboxPublished(ctx context.Context, outboxID string, publishedAt time.Time) error {
	if err := r.require(); err != nil {
		return err
	}
	id, err := uuidValue(outboxID)
	if err != nil {
		return err
	}
	if _, err = r.queries.MarkBillingOutboxPublished(ctx, sqlcgen.MarkBillingOutboxPublishedParams{
		OutboxID:    id,
		PublishedAt: timestamptzValue(publishedAt),
	}); err != nil {
		return fmt.Errorf("%w: mark outbox published: %w", ErrMicroleaseRepository, err)
	}
	return nil
}

func (r *MicroleaseRepository) MarkOutboxRetry(ctx context.Context, outboxID string, nextAttemptAt time.Time, _ string) error {
	if err := r.require(); err != nil {
		return err
	}
	id, err := uuidValue(outboxID)
	if err != nil {
		return err
	}
	if _, err = r.queries.MarkBillingOutboxFailed(ctx, sqlcgen.MarkBillingOutboxFailedParams{
		OutboxID:      id,
		NextAttemptAt: timestamptzValue(nextAttemptAt),
		UpdatedAt:     timestamptzValue(time.Now().UTC()),
	}); err != nil {
		return fmt.Errorf("%w: mark outbox retry: %w", ErrMicroleaseRepository, err)
	}
	return nil
}

func (r *MicroleaseRepository) require() error {
	if r == nil || r.pool == nil || r.queries == nil {
		return fmt.Errorf("%w: repository is not configured", ErrMicroleaseRepository)
	}
	return nil
}

func closeCommandUUIDs(cmd CloseMicroleaseCommand) (microleaseID, accountID, idempotencyID, outcomeID, ledgerID, outboxID pgtype.UUID, err error) {
	return parseSixUUIDs(cmd.MicroleaseID, cmd.AccountID, cmd.IdempotencyRecordID, cmd.StoredOutcomeID, cmd.LedgerEntryID, cmd.OutboxID)
}

func issueCommandUUIDs(cmd IssueMicroleaseCommand) (microleaseID, accountID, idempotencyID, outcomeID, ledgerID, outboxID pgtype.UUID, err error) {
	return parseSixUUIDs(cmd.MicroleaseID, cmd.AccountID, cmd.IdempotencyRecordID, cmd.StoredOutcomeID, cmd.LedgerEntryID, cmd.OutboxID)
}

type terminalCommandIDs struct {
	inboxID            pgtype.UUID
	microleaseID       pgtype.UUID
	childID            pgtype.UUID
	accountID          pgtype.UUID
	ledgerID           pgtype.UUID
	settlementEffectID pgtype.UUID
	outboxID           pgtype.UUID
}

func terminalCommandUUIDs(cmd TerminalSettlementCommand) (terminalCommandIDs, error) {
	inboxID, err := uuidValue(cmd.InboxID)
	if err != nil {
		return terminalCommandIDs{}, err
	}
	microleaseID, err := uuidValue(cmd.MicroleaseID)
	if err != nil {
		return terminalCommandIDs{}, err
	}
	childID, err := uuidValue(cmd.MicroleaseChildDebitID)
	if err != nil {
		return terminalCommandIDs{}, err
	}
	accountID, err := uuidValue(cmd.AccountID)
	if err != nil {
		return terminalCommandIDs{}, err
	}
	ledgerID, err := uuidValue(cmd.LedgerEntryID)
	if err != nil {
		return terminalCommandIDs{}, err
	}
	settlementEffectID, err := optionalUUID(cmd.SettlementEffectID)
	if err != nil {
		return terminalCommandIDs{}, err
	}
	outboxID, err := uuidValue(cmd.OutboxID)
	if err != nil {
		return terminalCommandIDs{}, err
	}
	return terminalCommandIDs{
		inboxID:            inboxID,
		microleaseID:       microleaseID,
		childID:            childID,
		accountID:          accountID,
		ledgerID:           ledgerID,
		settlementEffectID: settlementEffectID,
		outboxID:           outboxID,
	}, nil
}

type checkpointCommandIDs struct {
	inboxID      pgtype.UUID
	checkpointID pgtype.UUID
	microleaseID pgtype.UUID
	accountID    pgtype.UUID
}

func checkpointCommandUUIDs(cmd CheckpointCommand) (checkpointCommandIDs, error) {
	inboxID, err := uuidValue(cmd.InboxID)
	if err != nil {
		return checkpointCommandIDs{}, err
	}
	checkpointID, err := uuidValue(cmd.CheckpointID)
	if err != nil {
		return checkpointCommandIDs{}, err
	}
	microleaseID, err := uuidValue(cmd.MicroleaseID)
	if err != nil {
		return checkpointCommandIDs{}, err
	}
	accountID, err := uuidValue(cmd.AccountID)
	if err != nil {
		return checkpointCommandIDs{}, err
	}
	return checkpointCommandIDs{
		inboxID:      inboxID,
		checkpointID: checkpointID,
		microleaseID: microleaseID,
		accountID:    accountID,
	}, nil
}

func parseSixUUIDs(a, b, c, d, e, f string) (pgtype.UUID, pgtype.UUID, pgtype.UUID, pgtype.UUID, pgtype.UUID, pgtype.UUID, error) {
	first, err := uuidValue(a)
	if err != nil {
		return pgtype.UUID{}, pgtype.UUID{}, pgtype.UUID{}, pgtype.UUID{}, pgtype.UUID{}, pgtype.UUID{}, err
	}
	second, err := uuidValue(b)
	if err != nil {
		return pgtype.UUID{}, pgtype.UUID{}, pgtype.UUID{}, pgtype.UUID{}, pgtype.UUID{}, pgtype.UUID{}, err
	}
	third, err := uuidValue(c)
	if err != nil {
		return pgtype.UUID{}, pgtype.UUID{}, pgtype.UUID{}, pgtype.UUID{}, pgtype.UUID{}, pgtype.UUID{}, err
	}
	fourth, err := uuidValue(d)
	if err != nil {
		return pgtype.UUID{}, pgtype.UUID{}, pgtype.UUID{}, pgtype.UUID{}, pgtype.UUID{}, pgtype.UUID{}, err
	}
	fifth, err := uuidValue(e)
	if err != nil {
		return pgtype.UUID{}, pgtype.UUID{}, pgtype.UUID{}, pgtype.UUID{}, pgtype.UUID{}, pgtype.UUID{}, err
	}
	sixth, err := uuidValue(f)
	if err != nil {
		return pgtype.UUID{}, pgtype.UUID{}, pgtype.UUID{}, pgtype.UUID{}, pgtype.UUID{}, pgtype.UUID{}, err
	}
	return first, second, third, fourth, fifth, sixth, nil
}

func terminalLedgerParams(cmd TerminalSettlementCommand, balance sqlcgen.AccountBalance, ids terminalCommandIDs, safeMetadata []byte, now time.Time) sqlcgen.CreateLedgerEntryParams {
	reservedDelta := -(cmd.ChargedUSDAtoms + cmd.ReleasedUSDAtoms + cmd.WriteOffUSDAtoms)
	settledDelta := -cmd.ChargedUSDAtoms
	effect := "microlease_child_charge"
	amount := settledDelta
	if cmd.ChargedUSDAtoms == 0 {
		switch {
		case cmd.WriteOffUSDAtoms > 0:
			effect = "microlease_write_off"
			amount = -cmd.WriteOffUSDAtoms
			settledDelta = 0
			reservedDelta = -cmd.WriteOffUSDAtoms
		default:
			effect = "microlease_child_release"
			amount = -cmd.ReleasedUSDAtoms
			settledDelta = 0
			reservedDelta = -cmd.ReleasedUSDAtoms
		}
	}
	settledAfter := balance.SettledUsdAtoms + settledDelta
	reservedAfter := balance.ReservedUsdAtoms + reservedDelta
	return sqlcgen.CreateLedgerEntryParams{
		LedgerEntryID:          ids.ledgerID,
		AccountID:              ids.accountID,
		AccountScopeKey:        cmd.AccountScopeKey,
		EffectType:             effect,
		AmountUsdAtoms:         amount,
		SettledDeltaUsdAtoms:   settledDelta,
		ReservedDeltaUsdAtoms:  reservedDelta,
		PendingDeltaUsdAtoms:   0,
		SettledAfterUsdAtoms:   settledAfter,
		ReservedAfterUsdAtoms:  reservedAfter,
		AvailableAfterUsdAtoms: settledAfter - reservedAfter,
		PendingAfterUsdAtoms:   balance.PendingUsdAtoms,
		BalanceVersionAfter:    balance.Version + 1,
		SettlementEffectID:     ids.settlementEffectID,
		EffectiveAt:            timestamptzValue(now),
		CreatedAt:              timestamptzValue(now),
		CreatedByKind:          "worker",
		ReasonCode:             "microlease_terminal",
		SafeMetadata:           safeMetadata,
	}
}

func rollbackUnlessCommitted(ctx context.Context, tx pgx.Tx) {
	_ = tx.Rollback(ctx)
}

func marshalSafeObject(metadata map[string]string) ([]byte, error) {
	if metadata == nil {
		return []byte(`{}`), nil
	}
	payload, err := json.Marshal(metadata)
	if err != nil {
		return nil, fmt.Errorf("%w: marshal safe metadata: %w", ErrMicroleaseRepository, err)
	}
	return payload, nil
}

func safePayload(payload map[string]any) []byte {
	data, err := json.Marshal(payload)
	if err != nil {
		return []byte(`{}`)
	}
	return data
}

func mapMicroleaseRecord(row sqlcgen.SpendingMicrolease) MicroleaseRecord {
	return MicroleaseRecord{
		MicroleaseID:            uuidString(row.MicroleaseID),
		AccountID:               uuidString(row.AccountID),
		AccountScopeKey:         row.AccountScopeKey,
		State:                   row.State,
		IssuedCapUSDAtoms:       row.IssuedCapUsdAtoms,
		AvailableChildUSDAtoms:  row.AvailableChildCapUsdAtoms,
		TerminalChargedUSDAtoms: row.TerminalChargedUsdAtoms,
		TerminalReleaseUSDAtoms: row.TerminalReleasedUsdAtoms,
		WriteOffUSDAtoms:        row.WriteOffUsdAtoms,
		ExpiresAt:               row.ExpiresAt.Time,
	}
}

func nonZeroTime(t time.Time) time.Time {
	if t.IsZero() {
		return time.Now().UTC()
	}
	return t
}

func maxInt64(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}
