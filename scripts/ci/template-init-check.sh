#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
TEMPLATE_INIT_PROFILE="${TEMPLATE_INIT_PROFILE:-all}"
valid_profiles="all, minimal, postgres, grpc, authn"
# profile:messaging-nats-jetstream:start
valid_profiles="${valid_profiles}, messaging"
# profile:messaging-nats-jetstream:end
# profile:outbox-postgres:start
valid_profiles="${valid_profiles}, outbox"
# profile:outbox-postgres:end

case "${TEMPLATE_INIT_PROFILE}" in
	all | minimal | postgres | grpc | authn) ;;
	# profile:messaging-nats-jetstream:start
	messaging) ;;
	# profile:messaging-nats-jetstream:end
	# profile:outbox-postgres:start
	outbox) ;;
	# profile:outbox-postgres:end
	*)
		echo "TEMPLATE_INIT_PROFILE must be one of: ${valid_profiles}" >&2
		exit 2
		;;
esac

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
# glob_absent fails when any path matches, so a named-file inventory cannot go
# stale as new files join a profile. It reports the matches it found, because
# the useful part of the failure is which file was retained.
glob_absent() {
	local matches
	matches="$(compgen -G "$1" || true)"
	if [[ -n "${matches}" ]]; then
		echo "retained: ${matches}"
		return 1
	fi
}
file_present() { [[ -f "$1" ]]; }
same_text() { [[ "$1" == "$2" ]]; }
grep_absent() {
	local status
	if grep "$@"; then
		return 1
	else
		status=$?
	fi
	[[ "${status}" -eq 1 ]]
}

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
# Go and golangci-lint normally own the cache paths restored by CI. Keep those
# canonical paths here; the generated fixture lint below documents its one
# narrow isolation exception.
if [[ "${TEMPLATE_INIT_PROFILE}" != "postgres" ]]; then
	if [[ -n "${GOLANGCI_LINT_BIN:-}" ]]; then
		LINTER="${GOLANGCI_LINT_BIN}"
	else
		LINTER="$(go -C "${ROOT_DIR}/tools" tool -n golangci-lint)"
	fi
fi
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
		"${root}/internal/infra/http" \
		"${root}/internal/problem" \
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
	# The DATABASE=none profile template imports these packages, so the fixture has to
	# carry them: without them the `go mod tidy` initialization ends up trying to
	# resolve the generated module path over the network and fails with a confusing
	# "Repository not found".
	printf 'package httpx\n' \
		>"${root}/internal/infra/http/http.go"
	printf 'package problem\n\ntype Mapped struct{ Code string }\n\ntype Mapper func(error) (Mapped, bool)\n' \
		>"${root}/internal/problem/problem.go"
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

if [[ "${TEMPLATE_INIT_PROFILE}" != "postgres" ]]; then
	derived="$(new_fixture derived git@github.com:acme/orders.git true)"
	env_before="$(shasum -a 256 "${derived}/.env")"
	derived_workflow_before="$(workflow_snapshot "${derived}")"
	(
		cd "${derived}"
		CODEOWNER=@acme/platform bash "${ROOT_DIR}/scripts/init-module.sh"
	)

grep -Fqx "module github.com/acme/orders" "${derived}/go.mod"
grep -Fqx "module github.com/acme/orders/tools" "${derived}/tools/go.mod"
assert "template module survived initialization" grep_absent -R -Fq \
	"github.com/example/go-service-template-rest" "${derived}/internal" "${derived}/.golangci.yml"
if grep -v '^[[:space:]]*#' "${derived}/.github/CODEOWNERS" | grep -Fq "@Dankosik"; then
	echo "template initialization contract: template CODEOWNER survived initialization"
	exit 1
fi
grep -v '^[[:space:]]*#' "${derived}/.github/CODEOWNERS" | grep -Fq "@acme/platform"
grep -Fqx '  title: "orders"' "${derived}/api/openapi/service.yaml"
grep -Fqx 'SERVICE_NAME := orders' "${derived}/Makefile"
grep -Fq '"observability.otel.service_name":           "orders"' "${derived}/internal/config/defaults.go"
grep -Fq '"service.name", "orders"' "${derived}/cmd/service/internal/bootstrap/run.go"
grep -Fqx 'APP__OBSERVABILITY__OTEL__SERVICE_NAME=orders' "${derived}/env/.env.example"
grep -Fq '# orders' "${derived}/README.md"
assert "template actions URL survived initialization" grep_absent -Fq \
	'https://github.com/Dankosik/go-service-template-rest/actions' "${derived}/README.md"
assert "agent workflow changed during initialization" same_text "${derived_workflow_before}" "$(workflow_snapshot "${derived}")"
assert "specs/ must not survive initialization" path_absent "${derived}/specs"
assert "startup_dependencies.go is missing" file_present "${derived}/cmd/service/internal/bootstrap/startup_dependencies.go"
assert "scripts/profiles/ must not survive initialization" path_absent "${derived}/scripts/profiles"
assert "an existing .env was rewritten" same_text "${env_before}" "$(shasum -a 256 "${derived}/.env")"

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

# profile:messaging-nats-jetstream:start
malformed_messaging="$(new_fixture malformed-messaging git@github.com:acme/malformed-messaging.git)"
expect_unchanged_failure "${malformed_messaging}" \
	env CODEOWNER=@acme/platform MESSAGING=custom bash "${ROOT_DIR}/scripts/init-module.sh"
empty_messaging="$(new_fixture empty-messaging git@github.com:acme/empty-messaging.git)"
expect_unchanged_failure "${empty_messaging}" \
	env CODEOWNER=@acme/platform MESSAGING= bash "${ROOT_DIR}/scripts/init-module.sh"
# profile:messaging-nats-jetstream:end

# profile:outbox-postgres:start
malformed_outbox="$(new_fixture malformed-outbox git@github.com:acme/malformed-outbox.git)"
expect_unchanged_failure "${malformed_outbox}" \
	env CODEOWNER=@acme/platform OUTBOX=custom bash "${ROOT_DIR}/scripts/init-module.sh"
empty_outbox="$(new_fixture empty-outbox git@github.com:acme/empty-outbox.git)"
expect_unchanged_failure "${empty_outbox}" \
	env CODEOWNER=@acme/platform OUTBOX= bash "${ROOT_DIR}/scripts/init-module.sh"
invalid_outbox_database="$(new_fixture invalid-outbox-database git@github.com:acme/invalid-outbox-database.git)"
expect_unchanged_failure "${invalid_outbox_database}" \
	env CODEOWNER=@acme/platform DATABASE=none OUTBOX=postgres bash "${ROOT_DIR}/scripts/init-module.sh"
# profile:outbox-postgres:end

