#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
TOOLING_IMAGES_FILE="${ROOT_DIR}/build/docker/tooling-images.Dockerfile"

read_catalog_image() {
	local stage_name="$1"
	awk -v stage_name="${stage_name}" '
		BEGIN { wanted = tolower(stage_name) }
		toupper($1) == "FROM" {
			image = $2
			current_stage = ""
			if (NF >= 4 && tolower($3) == "as") {
				current_stage = $4
			}
			if (tolower(current_stage) == wanted) {
				print image
				found = 1
				exit 0
			}
		}
		END { if (!found) exit 1 }
	' "${TOOLING_IMAGES_FILE}"
}

require_catalog_image() {
	local stage_name="$1"
	local image

	if [[ ! -f "${TOOLING_IMAGES_FILE}" ]]; then
		echo "tooling image catalog not found: ${TOOLING_IMAGES_FILE}"
		exit 1
	fi

	image="$(read_catalog_image "${stage_name}" || true)"
	if [[ -z "${image}" ]]; then
		echo "tooling image catalog is missing stage '${stage_name}' in ${TOOLING_IMAGES_FILE}"
		exit 1
	fi

	printf '%s' "${image}"
}

GO_IMAGE_DEFAULT="$(require_catalog_image go_toolchain)"
NODE_IMAGE_DEFAULT="$(require_catalog_image node_toolchain)"
POSTGRES_IMAGE_DEFAULT="$(require_catalog_image postgres_tool)"
MIGRATE_IMAGE_DEFAULT="$(require_catalog_image migrate_tool)"
TRIVY_IMAGE_DEFAULT="$(require_catalog_image trivy_tool)"

GO_IMAGE="${GO_IMAGE:-${GO_IMAGE_DEFAULT}}"
NODE_IMAGE="${NODE_IMAGE:-${NODE_IMAGE_DEFAULT}}"
POSTGRES_IMAGE="${POSTGRES_IMAGE:-${POSTGRES_IMAGE_DEFAULT}}"
MIGRATE_IMAGE="${MIGRATE_IMAGE:-${MIGRATE_IMAGE_DEFAULT}}"
TRIVY_IMAGE="${TRIVY_IMAGE:-${TRIVY_IMAGE_DEFAULT}}"
REDOCLY_CLI_VERSION="${REDOCLY_CLI_VERSION:-2.20.3}"
TEST_REPORT_DIR="${TEST_REPORT_DIR:-.artifacts/test}"
TEST_JUNIT_FILE="${TEST_JUNIT_FILE:-${TEST_REPORT_DIR}/junit.xml}"
TEST_JSON_FILE="${TEST_JSON_FILE:-${TEST_REPORT_DIR}/test2json.json}"
COVERAGE_MIN="${COVERAGE_MIN:-80.0}"
COVERAGE_EXCLUDE_REGEX="${COVERAGE_EXCLUDE_REGEX:-(^|/)internal/api/openapi\\.gen\\.go:|(^|/)internal/infra/postgres/sqlcgen/|(^|/)cmd/service/main\\.go:|(^|/)cmd/migrate/main\\.go:}"
FUZZ_TIME="${FUZZ_TIME:-45s}"
GO_FILES_FIND="find . -type f -name '*.go' -not -path './vendor/*' -not -path './.cache/*'"
GOFUMPT_FILES_FIND="${GO_FILES_FIND} -not -path './internal/api/openapi.gen.go' -not -path './internal/infra/postgres/sqlcgen/*'"

host_uid="$(id -u 2>/dev/null || echo 0)"
host_gid="$(id -g 2>/dev/null || echo 0)"

