package billingauthority

import (
	"context"
	"errors"
	"testing"
	"time"
)

type fakeStore struct {
	resolveCalls        int
	balanceCalls        int
	reserveCalls        int
	terminalCalls       int
	reverseCalls        int
	readbackCalls       int
	reconciliationCalls int
	ledgerCalls         int
	exposureCalls       int

	resolveRequest        AccountResolveRequest
	balanceRequest        BalanceReadRequest
	terminalCommand       UsageTerminalCommand
	reverseCommand        UsageReversalCommand
	readbackRequest       UsageReadbackRequest
	reconciliationRequest ReconciliationCasesRequest
	ledgerRequest         AdminLedgerRequest
	exposureRequest       AdminExposureRequest

	resolveErr        error
	balanceErr        error
	reserveErr        error
	terminalErr       error
	reverseErr        error
	readbackErr       error
	reconciliationErr error
	ledgerErr         error
	exposureErr       error
}

func (s *fakeStore) ResolveAccount(_ context.Context, req AccountResolveRequest) (AccountSnapshot, error) {
	s.resolveCalls++
	s.resolveRequest = req
	if s.resolveErr != nil {
		return AccountSnapshot{}, s.resolveErr
	}
	return AccountSnapshot{AccountScopeKey: "user:1", AccountState: "active", ImportState: "accepted", MigrationState: "migrated", BalanceReadEligible: true}, nil
}

func (s *fakeStore) ReadBalance(_ context.Context, req BalanceReadRequest) (BalanceSnapshot, error) {
	s.balanceCalls++
	s.balanceRequest = req
	if s.balanceErr != nil {
		return BalanceSnapshot{}, s.balanceErr
	}
	return BalanceSnapshot{AccountScopeKey: "user:1", RuntimeGateState: "ready", ImportState: "accepted"}, nil
}

func (s *fakeStore) ReserveUsage(_ context.Context, cmd UsageReserveCommand) (UsageOperationSnapshot, error) {
	s.reserveCalls++
	if s.reserveErr != nil {
		return UsageOperationSnapshot{}, s.reserveErr
	}
	return UsageOperationSnapshot{UsageOperationID: cmd.UsageOperationID, State: "reserved", ResultCode: "accepted"}, nil
}

func (s *fakeStore) CompleteUsage(_ context.Context, cmd UsageTerminalCommand) (UsageOperationSnapshot, error) {
	s.terminalCalls++
	s.terminalCommand = cmd
	if s.terminalErr != nil {
		return UsageOperationSnapshot{}, s.terminalErr
	}
	return UsageOperationSnapshot{UsageOperationID: cmd.UsageOperationID, State: cmd.TerminalKind + "d", ResultCode: "accepted"}, nil
}

func (s *fakeStore) ReverseUsage(_ context.Context, cmd UsageReversalCommand) (UsageOperationSnapshot, error) {
	s.reverseCalls++
	s.reverseCommand = cmd
	if s.reverseErr != nil {
		return UsageOperationSnapshot{}, s.reverseErr
	}
	return UsageOperationSnapshot{UsageOperationID: cmd.UsageOperationID, State: "reversed", ResultCode: "accepted"}, nil
}

func (s *fakeStore) ReadUsageOperation(_ context.Context, req UsageReadbackRequest) (UsageOperationSnapshot, error) {
	s.readbackCalls++
	s.readbackRequest = req
	if s.readbackErr != nil {
		return UsageOperationSnapshot{}, s.readbackErr
	}
	return UsageOperationSnapshot{UsageOperationID: req.UsageOperationID, State: "reserved", ResultCode: "readback"}, nil
}

func (s *fakeStore) ListReconciliationCases(_ context.Context, req ReconciliationCasesRequest) ([]ReconciliationCase, error) {
	s.reconciliationCalls++
	s.reconciliationRequest = req
	if s.reconciliationErr != nil {
		return nil, s.reconciliationErr
	}
	return []ReconciliationCase{{ReconciliationCaseID: "case-1", AccountScopeKey: req.AccountScopeKey, Reason: "stale_microlease", State: "open", Severity: "critical"}}, nil
}

