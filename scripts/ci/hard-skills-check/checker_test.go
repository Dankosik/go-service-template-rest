package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
)

func TestInventoryContract(t *testing.T) {
	t.Parallel()
	if len(targetSkills) != 34 {
		t.Fatalf("target inventory has %d skills, want 34", len(targetSkills))
	}
	wantRenames := map[string]string{
		"go-api-contract-spec":               "api-contract-designer-spec",
		"go-system-architecture-spec":        "go-architect-spec",
		"go-data-architecture-spec":          "go-data-architect-spec",
		"go-implementation-ownership-spec":   "go-design-spec",
		"go-implementation-ownership-review": "go-design-review",
		"go-distributed-spec":                "go-distributed-architect-spec",
		"go-observability-spec":              "go-observability-engineer-spec",
		"go-delivery-platform-spec":          "go-devops-spec",
		"go-delivery-platform-review":        "go-devops-review",
		"go-test-design":                     "go-qa-tester-spec",
		"go-test-implementation":             "go-qa-tester",
		"go-test-review":                     "go-qa-review",
	}
	if !maps.Equal(renamedSkills, wantRenames) {
		t.Fatalf("rename mapping = %#v, want %#v", renamedSkills, wantRenames)
	}
	wantSelected := []string{"go-reliability-review", "go-security-spec", "go-observability-review", "go-coder", "go-verification-before-completion", "go-security-review", "go-implementation-ownership-review", "go-systematic-debugging", "go-specialist-router"}
	if !slices.Equal(selectedSkills, wantSelected) {
		t.Fatalf("selected skills = %v, want %v", selectedSkills, wantSelected)
	}
}

func TestStage2MetadataIsClosedAndDormant(t *testing.T) {
	t.Parallel()
	prompt := strings.Repeat("a", 64)
	commit := strings.Repeat("b", 40)
	valid := fmt.Sprintf(`{"schema_version":1,"prompt_sha256":%q,"repository_commit":%q,"routing_mode":"implicit","explicit_skill_mentions":[],"forced_skills":[],"selected_skills":[{"name":"go-specialist-router","source":"implicit"}],"provenance_source":"runtime_events"}`, prompt, commit)
	if err := validateStage2Metadata([]byte(valid), prompt, commit, map[string]bool{"go-security-review": true}); err != nil {
		t.Fatal(err)
	}
	tests := []struct{ name, metadata string }{
		{"wrong digest", strings.Replace(valid, prompt, strings.Repeat("c", 64), 1)},
		{"wrong commit", strings.Replace(valid, commit, strings.Repeat("c", 40), 1)},
		{"wrong mode", strings.Replace(valid, `"routing_mode":"implicit"`, `"routing_mode":"explicit"`, 1)},
		{"explicit list", strings.Replace(valid, `"explicit_skill_mentions":[]`, `"explicit_skill_mentions":["go-specialist-router"]`, 1)},
		{"forced list", strings.Replace(valid, `"forced_skills":[]`, `"forced_skills":["go-specialist-router"]`, 1)},
		{"omitted router", strings.Replace(valid, `[{"name":"go-specialist-router","source":"implicit"}]`, `[{"name":"agent-prompt-composer","source":"implicit"}]`, 1)},
		{"explicit-only specialist", strings.Replace(valid, `"go-specialist-router"`, `"go-security-review"`, 1)},
		{"non-implicit source", strings.Replace(valid, `"source":"implicit"`, `"source":"explicit"`, 1)},
		{"duplicate selection", strings.Replace(valid, `[{"name":"go-specialist-router","source":"implicit"}]`, `[{"name":"go-specialist-router","source":"implicit"},{"name":"go-specialist-router","source":"implicit"}]`, 1)},
		{"empty selection", strings.Replace(valid, `[{"name":"go-specialist-router","source":"implicit"}]`, `[]`, 1)},
		{"unknown key", strings.Replace(valid, `"runtime_events"}`, `"runtime_events","extra":true}`, 1)},
		{"non-runtime provenance", strings.Replace(valid, `"runtime_events"`, `"adapter_claim"`, 1)},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := validateStage2Metadata([]byte(test.metadata), prompt, commit, map[string]bool{"go-security-review": true}); err == nil {
				t.Fatal("invalid stage 2 metadata passed")
			}
		})
	}
}

func TestCatalogAndPolicyMutations(t *testing.T) {
	t.Run("non-hard name mismatch", func(t *testing.T) {
		root := makeValidRepository(t)
		writeSkill(t, root, "grilling", "wrong", validDescription(), "# Body\n")
		if got := strings.Join(validateCatalog(root), "\n"); !strings.Contains(got, "frontmatter name \"wrong\" does not match directory") {
			t.Fatalf("catalog mismatch = %s", got)
		}
	})
	for _, test := range []struct{ name, path, body string }{
		{"nested selector link", ".agents/skills/go-chi-review/references/index.md", "[missing](missing.md)\n"},
		{"router arbitration link", ".agents/skills/go-specialist-router/references/index.md", "[arbitration](../../missing-arbitration.md)\n"},
	} {
		t.Run(test.name, func(t *testing.T) {
			root := makeValidRepository(t)
			mustWriteFile(t, filepath.Join(root, filepath.FromSlash(test.path)), []byte(test.body))
			if got := strings.Join(validateCatalog(root), "\n"); !strings.Contains(got, "unresolved local Markdown link") {
				t.Fatalf("broken link passed: %s", got)
			}
		})
	}
	makePolicies := func(t *testing.T) (string, []string) {
		root := makeValidRepository(t)
		names, problems := specialistNames(root)
		if len(problems) != 0 || len(names) != 29 {
			t.Fatalf("derived specialists = %d, %v", len(names), problems)
		}
		for _, name := range names {
			mustWriteFile(t, filepath.Join(root, ".agents/skills", name, "agents/openai.yaml"), []byte("policy:\n  allow_implicit_invocation: false\n"))
		}
		return root, names
	}
	t.Run("zero policies", func(t *testing.T) {
		if got := validateImplicitPolicies(makeValidRepository(t)); len(got) != 0 {
			t.Fatal(got)
		}
	})
	t.Run("exact derived policies", func(t *testing.T) {
		root, _ := makePolicies(t)
		if got := validateImplicitPolicies(root); len(got) != 0 {
			t.Fatal(got)
		}
	})
	for _, test := range []struct {
		name   string
		mutate func(*testing.T, string, []string)
	}{
		{"partial policies", func(t *testing.T, root string, names []string) {
			mustRemoveAll(t, filepath.Join(root, ".agents/skills", names[0], "agents"))
		}},
		{"extra non-specialist", func(t *testing.T, root string, _ []string) {
			writeSkill(t, root, "grilling", "grilling", validDescription(), "# Body\n")
			mustWriteFile(t, filepath.Join(root, ".agents/skills/grilling/agents/openai.yaml"), []byte("policy:\n  allow_implicit_invocation: false\n"))
		}},
		{"router false", func(t *testing.T, root string, _ []string) {
			mustWriteFile(t, filepath.Join(root, ".agents/skills/go-specialist-router/agents/openai.yaml"), []byte("policy:\n  allow_implicit_invocation: false\n"))
		}},
		{"malformed policy", func(t *testing.T, root string, names []string) {
			mustWriteFile(t, filepath.Join(root, ".agents/skills", names[0], "agents/openai.yaml"), []byte("policy:\n  allow_implicit_invocation: true\n"))
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			root, names := makePolicies(t)
			test.mutate(t, root, names)
			if got := validateImplicitPolicies(root); len(got) == 0 {
				t.Fatal("invalid policy state passed")
			}
		})
	}
}

