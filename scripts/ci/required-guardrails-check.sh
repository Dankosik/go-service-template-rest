#!/usr/bin/env bash
set -euo pipefail

required_files=(
  "AGENTS.md"
  "SOUL.md"
  "README.md"
  "railway.toml"
  "Makefile"
  ".editorconfig"
  ".gitattributes"
  ".golangci.yml"
  ".redocly.yaml"
  ".codex/config.toml"
  ".github/CODEOWNERS"
  ".github/dependabot.yml"
  ".github/pull_request_template.md"
  ".github/workflows/ci.yml"
  ".github/workflows/cd.yml"
  ".github/workflows/nightly.yml"
  "CONTRIBUTING.md"
  "SECURITY.md"
  "LICENSE"
  "env/.env.example"
  "env/docker-compose.yml"
  "build/docker/Dockerfile"
  "build/docker/tooling-images.Dockerfile"
  "docs/subagent-contract.md"
  ".agents/skills/go-coder/references/boundary-decoding-and-validation.md"
  ".agents/skills/api-contract-designer-spec/references/resource-representations-and-lifecycle.md"
  ".agents/skills/api-contract-designer-spec/references/boundary-validation-and-freshness.md"
  "docs/subagent-brief-template.md"
  "docs/spec-first-workflow.md"
  "docs/spec-first-workflow/shared/artifact-model.md"
  "docs/spec-first-workflow/shared/subagents-and-handoff.md"
  "docs/spec-first-workflow/phases/intake.md"
  "docs/spec-first-workflow/phases/research.md"
  "docs/spec-first-workflow/phases/specification.md"
  "docs/spec-first-workflow/phases/specification-review.md"
  "docs/spec-first-workflow/phases/system-integration-design.md"
  "docs/spec-first-workflow/phases/go-code-ownership-design.md"
  "docs/spec-first-workflow/phases/technical-design-review.md"
  "docs/spec-first-workflow/phases/test-design.md"
  "docs/spec-first-workflow/phases/planning.md"
  "docs/spec-first-workflow/phases/task-review-readiness.md"
  "docs/spec-first-workflow/phases/implementation-validation-closeout.md"
  ".agents/skills/grilling/SKILL.md"
  ".agents/skills/grilling/evals/evals.json"
  ".agents/skills/agent-prompt-composer/SKILL.md"
  ".agents/skills/agent-prompt-composer/evals/evals.json"
  ".agents/skills/codex-goal-prompt-composer/SKILL.md"
  ".agents/skills/workflow-planning-session/SKILL.md"
  ".agents/skills/workflow-planning-session/evals/evals.json"
  ".agents/skills/workflow-plan-adequacy-challenge/SKILL.md"
  ".agents/skills/workflow-plan-adequacy-challenge/evals/evals.json"
  ".agents/skills/workflow-status/SKILL.md"
  ".agents/skills/workflow-status/evals/evals.json"
  ".agents/skills/test-design-session/SKILL.md"
  ".agents/skills/test-design-session/evals/evals.json"
  ".agents/skills/specification-review-session/SKILL.md"
  ".codex/agents/challenger-agent.toml"
  "specs/README.md"
  "scripts/ci/workflow-routing-check/main.go"
  "scripts/ci/workflow-routing-check/main_test.go"
  "scripts/ci/workflow-routing-check/testdata/cases.json"
  "scripts/ci/sync-mirror-integration-check.sh"
  "scripts/dev/sync-skills.sh"
  "scripts/dev/sync-agents.sh"
  "scripts/dev/module-origin.sh"
)

missing=()
for file in "${required_files[@]}"; do
  if [[ ! -f "${file}" ]]; then
    missing+=("${file}")
  fi
done

