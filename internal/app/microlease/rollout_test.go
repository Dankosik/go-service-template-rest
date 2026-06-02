package microlease

import (
	"testing"
	"time"
)

func TestRolloutGateDefaultsClosedAndSeparatesShadowFromSpend(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)

	inert := DecideRolloutGate(RolloutGateInput{Mode: RolloutModeInertExpand, Now: now})
	if inert.AllowIssue || inert.AllowExternalExecution || inert.AllowDirectReserveFallback || inert.Reason != "inert_expand" {
		t.Fatalf("inert decision = %+v, want closed inert expand", inert)
	}

	shadow := DecideRolloutGate(RolloutGateInput{
		Mode:             RolloutModeShadowNoSpend,
		Now:              now,
		BalanceParityOK:  true,
		ExposureParityOK: true,
	})
	if shadow.AllowIssue || shadow.AllowExternalExecution || shadow.AllowDirectReserveFallback || shadow.Reason != "shadow_no_spend" {
		t.Fatalf("shadow decision = %+v, want no-spend shadow", shadow)
	}
}

func TestRolloutGateRequiresParityAndNoDualWriterForMigratedCohorts(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)

	dualWriter := DecideRolloutGate(RolloutGateInput{
		Mode:                         RolloutModeMigrated,
		Now:                          now,
		BalanceParityOK:              true,
		ExposureParityOK:             true,
		OldProxyWriterDisabled:       false,
		DirectReserveFallbackEnabled: false,
	})
	if dualWriter.AllowIssue || dualWriter.Reason != "old_proxy_writer_enabled" {
		t.Fatalf("dual writer decision = %+v, want closed", dualWriter)
	}

	directFallback := DecideRolloutGate(RolloutGateInput{
		Mode:                         RolloutModeMigrated,
		Now:                          now,
		BalanceParityOK:              true,
		ExposureParityOK:             true,
		OldProxyWriterDisabled:       true,
		DirectReserveFallbackEnabled: true,
	})
	if directFallback.AllowIssue || directFallback.Reason != "direct_reserve_fallback_enabled" {
		t.Fatalf("direct fallback decision = %+v, want closed", directFallback)
	}

	enabled := DecideRolloutGate(RolloutGateInput{
		Mode:                         RolloutModeInternalCohort,
		InternalCohort:               true,
		Now:                          now,
		BalanceParityOK:              true,
		ExposureParityOK:             true,
		OldProxyWriterDisabled:       true,
		DirectReserveFallbackEnabled: false,
	})
	if !enabled.AllowIssue || !enabled.AllowExternalExecution || enabled.AllowDirectReserveFallback || enabled.Reason != "internal_cohort_enabled" {
		t.Fatalf("enabled decision = %+v, want migrated internal cohort", enabled)
	}
}

func TestRolloutRollbackUsesOnlyExistingMicroleasesUntilCutoff(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)

	validExisting := DecideRolloutGate(RolloutGateInput{
		Mode:               RolloutModeRollback,
		Now:                now,
		HasValidMicrolease: true,
		DebitCutoffAt:      now.Add(time.Second),
	})
	if !validExisting.AllowExternalExecution || validExisting.AllowIssue || validExisting.AllowDirectReserveFallback {
		t.Fatalf("rollback existing decision = %+v, want existing microlease only", validExisting)
	}

	cutoff := DecideRolloutGate(RolloutGateInput{
		Mode:               RolloutModeRollback,
		Now:                now,
		HasValidMicrolease: true,
		DebitCutoffAt:      now.Add(-time.Second),
	})
	if cutoff.AllowIssue || cutoff.AllowExternalExecution || cutoff.AllowDirectReserveFallback || cutoff.Reason != "rollback_fail_closed" {
		t.Fatalf("rollback cutoff decision = %+v, want fail closed", cutoff)
	}
}

func TestRolloutOperatorGatesFailClosed(t *testing.T) {
	t.Parallel()

	decision := DecideRolloutGate(RolloutGateInput{
		Mode:                         RolloutModeMigrated,
		BalanceParityOK:              true,
		ExposureParityOK:             true,
		OldProxyWriterDisabled:       true,
		DirectReserveFallbackEnabled: false,
		TerminalLagBucket:            "critical",
	})
	if decision.AllowIssue || decision.AllowExternalExecution || decision.Reason != "operator_gate_critical" {
		t.Fatalf("critical gate decision = %+v, want fail closed", decision)
	}
}
