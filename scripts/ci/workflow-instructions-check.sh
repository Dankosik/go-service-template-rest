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
  .agents/skills/grilling/SKILL.md
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
  'complete review result and stop at the requested review boundary' \
  docs/spec-first-workflow/shared/subagents-and-handoff.md \
  'a standalone review must preserve the complete review result'
require_text \
  '../../subagent-contract.md#shared-review-finding-envelope' \
  docs/spec-first-workflow/shared/subagents-and-handoff.md \
  'review returns must link to the shared finding envelope'
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
  'A small direct change does not need an independent reviewer merely because a Goal exists.' \
  AGENTS.md \
  'Goal closeout must not trigger review for small direct work'
require_text \
  'after acceptance, a fresh worker owns the next task' \
  AGENTS.md \
  'repository authority must require a fresh worker per accepted ledger task'
require_text \
  'Objective: <one next outcome>' \
  docs/spec-first-workflow/shared/subagents-and-handoff.md \
  'non-implementation handoffs must use Objective instead of Goal'
require_text \
  '`Goal:` is reserved for that implementation handoff' \
  docs/spec-first-workflow/shared/subagents-and-handoff.md \
  'Goal terminology must be reserved for the implementation handoff'
require_text \
  '### E42 — Autonomous Challenge Authority And Continuation' \
  docs/spec-first-workflow-evals.md \
  'the autonomous challenge authority and continuation eval is missing'
require_text \
  'The root returns `ACCEPT` or `OVERRIDE`' \
  docs/spec-first-workflow-evals.md \
  'the autonomous challenge eval must exercise a recorded root disposition'
require_text \
  '### E43 — Autonomous Challenge Exhaustion Freshness And Review Separation' \
  docs/spec-first-workflow-evals.md \
  'the autonomous challenge exhaustion and freshness eval is missing'
require_text \
  'the material authority change invalidates it and triggers one fresh probe' \
  docs/spec-first-workflow-evals.md \
  'the autonomous challenge eval must invalidate completion after a material change'
require_text \
  '"id": 9' \
  .agents/skills/grilling/evals/evals.json \
  'the grilling skill evals must cover internal completion and invalidation'
require_text \
  'run one autonomous read-only challenge probe before the separate reviewer' \
  docs/spec-first-workflow.md \
  'the workflow router must place one autonomous challenge before review'
require_text \
  'Specification, combined Technical Design, Test Design, Planning, and an explicit `research only` boundary' \
  docs/spec-first-workflow.md \
  'the workflow router must name every applicable owning boundary'
require_text \
  'Direct work, supporting steps, and Implementation/Validation/Closeout do not run this probe.' \
  docs/spec-first-workflow.md \
  'the workflow router must exclude direct, supporting, and implementation work'
require_text \
  '## Autonomous Pre-Review Challenge' \
  docs/spec-first-workflow/shared/subagents-and-handoff.md \
  'the canonical autonomous challenge protocol is missing'
require_text \
  '`QUESTION`, `HUMAN_REQUIRED`, `REOPEN`, or `DONE`' \
  docs/spec-first-workflow/shared/subagents-and-handoff.md \
  'the autonomous challenge must expose exactly one event vocabulary'
require_text \
  'Continue dependent turns through the same challenger with the exact latest candidate.' \
  docs/spec-first-workflow/shared/subagents-and-handoff.md \
  'dependent challenge turns must reuse the same child and latest candidate'
require_text \
  '`ACCEPT`, `OVERRIDE`, or `RECLASSIFY`' \
  docs/spec-first-workflow/shared/subagents-and-handoff.md \
  'the root challenge disposition envelope is incomplete'
require_text \
  '`CONTINUE_INDEPENDENT`, `WAIT_HUMAN`, or `REOPEN_OWNER`' \
  docs/spec-first-workflow/shared/subagents-and-handoff.md \
  'the root challenge continuation envelope is incomplete'
require_text \
  'The owning candidate is authoritative; the child transcript is not.' \
  docs/spec-first-workflow/shared/subagents-and-handoff.md \
  'the autonomous challenge must keep state in the owning candidate'
require_text \
  'Do not create a probe transcript, receipt, queue, status, lifecycle field, or review verdict.' \
  docs/spec-first-workflow/shared/subagents-and-handoff.md \
  'the autonomous challenge must not create a parallel lifecycle artifact'
require_text \
  'After `DONE`, a different read-only child reviews the exact latest candidate' \
  docs/spec-first-workflow/shared/subagents-and-handoff.md \
  'the challenger and required reviewer must remain separate'
require_text \
  'Explicit user-requested grilling remains a root-to-user dialogue' \
  docs/spec-first-workflow/shared/subagents-and-handoff.md \
  'explicit user grilling must not be relayed through the internal challenger'
