#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="${MIGRATION_REPO_ROOT:-$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)}"
cd "${ROOT_DIR}"

mode="${MIGRATION_HISTORY_MODE:-worktree}"
head_ref="${HEAD_REF:-HEAD}"
base_ref="${BASE_REF:-}"

case "${mode}" in
worktree)
	base_ref="HEAD"
	scope="worktree"
	diff_args=("${base_ref}")
	;;
merge-base)
	if [[ -z "${base_ref}" ]] || ! git cat-file -e "${base_ref}^{commit}" 2>/dev/null; then
		echo "migration history: BASE_REF must name a readable commit in merge-base mode" >&2
		exit 1
	fi
	base_ref="$(git merge-base "${base_ref}" "${head_ref}")"
	scope="merge-base:${base_ref}"
	diff_args=("${base_ref}")
	;;
exact-base)
	if [[ -z "${base_ref}" ]]; then
		echo "migration history: BASE_REF is required in exact-base mode" >&2
		exit 1
	fi
	if [[ "${base_ref}" == "0000000000000000000000000000000000000000" ]]; then
		base_ref="$(git hash-object -t tree /dev/null)"
		scope="exact-base:empty-tree"
	elif git cat-file -e "${base_ref}^{commit}" 2>/dev/null; then
		scope="exact-base:${base_ref}"
	else
		echo "migration history: exact BASE_REF ${base_ref} is unavailable" >&2
		exit 1
	fi
	if ! git cat-file -e "${head_ref}^{commit}" 2>/dev/null; then
		echo "migration history: HEAD_REF ${head_ref} is unavailable" >&2
		exit 1
	fi
	diff_args=("${base_ref}" "${head_ref}")
	;;
*)
	echo "migration history: unsupported MIGRATION_HISTORY_MODE ${mode}" >&2
	exit 2
	;;
esac

changes="$(
	git diff --name-status --find-renames --find-copies "${diff_args[@]}" -- migrations/ |
		awk '$1 !~ /^A$/ { print }'
)"
if [[ -n "${changes}" ]]; then
	echo "migration history: published migration files are append-only (${scope})" >&2
	printf '%s\n' "${changes}" >&2
	exit 1
fi

echo "migration history is append-only (${scope})"
