package telemetry

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestPolicyThresholdInputFromRailwayTOML(t *testing.T) {
	input, err := PolicyThresholdInputFromRailwayTOML(`
[deploy]
healthcheckTimeout = 180
restartPolicyMaxRetries = 5
overlapSeconds = 45
drainingSeconds = 30
# - production replica baseline: >=2
# - per-replica baseline: 2 vCPU / 2 GiB
`)
	if err != nil {
		t.Fatalf("PolicyThresholdInputFromRailwayTOML() error = %v", err)
	}

	evidence := BuildReleaseReadinessEvidence(ReleaseReadinessEvidenceInput{
		PolicyThresholds: input,
	})

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

func TestPolicyThresholdInputFromRailwayTOMLRejectsMissingFields(t *testing.T) {
	_, err := PolicyThresholdInputFromRailwayTOML(`
[deploy]
healthcheckTimeout = 180
`)
	if err == nil {
		t.Fatal("PolicyThresholdInputFromRailwayTOML() error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "restartPolicyMaxRetries") {
		t.Fatalf("error = %v, want missing restartPolicyMaxRetries", err)
	}
}

func TestPolicyThresholdInputFromRepositoryRailwayToml(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", ".."))
	policyPath := filepath.Join(repoRoot, "railway.toml")

	raw, err := os.ReadFile(policyPath)
	if err != nil {
		t.Fatalf("os.ReadFile(%q) error = %v", policyPath, err)
	}

	input, err := PolicyThresholdInputFromRailwayTOML(string(raw))
	if err != nil {
		t.Fatalf("PolicyThresholdInputFromRailwayTOML(repository railway.toml) error = %v", err)
	}

	evidence := BuildReleaseReadinessEvidence(ReleaseReadinessEvidenceInput{
		PolicyThresholds: input,
	})
	if evidence.PolicyThresholds.ReliabilityState != "pass" {
		t.Fatalf("PolicyThresholds.ReliabilityState = %q, want %q", evidence.PolicyThresholds.ReliabilityState, "pass")
	}
	if evidence.PolicyThresholds.BlockReleaseReadiness {
		t.Fatal("PolicyThresholds.BlockReleaseReadiness = true, want false")
	}
}
