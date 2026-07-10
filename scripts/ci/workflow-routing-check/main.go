package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

type fixtureFile struct {
	SchemaVersion int        `json:"schema_version"`
	Cases         []testCase `json:"cases"`
}

type evalManifest struct {
	SkillName string     `json:"skill_name"`
	Evals     []evalCase `json:"evals"`
}

type evalCase struct {
	ID             *int     `json:"id"`
	Prompt         string   `json:"prompt"`
	ExpectedOutput string   `json:"expected_output"`
	Files          []string `json:"files"`
	Expectations   []string `json:"expectations,omitempty"`
	Assertions     []string `json:"assertions,omitempty"`
}

type testCase struct {
	ID        string         `json:"id"`
	Family    string         `json:"family"`
	Covers    []string       `json:"covers"`
	Input     map[string]any `json:"input"`
	Want      map[string]any `json:"want,omitempty"`
	WantError string         `json:"want_error,omitempty"`
}

var ruleIDPattern = regexp.MustCompile(`^[A-Z][A-Z0-9-]+$`)

type valueKind string

const (
	kindBool        valueKind = "boolean"
	kindString      valueKind = "string"
	kindNumber      valueKind = "number"
	kindObject      valueKind = "object"
	kindStringArray valueKind = "string array"
	kindObjectArray valueKind = "object array"
)

type fieldSpec struct {
	Kind     valueKind
	Required bool
}

type ruleTrace map[string]struct{}

func (trace ruleTrace) mark(ids ...string) {
	if trace == nil {
		return
	}
	for _, id := range ids {
		if id != "" {
			trace[id] = struct{}{}
		}
	}
}

func (trace ruleTrace) markPrefix(rules map[string]string, prefix string) {
	for id := range rules {
		if strings.HasPrefix(id, prefix) {
			trace.mark(id)
		}
	}
}

func main() {
	verifyCoverage := flag.Bool("verify-coverage", false, "verify canonical rule and fixture coverage without executing cases")
	flag.Parse()

	root, err := repositoryRoot()
	if err != nil {
		fatal(err)
	}
	evalCount, err := validateEvalManifests(root)
	if err != nil {
		fatal(err)
	}

	rules := map[string]string{}
	for _, path := range []string{
		filepath.Join(root, "AGENTS.md"),
		filepath.Join(root, "docs/spec-first-workflow/shared/artifact-model.md"),
	} {
		if err := readRuleTables(path, rules); err != nil {
			fatal(err)
		}
	}
	if len(rules) == 0 {
		fatal(errors.New("no canonical workflow rule IDs found"))
	}

	fixturePath := filepath.Join(root, "scripts/ci/workflow-routing-check/testdata/cases.json")
	fixtures, err := readFixtures(fixturePath)
	if err != nil {
		fatal(err)
	}
	if err := validateFixtureMetadata(rules, fixtures.Cases); err != nil {
		fatal(err)
	}

	covered := map[string]string{}
	for _, tc := range fixtures.Cases {
		if err := validateFamilyInput(tc.Family, tc.Input); err != nil {
			if tc.WantError == "" || !strings.Contains(err.Error(), tc.WantError) {
				fatal(fmt.Errorf("case %s: %w", tc.ID, err))
			}
			if len(tc.Covers) != 0 {
				fatal(fmt.Errorf("case %s: schema-error fixture cannot claim canonical rule coverage", tc.ID))
			}
			continue
		}
		got, executed, err := evaluate(tc, rules)
		if tc.WantError != "" {
			if err == nil {
				fatal(fmt.Errorf("case %s: expected error containing %q, got success", tc.ID, tc.WantError))
			}
			if !strings.Contains(err.Error(), tc.WantError) {
				fatal(fmt.Errorf("case %s: error %q does not contain %q", tc.ID, err, tc.WantError))
			}
			if len(tc.Covers) != 0 {
				fatal(fmt.Errorf("case %s: evaluator-error fixture cannot claim canonical rule coverage", tc.ID))
			}
			continue
		}
		if err != nil {
			fatal(fmt.Errorf("case %s: %w", tc.ID, err))
		}
		if !equalJSON(got, tc.Want) {
			fatal(fmt.Errorf("case %s: got %s, want %s", tc.ID, compactJSON(got), compactJSON(tc.Want)))
		}
		if err := recordDeclaredCoverage(tc, executed, covered); err != nil {
			fatal(err)
		}
	}
	if err := verifyExecutedCoverage(rules, covered); err != nil {
		fatal(err)
	}
	if *verifyCoverage {
		fmt.Printf("workflow routing coverage verified: %d rules, %d executable cases, %d skill evals\n", len(rules), len(fixtures.Cases), evalCount)
		return
	}

	fmt.Printf("workflow routing check passed: %d rules, %d cases, %d skill evals\n", len(rules), len(fixtures.Cases), evalCount)
}

func repositoryRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if fileExists(filepath.Join(dir, "AGENTS.md")) && fileExists(filepath.Join(dir, "go.mod")) {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", errors.New("repository root not found")
		}
		dir = parent
	}
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func validateEvalManifests(root string) (int, error) {
	paths, err := filepath.Glob(filepath.Join(root, ".agents/skills/*/evals/evals.json"))
	if err != nil {
		return 0, err
	}
	if len(paths) == 0 {
		return 0, errors.New("no skill eval manifests found")
	}
	total := 0
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			return 0, err
		}
		var manifest evalManifest
		if err := decodeStrictJSON(data, &manifest); err != nil {
			return 0, fmt.Errorf("%s: %w", path, err)
		}
		if manifest.SkillName == "" || len(manifest.Evals) == 0 {
			return 0, fmt.Errorf("%s: skill_name and non-empty evals are required", path)
		}
		ids := map[int]struct{}{}
		for index, eval := range manifest.Evals {
			if eval.ID == nil {
				return 0, fmt.Errorf("%s: eval %d is missing integer id", path, index)
			}
			if _, exists := ids[*eval.ID]; exists {
				return 0, fmt.Errorf("%s: duplicate eval id %d", path, *eval.ID)
			}
			ids[*eval.ID] = struct{}{}
			if strings.TrimSpace(eval.Prompt) == "" || strings.TrimSpace(eval.ExpectedOutput) == "" || eval.Files == nil {
				return 0, fmt.Errorf("%s: eval %d requires prompt, expected_output, and files", path, *eval.ID)
			}
			if len(eval.Expectations) == 0 && len(eval.Assertions) == 0 {
				return 0, fmt.Errorf("%s: eval %d requires expectations or assertions", path, *eval.ID)
			}
			total++
		}
	}
	return total, nil
}

func decodeStrictJSON(data []byte, target any) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("multiple JSON documents")
		}
		return err
	}
	return nil
}

func readRuleTables(path string, rules map[string]string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	inside := false
	scanner := bufio.NewScanner(file)
	lineNo := 0
	for scanner.Scan() {
		lineNo++
		line := strings.TrimSpace(scanner.Text())
		switch line {
		case "<!-- workflow-rule-table:start -->":
			if inside {
				return fmt.Errorf("%s:%d: nested workflow rule table", path, lineNo)
			}
			inside = true
			continue
		case "<!-- workflow-rule-table:end -->":
			if !inside {
				return fmt.Errorf("%s:%d: unmatched workflow rule table end", path, lineNo)
			}
			inside = false
			continue
		}
		if !inside || !strings.HasPrefix(line, "|") {
			continue
		}
		cells := splitMarkdownRow(line)
		if len(cells) == 0 || cells[0] == "Rule ID" || strings.Trim(cells[0], "-: ") == "" {
			continue
		}
		id := strings.Trim(cells[0], "` ")
		if !ruleIDPattern.MatchString(id) {
			return fmt.Errorf("%s:%d: invalid rule ID %q", path, lineNo, id)
		}
		if previous, ok := rules[id]; ok {
			return fmt.Errorf("duplicate rule ID %s at %s:%d (first at %s)", id, path, lineNo, previous)
		}
		rules[id] = fmt.Sprintf("%s:%d", path, lineNo)
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	if inside {
		return fmt.Errorf("%s: unterminated workflow rule table", path)
	}
	return nil
}

func splitMarkdownRow(line string) []string {
	line = strings.TrimPrefix(line, "|")
	line = strings.TrimSuffix(line, "|")
	parts := strings.Split(line, "|")
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	return parts
}

func readFixtures(path string) (fixtureFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return fixtureFile{}, err
	}
	var fixtures fixtureFile
	if err := decodeStrictJSON(data, &fixtures); err != nil {
		return fixtureFile{}, err
	}
	if fixtures.SchemaVersion != 1 {
		return fixtureFile{}, fmt.Errorf("unsupported fixture schema_version %d", fixtures.SchemaVersion)
	}
	if len(fixtures.Cases) == 0 {
		return fixtureFile{}, errors.New("no workflow routing fixtures")
	}
	return fixtures, nil
}

