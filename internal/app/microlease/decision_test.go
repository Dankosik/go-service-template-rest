package microlease

import (
	"errors"
	"testing"
	"testing/quick"
	"time"
)

func TestDecideIssueAppliesCapFormulaAndSafetyFloor(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	decision, err := DecideIssue(validConfig(), AccountExposure{
		SettledUSDAtoms:          200_000_000,
		ReservedUSDAtoms:         10_000_000,
		ActiveMicroleaseUSDAtoms: 20_000_000,
		AdmissionControl:         openAdmission(now),
	}, validIssueRequest(now))
	if err != nil {
		t.Fatalf("DecideIssue() error = %v", err)
	}
	if decision.Result != IssueResultReducedIssued {
		t.Fatalf("result = %q, want reduced_issued", decision.Result)
	}
	if decision.IssuedUSDAtoms != 16_000_000 {
		t.Fatalf("issued = %d, want 16000000", decision.IssuedUSDAtoms)
	}
	if !decision.ExpiresAt.Equal(now.Add(30 * time.Second)) {
		t.Fatalf("expires = %v, want ttl-applied", decision.ExpiresAt)
	}
	if !decision.DebitCutoffAt.Equal(now.Add(25 * time.Second)) {
		t.Fatalf("cutoff = %v, want 25s", decision.DebitCutoffAt)
	}
}

func TestDecideIssueFailClosedAndStrictGates(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	tests := []struct {
		name     string
		exposure AccountExposure
		request  IssueRequest
		want     IssueResult
	}{
		{
			name: "missing admission control fails closed",
			exposure: AccountExposure{
				SettledUSDAtoms:  100_000_000,
				ReservedUSDAtoms: 0,
			},
			request: validIssueRequest(now),
			want:    IssueResultAdmissionFailClose,
		},
		{
			name: "manual review",
			exposure: AccountExposure{
				SettledUSDAtoms:  100_000_000,
				ReservedUSDAtoms: 0,
				ManualReview:     true,
				AdmissionControl: openAdmission(now),
			},
			request: validIssueRequest(now),
			want:    IssueResultManualReview,
		},
		{
			name: "strict admission",
			exposure: AccountExposure{
				SettledUSDAtoms:  100_000_000,
				ReservedUSDAtoms: 0,
				AdmissionControl: AdmissionControl{State: AdmissionStateStrict, Reason: "operator_strict", ExpiresAt: now.Add(time.Minute)},
			},
			request: validIssueRequest(now),
			want:    IssueResultStrictRequired,
		},
		{
			name: "stale pricing",
			exposure: AccountExposure{
				SettledUSDAtoms:  100_000_000,
				ReservedUSDAtoms: 0,
				AdmissionControl: openAdmission(now),
			},
			request: func() IssueRequest {
				req := validIssueRequest(now)
				req.Pricing.Fingerprint = ""
				return req
			}(),
			want: IssueResultStalePricing,
		},
		{
			name: "insufficient safety floor",
			exposure: AccountExposure{
				SettledUSDAtoms:  4_999_999,
				ReservedUSDAtoms: 0,
				AdmissionControl: openAdmission(now),
			},
			request: validIssueRequest(now),
			want:    IssueResultInsufficientFunds,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := DecideIssue(validConfig(), tt.exposure, tt.request)
			if err != nil {
				t.Fatalf("DecideIssue() error = %v", err)
			}
			if got.Result != tt.want {
				t.Fatalf("result = %q, want %q", got.Result, tt.want)
			}
		})
	}
}

