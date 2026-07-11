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
  'A Codex Goal is an execution control for the implementation/validation/closeout macro phase only.' \
  AGENTS.md \
  'Codex Goal lifecycle must be limited to implementation'
require_text \
  'Do not create or continue one during intake, research, specification, technical design, test design, planning' \
  AGENTS.md \
  'pre-implementation macro phases must not create or continue a Codex Goal'
require_text \
  'On entering implementation/validation/closeout, create or continue exactly one root-thread Codex Goal' \
  AGENTS.md \
  'implementation entry must establish exactly one root Codex Goal'
require_text \
  'Objective: <one next outcome>' \
  docs/spec-first-workflow/shared/subagents-and-handoff.md \
  'non-implementation handoffs must use Objective instead of Goal'
require_text \
  '`Goal:` is reserved for that implementation handoff' \
  docs/spec-first-workflow/shared/subagents-and-handoff.md \
  'Goal terminology must be reserved for the implementation handoff'
require_text \
  'Use only when entering the implementation/validation/closeout macro phase' \
  .agents/skills/codex-goal-prompt-composer/SKILL.md \
  'the Goal composer must be gated to implementation entry'
require_text \
  'Those are internal next actions, not handoffs.' \
  .agents/skills/codex-goal-prompt-composer/SKILL.md \
  'the Goal composer must reject internal-checkpoint handoffs'
require_text \
  '**Objective:** the requested action and outcome' \
  .agents/skills/agent-prompt-composer/SKILL.md \
  'generic handoffs must use Objective instead of Goal'
require_text \
  'Do not create or continue a Codex Goal while authoring or reviewing pre-implementation artifacts' \
  docs/spec-first-workflow-evals.md \
  'the structured-feature eval must enforce implementation-only Goal timing'
require_text \
  'Do not create or continue a Codex Goal for this planning-only request.' \
  docs/spec-first-workflow-evals.md \
  'the planning-boundary eval must reject a Codex Goal'

if grep -Fq -- 'Goal: <one next outcome>' docs/spec-first-workflow/shared/subagents-and-handoff.md; then
  echo 'workflow instruction check failed: non-implementation handoff still uses Goal'
  exit 1
fi

if grep -Fq -- '**Goal:** the requested action and outcome' .agents/skills/agent-prompt-composer/SKILL.md; then
  echo 'workflow instruction check failed: generic agent handoff still uses Goal'
  exit 1
fi

if grep -Eq -- '^Goal$' .agents/skills/agent-prompt-composer/references/example-transformations.md; then
  echo 'workflow instruction check failed: generic agent handoff example still uses Goal'
  exit 1
fi
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
  'independently review the fixed synthesis to a fresh `PASS`' \
  docs/spec-first-workflow-evals.md \
  'the structured research-only boundary must receive independent PASS review'
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
  'reviews each return fresh `PASS` before implementation' \
  docs/spec-first-workflow-evals.md \
  'helper skills must require PASS rather than concerns or no-blocker review'
require_text \
  '### E22 — External Evidence Before Invention' \
  docs/spec-first-workflow-evals.md \
  'external-platform design must research current evidence before invention'
require_text \
  '### E23 — Skill And Specialist Subagent Routing' \
  docs/spec-first-workflow-evals.md \
  'specialist routing must distinguish local skills from subagent lanes'
require_text \
  'every mandatory task and proof on every dependency path through the accepted completion' \
  docs/spec-first-workflow.md \
  'the canonical implementation-input closure rule must cover the full completion path'
require_text \
  'A ready `tasks.md` additionally satisfies [implementation-input closure]' \
  docs/spec-first-workflow/shared/artifact-model.md \
  'ledger readiness must require full-path input closure without overloading every upstream artifact'
require_text \
  'exact bytes covered by canonicalization, digest, or signature' \
  docs/spec-first-workflow/phases/system-integration-design.md \
  'byte-sensitive contracts must close schemas and bytes before implementation'
require_text \
  'inventory every input-bearing design surface on the current implementation completion path' \
  docs/spec-first-workflow/phases/technical-design-review.md \
  'technical-design review must attempt full-path input-bearing artifact construction'
require_text \
  'a phase-level `PASS` requires rechecking implementation-input closure across every materially distinct input-bearing surface on the current completion path' \
  docs/spec-first-workflow/phases/technical-design-review.md \
  'focused re-review must not bypass phase-level input closure'
require_text \
  'valid fail-before signal only when their expected contents are mechanically derivable' \
  docs/spec-first-workflow/phases/test-design.md \
  'test design must distinguish missing code from missing design inputs'
