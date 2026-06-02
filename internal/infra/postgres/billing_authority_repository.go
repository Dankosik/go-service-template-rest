package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Dankosik/billing-service/internal/app/billingauthority"
	"github.com/Dankosik/billing-service/internal/infra/postgres/sqlcgen"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrBillingAuthorityRepository = errors.New("billing authority repository")

type BillingAuthorityRepository struct {
	pool    *pgxpool.Pool
	queries *sqlcgen.Queries
}

type TerminalChildDebitRecord struct {
	MicroleaseChildDebitID       string
	MicroleaseID                 string
	DebitAuthorizationID         string
	UsageOperationID             string
	AccountID                    string
	AccountScopeKey              string
	ProxyAllocatorOwnerID        string
	MicroleaseGeneration         int64
	ChildSequence                int64
	ChildCapUSDAtoms             int64
	RequestBasisFingerprint      string
	PricingSnapshotID            string
	PricingSnapshotFingerprint   string
	QualifiedInferenceEvidenceID string
}

func NewBillingAuthorityRepository(pool *Pool) (*BillingAuthorityRepository, error) {
	if pool == nil || pool.pool == nil {
		return nil, fmt.Errorf("%w: postgres pool is required", ErrBillingAuthorityRepository)
	}
	return NewBillingAuthorityRepositoryFromPGXPool(pool.pool)
}

func NewBillingAuthorityRepositoryFromPGXPool(pool *pgxpool.Pool) (*BillingAuthorityRepository, error) {
	if pool == nil {
		return nil, fmt.Errorf("%w: pgx pool is required", ErrBillingAuthorityRepository)
	}
	return &BillingAuthorityRepository{
		pool:    pool,
		queries: sqlcgen.New(pool),
	}, nil
}

func (r *BillingAuthorityRepository) ReadTerminalChildDebitByAuthorization(ctx context.Context, microleaseID string, debitAuthorizationID string) (TerminalChildDebitRecord, error) {
	if err := r.require(); err != nil {
		return TerminalChildDebitRecord{}, err
	}
	leaseID, err := uuidValue(microleaseID)
	if err != nil {
		return TerminalChildDebitRecord{}, err
	}
	row, err := r.queries.GetMicroleaseChildDebitByAuthorization(ctx, sqlcgen.GetMicroleaseChildDebitByAuthorizationParams{
		MicroleaseID:         leaseID,
		DebitAuthorizationID: debitAuthorizationID,
	})
	if err != nil {
		return TerminalChildDebitRecord{}, fmt.Errorf("%w: read terminal child debit: %w", ErrBillingAuthorityRepository, err)
	}
	return mapTerminalChildDebitRecord(row), nil
}

func (r *BillingAuthorityRepository) ResolveAccount(ctx context.Context, req billingauthority.AccountResolveRequest) (billingauthority.AccountSnapshot, error) {
	if err := r.require(); err != nil {
		return billingauthority.AccountSnapshot{}, err
	}
	accountScopeKey := resolveAccountScopeKey(req)
	account, err := r.queries.GetBillingAccountByScope(ctx, accountScopeKey)
	if errors.Is(err, pgx.ErrNoRows) {
		return billingauthority.AccountSnapshot{
			AccountScopeKey:     accountScopeKey,
			AccountState:        "not_found",
			ImportState:         "missing",
			MigrationState:      "legacy",
			BalanceReadEligible: false,
		}, nil
	}
	if err != nil {
		return billingauthority.AccountSnapshot{}, fmt.Errorf("%w: resolve account: %w", ErrBillingAuthorityRepository, err)
	}

	importState, err := r.importState(ctx, account.AccountScopeKey)
	if err != nil {
		return billingauthority.AccountSnapshot{}, err
	}
	var balanceVersion int64
	balance, err := r.queries.GetAccountBalanceByScope(ctx, account.AccountScopeKey)
	if err == nil {
		balanceVersion = balance.Version
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return billingauthority.AccountSnapshot{}, fmt.Errorf("%w: read account balance: %w", ErrBillingAuthorityRepository, err)
	}

	return billingauthority.AccountSnapshot{
		AccountID:           uuidString(account.AccountID),
		AccountScopeKey:     account.AccountScopeKey,
		AccountState:        account.State,
		ImportState:         importState,
		MigrationState:      "migrated",
		BalanceReadEligible: account.State == "active" && importState == "accepted",
		BalanceVersion:      balanceVersion,
		FailureClass:        accountResolveFailureClass(account.State, importState),
		Retryable:           importState == "pending",
	}, nil
}

func (r *BillingAuthorityRepository) ReadBalance(ctx context.Context, req billingauthority.BalanceReadRequest) (billingauthority.BalanceSnapshot, error) {
	if err := r.require(); err != nil {
		return billingauthority.BalanceSnapshot{}, err
	}
	account, err := r.queries.GetBillingAccountByScope(ctx, req.AccountScopeKey)
	if errors.Is(err, pgx.ErrNoRows) {
		return billingauthority.BalanceSnapshot{
			AccountScopeKey:   req.AccountScopeKey,
			AccountState:      "not_found",
			ImportState:       "missing",
			RuntimeGateState:  "fail_closed",
			ReasonCode:        "account_not_found",
			ManualReview:      false,
			BalanceVersion:    0,
			SettledUSDAtoms:   0,
			ReservedUSDAtoms:  0,
			AvailableUSDAtoms: 0,
		}, nil
	}
	if err != nil {
		return billingauthority.BalanceSnapshot{}, fmt.Errorf("%w: read account: %w", ErrBillingAuthorityRepository, err)
	}
	balance, err := r.queries.GetAccountBalanceByScope(ctx, req.AccountScopeKey)
	if err != nil {
		return billingauthority.BalanceSnapshot{}, fmt.Errorf("%w: read balance: %w", ErrBillingAuthorityRepository, err)
	}
	importState, err := r.importState(ctx, req.AccountScopeKey)
	if err != nil {
		return billingauthority.BalanceSnapshot{}, err
	}
	exposure, err := r.queries.GetActiveExposureByAccountScope(ctx, req.AccountScopeKey)
	if err != nil {
		return billingauthority.BalanceSnapshot{}, fmt.Errorf("%w: read active exposure: %w", ErrBillingAuthorityRepository, err)
	}

	manualReview := account.State == "manual_review" || importState == "mismatch"
	return billingauthority.BalanceSnapshot{
		AccountID:                    uuidString(account.AccountID),
		AccountScopeKey:              req.AccountScopeKey,
		AccountState:                 account.State,
		SettledUSDAtoms:              balance.SettledUsdAtoms,
		ReservedUSDAtoms:             balance.ReservedUsdAtoms,
		AvailableUSDAtoms:            balance.AvailableUsdAtoms,
		PendingUSDAtoms:              balance.PendingUsdAtoms,
		BalanceVersion:               balance.Version,
		ImportState:                  importState,
		RuntimeGateState:             runtimeStateForBalance(account.State, importState),
		ActiveMicroleaseUSDAtoms:     exposure.ActiveMicroleaseUsdAtoms,
		ActiveUsageHoldUSDAtoms:      exposure.ActiveUsageHoldUsdAtoms,
		UnresolvedChildDebitUSDAtoms: exposure.UnresolvedChildDebitUsdAtoms,
		ManualReview:                 manualReview,
		ReconciliationRequired:       importState == "mismatch",
		ReasonCode:                   accountResolveFailureClass(account.State, importState),
	}, nil
}

