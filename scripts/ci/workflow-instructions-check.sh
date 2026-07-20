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

require_link AGENTS.md 'docs/spec-first-workflow.md'
require_link AGENTS.md 'docs/spec-first-workflow/shared/autonomous-pre-review-challenge.md'
require_link docs/spec-first-workflow.md 'spec-first-workflow/shared/autonomous-pre-review-challenge.md'
require_link docs/spec-first-workflow.md 'spec-first-workflow/shared/subagents-and-handoff.md#review-independence'
require_link docs/spec-first-workflow/shared/subagents-and-handoff.md 'autonomous-pre-review-challenge.md'
require_link docs/subagent-contract.md 'spec-first-workflow/shared/autonomous-pre-review-challenge.md'
require_link .agents/skills/grilling/SKILL.md 'docs/spec-first-workflow/shared/autonomous-pre-review-challenge.md'
require_link docs/spec-first-workflow/phases/implementation-validation-closeout.md 'scripts/dev/codex-worktree-preflight.sh'
require_link docs/subagent-contract.md 'implementation-validation-closeout.md#optional-worker-execution'

bash scripts/dev/workflow-behavior-evals.sh check
printf 'workflow instruction check passed\n'