require_text \
  'only that owner may narrow or split the outcome' \
  docs/spec-first-workflow/phases/test-design.md \
  'test design must fail and reopen the accepted-outcome owner instead of changing scope itself'
require_text \
  'may remain only for a task and claim already outside the accepted current implementation completion' \
  docs/spec-first-workflow/phases/test-design.md \
  'unavailable test proof must be excluded from the accepted current completion'
require_text \
  'cold-walk every mandatory dependency path from the first task through final validation' \
  docs/spec-first-workflow/phases/planning.md \
  'planning must dry-run the full completion path'
require_text \
  'Never approve `PASS subject to gates` for a mandatory path.' \
  docs/spec-first-workflow/phases/planning.md \
  'planning must reject conditional readiness for mandatory work'
require_text \
  'Cold completion: can a fresh agent execute every mandatory task and required proof through final validation' \
  docs/spec-first-workflow/phases/task-review-readiness.md \
  'task readiness must test end-to-end cold completion'
require_text \
  'A ledger cannot receive `PASS subject to gates` when a gate can block mandatory completion.' \
  docs/spec-first-workflow/phases/task-review-readiness.md \
  'readiness must reject unavailable inputs on any mandatory completion path'
require_text \
  'every mandatory task and proof through current completion is executable from closed inputs' \
  docs/spec-first-workflow/phases/task-review-readiness.md \
  'readiness PASS must mean end-to-end execution readiness'
require_text \
  'return `FAIL` and reopen the accepted-outcome owner; the reviewer does not narrow scope.' \
  docs/spec-first-workflow/phases/task-review-readiness.md \
  'readiness review must not change the accepted outcome itself'
require_text \
  "router's [implementation-input closure]" \
  .agents/skills/technical-design-session/SKILL.md \
  'the technical-design session must route through canonical input closure'
require_text \
  '### E24 — Pre-Implementation Input Closure' \
  docs/spec-first-workflow-evals.md \
  'the pre-implementation input-closure behavior case is missing'
require_text \
  'registry-record metamodel but no concrete records' \
  docs/spec-first-workflow-evals.md \
  'the input-closure eval must cover metamodel-only concrete records'
require_text \
  'mechanically derivable without semantic choice' \
  docs/spec-first-workflow-evals.md \
  'the input-closure eval must reject semantic invention'
require_text \
  'Later mandatory T109 requires an unavailable externally owned `G-SCALE` targets/budgets packet, and final T110 depends on T109.' \
  docs/spec-first-workflow-evals.md \
  'the input-closure eval must cover a later unavailable gate that blocks final completion'
require_text \
  '`PASS subject to external gates` merely because earlier tasks can run' \
  docs/spec-first-workflow-evals.md \
  'the input-closure eval must reject conditional readiness for later mandatory gates'
require_text \
  '### E25 — Dependency Approval Evidence' \
  docs/spec-first-workflow-evals.md \
  'the dependency approval behavior case is missing'
require_text \
  'maintenance/releases, license, security or vulnerability posture, API stability, transitive cost, domain adoption, and repository/boundary fit' \
  docs/spec-first-workflow/phases/research.md \
  'dependency approval must retain current maintenance, license, security, stability, cost, adoption, and fit evidence'
require_text \
  '### E26 — Regression Fail-Before Proof' \
  docs/spec-first-workflow-evals.md \
  'the regression fail-before behavior case is missing'
require_text \
  'WORKFLOW_EVAL_BASE_REF' \
  scripts/dev/workflow-behavior-evals.sh \
  'behavior evals must support an explicit historical baseline'
require_text \
  'use pre-`c99e838` commit `1ddd7cc`' \
  docs/spec-first-workflow-evals.md \
  'the GPT-5.6 migration eval must name its pre-simplification baseline'
require_text \
  'For a regression, run or add the smallest proof that fails on the old behavior' \
  .agents/skills/go-coder/SKILL.md \
  'go-coder must preserve fail-before proof for regressions'
require_text \
  'update each task or checkpoint checkbox and evidence immediately after its proof' \
  docs/spec-first-workflow/phases/implementation-validation-closeout.md \
  'ledger progress must be durable after each proven checkpoint'
require_text \
  'Inspect current workspace and Git status' \
  docs/spec-first-workflow/shared/artifact-model.md \
  'resume must inspect current workspace state before trusting ledger progress'
require_text \
  'rerun the smallest ledger proof that can detect workspace drift affecting the next unchecked task' \
  docs/spec-first-workflow/shared/artifact-model.md \
  'resume must refresh the narrow proof that detects drift for the next task'
