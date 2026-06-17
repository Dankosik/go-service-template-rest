#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"

usage() {
	echo "usage: $0 openapi|sqlc"
}

check_generated_dir() {
	local label="$1"
	local path="$2"

	if ! git -C "${ROOT_DIR}" diff --quiet -- "${path}"; then
		echo "tracked ${label} drift detected in ${path}"
		git -C "${ROOT_DIR}" diff -- "${path}"
		exit 1
	fi

	local untracked
	untracked="$(git -C "${ROOT_DIR}" ls-files --others --exclude-standard -- "${path}")"
	if [[ -n "${untracked}" ]]; then
		echo "untracked ${label} artifacts detected in ${path}"
		echo "${untracked}"
		echo "run the matching generate target and commit updated generated files"
		exit 1
	fi
}

check_sqlc_stems() {
	local expected actual

	expected="$(
		for file in "${ROOT_DIR}"/internal/infra/postgres/queries/*.sql; do
			[[ -e "${file}" ]] || continue
			basename "${file}" .sql
		done | sort
	)"
	actual="$(
		for file in "${ROOT_DIR}"/internal/infra/postgres/sqlcgen/*.sql.go; do
			[[ -e "${file}" ]] || continue
			basename "${file}" .sql.go
		done | sort
	)"

	if [[ "${expected}" != "${actual}" ]]; then
		echo "sqlc query/source mismatch detected"
		echo "expected generated query stems:"
		printf '%s\n' "${expected}"
		echo "actual generated query stems:"
		printf '%s\n' "${actual}"
		echo "remove stale generated files and run 'make sqlc-generate'"
		exit 1
	fi
}

has_sqlc_queries() {
	find "${ROOT_DIR}/internal/infra/postgres/queries" -type f -name '*.sql' -print -quit | grep -q .
}

check_empty_sqlc_outputs() {
	local generated
	generated="$(find "${ROOT_DIR}/internal/infra/postgres/sqlcgen" -type f -name '*.go' -print 2>/dev/null || true)"
	if [[ -n "${generated}" ]]; then
		echo "stale sqlc artifacts detected without query sources"
		echo "${generated}"
		echo "remove generated sqlc output or add feature-owned queries"
		exit 1
	fi
}

case "${1:-}" in
openapi)
	check_generated_dir "openapi codegen" "internal/api/openapi.gen.go"
	;;
sqlc)
	if ! has_sqlc_queries; then
		check_empty_sqlc_outputs
		exit 0
	fi
	check_generated_dir "sqlc" "internal/infra/postgres/sqlcgen"
	check_sqlc_stems
	;;
*)
	usage
	exit 2
	;;
esac
