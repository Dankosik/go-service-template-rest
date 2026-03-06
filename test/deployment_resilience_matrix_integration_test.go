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
	name     string
	coverage []scenarioEvidenceCoverage
	run      func(t *testing.T, repoRoot string)
}

type scenarioEvidenceCoverage struct {
	scenarioID  string
	evidenceIDs []string
}

func coverage(scenarioID string, evidenceIDs ...string) scenarioEvidenceCoverage {
	return scenarioEvidenceCoverage{
		scenarioID:  scenarioID,
		evidenceIDs: evidenceIDs,
	}
}

var deploymentResilienceExecutors = []scenarioEvidenceExecutor{
	{
		name: "health-contract-readiness",
		coverage: []scenarioEvidenceCoverage{
			coverage("SCN-001", "EVID-002"),
			coverage("SCN-003", "EVID-002"),
			coverage("SCN-005", "EVID-002"),
			coverage("SCN-006", "EVID-002"),
		},
		run: func(t *testing.T, repoRoot string) {
			runCommand(t, repoRoot, 2*time.Minute,
				"go", "test", "./internal/app/health",
				"-run", "^(TestServiceReadySuccess|TestServiceReadyFail|TestServiceReadyDraining)$",
				"-count=1",
			)
			runCommand(t, repoRoot, 2*time.Minute,
				"go", "test", "./internal/infra/http",
				"-run", "^(TestOpenAPIRuntimeContractReadinessUnavailable|TestOpenAPIRuntimeContractReadinessUnavailableWhenDraining|TestOpenAPIRuntimeContractWrongHealthcheckPathRejected)$",
				"-count=1",
			)
		},
	},
	{
		name: "guardrails-ci-admission",
		coverage: []scenarioEvidenceCoverage{
			coverage("SCN-002", "EVID-001", "EVID-008"),
			coverage("SCN-009", "EVID-005", "EVID-011"),
		},
		run: func(t *testing.T, repoRoot string) {
			runCommand(t, repoRoot, 2*time.Minute,
				"bash", "scripts/ci/required-guardrails-check.sh",
			)
			runCommand(t, repoRoot, 2*time.Minute,
				"go", "test", "./cmd/service",
				"-run", "^(TestDeployTelemetryRecorderRecordConfigDriftLifecycle)$",
				"-count=1",
			)
			runCommand(t, repoRoot, 2*time.Minute,
				"go", "test", "./internal/infra/telemetry",
				"-run", "^(TestDeployRollbackAndDriftMetrics)$",
				"-count=1",
			)
		},
	},
	{
		name: "drain-and-network-policy",
		coverage: []scenarioEvidenceCoverage{
			coverage("SCN-004", "EVID-003"),
			coverage("SCN-010", "EVID-006"),
			coverage("SCN-013", "EVID-003"),
			coverage("SCN-018", "EVID-013"),
			coverage("SCN-019", "EVID-014"),
		},
		run: func(t *testing.T, repoRoot string) {
			runCommand(t, repoRoot, 3*time.Minute,
				"go", "test", "./cmd/service",
				"-run", "^(TestDrainAndShutdownOrdersDrainBeforeShutdown|TestNetworkPolicyEnforceIngressFailClosedWithoutException|TestNetworkPolicyEnforceIngressAllowsActiveException|TestNetworkPolicyEnforceIngressRejectsExpiredException|TestNetworkPolicyEnforceEgressTargetDeniesPublicHost|TestNetworkPolicyEnforceEgressTargetDeniesSchemeOutsideAllowlist|TestNetworkPolicyEmitEgressExceptionStateRejectsExpiredException)$",
				"-count=1",
			)
		},
	},
	{
		name: "workflow-concurrency-determinism",
		coverage: []scenarioEvidenceCoverage{
			coverage("SCN-008", "EVID-001"),
		},
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
		name: "capacity-and-threshold-packet",
		coverage: []scenarioEvidenceCoverage{
			coverage("SCN-007", "EVID-004"),
			coverage("SCN-011", "EVID-007"),
			coverage("SCN-012", "EVID-004", "EVID-007"),
			coverage("SCN-013", "EVID-007"),
		},
		run: func(t *testing.T, repoRoot string) {
			runCommand(t, repoRoot, 2*time.Minute,
				"go", "test", "./internal/infra/telemetry",
				"-run", "^(TestBuildReleaseReadinessEvidenceCapacityDegraded|TestBuildReleaseReadinessEvidenceNumericPolicyBlocked|TestPolicyThresholdInputFromRepositoryRailwayToml)$",
				"-count=1",
			)
		},
	},
	{
		name: "deploy-rollback-drift-observability",
		coverage: []scenarioEvidenceCoverage{
			coverage("SCN-014", "EVID-009"),
			coverage("SCN-015", "EVID-010"),
			coverage("SCN-016", "EVID-011"),
			coverage("SCN-017", "EVID-012"),
		},
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
	gotScenarioEvidence := map[string]map[string]struct{}{}

	for _, executor := range deploymentResilienceExecutors {
		if len(executor.coverage) == 0 {
			t.Fatalf("executor %q has empty scenario coverage", executor.name)
		}

		for _, covered := range executor.coverage {
			if strings.TrimSpace(covered.scenarioID) == "" {
				t.Fatalf("executor %q has empty scenario id", executor.name)
			}
			if len(covered.evidenceIDs) == 0 {
				t.Fatalf("executor %q has empty evidence coverage for scenario %q", executor.name, covered.scenarioID)
			}

			gotScenarios[covered.scenarioID] = struct{}{}
			if _, ok := gotScenarioEvidence[covered.scenarioID]; !ok {
				gotScenarioEvidence[covered.scenarioID] = map[string]struct{}{}
			}

			for _, evidenceID := range covered.evidenceIDs {
				gotEvidence[evidenceID] = struct{}{}
				gotScenarioEvidence[covered.scenarioID][evidenceID] = struct{}{}
			}
		}
	}

	assertExactIDSet(t, "scenario", gotScenarios, expectedIDSet("SCN-", 1, 19))
	assertExactIDSet(t, "evidence", gotEvidence, expectedIDSet("EVID-", 1, 14))
	assertExactScenarioEvidenceMap(t, gotScenarioEvidence, expectedScenarioEvidenceMap())
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
	if name == "go" && len(args) > 0 && args[0] == "test" && strings.Contains(string(output), "[no tests to run]") {
		t.Fatalf("command matched no tests: %s %s\noutput:\n%s", name, strings.Join(args, " "), string(output))
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

func expectedScenarioEvidenceMap() map[string]map[string]struct{} {
	return map[string]map[string]struct{}{
		"SCN-001": {"EVID-002": {}},
		"SCN-002": {"EVID-001": {}, "EVID-008": {}},
		"SCN-003": {"EVID-002": {}},
		"SCN-004": {"EVID-003": {}},
		"SCN-005": {"EVID-002": {}},
		"SCN-006": {"EVID-002": {}},
		"SCN-007": {"EVID-004": {}},
		"SCN-008": {"EVID-001": {}},
		"SCN-009": {"EVID-005": {}, "EVID-011": {}},
		"SCN-010": {"EVID-006": {}},
		"SCN-011": {"EVID-007": {}},
		"SCN-012": {"EVID-004": {}, "EVID-007": {}},
		"SCN-013": {"EVID-003": {}, "EVID-007": {}},
		"SCN-014": {"EVID-009": {}},
		"SCN-015": {"EVID-010": {}},
		"SCN-016": {"EVID-011": {}},
		"SCN-017": {"EVID-012": {}},
		"SCN-018": {"EVID-013": {}},
		"SCN-019": {"EVID-014": {}},
	}
}

func assertExactScenarioEvidenceMap(t *testing.T, got, want map[string]map[string]struct{}) {
	t.Helper()

	missingScenarios := make([]string, 0)
	extraScenarios := make([]string, 0)
	mismatchScenarios := make([]string, 0)

	for scenarioID, expectedEvidence := range want {
		actualEvidence, ok := got[scenarioID]
		if !ok {
			missingScenarios = append(missingScenarios, scenarioID)
			continue
		}

		missingEvidence := make([]string, 0)
		extraEvidence := make([]string, 0)

		for evidenceID := range expectedEvidence {
			if _, ok := actualEvidence[evidenceID]; !ok {
				missingEvidence = append(missingEvidence, evidenceID)
			}
		}
		for evidenceID := range actualEvidence {
			if _, ok := expectedEvidence[evidenceID]; !ok {
				extraEvidence = append(extraEvidence, evidenceID)
			}
		}

		sort.Strings(missingEvidence)
		sort.Strings(extraEvidence)
		if len(missingEvidence) > 0 || len(extraEvidence) > 0 {
			mismatchScenarios = append(mismatchScenarios, fmt.Sprintf("%s(missing=%v extra=%v)", scenarioID, missingEvidence, extraEvidence))
		}
	}

	for scenarioID := range got {
		if _, ok := want[scenarioID]; !ok {
			extraScenarios = append(extraScenarios, scenarioID)
		}
	}

	sort.Strings(missingScenarios)
	sort.Strings(extraScenarios)
	sort.Strings(mismatchScenarios)
	if len(missingScenarios) > 0 || len(extraScenarios) > 0 || len(mismatchScenarios) > 0 {
		t.Fatalf("scenario-evidence map mismatch: missing_scenarios=%v extra_scenarios=%v mismatches=%v", missingScenarios, extraScenarios, mismatchScenarios)
	}
}
