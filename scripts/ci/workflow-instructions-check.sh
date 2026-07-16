#!/usr/bin/env bash
set -euo pipefail

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
  docs/subagent-brief-template.md
)

skill_files=(
  .agents/skills/*session/SKILL.md
  .agents/skills/planning-and-task-breakdown/SKILL.md
  .agents/skills/spec-document-designer/SKILL.md
  .agents/skills/spec-clarification-challenge/SKILL.md
  .agents/skills/specification-review/SKILL.md
  .agents/skills/go-implementation-ownership-spec/SKILL.md
  .agents/skills/go-coder/SKILL.md
  .agents/skills/grilling/SKILL.md
  .agents/skills/codex-goal-prompt-composer/SKILL.md
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

t02_error() {
  printf 'workflow instruction check failed: T02 %s\n' "$*" >&2
  return 1
}

t02_require_files() {
  local file
  for file in "$@"; do
    [[ -f "${file}" ]] || t02_error "canonical owner is missing: ${file}" || return 1
  done
}

t02_require_absent_paths() {
  local file
  for file in "$@"; do
    [[ ! -e "${file}" ]] || t02_error "retired CLI Worker artifact exists: ${file}" || return 1
  done
}

t02_require_markdown_target() {
  local file="$1"
  local target="$2"
  if ! grep -Fq -- "](${target})" "${file}"; then
    t02_error "canonical link is missing: ${file} -> ${target}"
    return 1
  fi
}

t02_require_heading() {
  local file="$1"
  local heading="$2"
  if ! grep -Fqx -- "${heading}" "${file}"; then
    t02_error "structural heading is missing: ${file} -> ${heading}"
    return 1
  fi
}

t02_require_machine_token() {
  local file="$1"
  local token="$2"
  if ! grep -Fq -- "${token}" "${file}"; then
    t02_error "stable machine token is missing: ${file} -> ${token}"
    return 1
  fi
}

t02_check_no_semantic_exact_text_api() {
  local checker="$1"
  if grep -En -- '(^|[^[:alnum:]_])require_[t]ext([^[:alnum:]_]|$)' "${checker}" >/dev/null; then
    t02_error "generic semantic exact-text API found in ${checker}"
    return 1
  fi
}

t02_check_no_cli_worker_fallback() {
  local file="$1"
  local cli_pattern
  cli_pattern='codex[[:space:]]+exec|Codex CLI Worker|external CLI Worker|CLI fallback|worker-session\.sh|worker-result\.schema\.json|worker-session-check\.sh|--add-dir|workspace-write'
  if grep -Ein -- "${cli_pattern}" "${file}" >/dev/null; then
    t02_error "CLI Worker fallback found in ${file}"
    return 1
  fi
}

t02_check_no_unsupported_app_controls() {
  local file="$1"
  local pattern
  pattern='(launch|creation|dispatch).*(permissions|approval reviewer|provider|service tier|callback URL|CPU|RAM|timeout|max turns)|(permissions|approval reviewer|provider|service tier|callback URL|CPU|RAM|timeout|max turns).*(launch|creation|dispatch)'
  if grep -Eiq -- "${pattern}" "${file}"; then
    t02_error "unsupported App task-creation control claimed in ${file}"
    return 1
  fi
}

t02_check_app_worker_model_policy() {
  local file="$1"
  local token
  for token in \
    'explicitly selects and passes both the model and reasoning effort' \
    'never inherit an App default' \
    'task identity, selected model and effort, and a short basis' \
    'basis names the exact eval artifact and compared model/effort configuration' \
    'Choose model and effort independently from the App' \
    'task difficulty, ambiguity or evidence volume, latency/cost, reversibility, and consequence of error' \
    '[official Codex model guide](https://learn.chatgpt.com/docs/models#choosing-sol-terra-and-luna)' \
    '`gpt-5.6-sol` for complex, open-ended, ambiguous, difficult, or high-value coding, research, or security work' \
    '`gpt-5.6-terra` for pragmatic everyday work needing strong reasoning and tool use' \
    '`gpt-5.6-luna` for clear, specific, repeatable, high-volume work with a known result' \
    'These are task-specific guides, not mandatory keyword routing or a Terra default' \
    'Use the lowest effort likely to produce the required result' \
    '`low` for quick, well-scoped work' \
    '`medium` for more planning and a speed/depth balance' \
    '`high` or `xhigh` for difficult work with multiple steps, sources, or tradeoffs' \
    '`max` for the hardest single quality-first task' \
    '`Ultra` is subagent parallelism, not more single-task reasoning' \
    'this Worker cannot delegate, so do not route this single-Worker phase to Ultra' \
    'There is no fixed model/effort baseline' \
    'risk signals, not automatic Sol or highest-effort triggers' \
    'Evals may inform the basis when already available, but are never a prerequisite' \
    'do not pause dispatch or create/run an eval solely to justify a model or effort choice' \
    'Same-task model and effort may change when remaining work or observed Worker evidence justifies it, without an eval prerequisite'; do
    t02_require_machine_token "${file}" "${token}" || return 1
  done
  if grep -Eiq -- 'Omit model and reasoning-effort overrides|model or effort pinning without the user' "${file}"; then
    t02_error "default-inherited App Worker selection found in ${file}"
    return 1
  fi
}

t02_check_challenge_owner() {
  local file="$1"
  local heading token event count
  [[ -f "${file}" ]] || { t02_error "challenge owner is missing: ${file}"; return 1; }

  for heading in \
    '# Autonomous Pre-Review Challenge' \
    '## Protocol' \
    '## Authority' \
    '## State And Continuation' \
    '## Exhaustion And Invalidation' \
    '## Reviewer Separation'; do
    t02_require_heading "${file}" "${heading}" || return 1
  done

  for event in QUESTION HUMAN_REQUIRED REOPEN DONE; do
    count="$(grep -Fxc -- "${event}" "${file}" || true)"
    if [[ "${count}" -ne 1 ]]; then
      t02_error "challenge event shape ${event} must occur exactly once in ${file}; found ${count}"
      return 1
    fi
  done

  for token in ACCEPT OVERRIDE RECLASSIFY CONTINUE_INDEPENDENT WAIT_HUMAN REOPEN_OWNER; do
    if ! grep -Fq -- "\`${token}\`" "${file}"; then
      t02_error "challenge machine token is missing from ${file}: ${token}"
      return 1
    fi
  done
}

t02_check_single_challenge_owner() {
  local directory="$1"
  local count
  count="$(find "${directory}" -maxdepth 1 -type f -iname '*autonomous*challenge*.md' -print | wc -l | tr -d '[:space:]')"
  if [[ "${count}" -ne 1 ]]; then
    t02_error "expected one focused autonomous-challenge owner in ${directory}; found ${count}"
    return 1
  fi
}

t02_check_challenge_compatibility_route() {
  local file="$1"
  local section
  section="$(awk '
    $0 == "## Autonomous Pre-Review Challenge" { active = 1; next }
    active && /^## / { exit }
    active { print }
  ' "${file}")"
  if [[ -z "${section}" ]]; then
    t02_error "compatibility challenge section is empty in ${file}"
    return 1
  fi
  if ! printf '%s\n' "${section}" | grep -Fq -- '](autonomous-pre-review-challenge.md)'; then
    t02_error "compatibility challenge route is missing in ${file}"
    return 1
  fi
  if printf '%s\n' "${section}" | grep -Eq -- '^(```text|QUESTION|HUMAN_REQUIRED|REOPEN|DONE)$|`(ACCEPT|OVERRIDE|RECLASSIFY|CONTINUE_INDEPENDENT|WAIT_HUMAN|REOPEN_OWNER)`'; then
    t02_error "stale competing challenge protocol found in ${file}"
    return 1
  fi
}

t02_check_no_challenge_protocol_copy() {
  local file="$1"
  if grep -Eq -- '^(QUESTION|HUMAN_REQUIRED|REOPEN|DONE)$' "${file}"; then
    t02_error "challenge event shapes copied outside the focused owner: ${file}"
    return 1
  fi
  if grep -Eq -- '`ACCEPT`.*`OVERRIDE`.*`RECLASSIFY`|`CONTINUE_INDEPENDENT`.*`WAIT_HUMAN`.*`REOPEN_OWNER`' "${file}"; then
    t02_error "challenge continuation protocol copied outside the focused owner: ${file}"
    return 1
  fi
}

t02_check_no_competing_challenge_protocol() {
  local directory="$1"
  local owner="$2"
  local file
  for file in "${directory}"/*.md; do
    [[ "${file}" == "${owner}" ]] && continue
    t02_check_no_challenge_protocol_copy "${file}" || return 1
  done
}

t02_check_forbidden_legacy() {
  local file="$1"
  local legacy_pattern
  legacy_pattern='The root may implement direct work locally|Implementation may be local or delegated|root repairs only direct work|Use it for every implementation Worker; model selection is not task-specific\.|worker delegation is mandatory|implementation (may|can|should) (use|launch) (a )?built-in (subagent|worker)|root (may|can|should) (author|repair|implement) Worker-owned'
  if grep -Eiq -- "${legacy_pattern}" "${file}"; then
    t02_error "forbidden legacy behavior found in ${file}"
    return 1
  fi
}

t02_expect_fixture_failure() {
  local name="$1"
  shift
  if "$@" >/dev/null 2>&1; then
    t02_error "mutation fixture unexpectedly passed: ${name}"
    return 1
  fi
}

t02_run_mutation_fixtures() {
  local fixture_root owner safe_owner consumer phase shared eval skill agent compatibility legacy checker_fixture owner_dir cli_artifact forbidden_helper
  fixture_root="$(mktemp -d -t workflow-instructions-t02.XXXXXX)"
  owner="${fixture_root}/owner.md"
  safe_owner="${fixture_root}/safe-owner.md"
  consumer="${fixture_root}/consumer.md"
  phase="${fixture_root}/phase.md"
  shared="${fixture_root}/shared.md"
  eval="${fixture_root}/eval.md"
  skill="${fixture_root}/skill.md"
  agent="${fixture_root}/agent.toml"
  compatibility="${fixture_root}/compatibility.md"
  legacy="${fixture_root}/legacy.md"
  checker_fixture="${fixture_root}/checker.sh"
  owner_dir="${fixture_root}/shared"
  mkdir -p "${owner_dir}"

  cat >"${owner}" <<'EOF'
# Autonomous Pre-Review Challenge
## Protocol
QUESTION
HUMAN_REQUIRED
REOPEN
DONE
## Authority
`ACCEPT` `OVERRIDE` `RECLASSIFY`
## State And Continuation
`CONTINUE_INDEPENDENT` `WAIT_HUMAN` `REOPEN_OWNER`
## Exhaustion And Invalidation
Fixture wording A.
## Reviewer Separation
Fixture wording B.
EOF
  cat >"${safe_owner}" <<'EOF'
# Autonomous Pre-Review Challenge
Safe canonical prose can be rewritten completely.
## Protocol
QUESTION
HUMAN_REQUIRED
REOPEN
DONE
## Authority
Different prose around `ACCEPT`, `OVERRIDE`, and `RECLASSIFY`.
## State And Continuation
Different prose around `CONTINUE_INDEPENDENT`, `WAIT_HUMAN`, and `REOPEN_OWNER`.
## Exhaustion And Invalidation
No sentence from the repository owner is required here.
## Reviewer Separation
Only structural ownership and machine identifiers remain stable.
EOF
  printf '%s\n' 'Safely reworded consumer consequence. [Owner](owner.md)' >"${consumer}"
  cat >"${phase}" <<'EOF'
## Stop Rule
Phase prose variant A around `E43`.
EOF
  printf '%s\n' 'Shared prose variant A. [Owner](owner.md)' >"${shared}"
  printf '%s\n' 'Eval prose variant A around `E43`.' >"${eval}"
  cat >"${skill}" <<'EOF'
## Proof, Return, And Stop
Skill prose variant A.
EOF
  printf '%s\n' 'developer_instructions = "Agent prose variant A with READ_ONLY"' >"${agent}"
  cat >"${compatibility}" <<'EOF'
## Autonomous Pre-Review Challenge
[Owner](autonomous-pre-review-challenge.md)
## Review Independence
EOF
  printf '%s\n' 'Safe current behavior.' >"${legacy}"
  cat >"${checker_fixture}" <<'EOF'
t02_require_heading "${phase}" '## Stop Rule'
t02_require_markdown_target "${shared}" 'owner.md'
t02_require_machine_token "${eval}" '`E43`'
t02_require_heading "${skill}" '## Proof, Return, And Stop'
t02_require_machine_token "${agent}" 'READ_ONLY'
EOF
  cp "${owner}" "${owner_dir}/autonomous-pre-review-challenge.md"

  t02_check_challenge_owner "${owner}" || return 1
  t02_check_challenge_owner "${safe_owner}" || return 1
  t02_require_markdown_target "${consumer}" 'owner.md' || return 1
  t02_require_heading "${phase}" '## Stop Rule' || return 1
  t02_require_machine_token "${phase}" '`E43`' || return 1
  t02_require_markdown_target "${shared}" 'owner.md' || return 1
  t02_require_machine_token "${eval}" '`E43`' || return 1
  t02_require_heading "${skill}" '## Proof, Return, And Stop' || return 1
  t02_require_machine_token "${agent}" 'READ_ONLY' || return 1
  t02_check_no_cli_worker_fallback "${phase}" || return 1
  t02_check_no_unsupported_app_controls "${phase}" || return 1
  t02_check_challenge_compatibility_route "${compatibility}" || return 1
  t02_check_forbidden_legacy "${legacy}" || return 1
  t02_check_no_semantic_exact_text_api "${checker_fixture}" || return 1
  t02_check_single_challenge_owner "${owner_dir}" || return 1
  t02_check_no_competing_challenge_protocol \
    "${owner_dir}" "${owner_dir}/autonomous-pre-review-challenge.md" || return 1

  cat >"${phase}" <<'EOF'
## Stop Rule
Completely different phase prose around `E43`.
EOF
  printf '%s\n' 'Completely different shared prose. [Owner](owner.md)' >"${shared}"
  printf '%s\n' 'Completely different eval prose around `E43`.' >"${eval}"
  cat >"${skill}" <<'EOF'
## Proof, Return, And Stop
Completely different skill prose.
EOF
  printf '%s\n' 'developer_instructions = "Completely different agent prose with READ_ONLY"' >"${agent}"
  t02_require_heading "${phase}" '## Stop Rule' || return 1
  t02_require_machine_token "${phase}" '`E43`' || return 1
  t02_require_markdown_target "${shared}" 'owner.md' || return 1
  t02_require_machine_token "${eval}" '`E43`' || return 1
  t02_require_heading "${skill}" '## Proof, Return, And Stop' || return 1
  t02_require_machine_token "${agent}" 'READ_ONLY' || return 1

  printf '%s\n' 'Phase prose without its stable heading.' >"${phase}"
  t02_expect_fixture_failure 'phase structure removal' t02_require_heading "${phase}" '## Stop Rule' || return 1
  printf '%s\n' 'Shared prose without its owner route.' >"${shared}"
  t02_expect_fixture_failure 'shared owner-link removal' t02_require_markdown_target "${shared}" 'owner.md' || return 1
  printf '%s\n' 'Eval prose without its stable case ID.' >"${eval}"
  t02_expect_fixture_failure 'eval stable-ID removal' t02_require_machine_token "${eval}" '`E43`' || return 1
  printf '%s\n' 'Skill prose without its stable section.' >"${skill}"
  t02_expect_fixture_failure 'skill structure removal' t02_require_heading "${skill}" '## Proof, Return, And Stop' || return 1
  printf '%s\n' 'developer_instructions = "Agent prose without its authority token"' >"${agent}"
  t02_expect_fixture_failure 'agent authority-token removal' t02_require_machine_token "${agent}" 'READ_ONLY' || return 1

  t02_expect_fixture_failure 'missing owner' t02_check_challenge_owner "${fixture_root}/missing.md" || return 1
  printf '%s\n' 'Consumer without its owner link.' >"${consumer}"
  t02_expect_fixture_failure 'missing link' t02_require_markdown_target "${consumer}" 'owner.md' || return 1
  printf '%s\n' 'Fallback to codex exec if the App task cannot start.' >"${phase}"
  t02_expect_fixture_failure 'restored CLI Worker fallback' t02_check_no_cli_worker_fallback "${phase}" || return 1
  printf '%s\n' 'Configure App task launch permissions and max turns.' >"${phase}"
  t02_expect_fixture_failure 'invented App launch controls' t02_check_no_unsupported_app_controls "${phase}" || return 1
  cli_artifact="${fixture_root}/worker-session.sh"
  : >"${cli_artifact}"
  t02_expect_fixture_failure 'restored CLI Worker artifact' t02_require_absent_paths "${cli_artifact}" || return 1
  rm "${cli_artifact}"
  cat >"${compatibility}" <<'EOF'
## Autonomous Pre-Review Challenge
[Owner](autonomous-pre-review-challenge.md)
```text
QUESTION
HUMAN_REQUIRED
REOPEN
DONE
```
## Review Independence
EOF
  t02_expect_fixture_failure 'stale challenge copy' t02_check_challenge_compatibility_route "${compatibility}" || return 1
  cp "${owner}" "${owner_dir}/pre-review-protocol.md"
  t02_expect_fixture_failure \
    'arbitrarily named competing challenge protocol' \
    t02_check_no_competing_challenge_protocol \
    "${owner_dir}" "${owner_dir}/autonomous-pre-review-challenge.md" || return 1
  rm "${owner_dir}/pre-review-protocol.md"
  cp "${owner}" "${owner_dir}/autonomous-pre-review-challenge-copy.md"
  t02_expect_fixture_failure 'duplicate challenge owner' t02_check_single_challenge_owner "${owner_dir}" || return 1
  printf '%s\n' 'Implementation may be local or delegated.' >"${legacy}"
  t02_expect_fixture_failure 'forbidden legacy behavior' t02_check_forbidden_legacy "${legacy}" || return 1
  forbidden_helper='require_'
  forbidden_helper+='text'
  printf '%s() { :; }\n' "${forbidden_helper}" >"${checker_fixture}"
  t02_expect_fixture_failure \
    'restored generic semantic exact-text API' \
    t02_check_no_semantic_exact_text_api "${checker_fixture}" || return 1

  rm -rf "${fixture_root}"
}

t02_require_files \
  docs/spec-first-workflow/shared/autonomous-pre-review-challenge.md \
  .agents/skills/grilling/SKILL.md \
  .agents/skills/go-coder/SKILL.md \
  .agents/skills/codex-goal-prompt-composer/SKILL.md \
  .agents/skills/validation-closeout-session/SKILL.md \
  .codex/agents/challenger-agent.toml

t02_require_absent_paths \
  docs/codex-cli-worker-operations.md \
  scripts/dev/worker-session.sh \
  scripts/dev/worker-result.schema.json \
  scripts/ci/worker-session-check.sh

t02_link_rows=(
  'AGENTS.md|docs/spec-first-workflow/phases/implementation-validation-closeout.md#worker-assignment-and-acceptance'
  'AGENTS.md|docs/spec-first-workflow/shared/autonomous-pre-review-challenge.md'
  'README.md|AGENTS.md#working-contract'
  'README.md|docs/spec-first-workflow/phases/implementation-validation-closeout.md#worker-assignment-and-acceptance'
  'README.md|docs/spec-first-workflow/shared/autonomous-pre-review-challenge.md'
  'docs/spec-first-workflow.md|spec-first-workflow/phases/implementation-validation-closeout.md#worker-assignment-and-acceptance'
  'docs/spec-first-workflow.md|spec-first-workflow/shared/autonomous-pre-review-challenge.md'
  'docs/spec-first-workflow.md|spec-first-workflow/shared/subagents-and-handoff.md#review-independence'
  'docs/spec-first-workflow/phases/implementation-validation-closeout.md|../shared/artifact-model.md#resume-order'
  'docs/spec-first-workflow/shared/subagents-and-handoff.md|autonomous-pre-review-challenge.md'
  'docs/spec-first-workflow/shared/subagents-and-handoff.md|../phases/implementation-validation-closeout.md#worker-assignment-and-acceptance'
  'docs/spec-first-workflow/shared/subagents-and-handoff.md|../../subagent-contract.md#shared-review-finding-envelope'
  'docs/subagent-contract.md|../AGENTS.md#working-contract'
  'docs/subagent-contract.md|spec-first-workflow/phases/implementation-validation-closeout.md#worker-assignment-and-acceptance'
  'docs/subagent-contract.md|spec-first-workflow/shared/autonomous-pre-review-challenge.md'
  '.agents/skills/go-coder/SKILL.md|../../../docs/spec-first-workflow/phases/implementation-validation-closeout.md'
  '.agents/skills/codex-goal-prompt-composer/SKILL.md|../../../docs/spec-first-workflow/phases/implementation-validation-closeout.md'
  '.agents/skills/validation-closeout-session/SKILL.md|../../../docs/spec-first-workflow/phases/implementation-validation-closeout.md'
)
for row in "${t02_link_rows[@]}"; do
  IFS='|' read -r file target <<<"${row}"
  t02_require_markdown_target "${file}" "${target}"
done

t02_check_no_semantic_exact_text_api scripts/ci/workflow-instructions-check.sh

for heading in \
  '## Authorization' \
  '## Working Contract' \
  '## Instruction Ownership'; do
  t02_require_heading AGENTS.md "${heading}"
done
for token in 'Codex Goal' 'Codex App' '`spawn_agent`' '`agent_type="worker"`'; do
  t02_require_machine_token AGENTS.md "${token}"
done
t02_require_machine_token README.md 'native App Worker'

for heading in \
  '## Delegation Decision' \
  '## Autonomous Pre-Review Challenge' \
  '## Review Independence' \
  '## Fan-In' \
  '## Resume' \
  '## Handoff'; do
  t02_require_heading docs/spec-first-workflow/shared/subagents-and-handoff.md "${heading}"
done
for token in '`PASS`' '`CONCERNS`' '`FAIL`' 'Objective:' '`Goal:`'; do
  t02_require_machine_token docs/spec-first-workflow/shared/subagents-and-handoff.md "${token}"
done
t02_require_heading docs/subagent-contract.md '## Shared Review Finding Envelope'

for heading in \
  '### Worker Assignment And Acceptance' \
  '### Worker Brief And Result Intake' \
  '## Review' \
  '## Validate' \
  '## Close Out' \
  '## Stop Rule'; do
  t02_require_heading docs/spec-first-workflow/phases/implementation-validation-closeout.md "${heading}"
done
for token in \
  'Codex-managed Git worktree' \
  'repository project' \
  'managed-worktree environment' \
  'omit the optional starting state' \
  'select an existing branch' \
  'select the working tree only when required accepted changes are uncommitted' \
  'returned App task, thread, and managed-worktree identity' \
  '`turn/started`' \
  '`item/*`' \
  '`turn/completed`' \
  '`thread/status/changed`' \
  'does not actively poll or narrate unchanged state' \
  'future turn' \
  "active turn's effective model or effort" \
  'same App task' \
  'fresh App task' \
  'Goal / context:' \
  'Constraints:' \
  'Evidence:' \
  'Success:' \
  'Output:' \
  'Stop:'; do
  t02_require_machine_token docs/spec-first-workflow/phases/implementation-validation-closeout.md "${token}"
done
t02_check_app_worker_model_policy docs/spec-first-workflow/phases/implementation-validation-closeout.md
for token in \
  '`turn/started`' \
  '`item/*`' \
  '`turn/completed`' \
  '`thread/status/changed`' \
  'does not actively poll or narrate unchanged state' \
  'future turn' \
  "active turn's effective model or effort" \
  '### E45 — Explicit App Worker Model And Effort Routing' \
  '`gpt-5.6-luna` with `low` is a valid T01 choice' \
  '`gpt-5.6-terra` with `medium` is a valid T02 choice' \
  '`gpt-5.6-sol` with `high` is a valid T03 choice' \
  '`high`/`xhigh` difficult multi-step, source, or tradeoff work' \
  '`max` only the hardest single quality-first task' \
  'Do not route these single-Worker tasks to `Ultra`' \
  'Evals may inform a basis if already available, but do not request, create, run, or pause dispatch for one' \
  'Do not claim that this manifest check proves live benchmark superiority'; do
  t02_require_machine_token docs/spec-first-workflow-evals.md "${token}"
done

t02_check_challenge_owner docs/spec-first-workflow/shared/autonomous-pre-review-challenge.md
t02_check_single_challenge_owner docs/spec-first-workflow/shared
t02_check_challenge_compatibility_route docs/spec-first-workflow/shared/subagents-and-handoff.md
t02_check_no_competing_challenge_protocol \
  docs/spec-first-workflow/shared \
  docs/spec-first-workflow/shared/autonomous-pre-review-challenge.md

for file in \
  .agents/skills/grilling/SKILL.md \
  .codex/agents/challenger-agent.toml; do
  t02_check_no_challenge_protocol_copy "${file}"
done

for file in \
  .agents/skills/go-coder/SKILL.md \
  .agents/skills/codex-goal-prompt-composer/SKILL.md \
  .agents/skills/validation-closeout-session/SKILL.md; do
  t02_check_no_cli_worker_fallback "${file}"
  t02_check_forbidden_legacy "${file}"
done

t02_require_heading .agents/skills/go-coder/SKILL.md '# Go Coder'
t02_require_heading .agents/skills/codex-goal-prompt-composer/SKILL.md '# Codex Goal Prompt Composer'
t02_require_heading .agents/skills/validation-closeout-session/SKILL.md '# Validation Closeout Session'

for token in \
  'sandbox_mode = "read-only"' \
  'docs/spec-first-workflow/shared/autonomous-pre-review-challenge.md' \
  '`workflow-plan-adequacy-challenge`' \
  '`pre-spec-challenge`' \
  '`spec-clarification-challenge`'; do
  t02_require_machine_token .codex/agents/challenger-agent.toml "${token}"
done

t02_unprotected_consumers=(
  AGENTS.md
  README.md
  docs/spec-first-workflow.md
  docs/spec-first-workflow/shared/subagents-and-handoff.md
  docs/subagent-contract.md
)
for file in "${t02_unprotected_consumers[@]}"; do
  t02_check_no_challenge_protocol_copy "${file}"
done

t02_worker_policy_files=(
  AGENTS.md
  README.md
  docs/spec-first-workflow.md
  docs/spec-first-workflow/phases/implementation-validation-closeout.md
  docs/spec-first-workflow/shared/subagents-and-handoff.md
  docs/subagent-contract.md
)
for file in "${t02_worker_policy_files[@]}"; do
  t02_check_no_cli_worker_fallback "${file}"
  t02_check_forbidden_legacy "${file}"
done
t02_check_no_cli_worker_fallback docs/spec-first-workflow-evals.md
t02_check_no_unsupported_app_controls docs/spec-first-workflow/phases/implementation-validation-closeout.md

if grep -Eq -- 'read/write|read-only/write' \
  docs/spec-first-workflow/shared/subagents-and-handoff.md docs/subagent-brief-template.md; then
  t02_error 'a built-in subagent brief still permits writes'
fi
t02_run_mutation_fixtures
echo 'workflow instruction check passed: T02 focused owner/link/rewording/duplicate/legacy/no-prose-API fixtures'


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
if grep -Fq -- 'In standalone use this ends' \
  .agents/skills/go-verification-before-completion/SKILL.md; then
  echo 'workflow instruction check failed: verification stop boundary is duplicated'
  exit 1
fi

if grep -Eq -- 'implementation_with_accepted_risks|readiness = WAIVED' \
  .agents/skills/go-domain-invariant-spec/references/state-machine-and-transition-rules.md; then
  echo 'workflow instruction check failed: lifecycle example still bypasses PASS-only readiness'
  exit 1
fi

if grep -Fq -- 'Use only for a ready spec revision' .agents/skills/specification-review/SKILL.md; then
  echo 'workflow instruction check failed: specification review still requires pre-review ready status'
  exit 1
fi

if grep -Eq -- 'proof_only|follow_up_only|no new decision required in' \
  .agents/skills/go-test-design/SKILL.md; then
  echo 'workflow instruction check failed: test strategy uses an ambiguous disposition label'
  exit 1
fi
if grep -Eq -- 'plausible incorrect implementation|Missing test code or fixtures are a valid fail-before signal' \
  docs/spec-first-workflow/phases/test-design.md .agents/skills/go-test-design/SKILL.md; then
  echo 'workflow instruction check failed: test design retains implementation-mirroring or false RED wording'
  exit 1
fi

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

for reference in .agents/skills/go-coder/references/*.md; do
  reference_link="references/$(basename "${reference}")"
  if ! grep -Fq -- "](${reference_link})" .agents/skills/go-coder/SKILL.md; then
    echo "workflow instruction check failed: go-coder reference is not routed"
    echo "  reference: ${reference}"
    exit 1
  fi
done

for reference in .agents/skills/go-test-implementation/references/*.md; do
  reference_link="references/$(basename "${reference}")"
  if ! grep -Fq -- "](${reference_link})" .agents/skills/go-test-implementation/SKILL.md; then
    echo "workflow instruction check failed: go-test-implementation reference is not routed"
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