func (s *fakeStore) ListLedgerEntries(_ context.Context, req AdminLedgerRequest) ([]LedgerEntry, error) {
	s.ledgerCalls++
	s.ledgerRequest = req
	if s.ledgerErr != nil {
		return nil, s.ledgerErr
	}
	return []LedgerEntry{{LedgerEntryID: "ledger-1", EffectType: "usage_charge", AmountUSDAtoms: -10, BalanceVersionAfter: 7}}, nil
}

func (s *fakeStore) ReadExposure(_ context.Context, req AdminExposureRequest) (ExposureSnapshot, error) {
	s.exposureCalls++
	s.exposureRequest = req
	if s.exposureErr != nil {
		return ExposureSnapshot{}, s.exposureErr
	}
	return ExposureSnapshot{AccountScopeKey: req.AccountScopeKey, RuntimeGateState: "ready", ActiveMicroleaseUSDAtoms: 40, UnresolvedChildDebitUSDAtoms: 10}, nil
}

func TestNewRejectsMissingStoreAndClock(t *testing.T) {
	t.Parallel()

	if _, err := New(nil); !errors.Is(err, ErrNotReady) {
		t.Fatalf("New(nil) error = %v, want ErrNotReady", err)
	}
	if _, err := New(&fakeStore{}, WithClock(nil)); !errors.Is(err, ErrNotReady) {
		t.Fatalf("New(nil clock) error = %v, want ErrNotReady", err)
	}
}

