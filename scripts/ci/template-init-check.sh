#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
TEMP_ROOT="$(mktemp -d -t template-init-check.XXXXXX)"
export GOCACHE="${GOCACHE:-${ROOT_DIR}/.cache/go-build}"
export GOLANGCI_LINT_CACHE="${GOLANGCI_LINT_CACHE:-${ROOT_DIR}/.cache/golangci-lint}"
mkdir -p "${GOCACHE}" "${GOLANGCI_LINT_CACHE}"
LINTER="$(go -C "${ROOT_DIR}/tools" tool -n golangci-lint)"
trap 'rm -rf "${TEMP_ROOT}"' EXIT

new_fixture() {
	local name="$1"
	local origin="${2:-}"
	local existing_env="${3:-false}"
	local root="${TEMP_ROOT}/${name}"

	mkdir -p \
		"${root}/.github" \
		"${root}/api/openapi" \
		"${root}/cmd/service/internal/bootstrap" \
		"${root}/env" \
		"${root}/internal/config" \
		"${root}/internal/example" \
		"${root}/internal/health" \
		"${root}/internal/infra/example" \
		"${root}/scripts/profiles/database-none" \
		"${root}/tools"

	{
		echo "module github.com/example/go-service-template-rest"
		echo
		echo "go 1.26"
	} >"${root}/go.mod"
	{
		echo "module github.com/example/go-service-template-rest/tools"
		echo
		echo "go 1.26"
	} >"${root}/tools/go.mod"
	cp "${ROOT_DIR}/.golangci.yml" "${root}/.golangci.yml"
	cp "${ROOT_DIR}/.github/CODEOWNERS" "${root}/.github/CODEOWNERS"
	cp "${ROOT_DIR}/api/openapi/service.yaml" "${root}/api/openapi/service.yaml"
	cp "${ROOT_DIR}/env/.env.example" "${root}/env/.env.example"
	printf 'SERVICE_NAME := service\nSERVICE_CMD := ./cmd/service\n' >"${root}/Makefile"
	printf '# Template README\nhttps://github.com/Dankosik/go-service-template-rest\n' >"${root}/README.md"
	printf 'package config\n\ntype Config struct { Postgres PostgresConfig }\ntype PostgresConfig struct { Enabled bool }\n\nvar defaults = map[string]any{\n\t"observability.otel.service_name":           "service",\n}\n' \
		>"${root}/internal/config/defaults.go"
	printf 'package bootstrap\n\nimport "github.com/example/go-service-template-rest/internal/config"\n\ntype startupBootstrap struct{ cfg config.Config }\n\nvar bootstrapIdentity = []string{"service.name", "service"}\n' \
		>"${root}/cmd/service/internal/bootstrap/run.go"
	printf 'package health\n\ntype Service struct{}\n\nfunc New() *Service { return &Service{} }\n' \
		>"${root}/internal/health/service.go"
	cp \
		"${ROOT_DIR}/scripts/profiles/database-none/startup_dependencies.go.tmpl" \
		"${root}/scripts/profiles/database-none/startup_dependencies.go.tmpl"
	mkdir -p "${root}/.agents" "${root}/.codex" "${root}/.claude" "${root}/.qwen" "${root}/specs"
	touch "${root}/.agents/fixture" "${root}/.codex/fixture" "${root}/.claude/fixture" "${root}/.qwen/fixture" "${root}/specs/fixture"
	printf '# Full agent contract\n' >"${root}/AGENTS.md"
	printf '# Claude\n' >"${root}/CLAUDE.md"
	printf '# Qwen\n' >"${root}/QWEN.md"

	{
		echo "package example"
		echo
		echo 'const Module = "github.com/example/go-service-template-rest"'
	} >"${root}/internal/example/example.go"
	echo "package example" >"${root}/internal/infra/example/example.go"

	if [[ "${existing_env}" == true ]]; then
		echo "PRESERVE_ME=1" >"${root}/.env"
	fi

	git -C "${root}" init -q
	if [[ -n "${origin}" ]]; then
		git -C "${root}" remote add origin "${origin}"
	fi
	printf '%s\n' "${root}"
}

snapshot() {
	local root="$1"
	(
		cd "${root}"
		find . -type f -not -path './.git/*' -exec shasum -a 256 {} + |
			LC_ALL=C sort
	)
}