usage() {
	echo "usage: $0 <command> [args]"
	echo "commands:"
	echo "  doctor"
	echo "  pull-images"
	echo "  init-module [module-path]   (uses CODEOWNER env optionally; auto-detects from origin when omitted)"
	echo "  mod-check"
	echo "  fmt"
	echo "  fmt-check"
	echo "  test"
	echo "  test-summary"
	echo "  vet"
	echo "  test-race"
	echo "  test-cover"
	echo "  test-report"
	echo "  test-fuzz-smoke"
	echo "  test-flake-smoke"
	echo "  test-integration"
	echo "  sqlc-generate"
	echo "  sqlc-check"
	echo "  lint"
	echo "  modernize-check"
	echo "  test-parallelism-check"
	echo "  openapi-generate"
	echo "  openapi-drift-check"
	echo "  openapi-runtime-contract-check"
	echo "  openapi-lint"
	echo "  openapi-validate"
	echo "  openapi-breaking <base-openapi>"
	echo "  openapi-check"
	echo "  govulncheck"
	echo "  gosec"
	echo "  go-security"
	echo "  secret-scan"
	echo "  workflow-routing-check"
	echo "  guardrails-check"
	echo "  migration-validate"
	echo "  container-security"
	echo "  ci"
}

ensure_docker() {
	if ! command -v docker >/dev/null 2>&1; then
		echo "docker is required for zero-setup mode"
		exit 1
	fi
	if ! docker info >/dev/null 2>&1; then
		echo "docker daemon is not reachable. start Docker Desktop/Engine and retry"
		exit 1
	fi
}

run_go() {
	ensure_docker
	mkdir -p "${ROOT_DIR}/.cache/go-mod" "${ROOT_DIR}/.cache/go-build"
	docker run \
		--rm \
		-u "${host_uid}:${host_gid}" \
		-v "${ROOT_DIR}:/workspace" \
		-w /workspace \
		-e GOMODCACHE=/workspace/.cache/go-mod \
		-e GOCACHE=/workspace/.cache/go-build \
		"${GO_IMAGE}" \
		bash -lc "export PATH=/usr/local/go/bin:\${PATH}; $*"
}

