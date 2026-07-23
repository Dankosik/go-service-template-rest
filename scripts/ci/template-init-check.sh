#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
TEMP_ROOT="$(mktemp -d -t template-init-check.XXXXXX)"
export GOCACHE="${GOCACHE:-${ROOT_DIR}/.cache/go-build}"
export GOLANGCI_LINT_CACHE="${GOLANGCI_LINT_CACHE:-${ROOT_DIR}/.cache/golangci-lint}"
mkdir -p "${GOCACHE}" "${GOLANGCI_LINT_CACHE}"
LINTER="$(cd "${ROOT_DIR}" && go tool -n golangci-lint)"
trap 'rm -rf "${TEMP_ROOT}"' EXIT

new_fixture() {
	local name="$1"
	local origin="${2:-}"
	local existing_env="${3:-false}"
	local root="${TEMP_ROOT}/${name}"

	mkdir -p \
		"${root}/.github" \
		"${root}/api/openapi" \
		"${root}/env" \
		"${root}/internal/example" \
		"${root}/internal/infra/example"

	{
		echo "module github.com/example/go-service-template-rest"
		echo
		echo "go 1.26"
	} >"${root}/go.mod"
	cp "${ROOT_DIR}/.golangci.yml" "${root}/.golangci.yml"
	cp "${ROOT_DIR}/.github/CODEOWNERS" "${root}/.github/CODEOWNERS"
	cp "${ROOT_DIR}/api/openapi/service.yaml" "${root}/api/openapi/service.yaml"
	cp "${ROOT_DIR}/env/.env.example" "${root}/env/.env.example"

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
! grep -R -Fq "github.com/example/go-service-template-rest" \
	"${derived}/internal" "${derived}/.golangci.yml"
! grep -v '^[[:space:]]*#' "${derived}/.github/CODEOWNERS" | grep -Fq "@Dankosik"
grep -v '^[[:space:]]*#' "${derived}/.github/CODEOWNERS" | grep -Fq "@acme/platform"
grep -Fqx '  title: "orders"' "${derived}/api/openapi/service.yaml"
[[ "${env_before}" == "$(shasum -a 256 "${derived}/.env")" ]]

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

echo "template initialization contract passed"