copy_template_checkout() {
	local name="$1"
	local origin="$2"
	local root="${TEMP_ROOT}/${name}"

	mkdir -p "${root}"
	while IFS= read -r -d '' file; do
		[[ -f "${ROOT_DIR}/${file}" || -L "${ROOT_DIR}/${file}" ]] || continue
		mkdir -p "${root}/$(dirname "${file}")"
		cp -P "${ROOT_DIR}/${file}" "${root}/${file}"
	done < <(git -C "${ROOT_DIR}" ls-files -z --cached --others --exclude-standard)

	git -C "${root}" init -q
	git -C "${root}" remote add origin "${origin}"
	printf '%s\n' "${root}"
}

expect_unchanged_failure() {
	local root="$1"
	shift
	local before after

	before="$(snapshot "${root}")"
	if (
		cd "${root}"
		"$@"
	); then
		echo "expected template initialization to fail: $*"
		exit 1
	fi
	after="$(snapshot "${root}")"
	[[ "${before}" == "${after}" ]] || {
		echo "failed initialization mutated fixture ${root}"
		diff <(printf '%s\n' "${before}") <(printf '%s\n' "${after}") || true
		exit 1
	}
}

derived="$(new_fixture derived git@github.com:acme/orders.git true)"
env_before="$(shasum -a 256 "${derived}/.env")"
(
	cd "${derived}"
	CODEOWNER=@acme/platform bash "${ROOT_DIR}/scripts/init-module.sh"
)

grep -Fqx "module github.com/acme/orders" "${derived}/go.mod"
grep -Fqx "module github.com/acme/orders/tools" "${derived}/tools/go.mod"
! grep -R -Fq "github.com/example/go-service-template-rest" \
	"${derived}/internal" "${derived}/.golangci.yml"
! grep -v '^[[:space:]]*#' "${derived}/.github/CODEOWNERS" | grep -Fq "@Dankosik"
grep -v '^[[:space:]]*#' "${derived}/.github/CODEOWNERS" | grep -Fq "@acme/platform"
grep -Fqx '  title: "orders"' "${derived}/api/openapi/service.yaml"
grep -Fqx 'SERVICE_NAME := orders' "${derived}/Makefile"
grep -Fq '"observability.otel.service_name":           "orders"' "${derived}/internal/config/defaults.go"
grep -Fq '"service.name", "orders"' "${derived}/cmd/service/internal/bootstrap/run.go"
grep -Fqx 'APP__OBSERVABILITY__OTEL__SERVICE_NAME=orders' "${derived}/env/.env.example"
grep -Fq '# orders' "${derived}/README.md"
! grep -Fq 'https://github.com/Dankosik/go-service-template-rest/actions' "${derived}/README.md"
grep -Fq 'Keep production Go under `internal/`' "${derived}/AGENTS.md"
for removed in .agents .codex .claude .qwen specs CLAUDE.md QWEN.md; do
	[[ ! -e "${derived}/${removed}" ]]
done
[[ -f "${derived}/cmd/service/internal/bootstrap/startup_dependencies.go" ]]
[[ ! -e "${derived}/scripts/profiles/database-none" ]]
[[ "${env_before}" == "$(shasum -a 256 "${derived}/.env")" ]]

full="$(new_fixture full git@github.com:acme/payments.git)"
(
	cd "${full}"
	CODEOWNER=@acme/platform DATABASE=postgres AGENT_WORKFLOW=full \
		bash "${ROOT_DIR}/scripts/init-module.sh"
)
for retained in .agents .codex .claude .qwen specs CLAUDE.md QWEN.md; do
	[[ -e "${full}/${retained}" ]]
done

{
	echo "package example"
	echo
	echo 'import _ "github.com/acme/orders/internal/infra/example"'
} >"${derived}/internal/example/forbidden.go"
if (
	cd "${derived}"
	"${LINTER}" run --enable-only=depguard ./... >"${TEMP_ROOT}/depguard.log" 2>&1
); then
	echo "depguard accepted a feature-to-infra import after module initialization"
	exit 1
fi
grep -Fq "depguard" "${TEMP_ROOT}/depguard.log" || {
	cat "${TEMP_ROOT}/depguard.log"
	exit 1
}

source_checkout="$(new_fixture source git@github.com:Dankosik/go-service-template-rest.git true)"
source_before="$(snapshot "${source_checkout}")"
(
	cd "${source_checkout}"
	bash "${ROOT_DIR}/scripts/init-module.sh"
)
[[ "${source_before}" == "$(snapshot "${source_checkout}")" ]] || {
	echo "template source checkout changed without CODEOWNER"
	exit 1
}

