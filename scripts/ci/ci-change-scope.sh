#!/usr/bin/env bash
set -euo pipefail

is_docs_path() {
  case "$1" in
    docs/*.md | specs/*.md | .agents/*.md | .codex/*.md | .claude/*.md | .qwen/*.md | \
      AGENTS.md | CLAUDE.md | QWEN.md | CONTRIBUTING.md | README.md | SECURITY.md | CODE_OF_CONDUCT.md)
      return 0
      ;;
    *)
      return 1
      ;;
  esac
}

classify() {
  local path
  local scope="docs-only"
  local seen="false"

  while IFS= read -r -d '' path; do
    seen="true"
    if ! is_docs_path "${path}"; then
      scope="full"
    fi
  done

  if [[ "${seen}" != "true" ]]; then
    scope="full"
  fi

  printf '%s\n' "${scope}"
}

is_template_independent_path() {
  local path="$1"

  if is_docs_path "${path}"; then
    return 0
  fi

  case "${path}" in
    internal/config/* | \
      internal/infra/postgres/* | \
      internal/infra/postgresmigrate/* | \
      internal/infra/httpclient/* | \
      internal/infra/telemetry/telemetrytest/metrics.go | \
      cmd/migrate/* | \
      cmd/service/internal/bootstrap/* | \
      test/postgres_*_test.go | \
      examples/* | \
      migrations/* | \
      scripts/profiles/* | \
      scripts/init-module.sh | \
      scripts/ci/template-init-check.sh | \
      api/openapi/service.yaml | \
      go.mod | go.sum | tools/* | \
      Makefile | railway.toml | build/* | env/* | .github/* | .golangci.yml)
      return 1
      ;;
    internal/*.go | cmd/*.go | test/*.go | api/*)
      return 0
      ;;
    *)
      return 1
      ;;
  esac
}

template_required() {
  local path
  local seen="false"

  while IFS= read -r -d '' path; do
    seen="true"
    if ! is_template_independent_path "${path}"; then
      printf 'true\n'
      return
    fi
  done

  if [[ "${seen}" != "true" ]]; then
    printf 'true\n'
  else
    printf 'false\n'
  fi
}

assert_scope() {
  local expected="$1"
  shift
  local actual
  actual="$(printf '%s\0' "$@" | classify)"
  [[ "${actual}" == "${expected}" ]] || {
    echo "expected ${expected}, got ${actual}: $*" >&2
    exit 1
  }
}

assert_template_required() {
  local expected="$1"
  shift
  local actual
  actual="$(printf '%s\0' "$@" | template_required)"
  [[ "${actual}" == "${expected}" ]] || {
    echo "expected template-required=${expected}, got ${actual}: $*" >&2
    exit 1
  }
}

self_test() {
  assert_scope docs-only docs/ci.md specs/tooling/tasks.md AGENTS.md
  assert_scope docs-only .agents/skills/example/SKILL.md
  assert_scope full
  assert_scope full internal/service.go
  assert_scope full docs/diagram.png
  assert_scope full docs/ci.md .github/workflows/ci.yml
  assert_template_required false docs/ci.md AGENTS.md
  assert_template_required false \
    internal/greeting/service.go \
    internal/infra/http/greeting_api.go \
    cmd/service/main.go \
    test/health_test.go \
    api/openapi/oapi-codegen.yaml
  assert_template_required true
  assert_template_required true scripts/init-module.sh
  assert_template_required true scripts/profiles/database-none/startup_dependencies.go.tmpl
  assert_template_required true internal/config/defaults.go
  assert_template_required true cmd/service/internal/bootstrap/run.go
  assert_template_required true internal/infra/postgres/repository.go
  assert_template_required true test/postgres_integration_test.go
  assert_template_required true go.mod
  assert_template_required true .github/workflows/ci.yml
  assert_template_required true assets/logo.png
  assert_template_required true internal/greeting/service.go Makefile
  echo "CI change-scope routing check passed"
}

case "${1:-}" in
  classify)
    classify
    ;;
  template-required)
    template_required
    ;;
  self-test)
    self_test
    ;;
  *)
    echo "usage: $0 {classify|template-required|self-test}" >&2
    exit 2
    ;;
esac