if [[ ${#missing[@]} -gt 0 ]]; then
  echo "required repository guardrails are missing:"
  for file in "${missing[@]}"; do
    echo "- ${file}"
  done
  exit 1
fi

require_regex() {
  local pattern="$1"
  local file="$2"
  local message="$3"
  if ! grep -Eq -- "${pattern}" "${file}"; then
    echo "guardrail check failed: ${message}"
    echo "  file: ${file}"
    echo "  expected pattern: ${pattern}"
    exit 1
  fi
}

require_absent_regex() {
  local pattern="$1"
  local file="$2"
  local message="$3"
  if grep -Eq -- "${pattern}" "${file}"; then
    echo "guardrail check failed: ${message}"
    echo "  file: ${file}"
    echo "  forbidden pattern: ${pattern}"
    exit 1
  fi
}

require_markdown_links_exist() {
  local file="$1"
  local prefix="$2"
  local base_dir="$3"
  local message="$4"
  local count=0
  local missing=()
  local link

  while IFS= read -r link; do
    [[ -z "${link}" ]] && continue
    count=$((count + 1))
    if [[ ! -f "${base_dir}/${link}" ]]; then
      missing+=("${link}")
    fi
  done < <(grep -Eo "\\(${prefix}[^)]*\\.md\\)" "${file}" | tr -d '()' || true)

  if [[ "${count}" -eq 0 ]]; then
    echo "guardrail check failed: ${message}"
    echo "  file: ${file}"
    echo "  expected at least one markdown link with prefix: ${prefix}"
    exit 1
  fi

  if [[ ${#missing[@]} -gt 0 ]]; then
    echo "guardrail check failed: ${message}"
    echo "  file: ${file}"
    echo "  missing linked files:"
    for link in "${missing[@]}"; do
      echo "  - ${base_dir}/${link}"
    done
    exit 1
  fi
}

require_no_forbidden_go_imports() {
  local message="$1"
  local pattern="$2"
  shift 2

  local imports
  imports="$(go list -f '{{range .Imports}}{{printf "%s\t%s\n" $.ImportPath .}}{{end}}{{range .TestImports}}{{printf "%s\t%s\n" $.ImportPath .}}{{end}}{{range .XTestImports}}{{printf "%s\t%s\n" $.ImportPath .}}{{end}}' "$@")"

  local forbidden
  forbidden="$(printf '%s\n' "${imports}" | grep -E -- "${pattern}" || true)"
  if [[ -n "${forbidden}" ]]; then
    echo "guardrail check failed: ${message}"
    printf '%s\n' "${forbidden}" | sed 's/^/  /'
    exit 1
  fi
}

# Keep Railway deployment policy deterministic and repo-reviewable.
require_regex '^builder = "DOCKERFILE"$' "railway.toml" "railway build policy must use DOCKERFILE builder"
require_regex '^dockerfilePath = "build/docker/Dockerfile"$' "railway.toml" "railway build policy must point to build/docker/Dockerfile"
require_regex '^preDeployCommand = \["/migrate"\]$' "railway.toml" "railway pre-deploy command must run the dedicated migration binary"
require_regex '^healthcheckPath = "/health/ready"$' "railway.toml" "railway deploy healthcheck path must be /health/ready"
require_regex '^healthcheckTimeout = 180$' "railway.toml" "railway deploy healthcheck timeout must be 180 seconds"
require_regex '^restartPolicyType = "ON_FAILURE"$' "railway.toml" "railway restart policy type must be ON_FAILURE"
require_regex '^restartPolicyMaxRetries = 5$' "railway.toml" "railway restart retries must be locked to 5"
require_regex '^overlapSeconds = 45$' "railway.toml" "railway overlap window must be 45 seconds"
require_regex '^drainingSeconds = 30$' "railway.toml" "railway draining window must be 30 seconds"
require_regex '^# - production replica baseline: >=2$' "railway.toml" "railway policy baseline comment must define replica floor"
require_regex '^# - per-replica baseline: 2 vCPU / 2 GiB$' "railway.toml" "railway policy baseline comment must define per-replica CPU and memory"

go_version="$(go list -m -f '{{.GoVersion}}')"
# Keep Go toolchain pins aligned across local, Docker, and CI surfaces.
require_regex "^FROM --platform=\\\$BUILDPLATFORM golang:${go_version}-bookworm@sha256:[[:xdigit:]]{64} AS build$" "build/docker/Dockerfile" "runtime Docker build Go image must match go.mod"
require_regex '^COPY --from=build /out/migrate /migrate$' "build/docker/Dockerfile" "runtime image must ship the dedicated migration binary"
require_regex '^COPY --from=build /src/env/migrations /env/migrations$' "build/docker/Dockerfile" "runtime image must ship migration files for Railway pre-deploy"
require_regex '^!env/migrations$' ".dockerignore" "docker build context must re-include env/migrations"
require_regex '^!env/migrations/\*\*$' ".dockerignore" "docker build context must include migration files under env/migrations"
require_regex "^FROM golang:${go_version}-bookworm@sha256:[[:xdigit:]]{64} AS go_toolchain$" "build/docker/tooling-images.Dockerfile" "Docker tooling Go image must match go.mod"
if grep -Eq 'golangci/golangci-lint:' "build/docker/tooling-images.Dockerfile"; then
  echo "guardrail check failed: golangci-lint tooling image must be removed; Docker lint uses go tool golangci-lint"
  echo "  file: build/docker/tooling-images.Dockerfile"
  exit 1
fi

# Keep canonical build path aligned with hardened repository Dockerfile.
require_regex 'docker build' ".github/workflows/cd.yml" "cd workflow must build with docker build"
require_regex '-f build/docker/Dockerfile' ".github/workflows/cd.yml" "cd workflow must explicitly use build/docker/Dockerfile"

# Keep the runtime bridge from AGENTS.md to the detailed workflow reference.
require_regex 'docs/spec-first-workflow\.md' "AGENTS.md" "AGENTS.md must point to docs/spec-first-workflow.md for non-trivial workflow execution"
require_regex 'SOUL\.md' "AGENTS.md" "AGENTS.md must reference SOUL.md for orchestrator personality guidance"
require_regex '^@SOUL\.md$' "AGENTS.md" "AGENTS.md must include SOUL.md using the repository include convention"
require_regex 'lower-precedence orchestrator personality guidance' "AGENTS.md" "AGENTS.md must keep SOUL.md lower-precedence"
require_regex 'AGENTS\.md`, `docs/spec-first-workflow\.md`, task-local artifacts, and explicit user/system/developer instructions override `SOUL\.md`' "AGENTS.md" "AGENTS.md must keep operational authority above SOUL.md"
require_regex '`answer`, `explain`, `review`, `diagnose`, and `plan` authorize inspection and reporting, not implementation' "AGENTS.md" "read-oriented requests must not imply implementation authority"
require_regex '`change`, `build`, and `fix` authorize in-scope local edits plus relevant non-destructive validation' "AGENTS.md" "change-oriented requests must authorize bounded local execution"
require_regex 'Require confirmation only for external writes, destructive actions, purchases, or material scope expansion' "AGENTS.md" "confirmation policy must stay compact and risk-based"
require_regex 'Specification review is mandatory' "AGENTS.md" "AGENTS.md must keep mandatory post-specification review"
require_regex 'Replaced or unused legacy code is not acceptable as remembered-later cleanup' "AGENTS.md" "AGENTS.md must keep the legacy cleanup invariant"
require_regex 'current owner, reason, proof of continued need, and exit condition' "AGENTS.md" "legacy cleanup invariant must require bounded retained-surface proof"
require_regex 'Ordinary intake reconstructs the likely brief first' "AGENTS.md" "ordinary intake must reconstruct the likely brief before questioning"
require_regex 'resolves repository-factual uncertainty through bounded inspection' "AGENTS.md" "ordinary intake must inspect repository facts instead of asking the user"
require_regex 'asks only the smallest decision-changing question, one at a time' "AGENTS.md" "ordinary intake must ask one decision-changing question at a time"
require_regex 'State safe assumptions with reopen triggers and continue' "AGENTS.md" "ordinary intake must continue on safe bounded assumptions"
require_regex 'Stop when objective, scope/non-goals, constraints, success criteria, and reopen conditions are sufficient for routing' "AGENTS.md" "ordinary intake must stop once the brief is routing-sufficient"
require_regex 'Use the repo-local `grilling` skill only when the user explicitly asks to grill, stress-test, challenge every branch, or conduct an exhaustive design interview' "AGENTS.md" "AGENTS.md must reserve grilling for explicit exhaustive requests"
require_absent_regex 'use the repo-local `grilling` skill when user-owned decisions remain' "AGENTS.md" "AGENTS.md must not require grilling for ordinary intake"
require_regex 'Ordinary Phase 0 is a minimal clarification pass, not an exhaustive interview' "docs/spec-first-workflow/phases/intake.md" "intake must distinguish ordinary clarification from exhaustive grilling"
require_regex 'Ask one decision-changing question at a time' "docs/spec-first-workflow/phases/intake.md" "intake must ask one decision-changing question at a time"
require_regex 'Prefer a safe bounded assumption over another interview round' "docs/spec-first-workflow/phases/intake.md" "intake must continue on safe bounded assumptions"
require_regex 'Use the repo-local `grilling` skill only when the user explicitly asks to grill, stress-test, challenge every branch, or conduct an exhaustive design interview' "docs/spec-first-workflow/phases/intake.md" "intake must reserve grilling for explicit exhaustive requests"
require_absent_regex 'Use the repo-local `grilling` skill as the interview method whenever Phase 0 needs user answers' "docs/spec-first-workflow/phases/intake.md" "intake must not bind ordinary clarification to grilling"
require_regex 'Do not use for ordinary Phase 0 clarification' ".agents/skills/grilling/SKILL.md" "grilling must exclude ordinary Phase 0 clarification"
require_regex 'Ask exactly one question at a time' ".agents/skills/grilling/SKILL.md" "grilling must ask one question at a time"
require_regex 'Stop when no unresolved decision remains that can materially change the plan or build choice' ".agents/skills/grilling/SKILL.md" "grilling must stop when material decisions are resolved"
require_absent_regex 'Interview me relentlessly about every aspect|Walk down each branch of the design tree' ".agents/skills/grilling/SKILL.md" "grilling must not require unbounded questioning"
require_regex 'reserve `grilling` for an explicit exhaustive stress-test request' "README.md" "README ordinary-intake example must reserve grilling for explicit requests"
require_regex 'the user explicitly asks to grill, stress-test, challenge every branch, or conduct an exhaustive design interview' "README.md" "README skill catalog must expose the explicit grilling trigger"
require_regex 'lower-precedence personality and engineering-judgment guidance' "SOUL.md" "SOUL.md must describe itself as lower-precedence personality guidance"
require_regex 'conflicts with `AGENTS\.md`, the detailed workflow companion, or task-local artifacts, follow the authoritative artifact and treat the conflict as drift to repair' "SOUL.md" "SOUL.md must preserve the AGENTS/task-local precedence boundary"
require_regex 'follow `AGENTS\.md`' "docs/spec-first-workflow.md" "spec-first-workflow doc must declare AGENTS.md as the controlling contract"
require_markdown_links_exist "docs/spec-first-workflow.md" "spec-first-workflow/" "docs" "spec-first-workflow router links must point to existing split docs"
for workflow_detail_doc in docs/spec-first-workflow/shared/*.md docs/spec-first-workflow/phases/*.md; do
  require_regex '^## Read When$' "${workflow_detail_doc}" "split workflow docs must state when to read the file"
  require_regex '^## Inputs$' "${workflow_detail_doc}" "split workflow docs must state phase/shared inputs"
  require_regex '^## Outputs$' "${workflow_detail_doc}" "split workflow docs must state phase/shared outputs"
  require_regex '^## Stop Rule$' "${workflow_detail_doc}" "split workflow docs must state the stop rule"
done
require_regex 'known in-scope legacy surfaces are represented as removal/refactor work' "docs/spec-first-workflow/phases/task-review-readiness.md" "task-review doc must keep legacy cleanup readiness mechanics"
require_regex 'targeted negative searches or reads for retired identifiers' "docs/spec-first-workflow/phases/implementation-validation-closeout.md" "implementation/validation doc must keep legacy cleanup validation proof mechanics"
require_regex 'Legacy cleanup audit' "docs/spec-first-workflow/phases/planning.md" "planning doc must keep the per-surface legacy cleanup audit table"
require_regex 'No known replacement surface' "docs/spec-first-workflow/phases/specification.md" "specification doc must keep the no-replacement explicit path"
require_regex 'artifact_state: draft \| review_ready \| blocked' "docs/spec-first-workflow/phases/specification.md" "specification template must use the canonical artifact lifecycle"
require_regex 'No behavior/contract delta' "docs/spec-first-workflow/phases/specification.md" "specification doc must keep the explicit no-delta path"
require_regex 'Disposition table' "docs/spec-first-workflow/phases/specification.md" "specification clarification fan-in must keep per-question disposition"
require_regex 'Legacy cleanup audit' "docs/spec-first-workflow/phases/specification.md" "specification doc must keep the legacy cleanup audit table"
require_regex 'accepted risks and proof obligations are named' "docs/spec-first-workflow/phases/specification.md" "specification review-ready bar must keep accepted-risk and proof-obligation naming"
require_regex 'Fan-in destination' "docs/subagent-contract.md" "subagent fan-in must require material findings to name their destination"
require_regex 'TBD`, open alternatives, or implementation-time product choices' "docs/spec-first-workflow/shared/artifact-model.md" "artifact model must block unresolved decision placeholders"
require_regex 'claim, evidence, freshness, negative proof, and carrying artifact' "docs/spec-first-workflow/phases/planning.md" "planning proof obligations must keep claim/evidence/freshness/negative-proof shape"
require_regex 'treat `defer_to_design` as `FAIL`' "docs/spec-first-workflow/phases/specification-review.md" "specification review must fail deferred production-readiness decisions"
require_regex 'Do not point agents at a specific task-local `specs/\.\.\.` bundle as required precedent unless that directory exists in the current checkout' "docs/spec-first-workflow/shared/artifact-model.md" "artifact-model doc must warn against non-existent task-local specs examples"
require_regex 'workflow-plans/specification-review\.md' "docs/spec-first-workflow/shared/artifact-model.md" "artifact-model doc must name the durable specification-review phase surface"
require_absent_regex 'study `specs/[^`]+`' "docs/spec-first-workflow/shared/artifact-model.md" "artifact-model doc must not require studying a concrete specs bundle that may be absent"
require_regex 'Do not create synthetic bundles as examples' "specs/README.md" "specs README must prevent fake example bundles"
require_absent_regex '^max_threads[[:space:]]*=' ".codex/config.toml" "Codex runtime capacity must not be replaced with a repository hard-coded thread cap"
require_regex '^max_depth = 1$' ".codex/config.toml" "Codex subagent nesting depth must stay at the documented default"
require_regex 'agents\.<name>\.config_file' ".codex/config.toml" "Codex registry compatibility note must stay documented"
require_absent_regex '^model[[:space:]]*=' ".codex/config.toml" "root orchestrator model selection must remain user-owned"
require_absent_regex '^model_reasoning_effort[[:space:]]*=' ".codex/config.toml" "root orchestrator reasoning selection must remain user-owned"
require_absent_regex '^(ANTHROPIC(_DEFAULT_[A-Z]+)?_MODEL|CLAUDE_CODE_SUBAGENT_MODEL)[[:space:]]*=' ".codex/config.toml" "subprocess model selection must remain per-launch and orchestrator-owned"
for agent_config in .codex/agents/*.toml; do
  require_regex '^sandbox_mode = "read-only"$' "${agent_config}" "Codex subagent source configs must enforce read-only execution"
  require_regex 'Apply `docs/subagent-contract\.md`' "${agent_config}" "Codex subagent configs must consume the canonical shared contract"
  require_absent_regex '^model[[:space:]]*=' "${agent_config}" "Codex subagent configs must not pin a model"
  require_absent_regex '^model_reasoning_effort[[:space:]]*=' "${agent_config}" "Codex subagent configs must not pin reasoning effort"
  require_absent_regex '^Shared contract$|^Required input bundle$|^Handoff classification$|^Input-gap behavior$' "${agent_config}" "Codex subagent configs must contain only domain deltas"
done
require_regex 'Before launching any subagent or subprocess, the root chooses the exact currently available model' "AGENTS.md" "root must choose every child model before launch"
require_regex 'Before every launch, the root records the exact model identifier' "docs/spec-first-workflow/shared/subagents-and-handoff.md" "child model routing must be explicit and task-shaped"
require_regex 'silent inheritance from the root session is invalid' "docs/spec-first-workflow/phases/implementation-validation-closeout.md" "worker launches must enforce the selected model route"
require_regex 'make agents-check' ".github/workflows/ci.yml" "CI must check Codex/Claude agent mirror drift"
require_regex 'AGENTS_SYNC_SCRIPT' "Makefile" "Makefile must expose agent mirror sync/check targets"
require_regex 'Subagent gate: complete \| scoped_down \| local_only \| waived \| not_expected \| blocked' "AGENTS.md" "AGENTS.md must keep subagent gate readiness status explicit"
require_regex 'missing explicit subagent authorization' "AGENTS.md" "AGENTS.md must block instead of local-only when subagent authorization is missing"
require_regex 'Direct-path code writing is the narrow exception' "AGENTS.md" "AGENTS.md must preserve the current-session direct writer exception"
require_regex 'For approved ledgers with code-writing implementation, worker delegation is mandatory' "AGENTS.md" "AGENTS.md must preserve ledger-backed worker execution"
require_regex 'Design fan-out: complete \| scoped_down \| local_only \| blocked' "AGENTS.md" "AGENTS.md must keep design-checkpoint authoring fan-out status explicit"
require_regex 'Use subagents only when work divides into concrete, independent, bounded questions and separate context materially improves speed or quality' "AGENTS.md" "canonical fan-out must be question-driven"
require_regex 'Default to no more than three concurrently active subagent lanes for one root task' "AGENTS.md" "normal subagent concurrency must stay bounded"
require_regex 'FANOUT-INDEPENDENT' "AGENTS.md" "canonical fan-out decision must remain machine-visible"
require_regex 'FANOUT-LOCAL' "AGENTS.md" "canonical local-work decision must remain machine-visible"
require_regex 'FANOUT-CONCURRENCY' "AGENTS.md" "canonical fan-out concurrency policy must remain machine-visible"
require_regex 'When the next phase is a design authoring checkpoint, the prompt must name exactly one checkpoint' "docs/spec-first-workflow/shared/subagents-and-handoff.md" "design checkpoint handoff prompts must name one checkpoint and start with design fan-out"
require_regex '^## Compact Handoff Contract$' "docs/spec-first-workflow/shared/subagents-and-handoff.md" "shared handoff doc must own the compact chat handoff contract"
require_regex 'This file is the single owner of the chat handoff contract' "docs/spec-first-workflow/shared/subagents-and-handoff.md" "shared handoff doc must remain the single prompt-contract owner"
require_regex '^### Skill Wrapper Boundary$' "docs/spec-first-workflow.md" "workflow router must define the thin session-wrapper boundary"
require_regex 'must not copy the global read order, typed-state tables, authorization line, final-handoff template' "docs/spec-first-workflow.md" "session wrappers must reference canonical owners instead of duplicating workflow scaffolding"
require_regex 'First, set a Codex Goal for this session: <durable objective>' "docs/spec-first-workflow/shared/subagents-and-handoff.md" "shared handoff owner must keep the compact Goal prompt start"
require_regex 'Completion condition: <one successful completion condition>' "docs/spec-first-workflow/shared/subagents-and-handoff.md" "shared handoff owner must keep completion separate from blocked stop"
require_regex 'This skill is a renderer, not a second contract owner' ".agents/skills/codex-goal-prompt-composer/SKILL.md" "Goal composer must render rather than duplicate the handoff contract"
require_regex '^## Conditional References$' ".agents/skills/agent-prompt-composer/SKILL.md" "agent prompt composer must select references conditionally"
require_regex 'If a reference cannot change the handoff, skip it' ".agents/skills/agent-prompt-composer/SKILL.md" "agent prompt composer must skip irrelevant references"
require_regex 'Omit empty headings' ".agents/skills/agent-prompt-composer/SKILL.md" "agent prompt composer must not force a global section template"
require_regex 'single repository-wide owner of worker launch, resume, sandbox, prompt-file lifecycle, patch-intake, and integration-proof mechanics' "docs/spec-first-workflow/phases/implementation-validation-closeout.md" "implementation phase must own detailed worker execution mechanics"
require_regex '--ask-for-approval never' "docs/spec-first-workflow/phases/implementation-validation-closeout.md" "canonical worker launch must retain explicit non-interactive approval policy"
for compact_prompt_surface in \
  ".agents/skills/codex-goal-prompt-composer/SKILL.md" \
  "docs/spec-first-workflow/shared/subagents-and-handoff.md" \
  "docs/spec-first-workflow/phases/planning.md" \
  "docs/spec-first-workflow/phases/task-review-readiness.md"; do
  require_absent_regex 'codex exec --cd|codex --cd .*exec|--sandbox workspace-write|--ask-for-approval never' "${compact_prompt_surface}" "worker command/manual syntax must stay in implementation-validation-closeout.md"
done
require_absent_regex 'Orchestrator control posture' ".agents/skills/codex-goal-prompt-composer/SKILL.md" "Goal composer must not embed a generic orchestrator manual"
require_absent_regex 'Orchestrator control posture' "docs/spec-first-workflow/shared/subagents-and-handoff.md" "compact handoff contract must not embed a generic orchestrator manual"

session_wrappers=(
  "workflow-planning-session"
  "research-session"
  "specification-session"
  "specification-review-session"
  "technical-design-session"
  "test-design-session"
  "planning-session"
  "validation-closeout-session"
  "workflow-status"
  "workflow-plan-adequacy-challenge"
)
for wrapper in "${session_wrappers[@]}"; do
  wrapper_file=".agents/skills/${wrapper}/SKILL.md"
  require_regex '^## Canonical Owners$' "${wrapper_file}" "${wrapper} must link to canonical workflow owners"
  require_regex 'docs/spec-first-workflow' "${wrapper_file}" "${wrapper} must reference canonical workflow docs"
  require_absent_regex '^## Read First$|^## Artifact Read Order$' "${wrapper_file}" "${wrapper} must not duplicate the canonical read order"
  require_absent_regex '^## Required Final Chat Handoff$|^## Required Final Chat Shape$' "${wrapper_file}" "${wrapper} must not duplicate the canonical handoff template"
  require_absent_regex '^## Anti-Patterns$' "${wrapper_file}" "${wrapper} must not carry generic wrapper anti-pattern scaffolding"
  wrapper_words="$(wc -w <"${wrapper_file}")"
  if [[ "${wrapper_words}" -gt 900 ]]; then
    echo "guardrail check failed: ${wrapper} is no longer a thin wrapper"
    echo "  file: ${wrapper_file}"
    echo "  words: ${wrapper_words}"
    exit 1
  fi
done

for skill_file in .agents/skills/*/SKILL.md; do
  if [[ "${skill_file}" != ".agents/skills/go-structural-quality-review/SKILL.md" ]]; then
    require_absent_regex '^## Outcome-First Operating Rules$' "${skill_file}" "generic Outcome-First scaffolding must not be copied into skills"
  fi
  require_absent_regex '^Subagent authorization: I explicitly request and authorize' "${skill_file}" "the exact subagent authorization line belongs only to the shared handoff owner"
done

require_regex 'Subagent Gate Decision' "docs/spec-first-workflow/phases/specification.md" "lean spec template must keep Subagent Gate Decision"
require_regex 'System / Integration Design Phase' "docs/spec-first-workflow/phases/system-integration-design.md" "system/integration design phase doc must exist"
require_regex 'Go Code / Ownership Design Phase' "docs/spec-first-workflow/phases/go-code-ownership-design.md" "go code ownership design phase doc must exist"
require_regex 'Test Design Phase' "docs/spec-first-workflow/phases/test-design.md" "test-design phase doc must exist"
require_regex 'Design fan-out \(system/integration\): complete \| scoped_down \| local_only \| blocked' "docs/spec-first-workflow/phases/system-integration-design.md" "system/integration design must require checkpoint-scoped design fan-out"
require_regex 'Design fan-out \(go-code/ownership\): complete \| scoped_down \| local_only \| blocked' "docs/spec-first-workflow/phases/go-code-ownership-design.md" "go code ownership design must require checkpoint-scoped design fan-out"
require_regex 'Test-design fan-out: complete \| scoped_down \| local_only \| blocked \| not_expected' "docs/spec-first-workflow/phases/test-design.md" "test-design must require checkpoint-scoped test-design fan-out"
require_regex 'system-integration-design -> go-code-ownership-design -> technical design review' ".agents/skills/technical-design-session/SKILL.md" "technical-design session must route through both design checkpoints"
require_regex 'technical design review -> test-design -> planning' ".agents/skills/test-design-session/SKILL.md" "test-design session must sit between technical design review and planning"
require_regex 'Design fan-out status' "docs/spec-first-workflow/phases/task-review-readiness.md" "task-review handoff must carry design fan-out status into implementation"
require_regex 'Test-design consumed' "docs/spec-first-workflow/phases/planning.md" "planning handoff must carry consumed test-design status"
require_regex 'Test-design consumed' "docs/spec-first-workflow/phases/task-review-readiness.md" "task-review handoff must verify consumed test-design status"
require_regex 'Subagent gates consumed' "docs/spec-first-workflow/phases/planning.md" "tasks template must record consumed subagent gates"
require_regex 'Ledger-review fan-out rationale' "docs/spec-first-workflow/phases/task-review-readiness.md" "task-review handoff must record ledger-review fan-out rationale"
require_regex 'Subagent authorization: I explicitly request and authorize read-only subagents, delegation, and parallel agent work' "docs/spec-first-workflow/shared/subagents-and-handoff.md" "workflow handoff prompts must carry explicit subagent authorization"
require_regex 'Read-only enforcement' "docs/subagent-brief-template.md" "subagent brief template must require read-only enforcement, not prompt-only boundary"
require_regex '^## Approval Or Review Gate Variant$' "docs/subagent-brief-template.md" "subagent brief template must keep one compact approval/review variant"
require_regex 'Finding format: <artifact anchor; evidence; impact; classification; owner/reopen target; why not stronger/weaker>' "docs/subagent-brief-template.md" "approval/review briefs must require anchored findings"
require_regex 'lens coverage table' "docs/spec-first-workflow/phases/specification-review.md" "specification-review doc must require specification-review lens coverage"
require_regex '^## Gate Preservation$' "docs/subagent-contract.md" "subagent contract must preserve mandatory independent gates without broad fan-out"
require_regex 'one concrete, independent, bounded question' "docs/subagent-contract.md" "subagent contract must keep one-question ownership"
require_regex 'root orchestrator owns' "docs/subagent-contract.md" "subagent contract must keep root synthesis authoritative"
require_regex 'missing explicit subagent authorization' "docs/subagent-contract.md" "subagent contract must block when authorization is missing instead of silently going local-only"
require_regex '^## Shared Review Finding Envelope$' "docs/subagent-contract.md" "subagent contract must own the shared review finding envelope"

router_skills=(
  "go-coder"
  "go-systematic-debugging"
  "planning-and-task-breakdown"
  "api-contract-designer-spec"
  "go-data-architect-spec"
  "go-design-spec"
  "go-architect-spec"
)
for router_skill in "${router_skills[@]}"; do
  router_file=".agents/skills/${router_skill}/SKILL.md"
  require_regex '^## Symptom-Driven Reference Selector$' "${router_file}" "${router_skill} must remain a symptom-driven router"
  router_words="$(wc -w <"${router_file}")"
  if [[ "${router_words}" -gt 1500 ]]; then
    echo "guardrail check failed: ${router_skill} entrypoint is no longer compact"
    echo "  file: ${router_file}"
    echo "  words: ${router_words}"
    exit 1
  fi
done

for review_file in .agents/skills/*-review/SKILL.md; do
  require_regex 'shared review finding envelope' "${review_file}" "review skills must inherit the shared finding envelope"
  require_absent_regex '^## Finding Quality Bar$|^## Deliverable Shape$' "${review_file}" "review skills must not duplicate the shared finding envelope"
done

for compact_review in go-concurrency-review go-security-review go-performance-review go-chi-review; do
  compact_review_file=".agents/skills/${compact_review}/SKILL.md"
  require_regex '^## Symptom-Driven Reference Selector$' "${compact_review_file}" "${compact_review} must remain a symptom-driven router"
  compact_review_words="$(wc -w <"${compact_review_file}")"
  if [[ "${compact_review_words}" -gt 900 ]]; then
    echo "guardrail check failed: ${compact_review} entrypoint is no longer compact"
    echo "  file: ${compact_review_file}"
    echo "  words: ${compact_review_words}"
    exit 1
  fi
done
require_regex '^# Specification Review Session$' ".agents/skills/specification-review-session/SKILL.md" "specification-review session skill must exist"
require_regex 'docs/spec-first-workflow/phases/specification-review.md' ".agents/skills/specification-review-session/SKILL.md" "specification-review wrapper must delegate lens and finding rules to the canonical phase owner"
require_regex 'mandatory specification review' ".agents/skills/planning-session/SKILL.md" "planning session must block on missing mandatory specification review"
require_regex 'completed design fan-out result' ".agents/skills/planning-session/SKILL.md" "planning session must block on missing design fan-out"
require_regex 'missing explicit subagent authorization is not a valid `Ledger-review fan-out rationale:`' ".agents/skills/planning-session/SKILL.md" "planning session must not convert missing subagent authorization into local review"
require_regex 'specification-review-approved `spec\.md`' ".agents/skills/technical-design-session/SKILL.md" "technical-design session must start only from specification-review-approved spec"
require_regex 'specification-review-approved `spec\.md`' ".agents/skills/planning-and-task-breakdown/SKILL.md" "planning-and-task-breakdown must start only from specification-review-approved spec"
require_regex 'Missing explicit subagent authorization is not a valid `Ledger-review fan-out rationale:`' ".agents/skills/planning-and-task-breakdown/SKILL.md" "planning-and-task-breakdown must not convert missing subagent authorization into local review"
require_regex 'Execution shape: <canonical value, matched SHAPE-\* rule, decisive evidence>' ".agents/skills/workflow-status/SKILL.md" "workflow-status must report canonical shape and evidence"
require_regex 'Adequacy: <required by ADEQUACY-\* yes/no; result, evidence, validity>' ".agents/skills/workflow-status/SKILL.md" "workflow-status must report adequacy state and evidence"
require_regex 'Do not create or repair `test-plan\.md` during planning' ".agents/skills/planning-and-task-breakdown/SKILL.md" "planning-and-task-breakdown must not create or repair test-plan during planning"
require_regex 'Formal `spec-clarification-challenge` is not waivable' "AGENTS.md" "formal clarification must not be waived while full/protected triggers remain"
require_regex 'Use no more than three concurrently active subagent lanes per root task by default' "docs/subagent-contract.md" "shared contract must keep normal fan-out bounded"
require_regex 'Pattern Fit Diligence' "AGENTS.md" "AGENTS.md must require pattern-fit diligence for non-trivial design choices"
require_regex 'research/pattern-fit\.md' "docs/spec-first-workflow/phases/research.md" "research doc must name the durable pattern-fit research surface"
require_regex 'Evidence boundary: <what counts; relevant constraints and non-goals>' "docs/subagent-brief-template.md" "compact subagent briefs must carry task-specific evidence boundaries"
require_regex 'open a Pattern Fit research or review lane' ".agents/skills/technical-design-session/SKILL.md" "technical design session must route pattern-fit evidence when relevant"
require_regex 'Before selecting an architecture, workflow, integration, resilience, consistency, data-flow, or abstraction shape, perform Pattern Fit Diligence' ".agents/skills/go-design-spec/SKILL.md" "design skill must require pattern-fit diligence before selecting design shape"
require_regex 'Confirm Pattern Fit Diligence decisions are explicit' ".agents/skills/planning-and-task-breakdown/SKILL.md" "planning skill must check approved pattern-fit decisions"
require_regex 'Implement approved design patterns without inventing new pattern-shaped machinery' ".agents/skills/go-coder/SKILL.md" "coder skill must implement approved patterns rather than inventing local shapes"
require_regex 'Prefer the current Go language and standard library, then established repository patterns' ".agents/skills/go-coder/SKILL.md" "coder skill must keep Go-native and repository-native choices ahead of local pattern machinery"
require_regex 'code-level pattern' ".agents/skills/go-language-simplifier-review/SKILL.md" "language simplifier review must check local code-level pattern simplification"
require_regex 'Code-level pattern fit is a coding and review concern' "docs/spec-first-workflow/shared/artifact-model.md" "artifact-model doc must separate code-level patterns from architecture pattern fit"
require_regex 'retained with owner, reason, proof, and exit condition' ".agents/skills/spec-document-designer/SKILL.md" "spec skill must require retained legacy-surface proof"
require_regex 'missing cleanup coverage is a planning blocker, not implementation discretion' "docs/spec-first-workflow/phases/task-review-readiness.md" "canonical task readiness must fail missing legacy cleanup tasking"
require_regex 'Cleanup required by the approved task is in scope' ".agents/skills/go-coder/SKILL.md" "coder skill must treat approved legacy cleanup as in scope"
require_regex 'File Responsibility Check' ".agents/skills/go-coder/SKILL.md" "coder skill must check file responsibility before growing hand-written files"
require_regex 'Maintainable implementation shape is part of decision quality' "docs/spec-first-workflow/shared/artifact-model.md" "artifact-model doc must treat focused file/package responsibility as decision quality"
require_regex 'unexplained surviving replaced or unused legacy surface' ".agents/skills/go-design-review/SKILL.md" "design review skill must flag unexplained legacy surfaces"
require_regex 'targeted negative proof for retired identifiers' ".agents/skills/go-verification-before-completion/SKILL.md" "verification skill must require legacy cleanup negative proof"
require_regex 'generic `rg legacy` is not sufficient' ".agents/skills/go-verification-before-completion/SKILL.md" "verification skill must reject generic legacy negative proof"

# Keep routing policy machine-checkable without promoting fixtures into authority.
require_regex '<!-- workflow-rule-table:start -->' "AGENTS.md" "AGENTS.md must expose marked canonical routing tables"
require_regex '<!-- workflow-rule-table:start -->' "docs/spec-first-workflow/shared/artifact-model.md" "artifact model must expose marked canonical state tables"
require_regex 'The marked Markdown tables in `AGENTS.md` and this artifact model are normative' "docs/spec-first-workflow/shared/artifact-model.md" "canonical tables must remain policy authority over checker fixtures"
require_regex 'Skill `evals/evals\.json` manifests are behavioral coverage assets' "docs/spec-first-workflow/shared/artifact-model.md" "behavioral eval assets must not be mislabeled as CI-executed proof"
require_regex 'func familyAllowsRule' "scripts/ci/workflow-routing-check/main.go" "routing checker must bind fixture coverage to evaluator families"
require_regex 'func validateFamilyInput' "scripts/ci/workflow-routing-check/main.go" "routing fixtures must use strict per-family input schemas"
require_regex 'claims rule %s without executing its rule-specific branch' "scripts/ci/workflow-routing-check/main.go" "declared fixture coverage must bind to executed rule branches"
require_regex 'type ruleTrace map\[string\]struct' "scripts/ci/workflow-routing-check/main.go" "routing coverage must be emitted from evaluator branches rather than reconstructed after execution"
require_regex 'func validateEvalManifests' "scripts/ci/workflow-routing-check/main.go" "skill eval manifests must be schema-checked with unique integer IDs"
require_regex 'TestRecordDeclaredCoverageRejectsCosmeticRuleClaim' "scripts/ci/workflow-routing-check/main_test.go" "routing checker must test rejection of cosmetic rule coverage"
require_regex 'TestValidateEvalManifestsRejectsMalformedManifests' "scripts/ci/workflow-routing-check/main_test.go" "routing checker must lock eval-manifest schema failures"
require_regex '^workflow-routing-check:$' "Makefile" "Makefile must expose workflow-routing-check"
require_regex 'bash scripts/ci/sync-mirror-integration-check\.sh' "Makefile" "workflow routing proof must run the hermetic mirror integration harness"
require_regex 'ci-local:.*' "Makefile" "Makefile must expose ci-local"
require_regex '\$\(MAKE\) mod-check workflow-routing-check guardrails-check' "Makefile" "ci-local must invoke workflow-routing-check directly"
require_regex '^docker-workflow-routing-check:$' "Makefile" "Makefile must expose Docker routing check"
require_regex 'workflow-routing-check\)' "scripts/dev/docker-tooling.sh" "Docker tooling must expose workflow-routing-check"
require_regex 'docker-tooling\.sh" workflow-routing-check' "scripts/dev/docker-tooling.sh" "Docker ci must invoke workflow-routing-check"
require_regex 'run: make workflow-routing-check' ".github/workflows/ci.yml" "GitHub CI must invoke workflow-routing-check explicitly"
require_regex 'run: make workflow-routing-check' ".github/workflows/cd.yml" "release preflight must invoke workflow-routing-check explicitly"

require_yaml_job_contains() {
  local job="$1"
  local pattern="$2"
  local file="$3"
  local message="$4"
  if ! awk -v job="${job}" -v pattern="${pattern}" '
    $0 ~ ("^  " job ":[[:space:]]*(#.*)?$") { inside = 1; next }
    inside && $0 ~ /^  [[:alnum:]_-]+:[[:space:]]*(#.*)?$/ { exit }
    inside {
      active = $0
      sub(/^[[:space:]]+/, "", active)
      if (active == pattern) found = 1
    }
    END { exit(found ? 0 : 1) }
  ' "${file}"; then
    echo "guardrail check failed: ${message}"
    echo "  file: ${file}"
    echo "  job: ${job}"
    echo "  expected text: ${pattern}"
    exit 1
  fi
}
require_yaml_job_contains "repo-integrity" "run: make workflow-routing-check" ".github/workflows/ci.yml" "repo-integrity must invoke workflow-routing-check"
require_yaml_job_contains "release-preflight" "run: make workflow-routing-check" ".github/workflows/cd.yml" "release-preflight must invoke workflow-routing-check"

# Behavioral evals are checked-in coverage assets; deterministic fixtures remain CI proof.
for marker in 'explicitly asks to grill' 'exactly one question' 'safe bounded assumption' 'bounded inspection' 'no unresolved decision remains'; do
  require_regex "${marker}" ".agents/skills/grilling/evals/evals.json" "grilling evals must cover ${marker}"
done
for marker in 'SHAPE-DIRECT' 'SHAPE-LEAN' 'SHAPE-FULL-FLOOR' 'AGENT-CAPABILITY' 'research_expectation=not_expected' 'ROUTING-GATE-NOT-FILE' 'TRANS-UPWARD'; do
  require_regex "${marker}" ".agents/skills/workflow-planning-session/evals/evals.json" "workflow-planning evals must cover ${marker}"
done
for marker in 'FULL-DATA' 'FULL-SECURITY' 'AGENT-CAPABILITY' 'FULL-AGENT-SUBSTANTIVE' 'TRANS-UPWARD' 'routing_revision'; do
  require_regex "${marker}" ".agents/skills/workflow-plan-adequacy-challenge/evals/evals.json" "adequacy evals must cover ${marker}"
done
for marker in 'STATUS-TASKS' 'STATUS-DIRECT-ENVELOPE' 'unsupported: no durable task state' 'routing_scope=current_session' 'legacy_unmapped'; do
  require_regex "${marker}" ".agents/skills/workflow-status/evals/evals.json" "workflow-status evals must cover ${marker}"
done
for marker in 'risk_challenge_outcome=RECLASSIFY_FULL' 'FULL-DATA' 'FULL-SECURITY' 'guarded upward reclassification'; do
  require_regex "${marker}" ".agents/skills/specification-session/evals/evals.json" "specification-session evals must cover Risk Challenge reclassification: ${marker}"
done
for marker in 'phase_control=required' 'phase_control=not_required' 'artifact_expectation=not_expected' 'waiver_disposition=none' 'durable_master_control=present'; do
  require_regex "${marker}" ".agents/skills/test-design-session/evals/evals.json" "test-design evals must cover conditional control and no-plan semantics: ${marker}"
done
require_regex 'Current `SHAPE-DIRECT` never enters task-ledger review' "docs/spec-first-workflow/phases/task-review-readiness.md" "direct path must remain outside task-ledger review"
require_absent_regex 'direct-path or explicitly user-requested prototype|tiny direct-path or' "docs/spec-first-workflow/phases/task-review-readiness.md" "task-review WAIVED must not use direct-path eligibility"
require_regex 'status-tasks-direct-route-never-authorizes' "scripts/ci/workflow-routing-check/testdata/cases.json" "status fixtures must prove direct path cannot enter task-ledger review"
require_regex 'sandbox_mode = "read-only"' ".codex/agents/challenger-agent.toml" "challenger agent must remain read-only"
require_regex 'canonical `ADEQUACY-\*` conditions' ".codex/agents/challenger-agent.toml" "challenger agent must consume the canonical adequacy predicate"
require_regex 'never classify, reclassify, edit, or approve state' ".codex/agents/challenger-agent.toml" "challenger agent must remain advisory"

# Mirror registries and observable states are checked-in generation policy.
require_regex '"claude_agents\|\.claude/agents\|optional"' "scripts/dev/sync-agents.sh" "agent mirror registry must name its optional Claude consumer"
for row in \
  '"claude_skills\|\.claude/skills\|optional"' \
  '"gemini_skills\|\.gemini/skills\|optional"' \
  '"github_skills\|\.github/skills\|optional"' \
  '"cursor_skills\|\.cursor/skills\|optional"' \
  '"opencode_skills\|\.opencode/skills\|optional"'; do
  require_regex "${row}" "scripts/dev/sync-skills.sh" "skill mirror registry row must remain explicit: ${row}"
done
for state in mirror_optional_absent mirror_present_in_sync mirror_present_stale mirror_required_missing mirror_render_failed mirror_compare_failed; do
  require_regex "${state}" "scripts/dev/sync-agents.sh" "agent sync must report ${state}"
  require_regex "${state}" "scripts/dev/sync-skills.sh" "skill sync must report ${state}"
done
require_regex 'skills mixed target aggregation' "scripts/ci/sync-mirror-integration-check.sh" "mirror integration harness must test mixed-target aggregation"
require_regex 'agents render error' "scripts/ci/sync-mirror-integration-check.sh" "mirror integration harness must test agent render failure"
require_regex 'skills render error' "scripts/ci/sync-mirror-integration-check.sh" "mirror integration harness must test skill render failure"
require_regex 'skills strict target-only' "scripts/ci/sync-mirror-integration-check.sh" "mirror integration harness must test strict target-only failure"
require_regex 'skills non-strict target-only' "scripts/ci/sync-mirror-integration-check.sh" "mirror integration harness must test non-strict target-only success"
require_regex 'repository state changed' "scripts/ci/sync-mirror-integration-check.sh" "mirror integration harness must prove repository state is unchanged"

# New normative output must not regress to retired routing/status terms.
stale_status_matches="$(grep -RInE --exclude-dir=specs --exclude='required-guardrails-check.sh' -- 'pending_task_review|draft_review_ready|eligible same-session collapse|same-session phase collapse|user-requested agent-backed|complex workflow-control|Ready for next session: maybe' AGENTS.md README.md docs .agents/skills .codex/agents scripts || true)"
if [[ -n "${stale_status_matches}" ]]; then
  echo "guardrail check failed: active workflow surfaces contain retired routing/status terminology"
  printf '%s\n' "${stale_status_matches}" | sed 's/^/  /'
  exit 1
fi

flat_status_matches="$(grep -RInE --exclude-dir=specs --exclude='artifact-model.md' --exclude='required-guardrails-check.sh' -- 'Phase status:|Artifact status:|Task ledger review:|Session boundary reached:|Ready for next session:' AGENTS.md README.md docs .agents/skills .codex/agents scripts || true)"
if [[ -n "${flat_status_matches}" ]]; then
  echo "guardrail check failed: active writer surfaces contain legacy flat workflow fields"
  printf '%s\n' "${flat_status_matches}" | sed 's/^/  /'
  exit 1
fi

lightweight_matches="$(grep -RInE --exclude-dir=specs --exclude='artifact-model.md' --exclude='main.go' --exclude='cases.json' --exclude='required-guardrails-check.sh' -- 'lightweight[ _-]local' AGENTS.md README.md docs .agents/skills .codex/agents scripts || true)"
if [[ -n "${lightweight_matches}" ]]; then
  echo "guardrail check failed: lightweight-local aliases are read-only legacy mappings, not new output"
  printf '%s\n' "${lightweight_matches}" | sed 's/^/  /'
  exit 1
fi

require_absent_regex 'high-risk|complex workflow-control' ".agents/skills/workflow-plan-adequacy-challenge/SKILL.md" "adequacy procedure must use only canonical trigger IDs"
require_absent_regex 'high-risk|complex workflow-control' ".agents/skills/workflow-planning-session/SKILL.md" "workflow planning must use only canonical trigger IDs"

require_absent_regex 'agent-backed' ".agents/skills/workflow-plan-adequacy-challenge/SKILL.md" "adequacy procedure must use canonical AGENT-* and FULL-AGENT-SUBSTANTIVE IDs instead of agent-backed prose predicates"
require_absent_regex 'agent-backed' ".agents/skills/workflow-planning-session/SKILL.md" "workflow planning must use canonical AGENT-* and FULL-AGENT-SUBSTANTIVE IDs instead of agent-backed prose predicates"

go run ./scripts/ci/workflow-routing-check --verify-coverage

branch_protection_contexts() {
  awk '
    /^required_contexts=\(/ { inside = 1; next }
    inside && /^\)/ { exit }
    inside {
      gsub(/^[[:space:]]*"/, "")
      gsub(/"[[:space:]]*$/, "")
      if ($0 != "") print
    }
  ' scripts/dev/configure-branch-protection.sh
}

required_contexts=()
while IFS= read -r context; do
  [[ -z "${context}" ]] && continue
  required_contexts+=("${context}")
done < <(branch_protection_contexts)
if [[ "${#required_contexts[@]}" -eq 0 ]]; then
  echo "guardrail check failed: branch protection required contexts could not be read"
  exit 1
fi

for context in "${required_contexts[@]}"; do
  require_regex "^[[:space:]]{2}${context}:" ".github/workflows/ci.yml" "ci workflow must expose '${context}' job context"
done

for context in "dependency-review" "repository-security" "govulncheck" "gosec"; do
  require_absent_regex "^[[:space:]]+\"${context}\"$|\"context\": \"${context}\"" "scripts/dev/configure-branch-protection.sh" "branch protection must not require optional/internal '${context}' context"
done

require_no_forbidden_go_imports \
  "internal/app must not import infra adapters, generated sqlc, or concrete DB drivers" \
  'github\.com/example/go-service-template-rest/internal/infra(/|$)|github\.com/jackc/pgx(/|$)' \
  ./internal/app/...

echo "required repository guardrails check passed"
