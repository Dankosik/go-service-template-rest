#!/usr/bin/env bash
# Focused validation receipts shared by unit-check, package tests, and verify.
#
#   proof-receipt.sh check PROFILE ITEM
#   proof-receipt.sh store PROFILE ITEM
#   proof-receipt.sh check-all PROFILE "ITEM ..."
#   proof-receipt.sh store-all PROFILE "ITEM ..."
#   proof-receipt.sh run-packages PROFILE "PKG ..." -- command...
set -euo pipefail

ROOT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)
cd "${ROOT_DIR}"

usage() {
	echo "usage: $0 check|store|check-all|store-all|run-packages|--self-test ..." >&2
	exit 2
}

receipt_root() {
	local common_dir
	if [[ -n ${PROOF_RECEIPT_DIR:-} ]]; then
		printf '%s\n' "${PROOF_RECEIPT_DIR}"
		return
	fi
	common_dir=$(git rev-parse --git-common-dir)
	[[ ${common_dir} == /* ]] || common_dir=${ROOT_DIR}/${common_dir}
	printf '%s\n' "${common_dir}/validation/proofs"
}

package_dir() {
	local pkg=$1
	if [[ ${pkg} == . ]]; then
		printf '%s\n' "${ROOT_DIR}"
	else
		printf '%s/%s\n' "${ROOT_DIR}" "${pkg#./}"
	fi
}

fingerprint() {
	local profile=$1 item=$2 dir file
	{
		printf 'profile=%s\n' "${profile}"
		printf 'item=%s\n' "${item}"
		printf 'head=%s\n' "$(git rev-parse HEAD 2>/dev/null || echo unavailable)"
		printf 'go=%s\n' "$(go version 2>/dev/null || echo unavailable)"
		printf 'GOFLAGS=%s\n' "${GOFLAGS:-}"
		printf 'GOTOOLCHAIN=%s\n' "${GOTOOLCHAIN:-}"
		case "${profile}" in
		fmt)
			if [[ -f ${item} ]]; then
				shasum -a 256 "${item}"
			else
				printf 'missing %s\n' "${item}"
			fi
			;;
		test | lint)
			dir=$(package_dir "${item}")
			if [[ -d ${dir} ]]; then
				while IFS= read -r file; do
					shasum -a 256 "${file}"
				done < <(find "${dir}" -maxdepth 1 -type f -name '*.go' | LC_ALL=C sort)
			else
				printf 'missing-package %s\n' "${item}"
			fi
			if [[ ${profile} == lint && -f .golangci.yml ]]; then
				shasum -a 256 .golangci.yml
			fi
			;;
		*)
			printf 'unknown-profile %s\n' "${profile}"
			;;
		esac
	} | shasum -a 256 | awk '{print $1}'
}

receipt_path() {
	local profile=$1 item=$2 key
	key=$(fingerprint "${profile}" "${item}")
	printf '%s/%s-%s.receipt\n' "$(receipt_root)" "${profile}" "${key}"
}

check_one() {
	local profile=$1 item=$2 path
	if [[ ${VERIFY_FORCE:-} == 1 ]]; then
		return 1
	fi
	path=$(receipt_path "${profile}" "${item}")
	[[ -f ${path} ]] || return 1
	grep -qx 'result=pass' "${path}"
}

store_one() {
	local profile=$1 item=$2 path tmp
	path=$(receipt_path "${profile}" "${item}")
	mkdir -p "$(dirname "${path}")"
	tmp=${path}.tmp.$$
	{
		printf 'result=pass\n'
		printf 'profile=%s\n' "${profile}"
		printf 'item=%s\n' "${item}"
		printf 'head=%s\n' "$(git rev-parse HEAD 2>/dev/null || echo unavailable)"
	} >"${tmp}"
	mv "${tmp}" "${path}"
}

check_all() {
	local profile=$1 item
	shift
	for item in "$@"; do
		[[ -n ${item} ]] || continue
		check_one "${profile}" "${item}" || return 1
	done
	return 0
}

store_all() {
	local profile=$1 item
	shift
	for item in "$@"; do
		[[ -n ${item} ]] || continue
		store_one "${profile}" "${item}"
	done
}

run_packages() {
	local profile=$1
	local packages=$2
	shift 2
	local pkg remaining=()
	if [[ ${1:-} != -- ]]; then
		usage
	fi
	shift
	if [[ -z ${packages} ]]; then
		"$@"
		return
	fi
	if [[ ${packages} == ./... ]]; then
		"$@" ./...
		return
	fi
	# shellcheck disable=SC2086
	for pkg in ${packages}; do
		if check_one "${profile}" "${pkg}"; then
			echo "reusing focused ${profile} receipt: ${pkg}"
			continue
		fi
		remaining+=("${pkg}")
	done
	if ((${#remaining[@]} == 0)); then
		return 0
	fi
	"$@" "${remaining[@]}"
	store_all "${profile}" "${remaining[@]}"
}

self_test() {
	local tmp item
	tmp=$(mktemp -d)
	trap 'rm -rf -- "${tmp}"' RETURN
	PROOF_RECEIPT_DIR="${tmp}"
	item=internal/failure/failure.go
	[[ -f ${item} ]]
	store_one fmt "${item}"
	check_one fmt "${item}"
	if PROOF_RECEIPT_DIR="${tmp}" VERIFY_FORCE=1 check_one fmt "${item}"; then
		echo "VERIFY_FORCE did not bypass a receipt" >&2
		return 1
	fi
	store_one test ./internal/failure
	check_one test ./internal/failure
	check_all test ./internal/failure
}

action=${1:-}
case "${action}" in
--self-test)
	self_test
	;;
check)
	[[ $# -eq 3 ]] || usage
	check_one "$2" "$3"
	;;
store)
	[[ $# -eq 3 ]] || usage
	store_one "$2" "$3"
	;;
check-all)
	[[ $# -ge 3 ]] || usage
	profile=$2
	shift 2
	check_all "${profile}" "$@"
	;;
store-all)
	[[ $# -ge 3 ]] || usage
	profile=$2
	shift 2
	store_all "${profile}" "$@"
	;;
run-packages)
	[[ $# -ge 4 ]] || usage
	run_packages "$2" "$3" "${@:4}"
	;;
*)
	usage
	;;
esac