func (r *BillingAuthorityRepository) ReserveUsage(ctx context.Context, cmd billingauthority.UsageReserveCommand) (billingauthority.UsageOperationSnapshot, error) {
	if err := r.require(); err != nil {
		return billingauthority.UsageOperationSnapshot{}, err
	}
	ids, err := reserveCommandIDs(cmd)
	if err != nil {
		return billingauthority.UsageOperationSnapshot{}, err
	}
	safeMetadata, err := marshalSafeObject(cmd.Metadata)
	if err != nil {
		return billingauthority.UsageOperationSnapshot{}, err
	}
	now := time.Now().UTC()

	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return billingauthority.UsageOperationSnapshot{}, fmt.Errorf("%w: begin reserve transaction: %w", ErrBillingAuthorityRepository, err)
	}
	defer rollbackUnlessCommitted(ctx, tx)
	q := r.queries.WithTx(tx)

	account, err := q.GetBillingAccountByScope(ctx, cmd.AccountScopeKey)
	if errors.Is(err, pgx.ErrNoRows) {
		return billingauthority.UsageOperationSnapshot{}, fmt.Errorf("%w: account_not_found", billingauthority.ErrRejected)
	}
	if err != nil {
		return billingauthority.UsageOperationSnapshot{}, fmt.Errorf("%w: read reserve account: %w", ErrBillingAuthorityRepository, err)
	}
	if account.State != "active" {
		return billingauthority.UsageOperationSnapshot{}, fmt.Errorf("%w: account_not_active", billingauthority.ErrRejected)
	}

	replay, handled, err := r.handleExistingIdempotency(ctx, q, account.AccountID, "reserve", cmd.IdempotencyKey, cmd.RequestFingerprint, cmd.UsageOperationID, now)
	if handled || err != nil {
		return replay, err
	}

	lease, err := q.LockSpendingMicrolease(ctx, ids.microleaseID)
	if errors.Is(err, pgx.ErrNoRows) {
		return billingauthority.UsageOperationSnapshot{}, fmt.Errorf("%w: microlease_not_found", billingauthority.ErrRejected)
	}
	if err != nil {
		return billingauthority.UsageOperationSnapshot{}, fmt.Errorf("%w: lock reserve microlease: %w", ErrBillingAuthorityRepository, err)
	}
	if err := validateReserveLease(cmd, account, lease, now); err != nil {
		return billingauthority.UsageOperationSnapshot{}, err
	}

	idempotencyID := newPGUUID()
	outcomeID := newPGUUID()
	if _, err = q.CreateIdempotencyRecord(ctx, sqlcgen.CreateIdempotencyRecordParams{
		IdempotencyRecordID: idempotencyID,
		AccountID:           account.AccountID,
		OperationKind:       "reserve",
		IdempotencyKey:      cmd.IdempotencyKey,
		RequestFingerprint:  cmd.RequestFingerprint,
		RetentionClass:      "hot_replay",
		FirstSeenAt:         timestamptzValue(now),
	}); err != nil {
		return billingauthority.UsageOperationSnapshot{}, translateAuthorityWriteError("create reserve idempotency", err)
	}
	operation, err := q.CreateUsageOperation(ctx, sqlcgen.CreateUsageOperationParams{
		UsageOperationID:           ids.usageOperationID,
		AccountID:                  account.AccountID,
		AccountScopeKey:            account.AccountScopeKey,
		State:                      "reserved",
		OperationKind:              "reserve",
		ClientUsageRequestID:       cmd.DebitAuthorizationID,
		RequestID:                  optionalString(cmd.RequestID),
		RequestBasisFingerprint:    cmd.RequestFingerprint,
		PricingSnapshotID:          cmd.Pricing.ID,
		PricingSnapshotFingerprint: cmd.Pricing.Fingerprint,
		QuoteExpiresAt:             timestamptzValue(now.Add(5 * time.Minute)),
		FeePolicyVersion:           cmd.Pricing.PolicyVersion,
		ReservePolicyVersion:       "microlease-child-debit-v1",
		CreatedAt:                  timestamptzValue(now),
		ReservedAt:                 timestamptzValue(now),
	})
	if err != nil {
		return billingauthority.UsageOperationSnapshot{}, translateAuthorityWriteError("create usage operation", err)
	}
	if _, err = q.CreateOperationOutcome(ctx, sqlcgen.CreateOperationOutcomeParams{
		StoredOutcomeID:     outcomeID,
		IdempotencyRecordID: idempotencyID,
		AccountID:           account.AccountID,
		OperationKind:       "reserve",
		OutcomeStatus:       "success",
		PrimaryResourceType: "usage_operation",
		PrimaryResourceID:   cmd.UsageOperationID,
		SafeOutcome:         safePayload(map[string]any{"usage_operation_id": cmd.UsageOperationID, "microlease_child_debit_id": cmd.MicroleaseChildDebitID, "result": "reserved"}),
		CreatedAt:           timestamptzValue(now),
	}); err != nil {
		return billingauthority.UsageOperationSnapshot{}, translateAuthorityWriteError("create reserve outcome", err)
	}
	if _, err = q.MarkIdempotencyCommitted(ctx, sqlcgen.MarkIdempotencyCommittedParams{
		IdempotencyRecordID: idempotencyID,
		State:               "committed",
		StoredOutcomeID:     outcomeID,
		LastSeenAt:          timestamptzValue(now),
	}); err != nil {
		return billingauthority.UsageOperationSnapshot{}, fmt.Errorf("%w: commit reserve idempotency: %w", ErrBillingAuthorityRepository, err)
	}
	if _, err = q.CreateMicroleaseChildDebit(ctx, sqlcgen.CreateMicroleaseChildDebitParams{
		MicroleaseChildDebitID:     ids.childDebitID,
		MicroleaseID:               ids.microleaseID,
		DebitAuthorizationID:       cmd.DebitAuthorizationID,
		UsageOperationID:           ids.usageOperationID,
		AccountID:                  account.AccountID,
		AccountScopeKey:            account.AccountScopeKey,
		ProxyAllocatorOwnerID:      cmd.ProxyAllocatorOwnerID,
		MicroleaseGeneration:       cmd.MicroleaseGeneration,
		ChildSequence:              cmd.ChildSequence,
		ChildCapUsdAtoms:           cmd.ChildCapUSDAtoms,
		RequestBasisFingerprint:    cmd.RequestFingerprint,
		PricingSnapshotID:          cmd.Pricing.ID,
		PricingSnapshotFingerprint: cmd.Pricing.Fingerprint,
		TerminalKind:               "pending",
		State:                      "terminal_pending",
		SafeMetadata:               safeMetadata,
		CreatedAt:                  timestamptzValue(now),
	}); err != nil {
		return billingauthority.UsageOperationSnapshot{}, translateAuthorityWriteError("create reserve child debit", err)
	}
	if _, err = q.UpdateSpendingMicroleaseSettlementTotals(ctx, sqlcgen.UpdateSpendingMicroleaseSettlementTotalsParams{
		MicroleaseID:                      ids.microleaseID,
		State:                             lease.State,
		AvailableChildCapUsdAtoms:         lease.AvailableChildCapUsdAtoms - cmd.ChildCapUSDAtoms,
		AllocatedChildCapReportedUsdAtoms: lease.AllocatedChildCapReportedUsdAtoms + cmd.ChildCapUSDAtoms,
		TerminalChargedUsdAtoms:           lease.TerminalChargedUsdAtoms,
		TerminalReleasedUsdAtoms:          lease.TerminalReleasedUsdAtoms,
		WriteOffUsdAtoms:                  lease.WriteOffUsdAtoms,
		LastCheckpointSequence:            lease.LastCheckpointSequence,
		LastCheckpointFingerprint:         lease.LastCheckpointFingerprint,
		ClosedAt:                          lease.ClosedAt,
		UpdatedAt:                         timestamptzValue(now),
	}); err != nil {
		return billingauthority.UsageOperationSnapshot{}, fmt.Errorf("%w: update reserve microlease totals: %w", ErrBillingAuthorityRepository, err)
	}
	if err = tx.Commit(ctx); err != nil {
		return billingauthority.UsageOperationSnapshot{}, fmt.Errorf("%w: commit reserve transaction: %w", ErrBillingAuthorityRepository, err)
	}
	return usageSnapshot(operation, "accepted", cmd.IdempotencyKey, cmd.RequestFingerprint, uuidString(outcomeID)), nil
}

