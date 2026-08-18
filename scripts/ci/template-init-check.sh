#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
TEMPLATE_INIT_PROFILE="${TEMPLATE_INIT_PROFILE:-all}"
valid_profiles="all, minimal, postgres, grpc, authn, outbound-auth, http-idempotency, webhooks"
# profile:jobs-postgres:start
valid_profiles="${valid_profiles}, jobs"
# profile:jobs-postgres:end
# profile:object-storage:start
valid_profiles="${valid_profiles}, object-storage"
# profile:object-storage:end
# profile:messaging-nats-jetstream:start
valid_profiles="${valid_profiles}, messaging"
# profile:messaging-nats-jetstream:end
# profile:outbox-postgres:start
valid_profiles="${valid_profiles}, outbox"
# profile:outbox-postgres:end

case "${TEMPLATE_INIT_PROFILE}" in
	all | minimal | postgres | grpc | authn | outbound-auth | http-idempotency | webhooks) ;;
	# profile:jobs-postgres:start
	jobs) ;;
	# profile:jobs-postgres:end
	# profile:object-storage:start
	object-storage) ;;
	# profile:object-storage:end
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
		"${root}/internal/failure" \
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
	printf 'package config\n\ntype Config struct { Postgres PostgresConfig }\ntype PostgresConfig struct { Enabled bool }\n' \
		>"${root}/internal/config/defaults.go"
	# The gofmt column between this key and its value is deliberately not the one
	# the real observability_config.go carries: a fixture that mirrors one
	# alignment is how a rewrite keyed on that alignment passed here while
	# silently doing nothing to the real tree. The minimal profile below re-proves
	# the rewrite on the real file; this fixture proves only that the rewrite
	# ignores the column.
	printf 'package config\n\nfunc observabilityDefaults() map[string]any {\n\treturn map[string]any{\n\t\t"observability.otel.service_name": "service",\n\t}\n}\n' \
		>"${root}/internal/config/observability_config.go"
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
	printf 'package failure\n\ntype Mapper func(error) (struct{}, bool)\n' \
		>"${root}/internal/failure/failure.go"
	printf 'package problem\n' >"${root}/internal/problem/problem.go"
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

if [[ "${TEMPLATE_INIT_PROFILE}" != "postgres" && "${TEMPLATE_INIT_PROFILE}" != "outbound-auth" && "${TEMPLATE_INIT_PROFILE}" != "object-storage" && "${TEMPLATE_INIT_PROFILE}" != "jobs" && "${TEMPLATE_INIT_PROFILE}" != "webhooks" ]]; then
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
grep -Eq '"observability\.otel\.service_name": +"orders"' "${derived}/internal/config/observability_config.go"
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

removed_inbox="$(new_fixture removed-inbox git@github.com:acme/removed-inbox.git)"
expect_unchanged_failure "${removed_inbox}" \
	env CODEOWNER=@acme/platform INBOX=postgres bash "${ROOT_DIR}/scripts/init-module.sh"

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
			echo 'import _ "github.com/acme/feature-proof/internal/infra/http" // Prove depguard rejects the feature-to-infra boundary.'
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
		APP__MESSAGING__MIN_STREAM_REPLICAS=1 \
		APP__MESSAGING__MIN_STREAM_RETENTION=24h \
		go run ./cmd/service >"${TEMP_ROOT}/minimal-messaging.log" 2>&1; then
		echo "messaging-none service accepted messaging configuration"
		exit 1
	fi
	# profile:messaging-nats-jetstream:end
)
assert "agent workflow changed during minimal initialization" same_text "${minimal_workflow_before}" "$(workflow_snapshot "${minimal_checkout}")"
assert "minimal initialization removed transport-neutral failure policy" path_present "${minimal_checkout}/internal/failure"
# cmd/internal/runtimeopts ships in every profile and its diagnostics test
# reserves a listen address through this package, so no profile may take it.
assert "minimal initialization removed the shared test waits" path_present "${minimal_checkout}/internal/waittest"
# The identity rewrites above run against a fixture this script writes itself.
# These two files are the real tree, so they are the only proof that a derived
# service stops reporting the template's own name to a tracing backend. Keep
# both patterns free of the gofmt column: matching it is what let this rewrite
# silently do nothing once already.
grep -Eq '"observability\.otel\.service_name": +"feature-proof"' "${minimal_checkout}/internal/config/observability_config.go"
grep -Fq '"service.name", "feature-proof"' "${minimal_checkout}/cmd/service/internal/bootstrap/run.go"
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
	internal/domainevent \
	internal/infra/natsjs/outbox.go \
	internal/infra/natsjs/outbox_test.go \
	internal/infra/postgresoutbox \
	test/postgres_outbox_fixtures_integration_test.go \
	test/postgres_outbox_integration_test.go \
	test/postgres_outbox_natsjs_integration_test.go; do
	assert "${removed} must not survive OUTBOX=none initialization" path_absent "${minimal_checkout}/${removed}"
done
# profile:outbox-postgres:end
# A profile's paths are one list, asserted absent when the profile is off and
# present when it is on. Two lists would let a path be proven removed and never
# proven kept — which reads as full coverage and is how a path silently stops
# shipping to the services that selected the profile.
#
# These lists deliberately restate what scripts/init-module.sh removes instead of
# sharing one source with it. This file is the oracle for that script: a list
# derived from the thing it checks would pass by construction, and the failure
# worth catching is exactly a path the generator forgot.
postgres_paths=(
	cmd/migrate
	internal/infra/postgres
	internal/infra/postgresmigrate
	scripts/ci/migration-source-check.sh
	scripts/ci/migration-history-check.sh
	scripts/ci/migration-check-self-test.sh
	scripts/ci/migration-image-history-check.sh
	scripts/ci/migration-publication-check.sh
	env/docker-compose.yml
)
bounded_http_paths=(
	internal/infra/httpclient
)

