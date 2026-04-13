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
GOLANGCI_LINT_IMAGE_DEFAULT="$(require_catalog_image golangci_lint_tool)"
POSTGRES_IMAGE_DEFAULT="$(require_catalog_image postgres_tool)"
MIGRATE_IMAGE_DEFAULT="$(require_catalog_image migrate_tool)"
TRIVY_IMAGE_DEFAULT="$(require_catalog_image trivy_tool)"

GO_IMAGE="${GO_IMAGE:-${GO_IMAGE_DEFAULT}}"
NODE_IMAGE="${NODE_IMAGE:-${NODE_IMAGE_DEFAULT}}"
GOLANGCI_LINT_IMAGE="${GOLANGCI_LINT_IMAGE:-${GOLANGCI_LINT_IMAGE_DEFAULT}}"
POSTGRES_IMAGE="${POSTGRES_IMAGE:-${POSTGRES_IMAGE_DEFAULT}}"
MIGRATE_IMAGE="${MIGRATE_IMAGE:-${MIGRATE_IMAGE_DEFAULT}}"
TRIVY_IMAGE="${TRIVY_IMAGE:-${TRIVY_IMAGE_DEFAULT}}"
REDOCLY_CLI_VERSION="${REDOCLY_CLI_VERSION:-2.20.3}"
KIN_OPENAPI_VALIDATE_VERSION="${KIN_OPENAPI_VALIDATE_VERSION:-v0.133.0}"
GOVULNCHECK_VERSION="${GOVULNCHECK_VERSION:-v1.1.4}"
GOSEC_VERSION="${GOSEC_VERSION:-v2.24.7}"
GOIMPORTS_VERSION="${GOIMPORTS_VERSION:-v0.42.0}"
GITLEAKS_VERSION="${GITLEAKS_VERSION:-v8.30.0}"
TEST_REPORT_DIR="${TEST_REPORT_DIR:-.artifacts/test}"
TEST_JUNIT_FILE="${TEST_JUNIT_FILE:-${TEST_REPORT_DIR}/junit.xml}"
TEST_JSON_FILE="${TEST_JSON_FILE:-${TEST_REPORT_DIR}/test2json.json}"
COVERAGE_MIN="${COVERAGE_MIN:-65.0}"
COVERAGE_EXCLUDE_REGEX="${COVERAGE_EXCLUDE_REGEX:-(^|/)internal/api/openapi\\.gen\\.go:|(^|/)cmd/service/main\\.go:}"

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
	echo "  vet"
	echo "  test-race"
	echo "  test-cover"
	echo "  test-report"
	echo "  test-integration"
	echo "  stringer-generate"
	echo "  stringer-drift-check"
	echo "  sqlc-generate"
	echo "  sqlc-check"
	echo "  mocks-generate"
	echo "  mocks-drift-check"
	echo "  lint"
	echo "  openapi-generate"
	echo "  openapi-drift-check"
	echo "  openapi-runtime-contract-check"
	echo "  openapi-lint"
	echo "  openapi-validate"
	echo "  openapi-check"
	echo "  go-security"
	echo "  secrets-scan"
	echo "  guardrails-check"
	echo "  skills-check"
	echo "  agents-check"
	echo "  docs-drift-check <base-ref> <head-ref>"
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