require_text \
  'may apply a materially triggered specialist method locally but never delegates recursively' \
  docs/spec-first-workflow/shared/subagents-and-handoff.md \
  'the internal challenger must not delegate recursively'
require_text \
  "run the router's autonomous pre-review challenge before the separate required reviewer" \
  AGENTS.md \
  'the global contract must route applicable candidates through the autonomous challenge'
require_text \
  'spec-first-workflow/shared/subagents-and-handoff.md#autonomous-pre-review-challenge' \
  docs/subagent-contract.md \
  'the portable subagent contract must link to the canonical challenge protocol'
require_text \
  '## Explicit user mode' \
  .agents/skills/grilling/SKILL.md \
  'the grilling skill must preserve explicit root-to-user mode'
require_text \
  '## Internal challenger mode' \
  .agents/skills/grilling/SKILL.md \
  'the grilling skill must define internal challenger mode'
require_text \
  'Internal macro-phase grilling' \
  .codex/agents/challenger-agent.toml \
  'the existing challenger agent must dispatch internal grilling mode'
require_text \
  'Use only when entering the implementation/validation/closeout macro phase' \
  .agents/skills/codex-goal-prompt-composer/SKILL.md \
  'the Goal composer must be gated to implementation entry'
require_text \
  'Those are internal next actions, not handoffs.' \
  .agents/skills/codex-goal-prompt-composer/SKILL.md \
  'the Goal composer must reject internal-checkpoint handoffs'
require_text \
  'A Goal does not itself trigger independent review for small direct work.' \
  .agents/skills/codex-goal-prompt-composer/SKILL.md \
  'the Goal composer must not turn small direct work into reviewed work'
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
  'complete result with the revision anchor, stated evidence boundary, affected-lens dispositions, an explicit no-findings statement, and `PASS`' \
  docs/spec-first-workflow-evals.md \
  'a clean standalone Specification review must return a complete PASS result'
require_text \
  '### E18 — Implementation Worker Acceptance Loop' \
  docs/spec-first-workflow-evals.md \
  'the implementation worker acceptance behavior case is missing'
require_text \
  'T07 remains open and T08 does not start' \
  docs/spec-first-workflow-evals.md \
  'the worker loop must block advancement until root acceptance'
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
  'For each material mechanism decision' \
  docs/spec-first-workflow/phases/system-integration-design.md \
  'system design must tie each material mechanism decision to evidence and consequences'
require_text \
  'apply `go-architect-spec` and its Required Evidence/Deliverable and Stop Conditions' \
  docs/spec-first-workflow/phases/system-integration-design.md \
  'material architecture decisions must route through the canonical architecture method'
require_text \
  'package/file placement will not reopen a material decision about system boundaries/topology' \
  docs/spec-first-workflow/phases/system-integration-design.md \
  'system design must close architecture before Go ownership'
require_text \
  '[persistence trigger](../shared/artifact-model.md#when-to-persist) applies' \
  docs/spec-first-workflow/phases/system-integration-design.md \
  'system design artifacts must remain conditional'
require_text \
  'When a durable design artifact is triggered' \
  .agents/skills/technical-design-session/SKILL.md \
  'the technical-design wrapper must preserve conditional artifact creation'
require_text \
  "Follow the system/integration phase's fan-out and review rules." \
  .agents/skills/technical-design-session/SKILL.md \
  'the technical-design wrapper must not duplicate fan-out or review policy'
require_text \
 'inventory every input-bearing design surface on the current implementation completion path' \
 docs/spec-first-workflow/phases/technical-design-review.md \
 'technical-design review must attempt full-path input-bearing artifact construction'
require_text \
  'a phase-level `PASS` requires rechecking implementation-input closure across every materially distinct input-bearing surface on the current completion path' \
  docs/spec-first-workflow/phases/technical-design-review.md \
  'focused re-review must not bypass phase-level input closure'
require_text \
  'Missing test code or fixtures establish a proof gap, not behavioral fail-before evidence.' \
  docs/spec-first-workflow/phases/test-design.md \
  'test design must not treat absent test code as behavioral RED evidence'
require_text \
  'only that owner may narrow or split the outcome' \
  docs/spec-first-workflow/phases/test-design.md \
  'test design must fail and reopen the accepted-outcome owner instead of changing scope itself'
require_text \
  'may remain only for a task and claim already outside the accepted current implementation completion' \
  docs/spec-first-workflow/phases/test-design.md \
  'unavailable test proof must be excluded from the accepted current completion'
require_text \
  'Test Design does not edit `tasks.md`.' \
  docs/spec-first-workflow/phases/test-design.md \
  'test design must leave ledger ownership to planning'
require_text \
  'Omission is not disposition.' \
  docs/spec-first-workflow/phases/test-design.md \
  'test design must reconcile material claims in both traceability directions'