func TestAccountBalanceReadbackAndAdminMethodsValidateThenDelegate(t *testing.T) {
	t.Parallel()

	store := &fakeStore{}
	svc := mustService(t, store)

	if _, err := svc.ResolveAccount(context.Background(), AccountResolveRequest{}); !errors.Is(err, ErrInvalidRequest) {
		t.Fatalf("ResolveAccount(empty) error = %v, want ErrInvalidRequest", err)
	}
	if store.resolveCalls != 0 {
		t.Fatalf("resolve calls after invalid request = %d, want 0", store.resolveCalls)
	}
	account, err := svc.ResolveAccount(context.Background(), AccountResolveRequest{
		RepresentedSubjectID: "user-1",
		CallerPrincipal:      "svc:gonka-proxy",
		TraceRequestID:       "trace-1",
	})
	if err != nil {
		t.Fatalf("ResolveAccount() error = %v", err)
	}
	if !account.BalanceReadEligible || account.ImportState != "accepted" || store.resolveRequest.RepresentedSubjectID != "user-1" {
		t.Fatalf("ResolveAccount() = %+v request=%+v, want accepted represented user", account, store.resolveRequest)
	}

	if _, err := svc.ReadBalance(context.Background(), BalanceReadRequest{AccountScopeKey: "user:1"}); !errors.Is(err, ErrInvalidRequest) {
		t.Fatalf("ReadBalance(missing subject) error = %v, want ErrInvalidRequest", err)
	}
	balance, err := svc.ReadBalance(context.Background(), BalanceReadRequest{AccountScopeKey: "user:1", RepresentedSubjectID: "user-1"})
	if err != nil {
		t.Fatalf("ReadBalance() error = %v", err)
	}
	if balance.RuntimeGateState != "ready" || store.balanceRequest.AccountScopeKey != "user:1" {
		t.Fatalf("ReadBalance() = %+v request=%+v, want ready user:1", balance, store.balanceRequest)
	}

	if _, err := svc.ReadUsageOperation(context.Background(), UsageReadbackRequest{AccountScopeKey: "user:1"}); !errors.Is(err, ErrInvalidRequest) {
		t.Fatalf("ReadUsageOperation(missing usage) error = %v, want ErrInvalidRequest", err)
	}
	readback, err := svc.ReadUsageOperation(context.Background(), UsageReadbackRequest{AccountScopeKey: "user:1", UsageOperationID: "usage-1"})
	if err != nil {
		t.Fatalf("ReadUsageOperation() error = %v", err)
	}
	if readback.UsageOperationID != "usage-1" || readback.ResultCode != "readback" {
		t.Fatalf("ReadUsageOperation() = %+v, want usage-1 readback", readback)
	}

	cases, err := svc.ListReconciliationCases(context.Background(), ReconciliationCasesRequest{AccountScopeKey: "user:1"})
	if err != nil {
		t.Fatalf("ListReconciliationCases() error = %v", err)
	}
	if len(cases) != 1 || store.reconciliationRequest.Limit != 100 {
		t.Fatalf("cases=%+v request=%+v, want default limit 100 and one case", cases, store.reconciliationRequest)
	}

	if _, err := svc.ReadAdminLedger(context.Background(), AdminLedgerRequest{}); !errors.Is(err, ErrInvalidRequest) {
		t.Fatalf("ReadAdminLedger(empty) error = %v, want ErrInvalidRequest", err)
	}
	entries, err := svc.ReadAdminLedger(context.Background(), AdminLedgerRequest{AccountScopeKey: "user:1"})
	if err != nil {
		t.Fatalf("ReadAdminLedger() error = %v", err)
	}
	if len(entries) != 1 || store.ledgerRequest.Limit != 50 {
		t.Fatalf("entries=%+v request=%+v, want default limit 50 and one ledger entry", entries, store.ledgerRequest)
	}

	if _, err := svc.ReadAdminExposure(context.Background(), AdminExposureRequest{}); !errors.Is(err, ErrInvalidRequest) {
		t.Fatalf("ReadAdminExposure(empty) error = %v, want ErrInvalidRequest", err)
	}
	exposure, err := svc.ReadAdminExposure(context.Background(), AdminExposureRequest{AccountScopeKey: "user:1"})
	if err != nil {
		t.Fatalf("ReadAdminExposure() error = %v", err)
	}
	if exposure.ActiveMicroleaseUSDAtoms != 40 || store.exposureRequest.AccountScopeKey != "user:1" {
		t.Fatalf("ReadAdminExposure() = %+v request=%+v, want active exposure", exposure, store.exposureRequest)
	}
}

func TestReserveUsageRejectsDirectReserveFallbackBeforeStore(t *testing.T) {
	t.Parallel()

	store := &fakeStore{}
	svc := mustService(t, store)
	cmd := validReserveCommand()
	cmd.AuthorityMode = AuthorityModeDirectReserveFallback

	_, err := svc.ReserveUsage(context.Background(), cmd)
	if !errors.Is(err, ErrRejected) {
		t.Fatalf("ReserveUsage() error = %v, want ErrRejected", err)
	}
	if store.reserveCalls != 0 {
		t.Fatalf("reserve calls = %d, want 0", store.reserveCalls)
	}
}

func TestReserveUsageRequiresMicroleaseLineageAndSafeMetadata(t *testing.T) {
	t.Parallel()

	store := &fakeStore{}
	svc := mustService(t, store)
	cmd := validReserveCommand()
	cmd.MicroleaseID = ""

	_, err := svc.ReserveUsage(context.Background(), cmd)
	if !errors.Is(err, ErrInvalidRequest) {
		t.Fatalf("ReserveUsage() error = %v, want ErrInvalidRequest", err)
	}
	if store.reserveCalls != 0 {
		t.Fatalf("reserve calls = %d, want 0", store.reserveCalls)
	}

	cmd = validReserveCommand()
	cmd.Metadata = map[string]string{"raw_prompt": "secret"}
	_, err = svc.ReserveUsage(context.Background(), cmd)
	if err == nil {
		t.Fatal("ReserveUsage(unsafe metadata) error = nil, want error")
	}
}

