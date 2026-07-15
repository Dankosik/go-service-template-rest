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
  'are internal checkpoints of their non-implementation owning macro phase' \
  docs/spec-first-workflow/shared/subagents-and-handoff.md \
  'internal non-implementation review must remain inside its owning macro phase'
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
  'On entering implementation/validation/closeout, the root creates or continues exactly one root-thread Codex Goal' \
  AGENTS.md \
  'implementation entry must establish exactly one root Codex Goal'
require_text \
  'root acceptance of every Worker result, root review of the final integrated diff, and fresh validation evidence' \
  AGENTS.md \
  'Goal closeout must require root-owned implementation review'
require_text \
  'never launches a built-in subagent lane' \
  AGENTS.md \
  'repository authority must forbid built-in subagent lanes during implementation'
require_text \
  'An implementation Worker is a separate OS process launched with `codex exec` in an isolated Git worktree.' \
  AGENTS.md \
  'repository authority must define an external CLI Worker'
require_text \
  'It is never a built-in subagent, `spawn_agent`, `agent_type="worker"`, or another in-process role.' \
  AGENTS.md \
  'repository authority must reject built-in implementation workers'
require_text \
  'Every authorized implementation outcome is produced by an external Worker' \
  AGENTS.md \
  'repository authority must route direct and ledger implementation through external Workers'
require_text \
  'the root never authors or repairs Worker-owned implementation' \
  AGENTS.md \
  'the root must remain orchestration-only for implementation patches'
require_text \
  'is the sole owner of Worker assignment, lifecycle, launch and resume, brief, evidence, acceptance, and integration mechanics' \
  AGENTS.md \
  'repository authority must leave Worker mechanics to the implementation phase'
require_text \
  'Use this exact locally validated runtime contract:' \
  docs/spec-first-workflow/phases/implementation-validation-closeout.md \
  'the Worker runtime contract must be identified as locally validated'
require_text \
  'They never implement or repair code, config, docs, or tests.' \
  AGENTS.md \
  'built-in subagents must remain read-only'
require_text \
  'Clear read-only and external-action boundaries.' \
  docs/spec-first-workflow/shared/subagents-and-handoff.md \
  'subagent inputs must define an explicit read-only boundary'
require_text \
  'Constraints: <read-only boundary, non-goals, external-action limits>' \
  docs/spec-first-workflow/shared/subagents-and-handoff.md \
  'the canonical lane brief must remain read-only'
require_text \
  '<read-only boundary, non-goals, external-action limits>' \
  docs/subagent-brief-template.md \
  'the portable subagent brief must remain read-only'
if grep -Eq -- 'read/write|read-only/write' \
  docs/spec-first-workflow/shared/subagents-and-handoff.md docs/subagent-brief-template.md; then
  echo 'workflow instruction check failed: a built-in subagent brief still permits writes'
  exit 1
fi
require_text \
  'CLI Worker Launch And Resume' \
  docs/spec-first-workflow/phases/implementation-validation-closeout.md \
  'the implementation phase must own CLI Worker mechanics'
require_text \
  'The exact allowlisted Worker models are `gpt-5.6-terra` and `gpt-5.6-sol`, validated against the OpenAI latest-model guide and native Codex catalog on 2026-07-15.' \
  docs/spec-first-workflow/phases/implementation-validation-closeout.md \
  'the Worker model allowlist must record its current official authority and validation date'
require_text \
  'Before launch, the root sets and records `WORKER_MODEL` from observable characteristics of the accepted outcome; never inherit a default, use the floating `gpt-5.6` alias, or select another model.' \
  docs/spec-first-workflow/phases/implementation-validation-closeout.md \
  'Worker model selection must be explicit, recorded, exact, and task-specific'
require_text \
  'Terra is the normal efficiency choice only when the outcome is clear, bounded and local, low consequence, free of unresolved design or ownership judgment, covered by relevant automated proof, and readily inspectable and falsifiable by the root.' \
  docs/spec-first-workflow/phases/implementation-validation-closeout.md \
  'the Worker contract must define the conditional Terra route'
require_text \
  'Sol is required for material uncertainty or an ambiguous, open-ended, difficult-to-debug, cross-boundary or cross-cutting outcome; material architecture or product judgment; a protected or high-consequence domain named by repository authority; a large evidence, tool, or context load; or a result that is difficult for the root to falsify locally.' \
  docs/spec-first-workflow/phases/implementation-validation-closeout.md \
  'the Worker contract must define the complexity and risk Sol route'
require_text \
  'Before the next Worker launch after either a Codex CLI upgrade or a change in the latest-model guide' \
  docs/spec-first-workflow/phases/implementation-validation-closeout.md \
  'the Worker model allowlist must be revalidated after either freshness trigger'
require_text \
  'revalidate and coordinate updates to the exact model allowlist, catalog transformation, and instruction checks.' \
  docs/spec-first-workflow/phases/implementation-validation-closeout.md \
  'model freshness changes must update all coupled contract surfaces'
require_text \
  'If the selected model is unavailable, freshness validation fails, or the effective model differs, stop; never silently fall back or substitute another model.' \
  docs/spec-first-workflow/phases/implementation-validation-closeout.md \
  'model selection must fail closed without fallback'
