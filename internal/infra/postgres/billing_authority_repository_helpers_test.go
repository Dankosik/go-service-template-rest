package postgres

import (
	"errors"
	"testing"
	"time"

	"github.com/Dankosik/billing-service/internal/app/billingauthority"
	"github.com/Dankosik/billing-service/internal/infra/postgres/sqlcgen"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
)

const (
	testAccountUUID     = "11111111-1111-1111-1111-111111111111"
	testMicroleaseUUID  = "22222222-2222-2222-2222-222222222222"
	testChildDebitUUID  = "33333333-3333-3333-3333-333333333333"
	testUsageUUID       = "44444444-4444-4444-4444-444444444444"
	testIdempotencyUUID = "55555555-5555-5555-5555-555555555555"
	testOutcomeUUID     = "66666666-6666-6666-6666-666666666666"
	testLedgerUUID      = "77777777-7777-7777-7777-777777777777"
	testOutboxUUID      = "88888888-8888-8888-8888-888888888888"
	testTerminalUUID    = "99999999-9999-9999-9999-999999999999"
	testSettlementUUID  = "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"
	testCheckpointUUID  = "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"
	testInboxUUID       = "cccccccc-cccc-cccc-cccc-cccccccccccc"
	testImportUUID      = "dddddddd-dddd-dddd-dddd-dddddddddddd"
)

func TestRepositoryConstructorsRejectMissingPools(t *testing.T) {
	t.Parallel()

	if _, err := NewBillingAuthorityRepository(nil); !errors.Is(err, ErrBillingAuthorityRepository) {
		t.Fatalf("NewBillingAuthorityRepository(nil) error = %v, want repository error", err)
	}
	if _, err := NewBillingAuthorityRepositoryFromPGXPool(nil); !errors.Is(err, ErrBillingAuthorityRepository) {
		t.Fatalf("NewBillingAuthorityRepositoryFromPGXPool(nil) error = %v, want repository error", err)
	}
	if err := (*BillingAuthorityRepository)(nil).require(); !errors.Is(err, billingauthority.ErrNotReady) {
		t.Fatalf("nil authority require() error = %v, want ErrNotReady", err)
	}
	if _, err := NewMicroleaseRepository(nil); !errors.Is(err, ErrMicroleaseRepository) {
		t.Fatalf("NewMicroleaseRepository(nil) error = %v, want repository error", err)
	}
	if _, err := NewMicroleaseRepositoryFromPGXPool(nil); !errors.Is(err, ErrMicroleaseRepository) {
		t.Fatalf("NewMicroleaseRepositoryFromPGXPool(nil) error = %v, want repository error", err)
	}
	if err := (*MicroleaseRepository)(nil).require(); !errors.Is(err, ErrMicroleaseRepository) {
		t.Fatalf("nil microlease require() error = %v, want repository error", err)
	}
}