func TestTrialClassMatrixAndManifestStability(t *testing.T) {
	t.Parallel()
	root := makeValidRepository(t)
	bundle := validEvalBundle("go-reliability-review")
	if problems := validateEvalBundle(bundle.SkillName, bundle); len(problems) != 0 {
		t.Fatalf("empty core trial_class rejected: %v", problems)
	}
	aux := bundle.Evals[0]
	aux.ID = json.RawMessage(`"aux"`)
	aux.TrialClass = "standard"
	bundle.Evals = append(bundle.Evals, aux)
	if problems := validateEvalBundle(bundle.SkillName, bundle); len(problems) != 0 {
		t.Fatalf("standard rejected: %v", problems)
	}
	writeEvalBundle(t, root, bundle.SkillName, bundle)
	first := filepath.Join(t.TempDir(), "standard")
	if err := emitSelectedEvals(root, first); err != nil {
		t.Fatal(err)
	}
	bundle.Evals[len(bundle.Evals)-1].TrialClass = "safety_authority"
	if problems := validateEvalBundle(bundle.SkillName, bundle); len(problems) != 0 {
		t.Fatalf("safety_authority rejected: %v", problems)
	}
	writeEvalBundle(t, root, bundle.SkillName, bundle)
	second := filepath.Join(t.TempDir(), "safety")
	if err := emitSelectedEvals(root, second); err != nil {
		t.Fatal(err)
	}
	if got, want := snapshotTree(t, second), snapshotTree(t, first); got != want {
		t.Fatal("allowed auxiliary trial_class changed 36-core emission")
	}
	bundle.Evals[len(bundle.Evals)-1].TrialClass = "auxiliary"
	if got := strings.Join(validateEvalBundle(bundle.SkillName, bundle), "\n"); !strings.Contains(got, "invalid trial_class") {
		t.Fatalf("arbitrary trial_class passed: %s", got)
	}
}