require_text \
  'WORKER_MODEL_CATALOG="$(mktemp -t codex-worker-model-catalog.XXXXXX.json)"' \
  docs/spec-first-workflow/phases/implementation-validation-closeout.md \
  'the Worker model catalog must be transient'
require_text \
  '/opt/homebrew/bin/rtk proxy codex debug models | /opt/homebrew/bin/rtk proxy jq -e --arg worker_model "$WORKER_MODEL"' \
  docs/spec-first-workflow/phases/implementation-validation-closeout.md \
  'the Worker catalog must derive from the native full catalog through raw RTK proxy mode'
require_text \
  "type == \"object\" and (.models | type == \"array\")" \
  docs/spec-first-workflow/phases/implementation-validation-closeout.md \
  'the derived Worker catalog must be validated as raw JSON before launch'
require_text \
  'RTK filtering corrupts this large JSON stream; both commands must use raw proxy mode.' \
  docs/spec-first-workflow/phases/implementation-validation-closeout.md \
  'the Worker catalog must not use RTK-filtered JSON output'
require_text \
  '(["gpt-5.6-terra", "gpt-5.6-sol"] | index($worker_model)) as $allowlisted' \
  docs/spec-first-workflow/phases/implementation-validation-closeout.md \
  'catalog derivation must allow only the exact Terra and Sol slugs'
require_text \
  '([.models[] | select(.slug == $worker_model)] | length) as $matches' \
  docs/spec-first-workflow/phases/implementation-validation-closeout.md \
  'catalog derivation must count the pinned model exactly'
require_text \
  '| if $allowlisted == null then' \
  docs/spec-first-workflow/phases/implementation-validation-closeout.md \
  'catalog derivation must fail unless the selected model is allowlisted'
require_text \
  'elif $matches != 1 then' \
  docs/spec-first-workflow/phases/implementation-validation-closeout.md \
  'catalog derivation must fail unless exactly one selected model exists'
require_text \
  'error("expected exactly one selected Worker model")' \
  docs/spec-first-workflow/phases/implementation-validation-closeout.md \
  'catalog derivation must fail closed on a model-count mismatch'
require_text \
  '.models |= map(' \
  docs/spec-first-workflow/phases/implementation-validation-closeout.md \
  'catalog derivation must preserve the full model list'
require_text \
  'if .slug == $worker_model then .multi_agent_version = null else . end' \
  docs/spec-first-workflow/phases/implementation-validation-closeout.md \
  'catalog derivation must mutate only the pinned model multi-agent field'
require_text \
  'Retain this exact file unchanged for the whole Worker session, pass it again on every resume' \
  docs/spec-first-workflow/phases/implementation-validation-closeout.md \
  'launch and resume must reuse the exact derived catalog'
require_text \
  'Do not regenerate it between launch and resume, persist it as a workflow artifact, copy a pinned catalog into the repository, or mutate user/global Codex configuration.' \
  docs/spec-first-workflow/phases/implementation-validation-closeout.md \
  'catalog override must remain transient and session-scoped'
