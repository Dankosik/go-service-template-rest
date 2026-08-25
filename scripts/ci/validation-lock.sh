#!/usr/bin/env bash
set -euo pipefail

if [[ ${1:-} == --self-test ]]; then
	tmp=$(mktemp -d)
	trap 'rm -rf -- "${tmp}"' EXIT
	# An outer verify/unit-check may export VALIDATION_LOCK_HELD=1. The self-test
	# must still take its own temp lock, or it waits forever for owner.
	unset VALIDATION_LOCK_HELD
	VALIDATION_LOCK_DIR="${tmp}/lock" bash "$0" -- sh -c 'printf 1 >>"$1"; sleep 1; printf 2 >>"$1"' -- "${tmp}/order" &
	first_pid=$!
	deadline=$((SECONDS + 10))
	while [[ ! -f ${tmp}/lock/owner ]]; do
		if ((SECONDS >= deadline)); then
			echo "validation lock self-test never acquired ${tmp}/lock" >&2
			exit 1
		fi
		sleep 0.05
	done
	VALIDATION_LOCK_DIR="${tmp}/lock" bash "$0" -- sh -c 'printf 3 >>"$1"' -- "${tmp}/order"
	wait "${first_pid}"
	[[ $(<"${tmp}/order") == 123 ]] || {
		echo "validation lock self-test observed overlapping commands" >&2
		exit 1
	}
	exit
fi

if [[ ${1:-} != -- || $# -lt 2 ]]; then
	echo "usage: $0 -- command [args...]" >&2
	exit 2
fi
shift

if [[ ${VALIDATION_LOCK_HELD:-} == 1 ]]; then
	"$@"
	exit
fi

root=$(git rev-parse --show-toplevel)
common_dir=$(git rev-parse --git-common-dir)
[[ ${common_dir} == /* ]] || common_dir=${root}/${common_dir}
lock_dir=${VALIDATION_LOCK_DIR:-${common_dir}/codex/validation.lock}
owner_file=${lock_dir}/owner
timeout=${VALIDATION_LOCK_TIMEOUT_SECONDS:-900}
started=$(date +%s)
mkdir -p "$(dirname "${lock_dir}")"

while ! mkdir "${lock_dir}" 2>/dev/null; do
	owner_pid=$(awk -F= '$1 == "pid" { print $2; exit }' "${owner_file}" 2>/dev/null || true)
	if [[ -n ${owner_pid} ]] && ! kill -0 "${owner_pid}" 2>/dev/null; then
		rm -f "${owner_file}"
		rmdir "${lock_dir}" 2>/dev/null || true
		continue
	fi
	if (( $(date +%s) - started >= timeout )); then
		echo "validation lock timed out after ${timeout}s: ${lock_dir}" >&2
		[[ -f ${owner_file} ]] && cat "${owner_file}" >&2
		exit 75
	fi
	sleep 1
done

cleanup() {
	rm -f "${owner_file}"
	rmdir "${lock_dir}" 2>/dev/null || true
}
trap cleanup EXIT INT TERM

{
	printf 'pid=%s\n' "$$"
	printf 'candidate=%s\n' "$(git rev-parse HEAD)"
	printf 'command='
	printf '%q ' "$@"
	printf '\n'
} >"${owner_file}"

export VALIDATION_LOCK_HELD=1
"$@"
