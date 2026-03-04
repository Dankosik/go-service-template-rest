//go:build integration

package integration_test

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"
	"time"
)

type scenarioEvidenceExecutor struct {
	name        string
	scenarioIDs []string
	evidenceIDs []string
	run         func(t *testing.T, repoRoot string)
}

var deploymentResilienceExecutors = []scenarioEvidenceExecutor{
	{
		name:        "health-contract-readiness",
		scenarioIDs: []string{"SCN-001", "SCN-003", "SCN-005"},
		evidenceIDs: []string{"EVID-002"},
		run: func(t *testing.T, repoRoot string) {
			runCommand(t, repoRoot, 2*time.Minute,
				"go", "test", "./internal/app/health",
				"-run", "^(TestServiceReadySuccess|TestServiceReadyFail|TestServiceReadyDraining)$",
				"-count=1",
			)
		},
	},
	{
		name:        "guardrails-ci-admission",
		scenarioIDs: []string{"SCN-002", "SCN-006", "SCN-009"},
		evidenceIDs: []string{"EVID-001", "EVID-005", "EVID-008"},
		run: func(t *testing.T, repoRoot string) {
			runCommand(t, repoRoot, 2*time.Minute,
				"bash", "scripts/ci/required-guardrails-check.sh",
			)
		},
	},
	{
		name:        "drain-and-network-policy",
		scenarioIDs: []string{"SCN-004", "SCN-010", "SCN-013", "SCN-018", "SCN-019"},
		evidenceIDs: []string{"EVID-003", "EVID-006", "EVID-013", "EVID-014"},
		run: func(t *testing.T, repoRoot string) {
			runCommand(t, repoRoot, 3*time.Minute,
				"go", "test", "./cmd/service",
				"-run", "^(TestDrainAndShutdownOrdersDrainBeforeShutdown|TestNetworkPolicyEnforceIngressFailClosedWithoutException|TestNetworkPolicyEnforceIngressAllowsActiveException|TestNetworkPolicyEnforceEgressTargetDeniesPublicHost|TestNetworkPolicyEnforceEgressTargetDeniesSchemeOutsideAllowlist)$",
				"-count=1",
			)
		},
	},
	{
		name:        "workflow-concurrency-determinism",
		scenarioIDs: []string{"SCN-008"},
		evidenceIDs: []string{"EVID-001"},
		run: func(t *testing.T, repoRoot string) {
			ciWorkflow := readFile(t, filepath.Join(repoRoot, ".github/workflows/ci.yml"))
			requireContains(t, ciWorkflow, "group: ci-${{ github.workflow }}-${{ github.ref }}", "ci concurrency group by ref")
			requireContains(t, ciWorkflow, "cancel-in-progress: true", "ci must cancel in-progress runs for deterministic latest commit")

			cdWorkflow := readFile(t, filepath.Join(repoRoot, ".github/workflows/cd.yml"))
			requireContains(t, cdWorkflow, "github.event.workflow_run.conclusion == 'success'", "cd admission must require successful ci workflow_run")
			requireContains(t, cdWorkflow, "ref: ${{ github.event.workflow_run.head_sha }}", "cd must checkout ci head SHA for deterministic active revision")
		},
	},
	{
		name:        "capacity-and-threshold-packet",
		scenarioIDs: []string{"SCN-007", "SCN-011", "SCN-012"},
		evidenceIDs: []string{"EVID-004", "EVID-007"},
		run: func(t *testing.T, repoRoot string) {
			runCommand(t, repoRoot, 2*time.Minute,
				"go", "test", "./internal/infra/telemetry",
				"-run", "^(TestBuildReleaseReadinessEvidenceCapacityDegraded|TestBuildReleaseReadinessEvidenceNumericPolicyBlocked|TestPolicyThresholdInputFromRepositoryRailwayToml)$",
				"-count=1",
			)
		},
	},
	{
		name:        "deploy-rollback-drift-observability",
		scenarioIDs: []string{"SCN-014", "SCN-015", "SCN-016", "SCN-017"},
		evidenceIDs: []string{"EVID-009", "EVID-010", "EVID-011", "EVID-012"},
		run: func(t *testing.T, repoRoot string) {
			runCommand(t, repoRoot, 3*time.Minute,
				"go", "test", "./cmd/service",
				"-run", "^(TestDeployTelemetryRecorderRecordAdmissionEmitsLogAndMetrics|TestDeployTelemetryRecorderRecordRollbackIncludesCorrelation|TestDeployTelemetryRecorderRecordConfigDriftLifecycle|TestDeployTelemetryRecorderNetworkPolicySignals)$",
				"-count=1",
			)
			runCommand(t, repoRoot, 2*time.Minute,
				"go", "test", "./internal/infra/telemetry",
				"-run", "^(TestBuildReleaseReadinessEvidence|TestDeployRollbackAndDriftMetrics)$",
				"-count=1",
			)
		},
	},
}

