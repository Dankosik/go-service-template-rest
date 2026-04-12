#!/usr/bin/env bash
set -euo pipefail

required_files=(
  "AGENTS.md"
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
require_regex '^healthcheckPath = "/health/ready"$' "railway.toml" "railway deploy healthcheck path must be /health/ready"
require_regex '^healthcheckTimeout = 180$' "railway.toml" "railway deploy healthcheck timeout must be 180 seconds"
require_regex '^restartPolicyType = "ON_FAILURE"$' "railway.toml" "railway restart policy type must be ON_FAILURE"
require_regex '^restartPolicyMaxRetries = 5$' "railway.toml" "railway restart retries must be locked to 5"
require_regex '^overlapSeconds = 45$' "railway.toml" "railway overlap window must be 45 seconds"
require_regex '^drainingSeconds = 30$' "railway.toml" "railway draining window must be 30 seconds"
require_regex '^# - production replica baseline: >=2$' "railway.toml" "railway policy baseline comment must define replica floor"
require_regex '^# - per-replica baseline: 2 vCPU / 2 GiB$' "railway.toml" "railway policy baseline comment must define per-replica CPU and memory"

# Keep canonical build path aligned with hardened repository Dockerfile.
require_regex 'docker build' ".github/workflows/cd.yml" "cd workflow must build with docker build"
require_regex '-f build/docker/Dockerfile' ".github/workflows/cd.yml" "cd workflow must explicitly use build/docker/Dockerfile"

# Keep the runtime bridge from AGENTS.md to the detailed workflow reference.
require_regex 'docs/spec-first-workflow\.md' "AGENTS.md" "AGENTS.md must point to docs/spec-first-workflow.md for non-trivial workflow execution"
require_regex 'follow `AGENTS\.md`' "docs/spec-first-workflow.md" "spec-first-workflow doc must declare AGENTS.md as the controlling contract"
require_regex '^max_threads = 8$' ".codex/config.toml" "Codex subagent fan-out ceiling must stay explicit and bounded"
require_regex '^max_depth = 1$' ".codex/config.toml" "Codex subagent nesting depth must stay at the documented default"
require_regex 'agents\.<name>\.config_file' ".codex/config.toml" "Codex registry compatibility note must stay documented"
require_regex 'make agents-check' ".github/workflows/ci.yml" "CI must check Codex/Claude agent mirror drift"
require_regex 'AGENTS_SYNC_SCRIPT' "Makefile" "Makefile must expose agent mirror sync/check targets"

# Keep branch protection required checks aligned with CI job contexts.
required_contexts=(
  "repo-integrity"
  "lint"
  "openapi-contract"
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
  require_regex "\"context\": \"${context}\"" "scripts/dev/configure-branch-protection.sh" "branch protection must require '${context}' context"
  require_regex "^[[:space:]]{2}${context}:" ".github/workflows/ci.yml" "ci workflow must expose '${context}' job context"
done

require_no_forbidden_go_imports \
  "internal/app and internal/domain must not import infra adapters, generated sqlc, or concrete DB drivers" \
  'github\.com/example/go-service-template-rest/internal/infra(/|$)|github\.com/jackc/pgx(/|$)' \
  ./internal/app/... ./internal/domain/...

echo "required repository guardrails check passed"