func TestIssueDecisionNeverCreatesNegativeAvailableExposure(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	cfg := validConfig()
	err := quick.Check(func(settledInput uint32, reservedInput uint32, activeInput uint32, requestedInput uint32, childInput uint32) bool {
		settled := int64(settledInput%500_000_000) + 10_000_000
		reserved := int64(reservedInput % uint32(settled))
		active := int64(activeInput % 150_000_000)
		requested := int64(requestedInput%200_000_000) + 1
		child := int64(childInput%5_000_000) + 1
		req := validIssueRequest(now)
		req.RequestedUSDAtoms = requested
		req.MaxChildUSDAtoms = child
		decision, err := DecideIssue(cfg, AccountExposure{
			SettledUSDAtoms:          settled,
			ReservedUSDAtoms:         reserved,
			ActiveMicroleaseUSDAtoms: active,
			AdmissionControl:         openAdmission(now),
		}, req)
		if err != nil {
			return false
		}
		if decision.IssuedUSDAtoms == 0 {
			return true
		}
		availableAfter := settled - reserved - decision.IssuedUSDAtoms
		return decision.IssuedUSDAtoms <= requested &&
			decision.IssuedUSDAtoms <= cfg.MaxMicroleaseUSDAtoms &&
			decision.IssuedUSDAtoms <= child*16 &&
			active+decision.IssuedUSDAtoms <= cfg.AccountMicroleaseExposureCapAtom &&
			availableAfter >= 0
	}, &quick.Config{MaxCount: 500})
	if err != nil {
		t.Fatalf("microlease issue invariant failed: %v", err)
	}
}

func TestReplayOrConflict(t *testing.T) {
	t.Parallel()

	stored := StoredOutcome[string]{
		IdempotencyKey:     "idem",
		RequestFingerprint: "fingerprint-a",
		Outcome:            "stored-result",
	}
	got, err := ReplayOrConflict(stored, "fingerprint-a")
	if err != nil {
		t.Fatalf("ReplayOrConflict() error = %v", err)
	}
	if got != "stored-result" {
		t.Fatalf("stored outcome = %q", got)
	}
	_, err = ReplayOrConflict(stored, "fingerprint-b")
	if !errors.Is(err, ErrPayloadConflict) {
		t.Fatalf("ReplayOrConflict conflict error = %v, want ErrPayloadConflict", err)
	}
}

func TestTerminalAndCloseDecisionsPreserveCaps(t *testing.T) {
	t.Parallel()

	grant := GrantSnapshot{
		MicroleaseID:              "microlease-1",
		State:                     MicroleaseStateActive,
		IssuedCapUSDAtoms:         1_000,
		AvailableChildCapUSDAtoms: 500,
		AllocatedChildCapUSDAtoms: 500,
		ProxyAllocatorOwner:       "proxy-a",
		Generation:                1,
		LeaseFence:                "fence-a",
		ExpiresAt:                 time.Now().Add(time.Minute),
	}
	terminal := ApplyTerminal(grant, TerminalReport{
		Child:    ChildDebit{AuthorizationID: "child-1", Sequence: 1, ChildCapUSDAtoms: 100},
		Kind:     TerminalKindFinalize,
		Charged:  90,
		Released: 20,
	})
	if !terminal.ReconcileRequired || terminal.ReconciliationReason != "child_cap_exceeded" {
		t.Fatalf("terminal decision = %+v, want child cap reconciliation", terminal)
	}

	closeDecision := DecideClose(grant, CloseProof{
		Sequence:                   1,
		Kind:                       "close",
		AllocatedChildCapUSDAtoms:  500,
		UnresolvedChildCount:       1,
		UnresolvedChildCapUSDAtoms: 200,
		LocalRemainingUSDAtoms:     500,
		CheckpointFingerprint:      "checkpoint",
		ProxyAllocatorOwner:        "proxy-a",
		Generation:                 1,
		LeaseFence:                 "fence-a",
		ProofObservedAt:            time.Now(),
	})
	if closeDecision.ReleasedUSDAtoms != 500 || closeDecision.UnresolvedReservedUSDAtom != 200 {
		t.Fatalf("close decision = %+v, want only unallocated release and unresolved reserve", closeDecision)
	}
	if !closeDecision.ReconcileRequired {
		t.Fatal("close with unresolved child cap must open reconciliation")
	}

	expired := ExpiryDecision(grant, grant.ExpiresAt.Add(time.Second))
	if expired.ReleasedUSDAtoms != 0 || !expired.ReconcileRequired {
		t.Fatalf("expiry decision = %+v, want no release without proof", expired)
	}
}

