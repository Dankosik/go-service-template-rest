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

template_sync_behavior_check() (
	local fixture template target target_with_link outside sync_script
	fixture=$(mktemp -d "${TMPDIR:-/tmp}/template-sync-check.XXXXXX")
	trap 'rm -rf -- "${fixture}"' EXIT
	template="${fixture}/template"
	target="${fixture}/target"
	target_with_link="${fixture}/target-with-link"
	outside="${fixture}/outside"
	sync_script="$(pwd)/scripts/template-sync.sh"

	mkdir -p "${template}/owned" "${outside}"
	printf 'owned/\n' >"${template}/template-owned.paths"
	printf 'v1\n' >"${template}/owned/version"
	git -C "${template}" init -q
	git -C "${template}" add template-owned.paths owned/version
	git -C "${template}" -c user.name=template-sync-check -c user.email=template-sync-check@example.invalid commit -qm v1
	git clone -q "${template}" "${target}"

	printf 'v2\n' >"${template}/owned/version"
	git -C "${template}" add owned/version
	git -C "${template}" -c user.name=template-sync-check -c user.email=template-sync-check@example.invalid commit -qm v2

	printf 'owned/ignored.txt\n' >"${template}/.git/info/exclude"
	printf 'ignored source\n' >"${template}/owned/ignored.txt"
	if bash "${sync_script}" --apply --no-commit --from "${template}" --repo "${target}" >/dev/null 2>&1; then
		echo "template-owned purity: sync accepted ignored source content" >&2
		return 1
	fi
	grep -Fxq v1 "${target}/owned/version" || {
		echo "template-owned purity: ignored source content changed the target" >&2
		return 1
	}
	rm "${template}/owned/ignored.txt"

	printf 'owned/ignored.txt\n' >"${target}/.git/info/exclude"
	printf 'ignored target\n' >"${target}/owned/ignored.txt"
	if bash "${sync_script}" --apply --no-commit --from "${template}" --repo "${target}" >/dev/null 2>&1; then
		echo "template-owned purity: sync accepted ignored target content" >&2
		return 1
	fi
	grep -Fxq 'ignored target' "${target}/owned/ignored.txt" || {
		echo "template-owned purity: ignored target content was deleted" >&2
		return 1
	}
	rm "${target}/owned/ignored.txt"

	git -C "${target}" checkout -q --detach
	bash "${sync_script}" --apply --no-commit --from "${template}" --repo "${target}" >/dev/null
	grep -Fxq v2 "${target}/owned/version" || {
		echo "template-owned purity: detached --no-commit sync did not apply" >&2
		return 1
	}
	mkdir "${template}/owned/empty"
	bash "${sync_script}" --check --from "${template}" --repo "${target}" >/dev/null

	git clone -q "${template}" "${target_with_link}"
	printf 'outside\n' >"${outside}/sentinel"
	git -C "${target_with_link}" rm -qr owned
	ln -s "${outside}" "${target_with_link}/owned"
	git -C "${target_with_link}" add owned
	git -C "${target_with_link}" -c user.name=template-sync-check -c user.email=template-sync-check@example.invalid commit -qm symlink
	if bash "${sync_script}" --apply --no-commit --from "${template}" --repo "${target_with_link}" >/dev/null 2>&1; then
		echo "template-owned purity: sync followed a target symlink" >&2
		return 1
	fi
	grep -Fxq outside "${outside}/sentinel" || {
		echo "template-owned purity: symlink refusal changed outside content" >&2
		return 1
	}
	[[ ! -e "${outside}/version" ]] || {
		echo "template-owned purity: sync copied through a target symlink" >&2
		return 1
	}
)

template_sync_behavior_check
echo "template-owned manifest and sync behavior are safe"
