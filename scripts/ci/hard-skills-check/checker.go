package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"unicode"
)

const sharedContractPath = ".agents/skills/specialist-contract.md"

var markdownLinkPattern = regexp.MustCompile(`!?\[[^\]]*\]\(([^)\s]+)(?:\s+[^)]*)?\)`)

type skillDocument struct {
	Name        string
	Description string
	Body        []byte
	Path        string
}

type evalBundle struct {
	SkillName string     `json:"skill_name"`
	Evals     []evalCase `json:"evals"`
}

type evalCase struct {
	ID             json.RawMessage `json:"id"`
	Category       string          `json:"category"`
	TrialClass     string          `json:"trial_class"`
	Prompt         string          `json:"prompt"`
	ExpectedOutput string          `json:"expected_output"`
	Assertions     []string        `json:"assertions"`
	Expectations   []string        `json:"expectations"`
	Files          []string        `json:"files"`
}

func checkRepository(root string, skills []string, scoped bool) error {
	var problems []string
	if !scoped {
		problems = append(problems, validateInventory(root)...)
		problems = append(problems, validateSharedContract(root)...)
		problems = append(problems, validateRetiredIdentifiers(root)...)
	}

	for _, name := range skills {
		problems = append(problems, validateSkill(root, name)...)
	}
	if len(problems) > 0 {
		return errors.New(strings.Join(problems, "\n"))
	}
	return nil
}

func validateInventory(root string) []string {
	var problems []string
	skillsDir := filepath.Join(root, ".agents", "skills")
	targets := targetSkillSet()
	retired := retiredSkillSet()

	entries, err := os.ReadDir(skillsDir)
	if err != nil {
		return []string{fmt.Sprintf("read hard-skill inventory: %v", err)}
	}
	present := make(map[string]bool)
	for _, entry := range entries {
		name := entry.Name()
		if retired[name] {
			problems = append(problems, fmt.Sprintf("retired canonical skill path exists: %s", name))
			continue
		}
		if !entry.IsDir() {
			continue
		}
		if targets[name] {
			present[name] = true
			continue
		}
		if strings.HasPrefix(name, "go-") {
			problems = append(problems, fmt.Sprintf("unknown canonical hard-skill directory: %s", name))
		}
	}
	for _, name := range targetSkills {
		if !present[name] {
			problems = append(problems, fmt.Sprintf("missing canonical hard skill: %s", name))
		}
	}
	return problems
}

func validateSharedContract(root string) []string {
	contract := filepath.Join(root, filepath.FromSlash(sharedContractPath))
	info, err := os.Stat(contract)
	if err != nil || !info.Mode().IsRegular() {
		return []string{"missing non-triggerable shared specialist contract: " + sharedContractPath}
	}
	triggerable := filepath.Join(root, ".agents", "skills", "specialist-contract", "SKILL.md")
	if _, err := os.Lstat(triggerable); err == nil {
		return []string{"shared specialist contract must not be a triggerable SKILL.md"}
	} else if !errors.Is(err, fs.ErrNotExist) {
		return []string{fmt.Sprintf("inspect triggerable shared contract path: %v", err)}
	}
	data, err := os.ReadFile(contract)
	if err != nil {
		return []string{fmt.Sprintf("read shared specialist contract: %v", err)}
	}
	return validateMarkdownLinks(root, contract, data)
}

func validateSkill(root, name string) []string {
	var problems []string
	path := filepath.Join(root, ".agents", "skills", name, "SKILL.md")
	doc, err := readSkillDocument(path)
	if err != nil {
		return []string{fmt.Sprintf("%s: %v", name, err)}
	}
	if doc.Name != name {
		problems = append(problems, fmt.Sprintf("%s: frontmatter name %q does not match directory", name, doc.Name))
	}
	problems = append(problems, validateDescription(name, doc.Description)...)
	problems = append(problems, validateMarkdownLinks(root, path, doc.Body)...)
	if !executionSkills[name] {
		problems = append(problems, validateDirectSharedLink(root, path, doc.Body)...)
		bundle, err := readEvalBundle(root, name)
		if err != nil {
			problems = append(problems, fmt.Sprintf("%s: %v", name, err))
		} else {
			problems = append(problems, validateEvalBundle(name, bundle)...)
		}
	}
	return problems
}

func readSkillDocument(path string) (skillDocument, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return skillDocument{}, fmt.Errorf("read SKILL.md: %w", err)
	}
	return parseSkillDocument(data, path)
}

