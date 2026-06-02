package microlease

import (
	"fmt"
	"math"
	"time"
)

func ValidateConfig(cfg Config) error {
	switch {
	case cfg.MaxMicroleaseUSDAtoms <= 0:
		return fmt.Errorf("%w: max microlease cap must be positive", ErrInvalidConfig)
	case cfg.AccountMicroleaseExposureCapAtom <= 0:
		return fmt.Errorf("%w: account exposure cap must be positive", ErrInvalidConfig)
	case cfg.MinSafetyFloorUSDAtoms < 0:
		return fmt.Errorf("%w: safety floor must be non-negative", ErrInvalidConfig)
	case cfg.TTL <= 0:
		return fmt.Errorf("%w: ttl must be positive", ErrInvalidConfig)
	case cfg.DebitCutoffBeforeExpiry <= 0:
		return fmt.Errorf("%w: debit cutoff must be positive", ErrInvalidConfig)
	case cfg.DebitCutoffBeforeExpiry >= cfg.TTL:
		return fmt.Errorf("%w: debit cutoff must be before expiry", ErrInvalidConfig)
	case cfg.TerminalDeadline <= 0:
		return fmt.Errorf("%w: terminal deadline must be positive", ErrInvalidConfig)
	case cfg.StaleDebitWarningAge <= 0 || cfg.StaleDebitCriticalAge <= 0:
		return fmt.Errorf("%w: stale debit ages must be positive", ErrInvalidConfig)
	case cfg.StaleDebitWarningAge >= cfg.StaleDebitCriticalAge:
		return fmt.Errorf("%w: warning age must be below critical age", ErrInvalidConfig)
	case cfg.ReconciliationSLA <= 0:
		return fmt.Errorf("%w: reconciliation sla must be positive", ErrInvalidConfig)
	default:
		return nil
	}
}

func DecideIssue(cfg Config, exposure AccountExposure, req IssueRequest) (IssueDecision, error) {
	if err := ValidateConfig(cfg); err != nil {
		return IssueDecision{}, err
	}
	if err := validateIssueRequest(req); err != nil {
		return IssueDecision{}, err
	}
	if err := ValidateSafeMetadata(req.Metadata); err != nil {
		return IssueDecision{}, err
	}

	now := req.Now
	if now.IsZero() {
		now = time.Now().UTC()
	}
	if exposure.ManualReview || exposure.Suspended {
		return IssueDecision{Result: IssueResultManualReview, StrictReason: "account_not_admissible"}, nil
	}
	if exposure.TerminalLagCritical || exposure.ReconciliationBacklogBlock {
		return IssueDecision{Result: IssueResultReconcileRequired, StrictReason: "runtime_health_gate"}, nil
	}
	if !req.Pricing.Valid(now) {
		return IssueDecision{Result: IssueResultStalePricing, StrictReason: "pricing_snapshot_invalid"}, nil
	}
	if admissionResult, reason, closed := admissionControlBlock(exposure.AdmissionControl, now); closed {
		return IssueDecision{Result: admissionResult, StrictReason: reason}, nil
	}

	availableAfterSafety := exposure.AvailableUSDAtoms() - safetyFloor(cfg, req.MaxChildUSDAtoms)
	exposureRemaining := cfg.AccountMicroleaseExposureCapAtom - exposure.ActiveMicroleaseUSDAtoms
	capCandidate := minPositive(
		cfg.MaxMicroleaseUSDAtoms,
		saturatingMultiply(req.MaxChildUSDAtoms, 16),
		exposureRemaining,
		availableAfterSafety,
		req.RequestedUSDAtoms,
	)
	if capCandidate <= 0 {
		return IssueDecision{Result: IssueResultInsufficientFunds, StrictReason: "insufficient_available_after_safety_floor"}, nil
	}

	result := IssueResultIssued
	if capCandidate < req.RequestedUSDAtoms {
		result = IssueResultReducedIssued
	}
	if exposure.AdmissionControl.State == AdmissionStateStrict {
		result = IssueResultStrictRequired
	}

	expiresAt := now.Add(cfg.TTL)
	return IssueDecision{
		Result:                 result,
		StrictReason:           strictReason(exposure.AdmissionControl),
		IssuedUSDAtoms:         capCandidate,
		AvailableChildUSDAtoms: capCandidate,
		DebitCutoffAt:          expiresAt.Add(-cfg.DebitCutoffBeforeExpiry),
		ExpiresAt:              expiresAt,
	}, nil
}