func TestCheckRepositoryValidation(t *testing.T) {
	t.Parallel()

	t.Run("valid", func(t *testing.T) {
		root := makeValidRepository(t)
		if err := checkRepository(root, targetSkills, false); err != nil {
			t.Fatalf("valid repository failed: %v", err)
		}
	})

	tests := []struct {
		name   string
		mutate func(*testing.T, string)
		want   string
	}{
		{
			name: "missing inventory entry",
			mutate: func(t *testing.T, root string) {
				t.Helper()
				mustRemoveAll(t, filepath.Join(root, ".agents/skills/go-chi-review"))
			},
			want: "missing canonical hard skill: go-chi-review",
		},
		{
			name: "unknown hard skill",
			mutate: func(t *testing.T, root string) {
				t.Helper()
				mustMkdirAll(t, filepath.Join(root, ".agents/skills/go-unknown"))
			},
			want: "unknown canonical hard-skill directory: go-unknown",
		},
		{
			name: "frontmatter mismatch",
			mutate: func(t *testing.T, root string) {
				t.Helper()
				writeSkill(t, root, "go-coder", "wrong-name", validDescription(), "# Body\n")
			},
			want: "frontmatter name \"wrong-name\" does not match directory",
		},
		{
			name: "duplicate frontmatter name",
			mutate: func(t *testing.T, root string) {
				t.Helper()
				path := skillPath(root, "go-coder")
				mustWriteFile(t, path, []byte("---\nname: wrong-name\nname: go-coder\ndescription: \"Use when testing. Own behavior; Skip when outside scope.\"\n---\n# Body\n"))
			},
			want: "frontmatter must contain exactly one one-line name, got 2",
		},
		{
			name: "unquoted description",
			mutate: func(t *testing.T, root string) {
				t.Helper()
				path := skillPath(root, "go-coder")
				mustWriteFile(t, path, []byte("---\nname: go-coder\ndescription: Use when testing. Own behavior; Skip when outside scope.\n---\n# Body\n"))
			},
			want: "description must be a quoted one-line value",
		},
		{
			name: "third description sentence",
			mutate: func(t *testing.T, root string) {
				t.Helper()
				writeSkill(t, root, "go-coder", "go-coder", "Use when testing. Own behavior. Skip when outside scope.", "# Body\n")
			},
			want: "description exceeds two sentences",
		},
		{
			name: "missing routing clause",
			mutate: func(t *testing.T, root string) {
				t.Helper()
				writeSkill(t, root, "go-coder", "go-coder", "Use when testing. Own behavior.", "# Body\n")
			},
			want: "description is missing \"Skip when\"",
		},
		{
			name: "missing shared link",
			mutate: func(t *testing.T, root string) {
				t.Helper()
				writeSkill(t, root, "go-chi-review", "go-chi-review", validDescription(), "# Body\n")
			},
			want: "missing direct link",
		},
		{
			name: "triggerable shared contract",
			mutate: func(t *testing.T, root string) {
				t.Helper()
				mustWriteFile(t, filepath.Join(root, ".agents/skills/specialist-contract/SKILL.md"), []byte("triggerable\n"))
			},
			want: "must not be a triggerable SKILL.md",
		},
		{
			name: "broken markdown link",
			mutate: func(t *testing.T, root string) {
				t.Helper()
				body := "# Body\n\n[contract](../specialist-contract.md)\n[missing](references/missing.md#part)\n"
				writeSkill(t, root, "go-chi-review", "go-chi-review", validDescription(), body)
			},
			want: "unresolved local Markdown link",
		},
		{
			name: "stale eval skill name",
			mutate: func(t *testing.T, root string) {
				t.Helper()
				bundle := validEvalBundle("go-chi-review")
				bundle.SkillName = "go-chi-spec"
				writeEvalBundle(t, root, "go-chi-review", bundle)
			},
			want: "eval skill_name is \"go-chi-spec\"",
		},
		{
			name: "duplicate eval id",
			mutate: func(t *testing.T, root string) {
				t.Helper()
				bundle := validEvalBundle("go-chi-review")
				bundle.Evals[1].ID = bundle.Evals[0].ID
				writeEvalBundle(t, root, "go-chi-review", bundle)
			},
			want: "duplicate id",
		},
		{
			name: "missing category",
			mutate: func(t *testing.T, root string) {
				t.Helper()
				bundle := validEvalBundle("go-chi-review")
				bundle.Evals = bundle.Evals[:3]
				writeEvalBundle(t, root, "go-chi-review", bundle)
			},
			want: "missing eval category unresolved_policy",
		},
		{
			name: "empty prompt",
			mutate: func(t *testing.T, root string) {
				t.Helper()
				bundle := validEvalBundle("go-chi-review")
				bundle.Evals[0].Prompt = " "
				writeEvalBundle(t, root, "go-chi-review", bundle)
			},
			want: "prompt is empty",
		},
		{
			name: "selected file input",
			mutate: func(t *testing.T, root string) {
				t.Helper()
				bundle := validEvalBundle("go-reliability-review")
				bundle.Evals[0].Files = []string{"inputs/code.go"}
				writeEvalBundle(t, root, "go-reliability-review", bundle)
			},
			want: "selected eval must be self-contained",
		},
		{
			name: "retired canonical directory",
			mutate: func(t *testing.T, root string) {
				t.Helper()
				mustMkdirAll(t, filepath.Join(root, ".agents/skills/go-design-spec"))
			},
			want: "retired canonical skill path exists: go-design-spec",
		},
		{
			name: "retired authority identifier",
			mutate: func(t *testing.T, root string) {
				t.Helper()
				mustWriteFile(t, filepath.Join(root, "docs/runtime.md"), []byte("use go-design-spec\n"))
			},
			want: "retired skill identifier \"go-design-spec\" remains in docs/runtime.md",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			root := makeValidRepository(t)
			test.mutate(t, root)
			err := checkRepository(root, targetSkills, false)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error = %v, want substring %q", err, test.want)
			}
		})
	}
}

func TestScopedCheckIsolationAndSelection(t *testing.T) {
	t.Parallel()
	root := makeValidRepository(t)
	broken := validEvalBundle("go-security-review")
	broken.Evals = broken.Evals[:3]
	writeEvalBundle(t, root, "go-security-review", broken)

	if err := checkRepository(root, []string{"go-reliability-review"}, true); err != nil {
		t.Fatalf("unrelated sibling escaped scoped isolation: %v", err)
	}
	if err := checkRepository(root, targetSkills, false); err == nil || !strings.Contains(err.Error(), "go-security-review") {
		t.Fatalf("global check did not find sibling defect: %v", err)
	}

	writeSkill(t, root, "go-reliability-review", "go-reliability-review", validDescription(), "# No contract\n")
	if err := checkRepository(root, []string{"go-reliability-review"}, true); err == nil || !strings.Contains(err.Error(), "missing direct link") {
		t.Fatalf("scoped check hid selected defect: %v", err)
	}

	for _, test := range []struct {
		raw  string
		want string
	}{
		{"", "must not be empty"},
		{"go-coder,", "empty or non-exact"},
		{"go-coder, go-chi-review", "empty or non-exact"},
		{"go-coder,go-coder", "duplicate"},
		{"go-design-spec", "retired"},
		{"go-not-real", "unknown"},
	} {
		_, err := parseSkillScope(test.raw)
		if err == nil || !strings.Contains(err.Error(), test.want) {
			t.Errorf("parseSkillScope(%q) error = %v, want %q", test.raw, err, test.want)
		}
	}
	want := []string{"go-coder", "go-chi-review"}
	got, err := parseSkillScope(strings.Join(want, ","))
	if err != nil || !slices.Equal(got, want) {
		t.Fatalf("valid scope = %v, %v", got, err)
	}
}

