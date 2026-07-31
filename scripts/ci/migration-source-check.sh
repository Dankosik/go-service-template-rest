#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="${MIGRATION_REPO_ROOT:-$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)}"
cd "${ROOT_DIR}"

if [[ ! -d migrations ]]; then
	echo "migration source is current (pre-first-migration state)"
	exit 0
fi

failed=0
fail() {
	echo "migration source: $*" >&2
	failed=1
}

if [[ -z "$(find migrations -mindepth 1 -maxdepth 1 -print -quit)" ]]; then
	fail "migrations/ must not exist before the first owned migration"
fi

while IFS= read -r entry; do
	[[ -n "${entry}" ]] || continue
	if [[ -L "${entry}" ]]; then
		fail "${entry} is a symbolic link"
	elif [[ -d "${entry}" ]]; then
		fail "${entry} is nested; migrations must be flat"
	elif [[ ! -f "${entry}" ]]; then
		fail "${entry} is not a regular migration file"
	fi
done < <(find migrations -mindepth 1 -maxdepth 1 -print | LC_ALL=C sort)

versions=()
while IFS= read -r file; do
	[[ -n "${file}" ]] || continue
	name="${file##*/}"
	if [[ ! "${name}" =~ ^([0-9]{6})_[a-z][a-z0-9]*(_[a-z0-9]+)*\.sql$ ]]; then
		fail "${file} must match NNNNNN_lower_snake.sql"
		continue
	fi
	version="${BASH_REMATCH[1]}"
	if [[ "${version}" == "000000" ]]; then
		fail "${file} uses reserved version 000000"
	fi
	versions+=("${version}")

	up_count="$(grep -Ec '^[[:space:]]*-- \+goose Up[[:space:]]*$' "${file}" || true)"
	down_count="$(grep -Ec '^[[:space:]]*-- \+goose Down[[:space:]]*$' "${file}" || true)"
	if [[ "${up_count}" != "1" || "${down_count}" != "1" ]]; then
		fail "${file} must contain exactly one '-- +goose Up' and one '-- +goose Down'"
	else
		up_line="$(grep -En '^[[:space:]]*-- \+goose Up[[:space:]]*$' "${file}" | cut -d: -f1)"
		down_line="$(grep -En '^[[:space:]]*-- \+goose Down[[:space:]]*$' "${file}" | cut -d: -f1)"
		if ((up_line >= down_line)); then
			fail "${file} must place '-- +goose Up' before '-- +goose Down'"
		fi
	fi
	if grep -Eq '^[[:space:]]*-- \+goose NO TRANSACTION[[:space:]]*$' "${file}"; then
		fail "${file} cannot use '-- +goose NO TRANSACTION'"
	fi
	if grep -Eq '^[[:space:]]*-- \+goose ENVSUB (ON|OFF)[[:space:]]*$' "${file}"; then
		fail "${file} cannot use Goose environment substitution"
	fi
	if ! awk '
		/^[[:space:]]*--.*\+goose/ &&
			$0 !~ /^-- \+goose (Up|Down|StatementBegin|StatementEnd)\r?$/ {
			exit 1
		}
	' "${file}"; then
		fail "${file} contains a non-canonical or forbidden Goose annotation"
	fi
done < <(find migrations -mindepth 1 -maxdepth 1 -type f -print | LC_ALL=C sort)

if ((${#versions[@]} > 0)); then
	duplicates="$(printf '%s\n' "${versions[@]}" | LC_ALL=C sort | uniq -d)"
	if [[ -n "${duplicates}" ]]; then
		fail "duplicate migration versions: ${duplicates//$'\n'/, }"
	fi
fi

if ((failed != 0)); then
	exit 1
fi

bash ./scripts/run-go-tool.sh goose -dir migrations validate
echo "migration source is current"
