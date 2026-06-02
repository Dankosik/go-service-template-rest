package billingauthority

import (
	"context"
	"fmt"
	"time"

	appmicrolease "github.com/Dankosik/billing-service/internal/app/microlease"
)

func New(store Store, opts ...Option) (*Service, error) {
	if store == nil {
		return nil, fmt.Errorf("%w: store is required", ErrNotReady)
	}
	svc := &Service{
		store: store,
		now: func() time.Time {
			return time.Now().UTC()
		},
	}
	for _, opt := range opts {
		opt(svc)
	}
	if svc.now == nil {
		return nil, fmt.Errorf("%w: clock is required", ErrNotReady)
	}
	return svc, nil
}

type Option func(*Service)

func WithClock(now func() time.Time) Option {
	return func(s *Service) {
		s.now = now
	}
}

func (s *Service) ResolveAccount(ctx context.Context, req AccountResolveRequest) (AccountSnapshot, error) {
	if err := s.require(); err != nil {
		return AccountSnapshot{}, err
	}
	if req.RepresentedSubjectID == "" && req.RepresentedAccountID == "" {
		return AccountSnapshot{}, fmt.Errorf("%w: represented subject is required", ErrInvalidRequest)
	}
	snapshot, err := s.store.ResolveAccount(ctx, req)
	if err != nil {
		return AccountSnapshot{}, fmt.Errorf("resolve account: %w", err)
	}
	return snapshot, nil
}

func (s *Service) ReadBalance(ctx context.Context, req BalanceReadRequest) (BalanceSnapshot, error) {
	if err := s.require(); err != nil {
		return BalanceSnapshot{}, err
	}
	if req.AccountScopeKey == "" || req.RepresentedSubjectID == "" {
		return BalanceSnapshot{}, fmt.Errorf("%w: account scope and subject are required", ErrInvalidRequest)
	}
	snapshot, err := s.store.ReadBalance(ctx, req)
	if err != nil {
		return BalanceSnapshot{}, fmt.Errorf("read balance: %w", err)
	}
	return snapshot, nil
}

func (s *Service) ReserveUsage(ctx context.Context, cmd UsageReserveCommand) (UsageOperationSnapshot, error) {
	if err := s.require(); err != nil {
		return UsageOperationSnapshot{}, err
	}
	if err := validateReserve(cmd); err != nil {
		return UsageOperationSnapshot{}, err
	}
	if cmd.AuthorityMode != AuthorityModeMicroleaseChildDebit {
		return UsageOperationSnapshot{}, fmt.Errorf("%w: direct_reserve_fallback_rejected", ErrRejected)
	}
	snapshot, err := s.store.ReserveUsage(ctx, cmd)
	if err != nil {
		return UsageOperationSnapshot{}, fmt.Errorf("reserve usage: %w", err)
	}
	return snapshot, nil
}

func (s *Service) FinalizeUsage(ctx context.Context, cmd UsageTerminalCommand) (UsageOperationSnapshot, error) {
	cmd.TerminalKind = "finalize"
	return s.completeUsage(ctx, cmd)
}

func (s *Service) WriteOffUsage(ctx context.Context, cmd UsageTerminalCommand) (UsageOperationSnapshot, error) {
	cmd.TerminalKind = "write_off"
	return s.completeUsage(ctx, cmd)
}

func (s *Service) ReverseUsage(ctx context.Context, cmd UsageReversalCommand) (UsageOperationSnapshot, error) {
	if err := s.require(); err != nil {
		return UsageOperationSnapshot{}, err
	}
	if cmd.AccountScopeKey == "" || cmd.UsageOperationID == "" || cmd.IdempotencyKey == "" || cmd.RequestFingerprint == "" {
		return UsageOperationSnapshot{}, fmt.Errorf("%w: reversal identity is required", ErrInvalidRequest)
	}
	if cmd.OriginalLedgerEntryID == "" || cmd.ReversalUSDAtoms <= 0 || cmd.ReasonCode == "" {
		return UsageOperationSnapshot{}, fmt.Errorf("%w: reversal lineage is required", ErrInvalidRequest)
	}
	if err := appmicrolease.ValidateSafeMetadata(cmd.Metadata); err != nil {
		return UsageOperationSnapshot{}, fmt.Errorf("validate reversal metadata: %w", err)
	}
	snapshot, err := s.store.ReverseUsage(ctx, cmd)
	if err != nil {
		return UsageOperationSnapshot{}, fmt.Errorf("reverse usage: %w", err)
	}
	return snapshot, nil
}

func (s *Service) ReadUsageOperation(ctx context.Context, req UsageReadbackRequest) (UsageOperationSnapshot, error) {
	if err := s.require(); err != nil {
		return UsageOperationSnapshot{}, err
	}
	if req.AccountScopeKey == "" || req.UsageOperationID == "" {
		return UsageOperationSnapshot{}, fmt.Errorf("%w: usage readback identity is required", ErrInvalidRequest)
	}
	snapshot, err := s.store.ReadUsageOperation(ctx, req)
	if err != nil {
		return UsageOperationSnapshot{}, fmt.Errorf("read usage operation: %w", err)
	}
	return snapshot, nil
}

func (s *Service) ListReconciliationCases(ctx context.Context, req ReconciliationCasesRequest) ([]ReconciliationCase, error) {
	if err := s.require(); err != nil {
		return nil, err
	}
	if req.Limit <= 0 {
		req.Limit = 100
	}
	cases, err := s.store.ListReconciliationCases(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("list reconciliation cases: %w", err)
	}
	return cases, nil
}