func TestConfigAndRequestValidationRejectUnsafeIssueInputs(t *testing.T) {
	t.Parallel()

	for _, mutate := range []func(*Config){
		func(c *Config) { c.MaxMicroleaseUSDAtoms = 0 },
		func(c *Config) { c.AccountMicroleaseExposureCapAtom = 0 },
		func(c *Config) { c.MinSafetyFloorUSDAtoms = -1 },
		func(c *Config) { c.TTL = 0 },
		func(c *Config) { c.DebitCutoffBeforeExpiry = 0 },
		func(c *Config) { c.DebitCutoffBeforeExpiry = c.TTL },
		func(c *Config) { c.TerminalDeadline = 0 },
		func(c *Config) { c.StaleDebitWarningAge = 0 },
		func(c *Config) { c.StaleDebitWarningAge = c.StaleDebitCriticalAge },
		func(c *Config) { c.ReconciliationSLA = 0 },
	} {
		cfg := validConfig()
		mutate(&cfg)
		if !errors.Is(ValidateConfig(cfg), ErrInvalidConfig) {
			t.Fatalf("ValidateConfig(%+v) did not return ErrInvalidConfig", cfg)
		}
	}

	now := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	for _, mutate := range []func(*IssueRequest){
		func(r *IssueRequest) { r.AccountScopeKey = "" },
		func(r *IssueRequest) { r.ProxyAllocatorOwner = "" },
		func(r *IssueRequest) { r.Generation = 0 },
		func(r *IssueRequest) { r.LeaseFence = "" },
		func(r *IssueRequest) { r.IdempotencyKey = "" },
		func(r *IssueRequest) { r.RequestFingerprint = "" },
		func(r *IssueRequest) { r.RequestedUSDAtoms = 0 },
		func(r *IssueRequest) { r.MaxChildUSDAtoms = 0 },
	} {
		req := validIssueRequest(now)
		mutate(&req)
		if _, err := DecideIssue(validConfig(), AccountExposure{AdmissionControl: openAdmission(now)}, req); !errors.Is(err, ErrInvalidRequest) {
			t.Fatalf("DecideIssue(%+v) error = %v, want ErrInvalidRequest", req, err)
		}
	}
}

