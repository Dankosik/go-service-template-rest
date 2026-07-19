package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func emitSelectedEvals(root, outputDir string) error {
	if strings.TrimSpace(outputDir) == "" {
		return fmt.Errorf("--output-dir is required")
	}
	if err := os.MkdirAll(outputDir, 0o750); err != nil {
		return fmt.Errorf("create output directory: %w", err)
	}
	outputRoot, err := os.OpenRoot(outputDir)
	if err != nil {
		return fmt.Errorf("open output directory: %w", err)
	}
	defer func() { _ = outputRoot.Close() }()

	manifest, err := outputRoot.OpenFile("manifest.tsv", os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return fmt.Errorf("create selected eval manifest: %w", err)
	}
	writer := bufio.NewWriter(manifest)
	closeManifest := func() error {
		if err := writer.Flush(); err != nil {
			_ = manifest.Close()
			return fmt.Errorf("flush selected eval manifest: %w", err)
		}
		return manifest.Close()
	}

	for _, skill := range selectedSkills {
		bundle, err := readEvalBundle(root, skill)
		if err != nil {
			_ = manifest.Close()
			return fmt.Errorf("%s: %w", skill, err)
		}
		if problems := validateEvalBundle(skill, bundle); len(problems) > 0 {
			_ = manifest.Close()
			return fmt.Errorf("%s", strings.Join(problems, "\n"))
		}
		byCategory := coreEvalsByCategory(bundle.Evals)
		for _, category := range evalCategories {
			eval := byCategory[category]
			caseDirRel := filepath.Join("cases", skill, category)
			if err := outputRoot.MkdirAll(caseDirRel, 0o750); err != nil {
				_ = manifest.Close()
				return fmt.Errorf("create selected eval case directory: %w", err)
			}
			promptRel := filepath.Join(caseDirRel, "prompt.txt")
			expectedRel := filepath.Join(caseDirRel, "expected.txt")
			if err := outputRoot.WriteFile(promptRel, []byte(eval.Prompt+"\n"), 0o600); err != nil {
				_ = manifest.Close()
				return fmt.Errorf("write selected eval prompt: %w", err)
			}
			var expected strings.Builder
			expected.WriteString(eval.ExpectedOutput)
			expected.WriteByte('\n')
			for _, assertion := range eval.Assertions {
				expected.WriteString(assertion)
				expected.WriteByte('\n')
			}
			if err := outputRoot.WriteFile(expectedRel, []byte(expected.String()), 0o600); err != nil {
				_ = manifest.Close()
				return fmt.Errorf("write selected eval expectation: %w", err)
			}
			caseID := skill + ":" + category
			if _, err := fmt.Fprintf(writer, "%s\t%s\t%s\t%s\t%s\n",
				caseID, skill, category, filepath.ToSlash(promptRel), filepath.ToSlash(expectedRel)); err != nil {
				_ = manifest.Close()
				return fmt.Errorf("write selected eval manifest: %w", err)
			}
		}
	}
	if err := closeManifest(); err != nil {
		return fmt.Errorf("close selected eval manifest: %w", err)
	}
	return nil
}

func coreEvalsByCategory(evals []evalCase) map[string]evalCase {
	byCategory := make(map[string]evalCase, len(evals))
	for _, eval := range evals {
		if strings.TrimSpace(eval.TrialClass) == "" {
			byCategory[eval.Category] = eval
		}
	}
	return byCategory
}