func validateFixtureMetadata(rules map[string]string, cases []testCase) error {
	caseIDs := map[string]struct{}{}
	for _, tc := range cases {
		if tc.ID == "" || tc.Family == "" {
			return errors.New("fixture id and family are required")
		}
		if len(tc.Covers) == 0 && tc.WantError == "" {
			return fmt.Errorf("fixture %s must cover at least one canonical rule", tc.ID)
		}
		if tc.Want == nil && tc.WantError == "" {
			return fmt.Errorf("fixture %s must define want or want_error", tc.ID)
		}
		if tc.Want != nil && tc.WantError != "" {
			return fmt.Errorf("fixture %s cannot define both want and want_error", tc.ID)
		}
		if tc.WantError != "" && len(tc.Covers) != 0 {
			return fmt.Errorf("fixture %s: expected-error cases cannot claim canonical rule coverage", tc.ID)
		}
		if _, exists := caseIDs[tc.ID]; exists {
			return fmt.Errorf("duplicate fixture ID %s", tc.ID)
		}
		caseIDs[tc.ID] = struct{}{}
		caseCoverage := map[string]struct{}{}
		for _, id := range tc.Covers {
			if _, ok := rules[id]; !ok {
				return fmt.Errorf("fixture %s references unknown rule ID %s", tc.ID, id)
			}
			if !familyAllowsRule(tc.Family, id) {
				return fmt.Errorf("fixture %s family %s cannot cover rule ID %s", tc.ID, tc.Family, id)
			}
			if _, exists := caseCoverage[id]; exists {
				return fmt.Errorf("fixture %s repeats rule ID %s", tc.ID, id)
			}
			caseCoverage[id] = struct{}{}
		}
	}
	return nil
}

func recordDeclaredCoverage(tc testCase, executed map[string]struct{}, covered map[string]string) error {
	for _, id := range tc.Covers {
		if _, ok := executed[id]; !ok {
			return fmt.Errorf("case %s claims rule %s without executing its rule-specific branch", tc.ID, id)
		}
		covered[id] = tc.ID
	}
	return nil
}

func verifyExecutedCoverage(rules map[string]string, covered map[string]string) error {
	var missing []string
	for id := range rules {
		if _, ok := covered[id]; !ok {
			missing = append(missing, id)
		}
	}
	sort.Strings(missing)
	if len(missing) > 0 {
		return fmt.Errorf("canonical rules without fixture coverage: %s", strings.Join(missing, ", "))
	}
	return nil
}

func familyAllowsRule(family, id string) bool {
	prefixes := map[string][]string{
		"shape":          {"SHAPE-", "DIRECT-", "LEAN-", "FULL-"},
		"agent_request":  {"AGENT-"},
		"fanout":         {"FANOUT-"},
		"artifact_depth": {"ARTIFACT-", "DEPTH-"},
		"risk_challenge": {"STATE-RISK-CHALLENGE"},
		"state":          {"STATE-"},
		"legacy":         {"LEGACY-"},
		"transition":     {"TRANS-"},
		"routing":        {"ROUTING-"},
		"adequacy":       {"ADEQUACY-"},
		"status":         {"STATUS-"},
		"mirror":         {"MIRROR-"},
	}
	allowed, ok := prefixes[family]
	if !ok {
		return false
	}
	for _, prefix := range allowed {
		if strings.HasPrefix(id, prefix) {
			return true
		}
	}
	return false
}

func validateFamilyInput(family string, input map[string]any) error {
	schemas := map[string]map[string]fieldSpec{
		"shape": {
			"intake_accepted":   {Kind: kindBool, Required: true},
			"agent_request":     {Kind: kindString, Required: true},
			"full_triggers":     {Kind: kindObject},
			"direct_predicates": {Kind: kindObject},
			"lean_predicates":   {Kind: kindObject},
		},
		"agent_request": {
			"authorized":    {Kind: kindBool, Required: true},
			"required_lane": {Kind: kindBool, Required: true},
			"substantive":   {Kind: kindBool, Required: true},
		},
		"fanout": {
			"independent_questions":    {Kind: kindBool, Required: true},
			"separate_context_benefit": {Kind: kindBool, Required: true},
			"small_or_sequential":      {Kind: kindBool, Required: true},
			"shared_mutable_state":     {Kind: kindBool, Required: true},
			"requested_concurrency":    {Kind: kindNumber, Required: true},
			"excess_reason":            {Kind: kindString, Required: true},
		},
		"artifact_depth": {
			"shape": {Kind: kindString, Required: true},
		},
		"risk_challenge": {
			"field":             {Kind: kindString, Required: true},
			"value":             {Kind: kindString, Required: true},
			"execution_shape":   {Kind: kindString, Required: true},
			"routing_scope":     {Kind: kindString, Required: true},
			"routing_revision":  {Kind: kindNumber, Required: true},
			"record_validity":   {Kind: kindString, Required: true},
			"proof_obligations": {Kind: kindStringArray, Required: true},
		},
		"state": {
			"kind":                      {Kind: kindString},
			"phrase":                    {Kind: kindString},
			"artifact_expectation":      {Kind: kindString},
			"artifact_state":            {Kind: kindString},
			"record_validity":           {Kind: kindString},
			"waiver_disposition":        {Kind: kindString},
			"waiver_eligible":           {Kind: kindBool},
			"waiver_rationale":          {Kind: kindString},
			"waiver_evidence":           {Kind: kindString},
			"waiver_reopen_trigger":     {Kind: kindString},
			"execution_shape":           {Kind: kindString},
			"phase_state":               {Kind: kindString},
			"procedural_gate_state":     {Kind: kindString},
			"review_verdict":            {Kind: kindString},
			"subagent_gate":             {Kind: kindString},
			"session_boundary":          {Kind: kindString},
			"handoff_readiness":         {Kind: kindString},
			"routing_scope":             {Kind: kindString},
			"routing_revision":          {Kind: kindNumber},
			"observed_routing_scope":    {Kind: kindString},
			"observed_routing_revision": {Kind: kindNumber},
		},
		"legacy": {
			"field": {Kind: kindString, Required: true},
			"value": {Kind: kindString, Required: true},
		},
		"transition": {
			"kind":                          {Kind: kindString, Required: true},
			"predicate_false_or_unknown":    {Kind: kindBool},
			"intake_accepted":               {Kind: kindBool},
			"agent_request":                 {Kind: kindString},
			"full_triggers":                 {Kind: kindObject},
			"direct_predicates":             {Kind: kindObject},
			"lean_predicates":               {Kind: kindObject},
			"source_scope":                  {Kind: kindString},
			"source_revision":               {Kind: kindNumber},
			"source_record_validity":        {Kind: kindString},
			"prior_edits_unapproved":        {Kind: kindBool},
			"prior_shape":                   {Kind: kindString},
			"target_shape":                  {Kind: kindString},
			"revision":                      {Kind: kindNumber},
			"master_scope":                  {Kind: kindString},
			"phase_scope":                   {Kind: kindString},
			"master_revision":               {Kind: kindNumber},
			"phase_revision":                {Kind: kindNumber},
			"fresh_task_review":             {Kind: kindBool},
			"blocker_recorded_in_tasks":     {Kind: kindBool},
			"readiness_stale":               {Kind: kindBool},
			"active_scope":                  {Kind: kindString},
			"active_revision":               {Kind: kindNumber},
			"review_scope":                  {Kind: kindString},
			"review_revision":               {Kind: kindNumber},
			"review_record_validity":        {Kind: kindString},
			"dependent_record_dispositions": {Kind: kindObject},
		},
		"routing": {
			"research_expectation":         {Kind: kindString, Required: true},
			"next_targets":                 {Kind: kindStringArray, Required: true},
			"phase_control_reasons":        {Kind: kindStringArray, Required: true},
			"mandatory_gate":               {Kind: kindBool},
			"dedicated_planning_requested": {Kind: kindBool, Required: true},
			"durable_routing":              {Kind: kindBool, Required: true},
			"intake_accepted":              {Kind: kindBool},
			"agent_request":                {Kind: kindString},
			"full_triggers":                {Kind: kindObject},
			"direct_predicates":            {Kind: kindObject},
			"lean_predicates":              {Kind: kindObject},
			"recorded_shape":               {Kind: kindString},
			"matched_rule":                 {Kind: kindString},
			"routing_scope":                {Kind: kindString},
			"routing_revision":             {Kind: kindNumber},
			"record_validity":              {Kind: kindString},
		},
		"adequacy": {
			"intake_accepted":               {Kind: kindBool, Required: true},
			"agent_request":                 {Kind: kindString, Required: true},
			"full_triggers":                 {Kind: kindObject, Required: true},
			"direct_predicates":             {Kind: kindObject, Required: true},
			"lean_predicates":               {Kind: kindObject, Required: true},
			"selected_shape":                {Kind: kindString, Required: true},
			"selected_rule":                 {Kind: kindString, Required: true},
			"durable_planning":              {Kind: kindBool, Required: true},
			"change_kind":                   {Kind: kindString, Required: true},
			"active_scope":                  {Kind: kindString, Required: true},
			"active_revision":               {Kind: kindNumber, Required: true},
			"observed_scope":                {Kind: kindString, Required: true},
			"observed_revision":             {Kind: kindNumber, Required: true},
			"record_validity":               {Kind: kindString, Required: true},
			"dependent_record_dispositions": {Kind: kindObject, Required: true},
		},
		"status": {
			"tasks_present":                  {Kind: kindBool},
			"tasks_scope":                    {Kind: kindString},
			"tasks_revision":                 {Kind: kindNumber},
			"tasks_record_validity":          {Kind: kindString},
			"tasks_blockers_clear":           {Kind: kindBool},
			"tasks_concerns_complete":        {Kind: kindBool},
			"tasks_waiver_eligible":          {Kind: kindBool},
			"durable_present":                {Kind: kindBool},
			"master_scope":                   {Kind: kindString},
			"phase_scope":                    {Kind: kindString},
			"master_revision":                {Kind: kindNumber},
			"phase_revision":                 {Kind: kindNumber},
			"master_record_validity":         {Kind: kindString},
			"phase_record_validity":          {Kind: kindString},
			"phase_artifact_present":         {Kind: kindBool},
			"phase_artifact_scope":           {Kind: kindString},
			"phase_artifact_revision":        {Kind: kindNumber},
			"phase_artifact_record_validity": {Kind: kindString},
			"phase_concerns_complete":        {Kind: kindBool},
			"direct_envelope_present":        {Kind: kindBool},
			"provenance":                     {Kind: kindString},
			"same_session":                   {Kind: kindBool},
			"direct_record_validity":         {Kind: kindString},
			"framing_accepted":               {Kind: kindBool},
			"trigger_audit_complete":         {Kind: kindBool},
			"direct_agent_request":           {Kind: kindString},
			"direct_full_triggers":           {Kind: kindObject},
			"direct_predicates":              {Kind: kindObject},
			"direct_lean_predicates":         {Kind: kindObject},
			"matched_rule":                   {Kind: kindString},
			"actor":                          {Kind: kindString},
			"routing_scope":                  {Kind: kindString},
			"routing_revision":               {Kind: kindNumber},
			"active_scope":                   {Kind: kindString},
			"active_revision":                {Kind: kindNumber},
			"active_execution_shape":         {Kind: kindString},
			"active_matched_rule":            {Kind: kindString},
			"proof_result":                   {Kind: kindString},
			"reopen_seam_present":            {Kind: kindBool},
			"report_execution_shape":         {Kind: kindString},
			"report_matched_rule":            {Kind: kindString},
			"report_shape_evidence":          {Kind: kindString},
			"report_adequacy_required":       {Kind: kindBool},
			"report_adequacy_result":         {Kind: kindString},
			"report_phase_state":             {Kind: kindString},
			"report_session_boundary":        {Kind: kindString},
			"report_artifact_expectation":    {Kind: kindString},
			"report_artifact_state":          {Kind: kindString},
			"report_record_validity":         {Kind: kindString},
			"report_gate_state":              {Kind: kindString},
			"report_review_verdict":          {Kind: kindString},
			"report_handoff_readiness":       {Kind: kindString},
			"report_allowed_writes":          {Kind: kindString},
			"report_next_phase":              {Kind: kindString},
			"report_next_action":             {Kind: kindString},
		},
		"mirror": {
			"canonical_available": {Kind: kindBool, Required: true},
			"render_ok":           {Kind: kindBool, Required: true},
			"compare_ok":          {Kind: kindBool},
			"present":             {Kind: kindBool},
			"required":            {Kind: kindBool},
			"in_sync":             {Kind: kindBool},
			"strict":              {Kind: kindBool},
			"target_only_files":   {Kind: kindBool},
			"targets":             {Kind: kindObjectArray},
		},
	}
	schema, ok := schemas[family]
	if !ok {
		return fmt.Errorf("unknown family %q", family)
	}
	for key := range input {
		if _, ok := schema[key]; !ok {
			return fmt.Errorf("family %s: unknown input field %q", family, key)
		}
	}
	for key, spec := range schema {
		value, present := input[key]
		if !present {
			if spec.Required {
				return fmt.Errorf("family %s: missing required input field %q", family, key)
			}
			continue
		}
		if !matchesKind(value, spec.Kind) {
			return fmt.Errorf("family %s: input field %q must be %s", family, key, spec.Kind)
		}
	}
	return nil
}

