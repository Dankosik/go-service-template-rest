package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type sizeMetrics struct {
	Lines  int
	Words  int
	Bytes  int
	Tokens int
}

type sizeRow struct {
	Skill              string
	BaselineSkill      string
	BaselineBody       sizeMetrics
	CandidateBody      sizeMetrics
	BaselineEffective  sizeMetrics
	CandidateEffective sizeMetrics
}

func writeSizeReport(root, baselineRef string, output io.Writer) error {
	if strings.TrimSpace(baselineRef) == "" {
		return fmt.Errorf("--baseline-ref is required")
	}
	baselineCommit, err := gitOutput(root, "rev-parse", "--verify", baselineRef+"^{commit}")
	if err != nil {
		return fmt.Errorf("resolve baseline ref %q: %w", baselineRef, err)
	}
	baselineCommit = strings.TrimSpace(baselineCommit)
	shared, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(sharedContractPath)))
	if err != nil {
		return fmt.Errorf("read shared specialist contract: %w", err)
	}
	sharedMetrics := measure(shared)

	rows := make([]sizeRow, 0, len(targetSkills))
	for _, skill := range targetSkills {
		baselineSkill := skill
		if old, ok := renamedSkills[skill]; ok {
			baselineSkill = old
		}
		baselineData, err := gitBytes(root, "show", baselineCommit+":.agents/skills/"+baselineSkill+"/SKILL.md")
		if err != nil {
			return fmt.Errorf("read baseline skill %s: %w", baselineSkill, err)
		}
		baselineDoc, err := parseSkillDocument(baselineData, baselineSkill)
		if err != nil {
			return fmt.Errorf("parse baseline skill %s: %w", baselineSkill, err)
		}
		candidateDoc, err := readSkillDocument(filepath.Join(root, ".agents", "skills", skill, "SKILL.md"))
		if err != nil {
			return fmt.Errorf("read candidate skill %s: %w", skill, err)
		}
		baselineBody := measure(baselineDoc.Body)
		candidateBody := measure(candidateDoc.Body)
		candidateEffective := candidateBody
		if !executionSkills[skill] {
			candidateEffective = addMetrics(candidateEffective, sharedMetrics)
		}
		rows = append(rows, sizeRow{
			Skill:              skill,
			BaselineSkill:      baselineSkill,
			BaselineBody:       baselineBody,
			CandidateBody:      candidateBody,
			BaselineEffective:  baselineBody,
			CandidateEffective: candidateEffective,
		})
	}

	if _, err := fmt.Fprintf(output, "baseline_ref\t%s\n", baselineRef); err != nil {
		return fmt.Errorf("write baseline ref: %w", err)
	}
	if _, err := fmt.Fprintf(output, "baseline_commit\t%s\n", baselineCommit); err != nil {
		return fmt.Errorf("write baseline commit: %w", err)
	}
	if _, err := fmt.Fprintln(output, strings.Join([]string{
		"skill", "baseline_skill",
		"baseline_body_lines", "baseline_body_words", "baseline_body_bytes", "baseline_body_tokens_ceil_bytes_div_4",
		"candidate_body_lines", "candidate_body_words", "candidate_body_bytes", "candidate_body_tokens_ceil_bytes_div_4",
		"baseline_effective_lines", "baseline_effective_words", "baseline_effective_bytes", "baseline_effective_tokens_ceil_bytes_div_4",
		"candidate_effective_lines", "candidate_effective_words", "candidate_effective_bytes", "candidate_effective_tokens_ceil_bytes_div_4",
	}, "\t")); err != nil {
		return fmt.Errorf("write size report header: %w", err)
	}

	var aggregate sizeRow
	aggregate.Skill = "AGGREGATE"
	aggregate.BaselineSkill = "-"
	for _, row := range rows {
		if err := writeSizeRow(output, row); err != nil {
			return err
		}
		aggregate.BaselineBody = addMetrics(aggregate.BaselineBody, row.BaselineBody)
		aggregate.CandidateBody = addMetrics(aggregate.CandidateBody, row.CandidateBody)
		aggregate.BaselineEffective = addMetrics(aggregate.BaselineEffective, row.BaselineEffective)
		aggregate.CandidateEffective = addMetrics(aggregate.CandidateEffective, row.CandidateEffective)
	}
	return writeSizeRow(output, aggregate)
}

func measure(data []byte) sizeMetrics {
	metrics := sizeMetrics{Bytes: len(data), Words: len(bytes.Fields(data))}
	if len(data) > 0 {
		metrics.Lines = bytes.Count(data, []byte{'\n'})
		if data[len(data)-1] != '\n' {
			metrics.Lines++
		}
	}
	metrics.Tokens = (metrics.Bytes + 3) / 4
	return metrics
}

func addMetrics(left, right sizeMetrics) sizeMetrics {
	result := sizeMetrics{
		Lines: left.Lines + right.Lines,
		Words: left.Words + right.Words,
		Bytes: left.Bytes + right.Bytes,
	}
	result.Tokens = (result.Bytes + 3) / 4
	return result
}

func writeSizeRow(output io.Writer, row sizeRow) error {
	if _, err := fmt.Fprintf(output, "%s\t%s\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\n",
		row.Skill, row.BaselineSkill,
		row.BaselineBody.Lines, row.BaselineBody.Words, row.BaselineBody.Bytes, row.BaselineBody.Tokens,
		row.CandidateBody.Lines, row.CandidateBody.Words, row.CandidateBody.Bytes, row.CandidateBody.Tokens,
		row.BaselineEffective.Lines, row.BaselineEffective.Words, row.BaselineEffective.Bytes, row.BaselineEffective.Tokens,
		row.CandidateEffective.Lines, row.CandidateEffective.Words, row.CandidateEffective.Bytes, row.CandidateEffective.Tokens); err != nil {
		return fmt.Errorf("write size report row %s: %w", row.Skill, err)
	}
	return nil
}

func gitOutput(root string, args ...string) (string, error) {
	data, err := gitBytes(root, args...)
	return string(data), err
}

func gitBytes(root string, args ...string) ([]byte, error) {
	command := exec.CommandContext(context.Background(), "git", append([]string{"-C", root}, args...)...)
	data, err := command.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("git %s: %s", strings.Join(args, " "), strings.TrimSpace(string(data)))
	}
	return data, nil
}