func TestRunCLICommands(t *testing.T) {
	root := makeValidRepository(t)
	t.Setenv("HARD_SKILLS_REPO_ROOT", root)
	output, err := os.Create(filepath.Join(t.TempDir(), "output.txt"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = output.Close() })

	tests := []struct {
		name    string
		args    []string
		wantErr string
	}{
		{name: "default check"},
		{name: "explicit check", args: []string{"check"}},
		{name: "scoped check", args: []string{"check", "--skills", "go-coder"}},
		{name: "check flag error", args: []string{"check", "--unknown"}, wantErr: "parse check flags"},
		{name: "check positional argument", args: []string{"check", "extra"}, wantErr: "accepts no positional arguments"},
		{name: "emit requires output", args: []string{"emit-selected-evals"}, wantErr: "--output-dir is required"},
		{name: "emit positional argument", args: []string{"emit-selected-evals", "extra"}, wantErr: "accepts no positional arguments"},
		{name: "emit selected evals", args: []string{"emit-selected-evals", "--output-dir", filepath.Join(t.TempDir(), "selected")}},
		{name: "size requires baseline", args: []string{"size-report"}, wantErr: "--baseline-ref is required"},
		{name: "size positional argument", args: []string{"size-report", "extra"}, wantErr: "accepts no positional arguments"},
		{name: "unknown command", args: []string{"unknown"}, wantErr: "usage:"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := runCLI(test.args, output)
			if test.wantErr == "" {
				if err != nil {
					t.Fatalf("runCLI(%q) error = %v", test.args, err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), test.wantErr) {
				t.Fatalf("runCLI(%q) error = %v, want %q", test.args, err, test.wantErr)
			}
		})
	}
}

func TestEmitSelectedEvalsDeterministically(t *testing.T) {
	t.Parallel()
	root := makeValidRepository(t)
	first := filepath.Join(t.TempDir(), "first")
	second := filepath.Join(t.TempDir(), "second")
	if err := emitSelectedEvals(root, first); err != nil {
		t.Fatal(err)
	}
	if err := emitSelectedEvals(root, second); err != nil {
		t.Fatal(err)
	}
	if got, want := snapshotTree(t, first), snapshotTree(t, second); got != want {
		t.Fatalf("emission is not deterministic:\nfirst:\n%s\nsecond:\n%s", got, want)
	}

	manifest := mustReadFile(t, filepath.Join(first, "manifest.tsv"))
	lines := strings.Split(strings.TrimSpace(string(manifest)), "\n")
	if len(lines) != 36 {
		t.Fatalf("manifest has %d cases, want 36", len(lines))
	}
	for index, line := range lines {
		fields := strings.Split(line, "\t")
		if len(fields) != 5 || fields[0] != fields[1]+":"+fields[2] {
			t.Fatalf("manifest line %d malformed: %q", index, line)
		}
	}
	expectedPath := filepath.Join(first, "cases/go-reliability-review/domain_defect/expected.txt")
	if got, want := string(mustReadFile(t, expectedPath)), "expected go-reliability-review domain_defect\nassert go-reliability-review domain_defect\n"; got != want {
		t.Fatalf("expected payload = %q, want %q", got, want)
	}

	bundle := validEvalBundle("go-security-spec")
	bundle.Evals[0].Category = "clean"
	writeEvalBundle(t, root, "go-security-spec", bundle)
	if err := emitSelectedEvals(root, filepath.Join(t.TempDir(), "bad")); err == nil {
		t.Fatal("selected category mismatch unexpectedly emitted")
	}
}

func TestSizeReportAccounting(t *testing.T) {
	t.Parallel()
	root, baseline := makeSizeRepository(t)
	var output bytes.Buffer
	if err := writeSizeReport(root, baseline, &output); err != nil {
		t.Fatal(err)
	}
	text := output.String()
	if !strings.Contains(text, "tokens_ceil_bytes_div_4") {
		t.Fatalf("token heuristic is not labelled:\n%s", text)
	}
	rows := parseSizeRows(t, text)
	skills, err := catalogSkillNames(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, skill := range skills {
		if _, ok := rows[skill]; !ok {
			t.Errorf("per-skill row missing: %s", skill)
		}
	}
	for _, family := range []string{"shared-contract-specialist", "direct-execution-method", "specialist-router", "workflow-session"} {
		row, ok := rows["FAMILY:"+family]
		if !ok || row[1] != family {
			t.Errorf("family row missing or wrong: %s", family)
		}
	}
	familyCount := 0
	for name := range rows {
		if strings.HasPrefix(name, "FAMILY:") {
			familyCount++
		}
	}
	if familyCount != 4 {
		t.Fatalf("family row count = %d, want 4", familyCount)
	}
	api := rows["go-api-contract-spec"]
	if api[2] != "api-contract-designer-spec" {
		t.Fatalf("rename mapping = %q", api[2])
	}
	reliability := rows["go-reliability-review"]
	sharedBytes := len([]byte("shared contract words\n"))
	assertEffectiveBytes(t, "specialist", reliability, mustAtoi(t, reliability[9])+sharedBytes)
	router := rows["go-specialist-router"]
	assertEffectiveBytes(t, "router", router, mustAtoi(t, router[9])+sharedBytes)
	assertEffectiveBytes(t, "direct execution", rows["go-coder"], mustAtoi(t, rows["go-coder"][9]))
	assertEffectiveBytes(t, "workflow/session", rows["planning-session"], mustAtoi(t, rows["planning-session"][9]))
	if _, ok := rows["CATALOG"]; !ok {
		t.Fatal("aggregate row missing")
	}
	if err := writeSizeReport(root, "missing", &bytes.Buffer{}); err == nil {
		t.Fatal("missing baseline ref unexpectedly passed")
	}
}

func assertEffectiveBytes(t *testing.T, label string, row []string, want int) {
	t.Helper()
	if got := mustAtoi(t, row[17]); got != want {
		t.Fatalf("%s effective bytes = %d, want %d", label, got, want)
	}
}

func TestHarnessManifestAndRequiredInputs(t *testing.T) {
	t.Parallel()
	fixture := makeHarnessFixture(t)
	output, err := runHarness(t, fixture, []string{"check"}, nil)
	if err != nil || !strings.Contains(output, "36 selected cases") {
		t.Fatalf("manifest check: %v\n%s", err, output)
	}

	tests := []struct {
		name string
		env  map[string]string
		want string
	}{
		{"missing baseline", map[string]string{}, "requires explicit WORKFLOW_EVAL_BASE_REF"},
		{"symbolic baseline", map[string]string{"WORKFLOW_EVAL_BASE_REF": "HEAD"}, "resolved immutable commit"},
		{"missing runner", map[string]string{"WORKFLOW_EVAL_BASE_REF": fixture.baseline}, "requires executable WORKFLOW_EVAL_RUNNER"},
		{"missing judge", map[string]string{"WORKFLOW_EVAL_BASE_REF": fixture.baseline, "WORKFLOW_EVAL_RUNNER": fixture.runner}, "requires executable WORKFLOW_EVAL_JUDGE"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			output, err := runHarness(t, fixture, []string{"run", filepath.Join(t.TempDir(), "artifacts")}, test.env)
			if err == nil || !strings.Contains(output, test.want) {
				t.Fatalf("error = %v, output = %s, want %q", err, output, test.want)
			}
		})
	}
}

