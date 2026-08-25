#!/usr/bin/env bash
# Emit unique changed paths, including both sides of renames and copies.
#
#   git-changed-paths.sh --worktree BASE_REF
#   git-changed-paths.sh --diff BASE HEAD
#   git-changed-paths.sh --self-test
set -euo pipefail

usage() {
	echo "usage: $0 --worktree BASE_REF | --diff BASE HEAD | --self-test" >&2
	exit 2
}

expand_name_status() {
	local status path
	while IFS= read -r -d '' status; do
		[[ -n ${status} ]] || continue
		IFS= read -r -d '' path || return 0
		[[ -n ${path} ]] && printf '%s\n' "${path}"
		case "${status}" in
		R* | C*)
			IFS= read -r -d '' path || return 0
			[[ -n ${path} ]] && printf '%s\n' "${path}"
			;;
		esac
	done
}

require_commit() {
	local ref=$1
	git rev-parse --verify "${ref}^{commit}" >/dev/null 2>&1 || {
		echo "verification base is unavailable: ${ref}; set BASE_REF to a readable commit" >&2
		exit 2
	}
}

worktree_paths() {
	local base_ref=$1
	require_commit "${base_ref}"
	{
		git diff --name-status -z --find-renames "${base_ref}...HEAD" | expand_name_status
		git diff --name-status -z --find-renames | expand_name_status
		git diff --cached --name-status -z --find-renames | expand_name_status
		git ls-files --others --exclude-standard
	} | LC_ALL=C sort -u
}

diff_paths() {
	local base=$1 head=$2
	require_commit "${base}"
	require_commit "${head}"
	git diff --name-status -z --find-renames "${base}" "${head}" | expand_name_status | LC_ALL=C sort -u
}

self_test() {
	local tmp repo output script
	tmp=$(mktemp -d)
	trap 'rm -rf -- "${tmp}"' RETURN
	script=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/git-changed-paths.sh
	repo=${tmp}/repo
	git init -q "${repo}"
	git -C "${repo}" config user.name changed-paths
	git -C "${repo}" config user.email changed-paths@example.invalid
	mkdir -p "${repo}/oldpkg"
	printf 'package oldpkg\n' >"${repo}/oldpkg/a.go"
	git -C "${repo}" add oldpkg/a.go
	git -C "${repo}" commit -qm start
	mkdir -p "${repo}/newpkg"
	git -C "${repo}" mv oldpkg/a.go newpkg/a.go
	git -C "${repo}" commit -qm move
	output=$(cd "${repo}" && bash "${script}" --diff HEAD~1 HEAD)
	grep -qx 'oldpkg/a.go' <<<"${output}"
	grep -qx 'newpkg/a.go' <<<"${output}"
	[[ $(grep -c . <<<"${output}") -eq 2 ]]
}

case "${1:-}" in
--self-test)
	self_test
	;;
--worktree)
	[[ $# -eq 2 ]] || usage
	worktree_paths "$2"
	;;
--diff)
	[[ $# -eq 3 ]] || usage
	diff_paths "$2" "$3"
	;;
*)
	usage
	;;
esac