func parseSkillDocument(data []byte, path string) (skillDocument, error) {
	if !bytes.HasPrefix(data, []byte("---\n")) {
		return skillDocument{}, errors.New("SKILL.md must start with YAML frontmatter")
	}
	closing := bytes.Index(data[4:], []byte("\n---\n"))
	if closing < 0 {
		return skillDocument{}, errors.New("SKILL.md has no closing frontmatter delimiter")
	}
	frontmatter := string(data[4 : 4+closing])
	body := data[4+closing+5:]

	var name string
	var description string
	nameCount := 0
	descriptionCount := 0
	for _, line := range strings.Split(frontmatter, "\n") {
		key, value, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		switch strings.TrimSpace(key) {
		case "name":
			nameCount++
			name = strings.TrimSpace(value)
		case "description":
			descriptionCount++
			parsed, err := parseQuotedDescription(strings.TrimSpace(value))
			if err != nil {
				return skillDocument{}, err
			}
			description = parsed
		}
	}
	if nameCount != 1 {
		return skillDocument{}, fmt.Errorf("frontmatter must contain exactly one one-line name, got %d", nameCount)
	}
	if name == "" {
		return skillDocument{}, errors.New("frontmatter name is empty")
	}
	if descriptionCount != 1 {
		return skillDocument{}, fmt.Errorf("frontmatter must contain exactly one one-line description, got %d", descriptionCount)
	}
	return skillDocument{Name: name, Description: description, Body: body, Path: path}, nil
}

func parseQuotedDescription(value string) (string, error) {
	if len(value) < 2 {
		return "", errors.New("frontmatter description must be a quoted one-line value")
	}
	if value[0] == '"' && value[len(value)-1] == '"' {
		parsed, err := strconv.Unquote(value)
		if err != nil || strings.ContainsAny(parsed, "\r\n") {
			return "", errors.New("frontmatter description must be a valid quoted one-line value")
		}
		return parsed, nil
	}
	if value[0] == '\'' && value[len(value)-1] == '\'' {
		parsed := strings.ReplaceAll(value[1:len(value)-1], "''", "'")
		if strings.ContainsAny(parsed, "\r\n") {
			return "", errors.New("frontmatter description must be a quoted one-line value")
		}
		return parsed, nil
	}
	return "", errors.New("frontmatter description must be a quoted one-line value")
}

func validateDescription(name, description string) []string {
	var problems []string
	if description == "" {
		problems = append(problems, fmt.Sprintf("%s: description is empty", name))
	}
	for _, clause := range []string{"Use when", "Own", "Skip when"} {
		if !strings.Contains(description, clause) {
			problems = append(problems, fmt.Sprintf("%s: description is missing %q", name, clause))
		}
	}
	if countSentences(description) > 2 {
		problems = append(problems, fmt.Sprintf("%s: description exceeds two sentences", name))
	}
	return problems
}

func countSentences(value string) int {
	runes := []rune(value)
	inCode := false
	count := 0
	for i, r := range runes {
		if r == '`' {
			inCode = !inCode
			continue
		}
		if inCode || (r != '.' && r != '?' && r != '!') {
			continue
		}
		if i == len(runes)-1 || unicode.IsSpace(runes[i+1]) {
			count++
		}
	}
	return count
}

func validateDirectSharedLink(root, source string, data []byte) []string {
	wanted := filepath.Clean(filepath.Join(root, filepath.FromSlash(sharedContractPath)))
	for _, match := range markdownLinkPattern.FindAllSubmatch(data, -1) {
		target, ok := resolveRelativeLink(source, string(match[1]))
		if ok && filepath.Clean(target) == wanted {
			return nil
		}
	}
	return []string{fmt.Sprintf("%s: missing direct link to %s", relativePath(root, source), sharedContractPath)}
}

func validateMarkdownLinks(root, source string, data []byte) []string {
	var problems []string
	for _, match := range markdownLinkPattern.FindAllSubmatch(data, -1) {
		raw := string(match[1])
		target, ok := resolveRelativeLink(source, raw)
		if !ok {
			continue
		}
		if _, err := os.Stat(target); err != nil {
			problems = append(problems, fmt.Sprintf("%s: unresolved local Markdown link %q", relativePath(root, source), raw))
		}
	}
	return problems
}

func resolveRelativeLink(source, raw string) (string, bool) {
	parsed, err := url.Parse(strings.Trim(raw, "<>"))
	if err != nil || parsed.Scheme != "" || parsed.Host != "" || parsed.Path == "" || filepath.IsAbs(parsed.Path) {
		return "", false
	}
	path, err := url.PathUnescape(parsed.Path)
	if err != nil {
		return "", false
	}
	return filepath.Clean(filepath.Join(filepath.Dir(source), filepath.FromSlash(path))), true
}

func readEvalBundle(root, name string) (evalBundle, error) {
	path := filepath.Join(root, ".agents", "skills", name, "evals", "evals.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return evalBundle{}, fmt.Errorf("read eval bundle: %w", err)
	}
	var bundle evalBundle
	if err := json.Unmarshal(data, &bundle); err != nil {
		return evalBundle{}, fmt.Errorf("parse eval bundle: %w", err)
	}
	return bundle, nil
}