func TestHarnessCheckOnlyAndCostAuthorizationMakeNoAdapterCalls(t *testing.T) {
	fixture := makeHarnessFixture(t)
	env := fixture.runEnv()
	if output, err := runHarness(t, fixture, []string{"check"}, env); err != nil {
		t.Fatalf("check failed: %v\n%s", err, output)
	}
	assertNoAdapterCalls(t, fixture)
	for _, test := range []struct {
		name  string
		value *string
	}{
		{"missing", nil}, {"false", func() *string { value := "false"; return &value }()},
	} {
		t.Run(test.name, func(t *testing.T) {
			fixture := makeHarnessFixture(t)
			env := fixture.runEnv()
			if test.value == nil {
				delete(env, "WORKFLOW_EVAL_COST_AUTHORIZED")
			} else {
				env["WORKFLOW_EVAL_COST_AUTHORIZED"] = *test.value
			}
			output, err := runHarness(t, fixture, []string{"run", filepath.Join(t.TempDir(), "artifacts")}, env)
			if err == nil || !strings.Contains(output, "WORKFLOW_EVAL_COST_AUTHORIZED=true before adapters") {
				t.Fatalf("authorization error = %v\n%s", err, output)
			}
			assertNoAdapterCalls(t, fixture)
		})
	}
}

func assertNoAdapterCalls(t *testing.T, fixture harnessFixture) {
	t.Helper()
	if data, err := os.ReadFile(fixture.runnerLog); err == nil && len(bytes.TrimSpace(data)) != 0 {
		t.Fatalf("runner called: %s", data)
	} else if err != nil && !errors.Is(err, fs.ErrNotExist) {
		t.Fatal(err)
	}
	if data, err := os.ReadFile(fixture.judgeCount); err == nil && len(bytes.TrimSpace(data)) != 0 {
		t.Fatalf("judge called: %s", data)
	} else if err != nil && !errors.Is(err, fs.ErrNotExist) {
		t.Fatal(err)
	}
}

func TestHarnessIsolationAndAdapterSymmetry(t *testing.T) {
	t.Parallel()
	fixture := makeHarnessFixture(t)
	artifactDir := filepath.Join(t.TempDir(), "artifacts")
	env := fixture.runEnv()
	env["FAKE_REQUIRE_CANDIDATE_FILE"] = ".agents/skills/go-reliability-review/references/untracked-authority.md"
	env["FAKE_FORBID_FILE"] = "unrelated-sentinel.txt"
	env["FAKE_ASSERT_SEALED"] = "true"
	output, err := runHarness(t, fixture, []string{"run", artifactDir}, env)
	if err != nil {
		t.Fatalf("run failed: %v\n%s", err, output)
	}
	logLines := nonEmptyLines(string(mustReadFile(t, fixture.runnerLog)))
	if len(logLines) != 72 {
		t.Fatalf("runner calls = %d, want 72", len(logLines))
	}
	for index := 0; index < len(logLines); index += 2 {
		baseline := strings.Split(logLines[index], "\t")
		candidate := strings.Split(logLines[index+1], "\t")
		if len(baseline) != 2 || len(candidate) != 2 || baseline[0] != "baseline" || candidate[0] != "candidate" || baseline[1] != candidate[1] {
			t.Fatalf("asymmetric pair: %q / %q", logLines[index], logLines[index+1])
		}
	}
	summary := string(mustReadFile(t, filepath.Join(artifactDir, "summary.tsv")))
	if strings.Count(strings.TrimSpace(summary), "\n") != 36 || strings.Contains(summary, "\tfalse\n") {
		t.Fatalf("unexpected summary:\n%s", summary)
	}
	runEnv := string(mustReadFile(t, filepath.Join(artifactDir, "run.env")))
	if !strings.Contains(runEnv, "baseline_commit="+fixture.baseline) || !strings.Contains(runEnv, "candidate_snapshot_commit=") {
		t.Fatalf("snapshot metadata missing:\n%s", runEnv)
	}
}

func TestHarnessDetectsBaselineAndCandidateMutation(t *testing.T) {
	t.Parallel()
	for _, variant := range []string{"baseline", "candidate"} {
		t.Run(variant, func(t *testing.T) {
			fixture := makeHarnessFixture(t)
			env := fixture.runEnv()
			env["FAKE_MUTATE_VARIANT"] = variant
			output, err := runHarness(t, fixture, []string{"run", filepath.Join(t.TempDir(), "artifacts")}, env)
			if err == nil || !strings.Contains(output, "adapter mutated") || !strings.Contains(output, variant+" snapshot") {
				t.Fatalf("mutation not detected: %v\n%s", err, output)
			}
		})
	}
}

func TestHarnessDetectsPromptAndExpectedPayloadMutation(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		mutate string
		want   string
	}{
		{name: "baseline runner prompt", mutate: "baseline:prompt", want: "prompt payload"},
		{name: "candidate runner prompt", mutate: "candidate:prompt", want: "prompt payload"},
		{name: "judge prompt", mutate: "judge:prompt", want: "prompt payload"},
		{name: "judge expected", mutate: "judge:expected", want: "expected payload"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			fixture := makeHarnessFixture(t)
			env := fixture.runEnv()
			env["FAKE_MUTATE_PAYLOAD_ON"] = test.mutate
			output, err := runHarness(t, fixture, []string{"run", filepath.Join(t.TempDir(), "artifacts")}, env)
			if err == nil || !strings.Contains(output, "adapter mutated") || !strings.Contains(output, test.want) {
				t.Fatalf("payload mutation %q not detected: %v\n%s", test.mutate, err, output)
			}
		})
	}
}

