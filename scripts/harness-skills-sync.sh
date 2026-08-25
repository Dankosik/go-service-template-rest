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

read_metadata() {
	local file=$1 line invocation_count=0 kind_count=0
	metadata_invocation=''
	metadata_kind=''
	metadata_disables_model=false
	while IFS= read -r line || [[ -n ${line} ]]; do
		case "${line}" in
		"  invocation: "*) metadata_invocation=${line#"  invocation: "}; ((invocation_count += 1)) ;;
		"  kind: "*) metadata_kind=${line#"  kind: "}; ((kind_count += 1)) ;;
		'disable-model-invocation: true') metadata_disables_model=true ;;
		esac
	done <"${file}"
	((invocation_count == 1 && kind_count == 1))
}

has_exact_line() {
	local expected=$1 file=$2 line
	while IFS= read -r line || [[ -n ${line} ]]; do
		[[ ${line} == "${expected}" ]] && return 0
	done <"${file}"
	return 1
}

validate_metadata() {
	local entry="$1" file invocation kind policy
	file="${entry}/SKILL.md"
	[[ -f "${file}" ]] || fail "${file#"${repo}/"} is missing"
	read_metadata "${file}" || fail "${file#"${repo}/"} has invalid invocation metadata"
	invocation=${metadata_invocation}
	kind=${metadata_kind}
	case "${invocation}/${kind}" in
	model/method) ;;
	user/workflow | role/carrier)
		[[ ${metadata_disables_model} == true ]] ||
			fail "${file#"${repo}/"} must disable implicit harness invocation"
		policy="${entry}/agents/openai.yaml"
		[[ -f "${policy}" ]] ||
			fail "${policy#"${repo}/"} is required for ${invocation} invocation"
		has_exact_line '  allow_implicit_invocation: false' "${policy}" ||
			fail "${policy#"${repo}/"} must disable implicit Codex invocation"
		;;
	*)
		fail "${file#"${repo}/"} has unsupported invocation/kind ${invocation}/${kind}"
		;;
	esac
	if [[ "${invocation}" == model && ${metadata_disables_model} == true ]]; then
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
	local actual entry expected link name i
	local failed=0 linked=0
	local -a actual_targets=() entries=() links=() names=()

	preflight
	skill_directories

	for entry in "${skill_dirs[@]}"; do
		name=${entry##*/}
		link="${links_root}/${name}"
		if [[ -L "${link}" ]]; then
			links+=("${link}")
			names+=("${name}")
		elif [[ -e "${link}" ]]; then
			printf '%s skills: %s is not a symlink\n' "${harness}" "${link#"${repo}/"}" >&2
			failed=1
		else
			printf '%s skills: %s is missing\n' "${harness}" "${link#"${repo}/"}" >&2
			failed=1
		fi
	done
	if ((${#links[@]} > 0)); then
		while IFS= read -r actual; do actual_targets+=("${actual}"); done < <(readlink "${links[@]}")
	fi
	for i in "${!links[@]}"; do
		link=${links[$i]}
		name=${names[$i]}
		actual=${actual_targets[$i]-}
		expected="../../.agents/skills/${name}"
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
	done

	if [[ -d "${links_root}" ]]; then
		shopt -s nullglob dotglob
		entries=("${links_root}"/*)
		shopt -u nullglob dotglob
		if ((${#entries[@]} > 0)); then
			for entry in "${entries[@]}"; do
				name=${entry##*/}
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
		name=${entry##*/}
		ln -s "../../.agents/skills/${name}" "${links_root}/${name}"
	done
	check_links
	;;
check)
	check_links
	;;
esac
