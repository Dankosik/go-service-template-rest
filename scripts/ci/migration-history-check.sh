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

BASELINE_FILE=".migration-baseline"

changes="$(
	git diff --name-status --find-renames --find-copies "${diff_args[@]}" -- migrations/ |
		awk '$1 !~ /^A$/ { print }'
)"
if [[ -z "${changes}" ]]; then
	echo "migration history is append-only (${scope})"
	exit 0
fi

# Rewriting a migration a database already ran leaves that database
# unmigratable, so this check treats every committed migration as applied. That
# is exact for a service and conservative for a template, whose packs are
# authored here and only ever copied into a new service at initialization.
#
# A pre-publication reset lifts it for one change by updating the baseline file
# in the same diff. The exception is deliberate, reviewable beside the migration
# it explains, and records the evidence that nothing had applied the files it
# rewrites. cd.yml holds the same line against the published runtime image and
# has no such escape, so a wrong declaration still cannot reach a registry.
if [[ -z "$(git diff --name-only "${diff_args[@]}" -- "${BASELINE_FILE}")" ]]; then
	echo "migration history: published migration files are append-only (${scope})" >&2
	printf '%s\n' "${changes}" >&2
	echo "migration history: declare a pre-publication reset in ${BASELINE_FILE} to rewrite them" >&2
	exit 1
fi

echo "migration history: pre-publication reset declared in ${BASELINE_FILE} (${scope})"
printf '%s\n' "${changes}"