minimal_checkout="$(copy_template_checkout full-minimal git@github.com:acme/feature-proof.git)"
minimal_source_revision="$(git -C "${minimal_checkout}" rev-parse HEAD)"
minimal_workflow_before="$(workflow_snapshot "${minimal_checkout}")"
(
	cd "${minimal_checkout}"
	CODEOWNER=@acme/platform DATABASE=none bash ./scripts/init-module.sh
	go test ./...
	go build ./cmd/service
	make mod-tidy-check
		# One full linter load proves both that profile removal stranded no symbols
		# and that initialization rewrote depguard's module-qualified rules. The
		# intentional violation must be the only reported issue. This generated
		# checkout reuses one module path under a fresh absolute temp path on every
		# run. Isolate only this invocation: shared golangci-lint cache entries can
		# otherwise return findings whose source paths belong to an already deleted
		# checkout.
		mkdir -p internal/depguardprobe
		mkdir -p "${TEMP_ROOT}/golangci-lint-cache"
		{
			echo "package depguardprobe"
			echo
			echo 'import _ "github.com/acme/feature-proof/internal/infra/http"'
		} >internal/depguardprobe/forbidden.go
		if GOLANGCI_LINT_CACHE="${TEMP_ROOT}/golangci-lint-cache" "${LINTER}" run \
		--allow-serial-runners \
		--timeout=3m \
		--show-stats=false \
		--output.text.colors=false \
		--output.text.print-issued-lines=false \
		--output.text.path="${TEMP_ROOT}/minimal-lint.log" \
		./...; then
		echo "depguard accepted a feature-to-infra import after module initialization"
		exit 1
	fi
	issue_count="$(grep -Ec '^[^:]+:[0-9]+:[0-9]+: .+ \([^()]+\)$' "${TEMP_ROOT}/minimal-lint.log" || true)"
	if [[ "${issue_count}" != 1 ]] || ! grep -Fq '(depguard)' "${TEMP_ROOT}/minimal-lint.log"; then
		cat "${TEMP_ROOT}/minimal-lint.log"
		echo "generated minimal profile must have exactly the intentional depguard issue"
		exit 1
	fi
	rm -rf internal/depguardprobe
	if APP__POSTGRES__ENABLED=true \
		APP__POSTGRES__DSN='postgres://app:app@127.0.0.1:5432/app?sslmode=disable' \
		go run ./cmd/service >"${TEMP_ROOT}/minimal-postgres.log" 2>&1; then
		echo "database-none service accepted enabled PostgreSQL"
		exit 1
	fi
	# profile:messaging-nats-jetstream:start
	if APP__MESSAGING__ENABLED=true \
		APP__MESSAGING__URLS='nats://127.0.0.1:4222' \
		go run ./cmd/service >"${TEMP_ROOT}/minimal-messaging.log" 2>&1; then
		echo "messaging-none service accepted messaging configuration"
		exit 1
	fi
	# profile:messaging-nats-jetstream:end
)
assert "agent workflow changed during minimal initialization" same_text "${minimal_workflow_before}" "$(workflow_snapshot "${minimal_checkout}")"
# This profile carries no PostgreSQL configuration at all, so an APP__POSTGRES__*
# variable is an unknown key rather than a runtime feature check. That is the
# stronger rejection: it names every key it refused and it happens before any
# dependency wiring runs.
grep -Fq 'unknown_key' "${TEMP_ROOT}/minimal-postgres.log"
grep -Fq 'postgres.enabled' "${TEMP_ROOT}/minimal-postgres.log"
grep -Fq 'postgres.dsn' "${TEMP_ROOT}/minimal-postgres.log"
# profile:messaging-nats-jetstream:start
grep -Fq 'unknown_key' "${TEMP_ROOT}/minimal-messaging.log"
grep -Fq 'messaging.enabled' "${TEMP_ROOT}/minimal-messaging.log"
grep -Fq 'messaging.urls' "${TEMP_ROOT}/minimal-messaging.log"
# profile:messaging-nats-jetstream:end
# profile:outbox-postgres:start
for removed in \
	cmd/outbox-relay \
	docs/postgres-transactional-outbox.md \
	internal/config/outbox_config_test.go \
	internal/infra/postgresoutbox \
	test/postgres_outbox_integration_test.go \
	test/postgres_outbox_natsjs_integration_test.go; do
	assert "${removed} must not survive OUTBOX=none initialization" path_absent "${minimal_checkout}/${removed}"
done
# profile:outbox-postgres:end
for removed in \
	buf.yaml \
	buf.gen.yaml \
	cmd/migrate \
	cmd/service/internal/bootstrap/startup_grpc.go \
	cmd/service/internal/bootstrap/startup_grpc_test.go \
	cmd/service/internal/bootstrap/startup_authn.go \
	docs/authentication.md \
	docs/grpc.md \
	internal/config/authn_config_test.go \
	internal/config/grpc_config_test.go \
	internal/infra/oidcjwt \
	internal/infra/httpclient \
	internal/infra/grpc \
	internal/infra/grpcclient \
	internal/infra/postgres \
	internal/infra/postgresmigrate \
	scripts/ci/migration-source-check.sh \
	scripts/ci/migration-history-check.sh \
	scripts/ci/migration-check-self-test.sh \
	scripts/ci/migration-image-history-check.sh \
	scripts/ci/migration-publication-check.sh \
	scripts/dev/benchmark-grpc-check.sh \
	scripts/proto.sh \
	scripts/run-buf.sh \
	scripts/ci/proto-check.sh \
	env/docker-compose.yml \
	test/grpc_process_integration_test.go \
	test/performance/grpc \
	test/postgres_integration_test.go \
	test/postgres_migrate_runner_integration_test.go \
	.github/assets \
	.github/ISSUE_TEMPLATE; do
	assert "${removed} must not survive DATABASE=none initialization" path_absent "${minimal_checkout}/${removed}"
done
# profile:messaging-nats-jetstream:start
for removed in \
	internal/infra/natsjs \
	cmd/worker \
	cmd/service/internal/bootstrap/startup_messaging.go \
	cmd/service/internal/bootstrap/startup_messaging_test.go \
	docs/durable-messaging.md \
	internal/config/messaging.go \
	internal/config/messaging_test.go \
	test/nats_messaging_integration_test.go; do
	assert "${removed} must not survive MESSAGING=none initialization" path_absent "${minimal_checkout}/${removed}"
done
# profile:messaging-nats-jetstream:end
# profile:outbox-postgres:start
grep -Fq 'outbox = "none"' "${minimal_checkout}/template.lock"
# profile:outbox-postgres:end
for benchmark_surface in \
	Makefile \
	scripts/dev/benchmark.sh \
	docs/benchmarking.md \
	docs/build-test-and-development-commands.md; do
	assert "GRPC=none retained gRPC reference benchmark commands in ${benchmark_surface}" \
		grep_absent -Eq 'bench-grpc|GRPC_BENCH|grpc-smoke|grpc-inspect' \
		"${minimal_checkout}/${benchmark_surface}"
