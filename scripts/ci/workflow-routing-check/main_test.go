package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateFamilyInputRejectsUnknownAndMistypedFields(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input map[string]any
		want  string
	}{
		{
			name: "unknown",
			input: map[string]any{
				"authorized": true, "required_lane": false, "substantive": false, "authorised": true,
			},
			want: "unknown input field",
		},
		{
			name: "mistyped",
			input: map[string]any{
				"authorized": "yes", "required_lane": false, "substantive": false,
			},
			want: "must be boolean",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := validateFamilyInput("agent_request", tt.input)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("validateFamilyInput() error = %v, want substring %q", err, tt.want)
			}
		})
	}
}

func TestRecordDeclaredCoverageRejectsCosmeticRuleClaim(t *testing.T) {
	t.Parallel()

	tc := testCase{ID: "cosmetic", Covers: []string{"FULL-DATA"}}
	executed := map[string]struct{}{"SHAPE-DIRECT": {}}
	err := recordDeclaredCoverage(tc, executed, map[string]string{})
	if err == nil || !strings.Contains(err.Error(), "without executing its rule-specific branch") {
		t.Fatalf("recordDeclaredCoverage() error = %v", err)
	}
}

func TestValidateFixtureMetadataRejectsCoverageOnExpectedError(t *testing.T) {
	t.Parallel()

	rules := map[string]string{"SHAPE-DIRECT": "AGENTS.md:1"}
	tc := testCase{
		ID:        "schema-error",
		Family:    "shape",
		Covers:    []string{"SHAPE-DIRECT"},
		Input:     map[string]any{"intake_accepted": "yes"},
		WantError: "must be boolean",
	}
	err := validateFixtureMetadata(rules, []testCase{tc})
	if err == nil || !strings.Contains(err.Error(), "cannot claim canonical rule coverage") {
		t.Fatalf("validateFixtureMetadata() error = %v", err)
	}
}

func TestVerifyExecutedCoverageRejectsMissingCanonicalRule(t *testing.T) {
	t.Parallel()

	rules := map[string]string{
		"SHAPE-DIRECT": "AGENTS.md:1",
		"SHAPE-LEAN":   "AGENTS.md:2",
	}
	err := verifyExecutedCoverage(rules, map[string]string{"SHAPE-DIRECT": "direct-case"})
	if err == nil || !strings.Contains(err.Error(), "SHAPE-LEAN") {
		t.Fatalf("verifyExecutedCoverage() error = %v", err)
	}
}

func TestValidateEvalManifestsRejectsMalformedManifests(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		body string
		want string
	}{
		{
			name: "unknown field",
			body: `{"skill_name":"example","evals":[{"id":1,"prompt":"p","expected_output":"o","files":[],"expectations":["e"],"surprise":true}]}`,
			want: "unknown field",
		},
		{
			name: "duplicate id",
			body: `{"skill_name":"example","evals":[{"id":1,"prompt":"p","expected_output":"o","files":[],"expectations":["e"]},{"id":1,"prompt":"p2","expected_output":"o2","files":[],"expectations":["e2"]}]}`,
			want: "duplicate eval id",
		},
		{
			name: "non integer id",
			body: `{"skill_name":"example","evals":[{"id":1.5,"prompt":"p","expected_output":"o","files":[],"expectations":["e"]}]}`,
			want: "cannot unmarshal number 1.5",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			path := filepath.Join(root, ".agents", "skills", "example", "evals", "evals.json")
			if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path, []byte(tt.body), 0o600); err != nil {
				t.Fatal(err)
			}
			_, err := validateEvalManifests(root)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("validateEvalManifests() error = %v, want substring %q", err, tt.want)
			}
		})
	}
}

func TestEvaluateStateTraceComesFromTypedFieldChecks(t *testing.T) {
	t.Parallel()

	input := map[string]any{
		"kind": "record", "execution_shape": "lean_local", "artifact_expectation": "expected",
		"artifact_state": "approved", "record_validity": "current", "phase_state": "complete",
		"procedural_gate_state": "complete", "review_verdict": "PASS", "subagent_gate": "complete",
		"waiver_disposition": "none", "session_boundary": "reached", "handoff_readiness": "ready",
		"routing_scope": "durable", "routing_revision": float64(1),
	}
	_, trace, err := evaluate(testCase{Family: "state", Input: input}, map[string]string{})
	if err != nil {
		t.Fatal(err)
	}
	for _, ruleID := range []string{
		"STATE-EXECUTION-SHAPE", "STATE-ARTIFACT-EXPECTATION", "STATE-ARTIFACT-LIFECYCLE",
		"STATE-RECORD-VALIDITY", "STATE-PHASE", "STATE-PROCEDURAL-GATE", "STATE-REVIEW-VERDICT",
		"STATE-SUBAGENT-GATE", "STATE-WAIVER", "STATE-SESSION-BOUNDARY", "STATE-HANDOFF",
		"STATE-ROUTING-SCOPE", "STATE-ROUTING-REVISION",
	} {
		if _, ok := trace[ruleID]; !ok {
			t.Errorf("missing evaluator-emitted trace for %s", ruleID)
		}
	}
}