func TestHarnessRejectsJudgeFailure(t *testing.T) {
	t.Parallel()
	fixture := makeHarnessFixture(t)
	env := fixture.runEnv()
	env["FAKE_JUDGE_MODE"] = "fail"
	output, err := runHarness(t, fixture, []string{"run", filepath.Join(t.TempDir(), "artifacts")}, env)
	if err == nil || !strings.Contains(output, "judge failed") {
		t.Fatalf("judge failure not propagated: %v\n%s", err, output)
	}
}

func TestHarnessRepeatPolicyAndCap(t *testing.T) {
	t.Parallel()
	t.Run("one disputed repeat", func(t *testing.T) {
		fixture := makeHarnessFixture(t)
		env := fixture.runEnv()
		env["FAKE_JUDGE_MODE"] = "disputed_once"
		artifactDir := filepath.Join(t.TempDir(), "artifacts")
		output, err := runHarness(t, fixture, []string{"run", artifactDir}, env)
		if err != nil {
			t.Fatalf("repeat run failed: %v\n%s", err, output)
		}
		if got := len(nonEmptyLines(string(mustReadFile(t, fixture.runnerLog)))); got != 74 {
			t.Fatalf("runner calls = %d, want 74", got)
		}
		summary := nonEmptyLines(string(mustReadFile(t, filepath.Join(artifactDir, "summary.tsv"))))
		if !strings.Contains(summary[1], "\t2\t") {
			t.Fatalf("first case did not record two attempts: %q", summary[1])
		}
	})

	t.Run("uncertainty cannot trigger third pair", func(t *testing.T) {
		fixture := makeHarnessFixture(t)
		env := fixture.runEnv()
		env["FAKE_JUDGE_MODE"] = "uncertain_twice"
		output, err := runHarness(t, fixture, []string{"run", filepath.Join(t.TempDir(), "artifacts")}, env)
		if err == nil || !strings.Contains(output, "after one repeat") {
			t.Fatalf("persistent uncertainty did not fail: %v\n%s", err, output)
		}
		if got := len(nonEmptyLines(string(mustReadFile(t, fixture.runnerLog)))); got != 4 {
			t.Fatalf("persistent uncertainty runner calls = %d, want capped 4", got)
		}
	})
}

type harnessFixture struct {
	root       string
	checker    string
	runner     string
	judge      string
	baseline   string
	runnerLog  string
	judgeCount string
}

func (fixture harnessFixture) runEnv() map[string]string {
	return map[string]string{
		"WORKFLOW_EVAL_BASE_REF":        fixture.baseline,
		"WORKFLOW_EVAL_RUNNER":          fixture.runner,
		"WORKFLOW_EVAL_JUDGE":           fixture.judge,
		"WORKFLOW_EVAL_COST_AUTHORIZED": "true",
		"FAKE_RUNNER_LOG":               fixture.runnerLog,
		"FAKE_JUDGE_COUNT":              fixture.judgeCount,
	}
}

func makeHarnessFixture(t *testing.T) harnessFixture {
	t.Helper()
	sourceRoot := findSourceRoot(t)
	parent := t.TempDir()
	root := filepath.Join(parent, "repo")
	mustMkdirAll(t, filepath.Join(root, "scripts/dev"))
	mustWriteFile(t, filepath.Join(root, "scripts/dev/hard-skills-evals.sh"), mustReadFile(t, filepath.Join(sourceRoot, "scripts/dev/hard-skills-evals.sh")))
	for _, skill := range selectedSkills {
		writeSkill(t, root, skill, skill, validDescription(), "# Baseline "+skill+"\n")
		writeEvalBundle(t, root, skill, validEvalBundle(skill))
	}
	runGit(t, root, "init", "--quiet")
	runGit(t, root, "add", "--all")
	runGit(t, root, "-c", "user.name=Test", "-c", "user.email=test@example.invalid", "commit", "--quiet", "-m", "baseline")
	baseline := strings.TrimSpace(runGit(t, root, "rev-parse", "HEAD"))
	writeSkill(t, root, "go-reliability-review", "go-reliability-review", validDescription(), "# Candidate reliability\n")
	mustWriteFile(t, filepath.Join(root, ".agents/skills/go-reliability-review/references/untracked-authority.md"), []byte("authoritative\n"))
	mustWriteFile(t, filepath.Join(root, "unrelated-sentinel.txt"), []byte("exclude me\n"))

	checker := filepath.Join(parent, "hard-skills-check")
	command := exec.CommandContext(t.Context(), "go", "build", "-o", checker, "./scripts/ci/hard-skills-check")
	command.Dir = sourceRoot
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("build checker: %v\n%s", err, output)
	}
	runner := filepath.Join(parent, "runner.sh")
	judge := filepath.Join(parent, "judge.sh")
	runnerLog := filepath.Join(parent, "runner.log")
	judgeCount := filepath.Join(parent, "judge.count")
	mustWriteExecutable(t, runner, fakeRunnerScript)
	mustWriteExecutable(t, judge, fakeJudgeScript)
	return harnessFixture{root: root, checker: checker, runner: runner, judge: judge, baseline: baseline, runnerLog: runnerLog, judgeCount: judgeCount}
}