//nolint:cyclop // Linear transaction orchestration keeps lock/write/commit order auditable for money effects.
func (r *BillingAuthorityRepository) CompleteUsage(ctx context.Context, cmd billingauthority.UsageTerminalCommand) (billingauthority.UsageOperationSnapshot, error) {
	if err := r.require(); err != nil {
		return billingauthority.UsageOperationSnapshot{}, err
	}
	ids, err := terminalCommandIDsFromAuthority(cmd)
	if err != nil {
		return billingauthority.UsageOperationSnapshot{}, err
	}
	safeMetadata, err := marshalSafeObject(cmd.Metadata)
	if err != nil {
		return billingauthority.UsageOperationSnapshot{}, err
	}
	now := time.Now().UTC()

	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return billingauthority.UsageOperationSnapshot{}, fmt.Errorf("%w: begin terminal transaction: %w", ErrBillingAuthorityRepository, err)
	}
	defer rollbackUnlessCommitted(ctx, tx)
	q := r.queries.WithTx(tx)

	account, err := q.GetBillingAccountByScope(ctx, cmd.AccountScopeKey)
	if errors.Is(err, pgx.ErrNoRows) {
		return billingauthority.UsageOperationSnapshot{}, fmt.Errorf("%w: account_not_found", billingauthority.ErrRejected)
	}
	if err != nil {
		return billingauthority.UsageOperationSnapshot{}, fmt.Errorf("%w: read terminal account: %w", ErrBillingAuthorityRepository, err)
	}
	replay, handled, err := r.handleExistingIdempotency(ctx, q, account.AccountID, cmd.TerminalKind, cmd.IdempotencyKey, cmd.RequestFingerprint, cmd.UsageOperationID, now)
	if handled || err != nil {
		return replay, err
	}

	_, child, lease, balance, err := lockTerminalRows(ctx, q, ids, account, cmd)
	if err != nil {
		return billingauthority.UsageOperationSnapshot{}, err
	}
	if err := validateTerminalAmounts(cmd, child, balance); err != nil {
		return billingauthority.UsageOperationSnapshot{}, err
	}

	idempotencyID := newPGUUID()
	outcomeID := newPGUUID()
	settlementEffectID := newPGUUID()
	ledgerID := newPGUUID()
	if _, err = q.CreateIdempotencyRecord(ctx, sqlcgen.CreateIdempotencyRecordParams{
		IdempotencyRecordID: idempotencyID,
		AccountID:           account.AccountID,
		OperationKind:       cmd.TerminalKind,
		IdempotencyKey:      cmd.IdempotencyKey,
		RequestFingerprint:  cmd.RequestFingerprint,
		RetentionClass:      "hot_replay",
		FirstSeenAt:         timestamptzValue(now),
	}); err != nil {
		return billingauthority.UsageOperationSnapshot{}, translateAuthorityWriteError("create terminal idempotency", err)
	}
	ledgerParams := terminalLedgerParams(terminalSettlementCommand(cmd, account, child, now), balance, terminalCommandIDs{
		accountID:          account.AccountID,
		ledgerID:           ledgerID,
		settlementEffectID: settlementEffectID,
	}, safeMetadata, now)
	ledgerParams.IdempotencyRecordID = idempotencyID
	ledgerParams.UsageOperationID = ids.usageOperationID
	ledgerParams.CreatedByKind = "service"
	ledger, err := q.CreateLedgerEntry(ctx, ledgerParams)
	if err != nil {
		return billingauthority.UsageOperationSnapshot{}, translateAuthorityWriteError("create terminal ledger", err)
	}
	if _, err = q.UpdateAccountBalanceAfterLedger(ctx, sqlcgen.UpdateAccountBalanceAfterLedgerParams{
		AccountID:         account.AccountID,
		SettledUsdAtoms:   ledger.SettledAfterUsdAtoms,
		ReservedUsdAtoms:  ledger.ReservedAfterUsdAtoms,
		AvailableUsdAtoms: ledger.AvailableAfterUsdAtoms,
		PendingUsdAtoms:   ledger.PendingAfterUsdAtoms,
		LastLedgerEntryID: ledger.LedgerEntryID,
		UpdatedAt:         timestamptzValue(now),
	}); err != nil {
		return billingauthority.UsageOperationSnapshot{}, fmt.Errorf("%w: update terminal balance: %w", ErrBillingAuthorityRepository, err)
	}
	if _, err = q.CreateOperationOutcome(ctx, sqlcgen.CreateOperationOutcomeParams{
		StoredOutcomeID:     outcomeID,
		IdempotencyRecordID: idempotencyID,
		AccountID:           account.AccountID,
		OperationKind:       cmd.TerminalKind,
		OutcomeStatus:       "success",
		PrimaryResourceType: "usage_operation",
		PrimaryResourceID:   cmd.UsageOperationID,
		LedgerEntryID:       ledger.LedgerEntryID,
		SettlementEffectID:  settlementEffectID,
		SafeOutcome:         safePayload(map[string]any{"usage_operation_id": cmd.UsageOperationID, "microlease_child_debit_id": cmd.MicroleaseChildDebitID, "result": cmd.TerminalKind}),
		CreatedAt:           timestamptzValue(now),
	}); err != nil {
		return billingauthority.UsageOperationSnapshot{}, translateAuthorityWriteError("create terminal outcome", err)
	}
	if _, err = q.MarkIdempotencyCommitted(ctx, sqlcgen.MarkIdempotencyCommittedParams{
		IdempotencyRecordID: idempotencyID,
		State:               "committed",
		StoredOutcomeID:     outcomeID,
		LastSeenAt:          timestamptzValue(now),
	}); err != nil {
		return billingauthority.UsageOperationSnapshot{}, fmt.Errorf("%w: commit terminal idempotency: %w", ErrBillingAuthorityRepository, err)
	}
	if _, err = q.CreateUsageTerminalOutcome(ctx, sqlcgen.CreateUsageTerminalOutcomeParams{
		UsageTerminalOutcomeID: ids.terminalOutcomeID,
		UsageOperationID:       ids.usageOperationID,
		TerminalKind:           cmd.TerminalKind,
		IdempotencyRecordID:    idempotencyID,
		StoredOutcomeID:        outcomeID,
		LedgerEntryID:          ledger.LedgerEntryID,
		SettlementEffectID:     settlementEffectID,
		ChargedUsdAtoms:        cmd.ChargedUSDAtoms,
		ReleasedUsdAtoms:       cmd.ReleasedUSDAtoms,
		WriteOffUsdAtoms:       cmd.WriteOffUSDAtoms,
		CreatedAt:              timestamptzValue(now),
	}); err != nil {
		return billingauthority.UsageOperationSnapshot{}, translateAuthorityWriteError("create terminal usage outcome", err)
	}
	terminalState := usageTerminalState(cmd.TerminalKind)
	operation, err := q.UpdateUsageOperationTerminal(ctx, sqlcgen.UpdateUsageOperationTerminalParams{
		UsageOperationID:         ids.usageOperationID,
		State:                    terminalState,
		OperationKind:            cmd.TerminalKind,
		TerminalBasisFingerprint: &cmd.TerminalFingerprint,
		TerminalOutcomeID:        outcomeID,
		SettlementEffectID:       settlementEffectID,
		UpdatedAt:                timestamptzValue(now),
	})
	if err != nil {
		return billingauthority.UsageOperationSnapshot{}, fmt.Errorf("%w: update terminal usage operation: %w", ErrBillingAuthorityRepository, err)
	}
	if _, err = q.UpdateMicroleaseChildDebitTerminal(ctx, sqlcgen.UpdateMicroleaseChildDebitTerminalParams{
		MicroleaseChildDebitID:       ids.childDebitID,
		TerminalBasisFingerprint:     &cmd.TerminalFingerprint,
		TerminalKind:                 cmd.TerminalKind,
		State:                        terminalState,
		ChargedUsdAtoms:              cmd.ChargedUSDAtoms,
		ReleasedUsdAtoms:             cmd.ReleasedUSDAtoms,
		WriteOffUsdAtoms:             cmd.WriteOffUSDAtoms,
		QualifiedInferenceEvidenceID: ids.qualifiedInferenceEvidenceID,
		LedgerEntryID:                ledger.LedgerEntryID,
		SettlementEffectID:           settlementEffectID,
		TerminalAt:                   timestamptzValue(now),
		SettledAt:                    timestamptzValue(now),
		UpdatedAt:                    timestamptzValue(now),
	}); err != nil {
		return billingauthority.UsageOperationSnapshot{}, fmt.Errorf("%w: update terminal child debit: %w", ErrBillingAuthorityRepository, err)
	}
	if _, err = q.UpdateSpendingMicroleaseSettlementTotals(ctx, sqlcgen.UpdateSpendingMicroleaseSettlementTotalsParams{
		MicroleaseID:                      ids.microleaseID,
		State:                             lease.State,
		AvailableChildCapUsdAtoms:         lease.AvailableChildCapUsdAtoms,
		AllocatedChildCapReportedUsdAtoms: lease.AllocatedChildCapReportedUsdAtoms,
		TerminalChargedUsdAtoms:           lease.TerminalChargedUsdAtoms + cmd.ChargedUSDAtoms,
		TerminalReleasedUsdAtoms:          lease.TerminalReleasedUsdAtoms + cmd.ReleasedUSDAtoms,
		WriteOffUsdAtoms:                  lease.WriteOffUsdAtoms + cmd.WriteOffUSDAtoms,
		LastCheckpointSequence:            lease.LastCheckpointSequence,
		LastCheckpointFingerprint:         lease.LastCheckpointFingerprint,
		ClosedAt:                          lease.ClosedAt,
		UpdatedAt:                         timestamptzValue(now),
	}); err != nil {
		return billingauthority.UsageOperationSnapshot{}, fmt.Errorf("%w: update terminal microlease totals: %w", ErrBillingAuthorityRepository, err)
	}
	if err = tx.Commit(ctx); err != nil {
		return billingauthority.UsageOperationSnapshot{}, fmt.Errorf("%w: commit terminal transaction: %w", ErrBillingAuthorityRepository, err)
	}
	return usageSnapshot(operation, "accepted", cmd.IdempotencyKey, cmd.RequestFingerprint, uuidString(outcomeID)), nil
}

