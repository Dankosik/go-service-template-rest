package telemetry

import (
	"testing"
	"time"
)

func TestBuildReleaseReadinessEvidence(t *testing.T) {
	evidence := BuildReleaseReadinessEvidence(ReleaseReadinessEvidenceInput{
		DeployHealthGood:  995,
		DeployHealthTotal: 1000,
		RollbackGood:      99,
		RollbackTotal:     100,
		ConfigDriftGood:   10,
		ConfigDriftTotal:  10,

		ErrorBudgetConsumedPercent: 20,

		DeployHealthBurnRate:    6.1,
		DeployHealthShortEvents: 3,
		DeployHealthLongEvents:  5,

		RollbackFailures:        1,
		ConfigDriftOpenDuration: 25 * time.Hour,
		PolicyThresholds: PolicyThresholdInput{
			DeployHealthSLOTargetPercent: 99.5,
			CapacityReplicaFloor:         2,
			CapacityPerReplicaCPU:        2,
			CapacityPerReplicaMemoryGiB:  2,
			PromotionTimeout:             180 * time.Second,
			DrainWindow:                  45 * time.Second,
			ShutdownTimeout:              30 * time.Second,
			RestartMaxRetries:            5,
		},
	})

	if evidence.DeployHealthAdmissionRatio != 0.995 {
		t.Fatalf("DeployHealthAdmissionRatio = %v, want 0.995", evidence.DeployHealthAdmissionRatio)
	}
	if evidence.RollbackRecoveryRatio != 0.99 {
		t.Fatalf("RollbackRecoveryRatio = %v, want 0.99", evidence.RollbackRecoveryRatio)
	}
	if evidence.ConfigDriftReconcileRatio != 1 {
		t.Fatalf("ConfigDriftReconcileRatio = %v, want 1", evidence.ConfigDriftReconcileRatio)
	}
	if evidence.BudgetState != "green" {
		t.Fatalf("BudgetState = %q, want %q", evidence.BudgetState, "green")
	}
	if !evidence.DeployHealthPage {
		t.Fatal("DeployHealthPage = false, want true")
	}
	if !evidence.RollbackFailurePage {
		t.Fatal("RollbackFailurePage = false, want true")
	}
	if !evidence.ConfigDriftTicket {
		t.Fatal("ConfigDriftTicket = false, want true")
	}
	if !evidence.ConfigDriftPage {
		t.Fatal("ConfigDriftPage = false, want true")
	}
	if !evidence.PolicyThresholds.SLOTargetCompliant {
		t.Fatal("PolicyThresholds.SLOTargetCompliant = false, want true")
	}
	if !evidence.PolicyThresholds.CapacityBaselineCompliant {
		t.Fatal("PolicyThresholds.CapacityBaselineCompliant = false, want true")
	}
	if !evidence.PolicyThresholds.NumericPolicyCompliant {
		t.Fatal("PolicyThresholds.NumericPolicyCompliant = false, want true")
	}
	if evidence.PolicyThresholds.ReliabilityState != "pass" {
		t.Fatalf("PolicyThresholds.ReliabilityState = %q, want %q", evidence.PolicyThresholds.ReliabilityState, "pass")
	}
	if evidence.PolicyThresholds.BlockReleaseReadiness {
		t.Fatal("PolicyThresholds.BlockReleaseReadiness = true, want false")
	}
}

func TestBuildReleaseReadinessEvidenceDefaults(t *testing.T) {
	evidence := BuildReleaseReadinessEvidence(ReleaseReadinessEvidenceInput{
		ErrorBudgetConsumedPercent: -10,
	})

	if evidence.DeployHealthAdmissionRatio != 1 {
		t.Fatalf("DeployHealthAdmissionRatio = %v, want 1", evidence.DeployHealthAdmissionRatio)
	}
	if evidence.RollbackRecoveryRatio != 1 {
		t.Fatalf("RollbackRecoveryRatio = %v, want 1", evidence.RollbackRecoveryRatio)
	}
	if evidence.ConfigDriftReconcileRatio != 1 {
		t.Fatalf("ConfigDriftReconcileRatio = %v, want 1", evidence.ConfigDriftReconcileRatio)
	}
	if evidence.ErrorBudgetConsumedPercent != 0 {
		t.Fatalf("ErrorBudgetConsumedPercent = %v, want 0", evidence.ErrorBudgetConsumedPercent)
	}
	if evidence.BudgetState != "green" {
		t.Fatalf("BudgetState = %q, want %q", evidence.BudgetState, "green")
	}
	if evidence.DeployHealthPage {
		t.Fatal("DeployHealthPage = true, want false")
	}
	if evidence.RollbackFailurePage {
		t.Fatal("RollbackFailurePage = true, want false")
	}
	if evidence.ConfigDriftTicket {
		t.Fatal("ConfigDriftTicket = true, want false")
	}
	if evidence.ConfigDriftPage {
		t.Fatal("ConfigDriftPage = true, want false")
	}
	if !evidence.PolicyThresholds.SLOTargetCompliant {
		t.Fatal("PolicyThresholds.SLOTargetCompliant = false, want true")
	}
	if !evidence.PolicyThresholds.CapacityBaselineCompliant {
		t.Fatal("PolicyThresholds.CapacityBaselineCompliant = false, want true")
	}
	if !evidence.PolicyThresholds.NumericPolicyCompliant {
		t.Fatal("PolicyThresholds.NumericPolicyCompliant = false, want true")
	}
	if evidence.PolicyThresholds.ReliabilityState != "pass" {
		t.Fatalf("PolicyThresholds.ReliabilityState = %q, want %q", evidence.PolicyThresholds.ReliabilityState, "pass")
	}
}