func matchesKind(value any, kind valueKind) bool {
	switch kind {
	case kindBool:
		_, ok := value.(bool)
		return ok
	case kindString:
		_, ok := value.(string)
		return ok
	case kindNumber:
		_, ok := value.(float64)
		return ok
	case kindObject:
		_, ok := value.(map[string]any)
		return ok
	case kindStringArray:
		values, ok := value.([]any)
		if !ok {
			return false
		}
		for _, item := range values {
			if _, ok := item.(string); !ok {
				return false
			}
		}
		return true
	case kindObjectArray:
		values, ok := value.([]any)
		if !ok {
			return false
		}
		for _, item := range values {
			if _, ok := item.(map[string]any); !ok {
				return false
			}
		}
		return true
	default:
		return false
	}
}

func legacyRuleID(field, rawValue string) string {
	value := normalizeLegacy(rawValue)
	switch field {
	case "shape":
		switch value {
		case "direct path", "direct_path":
			return "LEGACY-SHAPE-DIRECT"
		case "lean local", "lean_local", "lightweight local", "lightweight_local":
			return "LEGACY-SHAPE-LEAN"
		case "full orchestrated", "full_orchestrated":
			return "LEGACY-SHAPE-FULL"
		}
	case "phase":
		switch value {
		case "pending", "not_started":
			return "LEGACY-PHASE-NOT-STARTED"
		case "active", "in_progress", "in progress":
			return "LEGACY-PHASE-ACTIVE"
		case "complete", "completed", "done":
			return "LEGACY-PHASE-COMPLETE"
		case "blocked", "reopened":
			return "LEGACY-PHASE-BLOCKED-REOPENED"
		}
	case "artifact":
		switch value {
		case "approved", "draft", "blocked", "complete", "completed":
			return "LEGACY-ARTIFACT-LIFECYCLE"
		case "missing", "missing, expected later", "missing, expected next":
			return "LEGACY-ARTIFACT-MISSING"
		case "present, complete evidence":
			return "LEGACY-ARTIFACT-COMPLETE-EVIDENCE"
		case "conditional", "conditional, trigger unknown":
			return "LEGACY-ARTIFACT-CONDITIONAL"
		case "not expected":
			return "LEGACY-ARTIFACT-NOT-EXPECTED"
		case "waived":
			return "LEGACY-ARTIFACT-WAIVED"
		}
	case "gate":
		if contains([]string{"pending", "complete", "blocked", "waived", "not_expected"}, value) {
			return "LEGACY-GATE"
		}
	case "verdict":
		if contains([]string{"PASS", "CONCERNS", "FAIL", "WAIVED"}, value) {
			return "LEGACY-VERDICT"
		}
	case "session":
		if value == "yes" || value == "no" {
			return "LEGACY-SESSION"
		}
	case "handoff":
		if value == "yes" || value == "no" {
			return "LEGACY-HANDOFF"
		}
	}
	return "LEGACY-UNMAPPED"
}

func mirrorRuleID(state string) string {
	switch state {
	case "mirror_render_failed":
		return "MIRROR-RENDER-FAILED"
	case "mirror_compare_failed":
		return "MIRROR-COMPARE-FAILED"
	case "mirror_optional_absent":
		return "MIRROR-OPTIONAL-ABSENT"
	case "mirror_present_in_sync":
		return "MIRROR-PRESENT-IN-SYNC"
	case "mirror_present_stale":
		return "MIRROR-PRESENT-STALE"
	case "mirror_required_missing":
		return "MIRROR-REQUIRED-MISSING"
	default:
		return ""
	}
}

func evaluate(tc testCase, rules map[string]string) (output map[string]any, trace ruleTrace, err error) {
	trace = ruleTrace{}
	switch tc.Family {
	case "shape":
		output, err = evaluateShape(tc.Input, rules, trace)
	case "agent_request":
		output, err = evaluateAgentRequest(tc.Input, trace)
	case "fanout":
		output, err = evaluateFanout(tc.Input, trace)
	case "artifact_depth":
		output, err = evaluateArtifactDepth(tc.Input, trace)
	case "risk_challenge":
		output, err = evaluateRiskChallenge(tc.Input, trace)
	case "state":
		output, err = evaluateState(tc.Input, trace)
	case "legacy":
		output, err = evaluateLegacy(tc.Input, trace)
	case "transition":
		output, err = evaluateTransition(tc.Input, rules, trace)
	case "routing":
		output, err = evaluateRouting(tc.Input, rules, trace)
	case "adequacy":
		output, err = evaluateAdequacy(tc.Input, rules, trace)
	case "status":
		output, err = evaluateStatus(tc.Input, rules, trace)
	case "mirror":
		output, err = evaluateMirror(tc.Input, trace)
	default:
		err = fmt.Errorf("unknown family %q", tc.Family)
	}
	return output, trace, err
}

func evaluateShape(input map[string]any, rules map[string]string, trace ruleTrace) (map[string]any, error) {
	matched, outcome, err := classifyShape(input, rules, trace)
	if err != nil {
		return nil, err
	}
	return map[string]any{"matched_rule": matched, "outcome": outcome}, nil
}

func classifyShape(input map[string]any, rules map[string]string, trace ruleTrace) (string, string, error) {
	if !boolValue(input, "intake_accepted") {
		trace.mark("SHAPE-INTAKE")
		return "SHAPE-INTAKE", "intake_required", nil
	}
	full, err := ruleStringMap(input["full_triggers"], rules, "FULL-", []string{"true", "unknown", "false"})
	if err != nil {
		return "", "", err
	}
	if err := validateAgentShapeCoupling(stringValue(input, "agent_request"), full); err != nil {
		return "", "", err
	}
	fullFloor := false
	for id := range rules {
		if strings.HasPrefix(id, "FULL-") {
			value := full[id]
			if value == "true" || value == "unknown" {
				trace.mark(id)
			}
			fullFloor = fullFloor || value == "true" || value == "unknown"
		}
	}
	if fullFloor {
		trace.mark("SHAPE-FULL-FLOOR")
		return "SHAPE-FULL-FLOOR", "full_orchestrated", nil
	}
	direct, err := allRulePredicates(input["direct_predicates"], rules, "DIRECT-")
	if err != nil {
		return "", "", err
	}
	if direct {
		trace.mark("SHAPE-DIRECT")
		trace.markPrefix(rules, "DIRECT-")
		return "SHAPE-DIRECT", "direct_path", nil
	}
	lean, err := allRulePredicates(input["lean_predicates"], rules, "LEAN-")
	if err != nil {
		return "", "", err
	}
	if lean {
		trace.mark("SHAPE-LEAN")
		trace.markPrefix(rules, "LEAN-")
		return "SHAPE-LEAN", "lean_local", nil
	}
	trace.mark("SHAPE-FALLBACK-FULL")
	return "SHAPE-FALLBACK-FULL", "full_orchestrated", nil
}