run_go_with_docker_socket() {
	ensure_docker
	mkdir -p "${ROOT_DIR}/.cache/go-mod" "${ROOT_DIR}/.cache/go-build"

	socket_args=()
	if [[ -S /var/run/docker.sock ]]; then
		socket_args+=(-v /var/run/docker.sock:/var/run/docker.sock --group-add 0)
	fi

	docker run \
		--rm \
		-u "${host_uid}:${host_gid}" \
		-v "${ROOT_DIR}:/workspace" \
		-w /workspace \
		"${socket_args[@]}" \
		-e GOMODCACHE=/workspace/.cache/go-mod \
		-e GOCACHE=/workspace/.cache/go-build \
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

openapi_drift_check() {
	if ! git -C "${ROOT_DIR}" diff --quiet -- internal/api; then
		echo "tracked openapi codegen drift detected in internal/api"
		git -C "${ROOT_DIR}" diff -- internal/api
		exit 1
	fi

	untracked="$(git -C "${ROOT_DIR}" ls-files --others --exclude-standard -- internal/api)"
	if [[ -n "${untracked}" ]]; then
		echo "untracked openapi artifacts detected in internal/api"
		echo "${untracked}"
		echo "run 'make openapi-generate' and commit updated generated files"
		exit 1
	fi
}

mocks_drift_check() {
	if ! git -C "${ROOT_DIR}" diff --quiet -- ':(glob)**/*_mock_test.go'; then
		echo "tracked mockgen drift detected in *_mock_test.go files"
		git -C "${ROOT_DIR}" diff -- ':(glob)**/*_mock_test.go'
		exit 1
	fi

	untracked="$(git -C "${ROOT_DIR}" ls-files --others --exclude-standard -- ':(glob)**/*_mock_test.go')"
	if [[ -n "${untracked}" ]]; then
		echo "untracked mockgen artifacts detected"
		echo "${untracked}"
		echo "run 'make mocks-generate' and commit updated mock files"
		exit 1
	fi
}

stringer_drift_check() {
	if ! git -C "${ROOT_DIR}" diff --quiet -- ':(glob)**/*_string.go'; then
		echo "tracked stringer drift detected in *_string.go files"
		git -C "${ROOT_DIR}" diff -- ':(glob)**/*_string.go'
		exit 1
	fi

	untracked="$(git -C "${ROOT_DIR}" ls-files --others --exclude-standard -- ':(glob)**/*_string.go')"
	if [[ -n "${untracked}" ]]; then
		echo "untracked stringer artifacts detected"
		echo "${untracked}"
		echo "run 'make stringer-generate' and commit updated enum string files"
		exit 1
	fi
}

sqlc_drift_check() {
	if ! git -C "${ROOT_DIR}" diff --quiet -- internal/infra/postgres/sqlcgen; then
		echo "tracked sqlc drift detected in internal/infra/postgres/sqlcgen"
		git -C "${ROOT_DIR}" diff -- internal/infra/postgres/sqlcgen
		exit 1
	fi

	untracked="$(git -C "${ROOT_DIR}" ls-files --others --exclude-standard -- internal/infra/postgres/sqlcgen)"
	if [[ -n "${untracked}" ]]; then
		echo "untracked sqlc artifacts detected in internal/infra/postgres/sqlcgen"
		echo "${untracked}"
		echo "run 'make sqlc-generate' and commit updated sqlc generated files"
		exit 1
	fi

	expected_stems="$(
		for file in "${ROOT_DIR}"/internal/infra/postgres/queries/*.sql; do
			[[ -e "${file}" ]] || continue
			basename "${file}" .sql
		done | sort
	)"

	actual_stems="$(
		for file in "${ROOT_DIR}"/internal/infra/postgres/sqlcgen/*.sql.go; do
			[[ -e "${file}" ]] || continue
			basename "${file}" .sql.go
		done | sort
	)"

	if [[ "${expected_stems}" != "${actual_stems}" ]]; then
		echo "sqlc query/source mismatch detected"
		echo "expected generated query stems:"
		printf '%s\n' "${expected_stems}"
		echo "actual generated query stems:"
		printf '%s\n' "${actual_stems}"
		echo "remove stale generated files and run 'make sqlc-generate'"
		exit 1
	fi
}

run_coverage_check() {
	run_go "test -f coverage.out || (echo \"coverage.out not found; run 'test-cover' or 'test-report' first\"; exit 1); filtered_cov=\$(mktemp); grep -Ev '${COVERAGE_EXCLUDE_REGEX}' coverage.out > \"\${filtered_cov}\"; total=\$(go tool cover -func=\"\${filtered_cov}\" | awk '/^total:/ {gsub(/%/, \"\", \$3); print \$3}'); rm -f \"\${filtered_cov}\"; if [[ -z \"\${total}\" ]]; then echo \"failed to parse total coverage from coverage.out\"; exit 1; fi; awk -v total=\"\${total}\" -v minimum='${COVERAGE_MIN}' 'BEGIN { if ((total + 0) < (minimum + 0)) { printf \"coverage %.2f%% is below threshold %.2f%%\\n\", total, minimum; exit 1 } printf \"coverage %.2f%% meets threshold %.2f%%\\n\", total, minimum }'"
}

run_test_report() {
	run_go "mkdir -p '${TEST_REPORT_DIR}' && GOCOVERDIR= go tool gotestsum --format=standard-verbose --junitfile='${TEST_JUNIT_FILE}' --jsonfile='${TEST_JSON_FILE}' -- -race -covermode=atomic -coverprofile=coverage.out ./... && go tool cover -func=coverage.out"
	run_coverage_check
}

wait_for_postgres() {
	local container_name="$1"
	local attempts=60

	for _ in $(seq 1 "${attempts}"); do
		if docker exec "${container_name}" pg_isready -U app -d app >/dev/null 2>&1; then
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

	cleanup_migration() {
		local container_name="$1"
		local container_network="$2"

		docker rm -f "${container_name}" >/dev/null 2>&1 || true
		docker network rm "${container_network}" >/dev/null 2>&1 || true
	}
	trap "cleanup_migration '${postgres_container}' '${network_name}'" EXIT

	docker network create "${network_name}" >/dev/null
	docker run \
		-d \
		--name "${postgres_container}" \
		--network "${network_name}" \
		-e POSTGRES_DB=app \
		-e POSTGRES_USER=app \
		-e POSTGRES_PASSWORD=app \
		"${POSTGRES_IMAGE}" >/dev/null

	wait_for_postgres "${postgres_container}"

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
}

run_container_security_scan() {
	ensure_docker

	if [[ ! -S /var/run/docker.sock ]]; then
		echo "docker socket is unavailable; container scan cannot run in docker mode"
		exit 1
	fi

	docker build -f "${ROOT_DIR}/build/docker/Dockerfile" -t service:ci "${ROOT_DIR}"
	docker run \
		--rm \
		-v /var/run/docker.sock:/var/run/docker.sock \
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
	docker pull "${GOLANGCI_LINT_IMAGE}"
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
	git -C "${ROOT_DIR}" diff --exit-code -- go.mod go.sum
	;;
fmt)
	run_go "go run golang.org/x/tools/cmd/goimports@${GOIMPORTS_VERSION} -w \$(find . -type f -name '*.go' -not -path './vendor/*' -not -path './.cache/*')"
	;;
fmt-check)
	run_go "unformatted=\$(go run golang.org/x/tools/cmd/goimports@${GOIMPORTS_VERSION} -l \$(find . -type f -name '*.go' -not -path './vendor/*' -not -path './.cache/*')); if [[ -n \"\${unformatted}\" ]]; then echo 'goimports required for:'; echo \"\${unformatted}\"; exit 1; fi"
	;;
test)
	run_go "go test ./..."
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
test-integration)
	run_go_with_docker_socket "REQUIRE_DOCKER=${REQUIRE_DOCKER:-0} go test -tags=integration ./test/..."
	;;
stringer-generate)
	run_go "go generate -run \"stringer\" ./..."
	;;
stringer-drift-check)
	bash "${ROOT_DIR}/scripts/dev/docker-tooling.sh" stringer-generate
	stringer_drift_check
	;;
sqlc-generate)
	run_go "go tool sqlc generate -f internal/infra/postgres/sqlc.yaml"
	;;
sqlc-check)
	bash "${ROOT_DIR}/scripts/dev/docker-tooling.sh" sqlc-generate
	sqlc_drift_check
	;;
mocks-generate)
	run_go "go generate -run \"mockgen\" ./..."
	;;
mocks-drift-check)
	bash "${ROOT_DIR}/scripts/dev/docker-tooling.sh" mocks-generate
	mocks_drift_check
	;;
lint)
	run_lint "config verify && GOLANGCI_LINT_CACHE=/workspace/.cache/golangci-lint go tool golangci-lint run --timeout=3m"
	;;
openapi-generate)
	run_go "go generate ./internal/api"
	;;
openapi-drift-check)
	openapi_drift_check
	;;
openapi-runtime-contract-check)
	run_go "go test ./internal/infra/http -run '^TestOpenAPIRuntimeContract' -count=1"
	;;
openapi-lint)
	run_node "npx @redocly/cli@${REDOCLY_CLI_VERSION} lint --config .redocly.yaml api/openapi/service.yaml"
	;;
openapi-validate)
	run_go "go run github.com/getkin/kin-openapi/cmd/validate@${KIN_OPENAPI_VALIDATE_VERSION} -- api/openapi/service.yaml"
	;;
openapi-check)
	bash "${ROOT_DIR}/scripts/dev/docker-tooling.sh" openapi-generate
	bash "${ROOT_DIR}/scripts/dev/docker-tooling.sh" openapi-drift-check
	run_go "go test ./internal/api"
	bash "${ROOT_DIR}/scripts/dev/docker-tooling.sh" openapi-runtime-contract-check
	bash "${ROOT_DIR}/scripts/dev/docker-tooling.sh" openapi-lint
	bash "${ROOT_DIR}/scripts/dev/docker-tooling.sh" openapi-validate
	;;
go-security)
	run_go "go install golang.org/x/vuln/cmd/govulncheck@${GOVULNCHECK_VERSION} && \"\$(go env GOPATH)/bin/govulncheck\" ./... && go install github.com/securego/gosec/v2/cmd/gosec@${GOSEC_VERSION} && \"\$(go env GOPATH)/bin/gosec\" -exclude-generated -exclude-dir=.cache ./..."
	;;
secrets-scan)
	run_go "go run github.com/zricethezav/gitleaks/v8@${GITLEAKS_VERSION} git --no-banner --redact --exit-code 1 ."
	;;
guardrails-check)
	bash "${ROOT_DIR}/scripts/ci/required-guardrails-check.sh"
	;;
skills-check)
	bash "${ROOT_DIR}/scripts/dev/sync-skills.sh" --check
	;;
agents-check)
	bash "${ROOT_DIR}/scripts/dev/sync-agents.sh" --check
	;;
docs-drift-check)
	base_ref="${1:-}"
	head_ref="${2:-}"
	if [[ -z "${base_ref}" || -z "${head_ref}" ]]; then
		echo "docs-drift-check requires <base-ref> <head-ref>"
		exit 1
	fi
	bash "${ROOT_DIR}/scripts/ci/docs-drift-check.sh" "${base_ref}" "${head_ref}"
	;;
migration-validate)
	run_migration_validate
	;;
container-security)
	run_container_security_scan
	;;
ci)
	bash "${ROOT_DIR}/scripts/dev/docker-tooling.sh" mod-check
	bash "${ROOT_DIR}/scripts/dev/docker-tooling.sh" guardrails-check
	bash "${ROOT_DIR}/scripts/dev/docker-tooling.sh" agents-check
	bash "${ROOT_DIR}/scripts/dev/docker-tooling.sh" skills-check
	bash "${ROOT_DIR}/scripts/dev/docker-tooling.sh" fmt-check
	bash "${ROOT_DIR}/scripts/dev/docker-tooling.sh" lint
	bash "${ROOT_DIR}/scripts/dev/docker-tooling.sh" test
	bash "${ROOT_DIR}/scripts/dev/docker-tooling.sh" vet
	bash "${ROOT_DIR}/scripts/dev/docker-tooling.sh" test-race
	bash "${ROOT_DIR}/scripts/dev/docker-tooling.sh" test-report
	bash "${ROOT_DIR}/scripts/dev/docker-tooling.sh" mocks-drift-check
	bash "${ROOT_DIR}/scripts/dev/docker-tooling.sh" stringer-drift-check
	bash "${ROOT_DIR}/scripts/dev/docker-tooling.sh" sqlc-check
	REQUIRE_DOCKER=1 bash "${ROOT_DIR}/scripts/dev/docker-tooling.sh" test-integration
	bash "${ROOT_DIR}/scripts/dev/docker-tooling.sh" openapi-check
	bash "${ROOT_DIR}/scripts/dev/docker-tooling.sh" go-security
	bash "${ROOT_DIR}/scripts/dev/docker-tooling.sh" secrets-scan
	bash "${ROOT_DIR}/scripts/dev/docker-tooling.sh" migration-validate
	bash "${ROOT_DIR}/scripts/dev/docker-tooling.sh" container-security

	if [[ -n "${BASE_REF:-}" && -n "${HEAD_REF:-}" ]]; then
		bash "${ROOT_DIR}/scripts/dev/docker-tooling.sh" docs-drift-check "${BASE_REF}" "${HEAD_REF}"
	else
		echo "BASE_REF/HEAD_REF are not set, skipping docs drift check in docker-ci"
	fi
	;;
*)
	echo "unknown command: ${cmd}"
	usage
	exit 1
	;;
esac