func TestTerminalCloseAndExpiryEdgeCases(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	grant := GrantSnapshot{
		MicroleaseID:              "microlease-1",
		State:                     MicroleaseStateActive,
		IssuedCapUSDAtoms:         1_000,
		AvailableChildCapUSDAtoms: 500,
		ProxyAllocatorOwner:       "proxy-a",
		Generation:                1,
		LeaseFence:                "fence-a",
		ExpiresAt:                 now.Add(time.Minute),
	}
	if got := ApplyTerminal(grant, TerminalReport{Child: ChildDebit{ChildCapUSDAtoms: 0}}); !got.ReconcileRequired || got.ReconciliationReason != "invalid_child_cap" {
		t.Fatalf("ApplyTerminal(invalid child) = %+v, want invalid child reconciliation", got)
	}
	if got := ApplyTerminal(grant, TerminalReport{Child: ChildDebit{ChildCapUSDAtoms: 1_600}, Charged: 10}); !got.ReconcileRequired || got.ReconciliationReason != "parent_cap_exceeded" {
		t.Fatalf("ApplyTerminal(parent exceeded) = %+v, want parent reconciliation", got)
	}
	if got := ApplyTerminal(grant, TerminalReport{Child: ChildDebit{ChildCapUSDAtoms: 100}, Charged: -5, Released: 80}); got.ReconcileRequired || got.ChargedUSDAtoms != 0 || got.ReleasedUSDAtoms != 80 {
		t.Fatalf("ApplyTerminal(valid) = %+v, want non-negative applied amounts", got)
	}

	closeProof := CloseProof{
		Sequence:                  2,
		AllocatedChildCapUSDAtoms: 500,
		ProxyAllocatorOwner:       "proxy-a",
		Generation:                1,
		LeaseFence:                "fence-a",
	}
	if got := DecideClose(grant, CloseProof{ProxyAllocatorOwner: "other", Generation: 1, LeaseFence: "fence-a"}); !got.ReconcileRequired || got.ReconciliationReason != "owner_fence_mismatch" {
		t.Fatalf("DecideClose(owner mismatch) = %+v", got)
	}
	staleGrant := grant
	staleGrant.LastCheckpointSequence = 3
	if got := DecideClose(staleGrant, closeProof); !got.ReconcileRequired || got.ReconciliationReason != "stale_checkpoint" {
		t.Fatalf("DecideClose(stale) = %+v", got)
	}
	invalidProof := closeProof
	invalidProof.AllocatedChildCapUSDAtoms = 2_000
	if got := DecideClose(grant, invalidProof); !got.ReconcileRequired || got.ReconciliationReason != "invalid_close_cap_summary" {
		t.Fatalf("DecideClose(invalid cap) = %+v", got)
	}
	if got := DecideClose(grant, closeProof); got.State != MicroleaseStateClosed || got.ReleasedUSDAtoms != 500 || got.ReconcileRequired {
		t.Fatalf("DecideClose(valid) = %+v, want closed release", got)
	}
	if got := ExpiryDecision(grant, now); got.State != MicroleaseStateActive || got.ReconcileRequired {
		t.Fatalf("ExpiryDecision(before expiry) = %+v, want active", got)
	}
}

func TestValidateSafeMetadata(t *testing.T) {
	t.Parallel()

	if err := ValidateSafeMetadata(map[string]string{"reason": "terminal_lag_bucket_critical"}); err != nil {
		t.Fatalf("ValidateSafeMetadata(safe) error = %v", err)
	}
	if !errors.Is(ValidateSafeMetadata(map[string]string{"debug": "raw_prompt"}), ErrUnsafeMetadata) {
		t.Fatal("ValidateSafeMetadata(raw prompt) did not reject unsafe metadata")
	}
}

func validConfig() Config {
	return Config{
		MaxMicroleaseUSDAtoms:            100_000_000,
		AccountMicroleaseExposureCapAtom: 200_000_000,
		MinSafetyFloorUSDAtoms:           5_000_000,
		TTL:                              30 * time.Second,
		DebitCutoffBeforeExpiry:          5 * time.Second,
		TerminalDeadline:                 120 * time.Second,
		StaleDebitWarningAge:             60 * time.Second,
		StaleDebitCriticalAge:            180 * time.Second,
		ReconciliationSLA:                5 * time.Minute,
	}
}

func validIssueRequest(now time.Time) IssueRequest {
	return IssueRequest{
		AccountScopeKey:     "user:user-1",
		ProxyAllocatorOwner: "proxy-a",
		Generation:          1,
		LeaseFence:          "fence-a",
		IdempotencyKey:      "idem-issue",
		RequestFingerprint:  "request-fingerprint",
		RequestedUSDAtoms:   100_000_000,
		MaxChildUSDAtoms:    1_000_000,
		UseClass:            "chat",
		RiskClass:           "low",
		Pricing: PricingSnapshot{
			ID:              "pricing-snapshot",
			Fingerprint:     "pricing-fingerprint",
			PolicyVersion:   "pricing-v1",
			DecisionAt:      now.Add(-time.Second),
			SelectorKey:     "model:gpt-4.1:chat",
			ContractVersion: "pricing-contract-v1",
		},
		Metadata: map[string]string{"lag_bucket": "ok"},
		Now:      now,
	}
}

func openAdmission(now time.Time) AdmissionControl {
	return AdmissionControl{State: AdmissionStateOpen, Reason: "green", ExpiresAt: now.Add(time.Minute)}
}