func validateAgentShapeCoupling(agentRequest string, fullTriggers map[string]string) error {
	if !contains([]string{"absent", "capability_only", "substantive"}, agentRequest) {
		return fmt.Errorf("invalid agent_request %q", agentRequest)
	}
	want := "false"
	if agentRequest == "substantive" {
		want = "true"
	}
	if fullTriggers["FULL-AGENT-SUBSTANTIVE"] != want {
		return fmt.Errorf("agent_request=%s requires FULL-AGENT-SUBSTANTIVE=%s", agentRequest, want)
	}
	return nil
}

func allRulePredicates(raw any, rules map[string]string, prefix string) (bool, error) {
	values, ok := raw.(map[string]any)
	if !ok {
		return false, fmt.Errorf("missing %s predicate map", strings.TrimSuffix(prefix, "-"))
	}
	for id := range values {
		if _, ok := rules[id]; !ok || !strings.HasPrefix(id, prefix) {
			return false, fmt.Errorf("unknown %s predicate %s", strings.TrimSuffix(prefix, "-"), id)
		}
	}
	allTrue := true
	for id := range rules {
		if strings.HasPrefix(id, prefix) {
			value, ok := values[id].(bool)
			if !ok {
				return false, fmt.Errorf("missing or invalid predicate %s", id)
			}
			allTrue = allTrue && value
		}
	}
	return allTrue, nil
}

func ruleStringMap(raw any, rules map[string]string, prefix string, allowed []string) (map[string]string, error) {
	values, ok := raw.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("missing %s predicate map", strings.TrimSuffix(prefix, "-"))
	}
	result := make(map[string]string, len(values))
	for id, rawValue := range values {
		if _, ok := rules[id]; !ok || !strings.HasPrefix(id, prefix) {
			return nil, fmt.Errorf("unknown %s predicate %s", strings.TrimSuffix(prefix, "-"), id)
		}
		value, ok := rawValue.(string)
		if !ok || !contains(allowed, value) {
			return nil, fmt.Errorf("invalid %s predicate value for %s", strings.TrimSuffix(prefix, "-"), id)
		}
		result[id] = value
	}
	for id := range rules {
		if strings.HasPrefix(id, prefix) {
			if _, ok := result[id]; !ok {
				return nil, fmt.Errorf("missing %s trigger %s", strings.ToLower(strings.TrimSuffix(prefix, "-")), id)
			}
		}
	}
	return result, nil
}

func evaluateAgentRequest(input map[string]any, trace ruleTrace) (map[string]any, error) {
	gate := "not_applicable"
	if boolValue(input, "required_lane") && !boolValue(input, "authorized") {
		gate = "blocked_missing_authorization"
	}
	if boolValue(input, "substantive") {
		trace.mark("AGENT-SUBSTANTIVE")
		return map[string]any{"agent_request": "substantive", "full_floor": true, "required_lane_gate": gate}, nil
	}
	if boolValue(input, "authorized") {
		trace.mark("AGENT-CAPABILITY")
		return map[string]any{"agent_request": "capability_only", "full_floor": false, "required_lane_gate": gate}, nil
	}
	trace.mark("AGENT-ABSENT")
	return map[string]any{"agent_request": "absent", "full_floor": false, "required_lane_gate": gate}, nil
}

func evaluateFanout(input map[string]any, trace ruleTrace) (map[string]any, error) {
	requested := numberValue(input, "requested_concurrency")
	if requested < 0 || requested != float64(int64(requested)) {
		return nil, errors.New("requested_concurrency must be a non-negative integer")
	}
	local := !boolValue(input, "independent_questions") ||
		!boolValue(input, "separate_context_benefit") ||
		boolValue(input, "small_or_sequential") ||
		boolValue(input, "shared_mutable_state")
	if local {
		trace.mark("FANOUT-LOCAL")
		return map[string]any{"mode": "local", "concurrent_lanes": 0, "excess_reason_required": false}, nil
	}
	if requested < 1 {
		return nil, errors.New("fan-out requires at least one requested lane")
	}
	trace.mark("FANOUT-INDEPENDENT")
	trace.mark("FANOUT-CONCURRENCY")
	excess := requested > 3
	if excess && strings.TrimSpace(stringValue(input, "excess_reason")) == "" {
		return nil, errors.New("concurrency above three requires a task-specific reason")
	}
	return map[string]any{"mode": "fan_out", "concurrent_lanes": int(requested), "excess_reason_required": excess}, nil
}

func evaluateArtifactDepth(input map[string]any, trace ruleTrace) (map[string]any, error) {
	switch stringValue(input, "shape") {
	case "direct_path":
		trace.mark("ARTIFACT-DIRECT", "DEPTH-DIRECT")
		return map[string]any{"workflow_plan": "not_expected", "phase_control": "not_expected", "spec": "not_expected", "tasks": "not_expected", "companions_policy": "all_not_expected"}, nil
	case "lean_local":
		trace.mark("ARTIFACT-LEAN", "DEPTH-LEAN")
		return map[string]any{"workflow_plan": "conditional", "phase_control": "conditional", "spec": "expected", "tasks": "expected", "companions_policy": "resolve_each_from_owner_trigger"}, nil
	case "full_orchestrated":
		trace.mark("ARTIFACT-FULL", "DEPTH-FULL")
		return map[string]any{"workflow_plan": "expected", "phase_control": "conditional", "spec": "expected", "tasks": "expected", "companions_policy": "resolve_each_from_owner_trigger"}, nil
	default:
		return nil, errors.New("invalid artifact depth shape")
	}
}

func evaluateRiskChallenge(input map[string]any, trace ruleTrace) (map[string]any, error) {
	field := stringValue(input, "field")
	outcome := stringValue(input, "value")
	if field != "risk_challenge_outcome" || !contains([]string{"PASS", "CONCERNS", "RECLASSIFY_FULL"}, outcome) {
		return nil, errors.New("invalid risk challenge namespace or outcome")
	}
	if stringValue(input, "execution_shape") != "lean_local" ||
		!contains([]string{"current_session", "durable"}, stringValue(input, "routing_scope")) ||
		!positiveInteger(numberValue(input, "routing_revision")) ||
		stringValue(input, "record_validity") != "current" {
		return nil, errors.New("risk challenge requires current lean routing")
	}
	proofObligations := stringSlice(input["proof_obligations"])
	if outcome == "CONCERNS" && len(proofObligations) == 0 {
		return nil, errors.New("risk challenge concerns require named proof obligations")
	}
	trace.mark("STATE-RISK-CHALLENGE")
	nextAction := map[string]string{
		"PASS":            "specification_review",
		"CONCERNS":        "specification_review_with_obligations",
		"RECLASSIFY_FULL": "guarded_reclassification",
	}[outcome]
	return map[string]any{
		"blocks_lean_handoff": outcome == "RECLASSIFY_FULL",
		"field":               field,
		"next_action":         nextAction,
		"outcome":             outcome,
		"proof_obligations":   proofObligations,
		"valid":               true,
	}, nil
}