done
# The generated service must record where it came from, or a later upstream fix
# has no revision to be reviewed against.
grep -Fq 'database = "none"' "${minimal_checkout}/template.lock"
grep -Fq 'grpc = "none"' "${minimal_checkout}/template.lock"
grep -Fq 'authn = "none"' "${minimal_checkout}/template.lock"
grep -Fq 'outbound_http = "none"' "${minimal_checkout}/template.lock"
# profile:messaging-nats-jetstream:start
grep -Fq 'messaging = "none"' "${minimal_checkout}/template.lock"
# profile:messaging-nats-jetstream:end
# profile:outbox-postgres:start
minimal_outbox_snapshot="$(snapshot "${minimal_checkout}")"
(
	cd "${minimal_checkout}"
	CODEOWNER=@acme/platform DATABASE=none OUTBOX=none bash "${ROOT_DIR}/scripts/init-module.sh"
)
assert "explicit OUTBOX=none changed the default-none checkout" \
	same_text "${minimal_outbox_snapshot}" "$(snapshot "${minimal_checkout}")"
expect_unchanged_failure "${minimal_checkout}" \
	env CODEOWNER=@acme/platform DATABASE=postgres OUTBOX=postgres bash "${ROOT_DIR}/scripts/init-module.sh"
# profile:outbox-postgres:end
grep -Fqx "source_revision = \"${minimal_source_revision}\"" "${minimal_checkout}/template.lock"
# profile:messaging-nats-jetstream:start
expect_unchanged_failure "${minimal_checkout}" \
	env CODEOWNER=@acme/platform MESSAGING=nats-jetstream bash "${ROOT_DIR}/scripts/init-module.sh"
# profile:messaging-nats-jetstream:end
# profile:outbox-postgres:start
if make -C "${minimal_checkout}" help | grep -Fq 'outbox'; then
	echo "template initialization contract: OUTBOX=none help retained outbox commands"
	exit 1
fi
# profile:outbox-postgres:end
# A generated service owns no generator, so the initialization contract check
# reports that and succeeds instead of failing the first push of every service.
#
# The output goes to a file rather than into `grep -q`: a matching `grep -q`
# closes the pipe on its first hit, and the SIGPIPE that follows makes `make`
# report a write error and exit non-zero under pipefail.
(
	cd "${minimal_checkout}"
	TEMPLATE_INIT_PROFILE=minimal make template-init-check >"${TEMP_ROOT}/minimal-init-check.log"
	# CI runs this unconditionally; the target must survive DATABASE=none.
	make sqlc-check
	make proto-check
	make project-structure-check
)
grep -Fq 'upstream-only' "${TEMP_ROOT}/minimal-init-check.log"
if make -C "${minimal_checkout}" help | grep -Fq 'bench-db'; then
	echo "template initialization contract: DATABASE=none help retained database commands"
	exit 1
fi
# profile:messaging-nats-jetstream:start
if make -C "${minimal_checkout}" help | grep -Fq 'worker'; then
	echo "template initialization contract: MESSAGING=none help retained worker commands"
	exit 1
fi
# profile:messaging-nats-jetstream:end
# profile:outbox-postgres:start
assert "OUTBOX=none retained the relay image binary" grep_absent -Fq \
	'/out/outbox-relay' "${minimal_checkout}/build/docker/Dockerfile"
assert "OUTBOX=none retained runtime, documentation, test, config, or CI wiring" grep_absent -R -Eq \
	'postgresoutbox|outbox-relay|APP__OUTBOX__|OUTBOX_RACE|test-outbox|postgres-transactional-outbox|profile:outbox-postgres' \
	"${minimal_checkout}/Makefile" \
	"${minimal_checkout}/README.md" \
	"${minimal_checkout}/.golangci.yml" \
	"${minimal_checkout}/.github" \
	"${minimal_checkout}/build" \
	"${minimal_checkout}/cmd" \
	"${minimal_checkout}/docs" \
	"${minimal_checkout}/env" \
	"${minimal_checkout}/internal" \
	"${minimal_checkout}/scripts/ci" \
	"${minimal_checkout}/test"
# profile:outbox-postgres:end
assert "DATABASE=none retained the migration deploy command" grep_absent -Fq \
	'preDeployCommand = ["/migrate"]' "${minimal_checkout}/railway.toml"
assert "DATABASE=none retained the migration binary" grep_absent -Fq \
	'/out/migrate' "${minimal_checkout}/build/docker/Dockerfile"
# profile:messaging-nats-jetstream:start
assert "MESSAGING=none retained the worker binary" grep_absent -Fq \
	'/out/worker' "${minimal_checkout}/build/docker/Dockerfile"
assert "MESSAGING=none retained runtime, documentation, or CI wiring" grep_absent -R -Eq \
	'nats-jetstream|natsjs|cmd/worker|build-worker|run-worker|APP__MESSAGING__|TEMPLATE_INIT_PROFILE.*messaging|durable-messaging|messaging-race' \
	"${minimal_checkout}/Makefile" \
	"${minimal_checkout}/README.md" \
	"${minimal_checkout}/.github" \
	"${minimal_checkout}/build" \
	"${minimal_checkout}/cmd" \
	"${minimal_checkout}/docs" \
	"${minimal_checkout}/env" \
	"${minimal_checkout}/internal" \
	"${minimal_checkout}/railway.toml" \
	"${minimal_checkout}/scripts/ci" \
	"${minimal_checkout}/scripts/dev" \
	"${minimal_checkout}/test"
# profile:messaging-nats-jetstream:end
assert "DATABASE=none retained migration validation" grep_absent -Fq \
	'migration-validate:' "${minimal_checkout}/.github/workflows/ci.yml"
assert "DATABASE=none retained migration Make targets" grep_absent -Fq \
	'migration-check:' "${minimal_checkout}/Makefile"
assert "DATABASE=none retained PostgreSQL configuration" grep_absent -Fq \
	'APP__POSTGRES__ENABLED' "${minimal_checkout}/env/.env.example"
if (
	cd "${minimal_checkout}"
	go list -m all |
		grep -E 'github.com/(exaring/otelpgx|jackc/pgx|pressly/goose|testcontainers/testcontainers-go)'
); then
	echo "database-none profile retained PostgreSQL runtime dependencies"
	exit 1
fi
# profile:messaging-nats-jetstream:start
if (
	cd "${minimal_checkout}"
	go list -m all | grep -Fq 'github.com/nats-io/'
); then
	echo "messaging-none profile retained NATS dependencies"
	exit 1
fi
# profile:messaging-nats-jetstream:end
if (
	cd "${minimal_checkout}"
	go -C tools tool -n goose >/dev/null 2>&1
); then
	echo "database-none profile retained the Goose CLI tool"
	exit 1