func TestDeploymentResilienceScenarioEvidenceCoverageClosure(t *testing.T) {
	gotScenarios := map[string]struct{}{}
	gotEvidence := map[string]struct{}{}

	for _, executor := range deploymentResilienceExecutors {
		if len(executor.scenarioIDs) == 0 {
			t.Fatalf("executor %q has empty scenario coverage", executor.name)
		}
		if len(executor.evidenceIDs) == 0 {
			t.Fatalf("executor %q has empty evidence coverage", executor.name)
		}
		for _, scenarioID := range executor.scenarioIDs {
			gotScenarios[scenarioID] = struct{}{}
		}
		for _, evidenceID := range executor.evidenceIDs {
			gotEvidence[evidenceID] = struct{}{}
		}
	}

	assertExactIDSet(t, "scenario", gotScenarios, expectedIDSet("SCN-", 1, 19))
	assertExactIDSet(t, "evidence", gotEvidence, expectedIDSet("EVID-", 1, 14))
}

func TestDeploymentResilienceScenarioEvidenceExecutors(t *testing.T) {
	root := repositoryRoot(t)

	for _, executor := range deploymentResilienceExecutors {
		executor := executor
		t.Run(executor.name, func(t *testing.T) {
			executor.run(t, root)
		})
	}
}

func TestDeploymentResilienceNoOpCoverageDeclarations(t *testing.T) {
	root := repositoryRoot(t)
	testPlan := readFile(t, filepath.Join(root, "specs/railway-deployment-resilience/70-test-plan.md"))

	requireContains(t, testPlan, "1. No new product API contract in scope (`30` remains no-change).", "explicit API no-op declaration")
	requireContains(t, testPlan, "Status: no changes required.", "explicit data/cache no-op status declaration")
	requireContains(t, testPlan, "retain explicit no-op coverage reference to `40` under `TST-004`.", "explicit reference to no-op data/cache coverage")
}

func runCommand(t *testing.T, repoRoot string, timeout time.Duration, name string, args ...string) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = repoRoot
	output, err := cmd.CombinedOutput()

	if ctx.Err() == context.DeadlineExceeded {
		t.Fatalf("command timed out after %s: %s %s\noutput:\n%s", timeout, name, strings.Join(args, " "), string(output))
	}
	if err != nil {
		t.Fatalf("command failed: %s %s\nerror: %v\noutput:\n%s", name, strings.Join(args, " "), err, string(output))
	}
}

func repositoryRoot(t *testing.T) string {
	t.Helper()

	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}

	return filepath.Clean(filepath.Join(filepath.Dir(currentFile), ".."))
}

func readFile(t *testing.T, path string) string {
	t.Helper()

	raw, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		t.Fatalf("os.ReadFile(%q) error = %v", path, err)
	}
	return string(raw)
}

func requireContains(t *testing.T, text, needle, label string) {
	t.Helper()

	if !strings.Contains(text, needle) {
		t.Fatalf("missing %s: %q", label, needle)
	}
}

func expectedIDSet(prefix string, startInclusive, endInclusive int) map[string]struct{} {
	out := make(map[string]struct{}, endInclusive-startInclusive+1)
	for i := startInclusive; i <= endInclusive; i++ {
		id := fmt.Sprintf("%s%03d", prefix, i)
		out[id] = struct{}{}
	}
	return out
}

func assertExactIDSet(t *testing.T, kind string, got, want map[string]struct{}) {
	t.Helper()

	missing := make([]string, 0)
	extra := make([]string, 0)

	for id := range want {
		if _, ok := got[id]; !ok {
			missing = append(missing, id)
		}
	}
	for id := range got {
		if _, ok := want[id]; !ok {
			extra = append(extra, id)
		}
	}

	sort.Strings(missing)
	sort.Strings(extra)

	if len(missing) > 0 || len(extra) > 0 {
		t.Fatalf("%s set mismatch: missing=%v extra=%v", kind, missing, extra)
	}
}
