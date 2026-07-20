#!/usr/bin/env bash
set -euo pipefail

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

# Toolchain alignment.
go_version="$(go list -m -f '{{.GoVersion}}')"
require_regex "^FROM --platform=\\\$BUILDPLATFORM golang:${go_version}-bookworm@sha256:[[:xdigit:]]{64} AS build$" build/docker/Dockerfile "runtime build image must match go.mod"
require_regex "^FROM golang:${go_version}-bookworm@sha256:[[:xdigit:]]{64} AS go_toolchain$" build/docker/tooling-images.Dockerfile "tooling image must match go.mod"
require_regex '^COPY --from=build /out/migrate /migrate$' build/docker/Dockerfile "runtime image must ship the migration binary"
require_regex '^COPY --from=build /src/env/migrations /env/migrations$' build/docker/Dockerfile "runtime image must ship migration files"
require_regex '^!env/migrations$' .dockerignore "Docker context must include migrations"
require_regex '^!env/migrations/\*\*$' .dockerignore "Docker context must include nested migration files"
require_absent_regex 'golangci/golangci-lint:' build/docker/tooling-images.Dockerfile "Docker lint must use the Go toolchain dependency"

require_regex '^[[:space:]]+- iface$' .golangci.yml "golangci-lint must enable iface"
for analyzer in identical unused opaque unexported; do
  require_regex "^[[:space:]]+- ${analyzer}$" .golangci.yml "iface must enable ${analyzer}"
done
require_regex '^lint:$' Makefile "Makefile must expose the required lint target"
require_regex '^GO_FILES := .*git ls-files --cached --others --exclude-standard' Makefile "Go file discovery must avoid walking ignored work directories during every make invocation"
require_absent_regex '^GO_FILES := .*find \.' Makefile "Go file discovery must not recursively scan the whole checkout"
require_regex '^[[:space:]]+\$\(MAKE\) deadcode$' Makefile "lint must run deadcode"
require_regex '^[[:space:]]+\$\(MAKE\) nilaway$' Makefile "lint must run NilAway"
require_regex '^deadcode:$' Makefile "Makefile must expose deadcode"
require_regex 'go tool deadcode -test -tags=integration \./\.\.\.' Makefile "deadcode must cover tests and integration-tagged code"
require_regex '^nilaway:$' Makefile "Makefile must expose NilAway"
require_regex 'module_path="\$\$\(go list -m\)"' Makefile "NilAway must derive the first-party package prefix from go.mod"
require_regex 'go tool nilaway -include-pkgs="\$\$module_path" -test \./\.\.\.' Makefile "NilAway must cover current-module production and test code"
require_absent_regex 'go tool nilaway -include-pkgs=github\.com/example/go-service-template-rest' Makefile "NilAway must not retain the template module path"
require_absent_regex 'exclude-test-files|-test=false' Makefile "NilAway test analysis must not be disabled"
require_regex '^[[:space:]]+golang\.org/x/tools/cmd/deadcode$' go.mod "go.mod must register deadcode as a Go tool"
require_regex '^[[:space:]]+go\.uber\.org/nilaway/cmd/nilaway$' go.mod "go.mod must register NilAway as a Go tool"
require_regex '^[[:space:]]+go\.uber\.org/nilaway v0\.0\.0-20260717202641-aacdd5a364bf // indirect$' go.mod "NilAway must stay pinned to the accepted revision"
require_regex 'run_go "GOLANGCI_LINT_CACHE=/workspace/\.cache/golangci-lint make lint"' scripts/dev/docker-tooling.sh "Docker lint must route through make lint"
for workflow in .github/workflows/ci.yml .github/workflows/nightly.yml .github/workflows/cd.yml; do
  require_regex 'run: make lint' "${workflow}" "${workflow} must run the fail-closed repository lint target"
  require_absent_regex 'golangci/golangci-lint-action' "${workflow}" "${workflow} must not bypass the repository lint target"
done

require_regex 'docker build' .github/workflows/cd.yml "CD must build with docker build"
require_regex '-f build/docker/Dockerfile' .github/workflows/cd.yml "CD must use the repository Dockerfile"

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