fi
(
	cd "${minimal_checkout}"
	git apply --recount "${ROOT_DIR}/scripts/ci/fixtures/first-feature.patch"
	make openapi-generate
	# The health-only baseline needs no request binding helpers. The first
	# operation with a path or query parameter makes the generated code import
	# github.com/oapi-codegen/runtime, so the first feature tidies like any
	# other Go change that adds an import.
	go mod tidy
	gofmt -w internal/greeting internal/infra/http cmd/service/internal/bootstrap/run.go
	go test ./internal/greeting ./internal/infra/http ./cmd/service/internal/bootstrap
	make openapi-check
)
# The default profile must not hand a generated service the reference example's
# packages, second OpenAPI contract, or second main().
	assert "examples/ must not survive initialization" path_absent "${minimal_checkout}/examples"
fi

# profile:messaging-nats-jetstream:start
if [[ "${TEMPLATE_INIT_PROFILE}" == "all" || "${TEMPLATE_INIT_PROFILE}" == "messaging" ]]; then
	messaging_checkout="$(copy_template_checkout full-messaging git@github.com:acme/messaging-service.git)"
	messaging_revision="$(git -C "${messaging_checkout}" rev-parse HEAD)"
	(
		cd "${messaging_checkout}"
		CODEOWNER=@acme/platform DATABASE=none GRPC=none AUTHN=none MESSAGING=nats-jetstream \
			bash ./scripts/init-module.sh
		go test -vet=off ./...
		make build build-worker
		make mod-tidy-check
	)
	for retained in \
		cmd/worker \
		cmd/service/internal/bootstrap/startup_messaging.go \
		docs/durable-messaging.md \
		internal/config/messaging.go \
		internal/infra/natsjs \
		test/nats_messaging_integration_test.go; do
		assert "MESSAGING=nats-jetstream removed ${retained}" path_present "${messaging_checkout}/${retained}"
	done
	grep -Fq 'messaging = "nats-jetstream"' "${messaging_checkout}/template.lock"
	grep -Fqx "source_revision = \"${messaging_revision}\"" "${messaging_checkout}/template.lock"
	make -C "${messaging_checkout}" help >"${TEMP_ROOT}/messaging-help.log"
	grep -Fq 'run-worker' "${TEMP_ROOT}/messaging-help.log"
	(
		cd "${messaging_checkout}"
		go list -m -f '{{.Path}}' all | grep -Fx 'github.com/nats-io/nats.go'
	)
	messaging_marker='profile:messaging''-nats-jetstream:'
	assert "selected messaging checkout retained unresolved profile markers" \
		grep_absent -R -Fq "${messaging_marker}" \
		"${messaging_checkout}/.github" \
		"${messaging_checkout}/Makefile" \
		"${messaging_checkout}/README.md" \
		"${messaging_checkout}/build" \
		"${messaging_checkout}/cmd" \
		"${messaging_checkout}/docs" \
		"${messaging_checkout}/env" \
		"${messaging_checkout}/internal" \
		"${messaging_checkout}/scripts/ci"
	messaging_snapshot="$(snapshot "${messaging_checkout}")"
	(
		cd "${messaging_checkout}"
		CODEOWNER=@acme/platform DATABASE=none GRPC=none AUTHN=none MESSAGING=nats-jetstream \
			bash ./scripts/init-module.sh
	)
	assert "repeated messaging initialization changed the checkout" \
		same_text "${messaging_snapshot}" "$(snapshot "${messaging_checkout}")"
	expect_unchanged_failure "${messaging_checkout}" \
		env CODEOWNER=@acme/platform DATABASE=none GRPC=none AUTHN=none MESSAGING=none \
		bash "${ROOT_DIR}/scripts/init-module.sh"

	messaging_full_checkout="$(copy_template_checkout full-messaging-combination git@github.com:acme/messaging-full-service.git)"
	(
		cd "${messaging_full_checkout}"
		CODEOWNER=@acme/platform DATABASE=postgres GRPC=enabled AUTHN=oidc-jwt \
			OUTBOUND_HTTP=bounded MESSAGING=nats-jetstream REFERENCE_EXAMPLE=keep \
			bash ./scripts/init-module.sh
		go test -vet=off ./...
		go build ./cmd/service ./cmd/worker ./cmd/migrate
		make proto-check mod-tidy-check project-structure-check
	)
	for choice in \
		'database = "postgres"' \
		'grpc = "enabled"' \
		'authn = "oidc-jwt"' \
		'outbound_http = "bounded"' \
		'messaging = "nats-jetstream"' \
		'reference_example = "keep"'; do
		grep -Fqx "${choice}" "${messaging_full_checkout}/template.lock"
	done
	messaging_full_snapshot="$(snapshot "${messaging_full_checkout}")"
	(
		cd "${messaging_full_checkout}"
		CODEOWNER=@acme/platform DATABASE=postgres GRPC=enabled AUTHN=oidc-jwt \
			OUTBOUND_HTTP=bounded MESSAGING=nats-jetstream REFERENCE_EXAMPLE=keep \
			bash ./scripts/init-module.sh
	)
	assert "repeated full messaging initialization changed the checkout" \
		same_text "${messaging_full_snapshot}" "$(snapshot "${messaging_full_checkout}")"
fi
# profile:messaging-nats-jetstream:end

