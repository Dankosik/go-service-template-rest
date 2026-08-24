#!/usr/bin/env bash
# Expose the canonical `.agents/skills/` set to one supported harness through
# one generated relative symlink per skill.
set -euo pipefail

usage() {
	cat <<'EOF'
usage:
  harness-skills-sync.sh <claude|qwen> --preflight [--repo <repository>]
  harness-skills-sync.sh <claude|qwen> --apply     [--repo <repository>]
  harness-skills-sync.sh <claude|qwen> --check     [--repo <repository>]

  --preflight  refuse path shapes that an apply cannot replace safely
  --apply      rebuild the harness skill view from `.agents/skills`
  --check      verify exact link coverage and targets without changing files
  --repo       repository root (default: current working directory)
EOF
}

fail() {
	printf '%s skills: %s\n' "${harness:-harness}" "$1" >&2
	exit 1
}

harness="${1:-}"
case "${harness}" in
claude | qwen) shift ;;
*)
	usage >&2
	exit 2
	;;
esac

# shellcheck source=scripts/lib/sync-cli.sh
source "$(dirname -- "${BASH_SOURCE[0]}")/lib/sync-cli.sh"
sync_cli_parse "preflight apply check" "$@"
mode="${SYNC_MODE}"
repo="${SYNC_REPO}"

skills_root="${repo}/.agents/skills"
harness_root="${repo}/.${harness}"
links_root="${harness_root}/skills"

metadata_value() {
	local key="$1" file="$2"
	sed -n "s/^  ${key}: //p" "${file}"
}

validate_metadata() {
	local entry="$1" file invocation kind policy
	file="${entry}/SKILL.md"
	[[ -f "${file}" ]] || fail "${file#"${repo}/"} is missing"
	invocation=$(metadata_value invocation "${file}")
	kind=$(metadata_value kind "${file}")
	case "${invocation}/${kind}" in
	model/method) ;;
	user/workflow | role/carrier)
		grep -Fxq 'disable-model-invocation: true' "${file}" ||
			fail "${file#"${repo}/"} must disable implicit harness invocation"
		policy="${entry}/agents/openai.yaml"
		[[ -f "${policy}" ]] ||
			fail "${policy#"${repo}/"} is required for ${invocation} invocation"
		grep -Fxq '  allow_implicit_invocation: false' "${policy}" ||
			fail "${policy#"${repo}/"} must disable implicit Codex invocation"
		;;
	*)
		fail "${file#"${repo}/"} has unsupported invocation/kind ${invocation}/${kind}"
		;;
	esac
	if [[ "${invocation}" == model ]] && grep -Fxq 'disable-model-invocation: true' "${file}"; then
		fail "${file#"${repo}/"} disables its model invocation"
	fi
}

preflight() {
	local entry
	local -a entries=()

	[[ ! -L "${harness_root}" ]] ||
		fail ".${harness} is a symlink; generated links must stay inside the repository"
	if [[ -e "${harness_root}" && ! -d "${harness_root}" ]]; then
		fail ".${harness} is not a directory"
	fi
	[[ ! -L "${links_root}" ]] ||
		fail ".${harness}/skills is a symlink; per-skill links must be generated inside that directory"
	if [[ -e "${links_root}" && ! -d "${links_root}" ]]; then
		fail ".${harness}/skills is not a directory"
	fi
	if [[ -d "${links_root}" ]]; then
		shopt -s nullglob dotglob
		entries=("${links_root}"/*)
		shopt -u nullglob dotglob
		if ((${#entries[@]} > 0)); then
			for entry in "${entries[@]}"; do
				if [[ ! -L "${entry}" ]]; then
					fail "${entry#"${repo}/"} is not a generated symlink; move or remove it before rebuilding generated links"
				fi
			done
		fi
	fi
}

skill_directories() {
	local entry
	skill_dirs=()
	[[ -d "${skills_root}" ]] || fail ".agents/skills is missing"
	shopt -s nullglob
	for entry in "${skills_root}"/*; do
		[[ -d "${entry}" ]] || continue
		[[ ! -L "${entry}" ]] ||
			fail "${entry#"${repo}/"} is a symlink; canonical skill directories must be real"
		validate_metadata "${entry}"
		skill_dirs+=("${entry}")
	done
	shopt -u nullglob
	((${#skill_dirs[@]} > 0)) || fail ".agents/skills contains no skills"
}

check_links() {
	local actual entry expected link name
	local failed=0 linked=0
	local -a entries=()

	preflight
	skill_directories

	for entry in "${skill_dirs[@]}"; do
		name=$(basename -- "${entry}")
		link="${links_root}/${name}"
		expected="../../.agents/skills/${name}"
		if [[ -L "${link}" ]]; then
			actual=$(readlink "${link}")
			if [[ "${actual}" != "${expected}" ]]; then
				printf '%s skills: %s points at %s, expected %s\n' "${harness}" \
					"${link#"${repo}/"}" "${actual}" "${expected}" >&2
				failed=1
			elif [[ ! -f "${link}/SKILL.md" ]]; then
				printf '%s skills: %s does not resolve to a readable SKILL.md\n' \
					"${harness}" "${link#"${repo}/"}" >&2
				failed=1
			else
				linked=$((linked + 1))
			fi
		elif [[ -e "${link}" ]]; then
			printf '%s skills: %s is not a symlink\n' "${harness}" "${link#"${repo}/"}" >&2
			failed=1
		else
			printf '%s skills: %s is missing\n' "${harness}" "${link#"${repo}/"}" >&2
			failed=1
		fi
	done

	if [[ -d "${links_root}" ]]; then
		shopt -s nullglob dotglob
		entries=("${links_root}"/*)
		shopt -u nullglob dotglob
		if ((${#entries[@]} > 0)); then
			for entry in "${entries[@]}"; do
				name=$(basename -- "${entry}")
				if [[ ! -d "${skills_root}/${name}" ]]; then
					printf '%s skills: %s has no owner in .agents/skills\n' \
						"${harness}" "${entry#"${repo}/"}" >&2
					failed=1
				fi
			done
		fi
	fi

	((failed == 0)) || return 1
	printf '%s skills: %d skill links current\n' "${harness}" "${linked}"
}

case "${mode}" in
preflight)
	preflight
	;;
apply)
	preflight
	skill_directories
	mkdir -p "${links_root}"
	shopt -s nullglob dotglob
	entries=("${links_root}"/*)
	shopt -u nullglob dotglob
	((${#entries[@]} == 0)) || rm -f -- "${entries[@]}"
	for entry in "${skill_dirs[@]}"; do
		name=$(basename -- "${entry}")
		ln -s "../../.agents/skills/${name}" "${links_root}/${name}"
	done
	check_links
	;;
check)
	check_links
	;;
esac