for removed in \
	"${postgres_paths[@]}" \
	"${bounded_http_paths[@]}" \
	buf.yaml \
	buf.gen.yaml \
	cmd/service/internal/bootstrap/startup_grpc.go \
	cmd/service/internal/bootstrap/startup_grpc_test.go \
	cmd/service/internal/bootstrap/startup_authn.go \
	docs/authentication.md \
	docs/grpc.md \
	internal/config/authn_config_test.go \
	internal/config/grpc_config_test.go \
	internal/infra/oidcjwt \
	internal/infra/grpc \
	internal/infra/grpcclient \
	internal/packagetest \
	scripts/dev/benchmark-grpc-check.sh \
	scripts/proto.sh \
	scripts/run-buf.sh \
	scripts/ci/proto-check.sh \
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
	cmd/outbox-relay/internal/bootstrap/natsjs_publisher.go \
	cmd/outbox-relay/internal/bootstrap/natsjs_publisher_test.go \
	cmd/service/internal/bootstrap/startup_messaging.go \
	cmd/service/internal/bootstrap/startup_messaging_test.go \
	docs/durable-messaging.md \
	internal/config/messaging_config.go \
	internal/config/messaging_config_test.go \
	internal/config/configtest/messaging.go \
	test/nats_messaging_fixtures_integration_test.go \
	test/nats_messaging_delivery_integration_test.go; do
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
		internal/config/messaging_config.go \
		internal/infra/natsjs \
		test/nats_messaging_fixtures_integration_test.go \
		test/nats_messaging_delivery_integration_test.go; do
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
	# One list, asserted absent here and present under the complete
	# OUTBOX=postgres+MESSAGING=nats-jetstream profile below.
	outbox_paths=(
		cmd/outbox-relay
		docs/postgres-transactional-outbox.md
		internal/domainevent
		internal/infra/natsjs/outbox.go
		internal/infra/postgresoutbox
		test/postgres_outbox_fixtures_integration_test.go
		test/postgres_outbox_integration_test.go
		test/postgres_outbox_natsjs_integration_test.go
	)
	for removed in "${outbox_paths[@]}"; do
		assert "PostgreSQL OUTBOX=none retained ${removed}" path_absent "${outbox_none_checkout}/${removed}"
	done
	assert "generated service retained removed inbox SQLC output" path_absent \
		"${outbox_none_checkout}/internal/infra/postgres/sqlcgen/postgres_inbox.sql.go"
	assert "generated service retained removed inbox migration history" \
		glob_absent "${outbox_none_checkout}/migrations/*_postgres_inbox*.sql"
	assert "PostgreSQL OUTBOX=none retained the unused River schema" \
		path_absent "${outbox_none_checkout}/migrations/000008_river.sql"
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
		CODEOWNER=@acme/platform DATABASE=postgres OUTBOX=postgres MESSAGING=nats-jetstream bash ./scripts/init-module.sh
		go test -vet=off ./...
		go build ./cmd/service ./cmd/migrate ./cmd/worker ./cmd/outbox-relay
		make sqlc-check migration-check mod-tidy-check project-structure-check
		go list -deps ./cmd/outbox-relay | grep -Fx 'github.com/acme/outbox-service/internal/infra/postgresoutbox'
		go list -deps ./cmd/outbox-relay | grep -Fx 'github.com/acme/outbox-service/internal/infra/natsjs'
	)
	for retained in "${outbox_paths[@]}"; do
		assert "OUTBOX=postgres removed ${retained}" path_present "${outbox_checkout}/${retained}"
	done
	assert "generated service retained removed inbox SQLC output" path_absent \
		"${outbox_checkout}/internal/infra/postgres/sqlcgen/postgres_inbox.sql.go"
	assert "generated service retained removed inbox runtime" path_absent \
		"${outbox_checkout}/internal/infra/postgresinbox"
	assert "OUTBOX=postgres removed the shared River schema" path_present \
		"${outbox_checkout}/migrations/000008_river.sql"

	combined_checkout="$(copy_template_checkout combined-outbox-messaging git@github.com:acme/combined-service.git)"
	(
		cd "${combined_checkout}"
		CODEOWNER=@acme/platform DATABASE=postgres OUTBOX=postgres MESSAGING=nats-jetstream \
			bash ./scripts/init-module.sh
		go test -vet=off ./cmd/outbox-relay/... ./internal/infra/natsjs ./internal/infra/postgresoutbox
		go build ./cmd/outbox-relay
		go list -deps ./cmd/outbox-relay | grep -Fx 'github.com/acme/combined-service/internal/infra/natsjs'
		go list -deps ./cmd/outbox-relay | grep -Fx 'github.com/acme/combined-service/internal/infra/postgresoutbox'
		make sqlc-check mod-tidy-check project-structure-check
	)
	for retained in \
		internal/config/configtest/messaging.go \
		internal/infra/natsjs/outbox.go \
		test/postgres_outbox_natsjs_integration_test.go; do
		assert "combined outbox+messaging removed ${retained}" path_present "${combined_checkout}/${retained}"
	done
	assert "combined outbox+messaging retained removed inbox proof" path_absent \
		"${combined_checkout}/test/postgres_inbox_natsjs_integration_test.go"
	combined_snapshot="$(snapshot "${combined_checkout}")"
	(
		cd "${combined_checkout}"
		CODEOWNER=@acme/platform DATABASE=postgres OUTBOX=postgres MESSAGING=nats-jetstream \
			bash "${ROOT_DIR}/scripts/init-module.sh"
	)
	assert "repeated combined initialization changed the checkout" \
		same_text "${combined_snapshot}" "$(snapshot "${combined_checkout}")"

	assert "OUTBOX=postgres changed the River migration" same_text \
		"$(cat "${ROOT_DIR}/migrations/000008_river.sql")" \
		"$(cat "${outbox_checkout}/migrations/000008_river.sql")"
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
		printf '\n-- rewritten after generation\n' >>migrations/000008_river.sql
		if MIGRATION_REPO_ROOT="${rewrite_probe}" MIGRATION_HISTORY_MODE=worktree \
			bash ./scripts/ci/migration-history-check.sh >/dev/null 2>&1; then
			echo "template initialization contract: generated service accepted a migration rewrite" >&2
			exit 1
		fi
	)
	grep -Fqx 'database = "postgres"' "${outbox_checkout}/template.lock"
	grep -Fqx 'outbox = "postgres"' "${outbox_checkout}/template.lock"
	grep -Fqx "source_revision = \"${outbox_revision}\"" "${outbox_checkout}/template.lock"
	assert "generated outbox retained removed APP__OUTBOX__ config" \
		grep_absent -Fq 'APP__OUTBOX__' "${outbox_checkout}/env/.env.example"
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
		CODEOWNER=@acme/platform DATABASE=postgres OUTBOX=postgres MESSAGING=nats-jetstream bash ./scripts/init-module.sh
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
	postgres_paths=(
		cmd/migrate
		internal/infra/postgres
		internal/infra/postgresmigrate
		scripts/ci/migration-source-check.sh
		scripts/ci/migration-history-check.sh
		scripts/ci/migration-check-self-test.sh
		scripts/ci/migration-image-history-check.sh
		scripts/ci/migration-publication-check.sh
		env/docker-compose.yml
	)
	bounded_http_paths=(internal/infra/httpclient)
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
	for retained in "${postgres_paths[@]}" "${bounded_http_paths[@]}"; do
		assert "${retained} must survive DATABASE=postgres initialization" path_present "${postgres_checkout}/${retained}"
	done
	assert "generated PostgreSQL service retained removed inbox runtime" \
		path_absent "${postgres_checkout}/internal/infra/postgresinbox"
	assert "generated PostgreSQL service retained removed inbox SQLC output" \
		path_absent "${postgres_checkout}/internal/infra/postgres/sqlcgen/postgres_inbox.sql.go"
	assert "generated PostgreSQL service retained removed inbox migration history" \
		glob_absent "${postgres_checkout}/migrations/*_postgres_inbox*.sql"
	assert "generated PostgreSQL service retained removed inbox lock field" \
		grep_absent -Fq 'inbox = ' "${postgres_checkout}/template.lock"
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
# The comment and doc proofs in this package and internal/infra/oidcjwt walk
# their own source through it, so it leaves only when both profiles do — which
# is why AUTHN=none alone does not remove it here.
assert "gRPC enabled initialization removed the package source walk" \
	path_present "${grpc_checkout}/internal/packagetest"
# Asserted per profile as well as minimally, because the gRPC process suite is
# this package's heaviest consumer and would notice its loss first.
assert "gRPC enabled initialization removed the shared test waits" \
	path_present "${grpc_checkout}/internal/waittest"