func TestUUIDHelpersAndCommandIDParsing(t *testing.T) {
	t.Parallel()

	parsed, err := uuidValue(testAccountUUID)
	if err != nil {
		t.Fatalf("uuidValue() error = %v", err)
	}
	if got := uuidString(parsed); got != testAccountUUID {
		t.Fatalf("uuidString() = %q, want %q", got, testAccountUUID)
	}
	if _, err := uuidValue("not-a-uuid"); err == nil {
		t.Fatal("uuidValue(invalid) error = nil, want error")
	}
	empty, err := optionalUUID("")
	if err != nil {
		t.Fatalf("optionalUUID(empty) error = %v", err)
	}
	if empty.Valid {
		t.Fatalf("optionalUUID(empty) valid = true, want false")
	}
	present, err := optionalUUID(testUsageUUID)
	if err != nil {
		t.Fatalf("optionalUUID(valid) error = %v", err)
	}
	if got := optionalUUIDString(present); got != testUsageUUID {
		t.Fatalf("optionalUUIDString() = %q, want %q", got, testUsageUUID)
	}
	if got := mustUUIDOrZero("bad"); got.Valid {
		t.Fatalf("mustUUIDOrZero(invalid) = %+v, want invalid UUID", got)
	}

	reserveIDs, err := reserveCommandIDs(validPostgresReserveCommand())
	if err != nil {
		t.Fatalf("reserveCommandIDs() error = %v", err)
	}
	if uuidString(reserveIDs.usageOperationID) != testUsageUUID || uuidString(reserveIDs.microleaseID) != testMicroleaseUUID || uuidString(reserveIDs.childDebitID) != testChildDebitUUID {
		t.Fatalf("reserve IDs = %+v, want usage/microlease/child IDs", reserveIDs)
	}

	terminalIDs, err := terminalCommandIDsFromAuthority(validPostgresTerminalCommand())
	if err != nil {
		t.Fatalf("terminalCommandIDsFromAuthority() error = %v", err)
	}
	if uuidString(terminalIDs.terminalOutcomeID) != testTerminalUUID || terminalIDs.qualifiedInferenceEvidenceID.Valid {
		t.Fatalf("terminal IDs = %+v, want terminal and no optional evidence", terminalIDs)
	}

	terminal := validPostgresTerminalCommand()
	terminal.QualifiedInferenceEvidenceID = testInboxUUID
	terminalIDs, err = terminalCommandIDsFromAuthority(terminal)
	if err != nil {
		t.Fatalf("terminalCommandIDsFromAuthority(evidence) error = %v", err)
	}
	if uuidString(terminalIDs.qualifiedInferenceEvidenceID) != testInboxUUID {
		t.Fatalf("qualified evidence = %s, want %s", uuidString(terminalIDs.qualifiedInferenceEvidenceID), testInboxUUID)
	}

	if _, _, _, _, _, _, err := parseSixUUIDs(testMicroleaseUUID, testAccountUUID, testIdempotencyUUID, testOutcomeUUID, testLedgerUUID, "bad"); err == nil {
		t.Fatal("parseSixUUIDs(invalid) error = nil, want error")
	}
}

func TestMicroleaseCommandIDParsingAndSafePayloads(t *testing.T) {
	t.Parallel()

	issueMicroleaseID, _, _, _, _, _, err := issueCommandUUIDs(IssueMicroleaseCommand{
		MicroleaseID:        testMicroleaseUUID,
		AccountID:           testAccountUUID,
		IdempotencyRecordID: testIdempotencyUUID,
		StoredOutcomeID:     testOutcomeUUID,
		LedgerEntryID:       testLedgerUUID,
		OutboxID:            testOutboxUUID,
	})
	if err != nil {
		t.Fatalf("issueCommandUUIDs() error = %v", err)
	}
	if uuidString(issueMicroleaseID) != testMicroleaseUUID {
		t.Fatalf("issue microlease id = %s, want %s", uuidString(issueMicroleaseID), testMicroleaseUUID)
	}
	closeMicroleaseID, _, _, _, _, _, err := closeCommandUUIDs(CloseMicroleaseCommand{
		MicroleaseID:        testMicroleaseUUID,
		AccountID:           testAccountUUID,
		IdempotencyRecordID: testIdempotencyUUID,
		StoredOutcomeID:     testOutcomeUUID,
		LedgerEntryID:       testLedgerUUID,
		OutboxID:            testOutboxUUID,
	})
	if err != nil {
		t.Fatalf("closeCommandUUIDs() error = %v", err)
	}
	if uuidString(closeMicroleaseID) != testMicroleaseUUID {
		t.Fatalf("close microlease id = %s, want %s", uuidString(closeMicroleaseID), testMicroleaseUUID)
	}

	terminalIDs, err := terminalCommandUUIDs(validPostgresTerminalSettlementCommand())
	if err != nil {
		t.Fatalf("terminalCommandUUIDs() error = %v", err)
	}
	if uuidString(terminalIDs.settlementEffectID) != testSettlementUUID {
		t.Fatalf("settlement effect = %s, want %s", uuidString(terminalIDs.settlementEffectID), testSettlementUUID)
	}
	checkpointIDs, err := checkpointCommandUUIDs(CheckpointCommand{
		InboxID:         testInboxUUID,
		CheckpointID:    testCheckpointUUID,
		MicroleaseID:    testMicroleaseUUID,
		AccountID:       testAccountUUID,
		AccountScopeKey: "user:1",
	})
	if err != nil {
		t.Fatalf("checkpointCommandUUIDs() error = %v", err)
	}
	if uuidString(checkpointIDs.checkpointID) != testCheckpointUUID {
		t.Fatalf("checkpoint ID = %s, want %s", uuidString(checkpointIDs.checkpointID), testCheckpointUUID)
	}

	payload, err := marshalSafeObject(map[string]string{"surface": "worker"})
	if err != nil {
		t.Fatalf("marshalSafeObject() error = %v", err)
	}
	if string(payload) != `{"surface":"worker"}` {
		t.Fatalf("marshalSafeObject() = %s, want safe metadata JSON", payload)
	}
	if got := string(safePayload(map[string]any{"usage_operation_id": "usage-1", "bad": func() {}})); got != `{}` {
		t.Fatalf("safePayload(non-marshalable) = %s, want empty object", got)
	}
}