docker_socket_path() {
	local candidate
	if [[ "${DOCKER_HOST:-}" == unix://* ]]; then
		candidate="${DOCKER_HOST#unix://}"
		if [[ -S "${candidate}" ]]; then
			printf '%s' "${candidate}"
			return 0
		fi
	fi

	local context_host
	context_host="$(docker context inspect --format '{{ (index .Endpoints "docker").Host }}' 2>/dev/null || true)"
	if [[ "${context_host}" == unix://* ]]; then
		candidate="${context_host#unix://}"
		if [[ -S "${candidate}" ]]; then
			printf '%s' "${candidate}"
			return 0
		fi
	fi

	for candidate in /var/run/docker.sock "${HOME}/.docker/run/docker.sock" "${HOME}/.orbstack/run/docker.sock"; do
		if [[ -S "${candidate}" ]]; then
			printf '%s' "${candidate}"
			return 0
		fi
	done
}

run_go_with_docker_socket() {
	ensure_docker
	mkdir -p "${ROOT_DIR}/.cache/go-mod" "${ROOT_DIR}/.cache/go-build"

	local docker_args=(
		--rm
		-u "${host_uid}:${host_gid}"
		-v "${ROOT_DIR}:/workspace"
		-w /workspace
		-e GOMODCACHE=/workspace/.cache/go-mod
		-e GOCACHE=/workspace/.cache/go-build
	)
	local docker_socket
	docker_socket="$(docker_socket_path || true)"
	if [[ -n "${docker_socket}" ]]; then
		docker_args+=(
			-v "${docker_socket}:/var/run/docker.sock"
			-e DOCKER_HOST=unix:///var/run/docker.sock
			--group-add 0
		)
	fi

	docker run \
		"${docker_args[@]}" \
		"${GO_IMAGE}" \
		bash -lc "export PATH=/usr/local/go/bin:\${PATH}; $*"
}

run_node() {
	ensure_docker
	mkdir -p "${ROOT_DIR}/.cache/npm"
	docker run \
		--rm \
		-u "${host_uid}:${host_gid}" \
		-v "${ROOT_DIR}:/workspace" \
		-w /workspace \
		-e npm_config_cache=/workspace/.cache/npm \
		"${NODE_IMAGE}" \
		bash -lc "$*"
}

run_lint() {
	mkdir -p "${ROOT_DIR}/.cache/golangci-lint"
	run_go "GOLANGCI_LINT_CACHE=/workspace/.cache/golangci-lint go tool golangci-lint $*"
}

run_gosec() {
	run_go "GOCACHE=\$(mktemp -d) go tool gosec -exclude-generated -exclude-dir=.cache ./..."
}

generated_drift_check() {
	bash "${ROOT_DIR}/scripts/ci/generated-drift-check.sh" "$1"
}

has_sqlc_queries() {
	find "${ROOT_DIR}/internal/infra/postgres/queries" -type f -name '*.sql' -print -quit | grep -q .
}

run_coverage_check() {
	run_go "test -f coverage.out || (echo \"coverage.out not found; run 'test-cover' or 'test-report' first\"; exit 1); filtered_cov=\$(mktemp); grep -Ev '${COVERAGE_EXCLUDE_REGEX}' coverage.out > \"\${filtered_cov}\"; total=\$(go tool cover -func=\"\${filtered_cov}\" | awk '/^total:/ {gsub(/%/, \"\", \$3); print \$3}'); rm -f \"\${filtered_cov}\"; if [[ -z \"\${total}\" ]]; then echo \"failed to parse total coverage from coverage.out\"; exit 1; fi; awk -v total=\"\${total}\" -v minimum='${COVERAGE_MIN}' 'BEGIN { if ((total + 0) < (minimum + 0)) { printf \"coverage %.2f%% is below threshold %.2f%%\\n\", total, minimum; exit 1 } printf \"coverage %.2f%% meets threshold %.2f%%\\n\", total, minimum }'"
}

run_test_report() {
	run_go "mkdir -p '${TEST_REPORT_DIR}' && GOCOVERDIR= go tool gotestsum --format=standard-verbose --junitfile='${TEST_JUNIT_FILE}' --jsonfile='${TEST_JSON_FILE}' -- -covermode=atomic -coverprofile=coverage.out ./... && go tool cover -func=coverage.out"
	run_coverage_check
}

run_test_fuzz_smoke() {
	run_go "found=0; pkgs=\$(go list ./...) || exit \$?; for pkg in \${pkgs}; do fuzz_targets=\$(go test \"\${pkg}\" -list '^Fuzz' 2>&1) || { status=\$?; printf '%s\n' \"\${fuzz_targets}\"; exit \${status}; }; if printf '%s\n' \"\${fuzz_targets}\" | grep -q '^Fuzz'; then found=1; echo \"running fuzz smoke for \${pkg}\"; go test \"\${pkg}\" -run '^\$' -fuzz=Fuzz -fuzztime='${FUZZ_TIME}' || exit \$?; fi; done; if [[ \"\${found}\" -eq 0 ]]; then echo 'no fuzz targets found; skipping fuzz smoke run'; fi"
}

wait_for_postgres() {
	local container_name="$1"
	local container_network="$2"
	local attempts=60

	for _ in $(seq 1 "${attempts}"); do
		if docker run \
			--rm \
			--network "${container_network}" \
			"${POSTGRES_IMAGE}" \
			pg_isready -h "${container_name}" -U app -d app >/dev/null 2>&1; then
			return 0
		fi
		sleep 1
	done

	echo "postgres container did not become ready in time"
	return 1
}

run_migration_validate() {
	ensure_docker

	local network_name="go-service-template-migration-${host_uid}-$$"
	local postgres_container="go-service-template-postgres-${host_uid}-$$"
	local migration_dsn="postgres://app:app@${postgres_container}:5432/app?sslmode=disable"
	local runtime_image="go-service-template-rest:migration-smoke-${host_uid}-$$"

	cleanup_migration() {
		local container_name="$1"
		local container_network="$2"
		local image_name="$3"

		docker rm -f "${container_name}" >/dev/null 2>&1 || true
		docker network rm "${container_network}" >/dev/null 2>&1 || true
		docker image rm -f "${image_name}" >/dev/null 2>&1 || true
	}
	trap "cleanup_migration '${postgres_container}' '${network_name}' '${runtime_image}'" EXIT

	docker network create "${network_name}" >/dev/null
	docker run \
		-d \
		--name "${postgres_container}" \
		--network "${network_name}" \
		-e POSTGRES_DB=app \
		-e POSTGRES_USER=app \
		-e POSTGRES_PASSWORD=app \
		"${POSTGRES_IMAGE}" >/dev/null

	wait_for_postgres "${postgres_container}" "${network_name}"

	docker run \
		--rm \
		--network "${network_name}" \
		-v "${ROOT_DIR}/env/migrations:/migrations:ro" \
		"${MIGRATE_IMAGE}" \
		-path /migrations \
		-database "${migration_dsn}" up

	docker run \
		--rm \
		--network "${network_name}" \
		-v "${ROOT_DIR}/env/migrations:/migrations:ro" \
		"${MIGRATE_IMAGE}" \
		-path /migrations \
		-database "${migration_dsn}" down 1

	docker run \
		--rm \
		--network "${network_name}" \
		-v "${ROOT_DIR}/env/migrations:/migrations:ro" \
		"${MIGRATE_IMAGE}" \
		-path /migrations \
		-database "${migration_dsn}" up 1

	docker build -f "${ROOT_DIR}/build/docker/Dockerfile" -t "${runtime_image}" "${ROOT_DIR}"
	docker run \
		--rm \
		--network "${network_name}" \
		-e APP__POSTGRES__ENABLED=true \
		-e APP__POSTGRES__DSN="${migration_dsn}" \
		--entrypoint /migrate \
		"${runtime_image}"
}

run_container_security_scan() {
	ensure_docker

	local docker_socket
	docker_socket="$(docker_socket_path || true)"
	if [[ -z "${docker_socket}" ]]; then
		echo "docker socket is unavailable; container scan cannot run in docker mode"
		exit 1
	fi

	docker build -f "${ROOT_DIR}/build/docker/Dockerfile" -t service:ci "${ROOT_DIR}"
	docker run \
		--rm \
		-v "${docker_socket}:/var/run/docker.sock" \
		-e DOCKER_HOST=unix:///var/run/docker.sock \
		"${TRIVY_IMAGE}" image \
		--severity HIGH,CRITICAL \
		--ignore-unfixed \
		--exit-code 1 \
		--format table \
		service:ci
}

cmd="${1:-}"
if [[ -z "${cmd}" ]]; then
	usage
	exit 1
fi
shift || true

case "${cmd}" in
doctor)
	bash "${ROOT_DIR}/scripts/dev/doctor.sh" --mode docker
	;;
pull-images)
	ensure_docker
	docker pull "${GO_IMAGE}"
	docker pull "${NODE_IMAGE}"
	docker pull "${POSTGRES_IMAGE}"
	docker pull "${MIGRATE_IMAGE}"
	docker pull "${TRIVY_IMAGE}"
	;;
init-module)
	module_path="${1:-}"
	if [[ $# -gt 1 ]]; then
		echo "init-module accepts at most one argument: [module-path]"
		exit 1
	fi
	ensure_docker
	mkdir -p "${ROOT_DIR}/.cache/go-mod" "${ROOT_DIR}/.cache/go-build"
	if [[ -n "${module_path}" ]]; then
		docker run \
			--rm \
			-u "${host_uid}:${host_gid}" \
			-v "${ROOT_DIR}:/workspace" \
			-w /workspace \
			-e CODEOWNER="${CODEOWNER:-}" \
			-e GOMODCACHE=/workspace/.cache/go-mod \
			-e GOCACHE=/workspace/.cache/go-build \
			"${GO_IMAGE}" \
			bash -lc "export PATH=/usr/local/go/bin:\${PATH}; bash ./scripts/init-module.sh \"${module_path}\""
	else
		docker run \
			--rm \
			-u "${host_uid}:${host_gid}" \
			-v "${ROOT_DIR}:/workspace" \
			-w /workspace \
			-e CODEOWNER="${CODEOWNER:-}" \
			-e GOMODCACHE=/workspace/.cache/go-mod \
			-e GOCACHE=/workspace/.cache/go-build \
			"${GO_IMAGE}" \
			bash -lc "export PATH=/usr/local/go/bin:\${PATH}; bash ./scripts/init-module.sh"
	fi
	;;
mod-check)
	run_go "GOFLAGS= go mod tidy -diff && go mod verify"
	;;
fmt)
	run_go "go tool goimports -w \$(${GO_FILES_FIND}) && go tool gofumpt -w \$(${GOFUMPT_FILES_FIND})"
	;;
fmt-check)
	run_go "unformatted=\$(go tool goimports -l \$(${GO_FILES_FIND})); if [[ -n \"\${unformatted}\" ]]; then echo 'goimports required for:'; echo \"\${unformatted}\"; exit 1; fi; gofumpt_unformatted=\$(go tool gofumpt -l \$(${GOFUMPT_FILES_FIND})); if [[ -n \"\${gofumpt_unformatted}\" ]]; then echo 'gofumpt required for:'; echo \"\${gofumpt_unformatted}\"; exit 1; fi"
	;;