func evaluateState(input map[string]any, trace ruleTrace) (map[string]any, error) {
	switch stringValue(input, "kind") {
	case "display":
		trace.mark("STATE-COMPOSE-DISPLAY")
		return map[string]any{"projection": displayProjection(stringValue(input, "phrase"))}, nil
	case "conflict":
		trace.mark("STATE-COMPOSE-FAIL-CLOSED")
		return map[string]any{"authorizes": false, "status": "status_unclear", "valid": false}, nil
	case "record":
	default:
		return nil, errors.New("invalid state kind")
	}
	expectation := stringValue(input, "artifact_expectation")
	state := stringValue(input, "artifact_state")
	validity := stringValue(input, "record_validity")
	waiver := stringValue(input, "waiver_disposition")
	gate := stringValue(input, "procedural_gate_state")
	verdict := stringValue(input, "review_verdict")
	subagentGate := stringValue(input, "subagent_gate")
	sessionBoundary := stringValue(input, "session_boundary")
	handoff := stringValue(input, "handoff_readiness")
	valid := tracedEnum(trace, "STATE-ARTIFACT-EXPECTATION", expectation, []string{"expected", "conditional", "not_expected"})
	valid = tracedEnum(trace, "STATE-ARTIFACT-LIFECYCLE", state, []string{"absent", "draft", "review_ready", "approved", "complete", "blocked"}) && valid
	valid = tracedEnum(trace, "STATE-RECORD-VALIDITY", validity, []string{"current", "stale", "superseded"}) && valid
	valid = tracedEnum(trace, "STATE-WAIVER", waiver, []string{"none", "waived"}) && valid
	valid = tracedEnum(trace, "STATE-EXECUTION-SHAPE", stringValue(input, "execution_shape"), []string{"direct_path", "lean_local", "full_orchestrated"}) && valid
	valid = tracedEnum(trace, "STATE-PHASE", stringValue(input, "phase_state"), []string{"not_started", "active", "complete", "blocked", "reopened"}) && valid
	valid = tracedEnum(trace, "STATE-PROCEDURAL-GATE", gate, []string{"pending", "complete", "blocked", "waived", "not_expected"}) && valid
	valid = tracedEnum(trace, "STATE-REVIEW-VERDICT", verdict, []string{"pending", "PASS", "CONCERNS", "FAIL", "WAIVED"}) && valid
	valid = tracedEnum(trace, "STATE-SUBAGENT-GATE", subagentGate, []string{"complete", "scoped_down", "local_only", "waived", "not_expected", "blocked"}) && valid
	valid = tracedEnum(trace, "STATE-SESSION-BOUNDARY", sessionBoundary, []string{"open", "reached"}) && valid
	valid = tracedEnum(trace, "STATE-HANDOFF", handoff, []string{"not_ready", "ready", "blocked"}) && valid
	valid = tracedEnum(trace, "STATE-ROUTING-SCOPE", stringValue(input, "routing_scope"), []string{"current_session", "durable"}) && valid
	trace.mark("STATE-ROUTING-REVISION")
	valid = positiveInteger(numberValue(input, "routing_revision")) && valid
	if state != "absent" && expectation != "expected" {
		trace.mark("STATE-COMPOSE-EXPECTED")
		valid = false
	} else if state != "absent" {
		trace.mark("STATE-COMPOSE-EXPECTED")
	}
	if (expectation == "conditional" || expectation == "not_expected") && (state != "absent" || waiver != "none") {
		if expectation == "conditional" {
			trace.mark("STATE-COMPOSE-CONDITIONAL")
		} else {
			trace.mark("STATE-COMPOSE-NOT-EXPECTED")
		}
		valid = false
	} else if expectation == "conditional" {
		trace.mark("STATE-COMPOSE-CONDITIONAL")
	} else if expectation == "not_expected" {
		trace.mark("STATE-COMPOSE-NOT-EXPECTED")
	}
	if waiver == "waived" && (expectation != "expected" || state != "absent") {
		trace.mark("STATE-COMPOSE-WAIVER")
		valid = false
	}
	waiverFieldsPresent := hasInputField(input, "waiver_eligible") || hasInputField(input, "waiver_rationale") ||
		hasInputField(input, "waiver_evidence") || hasInputField(input, "waiver_reopen_trigger")
	if waiver == "waived" {
		trace.mark("STATE-COMPOSE-WAIVER")
		valid = valid && boolValue(input, "waiver_eligible") &&
			nonEmptyString(input, "waiver_rationale") &&
			nonEmptyString(input, "waiver_evidence") &&
			nonEmptyString(input, "waiver_reopen_trigger") &&
			gate == "waived" && verdict == "WAIVED"
	} else if waiverFieldsPresent || gate == "waived" || verdict == "WAIVED" {
		valid = false
	}
	gateVerdictValid := (gate == "pending" && verdict == "pending") ||
		(gate == "complete" && contains([]string{"PASS", "CONCERNS"}, verdict)) ||
		(gate == "blocked" && contains([]string{"pending", "FAIL"}, verdict)) ||
		(gate == "waived" && verdict == "WAIVED") ||
		(gate == "not_expected" && verdict == "pending")
	valid = valid && gateVerdictValid
	if handoff == "ready" && sessionBoundary != "reached" {
		valid = false
	}
	observedScope := stringValue(input, "observed_routing_scope")
	observedRevision := numberValue(input, "observed_routing_revision")
	if observedScope != "" || observedRevision != 0 {
		trace.mark("STATE-COMPOSE-REVISION")
		valid = valid && observedScope == stringValue(input, "routing_scope") && observedRevision == numberValue(input, "routing_revision")
	}
	if validity != "current" {
		trace.mark("STATE-COMPOSE-FRESHNESS")
	}
	authorizes := valid && validity == "current" && waiver == "none" && expectation == "expected" &&
		(state == "approved" || state == "complete") && stringValue(input, "phase_state") == "complete" &&
		gate == "complete" && verdict == "PASS" && subagentGate != "blocked" &&
		sessionBoundary == "reached" && handoff == "ready"
	return map[string]any{"authorizes": authorizes, "valid": valid}, nil
}

func hasInputField(input map[string]any, key string) bool {
	_, ok := input[key]
	return ok
}

func nonEmptyString(input map[string]any, key string) bool {
	value, ok := input[key].(string)
	return ok && strings.TrimSpace(value) != ""
}

func tracedEnum(trace ruleTrace, ruleID, value string, allowed []string) bool {
	trace.mark(ruleID)
	return contains(allowed, value)
}

func displayProjection(phrase string) string {
	switch phrase {
	case "missing, expected later":
		return "artifact_expectation=expected,artifact_state=absent"
	case "conditional, trigger unknown":
		return "artifact_expectation=conditional,artifact_state=absent"
	default:
		return "status_unclear"
	}
}

func evaluateLegacy(input map[string]any, trace ruleTrace) (map[string]any, error) {
	field := stringValue(input, "field")
	value := normalizeLegacy(stringValue(input, "value"))
	projection := legacyProjection(field, value)
	trace.mark(legacyRuleID(field, value))
	return map[string]any{"projection": projection}, nil
}

func normalizeLegacy(value string) string {
	value = strings.TrimSpace(value)
	value = strings.TrimSuffix(value, ".")
	value = strings.TrimSpace(value)
	value = strings.Trim(value, "`")
	value = strings.TrimSpace(value)
	return value
}

func legacyProjection(field, value string) string {
	switch field {
	case "shape":
		switch value {
		case "direct path", "direct_path":
			return "execution_shape=direct_path"
		case "lean local", "lean_local", "lightweight local", "lightweight_local":
			return "execution_shape=lean_local"
		case "full orchestrated", "full_orchestrated":
			return "execution_shape=full_orchestrated"
		}
	case "phase":
		switch value {
		case "pending", "not_started":
			return "phase_state=not_started"
		case "active", "in_progress", "in progress":
			return "phase_state=active"
		case "complete", "completed", "done":
			return "phase_state=complete"
		case "blocked", "reopened":
			return "phase_state=" + value
		}
	case "artifact":
		switch value {
		case "approved", "draft", "blocked", "complete":
			return "artifact_expectation=expected,artifact_state=" + value + ",record_validity=current"
		case "completed":
			return "artifact_expectation=expected,artifact_state=complete,record_validity=current"
		case "missing", "missing, expected later", "missing, expected next":
			return "artifact_expectation=expected,artifact_state=absent,record_validity=current"
		case "present, complete evidence":
			return "artifact_expectation=expected,artifact_state=complete,record_validity=current"
		case "conditional", "conditional, trigger unknown":
			return "artifact_expectation=conditional,artifact_state=absent,record_validity=current"
		case "not expected":
			return "artifact_expectation=not_expected,artifact_state=absent,record_validity=current"
		case "waived":
			return "artifact_expectation=expected,artifact_state=absent,waiver_disposition=waived,record_validity=current"
		}
	case "gate":
		if contains([]string{"pending", "complete", "blocked", "waived", "not_expected"}, value) {
			return "procedural_gate_state=" + value
		}
	case "verdict":
		if contains([]string{"PASS", "CONCERNS", "FAIL", "WAIVED"}, value) {
			return "review_verdict=" + value
		}
	case "session":
		if value == "yes" {
			return "session_boundary=reached"
		}
		if value == "no" {
			return "session_boundary=open"
		}
	case "handoff":
		if value == "yes" {
			return "handoff_readiness=ready"
		}
		if value == "no" {
			return "handoff_readiness=not_ready"
		}
	}
	return "legacy_unmapped"
}