func ReplayOrConflict[T any](stored StoredOutcome[T], requestFingerprint string) (T, error) {
	if stored.RequestFingerprint == requestFingerprint {
		return stored.Outcome, nil
	}
	var zero T
	return zero, ErrPayloadConflict
}

func ApplyTerminal(grant GrantSnapshot, report TerminalReport) TerminalDecision {
	if report.Child.ChildCapUSDAtoms <= 0 {
		return TerminalDecision{ReconcileRequired: true, ReconciliationReason: "invalid_child_cap"}
	}
	reportedTotal := nonNegative(report.Charged) + nonNegative(report.Released) + nonNegative(report.WriteOff)
	if reportedTotal > report.Child.ChildCapUSDAtoms {
		return TerminalDecision{
			ChargedUSDAtoms:        minInt64(nonNegative(report.Charged), report.Child.ChildCapUSDAtoms),
			ReleasedUSDAtoms:       0,
			WriteOffUSDAtoms:       0,
			ReconcileRequired:      true,
			ReconciliationReason:   "child_cap_exceeded",
			ParentRemainingUSDAtom: parentRemaining(grant),
		}
	}

	parentRemaining := parentRemaining(grant)
	if report.Child.ChildCapUSDAtoms > parentRemaining+grant.AvailableChildCapUSDAtoms {
		return TerminalDecision{
			ChargedUSDAtoms:        minInt64(nonNegative(report.Charged), report.Child.ChildCapUSDAtoms),
			ReleasedUSDAtoms:       nonNegative(report.Released),
			WriteOffUSDAtoms:       nonNegative(report.WriteOff),
			ReconcileRequired:      true,
			ReconciliationReason:   "parent_cap_exceeded",
			ParentRemainingUSDAtom: parentRemaining,
		}
	}

	return TerminalDecision{
		ChargedUSDAtoms:        nonNegative(report.Charged),
		ReleasedUSDAtoms:       nonNegative(report.Released),
		WriteOffUSDAtoms:       nonNegative(report.WriteOff),
		ReconcileRequired:      false,
		ParentRemainingUSDAtom: parentRemaining,
	}
}

func DecideClose(grant GrantSnapshot, proof CloseProof) CloseDecision {
	if proof.ProxyAllocatorOwner != grant.ProxyAllocatorOwner || proof.Generation != grant.Generation || proof.LeaseFence != grant.LeaseFence {
		return CloseDecision{
			State:                MicroleaseStateReconcileRequired,
			ReconcileRequired:    true,
			ReconciliationReason: "owner_fence_mismatch",
		}
	}
	if proof.Sequence < grant.LastCheckpointSequence {
		return CloseDecision{
			State:                MicroleaseStateReconcileRequired,
			ReconcileRequired:    true,
			ReconciliationReason: "stale_checkpoint",
		}
	}
	if proof.AllocatedChildCapUSDAtoms > grant.IssuedCapUSDAtoms || proof.UnresolvedChildCapUSDAtoms > proof.AllocatedChildCapUSDAtoms {
		return CloseDecision{
			State:                MicroleaseStateReconcileRequired,
			ReconcileRequired:    true,
			ReconciliationReason: "invalid_close_cap_summary",
		}
	}

	released := max(grant.IssuedCapUSDAtoms-proof.AllocatedChildCapUSDAtoms, 0)
	if proof.UnresolvedChildCapUSDAtoms > 0 || proof.UnresolvedChildCount > 0 {
		return CloseDecision{
			State:                     MicroleaseStateReconcileRequired,
			ReleasedUSDAtoms:          released,
			UnresolvedReservedUSDAtom: proof.UnresolvedChildCapUSDAtoms,
			ReconcileRequired:         true,
			ReconciliationReason:      "unresolved_child_cap_reserved",
		}
	}
	return CloseDecision{
		State:                     MicroleaseStateClosed,
		ReleasedUSDAtoms:          released,
		UnresolvedReservedUSDAtom: 0,
	}
}