func TestReserveLeaseValidationFailsClosed(t *testing.T) {
	t.Parallel()

	now := fixedPostgresTime()
	accountID := mustPGUUID(t, testAccountUUID)
	account := sqlcgen.BillingAccount{AccountID: accountID, AccountScopeKey: "user:1", State: "active"}
	lease := sqlcgen.SpendingMicrolease{
		AccountID:                 accountID,
		AccountScopeKey:           "user:1",
		ProxyAllocatorOwnerID:     "proxy-owner-1",
		MicroleaseGeneration:      3,
		LeaseFence:                "fence-1",
		State:                     "active",
		DebitCutoffAt:             timestamptzValue(now.Add(time.Minute)),
		AvailableChildCapUsdAtoms: 100,
	}
	cmd := validPostgresReserveCommand()
	cmd.MicroleaseGeneration = 3
	cmd.ChildCapUSDAtoms = 50

	if err := validateReserveLease(cmd, account, lease, now); err != nil {
		t.Fatalf("validateReserveLease(valid) error = %v", err)
	}

	lease.AccountScopeKey = "user:other"
	if err := validateReserveLease(cmd, account, lease, now); !errors.Is(err, billingauthority.ErrConflict) {
		t.Fatalf("validateReserveLease(account mismatch) error = %v, want conflict", err)
	}
	lease.AccountScopeKey = "user:1"
	lease.ProxyAllocatorOwnerID = "other-proxy"
	if err := validateReserveLease(cmd, account, lease, now); !errors.Is(err, billingauthority.ErrConflict) {
		t.Fatalf("validateReserveLease(lineage mismatch) error = %v, want conflict", err)
	}
	lease.ProxyAllocatorOwnerID = "proxy-owner-1"
	lease.State = "closed"
	if err := validateReserveLease(cmd, account, lease, now); !errors.Is(err, billingauthority.ErrRejected) {
		t.Fatalf("validateReserveLease(inactive) error = %v, want rejected", err)
	}
	lease.State = "active"
	lease.DebitCutoffAt = timestamptzValue(now)
	if err := validateReserveLease(cmd, account, lease, now); !errors.Is(err, billingauthority.ErrRejected) {
		t.Fatalf("validateReserveLease(cutoff) error = %v, want rejected", err)
	}
	lease.DebitCutoffAt = timestamptzValue(now.Add(time.Minute))
	lease.AvailableChildCapUsdAtoms = 49
	if err := validateReserveLease(cmd, account, lease, now); !errors.Is(err, billingauthority.ErrRejected) {
		t.Fatalf("validateReserveLease(cap exhausted) error = %v, want rejected", err)
	}
}

