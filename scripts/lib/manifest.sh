# shellcheck shell=bash
# Reading template-owned.paths.
#
# The sync writes into every path this returns and the purity check judges every
# path this returns, so the two must agree on what the file says: which text is a
# comment, what whitespace is, and that a path may not leave the repository. A
# second copy of that parse is a sync that mirrors an entry the check never saw.

# manifest_paths reads one manifest into the global `paths` array.
#
# An entry that escapes the repository is reported through the caller's `fail`,
# because the two callers answer it differently and both answers are right: the
# sync stops before it can write outside the target, and the purity check keeps
# going so one run names every bad entry.
manifest_paths() {
	local manifest_file="$1"
	local line

	paths=()
	while IFS= read -r line; do
		line="${line%%#*}"
		line="${line#"${line%%[![:space:]]*}"}"
		line="${line%"${line##*[![:space:]]}"}"
		[[ -n "${line}" ]] || continue
		case "${line}" in
		/* | */../* | ../* | */..) fail "manifest path escapes the repository: ${line}" ;;
		esac
		paths+=("${line}")
	done <"${manifest_file}"
}