assert "gRPC enabled initialization removed Buf config" file_present "${grpc_checkout}/buf.yaml"
assert "gRPC enabled initialization removed protobuf workflow" file_present "${grpc_checkout}/scripts/proto.sh"
assert "gRPC enabled initialization removed bootstrap wiring" file_present "${grpc_checkout}/cmd/service/internal/bootstrap/startup_grpc.go"
assert "gRPC enabled initialization removed guide" file_present "${grpc_checkout}/docs/grpc.md"
assert "gRPC enabled initialization removed reference" path_present "${grpc_checkout}/examples/grpc-reference-service"
assert "gRPC enabled initialization removed transport-neutral failure policy" path_present "${grpc_checkout}/internal/failure"
assert "gRPC enabled initialization removed composed failure contract" file_present "${grpc_checkout}/examples/reference-service/grpc_failure_mapping_contract_test.go"
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

grpc_http_checkout="$(copy_template_checkout grpc-http-reference git@github.com:acme/grpc-http-reference.git)"
(
	cd "${grpc_http_checkout}"
	CODEOWNER=@acme/platform DATABASE=none GRPC=none REFERENCE_EXAMPLE=keep \
		bash ./scripts/init-module.sh
	go test ./...
	go build ./cmd/service
)
assert "GRPC=none removed the retained HTTP reference" path_present "${grpc_http_checkout}/examples/reference-service"
assert "GRPC=none removed transport-neutral failure policy" path_present "${grpc_http_checkout}/internal/failure"
assert "GRPC=none removed HTTP already_exists proof" file_present "${grpc_http_checkout}/examples/reference-service/internal/article/errors_test.go"
assert "GRPC=none retained composed gRPC failure proof" path_absent "${grpc_http_checkout}/examples/reference-service/grpc_failure_mapping_contract_test.go"
assert "GRPC=none retained gRPC reference" path_absent "${grpc_http_checkout}/examples/grpc-reference-service"
assert "GRPC=none retained server adapter" path_absent "${grpc_http_checkout}/internal/infra/grpc"
assert "GRPC=none retained client adapter" path_absent "${grpc_http_checkout}/internal/infra/grpcclient"
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
			internal/authntrust \
			internal/config/authn_config_test.go \
			internal/infra/grpc/authn_health_test.go \
			internal/infra/http/authn_router_test.go \
			internal/infra/httpclient/authn_policy_test.go \
			internal/infra/oidcjwt; do
			assert "${fixture_name} retained ${removed}" path_absent "${checkout}/${removed}"
		done
		assert "${fixture_name} retained the old OIDC gRPC TLS proof path" \
			path_absent "${checkout}/internal/infra/oidcjwt/grpc_tls_test.go"
		assert "${fixture_name} retained the OIDC gRPC TLS contract proof" \
			path_absent "${checkout}/internal/infra/oidcjwt/grpc_tls_contract_test.go"
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
	assert "GRPC=none retained the old OIDC gRPC TLS proof path" \
		path_absent "${authn_http_checkout}/internal/infra/oidcjwt/grpc_tls_test.go"
	assert "GRPC=none retained the OIDC gRPC TLS contract proof" \
		path_absent "${authn_http_checkout}/internal/infra/oidcjwt/grpc_tls_contract_test.go"
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
	assert "AUTHN=oidc-jwt HTTP bounded profile retained the old gRPC TLS proof path" \
		path_absent "${authn_http_bounded_checkout}/internal/infra/oidcjwt/grpc_tls_test.go"
	assert "AUTHN=oidc-jwt HTTP bounded profile retained the gRPC TLS contract proof" \
		path_absent "${authn_http_bounded_checkout}/internal/infra/oidcjwt/grpc_tls_contract_test.go"
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
	assert "AUTHN=oidc-jwt with GRPC=enabled removed the gRPC TLS contract proof" \
		file_present "${authn_grpc_checkout}/internal/infra/oidcjwt/grpc_tls_contract_test.go"
	assert "AUTHN=oidc-jwt with GRPC=enabled retained the old gRPC TLS proof path" \
		path_absent "${authn_grpc_checkout}/internal/infra/oidcjwt/grpc_tls_test.go"
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
	assert "AUTHN=oidc-jwt gRPC bounded profile removed the gRPC TLS contract proof" \
		file_present "${authn_grpc_bounded_checkout}/internal/infra/oidcjwt/grpc_tls_contract_test.go"
	assert "AUTHN=oidc-jwt gRPC bounded profile retained the old gRPC TLS proof path" \
		path_absent "${authn_grpc_bounded_checkout}/internal/infra/oidcjwt/grpc_tls_test.go"
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