require_text \
  'a named non-test proof that can falsify the claim' \
  docs/spec-first-workflow/phases/test-design.md \
  'test design non-test proof must be falsifying rather than an escape label'
require_text \
  'smallest set of complementary proof boundaries' \
  docs/spec-first-workflow/phases/test-design.md \
  'test design must choose complementary proof instead of one broad level'
require_text \
  'plausible incorrect observable behavior or regression that its oracle would reject' \
  docs/spec-first-workflow/phases/test-design.md \
  'test design scenarios must discriminate behavior rather than mirror implementation'
require_text \
  'its name, prior green status, or coverage hit is not proof' \
  docs/spec-first-workflow/phases/test-design.md \
  'test design must inspect existing proof before reusing it'
require_text \
  'A proof command is adequate only when it executes the relevant path and its result can establish the named oracle' \
  docs/spec-first-workflow/phases/test-design.md \
  'test design must bind commands to the exercised path and oracle'
require_text \
  'Merge rows with the same risk, trigger, oracle, and reopen path.' \
  docs/spec-first-workflow/phases/test-design.md \
  'test design must normalize equivalent scenario rows'
require_text \
  'cold-walk every mandatory dependency path from each dependency root through final validation' \
  docs/spec-first-workflow/phases/planning.md \
  'planning must dry-run every root-to-validation completion path'
require_text \
  'Never approve `PASS subject to gates` for a mandatory path.' \
  docs/spec-first-workflow/phases/planning.md \
  'planning must reject conditional readiness for mandatory work'
require_text \
  'Before drafting tasks, identify every in-scope accepted obligation' \
  docs/spec-first-workflow/phases/planning.md \
  'planning must inventory accepted obligations before drafting tasks'
require_text \
  'A command is not proof unless its expected observable can establish that claim.' \
  docs/spec-first-workflow/phases/planning.md \
  'planning proof must bind its command to a claim and observable'
require_text \
  'unless the accepted proof strategy requires manual observation' \
  docs/spec-first-workflow/phases/planning.md \
  'planning must preserve an accepted manual proof strategy'
require_text \
  'name the deterministic placement rule or canonical source that resolves the file choice' \
  docs/spec-first-workflow/phases/planning.md \
  'planning must resolve bounded implementation discovery deterministically'
require_text \
  "every task must trace to an accepted obligation, every proof to its task's claim" \
  docs/spec-first-workflow/phases/planning.md \
  'planning must distinguish task traceability from proof linkage'
require_text \
  'missing input required by a mandatory path through the current completion condition belongs in `Blocked stop`' \
  docs/spec-first-workflow/phases/planning.md \
  'planning must block known gaps on the current completion path'
require_text \
  'Treat each task as one coherent outcome that is independently reviewable once its declared prerequisites are satisfied.' \
  docs/spec-first-workflow/phases/planning.md \
  'planning tasks must remain coherent and reviewable without suppressing real prerequisites'
require_text \
  '`Reopen if` is optional and records only a concrete objective future condition' \
  docs/spec-first-workflow/phases/planning.md \
  'planning must omit rather than invent future invalidation conditions'
require_text \
  'document order, review preference, and convenient sequencing are not dependencies.' \
  docs/spec-first-workflow/phases/planning.md \
  'planning dependencies must represent true execution or proof gates'
require_text \
  'Treat rationale, rejected alternatives, non-normative examples, and future ideas as context, not implementation work.' \
  docs/spec-first-workflow/phases/planning.md \
  'planning must filter non-normative source material before drafting tasks'
require_text \
  'A no-implementation disposition must cite either current authoritative evidence that the obligation is already satisfied or an accepted upstream decision that no implementation change is required, plus its proving surface or objective recheck condition.' \
  docs/spec-first-workflow/phases/planning.md \
  'planning must require an authoritative basis and recheck for no implementation'
require_text \
  'no implementation delta may be duplicated or fall between task boundaries' \
  docs/spec-first-workflow/phases/planning.md \
  'planning must neither duplicate nor omit an implementation delta'
require_text \
  'An obligation may span multiple tasks when each carries the relevant constraint' \
  docs/spec-first-workflow/phases/planning.md \
  'planning must preserve cross-cutting obligations without merging distinct task deltas'
require_text \
  'executable in dependency order from current inputs' \
  docs/spec-first-workflow/phases/planning.md \
  'planning readiness must mean executable dependency closure rather than simultaneous task startability'
require_text \
  'Cold completion: can a fresh agent execute every mandatory path from each dependency root through final validation in dependency order' \
  docs/spec-first-workflow/phases/task-review-readiness.md \
  'task readiness must test multi-root end-to-end cold completion'
require_text \
  'A ledger cannot receive `PASS subject to gates` when a gate can block mandatory completion.' \
  docs/spec-first-workflow/phases/task-review-readiness.md \
  'readiness must reject unavailable inputs on any mandatory completion path'
