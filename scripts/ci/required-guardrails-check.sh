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
  ".agents/skills/specification-review-session/SKILL.md"
  "specs/README.md"
  "scripts/dev/sync-skills.sh"
  "scripts/dev/sync-agents.sh"
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

require_golangci_lint_workflow_version() {
  local file="$1"
  local expected_version="$2"

  require_regex "^[[:space:]]{2}GOLANGCI_LINT_VERSION: ${expected_version}$" "${file}" "golangci-lint workflow pin must match go.mod"
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
golangci_lint_version="$(go list -m -f '{{.Version}}' github.com/golangci/golangci-lint/v2)"

# Keep Go and golangci-lint toolchain pins aligned across local, Docker, and CI surfaces.
require_regex "^FROM --platform=\\\$BUILDPLATFORM golang:${go_version}-bookworm@sha256:[[:xdigit:]]{64} AS build$" "build/docker/Dockerfile" "runtime Docker build Go image must match go.mod"
require_regex '^COPY --from=build /out/migrate /migrate$' "build/docker/Dockerfile" "runtime image must ship the dedicated migration binary"
require_regex '^COPY --from=build /src/env/migrations /env/migrations$' "build/docker/Dockerfile" "runtime image must ship migration files for Railway pre-deploy"
require_regex '^!env/migrations$' ".dockerignore" "docker build context must re-include env/migrations"
require_regex '^!env/migrations/\*\*$' ".dockerignore" "docker build context must include migration files under env/migrations"
require_regex "^FROM golang:${go_version}-bookworm@sha256:[[:xdigit:]]{64} AS go_toolchain$" "build/docker/tooling-images.Dockerfile" "Docker tooling Go image must match go.mod"
require_golangci_lint_workflow_version ".github/workflows/ci.yml" "${golangci_lint_version}"
require_golangci_lint_workflow_version ".github/workflows/nightly.yml" "${golangci_lint_version}"
require_golangci_lint_workflow_version ".github/workflows/cd.yml" "${golangci_lint_version}"

if grep -Eq '^[[:space:]]*FROM[[:space:]]+.*[[:space:]]+AS[[:space:]]+golangci_lint_tool$' "build/docker/tooling-images.Dockerfile"; then
  require_regex "^FROM golangci/golangci-lint:${golangci_lint_version}@sha256:[[:xdigit:]]{64} AS golangci_lint_tool$" "build/docker/tooling-images.Dockerfile" "retained golangci-lint tooling image must match go.mod and remain digest pinned"
elif grep -Eq 'golangci/golangci-lint:' "build/docker/tooling-images.Dockerfile"; then
  echo "guardrail check failed: golangci-lint tooling image must use the checked golangci_lint_tool stage or be removed"
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
require_regex 'known in-scope legacy surfaces are represented as removal/refactor work' "docs/spec-first-workflow.md" "spec-first-workflow doc must keep legacy cleanup task-ledger mechanics"
require_regex 'targeted negative searches or reads for retired identifiers' "docs/spec-first-workflow.md" "spec-first-workflow doc must keep legacy cleanup validation proof mechanics"
require_regex 'Legacy cleanup audit' "docs/spec-first-workflow.md" "spec-first-workflow doc must keep the per-surface legacy cleanup audit table"
require_regex 'No known replacement surface' "docs/spec-first-workflow.md" "spec-first-workflow doc must keep the no-replacement explicit path"
require_regex 'Do not point agents at a specific task-local `specs/\.\.\.` bundle as required precedent unless that directory exists in the current checkout' "docs/spec-first-workflow.md" "spec-first-workflow doc must warn against non-existent task-local specs examples"
require_regex 'workflow-plans/specification-review\.md' "docs/spec-first-workflow.md" "spec-first-workflow doc must name the durable specification-review phase surface"
require_absent_regex 'study `specs/[^`]+`' "docs/spec-first-workflow.md" "spec-first-workflow doc must not require studying a concrete specs bundle that may be absent"
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
require_regex 'Design fan-out: complete \| scoped_down \| local_only \| blocked' "AGENTS.md" "AGENTS.md must keep technical-design authoring fan-out status explicit"
require_regex 'For full-orchestrated, protected-domain, high-impact, or user-requested agent-backed technical design, `local_only` is invalid' "AGENTS.md" "serious technical design must not bypass subagents with local_only"
require_regex 'technical-design next-session prompt must explicitly instruct the next agent to first record or run `Design fan-out`' "AGENTS.md" "technical-design handoff prompts must start with design fan-out"
require_regex 'Subagent Gate Decision' "docs/spec-first-workflow.md" "lean spec template must keep Subagent Gate Decision"
require_regex 'Technical Design Authoring Fan-Out' "docs/spec-first-workflow.md" "spec-first-workflow doc must require technical-design authoring fan-out"
require_regex 'Design fan-out status' "docs/spec-first-workflow.md" "tasks handoff must carry design fan-out status into planning and implementation"
require_regex 'Subagent gates consumed' "docs/spec-first-workflow.md" "tasks template must record consumed subagent gates"
require_regex 'Ledger-review fan-out rationale' "docs/spec-first-workflow.md" "tasks template must record ledger-review fan-out rationale"
require_regex 'Read-only enforcement' "docs/subagent-brief-template.md" "subagent brief template must require read-only enforcement, not prompt-only boundary"
require_regex 'Specification review variant' "docs/subagent-brief-template.md" "subagent brief template must include specification-review lane guidance"
require_regex 'Spec anchor' "docs/subagent-brief-template.md" "subagent brief template must require anchored specification-review findings"
require_regex 'lens coverage table' "docs/spec-first-workflow.md" "spec-first-workflow doc must require specification-review lens coverage"
require_regex 'Specification review fan-in' "docs/subagent-contract.md" "subagent contract must define specification-review fan-in"
require_regex 'technical-design-authoring fan-out' "docs/subagent-contract.md" "subagent contract must distinguish design authoring fan-out from design review"
require_regex '^# Specification Review Session$' ".agents/skills/specification-review-session/SKILL.md" "specification-review session skill must exist"
require_regex 'lens coverage table' ".agents/skills/specification-review-session/SKILL.md" "specification-review session must require lens coverage before PASS"
require_regex 'Spec anchor' ".agents/skills/specification-review-session/SKILL.md" "specification-review session must require anchored findings"
require_regex 'mandatory specification review' ".agents/skills/planning-session/SKILL.md" "planning session must block on missing mandatory specification review"
require_regex 'completed design fan-out result' ".agents/skills/planning-session/SKILL.md" "planning session must block on missing design fan-out"
require_regex 'specification-review-approved `spec\.md`' ".agents/skills/technical-design-session/SKILL.md" "technical-design session must start only from specification-review-approved spec"
require_regex 'specification-review-approved `spec\.md`' ".agents/skills/planning-and-task-breakdown/SKILL.md" "planning-and-task-breakdown must start only from specification-review-approved spec"
require_regex 'Specification review: <PASS / CONCERNS / FAIL / missing / not expected / unknown' ".agents/skills/workflow-status/SKILL.md" "workflow-status must report specification-review gate state"
require_regex 'Formal `spec-clarification-challenge` is not waivable' "AGENTS.md" "formal clarification must not be waived while full/protected triggers remain"
require_regex 'local-only rationale is valid only when it lists the decision frontier' "docs/subagent-contract.md" "local-only rationale must stay auditable"
require_regex 'Pattern Fit Diligence' "AGENTS.md" "AGENTS.md must require pattern-fit diligence for non-trivial design choices"
require_regex 'research/pattern-fit\.md' "docs/spec-first-workflow.md" "workflow doc must name the durable pattern-fit research surface"
require_regex 'Pattern fit scope' "docs/subagent-brief-template.md" "subagent brief template must carry pattern-fit scope"
require_regex 'open a Pattern Fit research or review lane' ".agents/skills/technical-design-session/SKILL.md" "technical design session must route pattern-fit evidence when relevant"
require_regex 'Before selecting an architecture, workflow, integration, resilience, consistency, data-flow, or abstraction shape, perform Pattern Fit Diligence' ".agents/skills/go-design-spec/SKILL.md" "design skill must require pattern-fit diligence before selecting design shape"
require_regex 'Confirm Pattern Fit Diligence decisions are explicit' ".agents/skills/planning-and-task-breakdown/SKILL.md" "planning skill must check approved pattern-fit decisions"
require_regex 'Approved Patterns Before Local Invention' ".agents/skills/go-coder/SKILL.md" "coder skill must implement approved patterns rather than inventing local shapes"
require_regex 'Code-Level Patterns For Simpler Go' ".agents/skills/go-coder/SKILL.md" "coder skill must prefer small Go code patterns only when they simplify code"
require_regex 'code-level pattern' ".agents/skills/go-language-simplifier-review/SKILL.md" "language simplifier review must check local code-level pattern simplification"
require_regex 'Code-level pattern fit is a coding and review concern' "docs/spec-first-workflow.md" "workflow doc must separate code-level patterns from architecture pattern fit"
require_regex 'retained with owner, reason, proof, and exit condition' ".agents/skills/spec-document-designer/SKILL.md" "spec skill must require retained legacy-surface proof"
require_regex 'missing in-scope legacy cleanup is a planning-readiness failure' ".agents/skills/planning-and-task-breakdown/SKILL.md" "planning skill must fail missing legacy cleanup tasking"
require_regex 'Cleanup required by the approved task is in scope' ".agents/skills/go-coder/SKILL.md" "coder skill must treat approved legacy cleanup as in scope"
require_regex 'File Responsibility Check' ".agents/skills/go-coder/SKILL.md" "coder skill must check file responsibility before growing hand-written files"
require_regex 'Maintainable implementation shape is part of decision quality' "docs/spec-first-workflow.md" "workflow doc must treat focused file/package responsibility as decision quality"
require_regex 'unexplained surviving replaced or unused legacy surface' ".agents/skills/go-design-review/SKILL.md" "design review skill must flag unexplained legacy surfaces"
require_regex 'targeted negative proof for retired identifiers' ".agents/skills/go-verification-before-completion/SKILL.md" "verification skill must require legacy cleanup negative proof"
require_regex 'generic `rg legacy` is not sufficient' ".agents/skills/go-verification-before-completion/SKILL.md" "verification skill must reject generic legacy negative proof"

# Keep branch protection required checks aligned with CI job contexts.
required_contexts=(
  "repo-integrity"
  "lint"
  "openapi-contract"
  "openapi-breaking"
  "test"
  "test-race"
  "test-coverage"
  "test-integration"
  "migration-validate"
  "go-security"
  "secret-scan"
  "container-security"
)

for context in "${required_contexts[@]}"; do
  require_regex "^[[:space:]]+\"${context}\"$" "scripts/dev/configure-branch-protection.sh" "branch protection must require '${context}' context"
  require_regex "^[[:space:]]{2}${context}:" ".github/workflows/ci.yml" "ci workflow must expose '${context}' job context"
done

for context in "dependency-review" "repository-security" "govulncheck" "gosec"; do
  require_absent_regex "^[[:space:]]+\"${context}\"$|\"context\": \"${context}\"" "scripts/dev/configure-branch-protection.sh" "branch protection must not require optional/internal '${context}' context"
done

require_no_forbidden_go_imports \
  "internal/app and internal/domain must not import infra adapters, generated sqlc, or concrete DB drivers" \
  'github\.com/example/go-service-template-rest/internal/infra(/|$)|github\.com/jackc/pgx(/|$)' \
  ./internal/app/... ./internal/domain/...

echo "required repository guardrails check passed"
