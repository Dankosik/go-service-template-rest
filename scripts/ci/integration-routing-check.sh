#!/usr/bin/env bash
set -euo pipefail

package_is_routed() {
	local package=$1
	shift
	local route base
	for route in "$@"; do
		if [[ ${route} == "${package}" ]]; then return 0; fi
		if [[ ${route} == */... ]]; then
			base=${route%/...}
			if [[ ${package} == "${base}" || ${package} == "${base}/"* ]]; then return 0; fi
		fi
	done
	return 1
}

check_routes() {
	local root=$1 file package status=0 grep_status webhook_files
	local integration_packages=() webhook_race_packages=()
	read -r -a integration_packages <<<"${INTEGRATION_PACKAGES:-}"
	read -r -a webhook_race_packages <<<"${WEBHOOK_RACE_PACKAGES:-}"

	while IFS= read -r file; do
		[[ -n ${file} && -f ${root}/${file} ]] || continue
		package=./$(dirname "${file}")
		if ((${#integration_packages[@]} == 0)) || ! package_is_routed "${package}" "${integration_packages[@]}"; then
			printf '%s: integration package %s is absent from INTEGRATION_PACKAGES\n' "${file}" "${package}" >&2
			status=1
		fi
	done < <(git -C "${root}" ls-files --cached --others --exclude-standard -- '*_integration_test.go' | LC_ALL=C sort)

	set +e
	webhook_files=$(git -C "${root}" grep --untracked -l '^func TestWebhookNetwork' -- '*_integration_test.go')
	grep_status=$?
	set -e
	if ((grep_status > 1)); then
		echo "git grep failed while locating webhook race tests" >&2
		return "${grep_status}"
	fi
	while IFS= read -r file; do
		[[ -n ${file} && -f ${root}/${file} ]] || continue
		package=./$(dirname "${file}")
		if ((${#webhook_race_packages[@]} == 0)) || ! package_is_routed "${package}" "${webhook_race_packages[@]}"; then
			printf '%s: webhook race package %s is absent from WEBHOOK_RACE_PACKAGES\n' "${file}" "${package}" >&2
			status=1
		fi
	done <<<"${webhook_files}"
	return "${status}"
}

self_test() {
	local tmp output
	tmp=$(mktemp -d)
	trap 'rm -rf -- "${tmp}"' RETURN
	git -C "${tmp}" init -q
	git -C "${tmp}" config user.email fixture@example.invalid
	git -C "${tmp}" config user.name fixture
	mkdir -p "${tmp}/test"
	printf 'package test\n\nfunc TestWebhookNetworkFixture() {}\n' >"${tmp}/test/webhook_network_integration_test.go"
	git -C "${tmp}" add test/webhook_network_integration_test.go
	if output=$(INTEGRATION_PACKAGES=./test WEBHOOK_RACE_PACKAGES='' check_routes "${tmp}" 2>&1); then
		echo "integration routing self-test accepted an unrouted webhook race package" >&2
		return 1
	fi
	grep -q 'absent from WEBHOOK_RACE_PACKAGES' <<<"${output}"
}

if [[ ${1:-} == --self-test ]]; then
	self_test
	exit
fi

check_routes "${1:-.}"