require_text \
  'repair every in-scope implementation-owned finding, revalidate, and re-review the revised diff to `PASS`; only then' \
  docs/spec-first-workflow-evals.md \
  'the honest-blocker eval must repair local findings before external handoff'
require_text \
  'was available at readiness becomes unavailable only after an external provider-state change' \
  docs/spec-first-workflow-evals.md \
  'the honest-blocker eval must not normalize a known pre-implementation gate'
require_text \
  'Implementation-owned findings cannot be relabeled as `blocked` or handed to the user.' \
  docs/spec-first-workflow-evals.md \
  'the phase-spine eval must reject implementation-owned blocker handoffs'
require_text \
  'a matching selector name alone is not evidence' \
  docs/spec-first-workflow/phases/implementation-validation-closeout.md \
  'closeout must verify proof fidelity instead of trusting selector names'
require_text \
  'An implementation-owned failure never produces a next-session prompt.' \
  docs/spec-first-workflow/phases/implementation-validation-closeout.md \
  'implementation-owned failures must stay in same-session repair'
require_text \
  'Skills define method; subagents provide separate context and independence.' \
  docs/spec-first-workflow/shared/subagents-and-handoff.md \
  'delegation must distinguish a skill method from a separate specialist context'
require_text \
  'There is no fixed review-pass or reviewer count.' \
  docs/spec-first-workflow/shared/subagents-and-handoff.md \
  'review convergence must not use an arbitrary pass or reviewer cap'
require_text \
  'Repeat without an arbitrary pass-count limit' \
  docs/spec-first-workflow/shared/subagents-and-handoff.md \
  'review convergence must continue while evidence can change readiness'
require_text \
  'This limits concurrency only, not total lanes or sequential review waves.' \
  docs/spec-first-workflow/shared/subagents-and-handoff.md \
  'the three-lane default must not cap total justified review work'
require_text \
  'whole-artifact or whole-diff coherence pass' \
  docs/spec-first-workflow/shared/subagents-and-handoff.md \
  'specialist review coverage must retain a whole-revision coherence pass'
require_text \
  'A macro phase reaches review convergence only when' \
  docs/spec-first-workflow/shared/subagents-and-handoff.md \
  'the canonical review-convergence exit condition is missing'
require_text \
  '`CONCERNS` is non-terminal' \
  docs/spec-first-workflow/shared/subagents-and-handoff.md \
  'CONCERNS must not permit phase movement or closeout'
require_text \
  '`PASS` means every concern has a disposition, not that no residual risk exists' \
  docs/spec-first-workflow/shared/subagents-and-handoff.md \
  'PASS must represent complete concern disposition rather than zero residual risk'
require_text \
  'only `PASS` can move an artifact to `ready` or permit `done`' \
  docs/spec-first-workflow/shared/artifact-model.md \
  'artifact status movement must require PASS'
require_text \
  'shared `PASS`-only convergence rule' \
  docs/spec-first-workflow.md \
  'the router must expose PASS-only phase movement'
require_text \
  'Only a fresh `PASS` permits the next macro phase or closeout' \
  README.md \
  'the user-facing workflow summary must expose PASS-only movement'

for concerns_phase in \
  docs/spec-first-workflow/phases/research.md \
  docs/spec-first-workflow/phases/specification-review.md \
  docs/spec-first-workflow/phases/technical-design-review.md \
  docs/spec-first-workflow/phases/test-design.md \
  docs/spec-first-workflow/phases/task-review-readiness.md \
  docs/spec-first-workflow/phases/implementation-validation-closeout.md; do
  require_text \
    'disposition and fresh review' \
    "${concerns_phase}" \
    'every phase-local CONCERNS verdict must remain non-terminal'
done

for pass_only_phase in \
  docs/spec-first-workflow/phases/research.md \
  docs/spec-first-workflow/phases/specification.md \
  docs/spec-first-workflow/phases/system-integration-design.md \
  docs/spec-first-workflow/phases/go-code-ownership-design.md \
  docs/spec-first-workflow/phases/test-design.md \
  docs/spec-first-workflow/phases/planning.md \
  docs/spec-first-workflow/phases/implementation-validation-closeout.md; do
  require_text \
    'returned `PASS`' \
    "${pass_only_phase}" \
    'every macro-phase movement or closeout rule must require PASS'
done

require_text \
  'planning_repair when readiness = CONCERNS' \
  .agents/skills/go-domain-invariant-spec/references/state-machine-and-transition-rules.md \
  'the workflow lifecycle example must keep CONCERNS inside planning repair'
