#!/usr/bin/env bash
# Expose the canonical `.agents/skills/` set to Claude Code through one
# generated relative symlink per skill.
set -euo pipefail

usage() {
	cat <<'EOF'
usage:
  claude-skills-sync.sh --preflight [--repo <repository>]
  claude-skills-sync.sh --apply     [--repo <repository>]
  claude-skills-sync.sh --check     [--repo <repository>]

  --preflight  refuse path shapes that an apply cannot replace safely
  --apply      rebuild `.claude/skills` from `.agents/skills`
  --check      verify exact link coverage and targets without changing files
  --repo       repository root (default: current working directory)
EOF
}

fail() {
	printf 'claude skills: %s\n' "$1" >&2
	exit 1
}

mode=""
repo="${PWD}"

while (($# > 0)); do
	case "$1" in
	--preflight | --apply | --check)
		[[ -z "${mode}" ]] || fail "choose exactly one mode"
		mode="${1#--}"
		shift
		;;
	--repo)
		[[ $# -ge 2 ]] || fail "--repo needs a directory"
		repo="$2"
		shift 2
		;;
	-h | --help)
		usage
		exit 0
		;;
	*)
		fail "unknown argument: $1"
		;;
	esac
done

[[ -n "${mode}" ]] || {
	usage >&2
	exit 2
}

repo=$(CDPATH='' cd -- "${repo}" 2>/dev/null && pwd) ||
	fail "repository directory not found"
skills_root="${repo}/.agents/skills"
links_root="${repo}/.claude/skills"

preflight() {
	local entry
	local -a entries=()

	[[ ! -L "${repo}/.claude" ]] ||
		fail ".claude is a symlink; generated links must stay inside the repository"
	[[ ! -L "${links_root}" ]] ||
		fail ".claude/skills is a symlink; per-skill links must be generated inside that directory"
	if [[ -e "${links_root}" && ! -d "${links_root}" ]]; then
		fail ".claude/skills is not a directory"
	fi
	if [[ -d "${links_root}" ]]; then
		shopt -s nullglob dotglob
		entries=("${links_root}"/*)
		shopt -u nullglob dotglob
		for entry in "${entries[@]}"; do
			if [[ -d "${entry}" && ! -L "${entry}" ]]; then
				fail "${entry#"${repo}/"} is a real directory; move or remove it before rebuilding generated links"
			fi
		done
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
				printf 'claude skills: %s points at %s, expected %s\n' \
					"${link#"${repo}/"}" "${actual}" "${expected}" >&2
				failed=1
			elif [[ ! -f "${link}/SKILL.md" ]]; then
				printf 'claude skills: %s does not resolve to a readable SKILL.md\n' \
					"${link#"${repo}/"}" >&2
				failed=1
			else
				linked=$((linked + 1))
			fi
		elif [[ -e "${link}" ]]; then
			printf 'claude skills: %s is not a symlink\n' "${link#"${repo}/"}" >&2
			failed=1
		else
			printf 'claude skills: %s is missing\n' "${link#"${repo}/"}" >&2
			failed=1
		fi
	done

	if [[ -d "${links_root}" ]]; then
		shopt -s nullglob dotglob
		entries=("${links_root}"/*)
		shopt -u nullglob dotglob
		for entry in "${entries[@]}"; do
			name=$(basename -- "${entry}")
			if [[ ! -d "${skills_root}/${name}" ]]; then
				printf 'claude skills: %s has no owner in .agents/skills\n' \
					"${entry#"${repo}/"}" >&2
				failed=1
			fi
		done
	fi

	((failed == 0)) || return 1
	printf 'claude skills: %d skill links current\n' "${linked}"
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