require_text \
  'every mandatory task and proof through current completion is executable in dependency order from closed inputs' \
  docs/spec-first-workflow/phases/task-review-readiness.md \
  'readiness PASS must mean end-to-end execution readiness'
require_text \
  'invent a generic future trigger where no objective invalidation condition exists' \
  docs/spec-first-workflow/phases/task-review-readiness.md \
  'readiness must reject speculative reopen boilerplate'
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
  'fix the narrowest owning surface whose contract the reproducer proves is violated' \
  docs/spec-first-workflow-evals.md \
  'the regression eval must preserve the contract-owning repair boundary'
require_text \
  '### E27 — Independent Affected-Surface Review' \
  docs/spec-first-workflow-evals.md \
  'the independent affected-surface behavior case is missing'
require_text \
 '### E32 — Proportional Specification' \
 docs/spec-first-workflow-evals.md \
 'the proportional specification behavior case is missing'
require_text \
  '### E33 — System Mechanism Selection' \
  docs/spec-first-workflow-evals.md \
  'the system mechanism selection behavior case is missing'
require_text \
  'invariant/write/process authority, dominant workload, critical path' \
  docs/spec-first-workflow-evals.md \
  'system mechanism selection must use architecture decision drivers'
require_text \
  'naming either skill without the required result does not pass' \
  docs/spec-first-workflow-evals.md \
  'system mechanism selection must not reward bare skill routing claims'
require_text \
  'comparing only surviving substitutes at one live level' \
  docs/spec-first-workflow-evals.md \
  'system mechanism selection must compare substitutes at one decision level'
require_text \
  '### E34 — Proportional System Design' \
  docs/spec-first-workflow-evals.md \
  'the proportional system-design behavior case is missing'
require_text \
  'A passing response does not create a durable design artifact or invoke architecture/fan-out work.' \
  docs/spec-first-workflow-evals.md \
  'proportional system design must avoid untriggered ceremony'
require_text \
  '### E35 — Planning Bidirectional Coverage' \
  docs/spec-first-workflow-evals.md \
  'the bidirectional planning coverage behavior case is missing'
require_text \
  'map every accepted obligation to an executable task and adequate proof or an evidence-backed no-implementation disposition' \
  docs/spec-first-workflow-evals.md \
  'the planning coverage eval must reject omitted obligations and orphan tasks'
require_text \
  'preserving the legitimate bounded file choice, inspection bounds, and deterministic placement rule' \
  docs/spec-first-workflow-evals.md \
  'the planning coverage eval must preserve legitimate bounded discovery while rejecting avoidable discovery'
require_text \
  'one concise ready-for-Go-ownership disposition' \
  docs/spec-first-workflow-evals.md \
  'proportional system design must stop at a compact Go-ownership handoff'
require_text \
  'one clear, evidence-backed owner' \
  docs/spec-first-workflow/phases/go-code-ownership-design.md \
  'Go ownership design must produce one grounded owner per changed responsibility'
require_text \
  'the smallest interface in the consumer package' \
  docs/spec-first-workflow/phases/go-code-ownership-design.md \
  'Go ownership design must keep inversion boundaries consumer-owned and narrow'
require_text \
  '### E36 — Go Ownership Placement' \
  docs/spec-first-workflow-evals.md \
  'the focused Go ownership placement behavior case is missing'
require_text \
  'A competing proposal instead puts all new hand-written responsibilities into one already mixed file' \
  docs/spec-first-workflow-evals.md \
  'Go ownership placement must reject under-splitting into an already mixed owner'
require_text \
  'planning must still choose a material ownership, dependency, generated/manual, or exported-surface decision' \
  docs/spec-first-workflow-evals.md \
  'Go ownership placement must close implementation-facing ownership decisions'
require_text \
  '### E37 — Risk-Based Test Design' \
  docs/spec-first-workflow-evals.md \
  'the risk-based Test Design behavior case is missing'
require_text \
  'self-review or a merely declared QA `PASS` does not pass' \
  docs/spec-first-workflow-evals.md \
  'the Test Design eval must exhibit an independent fixed-revision review'
require_text \
  '### E38 — Adversarial Test-Plan Review' \
  docs/spec-first-workflow-evals.md \
  'the adversarial Test Design review case is missing'
require_text \
  'the vacuous `TD-006` oracle' \
  docs/spec-first-workflow-evals.md \
  'the adversarial Test Design review must reject vacuous assertions'
require_text \
  'not merely because the command is broad' \
  docs/spec-first-workflow-evals.md \
  'the adversarial Test Design review must reject proof gaps rather than broad commands'
require_text \
  '### E39 — Planning Source And Dependency Semantics' \
  docs/spec-first-workflow-evals.md \
  'the planning source and dependency semantics behavior case is missing'
