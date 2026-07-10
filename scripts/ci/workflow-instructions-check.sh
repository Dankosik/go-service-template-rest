#!/usr/bin/env bash
set -euo pipefail

workflow_files=(
  AGENTS.md
  README.md
  docs/spec-first-workflow.md
  docs/spec-first-workflow-evals.md
  docs/spec-first-workflow/shared/artifact-model.md
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
  docs/subagent-brief-template.md
)

skill_files=(
  .agents/skills/*session/SKILL.md
  .agents/skills/planning-and-task-breakdown/SKILL.md
  .agents/skills/spec-document-designer/SKILL.md
  .agents/skills/spec-clarification-challenge/SKILL.md
  .agents/skills/specification-review/SKILL.md
  .agents/skills/go-design-spec/SKILL.md
  .agents/skills/go-coder/SKILL.md
  .agents/skills/workflow-status/SKILL.md
  .agents/skills/workflow-plan-adequacy-challenge/SKILL.md
)

missing=()
for file in "${workflow_files[@]}"; do
  [[ -f "${file}" ]] || missing+=("${file}")
done

if [[ ${#missing[@]} -gt 0 ]]; then
  echo "workflow instruction check failed: missing canonical files"
  printf '  - %s\n' "${missing[@]}"
  exit 1
fi

for file in docs/spec-first-workflow/shared/*.md docs/spec-first-workflow/phases/*.md; do
  for heading in "Read When" "Inputs" "Outputs" "Stop Rule"; do
    if ! grep -Fqx "## ${heading}" "${file}"; then
      echo "workflow instruction check failed: ${file} is missing '## ${heading}'"
      exit 1
    fi
  done
done

check_links() {
  local file="$1"
  local base
  local target
  base="$(dirname "${file}")"

  while IFS= read -r target; do
    target="${target#(}"
    target="${target%)}"
    target="${target%%#*}"
    [[ -z "${target}" || "${target}" == http://* || "${target}" == https://* ]] && continue
    if [[ ! -e "${base}/${target}" ]]; then
      echo "workflow instruction check failed: broken markdown link"
      echo "  file: ${file}"
      echo "  target: ${target}"
      exit 1
    fi
  done < <(grep -Eo '\([^)]*\.md(#[^)]*)?\)' "${file}" || true)
}

for file in "${workflow_files[@]}" "${skill_files[@]}"; do
  check_links "${file}"
done

bash scripts/dev/workflow-behavior-evals.sh check

require_text() {
  local text="$1"
  local file="$2"
  local message="$3"
  if ! grep -Fq -- "${text}" "${file}"; then
    echo "workflow instruction check failed: ${message}"
    echo "  file: ${file}"
    exit 1
  fi
}

require_text \
  'A next-session handoff is permitted only when' \
  docs/spec-first-workflow/shared/subagents-and-handoff.md \
  'the canonical handoff gate is missing'
require_text \
  'including when the user explicitly requests that macro-phase handoff' \
  docs/spec-first-workflow/shared/subagents-and-handoff.md \
  'an explicit handoff request must be limited to the next macro phase'
require_text \
  'requesting a separate session for an internal checkpoint does not make it eligible' \
  docs/spec-first-workflow/shared/subagents-and-handoff.md \
  'an explicit request must not turn an internal checkpoint into a handoff'
require_text \
  'are internal checkpoints of their owning macro phase' \
  docs/spec-first-workflow/shared/subagents-and-handoff.md \
  'internal review must remain inside its owning macro phase'
require_text \
  'An explicitly user-requested standalone review remains read-only' \
  docs/spec-first-workflow/shared/subagents-and-handoff.md \
  'the standalone read-only review exception is missing'
require_text \
  'Those are internal next actions, not handoffs.' \
  .agents/skills/codex-goal-prompt-composer/SKILL.md \
  'the Goal composer must reject internal-checkpoint handoffs'
require_text \
  'label it as same-request work and do not propose a next-session prompt' \
  .agents/skills/workflow-status/SKILL.md \
  'workflow status must keep internal checkpoints in the current request'
require_text \
  '### E17 — Standalone Read-Only Review' \
  docs/spec-first-workflow-evals.md \
  'the standalone-review behavior case is missing'
require_text \
  '### E18 — Internal Review Loop' \
  docs/spec-first-workflow-evals.md \
  'the internal review-loop behavior case is missing'
require_text \
  'the user asks for a next-session prompt so review can be completed separately' \
  docs/spec-first-workflow-evals.md \
  'the internal review-loop case must cover an explicit prompt request'
require_text \
  '### E19 — Honest Blocker Handoff' \
  docs/spec-first-workflow-evals.md \
  'the honest-blocker handoff behavior case is missing'
require_text \
  '### E20 — Non-Trivial Phase Spine' \
  docs/spec-first-workflow-evals.md \
  'the strict phase-spine behavior case is missing'
require_text \
  '### E21 — Helper Skill Gate Bypass' \
  docs/spec-first-workflow-evals.md \
  'helper skills must not bypass required review gates'
require_text \
  '### E22 — External Evidence Before Invention' \
  docs/spec-first-workflow-evals.md \
  'external-platform design must research current evidence before invention'
require_text \
  '### E23 — Skill And Specialist Subagent Routing' \
  docs/spec-first-workflow-evals.md \
  'specialist routing must distinguish local skills from subagent lanes'
require_text \
  'Skills define method; subagents provide separate context and independence.' \
  docs/spec-first-workflow/shared/subagents-and-handoff.md \
  'delegation must distinguish a skill method from a separate specialist context'
require_text \
  'Evidence before invention.' \
  AGENTS.md \
  'the global external-evidence rule is missing'
require_text \
  'Do not substitute model memory for current external evidence.' \
  docs/spec-first-workflow/phases/research.md \
  'research must reject model-memory substitution for current external behavior'
require_text \
  'authoring leaves `status: draft`' \
  .agents/skills/spec-document-designer/SKILL.md \
  'spec authoring must not self-approve readiness'
require_text \
  'proceeds to planning only after independent technical-design review' \
  .agents/skills/go-design-spec/SKILL.md \
  'design authoring must route through independent review'
require_text \
  'runs independent task review/readiness' \
  .agents/skills/planning-and-task-breakdown/SKILL.md \
  'task authoring must route through readiness review'
require_text \
  'obtains independent QA review before planning' \
  .agents/skills/go-qa-tester-spec/SKILL.md \
  'test design must route through independent QA review'

forbidden_internal_target_pattern='next[_ -]?phase[=: ]+(specification[-_ ]?review|technical[-_ ]?design[-_ ]?review|test[-_ ]?design([-_ ]?qa)?[-_ ]?review|task[-_ ]?(review([/_ -]?readiness)?|readiness[-_ ]?review)|readiness[-_ ]?review|post[-_ ]?code[-_ ]?review|validation|closeout)'
for forbidden_target in \
  'next_phase=technical-design-review' \
  'next_phase=test-design-qa-review' \
  'next_phase=task-review/readiness'; do
  if ! printf '%s\n' "${forbidden_target}" | grep -Eiq -- "${forbidden_internal_target_pattern}"; then
    echo "workflow instruction check failed: internal-target guard misses ${forbidden_target}"
    exit 1
  fi
done
for valid_target in \
  'next_phase=technical-design' \
  'next_phase=test-design'; do
  if printf '%s\n' "${valid_target}" | grep -Eiq -- "${forbidden_internal_target_pattern}"; then
    echo "workflow instruction check failed: internal-target guard rejects ${valid_target}"
    exit 1
  fi
done

forbidden_internal_targets="$(grep -RIniE -- "${forbidden_internal_target_pattern}" \
  "${workflow_files[@]}" "${skill_files[@]}" || true)"
if [[ -n "${forbidden_internal_targets}" ]]; then
  echo "workflow instruction check failed: an internal checkpoint is configured as a next phase"
  printf '%s\n' "${forbidden_internal_targets}" | sed 's/^/  /'
  exit 1
fi

forbidden_prompt_pattern='copy-pastable[[:space:]]+fresh-review[[:space:]]+prompt|Subagent authorization:'
if ! printf '%s\n' 'Copy-pastable fresh-review prompt:' | grep -Eiq -- "${forbidden_prompt_pattern}"; then
  echo "workflow instruction check failed: prompt guard does not catch the observed regression"
  exit 1
fi
if ! printf '%s\n' 'Subagent authorization:' | grep -Eiq -- "${forbidden_prompt_pattern}"; then
  echo "workflow instruction check failed: prompt guard does not catch repeated authorization"
  exit 1
fi
forbidden_prompts="$(grep -RIniE -- "${forbidden_prompt_pattern}" \
  "${workflow_files[@]}" "${skill_files[@]}" || true)"
if [[ -n "${forbidden_prompts}" ]]; then
  echo "workflow instruction check failed: an active instruction still renders the observed internal-review handoff"
  printf '%s\n' "${forbidden_prompts}" | sed 's/^/  /'
  exit 1
fi

stale_pattern='SHAPE-[A-Z]|FULL-[A-Z]|DIRECT-[A-Z]|LEAN-[A-Z]|routing_revision|phase_state|procedural_gate_state|review_verdict|record_validity|handoff_readiness|artifact_expectation|ROUTING-PHASE-CONTROL|isolated-cli-worker|compact_sufficient|not_expected|review_ready|draft_review_ready|pending_task_review|Subagent gate|Design fan-out|Pattern Fit Diligence|worker-only|contract-design checkpoint|contract checkpoint|one user-started root session|cross-macro-phase collapse|worker delegation is mandatory|capability_only|workflow-plans/|lightweight local|lean local|full orchestrated'
for stale_example in \
  'Status: review_ready' \
  'Subagent gate: local_only' \
  'Pattern Fit Diligence' \
  'workflow-plans/specification.md' \
  'compact_sufficient' \
  'full orchestrated'; do
  if ! printf '%s\n' "${stale_example}" | grep -Eiq -- "${stale_pattern}"; then
    echo "workflow instruction check failed: stale guard misses ${stale_example}"
    exit 1
  fi
done
stale_matches="$(grep -RInE -- "${stale_pattern}" \
  AGENTS.md README.md SOUL.md \
  docs/spec-first-workflow.md \
  docs/spec-first-workflow-evals.md \
  docs/spec-first-workflow \
  docs/subagent-contract.md \
  docs/subagent-brief-template.md \
  .agents/skills \
  .codex/agents || true)"

if [[ -n "${stale_matches}" ]]; then
  echo "workflow instruction check failed: retired workflow-machine vocabulary remains"
  printf '%s\n' "${stale_matches}" | sed 's/^/  /'
  exit 1
fi

word_count="$(wc -w "${workflow_files[@]}" "${skill_files[@]}" | awk 'END { print $1 }')"

if (( word_count > 16000 )); then
  echo "workflow instruction check failed: prompt budget exceeded"
  echo "  words: ${word_count}"
  echo "  budget: 16000"
  exit 1
fi

echo "workflow instruction check passed: ${#workflow_files[@]} canonical files, ${word_count} words"
