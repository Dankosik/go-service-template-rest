package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type trackedString struct {
	value string
	set   bool
}

func (value *trackedString) String() string { return value.value }

func (value *trackedString) Set(next string) error {
	value.value = next
	value.set = true
	return nil
}

func main() {
	if err := runCLI(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "hard skills check failed:", err)
		os.Exit(1)
	}
}

func runCLI(args []string) error {
	root, err := repositoryRoot()
	if err != nil {
		return err
	}
	if len(args) == 0 {
		return checkRepository(root, targetSkills, false)
	}

	switch args[0] {
	case "check":
		flags := flag.NewFlagSet("check", flag.ContinueOnError)
		flags.SetOutput(os.Stderr)
		var skillNames trackedString
		flags.Var(&skillNames, "skills", "comma-separated exact skill names")
		if err := flags.Parse(args[1:]); err != nil {
			return fmt.Errorf("parse check flags: %w", err)
		}
		if flags.NArg() != 0 {
			return fmt.Errorf("check accepts no positional arguments")
		}
		if !skillNames.set {
			return checkRepository(root, targetSkills, false)
		}
		skills, err := parseSkillScope(skillNames.value)
		if err != nil {
			return err
		}
		return checkRepository(root, skills, true)
	default:
		return fmt.Errorf("usage: hard-skills-check [check [--skills names]]")
	}
}

func parseSkillScope(raw string) ([]string, error) {
	if raw == "" {
		return nil, fmt.Errorf("--skills must not be empty")
	}
	targets := targetSkillSet()
	retired := retiredSkillSet()
	seen := make(map[string]bool)
	parts := strings.Split(raw, ",")
	result := make([]string, 0, len(parts))
	for _, name := range parts {
		if name == "" || strings.TrimSpace(name) != name {
			return nil, fmt.Errorf("--skills contains an empty or non-exact name")
		}
		if seen[name] {
			return nil, fmt.Errorf("--skills contains duplicate name %q", name)
		}
		if retired[name] {
			return nil, fmt.Errorf("--skills contains retired name %q", name)
		}
		if !targets[name] {
			return nil, fmt.Errorf("--skills contains unknown name %q", name)
		}
		seen[name] = true
		result = append(result, name)
	}
	return result, nil
}

func repositoryRoot() (string, error) {
	if configured := os.Getenv("HARD_SKILLS_REPO_ROOT"); configured != "" {
		root, err := filepath.Abs(configured)
		if err != nil {
			return "", fmt.Errorf("resolve configured repository root: %w", err)
		}
		return root, nil
	}
	current, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("get working directory: %w", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(current, "go.mod")); err == nil {
			return current, nil
		}
		parent := filepath.Dir(current)
		if parent == current {
			return "", fmt.Errorf("cannot find repository root from working directory")
		}
		current = parent
	}
}