require_text \
  'citation alone counts as coverage' \
  docs/spec-first-workflow-evals.md \
  'the planning semantics eval must reject coverage by citation without an executable obligation'
require_text \
  'two independent dependency roots' \
  docs/spec-first-workflow-evals.md \
  'the planning semantics eval must exercise a multi-root dependency graph'
require_text \
  'omit the generic `Reopen if` entries' \
  docs/spec-first-workflow-evals.md \
  'the planning semantics eval must reject speculative reopen boilerplate'
require_text \
  'preserve the accepted schema-version trigger on the affected task' \
  docs/spec-first-workflow-evals.md \
  'the planning semantics eval must preserve a valid future invalidation trigger'
require_text \
  '### E40 — Implementation Bidirectional Closeout And Proof Integrity' \
  docs/spec-first-workflow-evals.md \
  'the implementation closeout eval must cover bidirectional completeness and proof integrity'
require_text \
  'map every accepted obligation and every ledger task on the current completion path to its implementation or an already accepted evidence-backed no-implementation disposition, and to adequate proof; map every material change back to accepted scope' \
  docs/spec-first-workflow-evals.md \
  'the implementation closeout eval must detect omitted accepted work'
require_text \
  'remove the unrelated helper' \
  docs/spec-first-workflow-evals.md \
  'the implementation closeout eval must remove unrelated implementation work'
require_text \
  'reject green obtained by weakening or removing an oracle or bypassing a triggered gate' \
  docs/spec-first-workflow-evals.md \
  'the implementation closeout eval must reject proof-surface greenwashing'
require_text \
  '### E41 — Implementation Verification Repair Ownership' \
  docs/spec-first-workflow-evals.md \
  'the implementation verification ownership eval is missing'
require_text \
  'treat `partially verified` as the verification-step result, not the root phase result' \
  docs/spec-first-workflow-evals.md \
  'the verification helper boundary must not stop implementation-owned repair'
require_text \
  'race instrumentation without an exercising scenario' \
  docs/spec-first-workflow-evals.md \
  'the adversarial Test Design review must reject instrumentation-only proof'
require_text \
 'Preserve that precedence as an explicitly accepted normative decision while grounding any factual claims separately' \
 docs/spec-first-workflow-evals.md \
 'an accepted normative decision must not become an assumption or evidence gap'
require_text \
  '43 cases, 39 invariants' \
  scripts/dev/workflow-behavior-evals.sh \
  'the behavior eval harness must report the expanded invariant manifest'
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
  'fix the narrowest owning surface whose contract the reproducer proves is violated' \
  docs/spec-first-workflow/phases/implementation-validation-closeout.md \
  'implementation must fix the narrowest owning contract surface rather than one reported entry point'
require_text \
  'Treat edits to tests, fixtures, golden files, skip or exclusion settings, lint/build configuration, and proof commands as proof-surface changes.' \
  docs/spec-first-workflow/phases/implementation-validation-closeout.md \
  'implementation review must treat proof-surface mutations as part of the candidate diff'
require_text \
  'They require an accepted task or behavior reason' \
  docs/spec-first-workflow/phases/implementation-validation-closeout.md \
  'proof-surface changes must trace to an accepted task or behavior reason'
require_text \
  'Reconcile both directions: every accepted obligation and every ledger task on the current completion path maps to its implementation or an already accepted evidence-backed no-implementation disposition, and to adequate proof; every material change maps back to accepted scope.' \
  docs/spec-first-workflow/phases/implementation-validation-closeout.md \
  'implementation closeout must reconcile accepted work and changed files in both directions'
require_text \
  'Return success only when every critical obligation has passing proof, the tests are deterministic and reviewable, and every positive claim is bounded by commands actually run. Otherwise report the exact unresolved owner without claiming readiness.' \
  .agents/skills/go-qa-tester/SKILL.md \
  'go-qa-tester must not report success while a critical obligation lacks passing proof'
require_text \
  'Inside an active implementation/validation/closeout request, return the failure signal to the root' \
  .agents/skills/go-verification-before-completion/SKILL.md \
  'verification helper failure must return to active implementation repair'
require_text \
  'returns worker-owned failures to their task worker, and repairs only direct work' \
  .agents/skills/go-verification-before-completion/SKILL.md \
  'verification repair routing must preserve worker task ownership'
if grep -Fq -- 'In standalone use this ends' \
  .agents/skills/go-verification-before-completion/SKILL.md; then
  echo 'workflow instruction check failed: verification stop boundary is duplicated'
  exit 1
fi
require_text \
  'If the task is accepted, update its checkbox and evidence immediately' \
  docs/spec-first-workflow/phases/implementation-validation-closeout.md \
  'ledger progress must be durable after root task acceptance'