func evaluateTransition(input map[string]any, rules map[string]string, trace ruleTrace) (map[string]any, error) {
	kind := stringValue(input, "kind")
	trace.mark(map[string]string{
		"direct_initial": "TRANS-DIRECT-INITIAL", "durable_initial": "TRANS-DURABLE-INITIAL",
		"direct_eligibility_loss": "TRANS-DIRECT-ELIGIBILITY-LOSS", "direct_escalate": "TRANS-DIRECT-ESCALATE",
		"upward": "TRANS-UPWARD", "downgrade": "TRANS-DOWNGRADE", "refresh": "TRANS-REFRESH",
		"durable_revision": "TRANS-DURABLE-REVISION", "partial_conflict": "TRANS-PARTIAL-CONFLICT",
		"durable_history": "TRANS-DURABLE-HISTORY", "post_ledger_reopen": "TRANS-POST-LEDGER-REOPEN",
	}[kind])
	switch kind {
	case "direct_initial":
		return map[string]any{"revision": float64(1), "scope": "current_session"}, nil
	case "durable_initial":
		return map[string]any{"revision": float64(1), "scope": "durable"}, nil
	case "direct_eligibility_loss":
		if err := requireInputFields(input, "predicate_false_or_unknown"); err != nil {
			return nil, err
		}
		return map[string]any{"direct_writes_allowed": !boolValue(input, "predicate_false_or_unknown")}, nil
	case "direct_escalate":
		if err := requireInputFields(input, "source_scope", "source_revision", "source_record_validity", "active_scope", "active_revision", "prior_edits_unapproved", "target_shape", "intake_accepted", "agent_request", "full_triggers", "direct_predicates", "lean_predicates"); err != nil {
			return nil, err
		}
		_, authoritativeShape, err := classifyShape(input, rules, nil)
		if err != nil {
			return nil, err
		}
		allowed := stringValue(input, "source_scope") == "current_session" &&
			positiveInteger(numberValue(input, "source_revision")) &&
			stringValue(input, "source_record_validity") == "current" &&
			stringValue(input, "active_scope") == stringValue(input, "source_scope") &&
			positiveInteger(numberValue(input, "active_revision")) &&
			numberValue(input, "active_revision") == numberValue(input, "source_revision") &&
			boolValue(input, "prior_edits_unapproved") &&
			contains([]string{"lean_local", "full_orchestrated"}, stringValue(input, "target_shape")) &&
			authoritativeShape == stringValue(input, "target_shape")
		if !allowed {
			return map[string]any{"allowed": false, "source_preserved": true}, nil
		}
		return map[string]any{"allowed": allowed, "prior_validity": "stale", "revision": float64(1), "scope": "durable"}, nil
	case "upward":
		if err := requireInputFields(input, "prior_shape", "target_shape", "intake_accepted", "agent_request", "full_triggers", "direct_predicates", "lean_predicates"); err != nil {
			return nil, err
		}
		_, authoritativeShape, err := classifyShape(input, rules, nil)
		if err != nil {
			return nil, err
		}
		pair := stringValue(input, "prior_shape") + "->" + stringValue(input, "target_shape")
		return map[string]any{"allowed": authoritativeShape == stringValue(input, "target_shape") && contains([]string{"direct_path->lean_local", "direct_path->full_orchestrated", "lean_local->full_orchestrated"}, pair)}, nil
	case "downgrade":
		if err := requireInputFields(input, "prior_shape", "target_shape", "intake_accepted", "agent_request", "full_triggers", "direct_predicates", "lean_predicates"); err != nil {
			return nil, err
		}
		_, authoritativeShape, err := classifyShape(input, rules, nil)
		if err != nil {
			return nil, err
		}
		pair := stringValue(input, "prior_shape") + "->" + stringValue(input, "target_shape")
		return map[string]any{"allowed": authoritativeShape == stringValue(input, "target_shape") && contains([]string{"full_orchestrated->lean_local", "full_orchestrated->direct_path", "lean_local->direct_path"}, pair)}, nil
	case "refresh":
		if err := requireInputFields(input, "revision", "dependent_record_dispositions"); err != nil {
			return nil, err
		}
		if !positiveInteger(numberValue(input, "revision")) {
			return nil, errors.New("refresh requires a positive prior revision")
		}
		dispositionsValid, err := validateDependentDispositions(input["dependent_record_dispositions"], true)
		if err != nil {
			return nil, err
		}
		if !dispositionsValid {
			return nil, errors.New("refresh requires explicit dependent record dispositions")
		}
		return map[string]any{
			"adequacy_validity": "stale", "artifact_validity": "stale", "gate_state": "pending",
			"handoff": "blocked", "invalidated_records": float64(len(input["dependent_record_dispositions"].(map[string]any))),
			"revision": numberValue(input, "revision") + 1, "verdict_validity": "stale",
		}, nil
	case "durable_revision":
		if err := requireInputFields(input, "revision"); err != nil {
			return nil, err
		}
		if !positiveInteger(numberValue(input, "revision")) {
			return nil, errors.New("durable revision requires a positive prior revision")
		}
		return map[string]any{"revision": numberValue(input, "revision") + 1, "scope": "durable"}, nil
	case "partial_conflict":
		if err := requireInputFields(input, "master_scope", "phase_scope", "master_revision", "phase_revision"); err != nil {
			return nil, err
		}
		ready := stringValue(input, "master_scope") == "durable" &&
			stringValue(input, "phase_scope") == "durable" &&
			positiveInteger(numberValue(input, "master_revision")) &&
			positiveInteger(numberValue(input, "phase_revision")) &&
			numberValue(input, "master_revision") == numberValue(input, "phase_revision")
		return map[string]any{"handoff": map[bool]string{true: "ready", false: "blocked"}[ready]}, nil
	case "durable_history":
		return map[string]any{"prior_validity": "superseded"}, nil
	case "post_ledger_reopen":
		if err := requireInputFields(input, "fresh_task_review", "blocker_recorded_in_tasks", "readiness_stale", "active_scope", "active_revision", "review_scope", "review_revision", "review_record_validity"); err != nil {
			return nil, err
		}
		identityMatch := stringValue(input, "active_scope") == "durable" &&
			stringValue(input, "review_scope") == "durable" &&
			positiveInteger(numberValue(input, "active_revision")) &&
			positiveInteger(numberValue(input, "review_revision")) &&
			numberValue(input, "active_revision") == numberValue(input, "review_revision") &&
			stringValue(input, "review_record_validity") == "current"
		authorizes := boolValue(input, "blocker_recorded_in_tasks") && boolValue(input, "readiness_stale") &&
			boolValue(input, "fresh_task_review") && identityMatch
		return map[string]any{"tasks_authorizes": authorizes, "tasks_is_state_source": true}, nil
	default:
		return nil, fmt.Errorf("invalid transition kind %q", kind)
	}
}

func evaluateRouting(input map[string]any, rules map[string]string, trace ruleTrace) (map[string]any, error) {
	research := stringValue(input, "research_expectation")
	trace.mark("ROUTING-RESEARCH", "ROUTING-NEXT-PHASE", "ROUTING-NO-COLLAPSE")
	reasons := stringSlice(input["phase_control_reasons"])
	for _, reason := range reasons {
		if !contains([]string{"multi_lane", "fan_in", "formal_challenge", "multi_session_stop", "named_review_checkpoint", "named_validation_checkpoint"}, reason) {
			return nil, fmt.Errorf("unknown phase-control reason %q", reason)
		}
	}
	nextTargets := stringSlice(input["next_targets"])
	next := ""
	if len(nextTargets) == 1 {
		next = nextTargets[0]
	}
	valid := contains([]string{"expected", "conditional", "not_expected"}, research) && len(nextTargets) == 1 && contains(canonicalRoutingTargets(), next)
	switch research {
	case "expected", "conditional":
		valid = valid && next == "research"
	case "not_expected":
		trace.mark("ROUTING-RESEARCH-SKIP")
		valid = valid && next == "specification"
	}
	phaseControlRequired := boolValue(input, "durable_routing") && len(reasons) > 0
	if len(reasons) > 0 {
		trace.mark("ROUTING-PHASE-CONTROL")
	}
	if boolValue(input, "mandatory_gate") && len(reasons) == 0 {
		trace.mark("ROUTING-GATE-NOT-FILE")
	}
	if len(reasons) > 0 && !boolValue(input, "durable_routing") {
		valid = false
	}
	dedicatedPlanning := "not_required"
	if boolValue(input, "dedicated_planning_requested") {
		trace.mark("ROUTING-DEDICATED-PLANNING")
		if !boolValue(input, "durable_routing") || len(reasons) == 0 {
			dedicatedPlanning = "rejected_missing_durable_shape_route"
			valid = false
		} else {
			if err := requireInputFields(input, "intake_accepted", "agent_request", "full_triggers", "direct_predicates", "lean_predicates", "recorded_shape", "matched_rule", "routing_scope", "routing_revision", "record_validity"); err != nil {
				return nil, err
			}
			authoritativeRule, authoritativeShape, err := classifyShape(input, rules, nil)
			if err != nil {
				return nil, err
			}
			currentShapeRoute := stringValue(input, "recorded_shape") == authoritativeShape &&
				stringValue(input, "matched_rule") == authoritativeRule &&
				stringValue(input, "routing_scope") == "durable" &&
				positiveInteger(numberValue(input, "routing_revision")) &&
				stringValue(input, "record_validity") == "current"
			if currentShapeRoute {
				dedicatedPlanning = "consume_existing_shape"
			} else {
				dedicatedPlanning = "rejected_missing_durable_shape_route"
				valid = false
			}
		}
	}
	return map[string]any{
		"dedicated_planning":    dedicatedPlanning,
		"next_phase":            next,
		"phase_control":         map[bool]string{true: "required", false: "not_required"}[phaseControlRequired],
		"same_session_collapse": "prohibited",
		"valid":                 valid,
	}, nil
}

func canonicalRoutingTargets() []string {
	return []string{
		"intake", "workflow-planning", "research", "specification", "specification-review",
		"system-integration-design", "go-code-ownership-design", "technical-design-review",
		"test-design", "planning", "task-review/readiness", "implementation",
		"validation-closeout", "user-decision", "specialist-decision",
	}
}

func evaluateAdequacy(input map[string]any, rules map[string]string, trace ruleTrace) (map[string]any, error) {
	if _, err := ruleStringMap(input["full_triggers"], rules, "FULL-", []string{"true", "unknown", "false"}); err != nil {
		return nil, err
	}
	if _, err := allRulePredicates(input["direct_predicates"], rules, "DIRECT-"); err != nil {
		return nil, err
	}
	if _, err := allRulePredicates(input["lean_predicates"], rules, "LEAN-"); err != nil {
		return nil, err
	}
	matchedRule, authoritativeShape, err := classifyShape(input, rules, nil)
	if err != nil {
		return nil, err
	}
	fullTrigger := false
	for _, value := range stringMap(input["full_triggers"]) {
		fullTrigger = fullTrigger || value == "true" || value == "unknown"
	}
	if stringValue(input, "selected_shape") == "full_orchestrated" {
		trace.mark("ADEQUACY-FULL-SHAPE")
	}
	if fullTrigger {
		trace.mark("ADEQUACY-FULL-TRIGGER")
	}
	if boolValue(input, "durable_planning") {
		trace.mark("ADEQUACY-DURABLE-PLANNING")
	}
	changeKind := stringValue(input, "change_kind")
	if !contains([]string{"none", "upward", "downgrade", "reclassification", "refresh"}, changeKind) {
		return nil, fmt.Errorf("invalid adequacy change_kind %q", changeKind)
	}
	reclassificationChange := changeKind == "downgrade" || changeKind == "reclassification"
	if reclassificationChange {
		trace.mark("ADEQUACY-RECLASSIFICATION")
	}
	required := stringValue(input, "selected_shape") == "full_orchestrated" ||
		fullTrigger || boolValue(input, "durable_planning") || reclassificationChange
	needsStaleDisposition := changeKind != "none"
	dispositionsValid, err := validateDependentDispositions(input["dependent_record_dispositions"], needsStaleDisposition)
	if err != nil {
		return nil, err
	}
	identityCurrent := stringValue(input, "active_scope") == "durable" &&
		stringValue(input, "observed_scope") == "durable" &&
		positiveInteger(numberValue(input, "active_revision")) &&
		positiveInteger(numberValue(input, "observed_revision")) &&
		numberValue(input, "active_revision") == numberValue(input, "observed_revision") &&
		stringValue(input, "record_validity") == "current"
	routeValid := stringValue(input, "selected_shape") == authoritativeShape &&
		stringValue(input, "selected_rule") == matchedRule &&
		identityCurrent && dispositionsValid
	return map[string]any{
		"advisory_only":       true,
		"authoritative_rule":  matchedRule,
		"authoritative_shape": authoritativeShape,
		"may_mutate":          false,
		"required":            required,
		"route_valid":         routeValid,
	}, nil
}