func runHarness(t *testing.T, fixture harnessFixture, args []string, overrides map[string]string) (string, error) {
	t.Helper()
	command := exec.CommandContext(t.Context(), "bash", append([]string{filepath.Join(fixture.root, "scripts/dev/hard-skills-evals.sh")}, args...)...)
	controlled := map[string]bool{
		"HARD_SKILLS_CHECKER":           true,
		"WORKFLOW_EVAL_BASE_REF":        true,
		"WORKFLOW_EVAL_RUNNER":          true,
		"WORKFLOW_EVAL_JUDGE":           true,
		"WORKFLOW_EVAL_COST_AUTHORIZED": true,
		"FAKE_RUNNER_LOG":               true,
		"FAKE_JUDGE_COUNT":              true,
		"FAKE_JUDGE_MODE":               true,
		"FAKE_MUTATE_VARIANT":           true,
		"FAKE_MUTATE_PAYLOAD_ON":        true,
		"FAKE_REQUIRE_CANDIDATE_FILE":   true,
		"FAKE_FORBID_FILE":              true,
		"FAKE_ASSERT_SEALED":            true,
		"FAKE_SELECTED_SKILLS":          true,
	}
	var env []string
	for _, entry := range os.Environ() {
		key, _, _ := strings.Cut(entry, "=")
		if !controlled[key] {
			env = append(env, entry)
		}
	}
	env = append(env, "HARD_SKILLS_CHECKER="+fixture.checker)
	env = append(env, "FAKE_SELECTED_SKILLS="+strings.Join(selectedSkills, ","))
	for key, value := range overrides {
		env = append(env, key+"="+value)
	}
	command.Env = env
	output, err := command.CombinedOutput()
	return string(output), err
}

const fakeRunnerScript = `#!/usr/bin/env bash
set -euo pipefail
argv="$*"
variant=""
repo=""
prompt_file=""
while [[ $# -gt 0 ]]; do
  case "$1" in
    --variant) variant="$2"; shift 2 ;;
    --repo) repo="$2"; shift 2 ;;
    --prompt-file) prompt_file="$2"; shift 2 ;;
    *) exit 64 ;;
  esac
done
case "$argv" in
  *--case-id*|*expected*|*answer-key*|*domain_defect*|*clean*|*neighbor*|*unresolved_policy*) echo "routing or answer-key hint leaked in runner argv" >&2; exit 67 ;;
esac
IFS=',' read -r -a selected_skills <<<"$FAKE_SELECTED_SKILLS"
for skill in "${selected_skills[@]}"; do
  if [[ "$argv" == *"$skill"* ]]; then echo "selected skill leaked in runner argv" >&2; exit 67; fi
  if [[ "${FAKE_ASSERT_SEALED:-}" == true && -e "$repo/.agents/skills/$skill/evals" ]]; then echo "selected eval directory leaked into $variant snapshot: $skill" >&2; exit 68; fi
done
case "$prompt_file" in
  *cases*|*go-*|*domain_defect*|*unresolved_policy*|*expected*) echo "routing hint leaked in prompt path" >&2; exit 67 ;;
esac
if find "$(dirname "$prompt_file")" -mindepth 1 -maxdepth 1 ! -name input.txt | grep -q .; then
  echo "non-prompt payload accessible in opaque runner directory" >&2
  exit 69
fi
printf '%s\t%s\n' "$variant" "$(cksum <"$prompt_file")" >>"$FAKE_RUNNER_LOG"
if [[ "${FAKE_MUTATE_VARIANT:-}" == "$variant" ]]; then
  printf 'mutation\n' >"$repo/adapter-mutation.txt"
fi
if [[ "${FAKE_MUTATE_PAYLOAD_ON:-}" == "$variant:prompt" ]]; then
  printf 'payload mutation\n' >"$prompt_file"
fi
if [[ "${FAKE_MUTATE_PAYLOAD_ON:-}" == "$variant:expected" ]]; then
  printf 'payload mutation\n' >"$(dirname "$prompt_file")/expected.txt"
fi
if [[ "$variant" == candidate && -n "${FAKE_REQUIRE_CANDIDATE_FILE:-}" && ! -f "$repo/$FAKE_REQUIRE_CANDIDATE_FILE" ]]; then
  echo "authoritative candidate input missing" >&2
  exit 65
fi
if [[ -n "${FAKE_FORBID_FILE:-}" && -e "$repo/$FAKE_FORBID_FILE" ]]; then
  echo "unrelated input leaked into $variant" >&2
  exit 66
fi
cat "$prompt_file"
`

const fakeJudgeScript = `#!/usr/bin/env bash
set -euo pipefail
expected_file=""
while [[ $# -gt 0 ]]; do
  case "$1" in
    --expected-file) expected_file="$2"; shift 2 ;;
    --case-id|--baseline-output|--candidate-output) shift 2 ;;
    *) exit 64 ;;
  esac
done
if [[ "${FAKE_MUTATE_PAYLOAD_ON:-}" == judge:prompt ]]; then
  printf 'payload mutation\n' >"$(dirname "$expected_file")/prompt.txt"
fi
if [[ "${FAKE_MUTATE_PAYLOAD_ON:-}" == judge:expected ]]; then
  printf 'payload mutation\n' >"$expected_file"
fi
count=0
if [[ -f "$FAKE_JUDGE_COUNT" ]]; then count="$(<"$FAKE_JUDGE_COUNT")"; fi
count=$((count + 1))
printf '%s\n' "$count" >"$FAKE_JUDGE_COUNT"
if [[ "${FAKE_JUDGE_MODE:-}" == fail ]]; then exit 70; fi
echo baseline_pass=true
echo candidate_pass=true
echo candidate_non_regression=true
if [[ "${FAKE_JUDGE_MODE:-}" == disputed_once && "$count" -eq 1 ]]; then echo disputed=true; fi
if [[ "${FAKE_JUDGE_MODE:-}" == uncertain_twice && "$count" -le 2 ]]; then echo judge_uncertain=true; fi
`

func makeValidRepository(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	mustWriteFile(t, filepath.Join(root, filepath.FromSlash(sharedContractPath)), []byte("# Shared specialist contract\n"))
	for _, skill := range targetSkills {
		body := "# Skill\n"
		if !executionSkills[skill] {
			body += "\n[Shared contract](../specialist-contract.md)\n"
		}
		writeEvalBundle(t, root, skill, validEvalBundle(skill))
		writeSkill(t, root, skill, skill, validDescription(), body)
	}
	return root
}

