# shellcheck shell=bash
# The `--<mode> [--repo <dir>]` command line the sync helpers share.
#
# It is one file because the helpers accept the same line, and the line is
# security-relevant in the same way for each: --repo is resolved to a real
# directory before anything writes into it, and an unrecognized flag is refused
# rather than ignored. Multiple copies would let one helper keep a rule another
# lost.
#
# What stays with each script is what differs: which modes it accepts, its usage
# text, and the prefix its failures carry. This calls back into the caller's
# `usage` and `fail` for exactly that reason — a shared parser that invented its
# own wording would report one harness failure as another.

# sync_cli_parse accepts the caller's modes and argv, and publishes the result as
# SYNC_MODE and SYNC_REPO. A caller defines `usage` and `fail` before calling it.
#
#   sync_cli_parse "preflight apply check" "$@"
sync_cli_parse() {
	local accepted=" $1 "
	shift

	SYNC_MODE=""
	SYNC_REPO="${PWD}"

	while (($# > 0)); do
		case "$1" in
		--repo)
			[[ $# -ge 2 ]] || fail "--repo needs a directory"
			SYNC_REPO="$2"
			shift 2
			;;
		-h | --help)
			usage
			exit 0
			;;
		--*)
			[[ "${accepted}" == *" ${1#--} "* ]] || fail "unknown argument: $1"
			[[ -z "${SYNC_MODE}" ]] || fail "choose exactly one mode"
			SYNC_MODE="${1#--}"
			shift
			;;
		*)
			fail "unknown argument: $1"
			;;
		esac
	done

	[[ -n "${SYNC_MODE}" ]] || {
		usage >&2
		exit 2
	}

	# Resolved here rather than at each use: every later path is built from it,
	# and a directory that does not exist must fail before the first write.
	SYNC_REPO=$(CDPATH='' cd -- "${SYNC_REPO}" 2>/dev/null && pwd) ||
		fail "repository directory not found"
}
