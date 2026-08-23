#!/usr/bin/env bash
set -euo pipefail

usage() {
	echo "usage: $0 compare <prior-dir> <candidate-dir> | package-mode | self-test" >&2
	exit 2
}

compare_corpus() {
	local prior="$1"
	local candidate="$2"
	local prior_max=0 version
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
		if version="$(migration_version "${relative}")" && ((version > prior_max)); then
			prior_max="${version}"
		fi
	done < <(find "${prior}" -type f -print | LC_ALL=C sort)

	while IFS= read -r candidate_file; do
		relative="${candidate_file#"${candidate}/"}"
		[[ -e "${prior}/${relative}" ]] && continue
		if version="$(migration_version "${relative}")" && ((version <= prior_max)); then
			echo "migration image history: new migration ${relative} is not newer than prior version ${prior_max}" >&2
			exit 1
		fi
	done < <(find "${candidate}" -type f -print | LC_ALL=C sort)

	echo "migration image history is append-only"
}

migration_version() {
	local name="${1##*/}"
	[[ "${name}" =~ ^([0-9]+)_.+\.sql$ ]] || return 1
	printf '%d\n' "$((10#${BASH_REMATCH[1]}))"
}

package_mode() {
	: "${GITHUB_REPOSITORY_OWNER:?GITHUB_REPOSITORY_OWNER is required}"
	: "${GITHUB_REPOSITORY_NAME:?GITHUB_REPOSITORY_NAME is required}"
	: "${GITHUB_REPOSITORY_OWNER_TYPE:?GITHUB_REPOSITORY_OWNER_TYPE is required}"
	: "${CANDIDATE_SHA:?CANDIDATE_SHA is required}"

	local packages_json versions_json endpoint versions_endpoint package_name
	package_name="$(printf '%s' "${GITHUB_REPOSITORY_NAME}" | tr '[:upper:]' '[:lower:]')"
	case "${GITHUB_REPOSITORY_OWNER_TYPE}" in
	Organization)
		endpoint="/orgs/${GITHUB_REPOSITORY_OWNER}/packages?package_type=container&per_page=100"
		versions_endpoint="/orgs/${GITHUB_REPOSITORY_OWNER}/packages/container/${package_name}/versions?per_page=100"
		;;
	User)
		endpoint="/users/${GITHUB_REPOSITORY_OWNER}/packages?package_type=container&per_page=100"
		versions_endpoint="/users/${GITHUB_REPOSITORY_OWNER}/packages/container/${package_name}/versions?per_page=100"
		;;
	*)
		echo "migration image history: unsupported repository owner type" >&2
		exit 1
		;;
	esac
	if [[ -n "${MIGRATION_PACKAGE_LIST_JSON:-}" ]]; then
		packages_json="${MIGRATION_PACKAGE_LIST_JSON}"
	else
		: "${GH_TOKEN:?GH_TOKEN is required}"
		packages_json="$(gh api --paginate --slurp "${endpoint}")"
	fi

	if ! jq -e 'type == "array" and all(.[]; type == "array")' \
		<<<"${packages_json}" >/dev/null; then
		echo "migration image history: package listing is incomplete or malformed" >&2
		exit 1
	fi
	if jq -e --arg name "${package_name}" 'any(.[][]; (.name | ascii_downcase) == $name)' \
		<<<"${packages_json}" >/dev/null; then
		if [[ -n "${MIGRATION_PACKAGE_VERSIONS_JSON:-}" ]]; then
			versions_json="${MIGRATION_PACKAGE_VERSIONS_JSON}"
		else
			: "${GH_TOKEN:?GH_TOKEN is required}"
			versions_json="$(gh api --paginate --slurp "${versions_endpoint}")"
		fi
		if ! jq -e 'type == "array" and all(.[]; type == "array")' \
			<<<"${versions_json}" >/dev/null; then
			echo "migration image history: package version listing is incomplete or malformed" >&2
			exit 1
		fi
		if jq -e 'any(.[][]; any(.metadata.container.tags[]?; . == "migration-history"))' \
			<<<"${versions_json}" >/dev/null; then
			if [[ -n "${MIGRATION_HISTORY_BOOTSTRAP_SHA:-}" ]]; then
				echo "migration image history: clear MIGRATION_HISTORY_BOOTSTRAP_SHA after bootstrap" >&2
				exit 1
			fi
			echo "history"
			return
		fi
	fi

	if [[ "${MIGRATION_HISTORY_BOOTSTRAP_SHA:-}" != "${CANDIDATE_SHA}" ]]; then
		echo "migration image history: missing marker requires bootstrap SHA equal to candidate" >&2
		exit 1
	fi
	echo "bootstrap"
}

