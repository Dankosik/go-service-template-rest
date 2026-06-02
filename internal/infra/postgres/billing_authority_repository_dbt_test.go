package postgres

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Dankosik/billing-service/internal/app/billingauthority"
	"github.com/Dankosik/billing-service/internal/infra/postgres/sqlcgen"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
)

func TestHandleExistingIdempotencyReplayAndConflictPaths(t *testing.T) {
	t.Parallel()

	repo := &BillingAuthorityRepository{}
	accountID := mustPGUUID(t, testAccountUUID)
	now := fixedPostgresTime()

	missing := sqlcgen.New(&queuedPostgresDBTX{rows: []queuedPostgresRow{{err: pgx.ErrNoRows}}})
	snapshot, handled, err := repo.handleExistingIdempotency(context.Background(), missing, accountID, "reserve", "idem-1", "fingerprint-1", testUsageUUID, now)
	if err != nil || handled || snapshot.UsageOperationID != "" {
		t.Fatalf("missing idempotency = snapshot:%+v handled:%v err:%v, want unhandled nil", snapshot, handled, err)
	}

	conflictDB := &queuedPostgresDBTX{rows: []queuedPostgresRow{
		{scan: scanIdempotencyRecord("old-fingerprint", "", "started")},
		{scan: scanIdempotencyRecord("old-fingerprint", "", "conflict")},
	}}
	_, handled, err = repo.handleExistingIdempotency(context.Background(), sqlcgen.New(conflictDB), accountID, "reserve", "idem-1", "new-fingerprint", testUsageUUID, now)
	if !handled || !errors.Is(err, billingauthority.ErrConflict) || conflictDB.calls != 2 {
		t.Fatalf("fingerprint conflict handled=%v calls=%d err=%v, want conflict with persisted mark", handled, conflictDB.calls, err)
	}

	inProgress := sqlcgen.New(&queuedPostgresDBTX{rows: []queuedPostgresRow{
		{scan: scanIdempotencyRecord("fingerprint-1", "", "started")},
	}})
	_, handled, err = repo.handleExistingIdempotency(context.Background(), inProgress, accountID, "reserve", "idem-1", "fingerprint-1", testUsageUUID, now)
	if !handled || !errors.Is(err, billingauthority.ErrConflict) {
		t.Fatalf("in-progress replay handled=%v err=%v, want conflict", handled, err)
	}

	replayDB := &queuedPostgresDBTX{rows: []queuedPostgresRow{
		{scan: scanIdempotencyRecord("fingerprint-1", testOutcomeUUID, "committed")},
		{scan: scanOperationOutcome(testOutcomeUUID, testUsageUUID)},
		{scan: scanUsageOperation()},
	}}
	snapshot, handled, err = repo.handleExistingIdempotency(context.Background(), sqlcgen.New(replayDB), accountID, "reserve", "idem-1", "fingerprint-1", testUsageUUID, now)
	if err != nil || !handled {
		t.Fatalf("stored replay handled=%v err=%v, want stored outcome replay", handled, err)
	}
	if snapshot.ResultCode != "duplicate_stored_outcome" || snapshot.UsageOperationID != testUsageUUID || snapshot.StoredOutcomeID != testOutcomeUUID {
		t.Fatalf("stored replay snapshot = %+v, want duplicate stored usage operation", snapshot)
	}
}