require_text \
  'implementation when readiness = fresh PASS for the current `tasks.md` revision' \
  .agents/skills/go-domain-invariant-spec/references/state-machine-and-transition-rules.md \
  'the workflow lifecycle example must reject stale readiness PASS'
require_text \
  'non-trivial `spec.md` has a fresh specification-review PASS' \
  .agents/skills/go-domain-invariant-spec/references/state-machine-and-transition-rules.md \
  'the workflow lifecycle example must require specification PASS'
require_text \
  'design context has a fresh technical-design-review PASS' \
  .agents/skills/go-domain-invariant-spec/references/state-machine-and-transition-rules.md \
  'the workflow lifecycle example must require design PASS when design is triggered'
require_text \
  'fixed, reviewable spec revision' \
  .agents/skills/specification-review/SKILL.md \
  'specification review must accept a draft candidate before PASS'

if grep -Eq -- 'implementation_with_accepted_risks|readiness = WAIVED' \
  .agents/skills/go-domain-invariant-spec/references/state-machine-and-transition-rules.md; then
  echo 'workflow instruction check failed: lifecycle example still bypasses PASS-only readiness'
  exit 1
fi

if grep -Fq -- 'Use only for a ready spec revision' .agents/skills/specification-review/SKILL.md; then
  echo 'workflow instruction check failed: specification review still requires pre-review ready status'
  exit 1
fi

require_text \
  'Any mutation after review' \
  docs/spec-first-workflow/shared/subagents-and-handoff.md \
  'post-review mutations must invalidate affected convergence evidence'
require_text \
  'After relevant validation, structured or orchestrated work requires independent read-only review of the exact candidate final diff and proof evidence before closeout.' \
  docs/spec-first-workflow/phases/implementation-validation-closeout.md \
  'structured and orchestrated implementation must receive independent final-diff review'
require_text \
  'Every review return also names the exact revision or diff' \
  docs/subagent-contract.md \
  'the shared reviewer envelope must preserve revision and affected-lens coverage'
require_text \
  'review of that lens finds a second material ripple defect' \
  docs/spec-first-workflow-evals.md \
  'the internal review-loop eval must exercise more than one repair cycle'
require_text \
  'an artificial pass-count limit' \
  docs/spec-first-workflow-evals.md \
  'the internal review-loop eval must reject fixed review iteration caps'
require_text \
  'five independent, decision-changing specialist review questions with distinct evidence boundaries' \
  docs/spec-first-workflow-evals.md \
  'the delegation eval must require more useful lanes than the concurrency limit'
require_text \
  'multiple sequential waves of at most three concurrent subagents' \
  docs/spec-first-workflow-evals.md \
  'the delegation eval must distinguish concurrent capacity from total useful lanes'
require_text \
  'a clean partial lane cannot close the gate' \
  docs/spec-first-workflow-evals.md \
  'a clean focused review must not close uncovered affected lenses'
require_text \
  '`CONCERNS` cannot permit phase movement even when it names only a bounded risk' \
  docs/spec-first-workflow-evals.md \
  'the internal review-loop eval must keep bounded concerns non-terminal'
require_text \
  'accepted risk recorded without fresh `PASS`' \
  docs/spec-first-workflow-evals.md \
  'accepted-risk disposition must still receive fresh PASS'
require_text \
  'produces a final diff whose first reviewer reports clean within API/data while explicitly leaving a triggered security lens uncovered' \
  docs/spec-first-workflow-evals.md \
  'specialist routing must exercise a partial clean final-diff review'
require_text \
  'covers the missing security lens in a later specialist wave' \
  docs/spec-first-workflow-evals.md \
  'specialist routing must cover the missing final-diff lens before convergence'
require_text \
  'Any post-review mutation requires revalidation and fresh affected-lens review.' \
  docs/spec-first-workflow-evals.md \
  'post-review implementation mutations must invalidate stale review evidence'
require_text \
  'it never permits phase movement or closeout' \
  docs/spec-first-workflow/shared/subagents-and-handoff.md \
  'CONCERNS must remain non-terminal even without a current-phase defect'
require_text \
  'When decision-changing research depends on conflicting sources' \
  docs/spec-first-workflow/phases/research.md \
  'decision-changing conflicting research must receive an independent semantic challenge'
require_text \
  'Ordinary supporting research that does not trigger the semantic-challenge rule above' \
  docs/spec-first-workflow/phases/research.md \
  'ordinary supporting research may avoid a duplicate gate without bypassing triggered challenge'
