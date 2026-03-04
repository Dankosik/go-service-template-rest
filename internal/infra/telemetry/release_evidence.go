package telemetry

import (
	"math"
	"time"
)

const (
	deployHealthBurnRatePageThreshold = 6.0
	deployHealthShortFloor            = 2
	deployHealthLongFloor             = 5
	configDriftTicketThreshold        = 4 * time.Hour
	configDriftPageThreshold          = 24 * time.Hour

	expectedDeployHealthSLOTargetPercent = 99.5
	expectedReplicaFloor                 = int64(2)
	expectedReplicaCPUVCores             = int64(2)
	expectedReplicaMemoryGiB             = int64(2)
	expectedPromotionTimeout             = 180 * time.Second
	expectedDrainWindow                  = 45 * time.Second
	expectedShutdownTimeout              = 30 * time.Second
	expectedRestartMaxRetries            = int64(5)
)

type ReleaseReadinessEvidenceInput struct {
	DeployHealthGood  int64
	DeployHealthTotal int64

	RollbackGood  int64
	RollbackTotal int64

	ConfigDriftGood  int64
	ConfigDriftTotal int64

	ErrorBudgetConsumedPercent float64

	DeployHealthBurnRate    float64
	DeployHealthShortEvents int64
	DeployHealthLongEvents  int64

	RollbackFailures int64

	ConfigDriftOpenDuration time.Duration

	PolicyThresholds PolicyThresholdInput
}

type PolicyThresholdInput struct {
	DeployHealthSLOTargetPercent float64

	CapacityReplicaFloor        int64
	CapacityPerReplicaCPU       int64
	CapacityPerReplicaMemoryGiB int64

	PromotionTimeout  time.Duration
	DrainWindow       time.Duration
	ShutdownTimeout   time.Duration
	RestartMaxRetries int64
}

type PolicyThresholdEvidence struct {
	DeployHealthSLOTargetPercent float64

	CapacityReplicaFloor        int64
	CapacityPerReplicaCPU       int64
	CapacityPerReplicaMemoryGiB int64

	PromotionTimeoutSeconds int64
	DrainWindowSeconds      int64
	ShutdownTimeoutSeconds  int64
	RestartMaxRetries       int64

	SLOTargetCompliant        bool
	CapacityBaselineCompliant bool
	NumericPolicyCompliant    bool

	CapacityState    string
	ThresholdState   string
	ReliabilityState string

	ScaleActionRequired   bool
	BlockReleaseReadiness bool
	Violations            []string
}

type ReleaseReadinessEvidence struct {
	DeployHealthAdmissionRatio float64
	RollbackRecoveryRatio      float64
	ConfigDriftReconcileRatio  float64

	ErrorBudgetConsumedPercent float64
	BudgetState                string

	DeployHealthPage    bool
	RollbackFailurePage bool
	ConfigDriftTicket   bool
	ConfigDriftPage     bool

	PolicyThresholds PolicyThresholdEvidence
}

func BuildReleaseReadinessEvidence(in ReleaseReadinessEvidenceInput) ReleaseReadinessEvidence {
	consumed := normalizeErrorBudgetConsumed(in.ErrorBudgetConsumedPercent)
	policy := buildPolicyThresholdEvidence(in.PolicyThresholds)

	return ReleaseReadinessEvidence{
		DeployHealthAdmissionRatio: ratio(in.DeployHealthGood, in.DeployHealthTotal),
		RollbackRecoveryRatio:      ratio(in.RollbackGood, in.RollbackTotal),
		ConfigDriftReconcileRatio:  ratio(in.ConfigDriftGood, in.ConfigDriftTotal),
		ErrorBudgetConsumedPercent: consumed,
		BudgetState:                budgetStateFromConsumed(consumed),
		DeployHealthPage:           shouldPageDeployHealth(in),
		RollbackFailurePage:        in.RollbackFailures > 0,
		ConfigDriftTicket:          in.ConfigDriftOpenDuration > configDriftTicketThreshold,
		ConfigDriftPage:            in.ConfigDriftOpenDuration > configDriftPageThreshold,
		PolicyThresholds:           policy,
	}
}

func ratio(good, total int64) float64 {
	if total <= 0 {
		return 1
	}
	if good < 0 {
		good = 0
	}
	if good > total {
		good = total
	}
	return float64(good) / float64(total)
}

func normalizeErrorBudgetConsumed(consumed float64) float64 {
	if consumed < 0 {
		return 0
	}
	return consumed
}

func budgetStateFromConsumed(consumed float64) string {
	switch {
	case consumed <= 25:
		return "green"
	case consumed <= 50:
		return "yellow"
	case consumed <= 100:
		return "orange"
	default:
		return "red"
	}
}