require_text \
  'Direct work that satisfies these conditions uses root diff inspection and bounded validation, not an independent reviewer.' \
  docs/spec-first-workflow.md \
  'the direct path must skip independent review by default'
require_text \
  'Small direct work uses root diff inspection and bounded validation under the same trigger rule.' \
  docs/spec-first-workflow/shared/subagents-and-handoff.md \
  'the shared review contract must skip independent review for small direct work'
require_text \
  'Small direct work follows the same explicit-or-risk trigger rule.' \
  docs/spec-first-workflow/phases/implementation-validation-closeout.md \
  'implementation closeout must skip independent review for small direct work'
require_text \
  'Do not launch an independent reviewer, workflow artifacts' \
  docs/spec-first-workflow-evals.md \
  'the small direct eval must reject review ceremony'
require_text \
  'inspect the resulting diff, run the focused proof' \
  docs/spec-first-workflow-evals.md \
  'the small direct eval must inspect the final diff before closeout'
require_text \
  'any required post-code review or focused re-review' \
  docs/spec-first-workflow/phases/implementation-validation-closeout.md \
  'implementation closeout must run review only when required or triggered'
if grep -Fq -- 'Validation, post-code review, in-scope repair, revalidation, fresh re-review, and closeout run automatically' \
  docs/spec-first-workflow/phases/implementation-validation-closeout.md; then
  echo 'workflow instruction check failed: implementation closeout still mandates review for every change'
  exit 1
fi
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
  'Implementation-owned gaps return to their task worker and cannot be relabeled as `blocked` or handed to the user.' \
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
  'When an independent gate is required, one whole-artifact or whole-diff reviewer is the default.' \
  docs/spec-first-workflow/shared/subagents-and-handoff.md \
  'triggered independent review must default to one coherence reviewer'
require_text \
  'Repeat only while a concrete new finding or semantic repair changes readiness.' \
  docs/spec-first-workflow/shared/subagents-and-handoff.md \
  'review convergence must continue only for concrete readiness-changing work'
require_text \
  'The one-at-a-time implementation worker loop is separate from that limit.' \
  docs/spec-first-workflow/shared/subagents-and-handoff.md \
  'delegation limits must not block the sequential implementation worker loop'
require_text \
  'applies compatible matching methods locally in one coherence pass' \
  docs/spec-first-workflow/shared/subagents-and-handoff.md \
  'the default reviewer must cover compatible lenses in one pass'
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
require_text \
  '# <User/operator-visible outcome>' \
  docs/spec-first-workflow/phases/specification.md \
  'the canonical spec shape must name the user/operator-visible outcome'
require_text \
  'Derive the affected behavior surface' \
  docs/spec-first-workflow/phases/specification.md \
  'spec authoring must derive affected lenses independently of existing spec content'
require_text \
  'Choose the smallest representation that removes ambiguity' \
  docs/spec-first-workflow/phases/specification.md \
  'spec authoring must adapt representation to the decision shape'
require_text \
  'Ground each decision-changing factual claim in current evidence and each' \
  docs/spec-first-workflow/phases/specification.md \
  'spec claims must distinguish evidence-backed facts from accepted normative decisions'
require_text \
  'Only uncertainty about proving an already accepted rule' \
  docs/spec-first-workflow/phases/specification.md \
  'a downstream proof obligation must not hide a missing specification decision'
require_text \
  'Reconstruct the affected behavior surface independently' \
  docs/spec-first-workflow/phases/specification-review.md \
  'spec review must reconstruct affected lenses independently of the spec'
require_text \
  'Do not treat omission from the spec as evidence' \
  docs/spec-first-workflow/phases/specification-review.md \
  'an omitted lens must not hide its own review trigger'
require_text \
  'two reasonable downstream implementations' \
  docs/spec-first-workflow/phases/specification-review.md \
  'spec review must detect behaviorally divergent compliant interpretations'
require_text \
  'not a second Specification template' \
  docs/spec-first-workflow/shared/artifact-model.md \
  'artifact status guidance must link to rather than duplicate the spec shape'

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
  'A semantic mutation after review' \
  docs/spec-first-workflow/shared/subagents-and-handoff.md \
  'semantic post-review mutations must invalidate affected convergence evidence'
require_text \
  'assign exactly that one task to one worker' \
  docs/spec-first-workflow/phases/implementation-validation-closeout.md \
  'ledger implementation must assign one task per worker'
require_text \
  'then launch a fresh worker for the next ready task' \
  docs/spec-first-workflow/phases/implementation-validation-closeout.md \
  'each accepted task must be followed by a fresh worker for the next task'
require_text \
  'return the same task to the same worker with concrete bounded gaps' \
  docs/spec-first-workflow/phases/implementation-validation-closeout.md \
  'the root must return incomplete work to its task worker'