self_test() (
	local fixture
	fixture="$(mktemp -d -t migration-image-history.XXXXXX)"
	trap 'rm -rf -- "${fixture}"' EXIT
	mkdir -p "${fixture}/prior" "${fixture}/candidate"
	printf 'one\n' >"${fixture}/prior/000001_create.sql"
	printf 'three\n' >"${fixture}/prior/000003_extend.sql"
	cp "${fixture}/prior/000001_create.sql" "${fixture}/candidate/000001_create.sql"
	cp "${fixture}/prior/000003_extend.sql" "${fixture}/candidate/000003_extend.sql"
	printf 'four\n' >"${fixture}/candidate/000004_add.sql"
	compare_corpus "${fixture}/prior" "${fixture}/candidate" >/dev/null
	printf 'changed\n' >"${fixture}/candidate/000001_create.sql"
	if (compare_corpus "${fixture}/prior" "${fixture}/candidate") >/dev/null 2>&1; then
		echo "migration image history self-test: rewrite passed" >&2
		exit 1
	fi
	cp "${fixture}/prior/000001_create.sql" "${fixture}/candidate/000001_create.sql"
	printf 'two\n' >"${fixture}/candidate/000002_late.sql"
	if (compare_corpus "${fixture}/prior" "${fixture}/candidate") >/dev/null 2>&1; then
		echo "migration image history self-test: out-of-order addition passed" >&2
		exit 1
	fi

	local packages='[[{"name":"service"}]]'
	local no_packages='[[]]'
	local no_marker='[[{"metadata":{"container":{"tags":["build-1"]}}}]]'
	local marker='[[{"metadata":{"container":{"tags":["migration-history"]}}}]]'
	local mode
	mode="$(
		GITHUB_REPOSITORY_OWNER=owner GITHUB_REPOSITORY_NAME=service GITHUB_REPOSITORY_OWNER_TYPE=Organization \
			CANDIDATE_SHA=candidate MIGRATION_HISTORY_BOOTSTRAP_SHA=candidate \
			MIGRATION_PACKAGE_LIST_JSON="${no_packages}" package_mode
	)"
	[[ "${mode}" == "bootstrap" ]] || { echo "migration image history self-test: empty package did not bootstrap" >&2; exit 1; }
	mode="$(
		GITHUB_REPOSITORY_OWNER=owner GITHUB_REPOSITORY_NAME=service GITHUB_REPOSITORY_OWNER_TYPE=Organization \
			CANDIDATE_SHA=candidate MIGRATION_HISTORY_BOOTSTRAP_SHA=candidate \
			MIGRATION_PACKAGE_LIST_JSON="${packages}" MIGRATION_PACKAGE_VERSIONS_JSON="${no_marker}" package_mode
	)"
	[[ "${mode}" == "bootstrap" ]] || { echo "migration image history self-test: package without marker did not recover" >&2; exit 1; }
	mode="$(
		GITHUB_REPOSITORY_OWNER=owner GITHUB_REPOSITORY_NAME=service GITHUB_REPOSITORY_OWNER_TYPE=Organization \
			CANDIDATE_SHA=candidate MIGRATION_HISTORY_BOOTSTRAP_SHA='' \
			MIGRATION_PACKAGE_LIST_JSON="${packages}" MIGRATION_PACKAGE_VERSIONS_JSON="${marker}" package_mode
	)"
	[[ "${mode}" == "history" ]] || { echo "migration image history self-test: marker did not select history" >&2; exit 1; }
	if (
		GITHUB_REPOSITORY_OWNER=owner GITHUB_REPOSITORY_NAME=service GITHUB_REPOSITORY_OWNER_TYPE=Organization \
			CANDIDATE_SHA=candidate MIGRATION_HISTORY_BOOTSTRAP_SHA=candidate \
			MIGRATION_PACKAGE_LIST_JSON="${packages}" MIGRATION_PACKAGE_VERSIONS_JSON="${marker}" package_mode
	) >/dev/null 2>&1; then
		echo "migration image history self-test: stale bootstrap SHA passed with marker" >&2
		exit 1
	fi
	echo "migration image history self-test passed"
)

case "${1:-}" in
compare)
	[[ $# -eq 3 ]] || usage
	compare_corpus "$2" "$3"
	;;
package-mode)
	[[ $# -eq 1 ]] || usage
	package_mode
	;;
self-test)
	[[ $# -eq 1 ]] || usage
	self_test
	;;
*)
	usage
	;;
esac