func TestLockTerminalRowsEnforcesLineageAndStateBeforeSettlement(t *testing.T) {
	t.Parallel()

	cmd := validPostgresTerminalCommand()
	ids, err := terminalCommandIDsFromAuthority(cmd)
	if err != nil {
		t.Fatalf("terminalCommandIDsFromAuthority() error = %v", err)
	}
	account := sqlcgen.BillingAccount{
		AccountID:       mustPGUUID(t, testAccountUUID),
		AccountScopeKey: "user:1",
		State:           "active",
	}
	success := &queuedPostgresDBTX{rows: []queuedPostgresRow{
		{scan: scanUsageOperation()},
		{scan: scanChildDebit(testChildDebitUUID, testUsageUUID, testMicroleaseUUID, "debit-auth-1", "terminal_pending")},
		{scan: scanSpendingMicrolease(testMicroleaseUUID)},
		{scan: scanAccountBalance(testAccountUUID, 200, 100)},
	}}
	operation, child, lease, balance, err := lockTerminalRows(context.Background(), sqlcgen.New(success), ids, account, cmd)
	if err != nil {
		t.Fatalf("lockTerminalRows(valid) error = %v", err)
	}
	if uuidString(operation.UsageOperationID) != testUsageUUID || uuidString(child.MicroleaseChildDebitID) != testChildDebitUUID ||
		uuidString(lease.MicroleaseID) != testMicroleaseUUID || balance.ReservedUsdAtoms != 100 {
		t.Fatalf("locked rows = operation:%+v child:%+v lease:%+v balance:%+v", operation, child, lease, balance)
	}

	missingChild := &queuedPostgresDBTX{rows: []queuedPostgresRow{
		{scan: scanUsageOperation()},
		{err: pgx.ErrNoRows},
	}}
	_, _, _, _, err = lockTerminalRows(context.Background(), sqlcgen.New(missingChild), ids, account, cmd)
	if !errors.Is(err, billingauthority.ErrRejected) {
		t.Fatalf("lockTerminalRows(missing child) error = %v, want rejected", err)
	}

	lineageMismatch := &queuedPostgresDBTX{rows: []queuedPostgresRow{
		{scan: scanUsageOperation()},
		{scan: scanChildDebit(testChildDebitUUID, testUsageUUID, testMicroleaseUUID, "other-auth", "terminal_pending")},
	}}
	_, _, _, _, err = lockTerminalRows(context.Background(), sqlcgen.New(lineageMismatch), ids, account, cmd)
	if !errors.Is(err, billingauthority.ErrConflict) {
		t.Fatalf("lockTerminalRows(lineage mismatch) error = %v, want conflict", err)
	}

	alreadyTerminal := &queuedPostgresDBTX{rows: []queuedPostgresRow{
		{scan: scanUsageOperation()},
		{scan: scanChildDebit(testChildDebitUUID, testUsageUUID, testMicroleaseUUID, "debit-auth-1", "finalized")},
	}}
	_, _, _, _, err = lockTerminalRows(context.Background(), sqlcgen.New(alreadyTerminal), ids, account, cmd)
	if !errors.Is(err, billingauthority.ErrConflict) {
		t.Fatalf("lockTerminalRows(already terminal) error = %v, want conflict", err)
	}
}

func TestTerminalSettlementCommandAndOptionalHelpers(t *testing.T) {
	t.Parallel()

	account := sqlcgen.BillingAccount{AccountID: mustPGUUID(t, testAccountUUID), AccountScopeKey: "user:1"}
	child := sqlcgen.MicroleaseChildDebit{
		ProxyAllocatorOwnerID:      "proxy-owner-1",
		MicroleaseGeneration:       7,
		ChildSequence:              3,
		ChildCapUsdAtoms:           100,
		RequestBasisFingerprint:    "request-fingerprint",
		PricingSnapshotID:          "old-pricing",
		PricingSnapshotFingerprint: "old-pricing-fingerprint",
	}
	cmd := validPostgresTerminalCommand()
	cmd.TerminalKind = "write_off"
	settlement := terminalSettlementCommand(cmd, account, child, fixedPostgresTime())
	if settlement.TerminalState != "written_off" || settlement.AccountID != testAccountUUID ||
		settlement.ProxyAllocatorOwnerID != "proxy-owner-1" || settlement.PricingSnapshotID != cmd.Pricing.ID {
		t.Fatalf("terminalSettlementCommand() = %+v, want lineage and pricing from command/child", settlement)
	}
	if id := newPGUUID(); !id.Valid {
		t.Fatal("newPGUUID() returned invalid UUID")
	}
	if optionalString("") != nil || optionalString("value") == nil {
		t.Fatal("optionalString did not preserve empty/non-empty semantics")
	}
	if got := optionalUUIDString(pgtype.UUID{}); got != "" {
		t.Fatalf("optionalUUIDString(invalid) = %q, want empty", got)
	}
}

type queuedPostgresDBTX struct {
	rows  []queuedPostgresRow
	calls int
}

type queuedPostgresRow struct {
	scan func(...any) error
	err  error
}