missing_owner="$(new_fixture missing-owner git@github.com:acme/missing-owner.git)"
expect_unchanged_failure "${missing_owner}" \
	env -u CODEOWNER bash "${ROOT_DIR}/scripts/init-module.sh"

missing_module="$(new_fixture missing-module)"
expect_unchanged_failure "${missing_module}" \
	env CODEOWNER=@acme/platform bash "${ROOT_DIR}/scripts/init-module.sh"

malformed_module="$(new_fixture malformed-module git@github.com:acme/malformed-module.git)"
expect_unchanged_failure "${malformed_module}" \
	env CODEOWNER=@acme/platform bash "${ROOT_DIR}/scripts/init-module.sh" "bad module"

malformed_owner="$(new_fixture malformed-owner git@github.com:acme/malformed-owner.git)"
expect_unchanged_failure "${malformed_owner}" \
	env CODEOWNER=acme/platform bash "${ROOT_DIR}/scripts/init-module.sh"

minimal_checkout="$(copy_template_checkout full-minimal git@github.com:acme/feature-proof.git)"
(
	cd "${minimal_checkout}"
	CODEOWNER=@acme/platform DATABASE=none AGENT_WORKFLOW=none bash ./scripts/init-module.sh
	go test ./...
	go build ./cmd/service
	make mod-check
	if APP__POSTGRES__ENABLED=true \
		APP__POSTGRES__DSN='postgres://app:app@127.0.0.1:5432/app?sslmode=disable' \
		go run ./cmd/service >"${TEMP_ROOT}/minimal-postgres.log" 2>&1; then
		echo "database-none service accepted enabled PostgreSQL"
		exit 1
	fi
)
grep -Fq 'postgres is not included in the DATABASE=none profile' "${TEMP_ROOT}/minimal-postgres.log"
for removed in \
	cmd/migrate \
	internal/infra/httpclient \
	internal/infra/postgres \
	internal/infra/postgresmigrate \
	env/docker-compose.yml \
	test/postgres_integration_test.go \
	test/postgres_migrate_runner_integration_test.go; do
	[[ ! -e "${minimal_checkout}/${removed}" ]]
done
! make -C "${minimal_checkout}" help | grep -Fq 'bench-db'
! grep -Fq 'preDeployCommand = ["/migrate"]' "${minimal_checkout}/railway.toml"
! grep -Fq '/out/migrate' "${minimal_checkout}/build/docker/Dockerfile"
! grep -Fq 'migration-validate:' "${minimal_checkout}/.github/workflows/ci.yml"
! grep -Fq 'APP__POSTGRES__ENABLED' "${minimal_checkout}/env/.env.example"
if (
	cd "${minimal_checkout}"
	go list -m all |
		grep -E 'github.com/(exaring/otelpgx|golang-migrate/migrate|jackc/pgx|testcontainers/testcontainers-go)'
); then
	echo "database-none profile retained PostgreSQL runtime dependencies"
	exit 1
fi
(
	cd "${minimal_checkout}"
	git apply "${ROOT_DIR}/scripts/ci/fixtures/first-feature.patch"
	make openapi-generate
	make openapi-lint openapi-validate
	gofmt -w internal/greeting internal/infra/http cmd/service/internal/bootstrap/run.go
	go test ./internal/greeting ./internal/infra/http ./cmd/service/internal/bootstrap
)

postgres_checkout="$(copy_template_checkout full-postgres git@github.com:acme/postgres-service.git)"
(
	cd "${postgres_checkout}"
	CODEOWNER=@acme/platform DATABASE=postgres OUTBOUND_HTTP=bounded AGENT_WORKFLOW=full bash ./scripts/init-module.sh
	go test ./...
	go build ./cmd/service ./cmd/migrate
)
for retained in \
	cmd/migrate \
	internal/infra/httpclient \
	internal/infra/postgres \
	internal/infra/postgresmigrate \
	env/docker-compose.yml; do
	[[ -e "${postgres_checkout}/${retained}" ]]
done

malformed_outbound="$(new_fixture malformed-outbound git@github.com:acme/malformed-outbound.git)"
expect_unchanged_failure "${malformed_outbound}" \
	env CODEOWNER=@acme/platform OUTBOUND_HTTP=custom bash "${ROOT_DIR}/scripts/init-module.sh"
if (
	cd "${postgres_checkout}"
	go list -deps ./cmd/service | grep -F 'github.com/golang-migrate/migrate'
); then
	echo "service dependency graph includes migration implementation"
	exit 1
fi

echo "template initialization contract passed"
