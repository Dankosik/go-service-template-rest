#!/usr/bin/env bash
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "${root}"

fail() {
  printf 'workflow instruction check failed: %s\n' "$1" >&2
  exit 1
}

require_file() {
  [[ -f "$1" ]] || fail "missing required owner: $1"
}

require_heading() {
  local file="$1" heading="$2"
  grep -Fqx -- "${heading}" "${file}" || fail "${file} is missing ${heading}"
}

require_regex() {
  local file="$1" pattern="$2"
  grep -Eq -- "${pattern}" "${file}" || fail "${file} is missing a required safety or authority invariant"
}

require_link() {
  local file="$1" target="$2"
  grep -Fq -- "${target}" "${file}" || fail "${file} is missing its canonical link to ${target}"
}

workflow_files=(
  AGENTS.md
  README.md
  docs/spec-first-workflow.md
  docs/spec-first-workflow-evals.md
  docs/spec-first-workflow/shared/artifact-model.md
  docs/spec-first-workflow/shared/autonomous-pre-review-challenge.md
  docs/spec-first-workflow/shared/subagents-and-handoff.md
  docs/spec-first-workflow/phases/intake.md
  docs/spec-first-workflow/phases/research.md
  docs/spec-first-workflow/phases/specification.md
  docs/spec-first-workflow/phases/specification-review.md
  docs/spec-first-workflow/phases/system-integration-design.md
  docs/spec-first-workflow/phases/go-code-ownership-design.md
  docs/spec-first-workflow/phases/technical-design-review.md
  docs/spec-first-workflow/phases/test-design.md
  docs/spec-first-workflow/phases/planning.md
  docs/spec-first-workflow/phases/task-review-readiness.md
  docs/spec-first-workflow/phases/implementation-validation-closeout.md
  docs/subagent-contract.md
)

skill_files=(
  .agents/skills/grilling/SKILL.md
  .agents/skills/grilling/evals/evals.json
  .agents/skills/planning-and-task-breakdown/SKILL.md
  .agents/skills/planning-session/SKILL.md
  .agents/skills/spec-document-designer/SKILL.md
  .agents/skills/specification-review/SKILL.md
  .agents/skills/specification-session/SKILL.md
  .agents/skills/technical-design-session/SKILL.md
  .agents/skills/test-design-session/SKILL.md
)

for file in "${workflow_files[@]}" "${skill_files[@]}"; do
  require_file "${file}"
done

