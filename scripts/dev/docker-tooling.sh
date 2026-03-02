#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"

GO_IMAGE="${GO_IMAGE:-golang:1.25.7-bookworm}"
NODE_IMAGE="${NODE_IMAGE:-node:20.19.0-bookworm}"
GOLANGCI_LINT_IMAGE="${GOLANGCI_LINT_IMAGE:-golangci/golangci-lint:v2.10.1}"
POSTGRES_IMAGE="${POSTGRES_IMAGE:-postgres:17}"
MIGRATE_IMAGE="${MIGRATE_IMAGE:-migrate/migrate:v4.19.0}"
TRIVY_IMAGE="${TRIVY_IMAGE:-aquasec/trivy:0.65.0}"
REDOCLY_CLI_VERSION="${REDOCLY_CLI_VERSION:-2.20.0}"
KIN_OPENAPI_VALIDATE_VERSION="${KIN_OPENAPI_VALIDATE_VERSION:-v0.133.0}"
GOVULNCHECK_VERSION="${GOVULNCHECK_VERSION:-v1.1.4}"
GOSEC_VERSION="${GOSEC_VERSION:-v2.24.6}"

host_uid="$(id -u 2>/dev/null || echo 0)"
host_gid="$(id -g 2>/dev/null || echo 0)"

usage() {
	echo "usage: $0 <command> [args]"
	echo "commands:"
	echo "  doctor"
	echo "  pull-images"
	echo "  init-module <module-path>   (uses CODEOWNER env optionally)"
	echo "  mod-check"
	echo "  fmt"
	echo "  fmt-check"
	echo "  test"
	echo "  test-race"
	echo "  test-cover"
	echo "  test-integration"
	echo "  lint"
	echo "  openapi-generate"
	echo "  openapi-drift-check"
	echo "  openapi-runtime-contract-check"
	echo "  openapi-lint"
	echo "  openapi-validate"
	echo "  openapi-check"
	echo "  go-security"
	echo "  guardrails-check"
	echo "  skills-check"
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
		bash -lc "$*"
}

run_go_with_docker_socket() {
	ensure_docker
	mkdir -p "${ROOT_DIR}/.cache/go-mod" "${ROOT_DIR}/.cache/go-build"

	socket_args=()
	if [[ -S /var/run/docker.sock ]]; then
		socket_args+=(-v /var/run/docker.sock:/var/run/docker.sock)
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
		bash -lc "$*"
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
	ensure_docker
	mkdir -p "${ROOT_DIR}/.cache/golangci-lint"
	docker run \
		--rm \
		-u "${host_uid}:${host_gid}" \
		-v "${ROOT_DIR}:/workspace" \
		-w /workspace \
		-v "${ROOT_DIR}/.cache/golangci-lint:/tmp/golangci-lint-cache" \
		-e GOLANGCI_LINT_CACHE=/tmp/golangci-lint-cache \
		--entrypoint /usr/bin/golangci-lint \
		"${GOLANGCI_LINT_IMAGE}" \
		"$@"
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
		docker rm -f "${postgres_container}" >/dev/null 2>&1 || true
		docker network rm "${network_name}" >/dev/null 2>&1 || true
	}
	trap cleanup_migration EXIT

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
	"${ROOT_DIR}/scripts/dev/doctor.sh" --mode docker
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
	if [[ -z "${module_path}" ]]; then
		echo "init-module requires <module-path>"
		exit 1
	fi
	shift || true
	if [[ $# -ne 0 ]]; then
		echo "init-module accepts only one argument: <module-path>"
		exit 1
	fi
	ensure_docker
	mkdir -p "${ROOT_DIR}/.cache/go-mod" "${ROOT_DIR}/.cache/go-build"
	docker run \
		--rm \
		-u "${host_uid}:${host_gid}" \
		-v "${ROOT_DIR}:/workspace" \
		-w /workspace \
		-e CODEOWNER="${CODEOWNER:-}" \
		-e GOMODCACHE=/workspace/.cache/go-mod \
		-e GOCACHE=/workspace/.cache/go-build \
		"${GO_IMAGE}" \
		bash -lc "./scripts/init-module.sh \"${module_path}\""
	;;
mod-check)
	run_go "GOFLAGS= go mod tidy -diff && go mod verify"
	git -C "${ROOT_DIR}" diff --exit-code -- go.mod go.sum
	;;
fmt)
	run_go "gofmt -w \$(find . -type f -name '*.go' -not -path './vendor/*')"
	;;