# profile:outbound-auth-oauth2-client-credentials:start
if [[ "${TEMPLATE_INIT_PROFILE}" == "all" || "${TEMPLATE_INIT_PROFILE}" == "outbound-auth" ]]; then
	outbound_auth_marker='profile:outbound-auth''-'
	assert_outbound_auth_tree() {
		local root="$1"
		local http="$2"
		local grpc="$3"

		for retained in \
			cmd/service/internal/bootstrap/startup_outbound_auth.go \
			internal/config/outbound_auth_config.go \
			internal/infra/oauth2clientcredentials \
			docs/outbound-machine-authentication.md; do
			assert "OAuth selection removed ${retained}" path_present "${root}/${retained}"
		done
		grep -Fqx 'outbound_auth = "oauth2-client-credentials"' "${root}/template.lock"
		grep -Fqx 'APP__OUTBOUND_AUTH__CLIENT_ID=' "${root}/env/.env.example"
		grep -Fqx 'APP__OUTBOUND_AUTH__CLIENT_SECRET=' "${root}/env/.env.example"
		grep -Fqx 'APP__OUTBOUND_AUTH__TOKEN_ENDPOINT=' "${root}/env/.env.example"
		grep -Fqx 'APP__OUTBOUND_AUTH__RESOURCE_AUTHORITY=' "${root}/env/.env.example"
		assert "OAuth selection removed credential HTTP policy" grep -Fq \
			'DisableInstrumentation' "${root}/internal/infra/httpclient/config.go"
		assert "OAuth selection retained unresolved markers" grep_absent -R -Fq \
			"${outbound_auth_marker}" "${root}/README.md" "${root}/cmd" "${root}/docs" \
			"${root}/env" "${root}/internal" "${root}/scripts/ci"

		if [[ "${http}" == "bounded" ]]; then
			assert "HTTP OAuth selection removed adapter" file_present \
				"${root}/internal/infra/oauth2clientcredentials/http.go"
			assert "HTTP OAuth selection removed attempt seam" file_present \
				"${root}/internal/infra/httpclient/attempt_authorization.go"
		else
			assert "gRPC-only OAuth retained HTTP adapter" path_absent \
				"${root}/internal/infra/oauth2clientcredentials/http.go"
			assert "gRPC-only OAuth retained HTTP attempt seam" path_absent \
				"${root}/internal/infra/httpclient/attempt_authorization.go"
			assert "gRPC-only OAuth removed token HTTP owner" path_present \
				"${root}/internal/infra/httpclient"
		fi

		if [[ "${grpc}" == "enabled" ]]; then
			assert "gRPC OAuth selection removed adapter" file_present \
				"${root}/internal/infra/oauth2clientcredentials/grpc.go"
		else
			assert "HTTP-only OAuth retained gRPC adapter" path_absent \
				"${root}/internal/infra/oauth2clientcredentials/grpc.go"
		fi
	}

	assert_outbound_auth_none_tree() {
		local root="$1"
		for removed in \
			cmd/service/internal/bootstrap/startup_outbound_auth.go \
			internal/config/outbound_auth_config.go \
			internal/infra/oauth2clientcredentials \
			docs/outbound-machine-authentication.md; do
			assert "OUTBOUND_AUTH=none retained ${removed}" path_absent "${root}/${removed}"
		done
		grep -Fqx 'outbound_auth = "none"' "${root}/template.lock"
		assert "OUTBOUND_AUTH=none retained config or environment" grep_absent -R -Eq \
			'OUTBOUND_AUTH|outbound_auth|oauth2clientcredentials|outbound-machine-authentication' \
			"${root}/README.md" "${root}/cmd" "${root}/docs" "${root}/env" "${root}/internal" "${root}/scripts/ci"
		assert "OUTBOUND_AUTH=none retained unresolved marker" grep_absent -R -Fq \
			"${outbound_auth_marker}" "${root}/README.md" "${root}/cmd" "${root}/docs" \
			"${root}/env" "${root}/internal" "${root}/scripts/ci"
	}

	check_outbound_auth_checkout() {
		(
			cd "$1"
			go test -vet=off ./...
			go build ./cmd/service
			make mod-tidy-check
		)
	}

	assert_outbound_auth_module_boundary() {
		local root="$1"
		local module

		assert "generated output imports golang.org/x/oauth2" grep_absent -R -Fq \
			--include='*.go' '"golang.org/x/oauth2' "${root}"
		module="$({
			cd "${root}"
			go list -m -f '{{.Path}} {{.Version}} {{.Indirect}}' all
		} | awk '$1 == "golang.org/x/oauth2" { print $1, $2, "indirect=" $3 }')"
		if [[ -n "${module}" && "${module}" != 'golang.org/x/oauth2 v0.36.0 indirect=true' ]]; then
			echo "unexpected OAuth module attribution: ${module}"
			return 1
		fi
	}

	outbound_auth_module_attribution() {
		local root="$1"
		(
			cd "${root}"
			go list -m -f '{{.Path}} {{.Version}} {{.Indirect}}' all |
				awk '$1 == "golang.org/x/oauth2" { print $1, $2, "indirect=" $3 }'
			go mod graph | awk '$2 == "golang.org/x/oauth2@v0.36.0"' | LC_ALL=C sort
		)
	}

	outbound_auth_oauth2_packages() {
		(
			cd "$1"
			go list -deps -test ./... | awk '/^golang.org\/x\/oauth2($|\/)/' | LC_ALL=C sort -u
		)
	}

	for invalid in '' custom; do
		invalid_outbound_auth="$(copy_template_checkout outbound-auth-invalid git@github.com:acme/outbound-auth-invalid.git)"
		if [[ -z "${invalid}" ]]; then
			expect_unchanged_failure "${invalid_outbound_auth}" env CODEOWNER=@acme/platform OUTBOUND_AUTH= \
				bash "${ROOT_DIR}/scripts/init-module.sh"
		else
			expect_unchanged_failure "${invalid_outbound_auth}" env CODEOWNER=@acme/platform OUTBOUND_AUTH="${invalid}" \
				bash "${ROOT_DIR}/scripts/init-module.sh"
		fi
	done
	no_consumer_outbound_auth="$(copy_template_checkout outbound-auth-no-consumer git@github.com:acme/outbound-auth-no-consumer.git)"
	expect_unchanged_failure "${no_consumer_outbound_auth}" env CODEOWNER=@acme/platform DATABASE=postgres GRPC=none \
		OUTBOUND_HTTP=none OUTBOUND_AUTH=oauth2-client-credentials bash "${ROOT_DIR}/scripts/init-module.sh"

	for transport in http grpc both; do
		case "${transport}" in
		http) outbound_http=bounded; grpc=none ;;
		grpc) outbound_http=none; grpc=enabled ;;
		both) outbound_http=bounded; grpc=enabled ;;
		esac
		base_outbound_auth="$(copy_template_checkout "outbound-auth-${transport}-base" "git@github.com:acme/outbound-auth-${transport}.git")"
		omitted_outbound_auth="${TEMP_ROOT}/outbound-auth-${transport}-omitted"
		explicit_none_outbound_auth="${TEMP_ROOT}/outbound-auth-${transport}-explicit-none"
		selected_outbound_auth="${TEMP_ROOT}/outbound-auth-${transport}-selected"
		cp -R "${base_outbound_auth}" "${omitted_outbound_auth}"
		cp -R "${base_outbound_auth}" "${explicit_none_outbound_auth}"
		cp -R "${base_outbound_auth}" "${selected_outbound_auth}"
		(
			cd "${omitted_outbound_auth}"
			CODEOWNER=@acme/platform GRPC="${grpc}" OUTBOUND_HTTP="${outbound_http}" \
				bash ./scripts/init-module.sh
		)
		(
			cd "${explicit_none_outbound_auth}"
			CODEOWNER=@acme/platform GRPC="${grpc}" OUTBOUND_HTTP="${outbound_http}" OUTBOUND_AUTH=none \
				bash ./scripts/init-module.sh
		)
		assert "omitted OUTBOUND_AUTH differs from explicit none for ${transport}" \
			same_text "$(snapshot "${omitted_outbound_auth}")" "$(snapshot "${explicit_none_outbound_auth}")"
		assert_outbound_auth_none_tree "${omitted_outbound_auth}"
		check_outbound_auth_checkout "${omitted_outbound_auth}"
		check_outbound_auth_checkout "${explicit_none_outbound_auth}"
		omitted_snapshot="$(snapshot "${omitted_outbound_auth}")"
		(
			cd "${omitted_outbound_auth}"
			CODEOWNER=@acme/platform GRPC="${grpc}" OUTBOUND_HTTP="${outbound_http}" \
				bash ./scripts/init-module.sh
		)
		assert "repeated omitted ${transport} OAuth initialization changed the checkout" \
			same_text "${omitted_snapshot}" "$(snapshot "${omitted_outbound_auth}")"
		none_snapshot="$(snapshot "${explicit_none_outbound_auth}")"
		(
			cd "${explicit_none_outbound_auth}"
			CODEOWNER=@acme/platform GRPC="${grpc}" OUTBOUND_HTTP="${outbound_http}" OUTBOUND_AUTH=none \
				bash ./scripts/init-module.sh
		)
		assert "repeated explicit-none ${transport} OAuth initialization changed the checkout" \
			same_text "${none_snapshot}" "$(snapshot "${explicit_none_outbound_auth}")"
		(
			cd "${selected_outbound_auth}"
			CODEOWNER=@acme/platform GRPC="${grpc}" OUTBOUND_HTTP="${outbound_http}" \
				OUTBOUND_AUTH=oauth2-client-credentials bash ./scripts/init-module.sh
		)
		check_outbound_auth_checkout "${selected_outbound_auth}"
		assert_outbound_auth_tree "${selected_outbound_auth}" "${outbound_http}" "${grpc}"
		assert "omitted ${transport} output has invalid OAuth module ownership" \
			assert_outbound_auth_module_boundary "${omitted_outbound_auth}"
		assert "explicit-none ${transport} output has invalid OAuth module ownership" \
			assert_outbound_auth_module_boundary "${explicit_none_outbound_auth}"
		assert "selected ${transport} output has invalid OAuth module ownership" \
			assert_outbound_auth_module_boundary "${selected_outbound_auth}"
		omitted_module_attribution="$(outbound_auth_module_attribution "${omitted_outbound_auth}")"
		none_module_attribution="$(outbound_auth_module_attribution "${explicit_none_outbound_auth}")"
		selected_module_attribution="$(outbound_auth_module_attribution "${selected_outbound_auth}")"
		omitted_oauth2_packages="$(outbound_auth_oauth2_packages "${omitted_outbound_auth}")"
		none_oauth2_packages="$(outbound_auth_oauth2_packages "${explicit_none_outbound_auth}")"
		selected_oauth2_packages="$(outbound_auth_oauth2_packages "${selected_outbound_auth}")"
		assert "omitted and explicit-none ${transport} module attribution differs" \
			same_text "${omitted_module_attribution}" "${none_module_attribution}"
		assert "selected and none ${transport} module attribution differs" \
			same_text "${selected_module_attribution}" "${none_module_attribution}"
		assert "omitted and explicit-none ${transport} OAuth reachability differs" \
			same_text "${omitted_oauth2_packages}" "${none_oauth2_packages}"
		assert "selected and none ${transport} OAuth reachability differs" \
			same_text "${selected_oauth2_packages}" "${none_oauth2_packages}"
		if [[ "${grpc}" == "none" ]]; then
			assert "HTTP-only output can reach golang.org/x/oauth2 packages" \
				same_text '' "${selected_oauth2_packages}"
		else
			assert "gRPC output lost grpc-go OAuth attribution" grep -Fq \
				'google.golang.org/grpc@v1.83.0 golang.org/x/oauth2@v0.36.0' \
				<<<"${selected_module_attribution}"
		fi
		selected_snapshot="$(snapshot "${selected_outbound_auth}")"
		(
			cd "${selected_outbound_auth}"
			CODEOWNER=@acme/platform GRPC="${grpc}" OUTBOUND_HTTP="${outbound_http}" \
				OUTBOUND_AUTH=oauth2-client-credentials bash ./scripts/init-module.sh
		)
		assert "repeated ${transport} OAuth initialization changed the checkout" \
			same_text "${selected_snapshot}" "$(snapshot "${selected_outbound_auth}")"
	done

	none_union_outbound_auth="$(copy_template_checkout outbound-auth-none-union git@github.com:acme/outbound-auth-none-union.git)"
	(
		cd "${none_union_outbound_auth}"
		CODEOWNER=@acme/platform DATABASE=postgres AUTHN=none OUTBOUND_HTTP=bounded OUTBOUND_AUTH=none \
			bash ./scripts/init-module.sh
	)
	assert "no credential profile retained credential option" grep_absent -R -Fq \
		'DisableInstrumentation' "${none_union_outbound_auth}/internal/infra/httpclient"
	oidc_union_outbound_auth="$(copy_template_checkout outbound-auth-oidc-union git@github.com:acme/outbound-auth-oidc-union.git)"
	(
		cd "${oidc_union_outbound_auth}"
		CODEOWNER=@acme/platform DATABASE=postgres AUTHN=oidc-jwt OUTBOUND_HTTP=none OUTBOUND_AUTH=none \
			bash ./scripts/init-module.sh
	)
	grep -Fq 'DisableInstrumentation' "${oidc_union_outbound_auth}/internal/infra/httpclient/config.go"