# profile:outbox-postgres:start
if [[ "${TEMPLATE_INIT_PROFILE}" == "all" || "${TEMPLATE_INIT_PROFILE}" == "outbox" ]]; then
	outbox_none_checkout="$(copy_template_checkout outbox-none-postgres git@github.com:acme/outbox-none-postgres.git)"
	(
		cd "${outbox_none_checkout}"
		CODEOWNER=@acme/platform DATABASE=postgres bash ./scripts/init-module.sh
		go test -vet=off ./...
		go build ./cmd/service ./cmd/migrate
		make sqlc-check migration-check mod-tidy-check project-structure-check
	)
	assert "OUTBOX=none removed PostgreSQL" path_present "${outbox_none_checkout}/internal/infra/postgres"
	for removed in \
		cmd/outbox-relay \
		docs/postgres-transactional-outbox.md \
		internal/config/outbox_config_test.go \
		internal/infra/postgres/queries/postgres_outbox.sql \
		internal/infra/postgres/sqlcgen/postgres_outbox.sql.go \
		internal/infra/postgresoutbox \
		test/postgres_outbox_bench_integration_test.go \
		test/postgres_outbox_integration_test.go \
		test/postgres_outbox_natsjs_integration_test.go; do
		assert "PostgreSQL OUTBOX=none retained ${removed}" path_absent "${outbox_none_checkout}/${removed}"
	done
	# Every outbox migration, not a named list: one left behind runs against
	# tables this profile never creates, and only the generated service's own
	# migration run would notice.
	assert "PostgreSQL OUTBOX=none retained an outbox migration" \
		glob_absent "${outbox_none_checkout}/migrations/*_postgres_outbox*.sql"
	grep -Fqx 'database = "postgres"' "${outbox_none_checkout}/template.lock"
	grep -Fqx 'outbox = "none"' "${outbox_none_checkout}/template.lock"
	outbox_none_snapshot="$(snapshot "${outbox_none_checkout}")"
	(
		cd "${outbox_none_checkout}"
		CODEOWNER=@acme/platform DATABASE=postgres OUTBOX=none bash "${ROOT_DIR}/scripts/init-module.sh"
	)
	assert "explicit PostgreSQL OUTBOX=none changed default-none checkout" \
		same_text "${outbox_none_snapshot}" "$(snapshot "${outbox_none_checkout}")"
	expect_unchanged_failure "${outbox_none_checkout}" \
		env CODEOWNER=@acme/platform DATABASE=postgres OUTBOX=postgres bash "${ROOT_DIR}/scripts/init-module.sh"

	outbox_checkout="$(copy_template_checkout full-outbox git@github.com:acme/outbox-service.git)"
	outbox_revision="$(git -C "${outbox_checkout}" rev-parse HEAD)"
	(
		cd "${outbox_checkout}"
		CODEOWNER=@acme/platform DATABASE=postgres OUTBOX=postgres bash ./scripts/init-module.sh
		go test -vet=off ./...
		go build ./cmd/service ./cmd/migrate ./cmd/outbox-relay
		make sqlc-check migration-check mod-tidy-check project-structure-check
		if go run ./cmd/outbox-relay >"${TEMP_ROOT}/outbox-missing-publisher.log" 2>&1; then
			echo "outbox relay accepted a missing production publisher"
			exit 1
		fi
		go list -deps ./cmd/outbox-relay | grep -Fx 'github.com/acme/outbox-service/internal/infra/postgresoutbox'
	)
	grep -Fxq 'outbox relay failed: error_class=config' "${TEMP_ROOT}/outbox-missing-publisher.log"
	assert "outbox relay leaked raw missing-publisher error" \
		grep_absent -Fq 'outbox publisher builder is not registered' "${TEMP_ROOT}/outbox-missing-publisher.log"
	for retained in \
		cmd/outbox-relay \
		docs/postgres-transactional-outbox.md \
		internal/config/outbox_config_test.go \
		internal/infra/postgres/queries/postgres_outbox.sql \
		internal/infra/postgres/sqlcgen/postgres_outbox.sql.go \
		internal/infra/postgresoutbox \
		test/postgres_outbox_bench_integration_test.go \
		test/postgres_outbox_integration_test.go; do
		assert "OUTBOX=postgres removed ${retained}" path_present "${outbox_checkout}/${retained}"
	done
	# Compared against the template's own outbox migrations rather than a named
	# list, so a migration added later is covered the day it lands.
	assert "OUTBOX=postgres changed the outbox migration set" same_text \
		"$(cd "${ROOT_DIR}" && printf '%s\n' migrations/*_postgres_outbox*.sql)" \
		"$(cd "${outbox_checkout}" && printf '%s\n' migrations/*_postgres_outbox*.sql)"
	# The template edits its migrations in place because nothing has applied
	# them; a generated service is the opposite case and must refuse, with no
	# escape it could be talked into. Exercised on an isolated copy of the
	# generated tree so the fixture's own git state stays untouched.
	(
		rewrite_probe="${TEMP_ROOT}/outbox-rewrite-probe"
		mkdir -p "${rewrite_probe}"
		cp -R "${outbox_checkout}/migrations" "${rewrite_probe}/migrations"
		cp -R "${outbox_checkout}/scripts" "${rewrite_probe}/scripts"
		cd "${rewrite_probe}"
		git init -q .
		git add -A
		git -c user.email=init-check@example.invalid -c user.name=init-check commit -qm generated
		printf '\n-- rewritten after generation\n' >>migrations/000001_postgres_outbox.sql
		if MIGRATION_REPO_ROOT="${rewrite_probe}" MIGRATION_HISTORY_MODE=worktree \
			bash ./scripts/ci/migration-history-check.sh >/dev/null 2>&1; then
			echo "template initialization contract: generated service accepted a migration rewrite" >&2
			exit 1
		fi
	)
	grep -Fqx 'database = "postgres"' "${outbox_checkout}/template.lock"
	grep -Fqx 'outbox = "postgres"' "${outbox_checkout}/template.lock"
	grep -Fqx "source_revision = \"${outbox_revision}\"" "${outbox_checkout}/template.lock"
	grep -Fq 'APP__OUTBOX__ENABLED=true' "${outbox_checkout}/env/.env.example"
	grep -Fq 'run-outbox-relay' "${outbox_checkout}/Makefile"
	grep -Fq '/out/outbox-relay' "${outbox_checkout}/build/docker/Dockerfile"
	outbox_marker='profile:outbox''-postgres:'
	assert "selected outbox checkout retained unresolved profile markers" \
		grep_absent -R -Fq "${outbox_marker}" \
		"${outbox_checkout}/.github" \
		"${outbox_checkout}/Makefile" \
		"${outbox_checkout}/README.md" \
		"${outbox_checkout}/build" \
		"${outbox_checkout}/cmd" \
		"${outbox_checkout}/docs" \
		"${outbox_checkout}/env" \
		"${outbox_checkout}/internal" \
		"${outbox_checkout}/scripts/ci" \
		"${outbox_checkout}/test"
	outbox_snapshot="$(snapshot "${outbox_checkout}")"
	(
		cd "${outbox_checkout}"
		CODEOWNER=@acme/platform DATABASE=postgres OUTBOX=postgres bash ./scripts/init-module.sh
	)
	assert "repeated OUTBOX=postgres initialization changed the checkout" \
		same_text "${outbox_snapshot}" "$(snapshot "${outbox_checkout}")"
	expect_unchanged_failure "${outbox_checkout}" \
		env CODEOWNER=@acme/platform DATABASE=postgres OUTBOX=none bash "${ROOT_DIR}/scripts/init-module.sh"

	# profile:messaging-nats-jetstream:start
	outbox_messaging_checkout="$(copy_template_checkout outbox-messaging git@github.com:acme/outbox-messaging-service.git)"
	(
		cd "${outbox_messaging_checkout}"
		CODEOWNER=@acme/platform DATABASE=postgres OUTBOX=postgres MESSAGING=nats-jetstream \
			bash ./scripts/init-module.sh
		go test -vet=off ./...
		go build ./cmd/service ./cmd/migrate ./cmd/worker ./cmd/outbox-relay
		if go list -deps ./cmd/outbox-relay | grep -Fq '/internal/infra/natsjs'; then
			echo "outbox relay unexpectedly owns the selected messaging adapter"
			exit 1
		fi
	)
	assert "combined OUTBOX/MESSAGING profile removed broker conformance proof" \
		file_present "${outbox_messaging_checkout}/test/postgres_outbox_natsjs_integration_test.go"
	grep -Fqx 'outbox = "postgres"' "${outbox_messaging_checkout}/template.lock"
	grep -Fqx 'messaging = "nats-jetstream"' "${outbox_messaging_checkout}/template.lock"
	outbox_messaging_snapshot="$(snapshot "${outbox_messaging_checkout}")"
	(
		cd "${outbox_messaging_checkout}"
		CODEOWNER=@acme/platform DATABASE=postgres OUTBOX=postgres MESSAGING=nats-jetstream \
			bash ./scripts/init-module.sh
	)
	assert "repeated combined OUTBOX/MESSAGING initialization changed the checkout" \
		same_text "${outbox_messaging_snapshot}" "$(snapshot "${outbox_messaging_checkout}")"
	# profile:messaging-nats-jetstream:end