test)
	run_go "go test ./..."
	;;
test-summary)
	run_go "go tool gotestsum --format=pkgname-and-test-fails -- ./..."
	;;
vet)
	run_go "go vet ./..."
	;;
test-race)
	run_go "go test -race ./..."
	;;
test-cover)
	run_go "GOCOVERDIR= go test -covermode=atomic -coverprofile=coverage.out ./... && go tool cover -func=coverage.out"
	;;
test-report)
	run_test_report
	;;
test-fuzz-smoke)
	run_test_fuzz_smoke
	;;
test-flake-smoke)
	run_go "go test -count=5 -shuffle=on ./..."
	;;
test-integration)
	run_go_with_docker_socket "REQUIRE_DOCKER=${REQUIRE_DOCKER:-0} go test -tags=integration ./test/..."
	;;
sqlc-generate)
	if has_sqlc_queries; then
		run_go "go tool github.com/sqlc-dev/sqlc/cmd/sqlc generate -f internal/infra/postgres/sqlc.yaml"
	else
		echo "no sqlc query sources; skipping sqlc generation"
	fi
	;;
sqlc-check)
	bash "${ROOT_DIR}/scripts/dev/docker-tooling.sh" sqlc-generate
	generated_drift_check sqlc
	;;
lint)
	run_go "GOLANGCI_LINT_CACHE=/workspace/.cache/golangci-lint make lint"
	;;
