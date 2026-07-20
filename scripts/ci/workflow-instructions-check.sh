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

require_file AGENTS.md
require_file README.md
require_file docs/spec-first-workflow.md
require_file docs/spec-first-workflow/shared/artifact-model.md
require_file docs/spec-first-workflow/shared/subagents-and-handoff.md
require_file docs/spec-first-workflow/phases/implementation-validation-closeout.md
require_file docs/subagent-contract.md
require_file docs/spec-first-workflow-evals.md
require_file scripts/dev/workflow-behavior-evals.sh

require_heading AGENTS.md '## Routing'
require_heading AGENTS.md '## Validation Matrix'
require_heading docs/spec-first-workflow.md '## Choose A Path'
require_heading docs/spec-first-workflow.md '### Direct'
require_heading docs/spec-first-workflow.md '### Structured'
require_heading docs/spec-first-workflow.md '### Orchestrated'
require_heading docs/spec-first-workflow/phases/implementation-validation-closeout.md '### Local Execution'
require_heading docs/spec-first-workflow/phases/implementation-validation-closeout.md '### Optional Worker Execution'
require_heading docs/spec-first-workflow/phases/implementation-validation-closeout.md '### Immutable Evidence'
require_heading docs/spec-first-workflow/shared/artifact-model.md '## When To Persist'
require_heading docs/spec-first-workflow/shared/subagents-and-handoff.md '## Independent Review'

require_link AGENTS.md 'docs/spec-first-workflow.md'
require_link docs/spec-first-workflow.md 'spec-first-workflow/shared/artifact-model.md'
require_link docs/spec-first-workflow.md 'spec-first-workflow/shared/subagents-and-handoff.md'
require_link docs/spec-first-workflow/phases/implementation-validation-closeout.md 'scripts/dev/codex-worktree-preflight.sh'
require_link docs/subagent-contract.md 'implementation-validation-closeout.md#optional-worker-execution'

# These patterns protect authority and fail-closed boundaries without pinning
# explanatory prose or phase-specific implementation detail.
require_regex AGENTS.md 'external writes.*destructive actions'
require_regex AGENTS.md 'Direct:.*No Goal.*worktree.*independent review.*required'
require_regex AGENTS.md 'Goal.*long-running.*resumable'
require_regex AGENTS.md 'immutable.*tree.*byte-identical fast-forward'
require_regex docs/spec-first-workflow.md 'Self-review.*default'
require_regex docs/spec-first-workflow.md 'grilling.*not a default review stage'
require_regex docs/spec-first-workflow/phases/implementation-validation-closeout.md 'root inspects.*delegated diff.*proof'
require_regex docs/spec-first-workflow/phases/implementation-validation-closeout.md 'local direct change.*not need a commit'
require_regex docs/spec-first-workflow/shared/artifact-model.md 'Persist a result only when'

if rg -n --glob '!scripts/ci/workflow-instructions-check.sh' \
  'autonomous-pre-review-challenge|required for structured/orchestrated work|mandatory.*(App Worker|worktree|independent review)' \
  AGENTS.md README.md docs .agents/skills; then
  fail 'retired universal-workflow controls remain referenced'
fi

bash scripts/dev/workflow-behavior-evals.sh check
printf 'workflow instruction check passed\n'