func (db *queuedPostgresDBTX) Exec(context.Context, string, ...any) (pgconn.CommandTag, error) {
	db.calls++
	return pgconn.CommandTag{}, nil
}

func (db *queuedPostgresDBTX) Query(context.Context, string, ...any) (pgx.Rows, error) {
	db.calls++
	return nil, errors.New("queued query rows are not configured")
}

func (db *queuedPostgresDBTX) QueryRow(context.Context, string, ...any) pgx.Row {
	db.calls++
	if len(db.rows) == 0 {
		return queuedPostgresScanRow{err: errors.New("unexpected query row")}
	}
	row := db.rows[0]
	db.rows = db.rows[1:]
	return queuedPostgresScanRow(row)
}

type queuedPostgresScanRow queuedPostgresRow

func (row queuedPostgresScanRow) Scan(dest ...any) error {
	if row.err != nil {
		return row.err
	}
	if row.scan == nil {
		return nil
	}
	return row.scan(dest...)
}

func scanIdempotencyRecord(requestFingerprint, storedOutcomeRaw, state string) func(...any) error {
	return func(dest ...any) error {
		setPGUUID(dest[0], testIdempotencyUUID)
		setPGUUID(dest[1], testAccountUUID)
		setStringDest(dest[2], "reserve")
		setStringDest(dest[3], "idem-1")
		setStringDest(dest[4], requestFingerprint)
		setStringDest(dest[5], state)
		if storedOutcomeRaw != "" {
			setPGUUID(dest[6], storedOutcomeRaw)
		}
		setStringDest(dest[8], "hot_replay")
		setPGTime(dest[9], fixedPostgresTime())
		setPGTime(dest[10], fixedPostgresTime())
		setPGTime(dest[11], fixedPostgresTime())
		setPGTime(dest[12], fixedPostgresTime().Add(time.Hour))
		return nil
	}
}

func scanOperationOutcome(outcomeIDRaw, usageOperationIDRaw string) func(...any) error {
	return func(dest ...any) error {
		setPGUUID(dest[0], outcomeIDRaw)
		setPGUUID(dest[1], testIdempotencyUUID)
		setPGUUID(dest[2], testAccountUUID)
		setStringDest(dest[3], "reserve")
		setStringDest(dest[4], "accepted")
		setStringDest(dest[5], "usage_operation")
		setStringDest(dest[6], usageOperationIDRaw)
		setPGUUID(dest[7], testLedgerUUID)
		setPGUUID(dest[8], testSettlementUUID)
		setBytesDest(dest[11], []byte(`{"result":"accepted"}`))
		setPGTime(dest[12], fixedPostgresTime())
		return nil
	}
}

func scanUsageOperation() func(...any) error {
	return func(dest ...any) error {
		setPGUUID(dest[0], testUsageUUID)
		setPGUUID(dest[1], testAccountUUID)
		setStringDest(dest[2], "user:1")
		setStringDest(dest[3], "reserved")
		setStringDest(dest[4], "reserve")
		setStringDest(dest[5], "debit-auth-1")
		setStringDest(dest[7], "request-fingerprint")
		setStringDest(dest[9], "pricing-1")
		setStringDest(dest[10], "pricing-fingerprint")
		setPGTime(dest[11], fixedPostgresTime().Add(time.Minute))
		setStringDest(dest[12], "fee-policy")
		setStringDest(dest[13], "reserve-policy")
		setPGUUID(dest[14], testInboxUUID)
		setPGUUID(dest[15], testTerminalUUID)
		setPGUUID(dest[16], testSettlementUUID)
		setPGTime(dest[17], fixedPostgresTime())
		setPGTime(dest[18], fixedPostgresTime())
		setPGTime(dest[19], fixedPostgresTime())
		setPGTime(dest[20], fixedPostgresTime())
		return nil
	}
}