func TestReserveUsageAcceptsMicroleaseBackedCommand(t *testing.T) {
	t.Parallel()

	store := &fakeStore{}
	svc := mustService(t, store)

	got, err := svc.ReserveUsage(context.Background(), validReserveCommand())
	if err != nil {
		t.Fatalf("ReserveUsage() error = %v", err)
	}
	if got.UsageOperationID != "usage-1" || got.State != "reserved" || got.ResultCode != "accepted" {
		t.Fatalf("ReserveUsage() = %+v, want accepted reserved usage-1", got)
	}
	if store.reserveCalls != 1 {
		t.Fatalf("reserve calls = %d, want 1", store.reserveCalls)
	}
}

func TestTerminalUsageRequiresEffectAndLineage(t *testing.T) {
	t.Parallel()

	store := &fakeStore{}
	svc := mustService(t, store)
	cmd := validTerminalCommand()
	cmd.ChargedUSDAtoms = 0
	cmd.ReleasedUSDAtoms = 0

	_, err := svc.FinalizeUsage(context.Background(), cmd)
	if !errors.Is(err, ErrInvalidRequest) {
		t.Fatalf("FinalizeUsage() error = %v, want ErrInvalidRequest", err)
	}
	if store.terminalCalls != 0 {
		t.Fatalf("terminal calls = %d, want 0", store.terminalCalls)
	}
}

func TestTerminalUsageAppliesFinalizeAndExplicitWriteOffSemantics(t *testing.T) {
	t.Parallel()

	store := &fakeStore{}
	svc := mustService(t, store)

	finalized, err := svc.FinalizeUsage(context.Background(), validTerminalCommand())
	if err != nil {
		t.Fatalf("FinalizeUsage() error = %v", err)
	}
	if finalized.State != "finalized" || store.terminalCommand.TerminalKind != "finalize" {
		t.Fatalf("FinalizeUsage() = %+v terminal=%+v, want finalized kind", finalized, store.terminalCommand)
	}

	writeOff := validTerminalCommand()
	writeOff.ChargedUSDAtoms = 0
	writeOff.ReleasedUSDAtoms = 0
	writeOff.WriteOffUSDAtoms = 100
	writtenOff, err := svc.WriteOffUsage(context.Background(), writeOff)
	if err != nil {
		t.Fatalf("WriteOffUsage() error = %v", err)
	}
	if writtenOff.State != "write_offd" || store.terminalCommand.TerminalKind != "write_off" {
		t.Fatalf("WriteOffUsage() = %+v terminal=%+v, want explicit write off", writtenOff, store.terminalCommand)
	}

	invalidWriteOff := validTerminalCommand()
	invalidWriteOff.ChargedUSDAtoms = 1
	invalidWriteOff.ReleasedUSDAtoms = 0
	invalidWriteOff.WriteOffUSDAtoms = 100
	if _, err := svc.WriteOffUsage(context.Background(), invalidWriteOff); !errors.Is(err, ErrInvalidRequest) {
		t.Fatalf("WriteOffUsage(charged) error = %v, want ErrInvalidRequest", err)
	}
}