fi
# profile:outbox-postgres:end

if [[ "${TEMPLATE_INIT_PROFILE}" == "all" || "${TEMPLATE_INIT_PROFILE}" == "postgres" ]]; then
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
	scripts/ci/migration-source-check.sh \
	scripts/ci/migration-history-check.sh \
	scripts/ci/migration-check-self-test.sh \
	scripts/ci/migration-image-history-check.sh \
	scripts/ci/migration-publication-check.sh \
	env/docker-compose.yml; do
	assert "${retained} must survive DATABASE=postgres initialization" path_present "${postgres_checkout}/${retained}"
done
grep -Fq 'database = "postgres"' "${postgres_checkout}/template.lock"
grep -Fq 'outbound_http = "bounded"' "${postgres_checkout}/template.lock"
(
	cd "${postgres_checkout}"
	make template-init-check >"${TEMP_ROOT}/postgres-init-check.log"
	make project-structure-check
	make migration-check migration-publication-check
	go -C tools tool -n goose >/dev/null
	go -C tools tool -n sqlc >/dev/null
)
grep -Fq 'upstream-only' "${TEMP_ROOT}/postgres-init-check.log"

if [[ "${TEMPLATE_POSTGRES_PROOF:-0}" == "1" ]]; then
	(
		cd "${postgres_checkout}"
		git apply --recount "${ROOT_DIR}/scripts/ci/fixtures/postgres-post-feature.patch"
		make migration-check
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
		go list -deps ./cmd/service |
			grep -E 'github.com/pressly/goose|internal/infra/postgresmigrate'
	); then
		echo "service dependency graph includes migration implementation"
		exit 1
	fi
fi

if [[ "${TEMPLATE_INIT_PROFILE}" == "all" || "${TEMPLATE_INIT_PROFILE}" == "grpc" ]]; then
	malformed_grpc="$(new_fixture malformed-grpc git@github.com:acme/malformed-grpc.git)"
	expect_unchanged_failure "${malformed_grpc}" \
		env CODEOWNER=@acme/platform GRPC=custom bash "${ROOT_DIR}/scripts/init-module.sh"

	grpc_checkout="$(copy_template_checkout full-grpc git@github.com:acme/grpc-service.git)"
	grpc_workflow_before="$(workflow_snapshot "${grpc_checkout}")"
	(
		cd "${grpc_checkout}"
		CODEOWNER=@acme/platform DATABASE=none GRPC=enabled REFERENCE_EXAMPLE=keep \
			bash ./scripts/init-module.sh
		go test ./...
		go build ./cmd/service
		make proto-check
	)
assert "gRPC enabled initialization removed server adapter" path_present "${grpc_checkout}/internal/infra/grpc"
assert "gRPC enabled initialization removed client adapter" path_present "${grpc_checkout}/internal/infra/grpcclient"
assert "gRPC enabled initialization removed test descriptors" path_present "${grpc_checkout}/internal/infra/grpc/grpctest"
assert "gRPC enabled initialization removed Buf config" file_present "${grpc_checkout}/buf.yaml"
assert "gRPC enabled initialization removed protobuf workflow" file_present "${grpc_checkout}/scripts/proto.sh"
assert "gRPC enabled initialization removed bootstrap wiring" file_present "${grpc_checkout}/cmd/service/internal/bootstrap/startup_grpc.go"
assert "gRPC enabled initialization removed guide" file_present "${grpc_checkout}/docs/grpc.md"
assert "gRPC enabled initialization removed reference" path_present "${grpc_checkout}/examples/grpc-reference-service"
assert "gRPC reference profile removed benchmark lifecycle proof" file_present "${grpc_checkout}/scripts/dev/benchmark-grpc-check.sh"
assert "gRPC reference profile removed k6 scenario" file_present "${grpc_checkout}/test/performance/grpc/all-cardinalities.js"
grep -Fq 'bench-grpc-smoke' "${grpc_checkout}/Makefile"
grep -Fq 'GRPC_BENCH_SCRIPT' "${grpc_checkout}/scripts/dev/benchmark.sh"
grep -Fq 'bench-grpc-smoke' "${grpc_checkout}/docs/grpc.md"
assert "gRPC reference profile retained unresolved benchmark markers" \
	grep_absent -R -Fq 'profile:grpc-reference-benchmark:' \
	"${grpc_checkout}/Makefile" \
	"${grpc_checkout}/scripts/dev/benchmark.sh" \
	"${grpc_checkout}/docs"
grep -Fq 'grpc = "enabled"' "${grpc_checkout}/template.lock"
assert "agent workflow changed during gRPC initialization" same_text "${grpc_workflow_before}" "$(workflow_snapshot "${grpc_checkout}")"
grpc_snapshot="$(snapshot "${grpc_checkout}")"
(
	cd "${grpc_checkout}"
	CODEOWNER=@acme/platform DATABASE=none GRPC=enabled REFERENCE_EXAMPLE=keep \
		bash ./scripts/init-module.sh
)
assert "repeated gRPC initialization changed the checkout" same_text "${grpc_snapshot}" "$(snapshot "${grpc_checkout}")"

grpc_default_checkout="$(copy_template_checkout default-grpc git@github.com:acme/grpc-default-service.git)"
(
	cd "${grpc_default_checkout}"
	CODEOWNER=@acme/platform DATABASE=none GRPC=enabled \
		bash ./scripts/init-module.sh
	go test ./...
	go build ./cmd/service
	make proto-check
)
assert "default gRPC initialization removed server adapter" path_present "${grpc_default_checkout}/internal/infra/grpc"
assert "default gRPC initialization removed client adapter" path_present "${grpc_default_checkout}/internal/infra/grpcclient"
assert "default gRPC initialization retained reference examples" path_absent "${grpc_default_checkout}/examples"
assert "default gRPC initialization retained benchmark lifecycle proof" path_absent "${grpc_default_checkout}/scripts/dev/benchmark-grpc-check.sh"
assert "default gRPC initialization retained gRPC performance scenario" path_absent "${grpc_default_checkout}/test/performance/grpc"
for benchmark_surface in \
	Makefile \
	scripts/dev/benchmark.sh \
	docs/grpc.md \
	docs/benchmarking.md \
	docs/build-test-and-development-commands.md; do
	assert "REFERENCE_EXAMPLE=remove retained gRPC reference benchmark commands in ${benchmark_surface}" \
		grep_absent -Eq 'bench-grpc|GRPC_BENCH|grpc-smoke|grpc-inspect' \
		"${grpc_default_checkout}/${benchmark_surface}"
