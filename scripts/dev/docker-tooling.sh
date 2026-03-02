#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"

GO_IMAGE="${GO_IMAGE:-golang:1.25.7-bookworm}"
NODE_IMAGE="${NODE_IMAGE:-node:20.19.0-bookworm}"
GOLANGCI_LINT_IMAGE="${GOLANGCI_LINT_IMAGE:-golangci/golangci-lint:v2.10.1}"
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
ci)
	"${ROOT_DIR}/scripts/dev/docker-tooling.sh" mod-check
	"${ROOT_DIR}/scripts/dev/docker-tooling.sh" fmt-check
	"${ROOT_DIR}/scripts/dev/docker-tooling.sh" lint
	"${ROOT_DIR}/scripts/dev/docker-tooling.sh" test
	"${ROOT_DIR}/scripts/dev/docker-tooling.sh" test-race
	"${ROOT_DIR}/scripts/dev/docker-tooling.sh" test-cover
	"${ROOT_DIR}/scripts/dev/docker-tooling.sh" openapi-check
	"${ROOT_DIR}/scripts/dev/docker-tooling.sh" go-security
	;;
*)
	echo "unknown command: ${cmd}"
	usage
	exit 1
	;;
esac
