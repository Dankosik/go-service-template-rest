package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
)

func TestInventoryContract(t *testing.T) {
	t.Parallel()
	if len(targetSkills) != 22 {
		t.Fatalf("target inventory has %d skills, want 22", len(targetSkills))
	}
	if len(domainSkills) != 18 {
		t.Fatalf("domain inventory has %d skills, want 18", len(domainSkills))
	}
	if len(executionSkills) != 4 {
		t.Fatalf("execution inventory has %d skills, want 4", len(executionSkills))
	}
	if !retiredSkillSet()["go-specialist-router"] {
		t.Fatal("retired router identifier is not guarded")
	}
	wantSelected := []string{"go-reliability", "go-security", "go-observability", "go-coder", "go-verification-before-completion", "go-implementation-ownership", "go-systematic-debugging", "go-concurrency", "go-test-strategy"}
	if !slices.Equal(selectedSkills, wantSelected) {
		t.Fatalf("selected skills = %v, want %v", selectedSkills, wantSelected)
	}
}

func TestStage2MetadataIsClosedAndDormant(t *testing.T) {
	t.Parallel()
	prompt := strings.Repeat("a", 64)
	commit := strings.Repeat("b", 40)
	valid := fmt.Sprintf(`{"schema_version":1,"prompt_sha256":%q,"repository_commit":%q,"routing_mode":"implicit","explicit_skill_mentions":[],"forced_skills":[],"selected_skills":[{"name":"go-security","source":"implicit"},{"name":"go-reliability","source":"implicit"}],"provenance_source":"runtime_events"}`, prompt, commit)
	if err := validateStage2Metadata([]byte(valid), prompt, commit, nil); err != nil {
		t.Fatal(err)
	}
	tests := []struct{ name, metadata string }{
		{"wrong digest", strings.Replace(valid, prompt, strings.Repeat("c", 64), 1)},
		{"wrong commit", strings.Replace(valid, commit, strings.Repeat("c", 40), 1)},
		{"wrong mode", strings.Replace(valid, `"routing_mode":"implicit"`, `"routing_mode":"explicit"`, 1)},
		{"explicit list", strings.Replace(valid, `"explicit_skill_mentions":[]`, `"explicit_skill_mentions":["go-security"]`, 1)},
		{"forced list", strings.Replace(valid, `"forced_skills":[]`, `"forced_skills":["go-security"]`, 1)},
		{"non-domain skill", strings.Replace(valid, `[{"name":"go-security","source":"implicit"},{"name":"go-reliability","source":"implicit"}]`, `[{"name":"go-coder","source":"implicit"}]`, 1)},
		{"non-implicit source", strings.Replace(valid, `"source":"implicit"`, `"source":"explicit"`, 1)},
		{"duplicate selection", strings.Replace(valid, `[{"name":"go-security","source":"implicit"},{"name":"go-reliability","source":"implicit"}]`, `[{"name":"go-security","source":"implicit"},{"name":"go-security","source":"implicit"}]`, 1)},
		{"empty selection", strings.Replace(valid, `[{"name":"go-security","source":"implicit"},{"name":"go-reliability","source":"implicit"}]`, `[]`, 1)},
		{"unknown key", strings.Replace(valid, `"runtime_events"}`, `"runtime_events","extra":true}`, 1)},
		{"non-runtime provenance", strings.Replace(valid, `"runtime_events"`, `"adapter_claim"`, 1)},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := validateStage2Metadata([]byte(test.metadata), prompt, commit, nil); err == nil {
				t.Fatal("invalid stage 2 metadata passed")
			}
		})
	}
	if err := validateStage2Metadata([]byte(valid), prompt, commit, map[string]bool{"go-security": true}); err == nil {
		t.Fatal("explicit-only domain specialist passed")
	}
}