done
grep -Fq 'grpc = "enabled"' "${grpc_default_checkout}/template.lock"
grep -Fq 'reference_example = "remove"' "${grpc_default_checkout}/template.lock"
grpc_default_snapshot="$(snapshot "${grpc_default_checkout}")"
(
	cd "${grpc_default_checkout}"
	CODEOWNER=@acme/platform DATABASE=none GRPC=enabled \
		bash ./scripts/init-module.sh
)
assert "repeated default gRPC initialization changed the checkout" \
	same_text "${grpc_default_snapshot}" "$(snapshot "${grpc_default_checkout}")"
fi

if [[ "${TEMPLATE_INIT_PROFILE}" == "all" || "${TEMPLATE_INIT_PROFILE}" == "authn" ]]; then
	malformed_authn="$(new_fixture malformed-authn git@github.com:acme/malformed-authn.git)"
	expect_unchanged_failure "${malformed_authn}" \
		env CODEOWNER=@acme/platform AUTHN=custom bash "${ROOT_DIR}/scripts/init-module.sh"

	empty_authn="$(new_fixture empty-authn git@github.com:acme/empty-authn.git)"
	expect_unchanged_failure "${empty_authn}" \
		env CODEOWNER=@acme/platform AUTHN= bash "${ROOT_DIR}/scripts/init-module.sh"

	verify_authn_none_profile() {
		local authn_choice="$1"
		local grpc_choice="$2"
		local outbound_choice="$3"
		local fixture_name="authn-${authn_choice}-${grpc_choice}-${outbound_choice}"
		local checkout
		local revision
		local before
		local -a init_environment=(
			"CODEOWNER=@acme/platform"
			"DATABASE=none"
			"GRPC=${grpc_choice}"
			"OUTBOUND_HTTP=${outbound_choice}"
		)

		checkout="$(copy_template_checkout "${fixture_name}" "git@github.com:acme/${fixture_name}-service.git")"
		revision="$(git -C "${checkout}" rev-parse HEAD)"
		(
			cd "${checkout}"
			if [[ "${authn_choice}" == "default" ]]; then
				env -u AUTHN "${init_environment[@]}" bash ./scripts/init-module.sh
			else
				env "${init_environment[@]}" AUTHN=none bash ./scripts/init-module.sh
			fi
			go test -vet=off ./...
			go build ./cmd/service
			make openapi-check
			if [[ "${grpc_choice}" == "enabled" ]]; then
				make proto-check
			fi
			if go list -deps ./cmd/service | grep -E 'internal/infra/oidcjwt|go-jose/go-jose'; then
				echo "${fixture_name} production graph retained authentication"
				exit 1
			fi
			if grep -Fq 'github.com/go-jose/go-jose/v4' go.mod; then
				echo "${fixture_name} retained a direct or indirect JWT requirement in go.mod"
				exit 1
			fi
		)

		for removed in \
			cmd/service/internal/bootstrap/authn_bootstrap_test.go \
			cmd/service/internal/bootstrap/authn_readiness_test.go \
			cmd/service/internal/bootstrap/startup_authn.go \
			docs/authentication.md \
			internal/config/authn_config_test.go \
			internal/infra/grpc/authn_health_test.go \
			internal/infra/http/authn_router_test.go \
			internal/infra/httpclient/authn_policy_test.go \
			internal/infra/oidcjwt; do
			assert "${fixture_name} retained ${removed}" path_absent "${checkout}/${removed}"
		done
		assert "${fixture_name} retained bearer security" \
			grep_absent -Fq 'bearerAuth' "${checkout}/api/openapi/service.yaml"
		assert "${fixture_name} retained authentication environment" \
			grep_absent -Fq 'APP__AUTHN__' "${checkout}/env/.env.example"
		assert "${fixture_name} retained unresolved authentication markers" \
			grep_absent -R -Fq 'profile:authn-oidc-jwt:' \
			"${checkout}/README.md" \
			"${checkout}/api" \
			"${checkout}/cmd" \
			"${checkout}/docs" \
			"${checkout}/env" \
			"${checkout}/internal" \
			"${checkout}/.github"
		if [[ "${outbound_choice}" == "bounded" ]]; then
			assert "${fixture_name} removed requested bounded HTTP client" \
				path_present "${checkout}/internal/infra/httpclient"
		else
			assert "${fixture_name} retained dormant HTTP client" \
				path_absent "${checkout}/internal/infra/httpclient"
		fi
		grep -Fq 'authn = "none"' "${checkout}/template.lock"
		grep -Fq "grpc = \"${grpc_choice}\"" "${checkout}/template.lock"
		grep -Fq "outbound_http = \"${outbound_choice}\"" "${checkout}/template.lock"
		grep -Fqx "source_revision = \"${revision}\"" "${checkout}/template.lock"

		before="$(snapshot "${checkout}")"
		(
			cd "${checkout}"
			if [[ "${authn_choice}" == "default" ]]; then
				env -u AUTHN "${init_environment[@]}" bash ./scripts/init-module.sh
			else
				env "${init_environment[@]}" AUTHN=none bash ./scripts/init-module.sh
			fi
		)
		assert "repeated ${fixture_name} initialization changed the checkout" \
			same_text "${before}" "$(snapshot "${checkout}")"
	}

	for authn_choice in default explicit; do
		for grpc_choice in none enabled; do
			for outbound_choice in none bounded; do
				verify_authn_none_profile "${authn_choice}" "${grpc_choice}" "${outbound_choice}"
			done
		done
	done

	authn_http_checkout="$(copy_template_checkout authn-http git@github.com:acme/authn-http-service.git)"
	authn_http_revision="$(git -C "${authn_http_checkout}" rev-parse HEAD)"
	(
		cd "${authn_http_checkout}"
		CODEOWNER=@acme/platform DATABASE=none GRPC=none AUTHN=oidc-jwt OUTBOUND_HTTP=none \
			bash ./scripts/init-module.sh
		go test -vet=off ./...
		go build ./cmd/service
		make openapi-check
	)
	for retained in \
		cmd/service/internal/bootstrap/startup_authn.go \
		docs/authentication.md \
		internal/config/authn_config_test.go \
		internal/infra/httpclient \
		internal/infra/oidcjwt; do
		assert "AUTHN=oidc-jwt removed ${retained}" path_present "${authn_http_checkout}/${retained}"
	done
	assert "GRPC=none retained the OIDC gRPC adapter" \
		path_absent "${authn_http_checkout}/internal/infra/oidcjwt/grpc.go"
	assert "GRPC=none retained the OIDC gRPC proof" \
		path_absent "${authn_http_checkout}/internal/infra/oidcjwt/grpc_test.go"
	assert "AUTHN=oidc-jwt retained unresolved profile markers" \
		grep_absent -R -Fq 'profile:authn-oidc-jwt:' \
		"${authn_http_checkout}/README.md" \
		"${authn_http_checkout}/api" \
		"${authn_http_checkout}/cmd" \
		"${authn_http_checkout}/docs" \
		"${authn_http_checkout}/env" \
		"${authn_http_checkout}/internal" \
		"${authn_http_checkout}/.github"
	grep -Fq 'authn = "oidc-jwt"' "${authn_http_checkout}/template.lock"
	grep -Fqx "source_revision = \"${authn_http_revision}\"" "${authn_http_checkout}/template.lock"
	grep -Fq 'type: http' "${authn_http_checkout}/api/openapi/service.yaml"
	grep -Fq 'scheme: bearer' "${authn_http_checkout}/api/openapi/service.yaml"
	grep -Fq 'APP__AUTHN__ISSUER=' "${authn_http_checkout}/env/.env.example"
	(
		cd "${authn_http_checkout}"
		go list -m -f '{{.Path}}' all | grep -Fx 'github.com/go-jose/go-jose/v4'
	)
	authn_http_snapshot="$(snapshot "${authn_http_checkout}")"
	(
		cd "${authn_http_checkout}"
		CODEOWNER=@acme/platform DATABASE=none GRPC=none AUTHN=oidc-jwt OUTBOUND_HTTP=none \
			bash ./scripts/init-module.sh
	)
	assert "repeated HTTP OIDC initialization changed the checkout" \
		same_text "${authn_http_snapshot}" "$(snapshot "${authn_http_checkout}")"

	authn_http_bounded_checkout="$(copy_template_checkout authn-http-bounded git@github.com:acme/authn-http-bounded-service.git)"
	(
		cd "${authn_http_bounded_checkout}"
		CODEOWNER=@acme/platform DATABASE=none GRPC=none AUTHN=oidc-jwt OUTBOUND_HTTP=bounded \
			bash ./scripts/init-module.sh
		go test -vet=off ./...
		go build ./cmd/service
		make openapi-check
	)
	assert "AUTHN=oidc-jwt with OUTBOUND_HTTP=bounded removed shared HTTP client" \
		path_present "${authn_http_bounded_checkout}/internal/infra/httpclient"
	assert "AUTHN=oidc-jwt HTTP bounded profile retained the gRPC adapter" \
		path_absent "${authn_http_bounded_checkout}/internal/infra/oidcjwt/grpc.go"
	grep -Fq 'authn = "oidc-jwt"' "${authn_http_bounded_checkout}/template.lock"
	grep -Fq 'outbound_http = "bounded"' "${authn_http_bounded_checkout}/template.lock"
	authn_http_bounded_snapshot="$(snapshot "${authn_http_bounded_checkout}")"
	(
		cd "${authn_http_bounded_checkout}"
		CODEOWNER=@acme/platform DATABASE=none GRPC=none AUTHN=oidc-jwt OUTBOUND_HTTP=bounded \
			bash ./scripts/init-module.sh
	)
	assert "repeated bounded HTTP OIDC initialization changed the checkout" \
		same_text "${authn_http_bounded_snapshot}" "$(snapshot "${authn_http_bounded_checkout}")"

	authn_grpc_checkout="$(copy_template_checkout authn-grpc git@github.com:acme/authn-grpc-service.git)"
	(
		cd "${authn_grpc_checkout}"
		CODEOWNER=@acme/platform DATABASE=none GRPC=enabled AUTHN=oidc-jwt OUTBOUND_HTTP=none \
			bash ./scripts/init-module.sh
		go test -vet=off ./...
		go build ./cmd/service
		make proto-check
	)
	assert "AUTHN=oidc-jwt with GRPC=enabled removed the unary/stream adapter" \
		file_present "${authn_grpc_checkout}/internal/infra/oidcjwt/grpc.go"
	assert "AUTHN=oidc-jwt with GRPC=enabled removed gRPC parity proof" \
		file_present "${authn_grpc_checkout}/internal/infra/oidcjwt/grpc_test.go"
	grep -Fq 'authn = "oidc-jwt"' "${authn_grpc_checkout}/template.lock"
	grep -Fq 'grpc = "enabled"' "${authn_grpc_checkout}/template.lock"
	authn_grpc_snapshot="$(snapshot "${authn_grpc_checkout}")"
	(
		cd "${authn_grpc_checkout}"
		CODEOWNER=@acme/platform DATABASE=none GRPC=enabled AUTHN=oidc-jwt OUTBOUND_HTTP=none \
			bash ./scripts/init-module.sh
	)
	assert "repeated gRPC OIDC initialization changed the checkout" \
		same_text "${authn_grpc_snapshot}" "$(snapshot "${authn_grpc_checkout}")"

	authn_grpc_bounded_checkout="$(copy_template_checkout authn-grpc-bounded git@github.com:acme/authn-grpc-bounded-service.git)"
	(
		cd "${authn_grpc_bounded_checkout}"
		CODEOWNER=@acme/platform DATABASE=none GRPC=enabled AUTHN=oidc-jwt OUTBOUND_HTTP=bounded \
			bash ./scripts/init-module.sh
		go test -vet=off ./...
		go build ./cmd/service
		make proto-check
	)
	assert "AUTHN=oidc-jwt gRPC bounded profile removed shared HTTP client" \
		path_present "${authn_grpc_bounded_checkout}/internal/infra/httpclient"
	assert "AUTHN=oidc-jwt gRPC bounded profile removed gRPC adapter" \
		file_present "${authn_grpc_bounded_checkout}/internal/infra/oidcjwt/grpc.go"
	grep -Fq 'authn = "oidc-jwt"' "${authn_grpc_bounded_checkout}/template.lock"
	grep -Fq 'grpc = "enabled"' "${authn_grpc_bounded_checkout}/template.lock"
	grep -Fq 'outbound_http = "bounded"' "${authn_grpc_bounded_checkout}/template.lock"
	authn_grpc_bounded_snapshot="$(snapshot "${authn_grpc_bounded_checkout}")"
	(
		cd "${authn_grpc_bounded_checkout}"
		CODEOWNER=@acme/platform DATABASE=none GRPC=enabled AUTHN=oidc-jwt OUTBOUND_HTTP=bounded \
			bash ./scripts/init-module.sh
	)
	assert "repeated bounded gRPC OIDC initialization changed the checkout" \
		same_text "${authn_grpc_bounded_snapshot}" "$(snapshot "${authn_grpc_bounded_checkout}")"
fi

echo "template initialization contract passed"
