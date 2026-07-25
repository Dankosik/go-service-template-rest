#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"

# assert replaces a bare `[[ ... ]]` line. Under `set -e`, bash 3.2 — still the
# /bin/bash macOS ships — does not abort on a failing bare conditional, so those
# assertions silently passed locally and only failed on CI's bash 5. Routing them
# through a command that exits also gives the failure a name instead of an
# unexplained exit 1.
assert() {
	local description="$1"
	shift
	if ! "$@"; then
		echo "template initialization contract: ${description}"
		exit 1
	fi
}

path_absent() { [[ ! -e "$1" ]]; }
path_present() { [[ -e "$1" ]]; }
file_present() { [[ -f "$1" ]]; }
same_text() { [[ "$1" == "$2" ]]; }

# This check drives scripts/init-module.sh against fixtures, so it only means
# anything in the template source checkout. Initialization consumes and removes
# scripts/profiles/, so a generated service inherits a CI step with no generator
# left to verify. Say so and succeed: a service that owns no generator has not
# broken the generator contract, and failing here would make the first push of
# every generated repository red.
if [[ ! -d "${ROOT_DIR}/scripts/profiles" ]]; then
	echo "template initialization contract is upstream-only; scripts/profiles/ is absent, so this checkout is a generated service"
	exit 0
fi

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

