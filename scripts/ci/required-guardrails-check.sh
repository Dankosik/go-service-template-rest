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
  "docs/subagent-brief-template.md"
  "docs/spec-first-workflow.md"
  "docs/spec-first-workflow/shared/artifact-model.md"
  "docs/spec-first-workflow/shared/subagents-and-handoff.md"
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
  ".agents/skills/test-design-session/SKILL.md"
  ".agents/skills/specification-review-session/SKILL.md"
  "specs/README.md"
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
require_regex 'Specification review is mandatory' "AGENTS.md" "AGENTS.md must keep mandatory post-specification review"
require_regex 'Replaced or unused legacy code is not acceptable as remembered-later cleanup' "AGENTS.md" "AGENTS.md must keep the legacy cleanup invariant"
require_regex 'current owner, reason, proof of continued need, and exit condition' "AGENTS.md" "legacy cleanup invariant must require bounded retained-surface proof"
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
require_regex 'Status: draft \| review_ready \| blocked' "docs/spec-first-workflow/phases/specification.md" "specification template status must stay scoped to the specification phase"
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
require_regex '^max_threads = 20$' ".codex/config.toml" "Codex subagent fan-out ceiling must stay explicit"
require_regex '^max_depth = 1$' ".codex/config.toml" "Codex subagent nesting depth must stay at the documented default"
require_regex 'agents\.<name>\.config_file' ".codex/config.toml" "Codex registry compatibility note must stay documented"
for agent_config in .codex/agents/*.toml; do
  require_regex '^sandbox_mode = "read-only"$' "${agent_config}" "Codex subagent source configs must enforce read-only execution"
done
require_regex 'make agents-check' ".github/workflows/ci.yml" "CI must check Codex/Claude agent mirror drift"
require_regex 'AGENTS_SYNC_SCRIPT' "Makefile" "Makefile must expose agent mirror sync/check targets"
require_regex 'Subagent gate: complete \| scoped_down \| local_only \| waived \| not_expected \| blocked' "AGENTS.md" "AGENTS.md must keep subagent gate readiness status explicit"
require_regex 'missing explicit subagent authorization' "AGENTS.md" "AGENTS.md must block instead of local-only when subagent authorization is missing"
require_regex 'Design fan-out: complete \| scoped_down \| local_only \| blocked' "AGENTS.md" "AGENTS.md must keep design-checkpoint authoring fan-out status explicit"
require_regex 'For full-orchestrated, protected-domain, high-impact, or user-requested agent-backed technical design, `local_only` is invalid' "AGENTS.md" "serious technical design must not bypass subagents with local_only"
require_regex 'design-checkpoint next-session prompt must name exactly one design checkpoint' "AGENTS.md" "design checkpoint handoff prompts must name one checkpoint and start with design fan-out"
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
require_regex 'Specification review variant' "docs/subagent-brief-template.md" "subagent brief template must include specification-review lane guidance"
require_regex 'Spec anchor' "docs/subagent-brief-template.md" "subagent brief template must require anchored specification-review findings"
require_regex 'lens coverage table' "docs/spec-first-workflow/phases/specification-review.md" "specification-review doc must require specification-review lens coverage"
require_regex 'Specification review fan-in' "docs/subagent-contract.md" "subagent contract must define specification-review fan-in"
require_regex 'design-authoring fan-out' "docs/subagent-contract.md" "subagent contract must distinguish design authoring fan-out from design review"
require_regex 'missing explicit subagent authorization' "docs/subagent-contract.md" "subagent contract must block when authorization is missing instead of silently going local-only"
require_regex '^# Specification Review Session$' ".agents/skills/specification-review-session/SKILL.md" "specification-review session skill must exist"
require_regex 'lens coverage table' ".agents/skills/specification-review-session/SKILL.md" "specification-review session must require lens coverage before PASS"
require_regex 'Spec anchor' ".agents/skills/specification-review-session/SKILL.md" "specification-review session must require anchored findings"
require_regex 'mandatory specification review' ".agents/skills/planning-session/SKILL.md" "planning session must block on missing mandatory specification review"
require_regex 'completed design fan-out result' ".agents/skills/planning-session/SKILL.md" "planning session must block on missing design fan-out"
require_regex 'missing explicit subagent authorization is not a valid `Ledger-review fan-out rationale:`' ".agents/skills/planning-session/SKILL.md" "planning session must not convert missing subagent authorization into local review"
require_regex 'specification-review-approved `spec\.md`' ".agents/skills/technical-design-session/SKILL.md" "technical-design session must start only from specification-review-approved spec"
require_regex 'specification-review-approved `spec\.md`' ".agents/skills/planning-and-task-breakdown/SKILL.md" "planning-and-task-breakdown must start only from specification-review-approved spec"
require_regex 'Missing explicit subagent authorization is not a valid `Ledger-review fan-out rationale:`' ".agents/skills/planning-and-task-breakdown/SKILL.md" "planning-and-task-breakdown must not convert missing subagent authorization into local review"
require_regex 'Specification review: <PASS / CONCERNS / FAIL / missing / not expected / unknown' ".agents/skills/workflow-status/SKILL.md" "workflow-status must report specification-review gate state"
require_regex 'Test design: <approved / blocked / not expected / missing / unknown' ".agents/skills/workflow-status/SKILL.md" "workflow-status must report test-design gate state"
require_regex 'Do not create or repair `test-plan\.md` during planning' ".agents/skills/planning-and-task-breakdown/SKILL.md" "planning-and-task-breakdown must not create or repair test-plan during planning"
require_regex 'Formal `spec-clarification-challenge` is not waivable' "AGENTS.md" "formal clarification must not be waived while full/protected triggers remain"
require_regex 'local-only rationale is valid only when it lists the decision frontier' "docs/subagent-contract.md" "local-only rationale must stay auditable"
require_regex 'Pattern Fit Diligence' "AGENTS.md" "AGENTS.md must require pattern-fit diligence for non-trivial design choices"
require_regex 'research/pattern-fit\.md' "docs/spec-first-workflow/phases/research.md" "research doc must name the durable pattern-fit research surface"
require_regex 'Pattern fit scope' "docs/subagent-brief-template.md" "subagent brief template must carry pattern-fit scope"
require_regex 'open a Pattern Fit research or review lane' ".agents/skills/technical-design-session/SKILL.md" "technical design session must route pattern-fit evidence when relevant"
require_regex 'Before selecting an architecture, workflow, integration, resilience, consistency, data-flow, or abstraction shape, perform Pattern Fit Diligence' ".agents/skills/go-design-spec/SKILL.md" "design skill must require pattern-fit diligence before selecting design shape"
require_regex 'Confirm Pattern Fit Diligence decisions are explicit' ".agents/skills/planning-and-task-breakdown/SKILL.md" "planning skill must check approved pattern-fit decisions"
require_regex 'Approved Patterns Before Local Invention' ".agents/skills/go-coder/SKILL.md" "coder skill must implement approved patterns rather than inventing local shapes"
require_regex 'Code-Level Patterns For Simpler Go' ".agents/skills/go-coder/SKILL.md" "coder skill must prefer small Go code patterns only when they simplify code"
require_regex 'code-level pattern' ".agents/skills/go-language-simplifier-review/SKILL.md" "language simplifier review must check local code-level pattern simplification"
require_regex 'Code-level pattern fit is a coding and review concern' "docs/spec-first-workflow/shared/artifact-model.md" "artifact-model doc must separate code-level patterns from architecture pattern fit"
require_regex 'retained with owner, reason, proof, and exit condition' ".agents/skills/spec-document-designer/SKILL.md" "spec skill must require retained legacy-surface proof"
require_regex 'missing in-scope legacy cleanup is a planning-readiness failure' ".agents/skills/planning-and-task-breakdown/SKILL.md" "planning skill must fail missing legacy cleanup tasking"
require_regex 'Cleanup required by the approved task is in scope' ".agents/skills/go-coder/SKILL.md" "coder skill must treat approved legacy cleanup as in scope"
require_regex 'File Responsibility Check' ".agents/skills/go-coder/SKILL.md" "coder skill must check file responsibility before growing hand-written files"
require_regex 'Maintainable implementation shape is part of decision quality' "docs/spec-first-workflow/shared/artifact-model.md" "artifact-model doc must treat focused file/package responsibility as decision quality"
require_regex 'unexplained surviving replaced or unused legacy surface' ".agents/skills/go-design-review/SKILL.md" "design review skill must flag unexplained legacy surfaces"
require_regex 'targeted negative proof for retired identifiers' ".agents/skills/go-verification-before-completion/SKILL.md" "verification skill must require legacy cleanup negative proof"
require_regex 'generic `rg legacy` is not sufficient' ".agents/skills/go-verification-before-completion/SKILL.md" "verification skill must reject generic legacy negative proof"

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
