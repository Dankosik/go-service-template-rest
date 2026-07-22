#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT_DIR"

fail() {
	printf 'implementation-convergence-check: %s\n' "$*" >&2
	exit 1
}

require_text() {
	local file="$1"
	local expected="$2"
	grep -Fq -- "$expected" "$file" || fail "$file is missing required contract: $expected"
}

forbid_text() {
	local forbidden="$1"
	shift
	if grep -Fq -- "$forbidden" "$@"; then
		fail "superseded loop rule is still present: $forbidden"
	fi
}

phase="docs/spec-first-workflow/phases/implementation-validation-closeout.md"
harness="docs/agent-harness.md"
workflow="docs/spec-first-workflow.md"
planning="docs/spec-first-workflow/phases/planning.md"
subagents="docs/spec-first-workflow/shared/subagents-and-handoff.md"

# Implementation optimizes for finite contract closure, not open-ended improvement.
require_text "$phase" "## Acceptance Posture"
require_text "$phase" "Act as a contract closer"
require_text "$phase" "Acceptance is"
require_text "$phase" "terminal: close out when the criteria and mapped proof pass"
require_text "AGENTS.md" "[Acceptance Posture](docs/spec-first-workflow/phases/implementation-validation-closeout.md#acceptance-posture)"

# Scope Lock happens before the finite correction set exists.
require_text "$phase" "#### Scope Lock"
require_text "$phase" "A candidate is scope-valid only when every changed path is authorized"
require_text "$phase" "deterministic placement rule in its bounded"
require_text "$phase" 'git diff --name-only <recorded-base> <candidate-tree>'
require_text "$phase" 'git ls-files --others --exclude-standard'
require_text "$phase" "one disposition: reject it in full"
require_text "$phase" "required boundary expansion reopens the scope owner"

# Activity is progress only when the candidate or evidence state changes.
require_text "$phase" "#### Progress"
require_text "$phase" "Progress is a material candidate transition toward an open finding"
require_text "$phase" "write lane at the preceding frozen candidate"
require_text "$phase" "already have a disposition, even when the"
require_text "$phase" "Worker replacement is reserved for an execution stall"
require_text "$phase" "only returned-candidate convergence is verified"

# The correction set shrinks monotonically and never triggers a fresh review loop.
require_text "$phase" "#### Monotonic Acceptance"
require_text "$phase" "finite finding set containing only candidate-caused regressions"
require_text "$phase" "Correction verification covers the open findings"
require_text "$phase" "Adoption makes the correction the current frozen"
require_text "$phase" "open set a strict subset"
require_text "$phase" "empty set plus passing mapped"
require_text "$phase" "proof accepts the candidate immediately"
require_text "$phase" "#### Diagnostic Gate"
require_text "$phase" "one disposition: reject the entire delta"
require_text "$phase" "rejection removes write authority for that finding"
require_text "$phase" "Read-only diagnosis against"
require_text "$phase" "falsifies the rejected causal hypothesis"

# Model capacity and effort are selected independently; retries do not escalate effort by habit.
require_text "$harness" '| `medium` (default) |'
require_text "$harness" 'when task evidence or a representative evaluation shows'
require_text "$harness" "not by itself a reason to raise effort"
require_text "$workflow" "latest-model?model=gpt-5.6#prompting-best-practices"
require_text "AGENTS.md" "## Task Contract"
require_text "AGENTS.md" "State the goal, relevant context, hard constraints, authorization"
require_text "$workflow" "State each durable instruction once in its"
require_text "$workflow" "Phrase it as an observable trigger"
require_text "$workflow" "broad tone or brevity labels; name the required content"
require_text "$workflow" "and what may be omitted"

# Parallel execution stays bounded to positively independent native Workers.
require_text "$planning" "positively establishes pairwise independence"
require_text "$planning" "Absence of a dependency edge is necessary but not sufficient"
require_text "AGENTS.md" "one root control spans that outcome"
require_text "AGENTS.md" "[Progress](docs/spec-first-workflow/phases/implementation-validation-closeout.md#progress)"
require_text "AGENTS.md" "[Scope Lock](docs/spec-first-workflow/phases/implementation-validation-closeout.md#scope-lock)"
require_text "AGENTS.md" "[Monotonic Acceptance](docs/spec-first-workflow/phases/implementation-validation-closeout.md#monotonic-acceptance)"
require_text "AGENTS.md" "[Diagnostic Gate](docs/spec-first-workflow/phases/implementation-validation-closeout.md#diagnostic-gate)"
require_text "$subagents" "Built-in subagents are read-only research, challenge, or review lanes; they never implement or repair"
require_text "$workflow" "### Phase Lock"
require_text "$workflow" "commits the next transition to the first ready implementation task"
require_text "$workflow" "Status checks and compaction resume from artifacts without changing the"
require_text "$workflow" "proves canonical wording only"
require_text "$workflow" "scope-creep"
require_text "$workflow" "repeated-action interruption"
require_text "$workflow" "regression-correction rollback"
require_text "$workflow" 'planning-`PASS` transition'

checked=(AGENTS.md "$phase" "$harness" "$workflow" "$planning" "$subagents")
for forbidden in \
	"regression joins the current correction set" \
	"replace the Worker or switch to a materially different repair route" \
	"replace the worker only under the phase's no-progress rule" \
	"no fixed retry limit" \
	"non-shrinking evidence frontier" \
	"materially change the causal model or route" \
	"requires a materially different route" \
	"delta-aware re-review" \
	"final integrated-diff review still" \
	"root reviews every result independently" \
	"Continue only when current evidence already identifies a different bounded correction" \
	'hold `high`' \
	'`high` (default)' \
	"lowest reasoning effort likely to succeed" \
	"Always use the lowest effort likely to succeed" \
	"guides/prompt-guidance-gpt-5p6" \
	"## Working Contract" \
	"3. Choose the smallest path that preserves correctness." \
	"4. Public contracts, persisted data, security, money" \
	"5. Evidence before invention." \
	"7. Skills define method."; do
	forbid_text "$forbidden" "${checked[@]}"
done

printf 'implementation-convergence-check: PASS\n'
