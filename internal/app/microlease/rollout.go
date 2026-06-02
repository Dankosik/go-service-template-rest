package microlease

import "time"

type RolloutMode string

const (
	RolloutModeInertExpand    RolloutMode = "inert_expand"
	RolloutModeShadowNoSpend  RolloutMode = "shadow_no_spend"
	RolloutModeInternalCohort RolloutMode = "internal_cohort"
	RolloutModeMigrated       RolloutMode = "migrated"
	RolloutModeRollback       RolloutMode = "rollback"
)

type RolloutGateInput struct {
	Mode                         RolloutMode
	InternalCohort               bool
	HasValidMicrolease           bool
	DebitCutoffAt                time.Time
	Now                          time.Time
	BalanceParityOK              bool
	ExposureParityOK             bool
	TerminalLagBucket            string
	StaleExposureBucket          string
	ReconciliationBacklogBucket  string
	OldProxyWriterDisabled       bool
	DirectReserveFallbackEnabled bool
}

type RolloutGateDecision struct {
	AllowIssue                     bool
	AllowExternalExecution         bool
	AllowDirectReserveFallback     bool
	RequireOldProxyWriterDisabled  bool
	RequireOperatorVisibleReadback bool
	Reason                         string
}

func DecideRolloutGate(input RolloutGateInput) RolloutGateDecision {
	now := input.Now
	if now.IsZero() {
		now = time.Now().UTC()
	}
	if input.TerminalLagBucket == "critical" || input.StaleExposureBucket == "critical" || input.ReconciliationBacklogBucket == "critical" {
		return closedRolloutDecision("operator_gate_critical")
	}

	switch input.Mode {
	case RolloutModeInertExpand:
		return closedRolloutDecision("inert_expand")
	case RolloutModeShadowNoSpend:
		if !input.BalanceParityOK || !input.ExposureParityOK {
			return closedRolloutDecision("shadow_parity_failed")
		}
		return RolloutGateDecision{Reason: "shadow_no_spend", RequireOperatorVisibleReadback: true}
	case RolloutModeInternalCohort:
		if !input.InternalCohort {
			return closedRolloutDecision("cohort_not_enabled")
		}
		if !input.BalanceParityOK || !input.ExposureParityOK {
			return closedRolloutDecision("cohort_parity_failed")
		}
		return migratedDecision(input, "internal_cohort_enabled")
	case RolloutModeMigrated:
		return migratedDecision(input, "migrated_enabled")
	case RolloutModeRollback:
		if input.HasValidMicrolease && !input.DebitCutoffAt.IsZero() && now.Before(input.DebitCutoffAt) {
			return RolloutGateDecision{
				AllowExternalExecution:         true,
				RequireOldProxyWriterDisabled:  true,
				RequireOperatorVisibleReadback: true,
				Reason:                         "rollback_existing_microlease_until_cutoff",
			}
		}
		return closedRolloutDecision("rollback_fail_closed")
	default:
		return closedRolloutDecision("unknown_rollout_mode")
	}
}

func migratedDecision(input RolloutGateInput, reason string) RolloutGateDecision {
	if input.DirectReserveFallbackEnabled {
		return closedRolloutDecision("direct_reserve_fallback_enabled")
	}
	if !input.OldProxyWriterDisabled {
		return closedRolloutDecision("old_proxy_writer_enabled")
	}
	if !input.BalanceParityOK || !input.ExposureParityOK {
		return closedRolloutDecision("migrated_parity_failed")
	}
	return RolloutGateDecision{
		AllowIssue:                     true,
		AllowExternalExecution:         true,
		RequireOldProxyWriterDisabled:  true,
		RequireOperatorVisibleReadback: true,
		Reason:                         reason,
	}
}

func closedRolloutDecision(reason string) RolloutGateDecision {
	return RolloutGateDecision{
		RequireOldProxyWriterDisabled:  true,
		RequireOperatorVisibleReadback: true,
		Reason:                         reason,
	}
}