for file in docs/spec-first-workflow/shared/*.md docs/spec-first-workflow/phases/*.md; do
  for heading in "Read When" "Inputs" "Outputs" "Stop Rule"; do
    require_heading "${file}" "## ${heading}"
  done
done

require_heading AGENTS.md '## Routing'
require_heading AGENTS.md '### Required Spine'
require_heading AGENTS.md '## Validation Matrix'
require_heading docs/spec-first-workflow.md '### Required Spine'
require_heading docs/spec-first-workflow.md '## Phase Router'
require_heading docs/spec-first-workflow.md '### Review Routing'
require_heading docs/spec-first-workflow.md '## Phase Movement'
require_heading docs/spec-first-workflow/shared/subagents-and-handoff.md '## Autonomous Pre-Review Challenge'
require_heading docs/spec-first-workflow/shared/subagents-and-handoff.md '## Review Independence'
require_heading docs/spec-first-workflow/shared/subagents-and-handoff.md '## Handoff'
require_heading docs/spec-first-workflow/phases/implementation-validation-closeout.md '### Local Execution'
require_heading docs/spec-first-workflow/phases/implementation-validation-closeout.md '### Optional Worker Execution'
require_heading docs/spec-first-workflow/phases/implementation-validation-closeout.md '### Immutable Evidence'

for heading in   '# Autonomous Pre-Review Challenge'   '## Protocol'   '## Authority'   '## State And Continuation'   '## Exhaustion And Invalidation'   '## Reviewer Separation'; do
  require_heading docs/spec-first-workflow/shared/autonomous-pre-review-challenge.md "${heading}"
done

for event in QUESTION HUMAN_REQUIRED REOPEN DONE; do
  count="$(grep -Fxc -- "${event}" docs/spec-first-workflow/shared/autonomous-pre-review-challenge.md || true)"
  [[ "${count}" -eq 1 ]] || fail "challenge event ${event} must occur exactly once; found ${count}"
done
for token in ACCEPT OVERRIDE RECLASSIFY CONTINUE_INDEPENDENT WAIT_HUMAN REOPEN_OWNER; do
  require_regex docs/spec-first-workflow/shared/autonomous-pre-review-challenge.md "\`${token}\`"
done

challenge_owners="$(find docs/spec-first-workflow/shared -maxdepth 1 -type f -iname '*autonomous*challenge*.md' -print | wc -l | tr -d '[:space:]')"
[[ "${challenge_owners}" -eq 1 ]] || fail "expected one autonomous challenge owner; found ${challenge_owners}"

require_link AGENTS.md 'docs/spec-first-workflow.md'
require_link AGENTS.md 'docs/spec-first-workflow/shared/autonomous-pre-review-challenge.md'
require_link docs/spec-first-workflow.md 'spec-first-workflow/shared/autonomous-pre-review-challenge.md'
require_link docs/spec-first-workflow.md 'spec-first-workflow/shared/subagents-and-handoff.md#review-independence'
require_link docs/spec-first-workflow/shared/subagents-and-handoff.md 'autonomous-pre-review-challenge.md'
require_link docs/subagent-contract.md 'spec-first-workflow/shared/autonomous-pre-review-challenge.md'
require_link .agents/skills/grilling/SKILL.md 'docs/spec-first-workflow/shared/autonomous-pre-review-challenge.md'
require_link docs/spec-first-workflow/phases/implementation-validation-closeout.md 'scripts/dev/codex-worktree-preflight.sh'
require_link docs/subagent-contract.md 'implementation-validation-closeout.md#optional-worker-execution'

require_regex AGENTS.md 'Structured:.*reviewed .*spec.md.*reviewed .*tasks.md'
require_regex AGENTS.md 'complete specification and independent specification review'
require_regex AGENTS.md 'autonomous read-only challenge probe'
require_regex AGENTS.md 'different independent reviewer'
require_regex docs/spec-first-workflow.md 'Specification, combined Technical Design, Test Design, Planning, and an explicit .*research only'
require_regex docs/spec-first-workflow.md 'required non-implementation review.*PASS.*convergence'
require_regex docs/spec-first-workflow/phases/research.md 'research only.*independent read-only review'
require_regex docs/spec-first-workflow/phases/specification.md 'structured or orchestrated work, run the independent'
require_regex docs/spec-first-workflow/phases/system-integration-design.md 'structured or orchestrated work, run .*Technical Design Review'
require_regex docs/spec-first-workflow/phases/go-code-ownership-design.md 'structured or orchestrated work, use .*Technical Design Review'
require_regex docs/spec-first-workflow/phases/test-design.md 'structured or orchestrated work triggers test design, run an independent QA review'
require_regex docs/spec-first-workflow/phases/planning.md 'structured or orchestrated work, run independent .*Task Review'
require_regex docs/spec-first-workflow/shared/artifact-model.md 'When review is required, only .*PASS.*ready'
require_regex docs/spec-first-workflow/shared/subagents-and-handoff.md 'Structured and orchestrated work requires an independent reviewer'
require_regex docs/spec-first-workflow/shared/subagents-and-handoff.md 'required challenge .*DONE.*independent findings and verdict'
require_regex docs/spec-first-workflow/shared/subagents-and-handoff.md 'latest required review returns .*PASS'
require_regex docs/spec-first-workflow/shared/subagents-and-handoff.md 'final chat response MUST end with a copy-pastable next-session prompt'
require_regex .agents/skills/planning-and-task-breakdown/SKILL.md 'ends before the root-owned Readiness Review'
require_regex .agents/skills/planning-session/SKILL.md 'Readiness Review becomes available'
require_regex .agents/skills/specification-session/SKILL.md 'Required Review becomes available'
require_regex .agents/skills/technical-design-session/SKILL.md 'Required Review begins only'
require_regex .agents/skills/test-design-session/SKILL.md 'Required Review begins only'
require_regex .agents/skills/grilling/SKILL.md '^## Internal challenger mode$'
require_regex .agents/skills/grilling/evals/evals.json 'Internal macro-phase grilling mode'
require_regex docs/spec-first-workflow-evals.md '^### E42 .*Autonomous Challenge Authority And Continuation'
require_regex docs/spec-first-workflow-evals.md '^### E43 .*Autonomous Challenge Exhaustion Freshness And Review Separation'
require_regex docs/spec-first-workflow-evals.md 'different read-only child'
require_regex docs/spec-first-workflow-evals.md 'no transcript, receipt, queue, probe status, or lifecycle artifact'

if rg -n   'Self-review is the default|independent review follows only when|risk-triggered review'   docs/spec-first-workflow.md   docs/spec-first-workflow/phases/research.md   docs/spec-first-workflow/phases/specification.md   docs/spec-first-workflow/phases/system-integration-design.md   docs/spec-first-workflow/phases/go-code-ownership-design.md   docs/spec-first-workflow/phases/test-design.md   docs/spec-first-workflow/phases/planning.md   .agents/skills/planning-and-task-breakdown/SKILL.md   .agents/skills/planning-session/SKILL.md   .agents/skills/specification-session/SKILL.md   .agents/skills/technical-design-session/SKILL.md   .agents/skills/test-design-session/SKILL.md; then
  fail 'weakened risk-triggered non-implementation review wording remains'
fi

for file in AGENTS.md README.md docs/spec-first-workflow.md docs/spec-first-workflow/shared/subagents-and-handoff.md docs/subagent-contract.md .agents/skills/grilling/SKILL.md .codex/agents/challenger-agent.toml; do
  if grep -Eq -- '^(QUESTION|HUMAN_REQUIRED|REOPEN|DONE)$' "${file}"; then
    fail "challenge protocol copied outside its canonical owner: ${file}"
  fi
done

probe_lifecycle_artifacts="$(
  find docs .agents .codex specs -type f -print 2>/dev/null     | grep -Ei '(^|/)[^/]*(grill|probe|challenge)[^/]*(receipt|transcript|queue|status)[^/]*$'     || true
)"
[[ -z "${probe_lifecycle_artifacts}" ]] || fail "an autonomous challenge lifecycle artifact exists: ${probe_lifecycle_artifacts}"

# Preserve the current accelerated implementation/validation contract.
require_regex AGENTS.md 'Direct:.*No Goal.*worktree.*independent review.*required'
require_regex AGENTS.md 'Goal.*long-running.*resumable'
require_regex AGENTS.md 'immutable.*tree.*byte-identical fast-forward'
require_regex docs/spec-first-workflow/phases/implementation-validation-closeout.md 'root inspects every delegated diff and proof'
require_regex docs/spec-first-workflow/phases/implementation-validation-closeout.md 'local direct change does not need a commit'
require_regex docs/spec-first-workflow/phases/implementation-validation-closeout.md 'best-suited available model and reasoning effort'
require_regex docs/spec-first-workflow/phases/implementation-validation-closeout.md 'never inherit an App default'
require_regex docs/spec-first-workflow/phases/implementation-validation-closeout.md "user's standing request"
require_regex docs/spec-first-workflow/phases/implementation-validation-closeout.md 'same Worker and managed worktree'
require_regex docs/spec-first-workflow/phases/implementation-validation-closeout.md 'evidence-backed no progress'
require_regex docs/spec-first-workflow/phases/implementation-validation-closeout.md 'frozen combined candidate'
require_regex docs/spec-first-workflow/phases/implementation-validation-closeout.md 'local repository default/main is the authoritative integration branch'
require_regex docs/spec-first-workflow/phases/implementation-validation-closeout.md 'Remote push is outside this integration rule'
require_regex docs/spec-first-workflow-evals.md 'root edits and self-reviews the assigned checkout'
require_regex docs/spec-first-workflow-evals.md 'Reuse exact successful proof for an immutable tree'
require_regex docs/spec-first-workflow-evals.md 'root-local implementation'
require_regex Makefile '^GO_FILES := .*git ls-files --cached --others --exclude-standard'
require_regex Makefile '^instruction-evals-harness:'

bash scripts/dev/workflow-behavior-evals.sh check
printf 'workflow instruction check passed\n'
