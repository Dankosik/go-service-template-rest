#!/usr/bin/env bash
set -euo pipefail

integration_packages=()
webhook_race_packages=()
read -r -a integration_packages <<<"${INTEGRATION_PACKAGES:-}" || true
read -r -a webhook_race_packages <<<"${WEBHOOK_RACE_PACKAGES:-}" || true

package_is_routed() {
	local package=$1
	shift
	local route base
	for route in "$@"; do
		if [[ ${route} == "${package}" ]]; then
			return 0
		fi
		if [[ ${route} == */... ]]; then
			base=${route%/...}
			if [[ ${package} == "${base}" || ${package} == "${base}/"* ]]; then
				return 0
			fi
		fi
	done
	return 1
}

status=0
while IFS= read -r file; do
	file=${file#./}
	[[ -f ${file} ]] || continue
	package=./$(dirname "${file}")
	if ! package_is_routed "${package}" "${integration_packages[@]}"; then
		printf '%s: integration package %s is absent from INTEGRATION_PACKAGES\n' "${file}" "${package}" >&2
		status=1
	fi
done < <(git ls-files --cached --others --exclude-standard -- '*_integration_test.go' | LC_ALL=C sort)

while IFS= read -r file; do
	file=${file#./}
	[[ -f ${file} ]] || continue
	package=./$(dirname "${file}")
	if ! package_is_routed "${package}" "${webhook_race_packages[@]}"; then
		printf '%s: webhook race package %s is absent from WEBHOOK_RACE_PACKAGES\n' "${file}" "${package}" >&2
		status=1
	fi
done < <(rg -l '^func TestWebhookNetwork' --glob '*_integration_test.go' . || true)

exit "${status}"