func shouldPageDeployHealth(in ReleaseReadinessEvidenceInput) bool {
	return in.DeployHealthBurnRate >= deployHealthBurnRatePageThreshold &&
		in.DeployHealthShortEvents >= deployHealthShortFloor &&
		in.DeployHealthLongEvents >= deployHealthLongFloor
}

func buildPolicyThresholdEvidence(in PolicyThresholdInput) PolicyThresholdEvidence {
	sloTarget := withFloatDefault(in.DeployHealthSLOTargetPercent, expectedDeployHealthSLOTargetPercent)
	replicaFloor := withInt64Default(in.CapacityReplicaFloor, expectedReplicaFloor)
	cpu := withInt64Default(in.CapacityPerReplicaCPU, expectedReplicaCPUVCores)
	memory := withInt64Default(in.CapacityPerReplicaMemoryGiB, expectedReplicaMemoryGiB)
	promotionTimeout := withDurationDefault(in.PromotionTimeout, expectedPromotionTimeout)
	drainWindow := withDurationDefault(in.DrainWindow, expectedDrainWindow)
	shutdownTimeout := withDurationDefault(in.ShutdownTimeout, expectedShutdownTimeout)
	restartMaxRetries := withInt64Default(in.RestartMaxRetries, expectedRestartMaxRetries)

	sloTargetCompliant := nearlyEqual(sloTarget, expectedDeployHealthSLOTargetPercent)
	capacityCompliant := replicaFloor >= expectedReplicaFloor &&
		cpu == expectedReplicaCPUVCores &&
		memory == expectedReplicaMemoryGiB
	numericPolicyCompliant := promotionTimeout == expectedPromotionTimeout &&
		drainWindow == expectedDrainWindow &&
		shutdownTimeout == expectedShutdownTimeout &&
		restartMaxRetries == expectedRestartMaxRetries

	violations := make([]string, 0, 8)
	if !sloTargetCompliant {
		violations = append(violations, "deploy_health_slo_target")
	}
	if replicaFloor < expectedReplicaFloor {
		violations = append(violations, "replica_floor")
	}
	if cpu != expectedReplicaCPUVCores {
		violations = append(violations, "per_replica_cpu")
	}
	if memory != expectedReplicaMemoryGiB {
		violations = append(violations, "per_replica_memory")
	}
	if promotionTimeout != expectedPromotionTimeout {
		violations = append(violations, "promotion_timeout")
	}
	if drainWindow != expectedDrainWindow {
		violations = append(violations, "drain_window")
	}
	if shutdownTimeout != expectedShutdownTimeout {
		violations = append(violations, "shutdown_timeout")
	}
	if restartMaxRetries != expectedRestartMaxRetries {
		violations = append(violations, "restart_max_retries")
	}

	capacityState := "healthy"
	if !capacityCompliant {
		capacityState = "degraded"
	}

	thresholdState := "pass"
	if !sloTargetCompliant || !numericPolicyCompliant {
		thresholdState = "fail"
	}

	reliabilityState := "pass"
	switch {
	case thresholdState == "fail":
		reliabilityState = "blocked"
	case capacityState == "degraded":
		reliabilityState = "degraded"
	}

	return PolicyThresholdEvidence{
		DeployHealthSLOTargetPercent: sloTarget,
		CapacityReplicaFloor:         replicaFloor,
		CapacityPerReplicaCPU:        cpu,
		CapacityPerReplicaMemoryGiB:  memory,
		PromotionTimeoutSeconds:      int64(promotionTimeout / time.Second),
		DrainWindowSeconds:           int64(drainWindow / time.Second),
		ShutdownTimeoutSeconds:       int64(shutdownTimeout / time.Second),
		RestartMaxRetries:            restartMaxRetries,
		SLOTargetCompliant:           sloTargetCompliant,
		CapacityBaselineCompliant:    capacityCompliant,
		NumericPolicyCompliant:       numericPolicyCompliant,
		CapacityState:                capacityState,
		ThresholdState:               thresholdState,
		ReliabilityState:             reliabilityState,
		ScaleActionRequired:          !capacityCompliant,
		BlockReleaseReadiness:        reliabilityState != "pass",
		Violations:                   violations,
	}
}

func withFloatDefault(value, defaultValue float64) float64 {
	if value == 0 {
		return defaultValue
	}
	return value
}

func withInt64Default(value, defaultValue int64) int64 {
	if value == 0 {
		return defaultValue
	}
	return value
}

func withDurationDefault(value, defaultValue time.Duration) time.Duration {
	if value == 0 {
		return defaultValue
	}
	return value
}

func nearlyEqual(left, right float64) bool {
	return math.Abs(left-right) <= 1e-9
}