func scanChildDebit(childDebitIDRaw, usageOperationIDRaw, microleaseIDRaw, debitAuthorizationID, state string) func(...any) error {
	return func(dest ...any) error {
		setPGUUID(dest[0], childDebitIDRaw)
		setPGUUID(dest[1], microleaseIDRaw)
		setStringDest(dest[2], debitAuthorizationID)
		setPGUUID(dest[3], usageOperationIDRaw)
		setPGUUID(dest[4], testAccountUUID)
		setStringDest(dest[5], "user:1")
		setStringDest(dest[6], "proxy-owner-1")
		setInt64Dest(dest[7], 1)
		setInt64Dest(dest[8], 1)
		setInt64Dest(dest[9], 100)
		setStringDest(dest[13], "request-fingerprint")
		setStringDest(dest[15], "pricing-1")
		setStringDest(dest[16], "pricing-fingerprint")
		setStringDest(dest[17], "finalize")
		setStringDest(dest[18], state)
		setPGUUID(dest[19], testInboxUUID)
		setPGUUID(dest[21], testInboxUUID)
		setPGUUID(dest[22], testLedgerUUID)
		setPGUUID(dest[23], testSettlementUUID)
		setBytesDest(dest[24], []byte(`{"surface":"terminal"}`))
		setPGTime(dest[25], fixedPostgresTime())
		setPGTime(dest[26], fixedPostgresTime())
		setPGTime(dest[27], fixedPostgresTime())
		setPGTime(dest[28], fixedPostgresTime())
		return nil
	}
}

func scanSpendingMicrolease(microleaseIDRaw string) func(...any) error {
	return func(dest ...any) error {
		setPGUUID(dest[0], microleaseIDRaw)
		setPGUUID(dest[1], testAccountUUID)
		setStringDest(dest[2], "user:1")
		setStringDest(dest[3], "proxy-owner-1")
		setInt64Dest(dest[4], 1)
		setStringDest(dest[5], "fence-1")
		setStringDest(dest[6], "active")
		setInt64Dest(dest[7], 100)
		setInt64Dest(dest[8], 100)
		setStringDest(dest[13], "pricing-1")
		setStringDest(dest[14], "pricing-fingerprint")
		setPGTime(dest[16], fixedPostgresTime())
		setPGTime(dest[21], fixedPostgresTime())
		setPGTime(dest[22], fixedPostgresTime().Add(time.Hour))
		setPGTime(dest[23], fixedPostgresTime().Add(2*time.Hour))
		setPGUUID(dest[27], testIdempotencyUUID)
		setPGUUID(dest[28], testOutcomeUUID)
		setBytesDest(dest[29], []byte(`{"surface":"microlease"}`))
		setPGTime(dest[30], fixedPostgresTime())
		setPGTime(dest[31], fixedPostgresTime())
		return nil
	}
}

func scanAccountBalance(accountIDRaw string, settled, reserved int64) func(...any) error {
	return func(dest ...any) error {
		setPGUUID(dest[0], accountIDRaw)
		setStringDest(dest[1], "user:1")
		setStringDest(dest[2], "USD")
		setInt64Dest(dest[3], settled)
		setInt64Dest(dest[4], reserved)
		setInt64Dest(dest[5], settled-reserved)
		setInt64Dest(dest[6], 0)
		setInt64Dest(dest[7], 1)
		setPGUUID(dest[8], testLedgerUUID)
		setPGTime(dest[9], fixedPostgresTime())
		return nil
	}
}

func setStringDest(dest any, value string) {
	target, ok := dest.(*string)
	if !ok {
		panic("queued scan destination is not *string")
	}
	*target = value
}

func setInt64Dest(dest any, value int64) {
	target, ok := dest.(*int64)
	if !ok {
		panic("queued scan destination is not *int64")
	}
	*target = value
}

func setBytesDest(dest any, value []byte) {
	target, ok := dest.(*[]byte)
	if !ok {
		panic("queued scan destination is not *[]byte")
	}
	*target = value
}

func setPGUUID(dest any, raw string) {
	value, err := uuidValue(raw)
	if err != nil {
		panic(err)
	}
	target, ok := dest.(*pgtype.UUID)
	if !ok {
		panic("queued scan destination is not *pgtype.UUID")
	}
	*target = value
}

func setPGTime(dest any, value time.Time) {
	target, ok := dest.(*pgtype.Timestamptz)
	if !ok {
		panic("queued scan destination is not *pgtype.Timestamptz")
	}
	*target = timestamptzValue(value)
}
