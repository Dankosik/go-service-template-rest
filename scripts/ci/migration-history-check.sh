#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="${MIGRATION_REPO_ROOT:-$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)}"
cd "${ROOT_DIR}"

mode="${MIGRATION_HISTORY_MODE:-worktree}"
head_ref="${HEAD_REF:-HEAD}"
base_ref="${BASE_REF:-}"

if [[ "${mode}" == "self-test" ]]; then
	fixture="$(mktemp -d -t migration-history.XXXXXX)"
	trap 'rm -rf -- "${fixture}"' EXIT
	git -C "${fixture}" init -q
	git -C "${fixture}" config user.name migration-history
	git -C "${fixture}" config user.email migration-history@example.invalid
	mkdir -p "${fixture}/migrations"
	printf '%s\n' '-- +goose Up' 'SELECT 1;' '-- +goose Down' 'SELECT 1;' >"${fixture}/migrations/000001_create.sql"
	git -C "${fixture}" add .
	git -C "${fixture}" commit -qm baseline
	printf '%s\n' '-- rewritten' >>"${fixture}/migrations/000001_create.sql"
	if MIGRATION_REPO_ROOT="${fixture}" MIGRATION_HISTORY_MODE=worktree bash "${BASH_SOURCE[0]}" >/dev/null 2>&1; then
		echo "migration history self-test: rewrite passed" >&2
		exit 1
	fi
	git -C "${fixture}" restore migrations/000001_create.sql
	printf '%s\n' '-- +goose Up' 'SELECT 2;' '-- +goose Down' 'SELECT 2;' >"${fixture}/migrations/000002_add.sql"
	MIGRATION_REPO_ROOT="${fixture}" MIGRATION_HISTORY_MODE=worktree bash "${BASH_SOURCE[0]}" >/dev/null
	echo "migration history self-test passed"
	exit 0
fi

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

# Append-only belongs to the service, not to the template. A service applies its
# migrations to a database that keeps running, so rewriting one it already ran
# leaves that database unmigratable. The template only authors them: its
# migrations reach nothing but ephemeral test containers and the empty database
# of a service generated from it, so the target schema is edited in place rather
# than accumulated as the history of how it was written. scripts/profiles is
# what initialization removes, so its absence marks a generated service.
#
# Goose validation still holds canonical source structure on both sides, and
# cd.yml keeps published migrations immutable against the runtime image.
if [[ -d "${ROOT_DIR}/scripts/profiles" ]]; then
	echo "migration history: template source authors migrations in place (${scope})"
	exit 0
fi

changes="$(
	git diff --name-status --find-renames --find-copies "${diff_args[@]}" -- migrations/ |
		awk '$1 !~ /^A$/ { print }'
)"
if [[ -n "${changes}" ]]; then
	echo "migration history: applied migration files are append-only (${scope})" >&2
	printf '%s\n' "${changes}" >&2
	exit 1
fi

echo "migration history is append-only (${scope})"