require_text \
  'do not repair it in the root or start the next task' \
  docs/spec-first-workflow/phases/implementation-validation-closeout.md \
  'the root must not bypass worker task ownership or acceptance order'
require_text \
  'Every ledger task receives root acceptance review before the next task starts.' \
  docs/spec-first-workflow/phases/implementation-validation-closeout.md \
  'every worker task must receive root acceptance before advancement'
require_text \
  'implement one worker task at a time under root acceptance' \
  docs/spec-first-workflow.md \
  'the top-level router must use task-by-task worker acceptance'
require_text \
  'add independent final-diff review only when explicitly requested or concretely risk-triggered' \
  docs/spec-first-workflow.md \
  'the top-level router must keep final implementation review conditional'
if grep -Fq -- 'implement, validate, independently review the candidate final diff and evidence' \
  docs/spec-first-workflow.md; then
  echo 'workflow instruction check failed: the top-level router still mandates final review for every implementation'
  exit 1
fi
require_text \
  'then assigns the next ready task to a fresh worker' \
  docs/spec-first-workflow/shared/subagents-and-handoff.md \
  'the shared worker loop must use a fresh worker for the next task'
require_text \
  'For worker-owned implementation, return the findings to the worker that owns the affected task' \
  docs/spec-first-workflow/shared/subagents-and-handoff.md \
  'review findings must preserve implementation worker ownership'
require_text \
  'Every review return also names the exact revision or diff' \
  docs/subagent-contract.md \
  'the shared reviewer envelope must preserve revision and affected-lens coverage'
require_text \
  'assigns exactly one ready task to one worker' \
  docs/spec-first-workflow-evals.md \
  'the worker-loop eval must enforce one task per worker'
require_text \
  'returns concrete gaps to the same worker' \
  docs/spec-first-workflow-evals.md \
  'the worker-loop eval must preserve worker ownership across correction'
require_text \
  'reuses the T07 worker for T08' \
  docs/spec-first-workflow-evals.md \
  'the worker-loop eval must reject worker reuse across tasks'
require_text \
  'Compatible lenses fit one coherence review' \
  docs/spec-first-workflow-evals.md \
  'the delegation eval must keep compatible lenses in one reviewer'
require_text \
  'run only the one bounded security specialist' \
  docs/spec-first-workflow-evals.md \
  'the delegation eval must reject speculative specialist waves'
require_text \
  'launches a reviewer after a worker return' \
  docs/spec-first-workflow-evals.md \
  'the worker-loop eval must reject per-task reviewer lanes'
require_text \
  'produces a final diff with one concrete high-impact security question known before the final implementation gate' \
  docs/spec-first-workflow-evals.md \
  'specialist routing must identify the concrete question before the final gate'
require_text \
  'runs one bounded security specialist before the implementation gate' \
  docs/spec-first-workflow-evals.md \
  'specialist routing must fan in before one whole-diff review'
require_text \
  'A semantic post-review mutation requires revalidation and focused affected-lens review.' \
  docs/spec-first-workflow-evals.md \
  'post-review implementation mutations must trigger only affected focused review'
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
  'never approves its own readiness' \
  .agents/skills/spec-document-designer/SKILL.md \
  'the spec author wrapper must retain its unique self-approval boundary'
require_text \
  'Own the root Specification macro phase through its canonical Stop Rule' \
  .agents/skills/specification-session/SKILL.md \
  'the specification session wrapper must retain root phase ownership'
require_text \
  'Remain read-only' \
  .agents/skills/specification-review/SKILL.md \
  'the specification reviewer wrapper must retain its unique mutation boundary'
require_text \
  'Return the complete review result to the owning root' \
  .agents/skills/specification-review/SKILL.md \
  'the specification reviewer wrapper must preserve the complete review envelope'
require_text \
  'proceeds to planning only after independent technical-design review' \
  .agents/skills/go-design-spec/SKILL.md \
  'design authoring must route through independent review'
require_text \
  'runs independent task review/readiness' \
  .agents/skills/planning-and-task-breakdown/SKILL.md \
  'task authoring must route through readiness review'
require_text \
  'the canonical Stop Rule is unmet' \
  .agents/skills/planning-session/SKILL.md \
  'planning must not continue into implementation before readiness'
require_text \
  'run an independent QA review before planning' \
  docs/spec-first-workflow/phases/test-design.md \
  'test design must route through independent QA review'
require_text \
  'Own only that task until the root accepts it or returns concrete gaps' \
  .agents/skills/go-coder/SKILL.md \
  'the implementation worker must retain one task through corrections'
require_text \
  'root acceptance, not worker self-approval or a separate reviewer, decides when the next task starts' \
  .agents/skills/go-coder/SKILL.md \
  'the coder must return task acceptance authority to the root'