fi
# profile:outbound-auth-oauth2-client-credentials:end

# profile:http-idempotency-postgres:start
if [[ "${TEMPLATE_INIT_PROFILE}" == "all" || "${TEMPLATE_INIT_PROFILE}" == "http-idempotency" ]]; then
	http_idempotency_marker='profile:http-idempotency''-postgres:'
	assert_http_idempotency_none_tree() {
		local root="$1"
		for removed in \
			cmd/service/internal/bootstrap/startup_idempotency.go \
			internal/config/http_idempotency_config.go \
			internal/httpidempotency \
			internal/infra/postgresidempotency \
			migrations/000003_postgres_http_idempotency.sql \
			internal/infra/postgres/queries/postgres_http_idempotency.sql \
			docs/postgres-http-idempotency.md; do
			assert "HTTP_IDEMPOTENCY=none retained ${removed}" path_absent "${root}/${removed}"
		done
		grep -Fqx 'http_idempotency = "none"' "${root}/template.lock"
		assert "HTTP_IDEMPOTENCY=none retained a capability surface" grep_absent -R -E \
			'HTTP_IDEMPOTENCY|http_idempotency|httpidempotency|postgresidempotency|postgres-http-idempotency' \
			"${root}/README.md" "${root}/cmd" "${root}/docs" "${root}/env" "${root}/internal" "${root}/migrations"
		assert "HTTP_IDEMPOTENCY=none retained a marker" grep_absent -R -Fq \
			"${http_idempotency_marker}" \
			"${root}/README.md" "${root}/cmd" "${root}/docs" "${root}/env" "${root}/internal" "${root}/scripts/ci"
	}

	assert_http_idempotency_postgres_tree() {
		local root="$1"
		for retained in \
			cmd/service/internal/bootstrap/startup_idempotency.go \
			internal/config/http_idempotency_config.go \
			internal/httpidempotency \
			internal/infra/postgresidempotency \
			migrations/000003_postgres_http_idempotency.sql \
			internal/infra/postgres/queries/postgres_http_idempotency.sql \
			internal/infra/postgres/sqlcgen/postgres_http_idempotency.sql.go \
			docs/postgres-http-idempotency.md; do
			assert "HTTP_IDEMPOTENCY=postgres removed ${retained}" path_present "${root}/${retained}"
		done
		grep -Fqx 'http_idempotency = "postgres"' "${root}/template.lock"
		assert "HTTP_IDEMPOTENCY=postgres retained a marker" grep_absent -R -Fq \
			"${http_idempotency_marker}" \
			"${root}/README.md" "${root}/cmd" "${root}/docs" "${root}/env" "${root}/internal" "${root}/scripts/ci"
		assert "health route opted into idempotency" grep_absent -Fq \
			'x-idempotent' "${root}/api/openapi/service.yaml"
	}

	base_http_idempotency="$(copy_template_checkout http-idempotency-base git@github.com:acme/http-idempotency-service.git)"
	omitted_http_idempotency="${TEMP_ROOT}/http-idempotency-omitted"
	explicit_none_http_idempotency="${TEMP_ROOT}/http-idempotency-explicit-none"
	cp -R "${base_http_idempotency}" "${omitted_http_idempotency}"
	cp -R "${base_http_idempotency}" "${explicit_none_http_idempotency}"
	(
		cd "${omitted_http_idempotency}"
		CODEOWNER=@acme/platform DATABASE=postgres OUTBOX=postgres MESSAGING=nats-jetstream AUTHN=oidc-jwt GRPC=enabled OUTBOUND_HTTP=bounded \
			bash ./scripts/init-module.sh
		go test -vet=off ./...
		go build -o "${TEMP_ROOT}/http-idempotency-service" ./cmd/service
		make openapi-check
		make sqlc-check
	)
	(
		cd "${explicit_none_http_idempotency}"
		CODEOWNER=@acme/platform DATABASE=postgres HTTP_IDEMPOTENCY=none OUTBOX=postgres MESSAGING=nats-jetstream AUTHN=oidc-jwt GRPC=enabled OUTBOUND_HTTP=bounded \
			bash ./scripts/init-module.sh
	)
	assert "omitted HTTP_IDEMPOTENCY differs from explicit none" \
		same_text "$(snapshot "${omitted_http_idempotency}")" "$(snapshot "${explicit_none_http_idempotency}")"
	assert_http_idempotency_none_tree "${omitted_http_idempotency}"

	for invalid in '' custom; do
		invalid_http_idempotency="$(copy_template_checkout http-idempotency-invalid git@github.com:acme/http-idempotency-invalid.git)"
		if [[ -z "${invalid}" ]]; then
			expect_unchanged_failure "${invalid_http_idempotency}" env CODEOWNER=@acme/platform HTTP_IDEMPOTENCY= \
				bash "${ROOT_DIR}/scripts/init-module.sh"
		else
			expect_unchanged_failure "${invalid_http_idempotency}" env CODEOWNER=@acme/platform HTTP_IDEMPOTENCY="${invalid}" \
				bash "${ROOT_DIR}/scripts/init-module.sh"
		fi
	done
	invalid_http_idempotency_database="$(copy_template_checkout http-idempotency-database git@github.com:acme/http-idempotency-database.git)"
	expect_unchanged_failure "${invalid_http_idempotency_database}" env CODEOWNER=@acme/platform DATABASE=none HTTP_IDEMPOTENCY=postgres \
		bash "${ROOT_DIR}/scripts/init-module.sh"

	combinations=('none none none')
	# profile:messaging-nats-jetstream:start
	combinations+=('oidc-jwt postgres nats-jetstream')
	# profile:messaging-nats-jetstream:end
	for combination in "${combinations[@]}"; do
		read -r authn outbox selected_profile <<<"${combination}"
		init_env=(CODEOWNER=@acme/platform DATABASE=postgres HTTP_IDEMPOTENCY=postgres AUTHN="${authn}" OUTBOX="${outbox}")
		failure_env=(CODEOWNER=@acme/platform DATABASE=postgres HTTP_IDEMPOTENCY=none AUTHN="${authn}" OUTBOX="${outbox}")
		# profile:messaging-nats-jetstream:start
		if [[ "${selected_profile}" != "none" ]]; then
			init_env+=(MESSAGING="${selected_profile}")
			failure_env+=(MESSAGING="${selected_profile}")
		fi
		# profile:messaging-nats-jetstream:end
		selected_http_idempotency="$(copy_template_checkout "http-idempotency-${authn}-${outbox}-${selected_profile}" "git@github.com:acme/http-idempotency-${authn}-${outbox}-${selected_profile}.git")"
		(
			cd "${selected_http_idempotency}"
			env "${init_env[@]}" \
				bash ./scripts/init-module.sh
			go test -vet=off ./...
			go build ./cmd/service
			make openapi-check
			make sqlc-check
		)
		assert_http_idempotency_postgres_tree "${selected_http_idempotency}"
		selected_snapshot="$(snapshot "${selected_http_idempotency}")"
		(
			cd "${selected_http_idempotency}"
			env "${init_env[@]}" \
				bash ./scripts/init-module.sh
		)
		assert "repeated HTTP idempotency initialization changed the checkout" \
			same_text "${selected_snapshot}" "$(snapshot "${selected_http_idempotency}")"
		expect_unchanged_failure "${selected_http_idempotency}" env "${failure_env[@]}" \
			bash "${ROOT_DIR}/scripts/init-module.sh"
	done