func TestBuildReleaseReadinessEvidenceCapacityDegraded(t *testing.T) {
	evidence := BuildReleaseReadinessEvidence(ReleaseReadinessEvidenceInput{
		PolicyThresholds: PolicyThresholdInput{
			DeployHealthSLOTargetPercent: 99.5,
			CapacityReplicaFloor:         1,
			CapacityPerReplicaCPU:        2,
			CapacityPerReplicaMemoryGiB:  2,
			PromotionTimeout:             180 * time.Second,
			DrainWindow:                  45 * time.Second,
			ShutdownTimeout:              30 * time.Second,
			RestartMaxRetries:            5,
		},
	})

	if evidence.PolicyThresholds.CapacityState != "degraded" {
		t.Fatalf("PolicyThresholds.CapacityState = %q, want %q", evidence.PolicyThresholds.CapacityState, "degraded")
	}
	if evidence.PolicyThresholds.ReliabilityState != "degraded" {
		t.Fatalf("PolicyThresholds.ReliabilityState = %q, want %q", evidence.PolicyThresholds.ReliabilityState, "degraded")
	}
	if !evidence.PolicyThresholds.ScaleActionRequired {
		t.Fatal("PolicyThresholds.ScaleActionRequired = false, want true")
	}
	if !evidence.PolicyThresholds.BlockReleaseReadiness {
		t.Fatal("PolicyThresholds.BlockReleaseReadiness = false, want true")
	}
	if !containsViolation(evidence.PolicyThresholds.Violations, "replica_floor") {
		t.Fatalf("PolicyThresholds.Violations = %v, want to contain replica_floor", evidence.PolicyThresholds.Violations)
	}
}

func TestBuildReleaseReadinessEvidenceNumericPolicyBlocked(t *testing.T) {
	evidence := BuildReleaseReadinessEvidence(ReleaseReadinessEvidenceInput{
		PolicyThresholds: PolicyThresholdInput{
			DeployHealthSLOTargetPercent: 99.0,
			CapacityReplicaFloor:         2,
			CapacityPerReplicaCPU:        2,
			CapacityPerReplicaMemoryGiB:  2,
			PromotionTimeout:             200 * time.Second,
			DrainWindow:                  40 * time.Second,
			ShutdownTimeout:              30 * time.Second,
			RestartMaxRetries:            6,
		},
	})

	if evidence.PolicyThresholds.ThresholdState != "fail" {
		t.Fatalf("PolicyThresholds.ThresholdState = %q, want %q", evidence.PolicyThresholds.ThresholdState, "fail")
	}
	if evidence.PolicyThresholds.ReliabilityState != "blocked" {
		t.Fatalf("PolicyThresholds.ReliabilityState = %q, want %q", evidence.PolicyThresholds.ReliabilityState, "blocked")
	}
	if !evidence.PolicyThresholds.BlockReleaseReadiness {
		t.Fatal("PolicyThresholds.BlockReleaseReadiness = false, want true")
	}
	if !containsViolation(evidence.PolicyThresholds.Violations, "deploy_health_slo_target") {
		t.Fatalf("PolicyThresholds.Violations = %v, want to contain deploy_health_slo_target", evidence.PolicyThresholds.Violations)
	}
	if !containsViolation(evidence.PolicyThresholds.Violations, "promotion_timeout") {
		t.Fatalf("PolicyThresholds.Violations = %v, want to contain promotion_timeout", evidence.PolicyThresholds.Violations)
	}
	if !containsViolation(evidence.PolicyThresholds.Violations, "drain_window") {
		t.Fatalf("PolicyThresholds.Violations = %v, want to contain drain_window", evidence.PolicyThresholds.Violations)
	}
	if !containsViolation(evidence.PolicyThresholds.Violations, "restart_max_retries") {
		t.Fatalf("PolicyThresholds.Violations = %v, want to contain restart_max_retries", evidence.PolicyThresholds.Violations)
	}
}

func containsViolation(violations []string, needle string) bool {
	for _, violation := range violations {
		if violation == needle {
			return true
		}
	}
	return false
}
