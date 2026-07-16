#!/usr/bin/env bash
set -euo pipefail

required_files=(
  AGENTS.md
  README.md
  railway.toml
  Makefile
  .editorconfig
  .gitattributes
  .golangci.yml
  .redocly.yaml
  .codex/config.toml
  .github/CODEOWNERS
  .github/dependabot.yml
  .github/pull_request_template.md
  .github/workflows/ci.yml
  .github/workflows/cd.yml
  .github/workflows/nightly.yml
  CONTRIBUTING.md
  SECURITY.md
  LICENSE
  env/.env.example
  env/docker-compose.yml
  build/docker/Dockerfile
  build/docker/tooling-images.Dockerfile
  docs/spec-first-workflow.md
  docs/spec-first-workflow/shared/artifact-model.md
  docs/spec-first-workflow/shared/subagents-and-handoff.md
  docs/subagent-contract.md
  docs/subagent-brief-template.md
  scripts/ci/workflow-instructions-check.sh
  scripts/dev/workflow-behavior-evals.sh
  scripts/dev/module-origin.sh
)

for file in "${required_files[@]}"; do
  if [[ ! -f "${file}" ]]; then
    echo "guardrail check failed: missing ${file}"
    exit 1
  fi
done

require_regex() {
  local pattern="$1"
  local file="$2"
  local message="$3"
  if ! grep -Eq -- "${pattern}" "${file}"; then
    echo "guardrail check failed: ${message}"
    echo "  file: ${file}"
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
    exit 1
  fi
}

require_no_forbidden_go_imports() {
  local message="$1"
  local pattern="$2"
  shift 2

  local imports
  imports="$(go list -f '{{range .Imports}}{{printf "%s\t%s\n" $.ImportPath .}}{{end}}{{range .TestImports}}{{printf "%s\t%s\n" $.ImportPath .}}{{end}}{{range .XTestImports}}{{printf "%s\t%s\n" $.ImportPath .}}{{end}}' "$@")"
  if printf '%s\n' "${imports}" | grep -Eq -- "${pattern}"; then
    echo "guardrail check failed: ${message}"
    printf '%s\n' "${imports}" | grep -E -- "${pattern}" | sed 's/^/  /'
    exit 1
  fi
}

# Deployment policy.
require_regex '^builder = "DOCKERFILE"$' railway.toml "Railway must use the Dockerfile builder"
require_regex '^dockerfilePath = "build/docker/Dockerfile"$' railway.toml "Railway must use build/docker/Dockerfile"
require_regex '^preDeployCommand = \["/migrate"\]$' railway.toml "Railway must run the migration binary before deploy"
require_regex '^healthcheckPath = "/health/ready"$' railway.toml "Railway readiness path must remain stable"
require_regex '^healthcheckTimeout = 180$' railway.toml "Railway healthcheck timeout must remain explicit"
require_regex '^restartPolicyType = "ON_FAILURE"$' railway.toml "Railway restart policy must remain explicit"
require_regex '^restartPolicyMaxRetries = 5$' railway.toml "Railway restart retries must remain bounded"
require_regex '^overlapSeconds = 45$' railway.toml "Railway overlap window must remain explicit"
require_regex '^drainingSeconds = 30$' railway.toml "Railway draining window must remain explicit"
require_regex '^# - production replica baseline: >=2$' railway.toml "Railway policy must retain the production replica floor"
require_regex '^# - per-replica baseline: 2 vCPU / 2 GiB$' railway.toml "Railway policy must retain the per-replica resource baseline"

# Toolchain alignment.
go_version="$(go list -m -f '{{.GoVersion}}')"
require_regex "^FROM --platform=\\\$BUILDPLATFORM golang:${go_version}-bookworm@sha256:[[:xdigit:]]{64} AS build$" build/docker/Dockerfile "runtime build image must match go.mod"
require_regex "^FROM golang:${go_version}-bookworm@sha256:[[:xdigit:]]{64} AS go_toolchain$" build/docker/tooling-images.Dockerfile "tooling image must match go.mod"
require_regex '^COPY --from=build /out/migrate /migrate$' build/docker/Dockerfile "runtime image must ship the migration binary"
require_regex '^COPY --from=build /src/env/migrations /env/migrations$' build/docker/Dockerfile "runtime image must ship migration files"
require_regex '^!env/migrations$' .dockerignore "Docker context must include migrations"
require_regex '^!env/migrations/\*\*$' .dockerignore "Docker context must include nested migration files"
require_absent_regex 'golangci/golangci-lint:' build/docker/tooling-images.Dockerfile "Docker lint must use the Go toolchain dependency"