require_text \
  'When `research only` is the accepted macro-phase boundary' \
  docs/spec-first-workflow/phases/research.md \
  'standalone structured or orchestrated research must receive independent synthesis review'
require_text \
  'may not carry a missing answer owned by research' \
  docs/spec-first-workflow/phases/research.md \
  'research CONCERNS must not carry a research-owned evidence gap'
require_text \
  'independently challenge the decision-changing synthesis before design consumes it' \
  docs/spec-first-workflow-evals.md \
  'the external-evidence eval must exercise independent synthesis challenge'
require_text \
  'When the research phase requires an independent semantic challenge' \
  .agents/skills/research-session/SKILL.md \
  'the research session must route triggered synthesis through independent challenge'
require_text \
  'Required research-synthesis challenge' \
  docs/spec-first-workflow/shared/subagents-and-handoff.md \
  'research synthesis challenge must remain an internal macro-phase checkpoint'
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
require_text \
  'repair, revalidation, and fresh convergence review before closeout' \
  .agents/skills/go-coder/SKILL.md \
  'implementation helpers must route structured and orchestrated diffs through independent convergence review'
require_text \
  '`research only` is the structured/orchestrated macro-phase boundary' \
  .agents/skills/research-session/SKILL.md \
  'standalone structured or orchestrated research must route through independent convergence review'
require_text \
  'The reviewer returns one verdict under the shared convergence contract:' \
  docs/spec-first-workflow/phases/test-design.md \
  'test design review must use explicit convergence verdict semantics'
require_text \
  'including when the check is honestly unavailable outside current completion' \
  docs/spec-first-workflow/phases/test-design.md \
  'unavailable test proof must not receive PASS'
require_text \
  'authorized recorded residual-risk acceptance with evidence and reopen condition' \
  docs/spec-first-workflow/phases/test-design.md \
  'test design must allow PASS only after explicit residual-risk disposition'
require_text \
  'fixed test-plan review under the test-design phase verdict' \
  .codex/agents/qa-agent.toml \
  'the QA reviewer must route fixed test plans through phase verdict semantics'
require_text \
  'the phase verdict and shared convergence condition' \
  .agents/skills/go-qa-tester-spec/SKILL.md \
  'the test strategy skill must not self-approve a fixed test plan'
require_text \
  'required independent final-diff review has reached convergence' \
  .agents/skills/validation-closeout-session/SKILL.md \
  'validation closeout must not bypass final-diff review convergence'
require_text \
  'go-structural-quality-review' \
  .codex/agents/quality-agent.toml \
  'the quality reviewer must expose structural-quality review'
require_text \
  'This is a concurrency limit, not a cap on justified sequential lanes or review iterations.' \
  docs/subagent-contract.md \
  'the runtime lane contract must distinguish concurrency from total review work'
require_text \
  'structured/orchestrated implementation includes independent final-diff review before closeout' \
  README.md \
  'the user-facing workflow summary must name independent final-diff review'

for convergence_skill in \
  .agents/skills/specification-session/SKILL.md \
  .agents/skills/specification-review/SKILL.md \
  .agents/skills/technical-design-session/SKILL.md \
  .agents/skills/test-design-session/SKILL.md \
  .agents/skills/planning-session/SKILL.md \
  .agents/skills/spec-document-designer/SKILL.md \
  .agents/skills/go-design-spec/SKILL.md \
  .agents/skills/planning-and-task-breakdown/SKILL.md; do
  require_text \
    'shared convergence condition' \
    "${convergence_skill}" \
    'phase and authoring skills must not reduce readiness to a one-pass or no-blocker gate'
done

for reference in .agents/skills/go-coder/references/*.md; do
  reference_link="references/$(basename "${reference}")"
  if ! grep -Fq -- "](${reference_link})" .agents/skills/go-coder/SKILL.md; then
    echo "workflow instruction check failed: go-coder reference is not routed"
    echo "  reference: ${reference}"
    exit 1
  fi
done

forbidden_internal_target_pattern='next[_ -]?phase[=: ]+(research[-_ ]?synthesis[-_ ]?(challenge|review)|specification[-_ ]?review|technical[-_ ]?design[-_ ]?review|test[-_ ]?design([-_ ]?qa)?[-_ ]?review|task[-_ ]?(review([/_ -]?readiness)?|readiness[-_ ]?review)|readiness[-_ ]?review|post[-_ ]?code[-_ ]?review|validation|closeout)'
for forbidden_target in \
  'next_phase=research-synthesis-challenge' \
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

echo "workflow instruction check passed: ${#workflow_files[@]} canonical files"
