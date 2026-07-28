#!/usr/bin/env bash
# Validate the manifest boundaries that make `make template-sync` safe.
set -euo pipefail

manifest="template-owned.paths"
failed=0

fail() {
	printf 'template-owned purity: %s\n' "$1" >&2
	failed=$((failed + 1))
}

[[ -f "${manifest}" ]] || {
	printf 'template-owned purity: %s is missing\n' "${manifest}" >&2
	exit 1
}

paths=()
while IFS= read -r line; do
	line="${line%%#*}"
	line="${line#"${line%%[![:space:]]*}"}"
	line="${line%"${line##*[![:space:]]}"}"
	[[ -n "${line}" ]] || continue
	paths+=("${line}")
done <"${manifest}"

((${#paths[@]} > 0)) || fail "${manifest} lists no paths"

contains_path() {
	local expected="$1"
	local entry

	for entry in "${paths[@]}"; do
		[[ "${entry}" == "${expected}" ]] && return 0
	done
	return 1
}

# A path outside the repository would let a sync write anywhere on the machine.
for entry in "${paths[@]}"; do
	case "${entry}" in
	/* | */../* | ../* | */..) fail "${manifest} path escapes the repository: ${entry}" ;;
	esac
done

# Every listed path must exist, or a sync would fail midway through a target. An
# empty owned directory is worse than missing: `rsync --delete` from an empty
# source erases whatever the target keeps there.
for entry in "${paths[@]}"; do
	if [[ "${entry}" == */ ]]; then
		if [[ ! -d "${entry%/}" ]]; then
			fail "${manifest} lists a missing directory: ${entry}"
		elif [[ -z "$(find "${entry%/}" -type f -print -quit 2>/dev/null)" ]]; then
			fail "${manifest} lists an empty directory: ${entry}; a sync would erase the target's copy"
		fi
	else
		[[ -f "${entry}" ]] || fail "${manifest} lists a missing file: ${entry}"
	fi
done

# A path already covered by a listed directory would be mirrored twice and makes
# the manifest ambiguous about which entry owns it.
for entry in "${paths[@]}"; do
	[[ "${entry}" == */ ]] && continue
	for parent in "${paths[@]}"; do
		[[ "${parent}" == */ ]] || continue
		case "${entry}" in
		"${parent}"*) fail "${manifest} lists ${entry} inside ${parent}; remove the redundant entry" ;;
		esac
	done
done

# The mechanism has to travel with the instructions it carries. Without these
# entries a derived repository could never update itself again.
for required in "${manifest}" scripts/template-sync.sh scripts/ci/template-owned-purity-check.sh; do
	contains_path "${required}" ||
		fail "${manifest} must list ${required} so the sync mechanism propagates itself"
done

# These documents describe one specific service, proven by divergence across the
# repositories derived from this template. Owning them would overwrite real
# repository decisions on the next sync.
repo_owned=(
	README.md
	docs/build-test-and-development-commands.md
	docs/ci-cd-production-ready.md
	docs/first-production-feature.md
	docs/project-structure-and-module-organization.md
	docs/railway-deployment-profile.md
	docs/repo-architecture.md
)
for reserved in "${repo_owned[@]}"; do
	if contains_path "${reserved}"; then
		fail "${reserved} is repository-owned and must not appear in ${manifest}"
	fi
done

if ((failed != 0)); then
	exit 1
fi

echo "template-owned manifest is structurally safe"
