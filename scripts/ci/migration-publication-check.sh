#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
CHECK="${ROOT_DIR}/scripts/ci/migration-image-history-check.sh"
fixture="$(mktemp -d)"
trap 'rm -rf "${fixture}"' EXIT INT TERM

mkdir -p "${fixture}/prior" "${fixture}/candidate"
printf 'one\n' >"${fixture}/prior/000001_create.sql"
cp "${fixture}/prior/000001_create.sql" "${fixture}/candidate/000001_create.sql"
printf 'two\n' >"${fixture}/candidate/000002_add.sql"
bash "${CHECK}" compare "${fixture}/prior" "${fixture}/candidate" >/dev/null

printf 'changed\n' >"${fixture}/candidate/000001_create.sql"
if bash "${CHECK}" compare "${fixture}/prior" "${fixture}/candidate" >/dev/null 2>&1; then
	echo "migration publication self-test: changed prior file passed" >&2
	exit 1
fi
cp "${fixture}/prior/000001_create.sql" "${fixture}/candidate/000001_create.sql"
rm "${fixture}/candidate/000001_create.sql"
if bash "${CHECK}" compare "${fixture}/prior" "${fixture}/candidate" >/dev/null 2>&1; then
	echo "migration publication self-test: deleted prior file passed" >&2
	exit 1
fi

package_env=(
	GITHUB_REPOSITORY_OWNER=acme
	GITHUB_REPOSITORY_NAME=service
	GITHUB_REPOSITORY_OWNER_TYPE=Organization
	CANDIDATE_SHA=abc123
)
env "${package_env[@]}" \
	MIGRATION_PACKAGE_LIST_JSON='[[]]' \
	MIGRATION_HISTORY_BOOTSTRAP_SHA=abc123 \
	bash "${CHECK}" package-mode |
	grep -Fxq bootstrap

if env "${package_env[@]}" \
	MIGRATION_PACKAGE_LIST_JSON='[[]]' \
	MIGRATION_HISTORY_BOOTSTRAP_SHA=wrong \
	bash "${CHECK}" package-mode >/dev/null 2>&1; then
	echo "migration publication self-test: mismatched bootstrap SHA passed" >&2
	exit 1
fi
if env "${package_env[@]}" \
	MIGRATION_PACKAGE_LIST_JSON='[[{"name":"service"}]]' \
	MIGRATION_HISTORY_BOOTSTRAP_SHA=abc123 \
	bash "${CHECK}" package-mode >/dev/null 2>&1; then
	echo "migration publication self-test: stale bootstrap authority passed" >&2
	exit 1
fi
env "${package_env[@]}" \
	MIGRATION_PACKAGE_LIST_JSON='[[{"name":"service"}]]' \
	MIGRATION_HISTORY_BOOTSTRAP_SHA= \
	bash "${CHECK}" package-mode |
	grep -Fxq history

for malformed in '{}' 'null' '[[{"name":"other"}],null]'; do
	if env "${package_env[@]}" \
		MIGRATION_PACKAGE_LIST_JSON="${malformed}" \
		MIGRATION_HISTORY_BOOTSTRAP_SHA=abc123 \
		bash "${CHECK}" package-mode >/dev/null 2>&1; then
		echo "migration publication self-test: malformed package listing passed" >&2
		exit 1
	fi
done

assert_publication_order() {
	local job="$1"
	local promotion="$2"
	local section marker_line promotion_line

	section="$(
		awk -v heading="  ${job}:" '
			$0 == heading { inside = 1 }
			inside && $0 ~ /^  [[:alnum:]_-]+:/ && $0 != heading { exit }
			inside { print }
		' "${ROOT_DIR}/.github/workflows/cd.yml"
	)"
	[[ -n "${section}" ]] || {
		echo "migration publication self-test: ${job} is missing" >&2
		exit 1
	}
	# GitHub expression syntax is intentionally literal here.
	# shellcheck disable=SC2016
	grep -Fq 'group: migration-publication-${{ github.repository }}' <<<"${section}" || {
		echo "migration publication self-test: ${job} lacks shared publication concurrency" >&2
		exit 1
	}
	marker_line="$(
		grep -n -F -- '- name: Advance verified migration history marker' <<<"${section}" |
			cut -d: -f1
	)"
	promotion_line="$(
		grep -n -F -- "- name: ${promotion}" <<<"${section}" |
			cut -d: -f1
	)"
	if [[ ! "${marker_line}" =~ ^[0-9]+$ ]] ||
		[[ ! "${promotion_line}" =~ ^[0-9]+$ ]] ||
		((marker_line >= promotion_line)); then
		echo "migration publication self-test: ${job} promotes a public alias before the history marker" >&2
		exit 1
	fi
}

assert_publication_order publish-main "Promote verified main tag"
assert_publication_order publish-release "Promote verified release tags"

publish_main_guard="$(
	awk '
		$0 == "  publish-main:" { inside = 1 }
		inside && /^    if:/ { guard = 1 }
		guard && /^    runs-on:/ { exit }
		guard { print }
	' "${ROOT_DIR}/.github/workflows/cd.yml"
)"
grep -Fq "github.event.workflow_run.event == 'push'" <<<"${publish_main_guard}" || {
	echo "migration publication self-test: publish-main accepts non-push CI" >&2
	exit 1
}

echo "migration publication self-test passed"