//nolint:cyclop // Linear transaction orchestration keeps reversal write ordering auditable.
func (r *BillingAuthorityRepository) ReverseUsage(ctx context.Context, cmd billingauthority.UsageReversalCommand) (billingauthority.UsageOperationSnapshot, error) {
	if err := r.require(); err != nil {
		return billingauthority.UsageOperationSnapshot{}, err
	}
	usageOperationID, err := uuidValue(cmd.UsageOperationID)
	if err != nil {
		return billingauthority.UsageOperationSnapshot{}, fmt.Errorf("%w: usage operation id: %w", billingauthority.ErrInvalidRequest, err)
	}
	originalLedgerEntryID, err := uuidValue(cmd.OriginalLedgerEntryID)
	if err != nil {
		return billingauthority.UsageOperationSnapshot{}, fmt.Errorf("%w: original ledger entry id: %w", billingauthority.ErrInvalidRequest, err)
	}
	safeMetadata, err := marshalSafeObject(cmd.Metadata)
	if err != nil {
		return billingauthority.UsageOperationSnapshot{}, err
	}
	now := time.Now().UTC()

	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return billingauthority.UsageOperationSnapshot{}, fmt.Errorf("%w: begin reversal transaction: %w", ErrBillingAuthorityRepository, err)
	}
	defer rollbackUnlessCommitted(ctx, tx)
	q := r.queries.WithTx(tx)

	account, err := q.GetBillingAccountByScope(ctx, cmd.AccountScopeKey)
	if errors.Is(err, pgx.ErrNoRows) {
		return billingauthority.UsageOperationSnapshot{}, fmt.Errorf("%w: account_not_found", billingauthority.ErrRejected)
	}
	if err != nil {
		return billingauthority.UsageOperationSnapshot{}, fmt.Errorf("%w: read reversal account: %w", ErrBillingAuthorityRepository, err)
	}
	replay, handled, err := r.handleExistingIdempotency(ctx, q, account.AccountID, "reversal", cmd.IdempotencyKey, cmd.RequestFingerprint, cmd.UsageOperationID, now)
	if handled || err != nil {
		return replay, err
	}
	operation, err := q.LockUsageOperation(ctx, usageOperationID)
	if err != nil {
		return billingauthority.UsageOperationSnapshot{}, fmt.Errorf("%w: lock reversal usage operation: %w", ErrBillingAuthorityRepository, err)
	}
	if operation.AccountScopeKey != cmd.AccountScopeKey || operation.AccountID != account.AccountID {
		return billingauthority.UsageOperationSnapshot{}, fmt.Errorf("%w: usage_operation_account_mismatch", billingauthority.ErrConflict)
	}
	balance, err := q.LockAccountBalanceByAccountID(ctx, account.AccountID)
	if err != nil {
		return billingauthority.UsageOperationSnapshot{}, fmt.Errorf("%w: lock reversal balance: %w", ErrBillingAuthorityRepository, err)
	}

	idempotencyID := newPGUUID()
	outcomeID := newPGUUID()
	usageTerminalOutcomeID := newPGUUID()
	settlementEffectID := newPGUUID()
	ledgerID := newPGUUID()
	if _, err = q.CreateIdempotencyRecord(ctx, sqlcgen.CreateIdempotencyRecordParams{
		IdempotencyRecordID: idempotencyID,
		AccountID:           account.AccountID,
		OperationKind:       "reversal",
		IdempotencyKey:      cmd.IdempotencyKey,
		RequestFingerprint:  cmd.RequestFingerprint,
		RetentionClass:      "hot_replay",
		FirstSeenAt:         timestamptzValue(now),
	}); err != nil {
		return billingauthority.UsageOperationSnapshot{}, translateAuthorityWriteError("create reversal idempotency", err)
	}
	settledAfter := balance.SettledUsdAtoms + cmd.ReversalUSDAtoms
	ledger, err := q.CreateLedgerEntry(ctx, sqlcgen.CreateLedgerEntryParams{
		LedgerEntryID:           ledgerID,
		AccountID:               account.AccountID,
		AccountScopeKey:         account.AccountScopeKey,
		EffectType:              "microlease_reversal",
		AmountUsdAtoms:          cmd.ReversalUSDAtoms,
		SettledDeltaUsdAtoms:    cmd.ReversalUSDAtoms,
		SettledAfterUsdAtoms:    settledAfter,
		ReservedAfterUsdAtoms:   balance.ReservedUsdAtoms,
		AvailableAfterUsdAtoms:  settledAfter - balance.ReservedUsdAtoms,
		PendingAfterUsdAtoms:    balance.PendingUsdAtoms,
		BalanceVersionAfter:     balance.Version + 1,
		SettlementEffectID:      settlementEffectID,
		IdempotencyRecordID:     idempotencyID,
		UsageOperationID:        usageOperationID,
		ReversalOfLedgerEntryID: originalLedgerEntryID,
		EffectiveAt:             timestamptzValue(now),
		CreatedAt:               timestamptzValue(now),
		CreatedByKind:           "service",
		ReasonCode:              cmd.ReasonCode,
		SafeMetadata:            safeMetadata,
	})
	if err != nil {
		return billingauthority.UsageOperationSnapshot{}, translateAuthorityWriteError("create reversal ledger", err)
	}
	if _, err = q.UpdateAccountBalanceAfterLedger(ctx, sqlcgen.UpdateAccountBalanceAfterLedgerParams{
		AccountID:         account.AccountID,
		SettledUsdAtoms:   ledger.SettledAfterUsdAtoms,
		ReservedUsdAtoms:  ledger.ReservedAfterUsdAtoms,
		AvailableUsdAtoms: ledger.AvailableAfterUsdAtoms,
		PendingUsdAtoms:   ledger.PendingAfterUsdAtoms,
		LastLedgerEntryID: ledger.LedgerEntryID,
		UpdatedAt:         timestamptzValue(now),
	}); err != nil {
		return billingauthority.UsageOperationSnapshot{}, fmt.Errorf("%w: update reversal balance: %w", ErrBillingAuthorityRepository, err)
	}
	if _, err = q.CreateOperationOutcome(ctx, sqlcgen.CreateOperationOutcomeParams{
		StoredOutcomeID:     outcomeID,
		IdempotencyRecordID: idempotencyID,
		AccountID:           account.AccountID,
		OperationKind:       "reversal",
		OutcomeStatus:       "success",
		PrimaryResourceType: "usage_operation",
		PrimaryResourceID:   cmd.UsageOperationID,
		LedgerEntryID:       ledger.LedgerEntryID,
		SettlementEffectID:  settlementEffectID,
		SafeOutcome:         safePayload(map[string]any{"usage_operation_id": cmd.UsageOperationID, "result": "reversed"}),
		CreatedAt:           timestamptzValue(now),
	}); err != nil {
		return billingauthority.UsageOperationSnapshot{}, translateAuthorityWriteError("create reversal outcome", err)
	}
	if _, err = q.MarkIdempotencyCommitted(ctx, sqlcgen.MarkIdempotencyCommittedParams{
		IdempotencyRecordID: idempotencyID,
		State:               "committed",
		StoredOutcomeID:     outcomeID,
		LastSeenAt:          timestamptzValue(now),
	}); err != nil {
		return billingauthority.UsageOperationSnapshot{}, fmt.Errorf("%w: commit reversal idempotency: %w", ErrBillingAuthorityRepository, err)
	}
	if _, err = q.CreateUsageTerminalOutcome(ctx, sqlcgen.CreateUsageTerminalOutcomeParams{
		UsageTerminalOutcomeID: usageTerminalOutcomeID,
		UsageOperationID:       usageOperationID,
		TerminalKind:           "reversal",
		IdempotencyRecordID:    idempotencyID,
		StoredOutcomeID:        outcomeID,
		LedgerEntryID:          ledger.LedgerEntryID,
		SettlementEffectID:     settlementEffectID,
		CreatedAt:              timestamptzValue(now),
	}); err != nil {
		return billingauthority.UsageOperationSnapshot{}, translateAuthorityWriteError("create reversal usage outcome", err)
	}
	operation, err = q.UpdateUsageOperationTerminal(ctx, sqlcgen.UpdateUsageOperationTerminalParams{
		UsageOperationID:         usageOperationID,
		State:                    "reversed",
		OperationKind:            "reversal",
		TerminalBasisFingerprint: &cmd.RequestFingerprint,
		TerminalOutcomeID:        outcomeID,
		SettlementEffectID:       settlementEffectID,
		UpdatedAt:                timestamptzValue(now),
	})
	if err != nil {
		return billingauthority.UsageOperationSnapshot{}, fmt.Errorf("%w: update reversal usage operation: %w", ErrBillingAuthorityRepository, err)
	}
	if err = tx.Commit(ctx); err != nil {
		return billingauthority.UsageOperationSnapshot{}, fmt.Errorf("%w: commit reversal transaction: %w", ErrBillingAuthorityRepository, err)
	}
	return usageSnapshot(operation, "accepted", cmd.IdempotencyKey, cmd.RequestFingerprint, uuidString(outcomeID)), nil
}