func TestTerminalAmountValidationAndLedgerParamsConserveExposure(t *testing.T) {
	t.Parallel()

	accountID := mustPGUUID(t, testAccountUUID)
	child := sqlcgen.MicroleaseChildDebit{ChildCapUsdAtoms: 100}
	balance := sqlcgen.AccountBalance{AccountID: accountID, SettledUsdAtoms: 200, ReservedUsdAtoms: 100, PendingUsdAtoms: 7, Version: 4}
	cmd := validPostgresTerminalCommand()
	cmd.ChargedUSDAtoms = 40
	cmd.ReleasedUSDAtoms = 60

	if err := validateTerminalAmounts(cmd, child, balance); err != nil {
		t.Fatalf("validateTerminalAmounts(valid) error = %v", err)
	}
	tooLarge := cmd
	tooLarge.ChargedUSDAtoms = 101
	tooLarge.ReleasedUSDAtoms = 0
	if err := validateTerminalAmounts(tooLarge, child, balance); !errors.Is(err, billingauthority.ErrRejected) {
		t.Fatalf("validateTerminalAmounts(over child cap) error = %v, want rejected", err)
	}
	insufficientReserved := cmd
	insufficientReserved.ReleasedUSDAtoms = 61
	if err := validateTerminalAmounts(insufficientReserved, child, sqlcgen.AccountBalance{SettledUsdAtoms: 200, ReservedUsdAtoms: 100}); !errors.Is(err, billingauthority.ErrRejected) {
		t.Fatalf("validateTerminalAmounts(over reserved) error = %v, want rejected", err)
	}
	insufficientSettled := cmd
	insufficientSettled.ChargedUSDAtoms = 40
	if err := validateTerminalAmounts(insufficientSettled, child, sqlcgen.AccountBalance{SettledUsdAtoms: 39, ReservedUsdAtoms: 100}); !errors.Is(err, billingauthority.ErrRejected) {
		t.Fatalf("validateTerminalAmounts(insufficient settled) error = %v, want rejected", err)
	}

	ids := terminalCommandIDs{accountID: accountID, ledgerID: mustPGUUID(t, testLedgerUUID), settlementEffectID: mustPGUUID(t, testSettlementUUID)}
	params := terminalLedgerParams(validPostgresTerminalSettlementCommand(), balance, ids, []byte(`{"surface":"terminal"}`), fixedPostgresTime())
	if params.EffectType != "microlease_child_charge" || params.SettledDeltaUsdAtoms != -40 || params.ReservedDeltaUsdAtoms != -100 || params.AvailableAfterUsdAtoms != 160 {
		t.Fatalf("terminalLedgerParams(charge) = %+v, want charge releasing full reserved exposure", params)
	}
	release := validPostgresTerminalSettlementCommand()
	release.ChargedUSDAtoms = 0
	release.WriteOffUSDAtoms = 0
	params = terminalLedgerParams(release, balance, ids, nil, fixedPostgresTime())
	if params.EffectType != "microlease_child_release" || params.AmountUsdAtoms != -60 || params.SettledDeltaUsdAtoms != 0 {
		t.Fatalf("terminalLedgerParams(release) = %+v, want release effect", params)
	}
	writeOff := validPostgresTerminalSettlementCommand()
	writeOff.ChargedUSDAtoms = 0
	writeOff.ReleasedUSDAtoms = 0
	writeOff.WriteOffUSDAtoms = 60
	params = terminalLedgerParams(writeOff, balance, ids, nil, fixedPostgresTime())
	if params.EffectType != "microlease_write_off" || params.ReservedDeltaUsdAtoms != -60 {
		t.Fatalf("terminalLedgerParams(write off) = %+v, want write-off release", params)
	}
}