func ExpiryDecision(grant GrantSnapshot, now time.Time) CloseDecision {
	if now.Before(grant.ExpiresAt) {
		return CloseDecision{State: grant.State}
	}
	return CloseDecision{
		State:                     MicroleaseStateExpired,
		ReleasedUSDAtoms:          0,
		UnresolvedReservedUSDAtom: parentRemaining(grant),
		ReconcileRequired:         true,
		ReconciliationReason:      "expiry_requires_close_or_terminal_proof",
	}
}

func validateIssueRequest(req IssueRequest) error {
	switch {
	case req.AccountScopeKey == "":
		return fmt.Errorf("%w: account scope is required", ErrInvalidRequest)
	case req.ProxyAllocatorOwner == "":
		return fmt.Errorf("%w: proxy allocator owner is required", ErrInvalidRequest)
	case req.Generation <= 0:
		return fmt.Errorf("%w: generation must be positive", ErrInvalidRequest)
	case req.LeaseFence == "":
		return fmt.Errorf("%w: lease fence is required", ErrInvalidRequest)
	case req.IdempotencyKey == "":
		return fmt.Errorf("%w: idempotency key is required", ErrInvalidRequest)
	case req.RequestFingerprint == "":
		return fmt.Errorf("%w: request fingerprint is required", ErrInvalidRequest)
	case req.RequestedUSDAtoms <= 0:
		return fmt.Errorf("%w: requested amount must be positive", ErrInvalidRequest)
	case req.MaxChildUSDAtoms <= 0:
		return fmt.Errorf("%w: max child cap must be positive", ErrInvalidRequest)
	default:
		return nil
	}
}

func admissionControlBlock(control AdmissionControl, now time.Time) (IssueResult, string, bool) {
	if control.State == "" || control.ExpiresAt.IsZero() || !control.ExpiresAt.After(now) {
		return IssueResultAdmissionFailClose, "admission_control_missing_or_expired", true
	}
	switch control.State {
	case AdmissionStateFailClosed:
		return IssueResultAdmissionFailClose, reasonOrDefault(control.Reason, "admission_fail_closed"), true
	case AdmissionStateThrottle:
		return IssueResultRejected, reasonOrDefault(control.Reason, "admission_throttle"), true
	case AdmissionStateOpen, AdmissionStateStrict:
		return "", "", false
	default:
		return IssueResultAdmissionFailClose, "admission_control_malformed", true
	}
}

func strictReason(control AdmissionControl) string {
	if control.State == AdmissionStateStrict {
		return reasonOrDefault(control.Reason, "strict_mode")
	}
	return ""
}

func safetyFloor(cfg Config, maxChild int64) int64 {
	return maxInt64(cfg.MinSafetyFloorUSDAtoms, saturatingMultiply(maxChild, 2))
}

func minPositive(values ...int64) int64 {
	minValue := int64(math.MaxInt64)
	for _, value := range values {
		if value <= 0 {
			return 0
		}
		if value < minValue {
			minValue = value
		}
	}
	if minValue == math.MaxInt64 {
		return 0
	}
	return minValue
}

func saturatingMultiply(value, factor int64) int64 {
	if value <= 0 || factor <= 0 {
		return 0
	}
	if value > math.MaxInt64/factor {
		return math.MaxInt64
	}
	return value * factor
}

func parentRemaining(grant GrantSnapshot) int64 {
	remaining := grant.IssuedCapUSDAtoms - grant.TerminalChargedUSDAtoms - grant.TerminalReleasedUSDAtoms - grant.WriteOffUSDAtoms
	if remaining < 0 {
		return 0
	}
	return remaining
}

func nonNegative(value int64) int64 {
	if value < 0 {
		return 0
	}
	return value
}

func minInt64(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}

func maxInt64(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}

func reasonOrDefault(reason, fallback string) string {
	if reason != "" {
		return reason
	}
	return fallback
}