func TestStage2MetadataAcceptsOneImplicitDomain(t *testing.T) {
	t.Parallel()
	prompt := strings.Repeat("a", 64)
	commit := strings.Repeat("b", 40)
	metadata := fmt.Sprintf(`{"schema_version":1,"prompt_sha256":%q,"repository_commit":%q,"routing_mode":"implicit","explicit_skill_mentions":[],"forced_skills":[],"selected_skills":[{"name":"go-security","source":"implicit"}],"provenance_source":"runtime_events"}`, prompt, commit)
	if err := validateStage2Metadata([]byte(metadata), prompt, commit, nil); err != nil {
		t.Fatal(err)
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
		{"review selector link", ".agents/skills/go-chi/references/review/index.md", "[missing](missing.md)\n"},
		{"decision selector link", ".agents/skills/go-chi/references/decision/index.md", "[missing](missing.md)\n"},
	} {
		t.Run(test.name, func(t *testing.T) {
			root := makeValidRepository(t)
			mustWriteFile(t, filepath.Join(root, filepath.FromSlash(test.path)), []byte(test.body))
			if got := strings.Join(validateCatalog(root), "\n"); !strings.Contains(got, "unresolved local Markdown link") {
				t.Fatalf("broken link passed: %s", got)
			}
		})
	}
	t.Run("zero policies", func(t *testing.T) {
		root := makeValidRepository(t)
		if got := validateImplicitPolicies(root); len(got) != 0 {
			t.Fatal(got)
		}
	})
	t.Run("non-domain shared-contract link is not a domain policy", func(t *testing.T) {
		root := makeValidRepository(t)
		writeSkill(t, root, "grilling", "grilling", validDescription(), "# Skill\n\n[Shared contract](../specialist-contract.md)\n")
		mustWriteFile(t, filepath.Join(root, ".agents/skills/grilling/agents/openai.yaml"), []byte("policy:\n  allow_implicit_invocation: false\n"))
		if got := validateImplicitPolicies(root); len(got) != 0 {
			t.Fatalf("non-domain policy was classified as a domain policy: %v", got)
		}
	})
	t.Run("domain explicit-only policy", func(t *testing.T) {
		root := makeValidRepository(t)
		mustWriteFile(t, filepath.Join(root, ".agents/skills/go-security/agents/openai.yaml"), []byte("policy:\n  allow_implicit_invocation: false\n"))
		if got := validateImplicitPolicies(root); len(got) != 1 || !strings.Contains(got[0], "must remain implicitly invocable") {
			t.Fatalf("explicit-only domain policy passed: %v", got)
		}
	})
}