func TestAuthorityMappingHelpersPreserveReadbackAndFailClosedClasses(t *testing.T) {
	t.Parallel()

	accountID := mustPGUUID(t, testAccountUUID)
	usageID := mustPGUUID(t, testUsageUUID)
	childID := mustPGUUID(t, testChildDebitUUID)
	microleaseID := mustPGUUID(t, testMicroleaseUUID)
	inboxID := mustPGUUID(t, testInboxUUID)
	importID := mustPGUUID(t, testImportUUID)

	if got := resolveAccountScopeKey(billingauthority.AccountResolveRequest{RepresentedSubjectID: "user-1"}); got != "user:user-1" {
		t.Fatalf("resolveAccountScopeKey(subject) = %q, want user:user-1", got)
	}
	if got := resolveAccountScopeKey(billingauthority.AccountResolveRequest{RepresentedAccountID: "acct_1"}); got != "acct_1" {
		t.Fatalf("resolveAccountScopeKey(account) = %q, want acct_1", got)
	}
	for _, tc := range []struct {
		accountState string
		importState  string
		wantFailure  string
		wantRuntime  string
	}{
		{accountState: "suspended", importState: "accepted", wantFailure: "account_suspended", wantRuntime: "fail_closed"},
		{accountState: "manual_review", importState: "accepted", wantFailure: "manual_review", wantRuntime: "fail_closed"},
		{accountState: "active", importState: "missing", wantFailure: "import_required", wantRuntime: "not_ready"},
		{accountState: "active", importState: "mismatch", wantFailure: "legacy_import_mismatch", wantRuntime: "fail_closed"},
		{accountState: "active", importState: "pending", wantFailure: "import_pending", wantRuntime: "not_ready"},
		{accountState: "active", importState: "accepted", wantFailure: "", wantRuntime: "ready"},
	} {
		if got := accountResolveFailureClass(tc.accountState, tc.importState); got != tc.wantFailure {
			t.Fatalf("accountResolveFailureClass(%q,%q) = %q, want %q", tc.accountState, tc.importState, got, tc.wantFailure)
		}
		if got := runtimeStateForBalance(tc.accountState, tc.importState); got != tc.wantRuntime {
			t.Fatalf("runtimeStateForBalance(%q,%q) = %q, want %q", tc.accountState, tc.importState, got, tc.wantRuntime)
		}
	}

	for _, tc := range []struct {
		row  sqlcgen.ReconciliationCase
		want string
	}{
		{row: sqlcgen.ReconciliationCase{UsageOperationID: usageID}, want: testUsageUUID},
		{row: sqlcgen.ReconciliationCase{MicroleaseChildDebitID: childID}, want: testChildDebitUUID},
		{row: sqlcgen.ReconciliationCase{MicroleaseID: microleaseID}, want: testMicroleaseUUID},
		{row: sqlcgen.ReconciliationCase{BillingEventInboxID: inboxID}, want: testInboxUUID},
		{row: sqlcgen.ReconciliationCase{LegacyBalanceImportID: importID}, want: testImportUUID},
		{row: sqlcgen.ReconciliationCase{}, want: ""},
	} {
		if got := reconciliationSafeLineage(tc.row); got != tc.want {
			t.Fatalf("reconciliationSafeLineage(%+v) = %q, want %q", tc.row, got, tc.want)
		}
	}

	operation := sqlcgen.UsageOperation{UsageOperationID: usageID, AccountID: accountID, AccountScopeKey: "user:1", State: "reserved"}
	snapshot := usageSnapshot(operation, "accepted", "idem-1", "fingerprint-1", testOutcomeUUID)
	if snapshot.UsageOperationID != testUsageUUID || snapshot.StoredOutcomeID != testOutcomeUUID || snapshot.ResultCode != "accepted" {
		t.Fatalf("usageSnapshot() = %+v, want stored accepted usage", snapshot)
	}
	child := mapTerminalChildDebitRecord(sqlcgen.MicroleaseChildDebit{
		MicroleaseChildDebitID:       childID,
		MicroleaseID:                 microleaseID,
		DebitAuthorizationID:         "debit-auth-1",
		UsageOperationID:             usageID,
		AccountID:                    accountID,
		AccountScopeKey:              "user:1",
		ProxyAllocatorOwnerID:        "proxy-owner-1",
		MicroleaseGeneration:         3,
		ChildSequence:                5,
		ChildCapUsdAtoms:             100,
		RequestBasisFingerprint:      "request-basis",
		PricingSnapshotID:            "pricing-1",
		PricingSnapshotFingerprint:   "pricing-fingerprint",
		QualifiedInferenceEvidenceID: inboxID,
	})
	if child.UsageOperationID != testUsageUUID || child.QualifiedInferenceEvidenceID != testInboxUUID {
		t.Fatalf("mapTerminalChildDebitRecord() = %+v, want optional linked identities", child)
	}
}