modernize-check)
	run_lint "run --enable-only=modernize --timeout=3m"
	;;
test-parallelism-check)
	run_lint "run --enable-only=paralleltest,tparallel --timeout=3m --max-issues-per-linter=0 --max-same-issues=0"
	;;
openapi-generate)
	run_go "go generate ./internal/api"
	;;
openapi-drift-check)
	generated_drift_check openapi
	;;
openapi-runtime-contract-check)
	run_go "go test ./internal/infra/http -run '^TestOpenAPIRuntimeContract' -count=1"
	;;
openapi-lint)
	run_node "npx @redocly/cli@${REDOCLY_CLI_VERSION} lint --config .redocly.yaml api/openapi/service.yaml"
	;;
openapi-validate)
	run_go "go tool validate -- api/openapi/service.yaml"
	;;
openapi-breaking)
	base_openapi="${1:-${BASE_OPENAPI:-}}"
	if [[ -z "${base_openapi}" ]]; then
		echo "openapi-breaking requires <base-openapi> or BASE_OPENAPI"
		exit 1
	fi
	if [[ -f "${ROOT_DIR}/${base_openapi}" ]]; then
		workspace_base_openapi="${base_openapi}"
	elif [[ -f "${base_openapi}" ]]; then
		mkdir -p "${ROOT_DIR}/.cache"
		temp_base_openapi="$(mktemp "${ROOT_DIR}/.cache/openapi-base.XXXXXX")"
		cp "${base_openapi}" "${temp_base_openapi}"
		trap 'rm -f "${temp_base_openapi}"' EXIT
		workspace_base_openapi=".cache/$(basename "${temp_base_openapi}")"
	else
		echo "base OpenAPI spec not found: ${base_openapi}"
		exit 1
	fi
	run_go "go tool oasdiff breaking --fail-on ERR '${workspace_base_openapi}' api/openapi/service.yaml"
	;;