func (s *Service) ReadAdminLedger(ctx context.Context, req AdminLedgerRequest) ([]LedgerEntry, error) {
	if err := s.require(); err != nil {
		return nil, err
	}
	if req.AccountScopeKey == "" {
		return nil, fmt.Errorf("%w: account scope is required", ErrInvalidRequest)
	}
	if req.Limit <= 0 {
		req.Limit = 50
	}
	entries, err := s.store.ListLedgerEntries(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("read admin ledger: %w", err)
	}
	return entries, nil
}

func (s *Service) ReadAdminExposure(ctx context.Context, req AdminExposureRequest) (ExposureSnapshot, error) {
	if err := s.require(); err != nil {
		return ExposureSnapshot{}, err
	}
	if req.AccountScopeKey == "" {
		return ExposureSnapshot{}, fmt.Errorf("%w: account scope is required", ErrInvalidRequest)
	}
	exposure, err := s.store.ReadExposure(ctx, req)
	if err != nil {
		return ExposureSnapshot{}, fmt.Errorf("read admin exposure: %w", err)
	}
	return exposure, nil
}

func (s *Service) completeUsage(ctx context.Context, cmd UsageTerminalCommand) (UsageOperationSnapshot, error) {
	if err := s.require(); err != nil {
		return UsageOperationSnapshot{}, err
	}
	if err := validateTerminal(cmd); err != nil {
		return UsageOperationSnapshot{}, err
	}
	snapshot, err := s.store.CompleteUsage(ctx, cmd)
	if err != nil {
		return UsageOperationSnapshot{}, fmt.Errorf("complete usage: %w", err)
	}
	return snapshot, nil
}

func validateReserve(cmd UsageReserveCommand) error {
	if cmd.AccountScopeKey == "" || cmd.UsageOperationID == "" || cmd.IdempotencyKey == "" || cmd.RequestFingerprint == "" {
		return fmt.Errorf("%w: reserve identity is required", ErrInvalidRequest)
	}
	if cmd.MicroleaseID == "" || cmd.MicroleaseChildDebitID == "" || cmd.DebitAuthorizationID == "" ||
		cmd.ProxyAllocatorOwnerID == "" || cmd.MicroleaseGeneration <= 0 || cmd.LeaseFence == "" ||
		cmd.ChildSequence <= 0 || cmd.ChildCapUSDAtoms <= 0 {
		return fmt.Errorf("%w: microlease child lineage is required", ErrInvalidRequest)
	}
	if cmd.RepresentedSubjectID == "" {
		return fmt.Errorf("%w: represented subject is required", ErrInvalidRequest)
	}
	if err := validatePricing(cmd.Pricing); err != nil {
		return err
	}
	if err := appmicrolease.ValidateSafeMetadata(cmd.Metadata); err != nil {
		return fmt.Errorf("validate reserve metadata: %w", err)
	}
	return nil
}

func validateTerminal(cmd UsageTerminalCommand) error {
	if cmd.AccountScopeKey == "" || cmd.UsageOperationID == "" || cmd.IdempotencyKey == "" || cmd.RequestFingerprint == "" {
		return fmt.Errorf("%w: terminal identity is required", ErrInvalidRequest)
	}
	if err := validateTerminalLineage(cmd); err != nil {
		return err
	}
	if err := validateTerminalAmounts(cmd); err != nil {
		return err
	}
	if err := validatePricing(cmd.Pricing); err != nil {
		return err
	}
	if err := appmicrolease.ValidateSafeMetadata(cmd.Metadata); err != nil {
		return fmt.Errorf("validate terminal metadata: %w", err)
	}
	return nil
}

func validateTerminalLineage(cmd UsageTerminalCommand) error {
	if cmd.MicroleaseID == "" || cmd.MicroleaseChildDebitID == "" || cmd.DebitAuthorizationID == "" ||
		cmd.TerminalOutcomeID == "" || cmd.TerminalFingerprint == "" {
		return fmt.Errorf("%w: terminal lineage is required", ErrInvalidRequest)
	}
	return nil
}

func validateTerminalAmounts(cmd UsageTerminalCommand) error {
	if cmd.ChargedUSDAtoms < 0 || cmd.ReleasedUSDAtoms < 0 || cmd.WriteOffUSDAtoms < 0 {
		return fmt.Errorf("%w: terminal amounts must be non-negative", ErrInvalidRequest)
	}
	if cmd.ChargedUSDAtoms+cmd.ReleasedUSDAtoms+cmd.WriteOffUSDAtoms <= 0 {
		return fmt.Errorf("%w: terminal effect is required", ErrInvalidRequest)
	}
	switch cmd.TerminalKind {
	case "finalize":
		if cmd.WriteOffUSDAtoms != 0 {
			return fmt.Errorf("%w: finalize cannot write off", ErrInvalidRequest)
		}
	case "write_off":
		if cmd.ChargedUSDAtoms != 0 || cmd.ReleasedUSDAtoms != 0 || cmd.WriteOffUSDAtoms <= 0 {
			return fmt.Errorf("%w: write off requires only write off amount", ErrInvalidRequest)
		}
	default:
		return fmt.Errorf("%w: unsupported terminal kind", ErrInvalidRequest)
	}
	return nil
}

func validatePricing(snapshot PricingSnapshot) error {
	if snapshot.ID == "" || snapshot.Fingerprint == "" || snapshot.PolicyVersion == "" || snapshot.SelectorKey == "" || snapshot.ContractVersion == "" {
		return fmt.Errorf("%w: pricing snapshot evidence is required", ErrInvalidRequest)
	}
	if snapshot.DecisionAt.IsZero() {
		return fmt.Errorf("%w: pricing decision time is required", ErrInvalidRequest)
	}
	return nil
}

func (s *Service) require() error {
	if s == nil || s.store == nil {
		return fmt.Errorf("%w: service is not configured", ErrNotReady)
	}
	return nil
}