fi
# profile:http-idempotency-postgres:end

# profile:object-storage:start
if [[ "${TEMPLATE_INIT_PROFILE}" == "all" || "${TEMPLATE_INIT_PROFILE}" == "object-storage" ]]; then
	assert_object_storage_none_tree() {
		local root="$1"
		for removed in \
			cmd/service/internal/bootstrap/startup_object_storage.go \
			cmd/service/internal/bootstrap/startup_object_storage_test.go \
			internal/config/object_storage_config.go \
			internal/config/object_storage_config_test.go \
			internal/objectstorage \
			internal/infra/s3 \
			scripts/ci/s3-source-receipt.sh \
			test/s3conformance/conformance_test.go \
			docs/s3-compatible-object-storage.md; do
			assert "OBJECT_STORAGE=none retained ${removed}" path_absent "${root}/${removed}"
		done
		assert "OBJECT_STORAGE=none retained AWS dependencies" grep_absent -E \
			'aws-sdk-go-v2|smithy-go' "${root}/go.mod" "${root}/go.sum"
		assert "OBJECT_STORAGE=none retained object configuration" grep_absent -R -E \
			'object_storage|OBJECT_STORAGE|objectstorage|infra/s3|test-s3-' \
			"${root}/README.md" "${root}/Makefile" "${root}/cmd" "${root}/docs" "${root}/env" "${root}/internal" "${root}/test"
		assert "OBJECT_STORAGE=none retained a marker" grep_absent -R -Fq \
			'profile:object-storage:' \
			"${root}/.github" "${root}/Makefile" "${root}/README.md" "${root}/cmd" "${root}/docs" "${root}/env" "${root}/internal" "${root}/scripts/ci" "${root}/test"
	}

	assert_object_storage_s3_tree() {
		local root="$1"
		for retained in \
			cmd/service/internal/bootstrap/startup_object_storage.go \
			cmd/service/internal/bootstrap/startup_object_storage_test.go \
			internal/config/object_storage_config.go \
			internal/config/object_storage_config_test.go \
			internal/objectstorage \
			internal/infra/s3 \
			internal/infra/s3/image_root_bundle.go \
			internal/infra/s3/image_root_bundle_test.go \
			internal/infra/httpclient \
			scripts/ci/s3-source-receipt.sh \
			test/s3conformance/conformance_test.go \
			docs/s3-compatible-object-storage.md; do
			assert "OBJECT_STORAGE=s3 removed ${retained}" path_present "${root}/${retained}"
		done
		assert "OBJECT_STORAGE=s3 removed generic RootCAs policy" grep -Fq \
			'RootCAs' "${root}/internal/infra/httpclient/config.go"
		assert "OBJECT_STORAGE=s3 removed AWS dependencies" grep -Eq \
			'github.com/aws/aws-sdk-go-v2' "${root}/go.mod"
		assert "OBJECT_STORAGE=s3 retained a rejected client" grep_absent -R -E \
			'rhnvrm/simples3|kelindar/s3|manager.NewUploader|manager.NewDownloader' \
			"${root}/go.mod" "${root}/go.sum" "${root}/internal"
		assert "OBJECT_STORAGE=s3 generated a trust configuration path" grep_absent -R -E \
			'object_storage.*(ca|root)|OBJECT_STORAGE.*(CA|ROOT)' \
			"${root}/docs/s3-compatible-object-storage.md" \
			"${root}/env/.env.example" \
			"${root}/env/config/local.yaml" \
			"${root}/internal/config/object_storage_config.go"
		assert "OBJECT_STORAGE=s3 retained a marker" grep_absent -R -Eq \
			'profile:object-storage:(start|end)' \
			"${root}/.github" "${root}/Makefile" "${root}/README.md" "${root}/cmd" "${root}/docs" "${root}/env" "${root}/internal" "${root}/scripts/ci" "${root}/test"
	}

	for invalid in '' custom; do
		invalid_name=empty
		if [[ -n "${invalid}" ]]; then
			invalid_name="${invalid}"
		fi
		invalid_object_storage="$(copy_template_checkout "object-storage-invalid-${invalid_name}" "git@github.com:acme/object-storage-invalid-${invalid_name}.git")"
		if [[ -z "${invalid}" ]]; then
			expect_unchanged_failure "${invalid_object_storage}" env CODEOWNER=@acme/platform OBJECT_STORAGE= \
				bash "${ROOT_DIR}/scripts/init-module.sh"
		else
			expect_unchanged_failure "${invalid_object_storage}" env CODEOWNER=@acme/platform OBJECT_STORAGE="${invalid}" \
				bash "${ROOT_DIR}/scripts/init-module.sh"
		fi
	done

	for object_storage in none s3; do
		for outbound_http in none bounded; do
			checkout="$(copy_template_checkout "object-storage-${object_storage}-${outbound_http}" "git@github.com:acme/object-storage-${object_storage}-${outbound_http}.git")"
			init_log="${TEMP_ROOT}/object-storage-${object_storage}-${outbound_http}.log"
			(
				cd "${checkout}"
				CODEOWNER=@acme/platform DATABASE=postgres AUTHN=none OUTBOUND_HTTP="${outbound_http}" OBJECT_STORAGE="${object_storage}" \
					bash ./scripts/init-module.sh >"${init_log}"
				go test -vet=off ./...
				go test -vet=off -tags=integration ./test -run '^TestS3ObjectStorageConformanceRequiresProviderCertification$' -count=1
				go mod tidy
				make mod-tidy-check project-structure-check
			)
			grep -Fqx "object_storage = \"${object_storage}\"" "${checkout}/template.lock"
			grep -Fq "  object storage: ${object_storage}" "${init_log}"
			if [[ "${object_storage}" == "none" ]]; then
				assert_object_storage_none_tree "${checkout}"
				printf '%s\n' \
					'package config' \
					'import "testing"' \
					'func TestObjectStorageProfileRejectsUnknownConfig(t *testing.T) {' \
					'  resetConfigEnv(t)' \
					'  path := writeTempConfig(t, "object_storage:\\n  provider: amazon_s3")' \
					'  if _, _, err := LoadDetailed(LoadOptions{ConfigPath: path}); err == nil { t.Fatal("object-storage config was accepted") }' \
					'}' >"${checkout}/internal/config/object_storage_profile_test.go"
				(
					cd "${checkout}"
					go test -vet=off ./internal/config -run '^TestObjectStorageProfileRejectsUnknownConfig$' -count=1
				)
				rm -f "${checkout}/internal/config/object_storage_profile_test.go"
			else
				assert_object_storage_s3_tree "${checkout}"
			fi
			snapshot_before_repeat="$(snapshot "${checkout}")"
			(
				cd "${checkout}"
				CODEOWNER=@acme/platform DATABASE=postgres AUTHN=none OUTBOUND_HTTP="${outbound_http}" OBJECT_STORAGE="${object_storage}" \
					bash ./scripts/init-module.sh
			)
			assert "repeated OBJECT_STORAGE=${object_storage} initialization changed the checkout" \
				same_text "${snapshot_before_repeat}" "$(snapshot "${checkout}")"
			other_object_storage=none
			if [[ "${object_storage}" == "none" ]]; then
				other_object_storage=s3
			fi
			expect_unchanged_failure "${checkout}" env CODEOWNER=@acme/platform DATABASE=postgres AUTHN=none OUTBOUND_HTTP="${outbound_http}" OBJECT_STORAGE="${other_object_storage}" \
				bash "${ROOT_DIR}/scripts/init-module.sh"
		done
	done

	authn_control="$(copy_template_checkout object-storage-authn-control git@github.com:acme/object-storage-authn-control.git)"
	(
		cd "${authn_control}"
		CODEOWNER=@acme/platform DATABASE=postgres AUTHN=oidc-jwt OUTBOUND_HTTP=none OBJECT_STORAGE=none \
			bash ./scripts/init-module.sh
		go test -vet=off ./...
	)
	assert "AUTHN-only output removed shared HTTP client" path_present "${authn_control}/internal/infra/httpclient"
	assert_object_storage_none_tree "${authn_control}"
