#!/usr/bin/env bash
# Enforce the invariant that makes `make template-sync` safe: a template-owned
# path carries no repository-specific content, so mirroring it verbatim into a
# derived repository can never overwrite that repository's own decisions.
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
	printf '%s\n' "${paths[@]}" | grep -Fxq "${required}" ||
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
	if printf '%s\n' "${paths[@]}" | grep -Fxq "${reserved}"; then
		fail "${reserved} is repository-owned and must not appear in ${manifest}"
	fi
done

# An owned file must not name this repository. If it does, every derived
# repository would receive this template's identity instead of keeping its own.
if [[ -f go.mod ]]; then
	module=$(awk '$1 == "module" { print $2; exit }' go.mod)
	if [[ -n "${module}" ]]; then
		for entry in "${paths[@]}"; do
			while IFS= read -r hit; do
				[[ -n "${hit}" ]] || continue
				fail "${hit} names the module path ${module}; move that content to a repository-owned document"
			done < <(grep -rlF -- "${module}" "${entry%/}" 2>/dev/null || true)
		done
	fi
fi

if ((failed != 0)); then
	exit 1
fi

echo "template-owned paths carry no repository-specific content"