require_regex 'docker build' .github/workflows/cd.yml "CD must build with docker build"
require_regex '-f build/docker/Dockerfile' .github/workflows/cd.yml "CD must use the repository Dockerfile"

# Instruction ownership and runtime configuration.
require_regex 'docs/spec-first-workflow\.md' AGENTS.md "AGENTS.md must link the workflow router"
require_regex '^## Engineering Judgment$' AGENTS.md "AGENTS.md must include engineering judgment"
require_regex '^## Collaboration$' AGENTS.md "AGENTS.md must include collaboration guidance"
require_absent_regex '^@RTK\.md$' AGENTS.md "AGENTS.md must not duplicate user-level RTK instructions"
require_absent_regex '^model[[:space:]]*=' .codex/config.toml "root model selection must remain user-owned"
require_absent_regex '^model_reasoning_effort[[:space:]]*=' .codex/config.toml "root reasoning effort must remain user-owned"
require_regex '^max_depth = 1$' .codex/config.toml "nested subagent delegation must remain disabled by default"

for agent_config in .codex/agents/*.toml; do
  require_regex '^sandbox_mode = "read-only"$' "${agent_config}" "subagent configs must remain read-only"
  require_regex 'docs/subagent-contract\.md' "${agent_config}" "subagent configs must use the shared contract"
  require_absent_regex '^model[[:space:]]*=' "${agent_config}" "subagent configs must not pin models"
  require_absent_regex '^model_reasoning_effort[[:space:]]*=' "${agent_config}" "subagent configs must not pin reasoning effort"
done

# Workflow checks keep structure and links deterministic without locking prose.
bash scripts/ci/workflow-instructions-check.sh
require_regex '^workflow-routing-check:$' Makefile "Makefile must expose the compatibility workflow check target"
require_regex 'go run \.\/scripts/ci/hard-skills-check' Makefile "workflow check target must run the hard-skills checker"
require_regex 'bash scripts/ci/workflow-instructions-check\.sh' Makefile "workflow check target must run the lean instruction checker"
require_regex '^workflow-behavior-evals-check:$' Makefile "Makefile must expose the behavior eval manifest check"
require_regex '^workflow-behavior-evals:$' Makefile "Makefile must expose external behavior eval execution"
require_regex 'bash scripts/dev/workflow-behavior-evals\.sh check' scripts/ci/workflow-instructions-check.sh "workflow instruction check must validate the behavior eval manifest"
require_regex 'run: make workflow-routing-check' .github/workflows/ci.yml "CI must run the workflow instruction check"
require_regex 'run: make workflow-routing-check' .github/workflows/cd.yml "release preflight must run the workflow instruction check"
require_regex 'workflow-routing-check\)' scripts/dev/docker-tooling.sh "Docker tooling must expose the workflow check"
require_regex 'run_go "go run \.\/scripts/ci/hard-skills-check"' scripts/dev/docker-tooling.sh "Docker workflow check must run the hard-skills checker in the Go image"
require_regex 'bash "\$\{ROOT_DIR\}/scripts/ci/instruction-evals-check\.sh"' scripts/dev/docker-tooling.sh "Docker workflow check must run the instruction eval guard"
require_regex 'bash "\$\{ROOT_DIR\}/scripts/ci/workflow-instructions-check\.sh"' scripts/dev/docker-tooling.sh "Docker workflow check must run the workflow instruction guard"
# Branch-protection contexts must match CI jobs.
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

context_count=0
while IFS= read -r context; do
  [[ -z "${context}" ]] && continue
  context_count=$((context_count + 1))
  require_regex "^[[:space:]]{2}${context}:" .github/workflows/ci.yml "CI is missing branch-protection context ${context}"
done < <(branch_protection_contexts)

if (( context_count == 0 )); then
  echo "guardrail check failed: no branch-protection contexts were found"
  exit 1
fi

for context in dependency-review repository-security govulncheck gosec; do
  require_absent_regex "^[[:space:]]+\"${context}\"$|\"context\": \"${context}\"" scripts/dev/configure-branch-protection.sh "branch protection must not require optional context ${context}"
done

require_no_forbidden_go_imports \
  "internal/app must not import infrastructure adapters or concrete DB drivers" \
  'github\.com/example/go-service-template-rest/internal/infra(/|$)|github\.com/jackc/pgx(/|$)' \
  ./internal/app/...

echo "required repository guardrails check passed"