func TestMicroleaseMappingHelpersAndConflictTranslation(t *testing.T) {
	t.Parallel()

	row := sqlcgen.SpendingMicrolease{
		MicroleaseID:              mustPGUUID(t, testMicroleaseUUID),
		AccountID:                 mustPGUUID(t, testAccountUUID),
		AccountScopeKey:           "user:1",
		State:                     "active",
		IssuedCapUsdAtoms:         100,
		AvailableChildCapUsdAtoms: 40,
		TerminalChargedUsdAtoms:   10,
		TerminalReleasedUsdAtoms:  20,
		WriteOffUsdAtoms:          30,
		ExpiresAt:                 timestamptzValue(fixedPostgresTime()),
	}
	record := mapMicroleaseRecord(row)
	if record.MicroleaseID != testMicroleaseUUID || record.AvailableChildUSDAtoms != 40 || record.WriteOffUSDAtoms != 30 {
		t.Fatalf("mapMicroleaseRecord() = %+v, want mapped exposure fields", record)
	}
	if got := nonZeroTime(time.Time{}); got.IsZero() {
		t.Fatal("nonZeroTime(zero) returned zero")
	}
	if got := maxInt64(4, 9); got != 9 {
		t.Fatalf("maxInt64() = %d, want 9", got)
	}
	uniqueErr := &pgconn.PgError{Code: "23505"}
	if !isUniqueViolation(uniqueErr) {
		t.Fatal("isUniqueViolation(unique) = false, want true")
	}
	if err := translateAuthorityWriteError("create outcome", uniqueErr); !errors.Is(err, billingauthority.ErrConflict) {
		t.Fatalf("translateAuthorityWriteError(unique) = %v, want conflict", err)
	}
	if err := translateAuthorityWriteError("create outcome", errors.New("other")); !errors.Is(err, ErrBillingAuthorityRepository) {
		t.Fatalf("translateAuthorityWriteError(other) = %v, want repository error", err)
	}
	if got := usageTerminalState("write_off"); got != "written_off" {
		t.Fatalf("usageTerminalState(write_off) = %q, want written_off", got)
	}
	if got := usageTerminalState("finalize"); got != "finalized" {
		t.Fatalf("usageTerminalState(finalize) = %q, want finalized", got)
	}
}

func validPostgresReserveCommand() billingauthority.UsageReserveCommand {
	return billingauthority.UsageReserveCommand{
		AccountScopeKey:        "user:1",
		UsageOperationID:       testUsageUUID,
		MicroleaseID:           testMicroleaseUUID,
		MicroleaseChildDebitID: testChildDebitUUID,
		DebitAuthorizationID:   "debit-auth-1",
		ProxyAllocatorOwnerID:  "proxy-owner-1",
		MicroleaseGeneration:   1,
		LeaseFence:             "fence-1",
		ChildSequence:          1,
		ChildCapUSDAtoms:       100,
		RequestFingerprint:     "request-fingerprint",
	}
}

func validPostgresTerminalCommand() billingauthority.UsageTerminalCommand {
	return billingauthority.UsageTerminalCommand{
		AccountScopeKey:        "user:1",
		UsageOperationID:       testUsageUUID,
		MicroleaseID:           testMicroleaseUUID,
		MicroleaseChildDebitID: testChildDebitUUID,
		DebitAuthorizationID:   "debit-auth-1",
		TerminalOutcomeID:      testTerminalUUID,
		TerminalFingerprint:    "terminal-fingerprint",
		ChargedUSDAtoms:        40,
		ReleasedUSDAtoms:       60,
		TerminalKind:           "finalize",
		Pricing: billingauthority.PricingSnapshot{
			ID:          "pricing-1",
			Fingerprint: "pricing-fingerprint",
		},
	}
}

func validPostgresTerminalSettlementCommand() TerminalSettlementCommand {
	return TerminalSettlementCommand{
		InboxID:                    testInboxUUID,
		MicroleaseID:               testMicroleaseUUID,
		MicroleaseChildDebitID:     testChildDebitUUID,
		AccountID:                  testAccountUUID,
		AccountScopeKey:            "user:1",
		LedgerEntryID:              testLedgerUUID,
		SettlementEffectID:         testSettlementUUID,
		OutboxID:                   testOutboxUUID,
		ChargedUSDAtoms:            40,
		ReleasedUSDAtoms:           60,
		WriteOffUSDAtoms:           0,
		TerminalKind:               "finalize",
		TerminalBasisFingerprint:   "terminal-fingerprint",
		PricingSnapshotID:          "pricing-1",
		PricingSnapshotFingerprint: "pricing-fingerprint",
	}
}

func fixedPostgresTime() time.Time {
	return time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
}

func mustPGUUID(t *testing.T, raw string) pgtype.UUID {
	t.Helper()
	id, err := uuidValue(raw)
	if err != nil {
		t.Fatalf("uuidValue(%q): %v", raw, err)
	}
	return id
}
