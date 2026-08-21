#!/usr/bin/env bash
set -euo pipefail

is_docs_path() {
  case "$1" in
    docs/*.md | specs/*.md | .agents/*.md | .codex/*.md | .claude/*.md | .grok/*.md | .qwen/*.md | \
      AGENTS.md | CLAUDE.md | Grok.md | QWEN.md | CONTRIBUTING.md | README.md | SECURITY.md | CODE_OF_CONDUCT.md)
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

# Canonical job tokens. Profile-marked tokens drop out of generated services.
all_job_tokens() {
  printf 'minimal'
  # profile:object-storage:start
  printf ' object-storage'
  # profile:object-storage:end
  # profile:authn-oidc-jwt:start
  printf ' authn'
  # profile:authn-oidc-jwt:end
  # profile:grpc:start
  printf ' grpc'
  # profile:grpc:end
  # profile:messaging-nats-jetstream:start
  printf ' messaging'
  # profile:messaging-nats-jetstream:end
  # profile:outbox-postgres:start
  printf ' outbox'
  # profile:outbox-postgres:end
  # profile:jobs-postgres:start
  printf ' jobs'
  # profile:jobs-postgres:end
  # profile:webhooks-durable:start
  printf ' webhooks'
  # profile:webhooks-durable:end
  # profile:database-postgres:start
  printf ' postgres'
  # profile:database-postgres:end
  printf '\n'
}

append_unique() {
  local acc="$1"
  local job="$2"
  case " ${acc} " in
    *" ${job} "*)
      printf '%s' "${acc}"
      ;;
    *)
      if [[ -z "${acc}" ]]; then
        printf '%s' "${job}"
      else
        printf '%s %s' "${acc}" "${job}"
      fi
      ;;
  esac
}

merge_jobs() {
  local acc="$1"
  local extra="$2"
  local job
  for job in ${extra}; do
    acc="$(append_unique "${acc}" "${job}")"
  done
  printf '%s' "${acc}"
}

has_job() {
  case " $1 " in
    *" $2 "*) return 0 ;;
    *) return 1 ;;
  esac
}

ordered_jobs() {
  local selected="$1"
  local ordered=""
  local job
  for job in $(all_job_tokens); do
    if has_job "${selected}" "${job}"; then
      ordered="$(append_unique "${ordered}" "${job}")"
    fi
  done
  printf '%s' "${ordered}"
}

is_init_profile() {
  case "$1" in
    postgres) return 1 ;;
    *) return 0 ;;
  esac
}

jobs_to_json() {
  local selected="$1"
  local first="true"
  local job
  printf '['
  for job in $(ordered_jobs "${selected}"); do
    if [[ "${first}" == "true" ]]; then
      first="false"
    else
      printf ','
    fi
    printf '"%s"' "${job}"
  done
  printf ']\n'
}

init_profiles_json() {
  local selected="$1"
  local init=""
  local job
  for job in ${selected}; do
    if is_init_profile "${job}"; then
      init="$(append_unique "${init}" "${job}")"
    fi
  done
  jobs_to_json "${init}"
}

# Generator, module, workflow, and shared bootstrap changes re-prove every
# template job.
shared_generator_jobs() {
  all_job_tokens
}

# Space-separated template jobs that one path can falsify. Empty means skip.
profiles_for_path() {
  local path="$1"

  # profile:object-storage:start
  case "${path}" in
	# profile:http-idempotency-postgres:start
	  scripts/ci/runtime-image-build.sh | scripts/ci/fixtures/postgres-http-idempotency-active.patch | \
	  scripts/profiles/http-idempotency-postgres/* | internal/httpidempotency/* | internal/infra/postgresidempotency/*)
	    printf '%s\n' "postgres"
	    return
	    ;;
	# profile:http-idempotency-postgres:end
    internal/objectstorage/* | internal/infra/s3/* | \
      cmd/service/internal/bootstrap/startup_object_storage*.go | \
      internal/config/object_storage_config*.go | \
      test/s3conformance/conformance_test.go | \
      docs/s3-compatible-object-storage.md)
      printf '%s\n' "object-storage"
      return
      ;;
  esac
  # profile:object-storage:end

	# profile:webhooks-durable:start
	case "${path}" in
	  internal/outboundtrust/* | internal/infra/postgreswebhook/* | cmd/jobs-worker/builder.go | \
      internal/config/webhooks_config*.go | internal/config/jobs_worker_config*.go | \
      migrations/*_postgres_webhook*.sql | \
      test/postgres_webhook_*_test.go | test/webhook_*_integration_test.go | \
      docs/outbound-webhook-delivery.md)
      printf '%s\n' "webhooks"
      return
      ;;
	esac
	# profile:webhooks-durable:end

	if is_docs_path "${path}"; then
	  printf '\n'
	  return
	fi

	# profile:outbox-postgres:start
	case "${path}" in
	  internal/domainevent/* | internal/infra/natsjs/outbox* | \
	  internal/infra/postgresoutbox/* | cmd/outbox-relay/* | \
	  test/postgres_outbox_*_integration_test.go)
	    printf '%s\n' "outbox"
	    return
	    ;;
	esac
	# profile:outbox-postgres:end

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
      shared_generator_jobs
      return
      ;;
    internal/*.go | cmd/*.go | test/*.go | api/*)
      printf '\n'
      return
      ;;
    scripts/ci/ci-change-scope.sh | scripts/ci/instruction-evals-check.sh | evals/*)
      printf '\n'
      return
      ;;
    .agents/roles/* | .codex/agents/* | .claude/agents/* | .grok/roles/* | .grok/agents/* | \
      .grok/rules/* | .qwen/agents/* | \
      template-owned.paths | scripts/agent-roles-sync.sh | scripts/ci/template-owned-purity-check.sh)
      printf '%s\n' "minimal"
      return
      ;;
    *)
      all_job_tokens
      return
      ;;
  esac
}

is_template_independent_path() {
  local jobs
  jobs="$(profiles_for_path "$1")"
  jobs="${jobs//[$'\n']/}"
  [[ -z "${jobs// /}" ]]
}

collect_jobs() {
  local path
  local seen="false"
  local jobs=""
  local extra

  while IFS= read -r -d '' path; do
    seen="true"
    extra="$(profiles_for_path "${path}")"
    extra="${extra//[$'\n']/}"
    jobs="$(merge_jobs "${jobs}" "${extra}")"
  done

  if [[ "${seen}" != "true" ]]; then
    jobs="$(all_job_tokens)"
    jobs="${jobs//[$'\n']/}"
  fi

  ordered_jobs "${jobs}"
}

template_required() {
  local jobs
  jobs="$(collect_jobs)"
  if [[ -n "${jobs}" ]]; then
    printf 'true\n'
  else
    printf 'false\n'
  fi
}

template_scope() {
  local jobs required postgres
  jobs="$(collect_jobs)"
  required="false"
  postgres="false"
  if [[ -n "${jobs}" ]]; then
    required="true"
  fi
  if has_job "${jobs}" postgres; then
    postgres="true"
  fi
  printf 'required=%s\n' "${required}"
  printf 'init_profiles=%s\n' "$(init_profiles_json "${jobs}")"
  printf 'postgres_required=%s\n' "${postgres}"
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

assert_template_scope() {
  local expected_required="$1"
  local expected_init="$2"
  local expected_postgres="$3"
  shift 3
  local actual required init postgres
  actual="$(printf '%s\0' "$@" | template_scope)"
  required="$(printf '%s\n' "${actual}" | sed -n 's/^required=//p')"
  init="$(printf '%s\n' "${actual}" | sed -n 's/^init_profiles=//p')"
  postgres="$(printf '%s\n' "${actual}" | sed -n 's/^postgres_required=//p')"
  [[ "${required}" == "${expected_required}" &&
    "${init}" == "${expected_init}" &&
    "${postgres}" == "${expected_postgres}" ]] || {
    echo "expected required=${expected_required} init=${expected_init} postgres=${expected_postgres}" >&2
    printf '%s\n' "${actual}" >&2
    echo "paths: $*" >&2
    exit 1
  }
}

self_test() {
  local all_init
  all_init="$(init_profiles_json "$(all_job_tokens)")"
  all_init="${all_init%$'\n'}"

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
	# profile:http-idempotency-postgres:start
	assert_template_required true scripts/ci/runtime-image-build.sh
	assert_template_required true scripts/ci/fixtures/postgres-http-idempotency-active.patch
	assert_template_required true internal/httpidempotency/contract.go
	assert_template_required true internal/infra/postgresidempotency/store.go
	assert_template_scope true '[]' true internal/httpidempotency/contract.go
	# profile:http-idempotency-postgres:end
  # profile:object-storage:start
  assert_template_required true internal/objectstorage/store.go
  assert_template_required true internal/infra/s3/client.go
  assert_template_required true test/s3conformance/conformance_test.go
  assert_template_required true docs/s3-compatible-object-storage.md
  assert_template_scope true '["object-storage"]' false internal/objectstorage/store.go
  assert_template_scope true '["object-storage"]' false internal/infra/s3/client.go
  # profile:object-storage:end
  # profile:webhooks-durable:start
  assert_template_required true internal/outboundtrust/public_address.go
  assert_template_required true internal/infra/postgreswebhook/dispatcher.go
  assert_template_required true migrations/000007_postgres_webhooks_retire.sql
  assert_template_required true cmd/jobs-worker/builder.go
  assert_template_required true test/webhook_network_integration_test.go
  assert_template_required true docs/outbound-webhook-delivery.md
  assert_template_scope true '["webhooks"]' false internal/outboundtrust/public_address.go
  # profile:webhooks-durable:end
  # profile:outbox-postgres:start
  assert_template_scope true '["outbox"]' false internal/domainevent/event.go
  # profile:outbox-postgres:end
  assert_template_required true test/postgres_integration_test.go
  assert_template_required true go.mod
  assert_template_required true .github/workflows/ci.yml
  assert_template_required true assets/logo.png
  assert_template_required true internal/greeting/service.go Makefile
  assert_template_scope true "${all_init}" true scripts/init-module.sh
  assert_template_scope true "${all_init}" true .github/workflows/ci.yml
  assert_template_scope true "${all_init}" true assets/logo.png
  assert_template_scope true '["minimal"]' false .grok/roles/worker-agent.toml
  assert_template_scope true '["minimal"]' false template-owned.paths
  assert_template_scope true '["minimal"]' false \
    internal/greeting/service.go .agents/roles/worker-agent.toml
  assert_template_scope false '[]' false scripts/ci/ci-change-scope.sh
  assert_template_scope false '[]' false evals/instructions/evals.json
  echo "CI change-scope routing check passed"
}

case "${1:-}" in
  classify)
    classify
    ;;
  template-required)
    template_required
    ;;
  template-scope)
    template_scope
    ;;
  self-test)
    self_test
    ;;
  *)
    echo "usage: $0 {classify|template-required|template-scope|self-test}" >&2
    exit 2
    ;;
esac