func validateDependentDispositions(raw any, required bool) (bool, error) {
	values, ok := raw.(map[string]any)
	if !ok {
		return false, errors.New("dependent_record_dispositions must be an object")
	}
	if required && len(values) == 0 {
		return false, nil
	}
	if !required && len(values) != 0 {
		return false, nil
	}
	for record, rawDisposition := range values {
		if strings.TrimSpace(record) == "" {
			return false, errors.New("dependent record name must not be empty")
		}
		disposition, ok := rawDisposition.(string)
		if !ok {
			return false, fmt.Errorf("dependent record %s disposition must be a string", record)
		}
		if disposition == "stale" || disposition == "superseded" {
			continue
		}
		if strings.HasPrefix(disposition, "preserved:") && strings.TrimSpace(strings.TrimPrefix(disposition, "preserved:")) != "" {
			continue
		}
		return false, fmt.Errorf("invalid dependent record disposition for %s", record)
	}
	return true, nil
}

func evaluateStatus(input map[string]any, rules map[string]string, trace ruleTrace) (map[string]any, error) {
	if boolValue(input, "tasks_present") {
		trace.mark("STATUS-TASKS")
		if err := requireInputFields(input,
			"active_scope", "active_revision", "active_execution_shape", "active_matched_rule",
			"tasks_scope", "tasks_revision", "tasks_record_validity",
			"tasks_blockers_clear", "tasks_concerns_complete"); err != nil {
			return nil, err
		}
		reportComplete, err := validateStatusReport(input)
		if err != nil {
			return nil, err
		}
		activeShape := stringValue(input, "active_execution_shape")
		activeRule := stringValue(input, "active_matched_rule")
		activeRouteEligible := (activeShape == "lean_local" && activeRule == "SHAPE-LEAN") ||
			(activeShape == "full_orchestrated" && contains([]string{"SHAPE-FULL-FLOOR", "SHAPE-FALLBACK-FULL"}, activeRule))
		tasksRouteEligible := activeRouteEligible &&
			stringValue(input, "report_execution_shape") == activeShape &&
			stringValue(input, "report_matched_rule") == activeRule &&
			stringValue(input, "report_artifact_expectation") == "expected"
		sourceCurrent := tasksRouteEligible && durableIdentityMatches(
			stringValue(input, "active_scope"), numberValue(input, "active_revision"),
			stringValue(input, "tasks_scope"), numberValue(input, "tasks_revision"),
		) && stringValue(input, "tasks_record_validity") == "current" &&
			stringValue(input, "report_record_validity") == stringValue(input, "tasks_record_validity")
		reviewEligible := statusReviewEligible(input, boolValue(input, "tasks_concerns_complete"), boolValue(input, "tasks_waiver_eligible"))
		handoffAuthorized := sourceCurrent && reportComplete && statusReportAllowsHandoff(input) &&
			(stringValue(input, "report_next_phase") != "implementation" || reviewEligible)
		implementationAuthorized := handoffAuthorized && stringValue(input, "report_phase_state") == "complete" &&
			reviewEligible &&
			stringValue(input, "report_next_phase") == "implementation" &&
			stringValue(input, "report_allowed_writes") == "implementation" && boolValue(input, "tasks_blockers_clear")
		return statusResult("STATUS-TASKS", sourceCurrent, reportComplete, handoffAuthorized, implementationAuthorized), nil
	}
	if boolValue(input, "durable_present") {
		trace.mark("STATUS-DURABLE-CONTROL")
		if err := requireInputFields(input,
			"active_scope", "active_revision", "master_scope", "phase_scope", "master_revision", "phase_revision",
			"master_record_validity", "phase_record_validity"); err != nil {
			return nil, err
		}
		reportComplete, err := validateStatusReport(input)
		if err != nil {
			return nil, err
		}
		masterCurrent := durableIdentityMatches(
			stringValue(input, "active_scope"), numberValue(input, "active_revision"),
			stringValue(input, "master_scope"), numberValue(input, "master_revision"),
		) && stringValue(input, "master_record_validity") == "current"
		phaseCurrent := durableIdentityMatches(
			stringValue(input, "active_scope"), numberValue(input, "active_revision"),
			stringValue(input, "phase_scope"), numberValue(input, "phase_revision"),
		) && stringValue(input, "phase_record_validity") == "current"
		sourceCurrent := masterCurrent && phaseCurrent && stringValue(input, "report_record_validity") == "current"
		handoffAuthorized := sourceCurrent && reportComplete && statusReportAllowsHandoff(input)
		result := statusResult("STATUS-DURABLE-CONTROL", sourceCurrent, reportComplete, handoffAuthorized, false)
		result["conflict"] = !sourceCurrent
		return result, nil
	}
	if boolValue(input, "phase_artifact_present") {
		trace.mark("STATUS-PHASE-ARTIFACTS")
		if err := requireInputFields(input,
			"active_scope", "active_revision", "phase_artifact_scope", "phase_artifact_revision",
			"phase_artifact_record_validity", "phase_concerns_complete"); err != nil {
			return nil, err
		}
		reportComplete, err := validateStatusReport(input)
		if err != nil {
			return nil, err
		}
		sourceCurrent := durableIdentityMatches(
			stringValue(input, "active_scope"), numberValue(input, "active_revision"),
			stringValue(input, "phase_artifact_scope"), numberValue(input, "phase_artifact_revision"),
		) && stringValue(input, "phase_artifact_record_validity") == "current" &&
			stringValue(input, "report_record_validity") == stringValue(input, "phase_artifact_record_validity")
		handoffAuthorized := sourceCurrent && reportComplete && statusReportAllowsHandoff(input) &&
			statusReviewEligible(input, boolValue(input, "phase_concerns_complete"), false)
		return statusResult("STATUS-PHASE-ARTIFACTS", sourceCurrent, reportComplete, handoffAuthorized, false), nil
	}
	if boolValue(input, "direct_envelope_present") {
		trace.mark("STATUS-DIRECT-ENVELOPE")
		if err := requireInputFields(input,
			"provenance", "same_session", "direct_record_validity", "framing_accepted", "trigger_audit_complete",
			"direct_agent_request", "direct_full_triggers", "direct_predicates", "direct_lean_predicates", "matched_rule", "actor",
			"routing_scope", "routing_revision", "active_scope", "active_revision", "proof_result", "reopen_seam_present"); err != nil {
			return nil, err
		}
		reportComplete, err := validateStatusReport(input)
		if err != nil {
			return nil, err
		}
		if _, err := ruleStringMap(input["direct_full_triggers"], rules, "FULL-", []string{"true", "unknown", "false"}); err != nil {
			return nil, err
		}
		if _, err := allRulePredicates(input["direct_predicates"], rules, "DIRECT-"); err != nil {
			return nil, err
		}
		if _, err := allRulePredicates(input["direct_lean_predicates"], rules, "LEAN-"); err != nil {
			return nil, err
		}
		matched, shape, err := classifyShape(map[string]any{
			"intake_accepted":   boolValue(input, "framing_accepted"),
			"agent_request":     stringValue(input, "direct_agent_request"),
			"full_triggers":     input["direct_full_triggers"],
			"direct_predicates": input["direct_predicates"],
			"lean_predicates":   input["direct_lean_predicates"],
		}, rules, nil)
		if err != nil {
			return nil, err
		}
		sourceCurrent := stringValue(input, "provenance") == "orchestrator_current_session" &&
			boolValue(input, "same_session") &&
			stringValue(input, "direct_record_validity") == "current" &&
			boolValue(input, "framing_accepted") &&
			boolValue(input, "trigger_audit_complete") &&
			matched == "SHAPE-DIRECT" && shape == "direct_path" && stringValue(input, "matched_rule") == matched &&
			stringValue(input, "report_execution_shape") == shape && stringValue(input, "report_matched_rule") == matched &&
			stringValue(input, "actor") == "orchestrator" &&
			stringValue(input, "routing_scope") == "current_session" &&
			positiveInteger(numberValue(input, "routing_revision")) &&
			stringValue(input, "active_scope") == stringValue(input, "routing_scope") &&
			positiveInteger(numberValue(input, "active_revision")) &&
			numberValue(input, "active_revision") == numberValue(input, "routing_revision") &&
			stringValue(input, "report_record_validity") == stringValue(input, "direct_record_validity")
		implementationAuthorized := sourceCurrent && reportComplete &&
			stringValue(input, "proof_result") == "PASS" &&
			boolValue(input, "reopen_seam_present") && stringValue(input, "report_allowed_writes") == "implementation"
		handoffAuthorized := sourceCurrent && reportComplete && statusReportAllowsHandoff(input)
		return statusResult("STATUS-DIRECT-ENVELOPE", sourceCurrent, reportComplete, handoffAuthorized, implementationAuthorized), nil
	}
	trace.mark("STATUS-UNSUPPORTED")
	return statusResult("STATUS-UNSUPPORTED", false, false, false, false), nil
}