if ! awk '
  BEGIN {
    required[1] = "--model \"$WORKER_MODEL\""
    required[2] = "model_catalog_json=\\\"$WORKER_MODEL_CATALOG\\\""
    required[3] = "model_reasoning_effort=\\\"$WORKER_REASONING_EFFORT\\\""
    required[4] = "features.multi_agent=false"
    required[5] = "features.multi_agent_v2.enabled=false"
    required[6] = "features.multi_agent_v2.max_concurrent_threads_per_session=1"
    required[7] = "--cd \"$WORKTREE\""
    required[8] = "--sandbox workspace-write"
    required[9] = "--ask-for-approval never"
    required[10] = "--strict-config"
    required[11] = "--json"
    required[12] = "--output-schema \"$WORKER_SCHEMA\""
    required[13] = "-o \"$WORKER_FINAL\""
    required[14] = "--disable chronicle"
    required[15] = "--disable goals"
    required[16] = "--disable memories"
    required[17] = "--enable hooks"
    required[18] = "${WORKER_CAPABILITY_CONFIG[@]}"
    required[19] = "${WORKER_CODEGRAPH_CONFIG[@]}"
    required[20] = "2> \"$WORKER_STDERR\""
    required[21] = "notify=[]"
    required[22] = "check_for_update_on_startup=false"
  }
  /^### CLI Worker Launch And Resume$/ { section = 1; next }
  section && /^```bash$/ {
    candidate = 1
    next
  }
  section && candidate && /^```$/ {
    candidate = 0
    next
  }
  section && candidate && /^\/opt\/homebrew\/bin\/rtk proxy codex \\$/ {
    block = 1
    blocks++
    candidate = 0
    for (i = 1; i <= 22; i++) count[i] = 0
    next
  }
  section && block && /^```$/ {
    for (i = 1; i <= 22; i++) {
      if (count[i] != 1) {
        printf "worker command block %d requires exactly one %s; found %d\n", blocks, required[i], count[i] > "/dev/stderr"
        failed = 1
      }
    }
    block = 0
    next
  }
  section && block {
    for (i = 1; i <= 22; i++) {
      if (index($0, required[i])) count[i]++
    }
  }
  END { if (blocks != 2 || block || failed) exit 1 }
' docs/spec-first-workflow/phases/implementation-validation-closeout.md; then
  echo 'workflow instruction check failed: launch and resume must each contain every CLI Worker contract item exactly once'
  exit 1
fi
require_text \
  '- `WORKER_CAPABILITY_CONFIG`: task-gated optional-capability baseline.' \
  docs/spec-first-workflow/phases/implementation-validation-closeout.md \
  'the transient optional-capability baseline must be defined'
require_text \
  'The root removes or replaces only the matching baseline override when the accepted task or its automated proof requires that capability' \
  docs/spec-first-workflow/phases/implementation-validation-closeout.md \
  'optional capabilities must be task-gated and recorded'
require_text \
  'prefer a specific documentation MCP over broad web or shell access.' \
  docs/spec-first-workflow/phases/implementation-validation-closeout.md \
  'the capability policy must prefer a narrow documentation source'
require_text \
  'Hooks are a Worker readiness dependency: before launch, root verifies active command-hook definitions are reviewed and trusted; a changed or untrusted hook stops for review instead of using `--dangerously-bypass-hook-trust`.' \
  docs/spec-first-workflow/phases/implementation-validation-closeout.md \
  'Worker hooks must be reviewed and trusted before launch'
if grep -Fq -- '--disable hooks' docs/spec-first-workflow/phases/implementation-validation-closeout.md; then
  echo 'workflow instruction check failed: Worker hooks must remain enabled'
  exit 1
fi
if grep -F -- '--dangerously-bypass-hook-trust' \
  docs/spec-first-workflow/phases/implementation-validation-closeout.md \
  | grep -Fvx 'Hooks are a Worker readiness dependency: before launch, root verifies active command-hook definitions are reviewed and trusted; a changed or untrusted hook stops for review instead of using `--dangerously-bypass-hook-trust`.' \
  >/dev/null; then
  echo 'workflow instruction check failed: Worker hooks must not bypass trust'
  exit 1
fi
require_text \
  "If it fails or reports \`Not initialized\`, set \`WORKER_CODEGRAPH_CONFIG=(-c 'mcp_servers.codegraph.enabled=false')\`" \
  docs/spec-first-workflow/phases/implementation-validation-closeout.md \
  'uninitialized CodeGraph must be disabled for Worker launch and resume'
require_text \
  'record the raw-navigation fallback in the Worker brief' \
  docs/spec-first-workflow/phases/implementation-validation-closeout.md \
  'the CodeGraph fallback must be recorded in the Worker brief'
require_text \
  'Worker setup never initializes or reindexes CodeGraph.' \
  docs/spec-first-workflow/phases/implementation-validation-closeout.md \
  'Worker setup must not initialize or reindex CodeGraph'
require_text \
  'Both allowlisted models permit only `low`, `medium`, `high`, `xhigh`, or `max`; reject `ultra` because it requests automatic task delegation.' \
  docs/spec-first-workflow/phases/implementation-validation-closeout.md \
  'Worker reasoning effort must exclude the automatic-delegation mode'
require_text \
  'Select model and reasoning effort separately: do not compensate for a Sol-shaped task by maximizing Terra.' \
  docs/spec-first-workflow/phases/implementation-validation-closeout.md \
  'Worker model and reasoning effort selection must remain independent'
require_text \
  'Choose effort from task complexity, ambiguity, consequence of error, evidence/tool load, and latency/cost, and calibrate both routing and effort on representative local evals; do not assume the highest effort is best.' \
  docs/spec-first-workflow/phases/implementation-validation-closeout.md \
  'Worker reasoning effort must be workload-selected and eval-calibrated'
require_text \
  '`low` for mechanical, well-specified, low-risk work; `medium` as the ordinary implementation baseline; `high` or `xhigh` for complex or ambiguous work when representative evals show a material quality gain; and `max` only for the hardest quality-first work.' \
  docs/spec-first-workflow/phases/implementation-validation-closeout.md \
  'Worker reasoning effort levels must have explicit selection guidance'
require_text \
  'The command snippets are exact raw RTK proxy invocations; execute them as shown.' \
  docs/spec-first-workflow/phases/implementation-validation-closeout.md \
  'Worker launch commands must retain the exact raw RTK proxy invocation'
for worker_variable in WORKTREE WORKER_MODEL WORKER_BRIEF WORKER_SCHEMA WORKER_FINAL WORKER_EVENTS WORKER_STDERR; do
  require_text \
    "- \`${worker_variable}\`:" \
    docs/spec-first-workflow/phases/implementation-validation-closeout.md \
    "the transient Worker variable ${worker_variable} must be defined"
done
require_text \
  'Use distinct `WORKER_EVENTS`, `WORKER_FINAL`, and `WORKER_STDERR` paths for the initial launch and every resume attempt; never overwrite prior Worker evidence.' \
  docs/spec-first-workflow/phases/implementation-validation-closeout.md \
  'Worker launch and resume evidence paths must remain distinct'
require_text \
  'An `item.completed` event with `item.type="error"` is non-terminal by shape: inspect its message rather than treating it as automatic failure or ignoring it.' \
  docs/spec-first-workflow/phases/implementation-validation-closeout.md \
  'JSONL error-shaped items must receive semantic intake rather than automatic failure or suppression'
require_text \
  'Do not globally filter stderr or warning events.' \
  docs/spec-first-workflow/phases/implementation-validation-closeout.md \
  'Worker diagnostics must not be globally filtered'
require_text \
  'Technical success requires exit status zero, exactly one session ID, a completed turn, a schema-valid structured final result, the selected model with no reroute or substitution, and no unresolved permission or contract violation.' \
  docs/spec-first-workflow/phases/implementation-validation-closeout.md \
  'Worker intake must enforce the complete technical-success contract'
require_text \
  'Preserve and inspect `WORKER_STDERR`; benign diagnostics alone do not fail an otherwise complete run.' \
  docs/spec-first-workflow/phases/implementation-validation-closeout.md \
  'stderr must remain evidence without making benign diagnostics terminal'
require_text \
  'required repository skills when applicable' \
  docs/spec-first-workflow/phases/implementation-validation-closeout.md \
  'the outcome-first Worker brief must name required repository skills'
require_text \
  'top-level `--add-dir "$TASK_WRITABLE_PATH"` after `--cd` and before `exec`' \
  docs/spec-first-workflow/phases/implementation-validation-closeout.md \
  'read-only task paths must use a narrow top-level add-dir before exec'
require_text \
  'Grant only that path, keep `workspace-write` and approval `never`, and never broaden the whole sandbox.' \
  docs/spec-first-workflow/phases/implementation-validation-closeout.md \
  'read-only path recovery must preserve the narrow sandbox'
if grep -Eq -- 'model_reasoning_effort=.*(low|medium|high|xhigh|max|ultra)' \
  docs/spec-first-workflow/phases/implementation-validation-closeout.md docs/spec-first-workflow-evals.md; then
  echo 'workflow instruction check failed: Worker reasoning effort is hard-coded'
  exit 1
fi
if ! awk '
  /^  --disable chronicle/ { chronicle_line = NR; next }
  /^  --ask-for-approval never/ { approval_line = NR; next }
  /^  exec([[:space:]]|$)/ {
    if (!chronicle_line || chronicle_line > NR || !approval_line || approval_line > NR) exit 1
    pairs++
    chronicle_line = 0
    approval_line = 0
  }
  END { if (pairs != 2) exit 1 }
' docs/spec-first-workflow/phases/implementation-validation-closeout.md; then
  echo 'workflow instruction check failed: deterministic disables and --ask-for-approval must precede exec for launch and resume'
  exit 1
fi
require_text \
  'exec resume "$WORKER_SESSION_ID"' \
  docs/spec-first-workflow/phases/implementation-validation-closeout.md \
  'bounded corrections must use codex exec resume with the recorded session'
require_text \
  'After launch, read `WORKER_SESSION_ID` from the first `thread.started.thread_id` event in `WORKER_EVENTS`.' \
  docs/spec-first-workflow/phases/implementation-validation-closeout.md \
  'Worker intake must deterministically capture the launched session ID'
require_text \
  'A missing, blank, or ambiguous ID is a launch/intake failure; do not start a replacement Worker.' \
  docs/spec-first-workflow/phases/implementation-validation-closeout.md \
  'Worker intake failure must not create a replacement Worker'
require_text \
  'resume the same Worker with the same `WORKER_MODEL`, reasoning effort, unchanged catalog file, worktree, sandbox, approval policy, output schema, and multi-agent-disabled flags; never reroute the session or reselect its model' \
  docs/spec-first-workflow/phases/implementation-validation-closeout.md \
  'Worker correction resumes must preserve the full launch invariant'
require_text \
  'Because `codex exec --json` may omit the model, verify the effective model from persisted session or turn metadata whenever the event stream does not establish it.' \
  docs/spec-first-workflow/phases/implementation-validation-closeout.md \
  'Worker intake must verify the effective model from authoritative metadata when needed'
require_text \
  'explicitly selects `gpt-5.6-terra` unless inspection discovers a concrete Terra-disqualifying fact from the canonical criteria' \
  docs/spec-first-workflow-evals.md \
  'E02 must expect Terra for the clear local reversible case'
if grep -Fq -- 'Use it for every implementation Worker; model selection is not task-specific.' \
  docs/spec-first-workflow/phases/implementation-validation-closeout.md; then
  echo 'workflow instruction check failed: Worker instructions still blanket-pin Sol'
  exit 1
fi
if grep -Eiq -- '(always (use|choose|select)|use for every|required for (all|every))[^.]*gpt-5\.6-sol|gpt-5\.6-sol[^.]*(always|required for (all|every)|use for every)' \
  docs/spec-first-workflow/phases/implementation-validation-closeout.md; then
  echo 'workflow instruction check failed: Worker instructions blanket-route outcomes to Sol'
  exit 1
fi
if grep -E -- '^  --model ' docs/spec-first-workflow/phases/implementation-validation-closeout.md \
  | grep -Fvx -- '  --model "$WORKER_MODEL" \' >/dev/null; then
  echo 'workflow instruction check failed: Worker command inherits, floats, or hard-codes a model'
  exit 1
fi
require_text \
  'Do not use `--ephemeral` or `--skip-git-repo-check`.' \
  docs/spec-first-workflow/phases/implementation-validation-closeout.md \
  'task-bearing Worker sessions must remain resumable and Git-bound'
require_text \
  'must not create, continue, or complete a Goal; delegate; self-accept; update task or workflow status; start another task; or claim repository completion' \
  docs/spec-first-workflow/phases/implementation-validation-closeout.md \
  'the Worker brief must preserve root-only authority'
require_text \
  'completion covers the full affected deployment graph' \
  AGENTS.md \
  'repository authority must bind system completion to the affected deployment graph'
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
  'worker and Postgres placement may be in different regions' \
  docs/spec-first-workflow-evals.md \
  'the rollout eval must exercise deployment placement and network risk'
require_text \
  'update it, prove mixed-version compatibility from current evidence, or keep system completion blocked under its owner' \
  docs/spec-first-workflow-evals.md \
  'the contract eval must close every affected caller or consumer'
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
  'Implementation launches no built-in subagent lanes.' \
  .agents/skills/codex-goal-prompt-composer/SKILL.md \
  'the Goal composer must keep implementation review root-owned'
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
  'exactly one external `codex exec` Worker in an isolated Git worktree' \
  docs/spec-first-workflow-evals.md \
  'the direct-work eval must require an external CLI Worker'
require_text \
  'explicit non-inherited reasoning effort for the workload without assuming the highest setting' \
  docs/spec-first-workflow-evals.md \
  'the direct-work eval must exercise explicit workload-selected reasoning effort'
require_text \
  'resumes the same session in the same worktree with concrete gaps' \
  docs/spec-first-workflow-evals.md \
  'the ledger correction eval must resume the owning Worker session'
require_text \
  'adds only top-level `--add-dir "$TASK_WRITABLE_PATH"` before `exec`, preserving `workspace-write` and approval `never`' \
  docs/spec-first-workflow-evals.md \
  'the ledger correction eval must recover one read-only task path without sandbox escalation'
require_text \
  '`spawn_agent(agent_type="worker")` or any built-in subagent is used for implementation, acceptance, review, specialist analysis, re-review, or repair' \
  docs/spec-first-workflow-evals.md \
  'the behavior evals must reject every built-in implementation lane'
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
  '## Interaction And Data Flow' \
  docs/spec-first-workflow/phases/system-integration-design.md \
  'system design must own end-to-end request, event, and data-flow decisions'
require_text \
  'Interaction design is complete when a fresh reviewer can trace every material flow' \
  docs/spec-first-workflow/phases/system-integration-design.md \
  'system design must define an outcome-first interaction completion bar'
require_text \
  'A flow is material when its ordering, ownership, contract, authority, failure behavior, or finality can change implementation, rollout, or proof.' \
  docs/spec-first-workflow/phases/system-integration-design.md \
  'system design must bound material flows by implementation, rollout, or proof impact'
require_text \
  'broker destination and any material routing/partition/ordering key' \
  docs/spec-first-workflow/phases/system-integration-design.md \
  'event flow design must identify its contract, destination, and routing semantics'
require_text \
  'add a Mermaid diagram only when compact text is insufficient for a reviewer to validate' \
  docs/spec-first-workflow/phases/system-integration-design.md \
  'system design must route diagrams by review need rather than semantic shortcuts'
require_text \
  'derive the smallest affected deployment graph and its integrated release proof from the documented material flows' \
  docs/spec-first-workflow/phases/system-integration-design.md \
  'system design must derive release closure from documented material flows'
require_text \
  'a producer-only green build is not contract closure' \
  docs/spec-first-workflow/phases/system-integration-design.md \
  'system release closure must cover affected producers and consumers'
require_text \
  'An unverified cross-region or otherwise remote latency-sensitive path is a blocker' \
  docs/spec-first-workflow/phases/system-integration-design.md \
  'system release closure must reject unproven deployment placement and latency'
require_text \
  "provider deployment status or one component's health is insufficient" \
  docs/spec-first-workflow/phases/system-integration-design.md \
  'system release closure must require integrated proof rather than platform status alone'
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
  'Can a fresh reviewer trace every material flow from its actor or trigger to caller-visible completion or durable finality' \
  docs/spec-first-workflow/phases/technical-design-review.md \
  'technical-design review must falsify interaction and data-flow completeness'
require_text \
  'Where compact text is insufficient, does the smallest useful diagram clarify ordering, ownership, fan-out, recovery, or transformation' \
  docs/spec-first-workflow/phases/technical-design-review.md \
  'technical-design review must falsify diagram usefulness and agreement'
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
  'Include one compact Mermaid `sequenceDiagram` because the callback, status-lookup, reconciliation, and finality branches cannot be reliably validated from compact text alone.' \
  docs/spec-first-workflow-evals.md \
  'system mechanism selection must exercise a useful end-to-end flow diagram'
require_text \
  '### E34 — Proportional System Design' \
  docs/spec-first-workflow-evals.md \
  'the proportional system-design behavior case is missing'
require_text \
  'A passing response does not create another artifact or diagram or invoke architecture/fan-out work.' \
  docs/spec-first-workflow-evals.md \
  'proportional persisted system design must avoid untriggered diagram ceremony'
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
  'unrelated helper, and proof-surface regressions to their owning external Worker sessions for bounded repair' \
  docs/spec-first-workflow-evals.md \
  'the implementation closeout eval must return unrelated implementation work to its owning Worker'
require_text \
  'Reject green obtained by weakening or removing an oracle or bypassing a triggered gate' \
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
  'Quality bar: produce the smallest complete change that a maintainer can understand from the code and tests.' \
  .agents/skills/go-coder/SKILL.md \
  'go-coder must preserve the maintainer-readable quality bar'
require_text \
  'Avoid speculative abstractions, hidden coupling, duplicated policy, and comments that restate the code.' \
  .agents/skills/go-coder/SKILL.md \
  'go-coder must reject speculative and obscuring implementation choices'
require_text \
  'Tests must prove observable behavior and material failure paths.' \
  .agents/skills/go-coder/SKILL.md \
  'go-coder must require behavior-level proof of material failures'
require_text \
  'Before returning, remove replaced code and adjacent stale tests/config/docs, check the local diff for defects and scope drift across errors/context/resources, concurrency, generated drift, cleanup, and unapproved decisions, and report any trade-off or proof gap.' \
  .agents/skills/go-coder/SKILL.md \
  'go-coder must require cleanup, local defect and scope-drift checking, and honest closeout reporting'
require_text \
  'This check is task-local implementation feedback, not acceptance. Return the exact diff, criteria traceability, commands and raw results, and blockers to the root.' \
  .agents/skills/go-coder/SKILL.md \
  'go-coder must return deterministic feedback and traceability without self-acceptance'
require_text \
  'Worker checking is task-local deterministic implementation feedback, not acceptance: the Worker runs relevant automated checks, including behavior tests where relevant, and reports commands and raw results; its criteria mapping is traceability only.' \
  docs/spec-first-workflow/phases/implementation-validation-closeout.md \
  'Worker checking must remain deterministic task-local feedback rather than acceptance'
require_text \
  'The root independently judges business completeness, code quality, test adequacy, scope, and final acceptance.' \
  docs/spec-first-workflow/phases/implementation-validation-closeout.md \
  'the root alone must own qualitative judgment and final acceptance'
require_text \
  'The structured return contains status, summary, changed files, criteria traceability, commands/results, and blockers.' \
  docs/spec-first-workflow/phases/implementation-validation-closeout.md \
  'the Worker structured return must use criteria traceability'
require_text \
  'fix the narrowest owning surface whose contract the reproducer proves is violated' \
  docs/spec-first-workflow/phases/implementation-validation-closeout.md \
  'implementation must fix the narrowest owning contract surface rather than one reported entry point'
require_text \
  'Treat edits to tests, fixtures, golden files, skip or exclusion settings, lint/build configuration, and proof commands as proof-surface changes.' \
  docs/spec-first-workflow/phases/implementation-validation-closeout.md \
  'implementation review must treat proof-surface mutations as part of the candidate diff'
require_text \
  'integrated target-environment proof across the affected deployment graph' \
  docs/spec-first-workflow/phases/implementation-validation-closeout.md \
  'implementation closeout must require integrated proof for system-wide outcomes'
require_text \
  'an accidental cross-region path is blocking evidence' \
  .agents/skills/go-devops-spec/references/railway-release-runtime-policy.md \
  'Railway delivery guidance must verify placement and latency instead of assuming proximity'
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
  'resumes the external Worker that owns the direct outcome or task for repair' \
  .agents/skills/go-verification-before-completion/SKILL.md \
  'verification repair routing must preserve external Worker ownership'
if grep -Fq -- 'In standalone use this ends' \
  .agents/skills/go-verification-before-completion/SKILL.md; then
  echo 'workflow instruction check failed: verification stop boundary is duplicated'
  exit 1
fi
require_text \
  'After acceptance, the root records task evidence and launches a fresh Worker/session for the next ready task.' \
  docs/spec-first-workflow/phases/implementation-validation-closeout.md \
  'ledger progress must be durable after root task acceptance'
require_text \
  'Implementation review is always root-owned: matching review skills are applied locally and no built-in reviewer or specialist lane is launched.' \
  docs/spec-first-workflow.md \
  'the direct path must keep implementation review root-owned'
require_text \
  'Implementation/validation/closeout never launches a built-in subagent' \
  docs/spec-first-workflow/shared/subagents-and-handoff.md \
  'the shared review contract must forbid implementation subagent lanes'
require_text \
  'Never launch a built-in subagent, reviewer, specialist, or re-review lane anywhere inside implementation/validation/closeout.' \
  docs/spec-first-workflow/phases/implementation-validation-closeout.md \
  'implementation closeout must forbid every built-in review lane'
require_text \
  'unrequested workflow/review ceremony' \
  docs/spec-first-workflow-evals.md \
  'the small direct eval must reject review ceremony'
require_text \
  'inspects the returned diff and proof' \
  docs/spec-first-workflow-evals.md \
  'the small direct eval must inspect the final diff before closeout'
require_text \
  'Validation, in-scope Worker repair, root re-inspection, revalidation, and closeout run automatically' \
  docs/spec-first-workflow/phases/implementation-validation-closeout.md \
  'implementation closeout must keep correction review with the root'
require_text \
  'Inspect current workspace and Git status' \
  docs/spec-first-workflow/shared/artifact-model.md \
  'resume must inspect current workspace state before trusting ledger progress'
require_text \
  'rerun the smallest ledger proof that can detect workspace drift affecting the next unchecked task' \
  docs/spec-first-workflow/shared/artifact-model.md \
  'resume must refresh the narrow proof that detects drift for the next task'
require_text \
  'return every in-scope implementation-owned finding to its owning external Worker, revalidate, and have the root re-inspect the revised diff and affected lenses' \
  docs/spec-first-workflow-evals.md \
  'the honest-blocker eval must preserve Worker repair and root re-inspection before external handoff'
require_text \
  'was available at readiness becomes unavailable only after an external provider-state change' \
  docs/spec-first-workflow-evals.md \
  'the honest-blocker eval must not normalize a known pre-implementation gate'
require_text \
  'Implementation-owned gaps return to their Worker session and cannot be relabeled as `blocked` or handed to the user.' \
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
  'When a non-implementation independent gate is required, one whole-artifact reviewer is the default.' \
  docs/spec-first-workflow/shared/subagents-and-handoff.md \
  'non-implementation independent review must default to one coherence reviewer'
require_text \
  'Repeat only while a concrete new finding or semantic repair changes readiness.' \
  docs/spec-first-workflow/shared/subagents-and-handoff.md \
  'review convergence must continue only for concrete readiness-changing work'
require_text \
  'External CLI Workers are not subagents and follow the [implementation phase contract]' \
  docs/spec-first-workflow/shared/subagents-and-handoff.md \
  'subagent limits must not govern the external implementation Worker'
require_text \
  'applies compatible matching methods locally in one coherence pass' \
  docs/spec-first-workflow/shared/subagents-and-handoff.md \
  'the non-implementation reviewer must cover compatible lenses in one pass'
require_text \
  'A non-implementation macro phase reaches review convergence only when' \
  docs/spec-first-workflow/shared/subagents-and-handoff.md \
  'the non-implementation review-convergence exit condition is missing'
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
  'Required non-implementation reviews need a fresh `PASS`' \
  README.md \
  'the user-facing workflow summary must expose PASS-only non-implementation movement'

for concerns_phase in \
  docs/spec-first-workflow/phases/research.md \
  docs/spec-first-workflow/phases/specification-review.md \
  docs/spec-first-workflow/phases/technical-design-review.md \
  docs/spec-first-workflow/phases/test-design.md \
  docs/spec-first-workflow/phases/task-review-readiness.md; do
  require_text \
    'disposition and fresh review' \
    "${concerns_phase}" \
    'every non-implementation phase-local CONCERNS verdict must remain non-terminal'
done

for pass_only_phase in \
  docs/spec-first-workflow/phases/research.md \
  docs/spec-first-workflow/phases/specification.md \
  docs/spec-first-workflow/phases/system-integration-design.md \
  docs/spec-first-workflow/phases/go-code-ownership-design.md \
  docs/spec-first-workflow/phases/test-design.md \
  docs/spec-first-workflow/phases/planning.md; do
  require_text \
    'returned `PASS`' \
    "${pass_only_phase}" \
    'every non-implementation macro-phase movement rule must require PASS'
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
  'Ledger work assigns exactly one ready task to one Worker' \
  docs/spec-first-workflow/phases/implementation-validation-closeout.md \
  'ledger implementation must assign one task per Worker'
require_text \
  'launches a fresh Worker/session for the next ready task' \
  docs/spec-first-workflow/phases/implementation-validation-closeout.md \
  'each accepted task must be followed by a fresh Worker for the next task'
require_text \
  'resumes the same Worker session in the same worktree with concrete bounded gaps' \
  docs/spec-first-workflow/phases/implementation-validation-closeout.md \
  'the root must resume incomplete work with its owning Worker session'
require_text \
  'the root resumes the same Worker session in the same worktree with concrete bounded gaps and does not author the repair' \
  docs/spec-first-workflow/phases/implementation-validation-closeout.md \
  'the root must not bypass Worker ownership or acceptance order'
require_text \
  'Every ledger task receives root acceptance review before the next task starts.' \
  docs/spec-first-workflow/phases/implementation-validation-closeout.md \
  'every Worker task must receive root acceptance before advancement'
require_text \
  'execute implementation through the external CLI Worker contract, one direct outcome or one ledger task at a time under root acceptance' \
  docs/spec-first-workflow.md \
  'the top-level router must use the external Worker contract'
require_text \
  'the root reviews every Worker result and the final integrated diff, evaluates all affected lenses locally' \
  docs/spec-first-workflow.md \
  'the top-level router must keep implementation review root-owned'
require_text \
  'Return every implementation-owned finding to the Worker session that owns the affected direct outcome or task.' \
  docs/spec-first-workflow/phases/implementation-validation-closeout.md \
  'review findings must preserve external Worker ownership'
require_text \
  'Every review return also names the exact revision or diff' \
  docs/subagent-contract.md \
  'the shared reviewer envelope must preserve revision and affected-lens coverage'
require_text \
  'assigns exactly one ready task to one external `codex exec` Worker/session' \
  docs/spec-first-workflow-evals.md \
  'the Worker-loop eval must enforce one task per Worker'
require_text \
  'resumes the same session in the same worktree with concrete gaps' \
  docs/spec-first-workflow-evals.md \
  'the Worker-loop eval must preserve session ownership across correction'
require_text \
  'the T07 session is replaced for correction or reused for T08' \
  docs/spec-first-workflow-evals.md \
  'the Worker-loop eval must reject session replacement across correction and reuse across tasks'
require_text \
  'Compatible lenses fit one coherence review' \
  docs/spec-first-workflow-evals.md \
  'the non-implementation delegation eval must keep compatible lenses in one reviewer'
require_text \
  'run only the one bounded security specialist' \
  docs/spec-first-workflow-evals.md \
  'the non-implementation delegation eval must reject speculative specialist waves'
require_text \
  'any built-in subagent is used for implementation, acceptance, review, specialist analysis, re-review, or repair' \
  docs/spec-first-workflow-evals.md \
  'the worker-loop eval must reject all implementation subagent lanes'
require_text \
  'has one concrete high-impact security question at the technical-design gate' \
  docs/spec-first-workflow-evals.md \
  'specialist routing must identify the concrete question at a non-implementation gate'
require_text \
  'runs one bounded security specialist before the non-implementation technical-design gate' \
  docs/spec-first-workflow-evals.md \
  'specialist routing must fan in before one whole-artifact review'
require_text \
  'The root applies matching review skills and all affected specialist lenses locally, re-inspects Worker corrections' \
  docs/spec-first-workflow-evals.md \
  'implementation correction review must stay with the root'
require_text \
  'it never permits phase movement' \
  docs/spec-first-workflow/shared/subagents-and-handoff.md \
  'non-implementation CONCERNS must remain non-terminal even without a current-phase defect'
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
  'Own only that outcome or task until the root accepts it or resumes this session with concrete gaps' \
  .agents/skills/go-coder/SKILL.md \
  'the implementation Worker must retain one outcome or task through corrections'
require_text \
  'do not create or change a Goal, update workflow status, self-accept, start another task, delegate, or claim repository completion' \
  .agents/skills/go-coder/SKILL.md \
  'the coder must preserve root-only authority'
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
  'resume the same Worker session with concrete gaps and do not author the repair' \
  .agents/skills/validation-closeout-session/SKILL.md \
  'validation closeout must return implementation repair to the owning external Worker'
require_text \
  'Apply matching review skills and affected specialist lenses locally; do not launch a built-in subagent lane.' \
  .agents/skills/validation-closeout-session/SKILL.md \
  'validation closeout must keep implementation review methods and lenses with the root'
require_text \
  'The same authorized root session may continue into that boundary, but the review never becomes an implementation acceptance or closeout gate.' \
  .agents/skills/validation-closeout-session/SKILL.md \
  'validation closeout must allow same-session standalone review without making it an implementation gate'
require_text \
  'go-structural-quality-review' \
  .codex/agents/quality-agent.toml \
  'the quality reviewer must expose structural-quality review'
require_text \
  'External implementation Workers are separate `codex exec` processes' \
  docs/subagent-contract.md \
  'the runtime lane contract must distinguish CLI Workers from subagents'
require_text \
  'Every implementation is produced by an external `codex exec` Worker in an isolated Git worktree' \
  README.md \
  'the user-facing workflow summary must describe external Worker execution'
require_text \
  'one Worker for direct work, or one fresh Worker per ledger task with same-session correction until root acceptance' \
  README.md \
  'the user-facing workflow summary must cover direct and ledger Worker ownership'

root_local_implementation="$(grep -RInE -- \
  'The root may implement direct work locally|Implementation may be local or delegated|root repairs only direct work' \
  AGENTS.md README.md docs/spec-first-workflow.md docs/spec-first-workflow .agents/skills || true)"
if [[ -n "${root_local_implementation}" ]]; then
  echo 'workflow instruction check failed: root-local implementation wording remains'
  printf '%s\n' "${root_local_implementation}" | sed 's/^/  /'
  exit 1
fi

implementation_subagent_review="$(grep -RInE -- \
  'risk-triggered independent review|add independent final-diff review|independent final-diff review runs|Independent whole-diff review|same gate reviewer when independent review was triggered|required post-code review|one fresh final-diff review to|focused fresh review only for invalidated lenses|runs one bounded security specialist before the implementation gate|one read-only whole-diff reviewer|concretely triggered independent review' \
  AGENTS.md README.md docs/spec-first-workflow.md docs/spec-first-workflow docs/spec-first-workflow-evals.md .agents/skills || true)"
if [[ -n "${implementation_subagent_review}" ]]; then
  echo 'workflow instruction check failed: built-in implementation review lane wording remains'
  printf '%s\n' "${implementation_subagent_review}" | sed 's/^/  /'
  exit 1
fi

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