func (r *BillingAuthorityRepository) ReadUsageOperation(ctx context.Context, req billingauthority.UsageReadbackRequest) (billingauthority.UsageOperationSnapshot, error) {
	if err := r.require(); err != nil {
		return billingauthority.UsageOperationSnapshot{}, err
	}
	usageOperationID, err := uuidValue(req.UsageOperationID)
	if err != nil {
		return billingauthority.UsageOperationSnapshot{}, fmt.Errorf("%w: usage operation id: %w", billingauthority.ErrInvalidRequest, err)
	}
	operation, err := r.queries.GetUsageOperation(ctx, usageOperationID)
	if errors.Is(err, pgx.ErrNoRows) {
		return billingauthority.UsageOperationSnapshot{
			UsageOperationID: req.UsageOperationID,
			State:            "not_found",
			ResultCode:       "readback",
			ReasonCode:       "usage_operation_not_found",
		}, nil
	}
	if err != nil {
		return billingauthority.UsageOperationSnapshot{}, fmt.Errorf("%w: read usage operation: %w", ErrBillingAuthorityRepository, err)
	}
	if operation.AccountScopeKey != req.AccountScopeKey {
		return billingauthority.UsageOperationSnapshot{}, fmt.Errorf("%w: usage_operation_account_mismatch", billingauthority.ErrConflict)
	}
	snapshot := usageSnapshot(operation, "readback", "", operation.RequestBasisFingerprint, "")
	child, err := r.queries.GetMicroleaseChildDebitByUsageOperation(ctx, usageOperationID)
	if err == nil {
		snapshot.BillingOperationID = uuidString(child.MicroleaseChildDebitID)
	}
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return billingauthority.UsageOperationSnapshot{}, fmt.Errorf("%w: read usage child debit: %w", ErrBillingAuthorityRepository, err)
	}
	return snapshot, nil
}

