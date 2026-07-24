#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"

usage() {
	echo "usage: $0 openapi|sqlc"
}

check_openapi() (
	local paths=(
		"internal/openapi/openapi.gen.go"
		"examples/reference-service/internal/openapi/openapi.gen.go"
	)
	local path
	local snapshot
	local drift=0

	snapshot="$(mktemp -d)"
	trap 'rm -rf "${snapshot}"' EXIT
	for path in "${paths[@]}"; do
		mkdir -p "${snapshot}/$(dirname "${path}")"
		cp "${ROOT_DIR}/${path}" "${snapshot}/${path}"
	done

	(cd "${ROOT_DIR}" && go generate ./internal/openapi ./examples/reference-service/internal/openapi)

	for path in "${paths[@]}"; do
		if cmp -s "${snapshot}/${path}" "${ROOT_DIR}/${path}"; then
			continue
		fi
		drift=1
		echo "openapi codegen drift detected: generation changed ${path}"
		diff -u \
			--label "${path} (before generation)" \
			--label "${path} (after generation)" \
			"${snapshot}/${path}" "${ROOT_DIR}/${path}" || true
	done
	if [[ "${drift}" -ne 0 ]]; then
		echo "generated output was updated; review and commit it"
		exit 1
	fi

	echo "openapi generated output is current"
)

check_sqlc() (
	local path="internal/infra/postgres/sqlcgen"
	local snapshot

	snapshot="$(mktemp -d)"
	trap 'rm -rf "${snapshot}"' EXIT
	cp -R "${ROOT_DIR}/${path}/." "${snapshot}/"

	(cd "${ROOT_DIR}" && bash ./scripts/run-go-tool.sh sqlc generate -f internal/infra/postgres/sqlc.yaml)

	if ! diff -qr "${snapshot}" "${ROOT_DIR}/${path}" >/dev/null; then
		echo "sqlc drift detected: generation changed ${path}"
		diff -ruN "${snapshot}" "${ROOT_DIR}/${path}" || true
		echo "generated output was updated; review and commit it"
		exit 1
	fi

	check_sqlc_stems
	echo "sqlc generated output is current"
)

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
	find "${ROOT_DIR}/internal/infra/postgres/queries" -type f -name '*.sql' -print -quit 2>/dev/null | grep -q .
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
	check_openapi
	;;
sqlc)
	if ! has_sqlc_queries; then
		check_empty_sqlc_outputs
		echo "sqlc generated output is current (no query sources)"
		exit 0
	fi
	check_sqlc
	;;
*)
	usage
	exit 2
	;;
esac