workflow_snapshot() {
	local root="$1"
	(
		cd "${root}"
		{
			for path in .agents .codex .claude .qwen; do
				if [[ ! -e "${path}" && ! -L "${path}" ]]; then
					printf 'missing %s\n' "${path}"
					continue
				fi
				find "${path}" -type d -print | while IFS= read -r entry; do
					printf 'directory %s\n' "${entry}"
				done
				find "${path}" -type l -print | while IFS= read -r entry; do
					printf 'symlink %s -> %s\n' "${entry}" "$(readlink "${entry}")"
				done
				find "${path}" -type f -exec shasum -a 256 {} +
			done
			for path in AGENTS.md CLAUDE.md QWEN.md; do
				if [[ -f "${path}" ]]; then
					shasum -a 256 "${path}"
				else
					printf 'missing %s\n' "${path}"
				fi
			done
		} | LC_ALL=C sort
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
	# A repository created from the template has history, and initialization
	# records the revision it derived from. An empty commit gives the fixture a
	# HEAD to resolve without paying to stage the whole tree.
	git -C "${root}" -c user.email=template-init-check@example.com -c user.name=template-init-check \
		commit -q --allow-empty -m "template checkout"
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
derived_workflow_before="$(workflow_snapshot "${derived}")"
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
assert "agent workflow changed during initialization" same_text "${derived_workflow_before}" "$(workflow_snapshot "${derived}")"
assert "specs/ must not survive initialization" path_absent "${derived}/specs"
assert "startup_dependencies.go is missing" file_present "${derived}/cmd/service/internal/bootstrap/startup_dependencies.go"
assert "scripts/profiles/ must not survive initialization" path_absent "${derived}/scripts/profiles"
assert "an existing .env was rewritten" same_text "${env_before}" "$(shasum -a 256 "${derived}/.env")"

full="$(new_fixture full git@github.com:acme/payments.git)"
full_workflow_before="$(workflow_snapshot "${full}")"
(
	cd "${full}"
	CODEOWNER=@acme/platform DATABASE=postgres \
		bash "${ROOT_DIR}/scripts/init-module.sh"
)
assert "agent workflow changed during postgres initialization" same_text "${full_workflow_before}" "$(workflow_snapshot "${full}")"
assert "specs/ must not survive postgres initialization" path_absent "${full}/specs"

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
minimal_workflow_before="$(workflow_snapshot "${minimal_checkout}")"
(
	cd "${minimal_checkout}"
	CODEOWNER=@acme/platform DATABASE=none bash ./scripts/init-module.sh
	go test ./...
	go build ./cmd/service
	make mod-check
	# Lint the generated profile, not just the template. Profile file removal can
	# strand symbols whose only consumer was removed, which builds and tests
	# cannot catch.
	"${LINTER}" run --allow-parallel-runners --timeout=3m ./...
	if APP__POSTGRES__ENABLED=true \
		APP__POSTGRES__DSN='postgres://app:app@127.0.0.1:5432/app?sslmode=disable' \
		go run ./cmd/service >"${TEMP_ROOT}/minimal-postgres.log" 2>&1; then
		echo "database-none service accepted enabled PostgreSQL"
		exit 1
	fi
)
assert "agent workflow changed during minimal initialization" same_text "${minimal_workflow_before}" "$(workflow_snapshot "${minimal_checkout}")"
# This profile carries no PostgreSQL configuration at all, so an APP__POSTGRES__*
# variable is an unknown key rather than a runtime feature check. That is the
# stronger rejection: it names every key it refused and it happens before any
# dependency wiring runs.
grep -Fq 'unknown_key' "${TEMP_ROOT}/minimal-postgres.log"
grep -Fq 'postgres.enabled' "${TEMP_ROOT}/minimal-postgres.log"
grep -Fq 'postgres.dsn' "${TEMP_ROOT}/minimal-postgres.log"
# No PostgreSQL configuration surface survives in the generated Go sources.
if grep -rn 'Postgres\b' "${minimal_checkout}/internal/config" "${minimal_checkout}/cmd" --include='*.go'; then
	echo "database-none profile retained PostgreSQL configuration"
	exit 1
fi
if grep -rn 'profile:database-postgres' \
	"${minimal_checkout}/cmd" "${minimal_checkout}/internal" "${minimal_checkout}/env" \
	"${minimal_checkout}/.github" "${minimal_checkout}/build" \
	"${minimal_checkout}/Makefile" "${minimal_checkout}/railway.toml" 2>/dev/null; then
	echo "database-none profile left profile markers behind"
	exit 1
fi
for removed in \
	cmd/migrate \
	internal/infra/httpclient \
	internal/infra/postgres \
	internal/infra/postgresmigrate \
	env/docker-compose.yml \
	test/postgres_integration_test.go \
	test/postgres_migrate_runner_integration_test.go \
	.github/assets \
	.github/ISSUE_TEMPLATE; do
	assert "${removed} must not survive DATABASE=none initialization" path_absent "${minimal_checkout}/${removed}"
done
# The generated service must record where it came from, or a later upstream fix
# has no revision to be reviewed against.
grep -Fq 'database = "none"' "${minimal_checkout}/template.lock"
grep -Fq 'outbound_http = "none"' "${minimal_checkout}/template.lock"
grep -Eq '^source_revision = "[0-9a-f]{40}"$' "${minimal_checkout}/template.lock"
# A generated service owns no generator, so the initialization contract check
# reports that and succeeds instead of failing the first push of every service.
#
# The output goes to a file rather than into `grep -q`: a matching `grep -q`
# closes the pipe on its first hit, and the SIGPIPE that follows makes `make`
# report a write error and exit non-zero under pipefail.
(
	cd "${minimal_checkout}"
	make template-init-check >"${TEMP_ROOT}/minimal-init-check.log"
	# CI runs this unconditionally; the target must survive DATABASE=none.
	make sqlc-check
	make project-structure-check
)
grep -Fq 'upstream-only' "${TEMP_ROOT}/minimal-init-check.log"
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
	# The health-only baseline needs no request binding helpers. The first
	# operation with a path or query parameter makes the generated code import
	# github.com/oapi-codegen/runtime, so the first feature tidies like any
	# other Go change that adds an import.
	go mod tidy
	make openapi-lint openapi-validate
	gofmt -w internal/greeting internal/infra/http cmd/service/internal/bootstrap/run.go
	go test ./internal/greeting ./internal/infra/http ./cmd/service/internal/bootstrap
	make openapi-check
)
# The default profile must not hand a generated service the reference example's
# packages, second OpenAPI contract, or second main().
assert "examples/ must not survive initialization" path_absent "${minimal_checkout}/examples"

postgres_checkout="$(copy_template_checkout full-postgres git@github.com:acme/postgres-service.git)"
postgres_workflow_before="$(workflow_snapshot "${postgres_checkout}")"
(
	cd "${postgres_checkout}"
	CODEOWNER=@acme/platform DATABASE=postgres OUTBOUND_HTTP=bounded REFERENCE_EXAMPLE=keep \
		bash ./scripts/init-module.sh
	go test ./...
	go build ./cmd/service ./cmd/migrate
)
# REFERENCE_EXAMPLE=keep is the opt-in escape hatch for teams that want the
# worked example in tree.
assert "REFERENCE_EXAMPLE=keep did not retain examples/" path_present "${postgres_checkout}/examples/reference-service"
assert "agent workflow changed during postgres+bounded initialization" same_text "${postgres_workflow_before}" "$(workflow_snapshot "${postgres_checkout}")"
assert "specs/ must not survive postgres+bounded initialization" path_absent "${postgres_checkout}/specs"
assert "scripts/profiles/ must not survive postgres initialization" path_absent "${postgres_checkout}/scripts/profiles"
for retained in \
	cmd/migrate \
	internal/infra/httpclient \
	internal/infra/postgres \
	internal/infra/postgresmigrate \
	env/docker-compose.yml; do
	assert "${retained} must survive DATABASE=postgres initialization" path_present "${postgres_checkout}/${retained}"
done
# A resolved profile leaves no markers in either direction. They are generator
# inputs, and a retained profile has consumed them just as a removed one has.
if grep -rn 'profile:database-postgres' \
	"${postgres_checkout}/cmd" "${postgres_checkout}/internal" "${postgres_checkout}/env" \
	"${postgres_checkout}/.github" "${postgres_checkout}/build" \
	"${postgres_checkout}/Makefile" "${postgres_checkout}/railway.toml" 2>/dev/null; then
	echo "database-postgres profile left profile markers behind"
	exit 1
fi
grep -Fq 'database = "postgres"' "${postgres_checkout}/template.lock"
grep -Fq 'outbound_http = "bounded"' "${postgres_checkout}/template.lock"
(
	cd "${postgres_checkout}"
	make template-init-check >"${TEMP_ROOT}/postgres-init-check.log"
	make project-structure-check
)
grep -Fq 'upstream-only' "${TEMP_ROOT}/postgres-init-check.log"

if [[ "${TEMPLATE_POSTGRES_PROOF:-0}" == "1" ]]; then
	(
		cd "${postgres_checkout}"
		git apply --recount "${ROOT_DIR}/scripts/ci/fixtures/postgres-post-feature.patch"
		make openapi-generate sqlc-generate
		# handlers.go is deliberately absent: the feature composes through
		# Handlers.API, so it adds files instead of editing shared template source.
		gofmt -w \
			internal/article \
			internal/infra/http/article_api.go \
			internal/infra/postgres/article_repository.go \
			cmd/service/internal/bootstrap/run.go \
			test/postgres_article_feature_integration_test.go
		go test ./internal/article ./internal/infra/http ./internal/infra/postgres
		REQUIRE_DOCKER=1 go test -tags=integration ./test \
			-run '^TestPostgresArticleVerticalSlice$' \
			-count=1
	)
fi

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
