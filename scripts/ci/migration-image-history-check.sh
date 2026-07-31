#!/usr/bin/env bash
set -euo pipefail

usage() {
	echo "usage: $0 compare <prior-dir> <candidate-dir> | package-mode" >&2
	exit 2
}

compare_corpus() {
	local prior="$1"
	local candidate="$2"
	[[ -d "${prior}" ]] || {
		echo "migration image history: prior migration directory is missing" >&2
		exit 1
	}
	[[ -d "${candidate}" ]] || {
		echo "migration image history: candidate migration directory is missing" >&2
		exit 1
	}

	while IFS= read -r prior_file; do
		relative="${prior_file#"${prior}/"}"
		candidate_file="${candidate}/${relative}"
		if [[ ! -f "${candidate_file}" ]]; then
			echo "migration image history: candidate removed ${relative}" >&2
			exit 1
		fi
		if ! cmp -s "${prior_file}" "${candidate_file}"; then
			echo "migration image history: candidate changed ${relative}" >&2
			exit 1
		fi
	done < <(find "${prior}" -type f -print | LC_ALL=C sort)

	echo "migration image history is append-only"
}

package_mode() {
	: "${GITHUB_REPOSITORY_OWNER:?GITHUB_REPOSITORY_OWNER is required}"
	: "${GITHUB_REPOSITORY_NAME:?GITHUB_REPOSITORY_NAME is required}"
	: "${GITHUB_REPOSITORY_OWNER_TYPE:?GITHUB_REPOSITORY_OWNER_TYPE is required}"
	: "${CANDIDATE_SHA:?CANDIDATE_SHA is required}"

	local packages_json endpoint package_name
	package_name="$(printf '%s' "${GITHUB_REPOSITORY_NAME}" | tr '[:upper:]' '[:lower:]')"
	if [[ -n "${MIGRATION_PACKAGE_LIST_JSON:-}" ]]; then
		packages_json="${MIGRATION_PACKAGE_LIST_JSON}"
	else
		: "${GH_TOKEN:?GH_TOKEN is required}"
		case "${GITHUB_REPOSITORY_OWNER_TYPE}" in
		Organization) endpoint="/orgs/${GITHUB_REPOSITORY_OWNER}/packages?package_type=container&per_page=100" ;;
		User) endpoint="/users/${GITHUB_REPOSITORY_OWNER}/packages?package_type=container&per_page=100" ;;
		*)
			echo "migration image history: unsupported repository owner type" >&2
			exit 1
			;;
		esac
		packages_json="$(gh api --paginate --slurp "${endpoint}")"
	fi

	if ! jq -e 'type == "array" and all(.[]; type == "array")' \
		<<<"${packages_json}" >/dev/null; then
		echo "migration image history: package listing is incomplete or malformed" >&2
		exit 1
	fi
	if jq -e --arg name "${package_name}" \
		'any(.[][]; (.name | ascii_downcase) == $name)' \
		<<<"${packages_json}" >/dev/null; then
		if [[ -n "${MIGRATION_HISTORY_BOOTSTRAP_SHA:-}" ]]; then
			echo "migration image history: clear MIGRATION_HISTORY_BOOTSTRAP_SHA after bootstrap" >&2
			exit 1
		fi
		echo "history"
		return
	fi

	if [[ "${MIGRATION_HISTORY_BOOTSTRAP_SHA:-}" != "${CANDIDATE_SHA}" ]]; then
		echo "migration image history: empty package requires bootstrap SHA equal to candidate" >&2
		exit 1
	fi
	echo "bootstrap"
}

case "${1:-}" in
compare)
	[[ $# -eq 3 ]] || usage
	compare_corpus "$2" "$3"
	;;
package-mode)
	[[ $# -eq 1 ]] || usage
	package_mode
	;;
*)
	usage
	;;
esac