require_text \
  '`research only` is the structured/orchestrated macro-phase boundary' \
  .agents/skills/research-session/SKILL.md \
  'standalone structured or orchestrated research must route through independent convergence review'
require_text \
  'The reviewer returns one verdict under the shared convergence contract:' \
  docs/spec-first-workflow/phases/test-design.md \
  'test design review must use explicit convergence verdict semantics'
require_text \
  'standalone QA review returns the complete review result and stops read-only' \
  docs/spec-first-workflow/phases/test-design.md \
  'standalone test-design review must preserve the complete review envelope'
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
  'The owning root repairs review findings and re-reviews to the shared convergence condition' \
  .agents/skills/test-design-session/SKILL.md \
  'the test design wrapper must not self-approve a fixed proof disposition'
require_text \
  'Derive the oracle from approved behavior or an independent reference, not from production logic.' \
  .agents/skills/go-qa-tester-spec/SKILL.md \
  'the test strategy oracle must be independent from production logic'
require_text \
  'triggers a material fail path with no proof or authorized residual-risk disposition is a blocker' \
  .agents/skills/go-qa-tester-spec/SKILL.md \
  'the test strategy must block only triggered undispositioned fail paths'
require_text \
  'named local, CI, or controlled target environment' \
  .agents/skills/go-qa-tester-spec/SKILL.md \
  'test strategy commands must name their proving environment'
require_text \
  'Never rely on test order.' \
  .agents/skills/go-qa-tester-spec/SKILL.md \
  'test strategy must forbid order-dependent proof'
require_text \
  'if deterministic control is infeasible, any bounded wait must target a named observable condition' \
  .agents/skills/go-qa-tester-spec/SKILL.md \
  'test strategy wait fallback must remain bounded and observable'
require_text \
  'Keep proof inline when the canonical persistence rule does not require `test-plan.md`.' \
  .agents/skills/test-design-session/SKILL.md \
  'test strategy review must support inline proof when no test plan is persisted'
require_text \
  '[canonical Test Design Outputs](../../../docs/spec-first-workflow/phases/test-design.md#outputs)' \
  .agents/skills/go-qa-tester-spec/SKILL.md \
  'the test strategy skill must use the canonical scenario schema'
if grep -Eq -- 'proof_only|follow_up_only|no new decision required in' \
  .agents/skills/go-qa-tester-spec/SKILL.md; then
  echo 'workflow instruction check failed: test strategy uses an ambiguous disposition label'
  exit 1
fi
if grep -Eq -- 'plausible incorrect implementation|Missing test code or fixtures are a valid fail-before signal' \
  docs/spec-first-workflow/phases/test-design.md .agents/skills/go-qa-tester-spec/SKILL.md; then
  echo 'workflow instruction check failed: test design retains implementation-mirroring or false RED wording'
  exit 1
fi
require_text \
  'Accept the task and start the next worker only when its criteria and proof pass' \
  .agents/skills/validation-closeout-session/SKILL.md \
  'validation closeout must enforce root task acceptance before advancement'
require_text \
  'go-structural-quality-review' \
  .codex/agents/quality-agent.toml \
  'the quality reviewer must expose structural-quality review'
require_text \
  'The one-at-a-time implementation task loop is separate from that lane limit.' \
  docs/subagent-contract.md \
  'the runtime lane contract must distinguish task workers from review lanes'
require_text \
  'one worker owns one task until the root accepts its diff and proof or returns concrete gaps' \
  README.md \
  'the user-facing workflow summary must describe worker task acceptance'
require_text \
  'Small direct work uses root diff inspection and bounded validation without an independent reviewer unless the user or a concrete risk trigger requires one.' \
  README.md \
  'the user-facing workflow summary must skip review for small direct work'

for convergence_skill in \
  .agents/skills/technical-design-session/SKILL.md \
  .agents/skills/test-design-session/SKILL.md \
  .agents/skills/planning-session/SKILL.md \
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

for reference in .agents/skills/go-qa-tester/references/*.md; do
  reference_link="references/$(basename "${reference}")"
  if ! grep -Fq -- "](${reference_link})" .agents/skills/go-qa-tester/SKILL.md; then
    echo "workflow instruction check failed: go-qa-tester reference is not routed"
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

probe_lifecycle_artifacts="$(
  find docs .agents .codex specs -type f -print 2>/dev/null \
    | grep -Ei '(^|/)[^/]*(grill|probe|challenge)[^/]*(receipt|transcript|queue|status)[^/]*$' \
    || true
)"
if [[ -n "${probe_lifecycle_artifacts}" ]]; then
  echo 'workflow instruction check failed: an autonomous challenge lifecycle artifact exists'
  printf '%s\n' "${probe_lifecycle_artifacts}" | sed 's/^/  /'
  exit 1
fi

echo "workflow instruction check passed: ${#workflow_files[@]} canonical files"