fi
# profile:object-storage:end

# profile:jobs-postgres:start
if [[ "${TEMPLATE_INIT_PROFILE}" == "all" || "${TEMPLATE_INIT_PROFILE}" == "jobs" ]]; then
	jobs_marker='profile:jobs''-postgres:'
	assert_jobs_none_tree() {
		local root="$1"
		for removed in \
			cmd/jobs-worker \
			internal/config/jobs_config.go \
			internal/config/jobs_worker_config.go \
			migrations/000008_river.sql \
			migrations/000004_postgres_jobs.sql \
			docs/postgres-durable-background-jobs.md; do
			assert "JOBS=none retained ${removed}" path_absent "${root}/${removed}"
		done
		assert "JOBS=none retained jobs integration proof" glob_absent "${root}/test/postgres_jobs_*_test.go"
		assert "JOBS=none retained jobs markers" grep_absent -R -Fq \
			"${jobs_marker}" "${root}/.golangci.yml" "${root}/Makefile" "${root}/build" "${root}/cmd" \
			"${root}/docs" "${root}/env" "${root}/internal" "${root}/scripts/ci"
	}

	assert_jobs_postgres_tree() {
		local root="$1"
		for retained in \
			cmd/jobs-worker \
			internal/config/jobs_config.go \
			internal/config/jobs_worker_config.go \
			migrations/000008_river.sql \
			migrations/000004_postgres_jobs.sql \
			docs/postgres-durable-background-jobs.md; do
			assert "JOBS=postgres removed ${retained}" path_present "${root}/${retained}"
		done
		grep -Fqx 'jobs = "postgres"' "${root}/template.lock"
		assert "JOBS=postgres retained unresolved markers" grep_absent -R -Fq \
			"${jobs_marker}" "${root}/.golangci.yml" "${root}/Makefile" "${root}/build" "${root}/cmd" \
			"${root}/docs" "${root}/env" "${root}/internal" "${root}/scripts/ci"
	}

	jobs_none="$(copy_template_checkout jobs-none git@github.com:acme/jobs-none.git)"
	(
		cd "${jobs_none}"
		CODEOWNER=@acme/platform DATABASE=postgres bash ./scripts/init-module.sh
		go test -vet=off ./...
		make mod-tidy-check project-structure-check
	)
	assert_jobs_none_tree "${jobs_none}"
	jobs_none_snapshot="$(snapshot "${jobs_none}")"
	(
		cd "${jobs_none}"
		CODEOWNER=@acme/platform DATABASE=postgres bash ./scripts/init-module.sh
	)
	assert "repeated JOBS=none initialization changed the checkout" \
		same_text "${jobs_none_snapshot}" "$(snapshot "${jobs_none}")"

	for invalid in '' custom; do
		invalid_name=empty
		if [[ -n "${invalid}" ]]; then
			invalid_name="${invalid}"
		fi
		invalid_jobs="$(copy_template_checkout "jobs-invalid-${invalid_name}" "git@github.com:acme/jobs-invalid-${invalid_name}.git")"
		if [[ -z "${invalid}" ]]; then
			expect_unchanged_failure "${invalid_jobs}" env CODEOWNER=@acme/platform JOBS= \
				bash "${ROOT_DIR}/scripts/init-module.sh"
		else
			expect_unchanged_failure "${invalid_jobs}" env CODEOWNER=@acme/platform JOBS="${invalid}" \
				bash "${ROOT_DIR}/scripts/init-module.sh"
		fi
	done
	jobs_without_postgres="$(copy_template_checkout jobs-without-postgres git@github.com:acme/jobs-without-postgres.git)"
	expect_unchanged_failure "${jobs_without_postgres}" env CODEOWNER=@acme/platform DATABASE=none JOBS=postgres \
		bash "${ROOT_DIR}/scripts/init-module.sh"

	prove_jobs_profile() {
		local name="$1"
		local outbox="$2"
		local messaging="$3"
		local full="${4:-false}"
		local checkout sibling choice path jobs_snapshot retained
		local extra_profiles=(REFERENCE_EXAMPLE=remove)
		if [[ "${full}" == "true" ]]; then
			extra_profiles=(
				HTTP_IDEMPOTENCY=postgres
				WEBHOOKS=durable
				GRPC=enabled
				AUTHN=oidc-jwt
				OUTBOUND_HTTP=bounded
				OBJECT_STORAGE=s3
				OUTBOUND_AUTH=oauth2-client-credentials
				REFERENCE_EXAMPLE=remove
			)
		fi

		checkout="$(copy_template_checkout "jobs-${name}" "git@github.com:acme/jobs-${name}.git")"
		(
			cd "${checkout}"
			env CODEOWNER=@acme/platform DATABASE=postgres JOBS=postgres OUTBOX="${outbox}" MESSAGING="${messaging}" \
				"${extra_profiles[@]}" bash ./scripts/init-module.sh
			go test -vet=off ./cmd/jobs-worker/...
			go test -vet=off -run '^$' ./...
			make mod-tidy-check project-structure-check
		)
		assert_jobs_postgres_tree "${checkout}"
		for sibling in \
			"${outbox}:internal/infra/postgresoutbox" \
			"${messaging}:cmd/worker"; do
			choice="${sibling%%:*}"
			path="${sibling#*:}"
			if [[ "${choice}" == "none" ]]; then
				assert "jobs ${name} retained disabled sibling ${path}" path_absent "${checkout}/${path}"
			else
				assert "jobs ${name} removed selected sibling ${path}" path_present "${checkout}/${path}"
			fi
		done
		if [[ "${full}" == "true" ]]; then
			for retained in \
				internal/infra/postgreswebhook \
				internal/httpidempotency \
				internal/infra/oidcjwt \
				internal/infra/oauth2clientcredentials \
				internal/objectstorage \
				buf.yaml; do
				assert "jobs ${name} removed selected profile ${retained}" path_present "${checkout}/${retained}"
			done
		fi
		jobs_snapshot="$(snapshot "${checkout}")"
		(
			cd "${checkout}"
			env CODEOWNER=@acme/platform DATABASE=postgres JOBS=postgres OUTBOX="${outbox}" MESSAGING="${messaging}" \
				"${extra_profiles[@]}" bash ./scripts/init-module.sh
		)
		assert "repeated jobs ${name} initialization changed the checkout" \
			same_text "${jobs_snapshot}" "$(snapshot "${checkout}")"
	}

	prove_jobs_profile only none none
	prove_jobs_profile outbox postgres nats-jetstream
	prove_jobs_profile messaging none nats-jetstream
	prove_jobs_profile combined postgres nats-jetstream true