func TestTrialClassMatrix(t *testing.T) {
	t.Parallel()
	bundle := validEvalBundle("go-reliability")
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
	bundle.Evals[len(bundle.Evals)-1].TrialClass = "safety_authority"
	if problems := validateEvalBundle(bundle.SkillName, bundle); len(problems) != 0 {
		t.Fatalf("safety_authority rejected: %v", problems)
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
	t.Run("plain scalar description", func(t *testing.T) {
		root := makeValidRepository(t)
		mustWriteFile(t, skillPath(root, "go-coder"), []byte("---\nname: go-coder\ndescription: Use when testing. Own behavior; Skip when outside scope.\n---\n# Body\n"))
		if err := checkRepository(root, targetSkills, false); err != nil {
			t.Fatalf("plain scalar description failed: %v", err)
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
				mustRemoveAll(t, filepath.Join(root, ".agents/skills/go-chi"))
			},
			want: "missing canonical hard skill: go-chi",
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
			name: "empty description",
			mutate: func(t *testing.T, root string) {
				t.Helper()
				path := skillPath(root, "go-coder")
				mustWriteFile(t, path, []byte("---\nname: go-coder\ndescription:\n---\n# Body\n"))
			},
			want: "description must be a non-empty one-line value",
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
				writeSkill(t, root, "go-chi", "go-chi", validDescription(), "# Body\n")
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
				writeSkill(t, root, "go-chi", "go-chi", validDescription(), body)
			},
			want: "unresolved local Markdown link",
		},
		{
			name: "stale eval skill name",
			mutate: func(t *testing.T, root string) {
				t.Helper()
				bundle := validEvalBundle("go-chi")
				bundle.SkillName = "go-db-cache"
				writeEvalBundle(t, root, "go-chi", bundle)
			},
			want: "eval skill_name is \"go-db-cache\"",
		},
		{
			name: "duplicate eval id",
			mutate: func(t *testing.T, root string) {
				t.Helper()
				bundle := validEvalBundle("go-chi")
				bundle.Evals[1].ID = bundle.Evals[0].ID
				writeEvalBundle(t, root, "go-chi", bundle)
			},
			want: "duplicate id",
		},
		{
			name: "missing category",
			mutate: func(t *testing.T, root string) {
				t.Helper()
				bundle := validEvalBundle("go-chi")
				bundle.Evals = bundle.Evals[:3]
				writeEvalBundle(t, root, "go-chi", bundle)
			},
			want: "missing eval category unresolved_policy",
		},
		{
			name: "empty prompt",
			mutate: func(t *testing.T, root string) {
				t.Helper()
				bundle := validEvalBundle("go-chi")
				bundle.Evals[0].Prompt = " "
				writeEvalBundle(t, root, "go-chi", bundle)
			},
			want: "prompt is empty",
		},
		{
			name: "selected file input",
			mutate: func(t *testing.T, root string) {
				t.Helper()
				bundle := validEvalBundle("go-reliability")
				bundle.Evals[0].Files = []string{"inputs/code.go"}
				writeEvalBundle(t, root, "go-reliability", bundle)
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
	broken := validEvalBundle("go-security")
	broken.Evals = broken.Evals[:3]
	writeEvalBundle(t, root, "go-security", broken)

	if err := checkRepository(root, []string{"go-reliability"}, true); err != nil {
		t.Fatalf("unrelated sibling escaped scoped isolation: %v", err)
	}
	if err := checkRepository(root, targetSkills, false); err == nil || !strings.Contains(err.Error(), "go-security") {
		t.Fatalf("global check did not find sibling defect: %v", err)
	}

	writeSkill(t, root, "go-reliability", "go-reliability", validDescription(), "# No contract\n")
	if err := checkRepository(root, []string{"go-reliability"}, true); err == nil || !strings.Contains(err.Error(), "missing direct link") {
		t.Fatalf("scoped check hid selected defect: %v", err)
	}

	for _, test := range []struct {
		raw  string
		want string
	}{
		{"", "must not be empty"},
		{"go-coder,", "empty or non-exact"},
		{"go-coder, go-chi", "empty or non-exact"},
		{"go-coder,go-coder", "duplicate"},
		{"go-design-spec", "retired"},
		{"go-not-real", "unknown"},
	} {
		_, err := parseSkillScope(test.raw)
		if err == nil || !strings.Contains(err.Error(), test.want) {
			t.Errorf("parseSkillScope(%q) error = %v, want %q", test.raw, err, test.want)
		}
	}
	want := []string{"go-coder", "go-chi"}
	got, err := parseSkillScope(strings.Join(want, ","))
	if err != nil || !slices.Equal(got, want) {
		t.Fatalf("valid scope = %v, %v", got, err)
	}
}

func TestRunCLICommands(t *testing.T) {
	root := makeValidRepository(t)
	t.Setenv("HARD_SKILLS_REPO_ROOT", root)

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
		{name: "unknown command", args: []string{"unknown"}, wantErr: "usage:"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := runCLI(test.args)
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

func TestReadRootFileRejectsEscapingSymlink(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	outside := filepath.Join(t.TempDir(), "outside")
	mustWriteFile(t, outside, []byte("outside\n"))
	link := filepath.Join(root, "escape")
	if err := os.Symlink(outside, link); err != nil {
		t.Skipf("create symlink: %v", err)
	}
	if _, err := readRootFile(root, link); err == nil {
		t.Fatal("root-scoped read followed symlink outside repository")
	}
}

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

	bundle := validEvalBundle("go-security")
	bundle.Evals = append(bundle.Evals, evalCase{
		ID:             json.RawMessage(`"auxiliary-safety-case"`),
		Category:       "domain_defect",
		TrialClass:     "safety_authority",
		Prompt:         "read the focused fixture",
		ExpectedOutput: "return focused security decisions",
		Assertions:     []string{"keeps the auxiliary oracle"},
		Files:          []string{"evals/fixtures/context.md"},
	})

	if problems := validateEvalBundle("go-security", bundle); len(problems) != 0 {
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

func mustWriteFile(t *testing.T, path string, data []byte) {
	t.Helper()
	mustMkdirAll(t, filepath.Dir(path))
	if err := os.WriteFile(path, data, 0o644); err != nil {
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