func validateStatusReport(input map[string]any) (bool, error) {
	if err := requireInputFields(input,
		"report_execution_shape", "report_matched_rule", "report_shape_evidence", "report_adequacy_required",
		"report_adequacy_result", "report_phase_state", "report_session_boundary", "report_artifact_expectation",
		"report_artifact_state", "report_record_validity", "report_gate_state", "report_review_verdict",
		"report_handoff_readiness", "report_allowed_writes", "report_next_phase", "report_next_action"); err != nil {
		return false, err
	}
	shape := stringValue(input, "report_execution_shape")
	matchedRule := stringValue(input, "report_matched_rule")
	shapeRuleValid := (shape == "direct_path" && matchedRule == "SHAPE-DIRECT") ||
		(shape == "lean_local" && matchedRule == "SHAPE-LEAN") ||
		(shape == "full_orchestrated" && contains([]string{"SHAPE-FULL-FLOOR", "SHAPE-FALLBACK-FULL"}, matchedRule))
	adequacyRequired := boolValue(input, "report_adequacy_required")
	adequacyResult := stringValue(input, "report_adequacy_result")
	adequacyValid := (!adequacyRequired && adequacyResult == "not_required") ||
		(adequacyRequired && contains([]string{"PASS", "CONCERNS", "FAIL", "blocked"}, adequacyResult))
	expectation := stringValue(input, "report_artifact_expectation")
	artifactState := stringValue(input, "report_artifact_state")
	artifactCompositionValid := contains([]string{"expected", "conditional", "not_expected"}, expectation) &&
		contains([]string{"absent", "draft", "review_ready", "approved", "complete", "blocked"}, artifactState) &&
		(artifactState == "absent" || expectation == "expected")
	gate := stringValue(input, "report_gate_state")
	verdict := stringValue(input, "report_review_verdict")
	gateVerdictValid := (gate == "pending" && verdict == "pending") ||
		(gate == "complete" && contains([]string{"PASS", "CONCERNS"}, verdict)) ||
		(gate == "blocked" && contains([]string{"pending", "FAIL"}, verdict)) ||
		(gate == "waived" && verdict == "WAIVED") ||
		(gate == "not_expected" && verdict == "pending")
	handoff := stringValue(input, "report_handoff_readiness")
	session := stringValue(input, "report_session_boundary")
	return shapeRuleValid && nonEmptyString(input, "report_shape_evidence") && adequacyValid &&
		contains([]string{"not_started", "active", "complete", "blocked", "reopened"}, stringValue(input, "report_phase_state")) &&
		contains([]string{"open", "reached"}, session) && artifactCompositionValid &&
		contains([]string{"current", "stale", "superseded"}, stringValue(input, "report_record_validity")) &&
		gateVerdictValid && contains([]string{"not_ready", "ready", "blocked"}, handoff) &&
		(handoff != "ready" || session == "reached") &&
		contains([]string{"none", "workflow-control", "phase-artifacts", "implementation", "validation-only"}, stringValue(input, "report_allowed_writes")) &&
		contains(canonicalRoutingTargets(), stringValue(input, "report_next_phase")) &&
		nonEmptyString(input, "report_next_action"), nil
}

func durableIdentityMatches(activeScope string, activeRevision float64, recordScope string, recordRevision float64) bool {
	return activeScope == "durable" && recordScope == "durable" &&
		positiveInteger(activeRevision) && positiveInteger(recordRevision) && activeRevision == recordRevision
}

func statusReportAllowsHandoff(input map[string]any) bool {
	return stringValue(input, "report_record_validity") == "current" &&
		contains([]string{"complete", "blocked", "reopened"}, stringValue(input, "report_phase_state")) &&
		stringValue(input, "report_session_boundary") == "reached" &&
		stringValue(input, "report_handoff_readiness") == "ready"
}

func statusReviewEligible(input map[string]any, concernsComplete, waiverEligible bool) bool {
	gate := stringValue(input, "report_gate_state")
	verdict := stringValue(input, "report_review_verdict")
	return (gate == "complete" && verdict == "PASS") ||
		(gate == "complete" && verdict == "CONCERNS" && concernsComplete) ||
		(gate == "waived" && verdict == "WAIVED" && waiverEligible)
}

func statusResult(source string, sourceCurrent, reportComplete, handoffAuthorized, implementationAuthorized bool) map[string]any {
	return map[string]any{
		"handoff_authorized":        handoffAuthorized,
		"implementation_authorized": implementationAuthorized,
		"report_complete":           reportComplete,
		"source":                    source,
		"source_current":            sourceCurrent,
	}
}

func requireInputFields(input map[string]any, keys ...string) error {
	for _, key := range keys {
		if _, ok := input[key]; !ok {
			return fmt.Errorf("missing required input field %q", key)
		}
	}
	return nil
}

func evaluateMirror(input map[string]any, trace ruleTrace) (map[string]any, error) {
	trace.mark("MIRROR-CANONICAL-AVAILABLE")
	if !boolValue(input, "canonical_available") {
		return map[string]any{"pass": false, "state": "canonical_unavailable"}, nil
	}
	if !boolValue(input, "render_ok") {
		trace.mark("MIRROR-RENDER-FAILED")
		return map[string]any{"pass": false, "state": "mirror_render_failed"}, nil
	}
	if rawTargets, ok := input["targets"].([]any); ok {
		if len(rawTargets) == 0 {
			return nil, errors.New("mirror targets must not be empty")
		}
		states := make([]any, 0, len(rawTargets))
		pass := true
		for _, rawTarget := range rawTargets {
			target, ok := rawTarget.(map[string]any)
			if !ok {
				return nil, errors.New("invalid mirror target")
			}
			if err := validateMirrorTarget(target, true); err != nil {
				return nil, err
			}
			state, targetPass := mirrorTargetState(target, boolValue(input, "strict"))
			trace.mark(mirrorRuleID(state))
			states = append(states, state)
			pass = pass && targetPass
		}
		return map[string]any{"pass": pass, "states": states}, nil
	}
	if err := validateMirrorTarget(input, false); err != nil {
		return nil, err
	}
	state, pass := mirrorTargetState(input, boolValue(input, "strict"))
	trace.mark(mirrorRuleID(state))
	return map[string]any{"pass": pass, "state": state}, nil
}

func validateMirrorTarget(input map[string]any, nested bool) error {
	allowed := map[string]valueKind{
		"present": kindBool, "required": kindBool, "compare_ok": kindBool, "in_sync": kindBool, "target_only_files": kindBool,
	}
	if !nested {
		allowed["canonical_available"] = kindBool
		allowed["render_ok"] = kindBool
		allowed["strict"] = kindBool
		allowed["targets"] = kindObjectArray
	}
	for key, value := range input {
		kind, ok := allowed[key]
		if !ok {
			return fmt.Errorf("unknown mirror target field %q", key)
		}
		if !matchesKind(value, kind) {
			return fmt.Errorf("mirror target field %q must be %s", key, kind)
		}
	}
	if err := requireInputFields(input, "present", "required"); err != nil {
		return err
	}
	if boolValue(input, "present") {
		if err := requireInputFields(input, "compare_ok"); err != nil {
			return err
		}
		if boolValue(input, "compare_ok") {
			if err := requireInputFields(input, "in_sync"); err != nil {
				return err
			}
		}
	}
	return nil
}

func mirrorTargetState(input map[string]any, strict bool) (string, bool) {
	present := boolValue(input, "present")
	required := boolValue(input, "required")
	if !present && required {
		return "mirror_required_missing", false
	}
	if !present {
		return "mirror_optional_absent", true
	}
	if !boolValue(input, "compare_ok") {
		return "mirror_compare_failed", false
	}
	if boolValue(input, "in_sync") && (!strict || !boolValue(input, "target_only_files")) {
		return "mirror_present_in_sync", true
	}
	return "mirror_present_stale", false
}

func boolValue(input map[string]any, key string) bool {
	value, _ := input[key].(bool)
	return value
}

func stringValue(input map[string]any, key string) string {
	value, _ := input[key].(string)
	return value
}

func numberValue(input map[string]any, key string) float64 {
	value, _ := input[key].(float64)
	return value
}

func positiveInteger(value float64) bool {
	return value >= 1 && value == float64(int64(value))
}

func stringMap(raw any) map[string]string {
	result := map[string]string{}
	values, _ := raw.(map[string]any)
	for key, value := range values {
		result[key], _ = value.(string)
	}
	return result
}

func stringSlice(raw any) []string {
	values, _ := raw.([]any)
	result := make([]string, 0, len(values))
	for _, value := range values {
		item, _ := value.(string)
		result = append(result, item)
	}
	return result
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func equalJSON(a, b any) bool {
	return compactJSON(a) == compactJSON(b)
}

func compactJSON(value any) string {
	data, _ := json.Marshal(value)
	return string(data)
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "workflow routing check failed:", err)
	os.Exit(1)
}