fi
# profile:jobs-postgres:end

# profile:webhooks-durable:start
if [[ "${TEMPLATE_INIT_PROFILE}" == "all" || "${TEMPLATE_INIT_PROFILE}" == "webhooks" ]]; then
	webhook_paths=(
		docs/outbound-webhook-delivery.md
		internal/config/webhooks_config.go
		internal/infra/postgreswebhook
		internal/outboundtrust
		migrations/000005_postgres_webhooks.sql
		migrations/000006_postgres_webhook_reference_repairs.sql
		migrations/000007_postgres_webhooks_retire.sql
		test/postgres_webhook_acceptance_integration_test.go
		test/webhook_network_integration_test.go
	)

	webhooks_none="$(copy_template_checkout webhooks-none git@github.com:acme/webhooks-none.git)"
	(
		cd "${webhooks_none}"
		CODEOWNER=@acme/platform DATABASE=postgres bash ./scripts/init-module.sh
		go test -vet=off ./...
		make sqlc-check mod-tidy-check project-structure-check
	)
	for removed in "${webhook_paths[@]}"; do
		assert "WEBHOOKS=none retained ${removed}" path_absent "${webhooks_none}/${removed}"
	done
	grep -Fqx 'webhooks = "none"' "${webhooks_none}/template.lock"
	assert "WEBHOOKS=none retained webhook profile markers" grep_absent -R -Fq \
		'profile:webhooks-'"durable:" "${webhooks_none}/Makefile" "${webhooks_none}/README.md" \
		"${webhooks_none}/build" "${webhooks_none}/docs" "${webhooks_none}/env" \
		"${webhooks_none}/internal" "${webhooks_none}/scripts/ci"

	for invalid in '' custom; do
		invalid_name="${invalid:-empty}"
		invalid_webhooks="$(copy_template_checkout "webhooks-invalid-${invalid_name}" "git@github.com:acme/webhooks-invalid-${invalid_name}.git")"
		expect_unchanged_failure "${invalid_webhooks}" env CODEOWNER=@acme/platform WEBHOOKS="${invalid}" \
			bash "${ROOT_DIR}/scripts/init-module.sh"
	done
	webhooks_without_postgres="$(copy_template_checkout webhooks-without-postgres git@github.com:acme/webhooks-without-postgres.git)"
	expect_unchanged_failure "${webhooks_without_postgres}" env CODEOWNER=@acme/platform DATABASE=none JOBS=postgres WEBHOOKS=durable \
		bash "${ROOT_DIR}/scripts/init-module.sh"
	webhooks_without_jobs="$(copy_template_checkout webhooks-without-jobs git@github.com:acme/webhooks-without-jobs.git)"
	expect_unchanged_failure "${webhooks_without_jobs}" env CODEOWNER=@acme/platform DATABASE=postgres JOBS=none WEBHOOKS=durable \
		bash "${ROOT_DIR}/scripts/init-module.sh"

	webhooks_durable="$(copy_template_checkout webhooks-durable git@github.com:acme/webhooks-durable.git)"
	(
		cd "${webhooks_durable}"
		CODEOWNER=@acme/platform DATABASE=postgres JOBS=postgres WEBHOOKS=durable bash ./scripts/init-module.sh
		go test -vet=off ./...
		go build ./cmd/jobs-worker ./cmd/migrate
		make sqlc-check mod-tidy-check project-structure-check
	)
	for retained in "${webhook_paths[@]}"; do
		assert "WEBHOOKS=durable removed ${retained}" path_present "${webhooks_durable}/${retained}"
	done
	grep -Fqx 'webhooks = "durable"' "${webhooks_durable}/template.lock"
	assert "WEBHOOKS=durable retained unresolved markers" grep_absent -R -Fq \
		'profile:webhooks-'"durable:" "${webhooks_durable}/Makefile" "${webhooks_durable}/README.md" \
		"${webhooks_durable}/build" "${webhooks_durable}/docs" "${webhooks_durable}/env" \
		"${webhooks_durable}/internal" "${webhooks_durable}/scripts/ci"
	webhooks_snapshot="$(snapshot "${webhooks_durable}")"
	(
		cd "${webhooks_durable}"
		CODEOWNER=@acme/platform DATABASE=postgres JOBS=postgres WEBHOOKS=durable bash ./scripts/init-module.sh
	)
	assert "repeated WEBHOOKS=durable initialization changed the checkout" \
		same_text "${webhooks_snapshot}" "$(snapshot "${webhooks_durable}")"
fi
# profile:webhooks-durable:end

echo "template initialization contract passed"