fmt-check)
	"${ROOT_DIR}/scripts/dev/docker-tooling.sh" fmt
	git -C "${ROOT_DIR}" diff --exit-code
	;;
test)
	run_go "go test ./..."
	;;
test-race)
	run_go "go test -race ./..."
	;;
test-cover)
	run_go "go test -covermode=atomic -coverprofile=coverage.out ./... && go tool cover -func=coverage.out"
	;;
test-integration)
	run_go_with_docker_socket "REQUIRE_DOCKER=${REQUIRE_DOCKER:-0} go test -tags=integration ./test/..."
	;;
lint)
	run_lint run --timeout=3m
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
	"${ROOT_DIR}/scripts/dev/docker-tooling.sh" openapi-generate
	"${ROOT_DIR}/scripts/dev/docker-tooling.sh" openapi-drift-check
	run_go "go test ./internal/api"
	"${ROOT_DIR}/scripts/dev/docker-tooling.sh" openapi-runtime-contract-check
	"${ROOT_DIR}/scripts/dev/docker-tooling.sh" openapi-lint
	"${ROOT_DIR}/scripts/dev/docker-tooling.sh" openapi-validate
	;;
go-security)
	run_go "go install golang.org/x/vuln/cmd/govulncheck@${GOVULNCHECK_VERSION} && \"\$(go env GOPATH)/bin/govulncheck\" ./... && go install github.com/securego/gosec/v2/cmd/gosec@${GOSEC_VERSION} && \"\$(go env GOPATH)/bin/gosec\" -exclude-generated ./..."
	;;
guardrails-check)
	"${ROOT_DIR}/scripts/ci/required-guardrails-check.sh"
	;;
skills-check)
	"${ROOT_DIR}/scripts/dev/sync-skills.sh" --check
	;;
docs-drift-check)
	base_ref="${1:-}"
	head_ref="${2:-}"
	if [[ -z "${base_ref}" || -z "${head_ref}" ]]; then
		echo "docs-drift-check requires <base-ref> <head-ref>"
		exit 1
	fi
	"${ROOT_DIR}/scripts/ci/docs-drift-check.sh" "${base_ref}" "${head_ref}"
	;;
migration-validate)
	run_migration_validate
	;;
container-security)
	run_container_security_scan
	;;
ci)
	"${ROOT_DIR}/scripts/dev/docker-tooling.sh" mod-check
	"${ROOT_DIR}/scripts/dev/docker-tooling.sh" guardrails-check
	"${ROOT_DIR}/scripts/dev/docker-tooling.sh" skills-check
	"${ROOT_DIR}/scripts/dev/docker-tooling.sh" fmt-check
	"${ROOT_DIR}/scripts/dev/docker-tooling.sh" lint
	"${ROOT_DIR}/scripts/dev/docker-tooling.sh" test
	"${ROOT_DIR}/scripts/dev/docker-tooling.sh" test-race
	"${ROOT_DIR}/scripts/dev/docker-tooling.sh" test-cover
	REQUIRE_DOCKER=1 "${ROOT_DIR}/scripts/dev/docker-tooling.sh" test-integration
	"${ROOT_DIR}/scripts/dev/docker-tooling.sh" openapi-check
	"${ROOT_DIR}/scripts/dev/docker-tooling.sh" go-security
	"${ROOT_DIR}/scripts/dev/docker-tooling.sh" migration-validate
	"${ROOT_DIR}/scripts/dev/docker-tooling.sh" container-security

	if [[ -n "${BASE_REF:-}" && -n "${HEAD_REF:-}" ]]; then
		"${ROOT_DIR}/scripts/dev/docker-tooling.sh" docs-drift-check "${BASE_REF}" "${HEAD_REF}"
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