func TestReverseUsageRequiresLineageAndSafeMetadata(t *testing.T) {
	t.Parallel()

	store := &fakeStore{}
	svc := mustService(t, store)

	cmd := validReversalCommand()
	cmd.OriginalLedgerEntryID = ""
	if _, err := svc.ReverseUsage(context.Background(), cmd); !errors.Is(err, ErrInvalidRequest) {
		t.Fatalf("ReverseUsage(missing lineage) error = %v, want ErrInvalidRequest", err)
	}
	if store.reverseCalls != 0 {
		t.Fatalf("reverse calls after invalid lineage = %d, want 0", store.reverseCalls)
	}

	cmd = validReversalCommand()
	cmd.Metadata = map[string]string{"bearer_token": "secret"}
	if _, err := svc.ReverseUsage(context.Background(), cmd); err == nil {
		t.Fatal("ReverseUsage(unsafe metadata) error = nil, want error")
	}
	if store.reverseCalls != 0 {
		t.Fatalf("reverse calls after unsafe metadata = %d, want 0", store.reverseCalls)
	}

	cmd = validReversalCommand()
	reversed, err := svc.ReverseUsage(context.Background(), cmd)
	if err != nil {
		t.Fatalf("ReverseUsage() error = %v", err)
	}
	if reversed.State != "reversed" || store.reverseCommand.OriginalLedgerEntryID != "ledger-1" {
		t.Fatalf("ReverseUsage() = %+v command=%+v, want reversed ledger-1", reversed, store.reverseCommand)
	}
}

func mustService(t *testing.T, store Store) *Service {
	t.Helper()
	svc, err := New(store, WithClock(func() time.Time {
		return time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	}))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	return svc
}

func validReserveCommand() UsageReserveCommand {
	return UsageReserveCommand{
		AccountScopeKey:        "user:1",
		UsageOperationID:       "usage-1",
		AuthorityMode:          AuthorityModeMicroleaseChildDebit,
		IdempotencyKey:         "idem-1",
		RequestFingerprint:     "fingerprint-1",
		RequestID:              "request-1",
		MicroleaseID:           "microlease-1",
		MicroleaseChildDebitID: "child-1",
		DebitAuthorizationID:   "debit-1",
		ProxyAllocatorOwnerID:  "proxy-owner-1",
		MicroleaseGeneration:   1,
		LeaseFence:             "fence-1",
		ChildSequence:          1,
		ChildCapUSDAtoms:       100,
		RepresentedSubjectID:   "user-1",
		Pricing:                validPricingSnapshot(),
		Metadata:               map[string]string{"surface": "chat"},
	}
}

func validTerminalCommand() UsageTerminalCommand {
	return UsageTerminalCommand{
		AccountScopeKey:        "user:1",
		UsageOperationID:       "usage-1",
		IdempotencyKey:         "idem-terminal-1",
		RequestFingerprint:     "terminal-fingerprint-1",
		MicroleaseID:           "microlease-1",
		MicroleaseChildDebitID: "child-1",
		DebitAuthorizationID:   "debit-1",
		TerminalOutcomeID:      "terminal-1",
		TerminalFingerprint:    "terminal-basis-1",
		ChargedUSDAtoms:        50,
		ReleasedUSDAtoms:       50,
		Pricing:                validPricingSnapshot(),
	}
}

func validReversalCommand() UsageReversalCommand {
	return UsageReversalCommand{
		AccountScopeKey:       "user:1",
		UsageOperationID:      "usage-1",
		IdempotencyKey:        "idem-reversal-1",
		RequestFingerprint:    "reversal-fingerprint-1",
		OriginalLedgerEntryID: "ledger-1",
		ReversalUSDAtoms:      50,
		ReasonCode:            "operator_reversal",
		CallerPrincipal:       "svc:gonka-proxy",
		CallerScopeKey:        "proxy-owner-1",
		TraceRequestID:        "trace-reversal-1",
		DeadlineAt:            time.Date(2026, 6, 1, 12, 1, 0, 0, time.UTC),
		Metadata:              map[string]string{"surface": "admin_readback"},
	}
}

func validPricingSnapshot() PricingSnapshot {
	return PricingSnapshot{
		ID:              "pricing-1",
		Fingerprint:     "pricing-fingerprint-1",
		PolicyVersion:   "pricing-policy-v1",
		DecisionAt:      time.Date(2026, 6, 1, 11, 59, 0, 0, time.UTC),
		SelectorKey:     "gnk_usdt:usage_reserve",
		UseClass:        "usage_reserve",
		ContractVersion: "pricing.v1",
	}
}