func validDescription() string {
	return "Use when testing. Own behavior; Skip when outside scope."
}

func validEvalBundle(skill string) evalBundle {
	bundle := evalBundle{SkillName: skill}
	for index, category := range evalCategories {
		id := json.RawMessage(strconv.Quote(fmt.Sprintf("%s-%d", skill, index)))
		bundle.Evals = append(bundle.Evals, evalCase{
			ID:             id,
			Category:       category,
			Prompt:         "prompt " + skill + " " + category,
			ExpectedOutput: "expected " + skill + " " + category,
			Assertions:     []string{"assert " + skill + " " + category},
			Files:          []string{},
		})
	}
	return bundle
}

func TestSelectedEvalBundleAllowsAuxiliaryTrialCases(t *testing.T) {
	t.Parallel()

	bundle := validEvalBundle("go-security-spec")
	bundle.Evals = append(bundle.Evals, evalCase{
		ID:             json.RawMessage(`"auxiliary-safety-case"`),
		Category:       "domain_defect",
		TrialClass:     "safety_authority",
		Prompt:         "read the focused fixture",
		ExpectedOutput: "return focused security decisions",
		Assertions:     []string{"keeps the auxiliary oracle"},
		Files:          []string{"evals/fixtures/context.md"},
	})

	if problems := validateEvalBundle("go-security-spec", bundle); len(problems) != 0 {
		t.Fatalf("selected bundle rejected an auxiliary trial case: %v", problems)
	}
}

func writeEvalBundle(t *testing.T, root, skill string, bundle evalBundle) {
	t.Helper()
	data, err := json.MarshalIndent(bundle, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	data = append(data, '\n')
	mustWriteFile(t, filepath.Join(root, ".agents/skills", skill, "evals/evals.json"), data)
}

func writeSkill(t *testing.T, root, directory, name, description, body string) {
	t.Helper()
	content := fmt.Sprintf("---\nname: %s\ndescription: %s\n---\n%s", name, strconv.Quote(description), body)
	mustWriteFile(t, skillPath(root, directory), []byte(content))
}

func skillPath(root, skill string) string {
	return filepath.Join(root, ".agents", "skills", skill, "SKILL.md")
}

func makeSizeRepository(t *testing.T) (string, string) {
	t.Helper()
	root := t.TempDir()
	for _, skill := range targetSkills {
		baselineSkill := skill
		if old, ok := renamedSkills[skill]; ok {
			baselineSkill = old
		}
		writeSkill(t, root, baselineSkill, baselineSkill, validDescription(), "baseline "+baselineSkill+"\n")
	}
	writeSkill(t, root, "planning-session", "planning-session", validDescription(), "baseline planning-session\n")
	mustWriteFile(t, filepath.Join(root, filepath.FromSlash(sharedContractPath)), []byte("baseline shared\n"))
	runGit(t, root, "init", "--quiet")
	runGit(t, root, "add", "--all")
	runGit(t, root, "-c", "user.name=Test", "-c", "user.email=test@example.invalid", "commit", "--quiet", "-m", "baseline")
	baseline := strings.TrimSpace(runGit(t, root, "rev-parse", "HEAD"))
	for _, skill := range targetSkills {
		if old, ok := renamedSkills[skill]; ok {
			mustRemoveAll(t, filepath.Join(root, ".agents/skills", old))
		}
		body := "candidate " + skill + "\n"
		if !executionSkills[skill] {
			body += "[Shared contract](../specialist-contract.md)\n"
		}
		writeSkill(t, root, skill, skill, validDescription(), body)
	}
	writeSkill(t, root, "planning-session", "planning-session", validDescription(), "candidate planning-session\n")
	mustWriteFile(t, filepath.Join(root, filepath.FromSlash(sharedContractPath)), []byte("shared contract words\n"))
	return root, baseline
}

func parseSizeRows(t *testing.T, report string) map[string][]string {
	t.Helper()
	rows := make(map[string][]string)
	for _, line := range strings.Split(report, "\n") {
		fields := strings.Split(line, "\t")
		if len(fields) == 19 && fields[0] != "skill" {
			rows[fields[0]] = fields
		}
	}
	return rows
}

func mustAtoi(t *testing.T, value string) int {
	t.Helper()
	result, err := strconv.Atoi(value)
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func snapshotTree(t *testing.T, root string) string {
	t.Helper()
	var result strings.Builder
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return fmt.Errorf("relative emitted path: %w", err)
		}
		if entry.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return fmt.Errorf("relative emitted path: %w", err)
		}
		fmt.Fprintf(&result, "%s\n%s\n", filepath.ToSlash(rel), mustReadFile(t, path))
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return result.String()
}

func findSourceRoot(t *testing.T) string {
	t.Helper()
	current, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(current, "go.mod")); err == nil {
			return current
		}
		parent := filepath.Dir(current)
		if parent == current {
			t.Fatal("source root not found")
		}
		current = parent
	}
}

func runGit(t *testing.T, root string, args ...string) string {
	t.Helper()
	command := exec.CommandContext(t.Context(), "git", append([]string{"-C", root}, args...)...)
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, output)
	}
	return string(output)
}

func mustWriteFile(t *testing.T, path string, data []byte) {
	t.Helper()
	mustMkdirAll(t, filepath.Dir(path))
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
}

func mustWriteExecutable(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o755); err != nil {
		t.Fatal(err)
	}
}

func mustMkdirAll(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatal(err)
	}
}

func mustRemoveAll(t *testing.T, path string) {
	t.Helper()
	if err := os.RemoveAll(path); err != nil {
		t.Fatal(err)
	}
}

func mustReadFile(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func nonEmptyLines(value string) []string {
	var result []string
	for _, line := range strings.Split(value, "\n") {
		if line != "" {
			result = append(result, line)
		}
	}
	return result
}