openapi-check)
	bash "${ROOT_DIR}/scripts/dev/docker-tooling.sh" openapi-generate
	bash "${ROOT_DIR}/scripts/dev/docker-tooling.sh" openapi-drift-check
	run_go "go test ./internal/api"
	bash "${ROOT_DIR}/scripts/dev/docker-tooling.sh" openapi-runtime-contract-check
	bash "${ROOT_DIR}/scripts/dev/docker-tooling.sh" openapi-lint
	bash "${ROOT_DIR}/scripts/dev/docker-tooling.sh" openapi-validate
	;;
govulncheck)
	run_go "go tool govulncheck ./..."
	;;
gosec)
	run_gosec
	;;
go-security)
	bash "${ROOT_DIR}/scripts/dev/docker-tooling.sh" govulncheck
	bash "${ROOT_DIR}/scripts/dev/docker-tooling.sh" gosec
	;;
secret-scan)
	run_go "go tool gitleaks git --no-banner --redact --exit-code 1 --baseline-path .gitleaks.baseline.json ."
	;;
workflow-routing-check)
	# Keep the Go checker in the pinned toolchain image. The shell-only guards
	# use host jq and filesystem semantics; the base Go image intentionally does
	# not bundle those tools.
	run_go "go run ./scripts/ci/hard-skills-check"
	bash "${ROOT_DIR}/scripts/ci/workflow-instructions-check.sh"
	;;
instruction-evals-harness)
	bash "${ROOT_DIR}/scripts/ci/instruction-evals-check.sh"
	;;
guardrails-check)
	run_go "bash ./scripts/ci/required-guardrails-check.sh"
	;;
migration-validate)
	run_migration_validate
	;;
container-security)
	run_container_security_scan
	;;
ci)
	bash "${ROOT_DIR}/scripts/dev/docker-tooling.sh" mod-check
	bash "${ROOT_DIR}/scripts/dev/docker-tooling.sh" workflow-routing-check
	bash "${ROOT_DIR}/scripts/dev/docker-tooling.sh" guardrails-check
	bash "${ROOT_DIR}/scripts/dev/docker-tooling.sh" fmt-check
	bash "${ROOT_DIR}/scripts/dev/docker-tooling.sh" lint
	bash "${ROOT_DIR}/scripts/dev/docker-tooling.sh" test
	bash "${ROOT_DIR}/scripts/dev/docker-tooling.sh" vet
	bash "${ROOT_DIR}/scripts/dev/docker-tooling.sh" test-race
	bash "${ROOT_DIR}/scripts/dev/docker-tooling.sh" test-report
	bash "${ROOT_DIR}/scripts/dev/docker-tooling.sh" sqlc-check
	REQUIRE_DOCKER=1 bash "${ROOT_DIR}/scripts/dev/docker-tooling.sh" test-integration
	bash "${ROOT_DIR}/scripts/dev/docker-tooling.sh" openapi-check
	bash "${ROOT_DIR}/scripts/dev/docker-tooling.sh" go-security
	bash "${ROOT_DIR}/scripts/dev/docker-tooling.sh" secret-scan
	bash "${ROOT_DIR}/scripts/dev/docker-tooling.sh" migration-validate
	bash "${ROOT_DIR}/scripts/dev/docker-tooling.sh" container-security
	;;
*)
	echo "unknown command: ${cmd}"
	usage
	exit 1
	;;
esac