func (r *BillingAuthorityRepository) ListReconciliationCases(ctx context.Context, req billingauthority.ReconciliationCasesRequest) ([]billingauthority.ReconciliationCase, error) {
	if err := r.require(); err != nil {
		return nil, err
	}
	if req.AccountScopeKey == "" {
		return []billingauthority.ReconciliationCase{}, nil
	}
	account, err := r.queries.GetBillingAccountByScope(ctx, req.AccountScopeKey)
	if errors.Is(err, pgx.ErrNoRows) {
		return []billingauthority.ReconciliationCase{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("%w: read reconciliation account: %w", ErrBillingAuthorityRepository, err)
	}
	limit := req.Limit
	if limit <= 0 {
		limit = 100
	}
	rows, err := r.queries.ListReconciliationCasesByAccount(ctx, sqlcgen.ListReconciliationCasesByAccountParams{
		AccountID: account.AccountID,
		Limit:     limit,
	})
	if err != nil {
		return nil, fmt.Errorf("%w: list reconciliation cases: %w", ErrBillingAuthorityRepository, err)
	}
	cases := make([]billingauthority.ReconciliationCase, 0, len(rows))
	for _, row := range rows {
		if req.State != "" && row.State != req.State {
			continue
		}
		if req.Severity != "" && row.Severity != req.Severity {
			continue
		}
		cases = append(cases, billingauthority.ReconciliationCase{
			ReconciliationCaseID: uuidString(row.ReconciliationCaseID),
			AccountScopeKey:      req.AccountScopeKey,
			Reason:               row.Reason,
			State:                row.State,
			Severity:             row.Severity,
			SafeLineageID:        reconciliationSafeLineage(row),
		})
	}
	return cases, nil
}

func (r *BillingAuthorityRepository) ListLedgerEntries(ctx context.Context, req billingauthority.AdminLedgerRequest) ([]billingauthority.LedgerEntry, error) {
	if err := r.require(); err != nil {
		return nil, err
	}
	account, err := r.queries.GetBillingAccountByScope(ctx, req.AccountScopeKey)
	if errors.Is(err, pgx.ErrNoRows) {
		return []billingauthority.LedgerEntry{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("%w: read ledger account: %w", ErrBillingAuthorityRepository, err)
	}
	limit := req.Limit
	if limit <= 0 {
		limit = 50
	}
	rows, err := r.queries.ListLedgerEntriesByAccount(ctx, sqlcgen.ListLedgerEntriesByAccountParams{
		AccountID: account.AccountID,
		Limit:     limit,
	})
	if err != nil {
		return nil, fmt.Errorf("%w: list ledger entries: %w", ErrBillingAuthorityRepository, err)
	}
	entries := make([]billingauthority.LedgerEntry, 0, len(rows))
	for _, row := range rows {
		entries = append(entries, billingauthority.LedgerEntry{
			LedgerEntryID:       uuidString(row.LedgerEntryID),
			EffectType:          row.EffectType,
			AmountUSDAtoms:      row.AmountUsdAtoms,
			BalanceVersionAfter: row.BalanceVersionAfter,
		})
	}
	return entries, nil
}

func (r *BillingAuthorityRepository) ReadExposure(ctx context.Context, req billingauthority.AdminExposureRequest) (billingauthority.ExposureSnapshot, error) {
	if err := r.require(); err != nil {
		return billingauthority.ExposureSnapshot{}, err
	}
	exposure, err := r.queries.GetActiveExposureByAccountScope(ctx, req.AccountScopeKey)
	if err != nil {
		return billingauthority.ExposureSnapshot{}, fmt.Errorf("%w: read admin exposure: %w", ErrBillingAuthorityRepository, err)
	}
	return billingauthority.ExposureSnapshot{
		AccountScopeKey:              req.AccountScopeKey,
		RuntimeGateState:             "ready",
		ActiveMicroleaseUSDAtoms:     exposure.ActiveMicroleaseUsdAtoms,
		ActiveUsageHoldUSDAtoms:      exposure.ActiveUsageHoldUsdAtoms,
		UnresolvedChildDebitUSDAtoms: exposure.UnresolvedChildDebitUsdAtoms,
	}, nil
}

func (r *BillingAuthorityRepository) require() error {
	if r == nil || r.pool == nil || r.queries == nil {
		return fmt.Errorf("%w: %w: repository is not configured", ErrBillingAuthorityRepository, billingauthority.ErrNotReady)
	}
	return nil
}

func (r *BillingAuthorityRepository) importState(ctx context.Context, accountScopeKey string) (string, error) {
	row, err := r.queries.GetLatestAcceptedLegacyImportByAccountScope(ctx, accountScopeKey)
	if errors.Is(err, pgx.ErrNoRows) {
		return "missing", nil
	}
	if err != nil {
		return "", fmt.Errorf("%w: read latest accepted legacy import: %w", ErrBillingAuthorityRepository, err)
	}
	switch row.ParityStatus {
	case "matched", "corrected":
		return "accepted", nil
	case "mismatch":
		return "mismatch", nil
	default:
		return "pending", nil
	}
}

func (r *BillingAuthorityRepository) handleExistingIdempotency(ctx context.Context, q *sqlcgen.Queries, accountID pgtype.UUID, operationKind, idempotencyKey, requestFingerprint, usageOperationID string, now time.Time) (billingauthority.UsageOperationSnapshot, bool, error) {
	record, err := q.LockIdempotencyRecord(ctx, sqlcgen.LockIdempotencyRecordParams{
		AccountID:      accountID,
		OperationKind:  operationKind,
		IdempotencyKey: idempotencyKey,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return billingauthority.UsageOperationSnapshot{}, false, nil
	}
	if err != nil {
		return billingauthority.UsageOperationSnapshot{}, true, fmt.Errorf("%w: lock idempotency: %w", ErrBillingAuthorityRepository, err)
	}
	if record.RequestFingerprint != requestFingerprint {
		if !record.StoredOutcomeID.Valid {
			reason := "changed_fingerprint"
			if _, markErr := q.MarkIdempotencyConflict(ctx, sqlcgen.MarkIdempotencyConflictParams{
				IdempotencyRecordID: record.IdempotencyRecordID,
				ConflictReason:      &reason,
				LastSeenAt:          timestamptzValue(now),
			}); markErr != nil {
				return billingauthority.UsageOperationSnapshot{}, true, fmt.Errorf("%w: mark idempotency conflict: %w", ErrBillingAuthorityRepository, markErr)
			}
		}
		return billingauthority.UsageOperationSnapshot{}, true, fmt.Errorf("%w: payload_conflict", billingauthority.ErrConflict)
	}
	if !record.StoredOutcomeID.Valid {
		return billingauthority.UsageOperationSnapshot{}, true, fmt.Errorf("%w: idempotency_in_progress", billingauthority.ErrConflict)
	}
	outcome, err := q.GetOperationOutcomeByID(ctx, record.StoredOutcomeID)
	if err != nil {
		return billingauthority.UsageOperationSnapshot{}, true, fmt.Errorf("%w: read stored outcome: %w", ErrBillingAuthorityRepository, err)
	}
	resourceID := usageOperationID
	if outcome.PrimaryResourceID != "" {
		resourceID = outcome.PrimaryResourceID
	}
	operation, err := q.GetUsageOperation(ctx, mustUUIDOrZero(resourceID))
	if err != nil {
		return billingauthority.UsageOperationSnapshot{}, true, fmt.Errorf("%w: read replay usage operation: %w", ErrBillingAuthorityRepository, err)
	}
	return usageSnapshot(operation, "duplicate_stored_outcome", record.IdempotencyKey, record.RequestFingerprint, uuidString(record.StoredOutcomeID)), true, nil
}

type reserveIDs struct {
	usageOperationID pgtype.UUID
	microleaseID     pgtype.UUID
	childDebitID     pgtype.UUID
}

func reserveCommandIDs(cmd billingauthority.UsageReserveCommand) (reserveIDs, error) {
	usageOperationID, err := uuidValue(cmd.UsageOperationID)
	if err != nil {
		return reserveIDs{}, fmt.Errorf("%w: usage operation id: %w", billingauthority.ErrInvalidRequest, err)
	}
	microleaseID, err := uuidValue(cmd.MicroleaseID)
	if err != nil {
		return reserveIDs{}, fmt.Errorf("%w: microlease id: %w", billingauthority.ErrInvalidRequest, err)
	}
	childDebitID, err := uuidValue(cmd.MicroleaseChildDebitID)
	if err != nil {
		return reserveIDs{}, fmt.Errorf("%w: microlease child debit id: %w", billingauthority.ErrInvalidRequest, err)
	}
	return reserveIDs{usageOperationID: usageOperationID, microleaseID: microleaseID, childDebitID: childDebitID}, nil
}

type authorityTerminalIDs struct {
	usageOperationID             pgtype.UUID
	microleaseID                 pgtype.UUID
	childDebitID                 pgtype.UUID
	terminalOutcomeID            pgtype.UUID
	qualifiedInferenceEvidenceID pgtype.UUID
}

func terminalCommandIDsFromAuthority(cmd billingauthority.UsageTerminalCommand) (authorityTerminalIDs, error) {
	usageOperationID, err := uuidValue(cmd.UsageOperationID)
	if err != nil {
		return authorityTerminalIDs{}, fmt.Errorf("%w: usage operation id: %w", billingauthority.ErrInvalidRequest, err)
	}
	microleaseID, err := uuidValue(cmd.MicroleaseID)
	if err != nil {
		return authorityTerminalIDs{}, fmt.Errorf("%w: microlease id: %w", billingauthority.ErrInvalidRequest, err)
	}
	childDebitID, err := uuidValue(cmd.MicroleaseChildDebitID)
	if err != nil {
		return authorityTerminalIDs{}, fmt.Errorf("%w: microlease child debit id: %w", billingauthority.ErrInvalidRequest, err)
	}
	terminalOutcomeID, err := uuidValue(cmd.TerminalOutcomeID)
	if err != nil {
		return authorityTerminalIDs{}, fmt.Errorf("%w: terminal outcome id: %w", billingauthority.ErrInvalidRequest, err)
	}
	qualifiedInferenceEvidenceID, err := optionalUUID(cmd.QualifiedInferenceEvidenceID)
	if err != nil {
		return authorityTerminalIDs{}, fmt.Errorf("%w: qualified inference evidence id: %w", billingauthority.ErrInvalidRequest, err)
	}
	return authorityTerminalIDs{
		usageOperationID:             usageOperationID,
		microleaseID:                 microleaseID,
		childDebitID:                 childDebitID,
		terminalOutcomeID:            terminalOutcomeID,
		qualifiedInferenceEvidenceID: qualifiedInferenceEvidenceID,
	}, nil
}

func validateReserveLease(cmd billingauthority.UsageReserveCommand, account sqlcgen.BillingAccount, lease sqlcgen.SpendingMicrolease, now time.Time) error {
	if lease.AccountID != account.AccountID || lease.AccountScopeKey != account.AccountScopeKey {
		return fmt.Errorf("%w: microlease_account_mismatch", billingauthority.ErrConflict)
	}
	if lease.ProxyAllocatorOwnerID != cmd.ProxyAllocatorOwnerID ||
		lease.MicroleaseGeneration != cmd.MicroleaseGeneration ||
		lease.LeaseFence != cmd.LeaseFence {
		return fmt.Errorf("%w: microlease_lineage_mismatch", billingauthority.ErrConflict)
	}
	if lease.State != "active" {
		return fmt.Errorf("%w: microlease_not_active", billingauthority.ErrRejected)
	}
	if lease.DebitCutoffAt.Valid && !now.Before(lease.DebitCutoffAt.Time) {
		return fmt.Errorf("%w: microlease_cutoff", billingauthority.ErrRejected)
	}
	if lease.AvailableChildCapUsdAtoms < cmd.ChildCapUSDAtoms {
		return fmt.Errorf("%w: microlease_child_cap_exhausted", billingauthority.ErrRejected)
	}
	return nil
}

func lockTerminalRows(ctx context.Context, q *sqlcgen.Queries, ids authorityTerminalIDs, account sqlcgen.BillingAccount, cmd billingauthority.UsageTerminalCommand) (sqlcgen.UsageOperation, sqlcgen.MicroleaseChildDebit, sqlcgen.SpendingMicrolease, sqlcgen.AccountBalance, error) {
	operation, err := q.LockUsageOperation(ctx, ids.usageOperationID)
	if err != nil {
		return sqlcgen.UsageOperation{}, sqlcgen.MicroleaseChildDebit{}, sqlcgen.SpendingMicrolease{}, sqlcgen.AccountBalance{}, fmt.Errorf("%w: lock usage operation: %w", ErrBillingAuthorityRepository, err)
	}
	child, err := q.LockMicroleaseChildDebit(ctx, ids.childDebitID)
	if errors.Is(err, pgx.ErrNoRows) {
		return sqlcgen.UsageOperation{}, sqlcgen.MicroleaseChildDebit{}, sqlcgen.SpendingMicrolease{}, sqlcgen.AccountBalance{}, fmt.Errorf("%w: child_debit_not_found", billingauthority.ErrRejected)
	}
	if err != nil {
		return sqlcgen.UsageOperation{}, sqlcgen.MicroleaseChildDebit{}, sqlcgen.SpendingMicrolease{}, sqlcgen.AccountBalance{}, fmt.Errorf("%w: lock child debit: %w", ErrBillingAuthorityRepository, err)
	}
	if operation.AccountID != account.AccountID || operation.AccountScopeKey != account.AccountScopeKey ||
		child.AccountID != account.AccountID || child.AccountScopeKey != account.AccountScopeKey ||
		child.UsageOperationID != ids.usageOperationID ||
		child.MicroleaseID != ids.microleaseID ||
		child.DebitAuthorizationID != cmd.DebitAuthorizationID {
		return sqlcgen.UsageOperation{}, sqlcgen.MicroleaseChildDebit{}, sqlcgen.SpendingMicrolease{}, sqlcgen.AccountBalance{}, fmt.Errorf("%w: terminal_lineage_mismatch", billingauthority.ErrConflict)
	}
	if child.State != "terminal_pending" {
		return sqlcgen.UsageOperation{}, sqlcgen.MicroleaseChildDebit{}, sqlcgen.SpendingMicrolease{}, sqlcgen.AccountBalance{}, fmt.Errorf("%w: child_debit_already_terminal", billingauthority.ErrConflict)
	}
	lease, err := q.LockSpendingMicrolease(ctx, ids.microleaseID)
	if err != nil {
		return sqlcgen.UsageOperation{}, sqlcgen.MicroleaseChildDebit{}, sqlcgen.SpendingMicrolease{}, sqlcgen.AccountBalance{}, fmt.Errorf("%w: lock terminal microlease: %w", ErrBillingAuthorityRepository, err)
	}
	balance, err := q.LockAccountBalanceByAccountID(ctx, account.AccountID)
	if err != nil {
		return sqlcgen.UsageOperation{}, sqlcgen.MicroleaseChildDebit{}, sqlcgen.SpendingMicrolease{}, sqlcgen.AccountBalance{}, fmt.Errorf("%w: lock terminal balance: %w", ErrBillingAuthorityRepository, err)
	}
	return operation, child, lease, balance, nil
}

func validateTerminalAmounts(cmd billingauthority.UsageTerminalCommand, child sqlcgen.MicroleaseChildDebit, balance sqlcgen.AccountBalance) error {
	terminalTotal := cmd.ChargedUSDAtoms + cmd.ReleasedUSDAtoms + cmd.WriteOffUSDAtoms
	if terminalTotal > child.ChildCapUsdAtoms {
		return fmt.Errorf("%w: terminal_total_exceeds_child_cap", billingauthority.ErrRejected)
	}
	if balance.ReservedUsdAtoms < terminalTotal {
		return fmt.Errorf("%w: terminal_reserved_exposure_missing", billingauthority.ErrRejected)
	}
	if balance.SettledUsdAtoms < cmd.ChargedUSDAtoms {
		return fmt.Errorf("%w: terminal_settled_balance_insufficient", billingauthority.ErrRejected)
	}
	return nil
}

func terminalSettlementCommand(cmd billingauthority.UsageTerminalCommand, account sqlcgen.BillingAccount, child sqlcgen.MicroleaseChildDebit, now time.Time) TerminalSettlementCommand {
	return TerminalSettlementCommand{
		MicroleaseChildDebitID:     cmd.MicroleaseChildDebitID,
		MicroleaseID:               cmd.MicroleaseID,
		DebitAuthorizationID:       cmd.DebitAuthorizationID,
		AccountID:                  uuidString(account.AccountID),
		AccountScopeKey:            account.AccountScopeKey,
		ProxyAllocatorOwnerID:      child.ProxyAllocatorOwnerID,
		MicroleaseGeneration:       child.MicroleaseGeneration,
		ChildSequence:              child.ChildSequence,
		ChildCapUSDAtoms:           child.ChildCapUsdAtoms,
		ChargedUSDAtoms:            cmd.ChargedUSDAtoms,
		ReleasedUSDAtoms:           cmd.ReleasedUSDAtoms,
		WriteOffUSDAtoms:           cmd.WriteOffUSDAtoms,
		RequestBasisFingerprint:    child.RequestBasisFingerprint,
		TerminalBasisFingerprint:   cmd.TerminalFingerprint,
		PricingSnapshotID:          cmd.Pricing.ID,
		PricingSnapshotFingerprint: cmd.Pricing.Fingerprint,
		TerminalKind:               cmd.TerminalKind,
		TerminalState:              usageTerminalState(cmd.TerminalKind),
		TerminalAt:                 now,
		SettledAt:                  now,
		SafeMetadata:               cmd.Metadata,
	}
}

func usageTerminalState(kind string) string {
	switch kind {
	case "write_off":
		return "written_off"
	default:
		return "finalized"
	}
}

func usageSnapshot(operation sqlcgen.UsageOperation, resultCode, idempotencyKey, requestFingerprint, storedOutcomeID string) billingauthority.UsageOperationSnapshot {
	return billingauthority.UsageOperationSnapshot{
		UsageOperationID:   uuidString(operation.UsageOperationID),
		State:              operation.State,
		ResultCode:         resultCode,
		IdempotencyKey:     idempotencyKey,
		RequestFingerprint: requestFingerprint,
		StoredOutcomeID:    storedOutcomeID,
	}
}

func mapTerminalChildDebitRecord(row sqlcgen.MicroleaseChildDebit) TerminalChildDebitRecord {
	return TerminalChildDebitRecord{
		MicroleaseChildDebitID:       uuidString(row.MicroleaseChildDebitID),
		MicroleaseID:                 uuidString(row.MicroleaseID),
		DebitAuthorizationID:         row.DebitAuthorizationID,
		UsageOperationID:             optionalUUIDString(row.UsageOperationID),
		AccountID:                    uuidString(row.AccountID),
		AccountScopeKey:              row.AccountScopeKey,
		ProxyAllocatorOwnerID:        row.ProxyAllocatorOwnerID,
		MicroleaseGeneration:         row.MicroleaseGeneration,
		ChildSequence:                row.ChildSequence,
		ChildCapUSDAtoms:             row.ChildCapUsdAtoms,
		RequestBasisFingerprint:      row.RequestBasisFingerprint,
		PricingSnapshotID:            row.PricingSnapshotID,
		PricingSnapshotFingerprint:   row.PricingSnapshotFingerprint,
		QualifiedInferenceEvidenceID: optionalUUIDString(row.QualifiedInferenceEvidenceID),
	}
}

func resolveAccountScopeKey(req billingauthority.AccountResolveRequest) string {
	if req.RepresentedSubjectID != "" {
		return "user:" + req.RepresentedSubjectID
	}
	return req.RepresentedAccountID
}

func accountResolveFailureClass(accountState, importState string) string {
	switch {
	case accountState == "suspended":
		return "account_suspended"
	case accountState == "manual_review":
		return "manual_review"
	case importState == "missing":
		return "import_required"
	case importState == "mismatch":
		return "legacy_import_mismatch"
	case importState == "pending":
		return "import_pending"
	default:
		return ""
	}
}

func runtimeStateForBalance(accountState, importState string) string {
	if accountState != "active" || importState == "mismatch" {
		return "fail_closed"
	}
	if importState == "pending" || importState == "missing" {
		return "not_ready"
	}
	return "ready"
}

func reconciliationSafeLineage(row sqlcgen.ReconciliationCase) string {
	switch {
	case row.UsageOperationID.Valid:
		return uuidString(row.UsageOperationID)
	case row.MicroleaseChildDebitID.Valid:
		return uuidString(row.MicroleaseChildDebitID)
	case row.MicroleaseID.Valid:
		return uuidString(row.MicroleaseID)
	case row.BillingEventInboxID.Valid:
		return uuidString(row.BillingEventInboxID)
	case row.LegacyBalanceImportID.Valid:
		return uuidString(row.LegacyBalanceImportID)
	default:
		return ""
	}
}

func translateAuthorityWriteError(operation string, err error) error {
	if isUniqueViolation(err) {
		return fmt.Errorf("%w: %s: %w", billingauthority.ErrConflict, operation, err)
	}
	return fmt.Errorf("%w: %s: %w", ErrBillingAuthorityRepository, operation, err)
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

func newPGUUID() pgtype.UUID {
	id := uuid.New()
	return pgtype.UUID{Bytes: id, Valid: true}
}

func mustUUIDOrZero(raw string) pgtype.UUID {
	id, err := uuidValue(raw)
	if err != nil {
		return pgtype.UUID{}
	}
	return id
}

func optionalString(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func optionalUUIDString(id pgtype.UUID) string {
	if !id.Valid {
		return ""
	}
	return uuidString(id)
}