func validateEvalBundle(name string, bundle evalBundle) []string {
	var problems []string
	if bundle.SkillName != name {
		problems = append(problems, fmt.Sprintf("%s: eval skill_name is %q", name, bundle.SkillName))
	}
	seenIDs := make(map[string]bool)
	categoryCounts := make(map[string]int)
	selected := isSelectedSkill(name)
	selectedCoreCount := 0
	validCategories := make(map[string]bool, len(evalCategories))
	for _, category := range evalCategories {
		validCategories[category] = true
	}
	for index, eval := range bundle.Evals {
		label := fmt.Sprintf("%s eval[%d]", name, index)
		selectedCore := selected && strings.TrimSpace(eval.TrialClass) == ""
		id := strings.TrimSpace(string(eval.ID))
		switch {
		case id == "" || id == "null":
			problems = append(problems, label+": id is empty")
		case seenIDs[id]:
			problems = append(problems, label+": duplicate id "+id)
		default:
			seenIDs[id] = true
		}
		if !validCategories[eval.Category] {
			problems = append(problems, label+": invalid category "+strconv.Quote(eval.Category))
		} else if !selected || selectedCore {
			categoryCounts[eval.Category]++
		}
		if selectedCore {
			selectedCoreCount++
		}
		problems = append(problems, validateEvalCase(label, eval, selectedCore)...)
	}
	for _, category := range evalCategories {
		if categoryCounts[category] == 0 {
			problems = append(problems, fmt.Sprintf("%s: missing eval category %s", name, category))
		}
		if selected && categoryCounts[category] != 1 {
			problems = append(problems, fmt.Sprintf("%s: selected eval category %s must occur exactly once", name, category))
		}
	}
	if selected && selectedCoreCount != len(evalCategories) {
		problems = append(problems, fmt.Sprintf("%s: selected eval bundle must contain exactly four core cases", name))
	}
	return problems
}

func validateEvalCase(label string, eval evalCase, selected bool) []string {
	var problems []string
	if strings.TrimSpace(eval.Prompt) == "" {
		problems = append(problems, label+": prompt is empty")
	}
	if strings.TrimSpace(eval.ExpectedOutput) == "" && len(eval.Assertions) == 0 && len(eval.Expectations) == 0 {
		problems = append(problems, label+": expected output, assertions, and expectations are all empty")
	}
	problems = append(problems, validateNonEmptyStrings(label+" assertions", eval.Assertions)...)
	problems = append(problems, validateNonEmptyStrings(label+" expectations", eval.Expectations)...)
	if selected {
		problems = append(problems, validateSelectedEvalCase(label, eval)...)
	}
	return problems
}

func validateSelectedEvalCase(label string, eval evalCase) []string {
	var problems []string
	if len(eval.Files) != 0 {
		problems = append(problems, label+": selected eval must be self-contained")
	}
	if strings.TrimSpace(eval.ExpectedOutput) == "" || strings.ContainsAny(eval.ExpectedOutput, "\r\n") {
		problems = append(problems, label+": selected eval expected_output must be one non-empty line")
	}
	if len(eval.Assertions) == 0 {
		problems = append(problems, label+": selected eval assertions are empty")
	}
	if len(eval.Expectations) != 0 {
		problems = append(problems, label+": selected eval must not use expectations")
	}
	for _, assertion := range eval.Assertions {
		if strings.ContainsAny(assertion, "\r\n") {
			problems = append(problems, label+": selected eval assertions must be one line each")
		}
	}
	return problems
}

func validateNonEmptyStrings(label string, values []string) []string {
	for index, value := range values {
		if strings.TrimSpace(value) == "" {
			return []string{fmt.Sprintf("%s[%d] is empty", label, index)}
		}
	}
	return nil
}

func validateRetiredIdentifiers(root string) []string {
	var problems []string
	retired := retiredSkillSet()
	retiredNames := make([]string, 0, len(retired))
	for name := range retired {
		retiredNames = append(retiredNames, name)
	}
	slices.Sort(retiredNames)
	seen := make(map[string]bool)
	visit := func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			problems = append(problems, fmt.Sprintf("scan retired identifiers: %v", err))
			return nil
		}
		if entry.IsDir() || entry.Type()&os.ModeSymlink != 0 {
			return nil
		}
		rel := filepath.ToSlash(relativePath(root, path))
		if seen[rel] || strings.HasPrefix(rel, "scripts/ci/hard-skills-check/") {
			return nil
		}
		seen[rel] = true
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			problems = append(problems, fmt.Sprintf("scan %s: %v", rel, readErr))
			return nil
		}
		for _, name := range retiredNames {
			if bytes.Contains(data, []byte(name)) {
				problems = append(problems, fmt.Sprintf("retired skill identifier %q remains in %s", name, rel))
			}
		}
		return nil
	}

	for _, rel := range []string{
		".agents/skills", ".codex/agents", "docs", "scripts",
	} {
		path := filepath.Join(root, filepath.FromSlash(rel))
		if _, err := os.Lstat(path); errors.Is(err, fs.ErrNotExist) {
			continue
		}
		_ = filepath.WalkDir(path, visit)
	}
	for _, rel := range []string{"AGENTS.md", "README.md", "Makefile", "Dockerfile", ".gitleaks.toml"} {
		path := filepath.Join(root, rel)
		if info, err := os.Stat(path); err == nil && info.Mode().IsRegular() {
			_ = visit(path, fs.FileInfoToDirEntry(info), nil)
		}
	}
	return problems
}

func relativePath(root, path string) string {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return path
	}
	return rel
}
