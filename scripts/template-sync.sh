#!/usr/bin/env bash
# Mirror the template-owned instruction surface between this template and the
# repositories derived from it. The manifest `template-owned.paths` is the only
# authority for what moves; nothing outside it is read, written, or staged.
#
# The sync never commits work it did not produce. A target whose manifest paths
# hold uncommitted changes is refused, because overwriting them would destroy
# them and committing them would push unfinished work toward a release.
set -euo pipefail

usage() {
	cat <<'EOF'
usage:
  template-sync.sh --check  [--from <template-dir>] [--repo <target-dir>]
  template-sync.sh --apply  [--from <template-dir>] [--repo <target-dir>] [--no-commit]
  template-sync.sh --apply  --targets <dir> [<dir>...] [--no-commit]

  --check     report drift for every manifest path; exit 1 when any target drifts
  --apply     mirror the manifest into the target, then commit only those paths
  --from      template checkout that owns the manifest (default: this script's repo)
  --repo      target repository (default: the current working directory)
  --targets   apply to several targets in one run
  --no-commit leave the mirrored result in the working tree without committing

Uncommitted work outside the manifest is never read, staged, or touched: a target
can carry any amount of work in progress and still sync. Uncommitted work inside
the manifest refuses that target by name, before anything is written, and the run
exits non-zero. Commit or discard those paths yourself, then sync again.
EOF
}

fail() {
	printf 'template-sync: %s\n' "$1" >&2
	exit 1
}

# A target-level problem: report it, keep going, exit non-zero at the end.
reject() {
	printf '   refused: %s\n' "$1"
	rejected_targets=$((rejected_targets + 1))
}

script_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
default_template=$(CDPATH= cd -- "${script_dir}/.." && pwd)

mode=""
template="${default_template}"
targets=()
explicit_repo=""
commit=true

while (($# > 0)); do
	case "$1" in
	--check | --apply)
		[[ -z "${mode}" ]] || fail "choose exactly one of --check or --apply"
		mode="${1#--}"
		shift
		;;
	--from)
		[[ $# -ge 2 ]] || fail "--from needs a directory"
		template="$2"
		shift 2
		;;
	--repo)
		[[ $# -ge 2 ]] || fail "--repo needs a directory"
		explicit_repo="$2"
		shift 2
		;;
	--targets)
		shift
		[[ $# -ge 1 ]] || fail "--targets needs at least one directory"
		while (($# > 0)) && [[ "$1" != --* ]]; do
			targets+=("$1")
			shift
		done
		;;
	--no-commit)
		commit=false
		shift
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

command -v rsync >/dev/null 2>&1 || fail "rsync is required"

template=$(CDPATH= cd -- "${template}" 2>/dev/null && pwd) || fail "template directory not found"
manifest="${template}/template-owned.paths"
[[ -f "${manifest}" ]] || fail "manifest not found: ${manifest}"

if ((${#targets[@]} == 0)); then
	targets+=("${explicit_repo:-$PWD}")
fi

# Manifest entries, comments and blanks removed. A manifest path must stay inside
# the repository: an absolute path or a parent traversal would let a sync write
# anywhere on the machine.
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
done <"${manifest}"
((${#paths[@]} > 0)) || fail "manifest lists no paths"

# Tracked paths the sync regenerates as a side effect of mirroring `.agents/skills`.
# They are generated, so they are absent from the manifest, but the sync still has
# to commit what it changed rather than leave the target dirty behind it.
generated_paths=(.claude/skills)

template_revision=$(git -C "${template}" rev-parse --short HEAD 2>/dev/null || echo "")

# `.template-sync` records the revision a target was synced from, so the source
# tree has to match that revision. Mirroring uncommitted template edits would
# stamp every target with a revision that never contained what they received.
if [[ "${mode}" == "apply" ]]; then
	[[ -n "${template_revision}" ]] ||
		fail "template is not a git repository, so no source revision can be recorded: ${template}"
	template_pathspecs=()
	for entry in "${paths[@]}"; do template_pathspecs+=("${entry%/}"); done
	if [[ -n "$(git -C "${template}" status --porcelain -- "${template_pathspecs[@]}")" ]]; then
		printf 'template-sync: the template has uncommitted changes inside its own manifest:\n' >&2
		git -C "${template}" status --porcelain -- "${template_pathspecs[@]}" | sed 's/^/  /' >&2
		fail "commit them first so .template-sync can name the revision targets actually received"
	fi
fi

# Compare one manifest entry. Prints a human-readable delta, returns 1 on drift.
diff_entry() {
	local repo="$1" entry="$2" source="${template}/$2" destination
	destination="${repo}/${entry%/}"
	if [[ "${entry}" == */ ]]; then
		[[ -d "${source}" ]] || fail "manifest lists a missing template directory: ${entry}"
		if [[ ! -d "${destination}" ]]; then
			printf '  + %s (absent in target)\n' "${entry}"
			return 1
		fi
		local delta
		delta=$(diff -rq "${source%/}" "${destination}" 2>&1 || true)
		[[ -z "${delta}" ]] && return 0
		printf '%s\n' "${delta}" | sed "s|${template}/||g; s|${repo}/||g; s|^|  |"
		return 1
	fi
	[[ -f "${source}" ]] || fail "manifest lists a missing template file: ${entry}"
	if [[ ! -f "${destination}" ]]; then
		printf '  + %s (absent in target)\n' "${entry}"
		return 1
	fi
	cmp -s "${source}" "${destination}" && return 0
	printf '  M %s\n' "${entry}"
	return 1
}

# Copy one manifest entry, mirroring deletions inside directories.
apply_entry() {
	local repo="$1" entry="$2" source="${template}/$2"
	if [[ "${entry}" == */ ]]; then
		mkdir -p "${repo}/${entry%/}"
		rsync -a --delete "${source%/}/" "${repo}/${entry%/}/"
		return
	fi
	mkdir -p "$(dirname -- "${repo}/${entry}")"
	cp -p "${source}" "${repo}/${entry}"
}

# Manifest pathspecs that exist in the target right now. git rejects a pathspec
# matching nothing, and a first sync legitimately introduces paths the target has
# never had. Avoids mapfile: this repository's scripts must run on bash 3.2.
collect_present() {
	local repo="$1" entry
	present=()
	for entry in "${paths[@]}" "${generated_paths[@]}"; do
		if [[ -e "${repo}/${entry%/}" ]]; then present+=("${entry%/}"); fi
	done
	# A trailing false test would return non-zero and `set -e` would end the run.
	return 0
}

# A file where the target keeps a directory (or the reverse) cannot be copied over.
# Report it as a target problem instead of letting `cp` fail mid-mirror.
type_conflicts() {
	local repo="$1" entry conflicts=""
	for entry in "${paths[@]}"; do
		if [[ "${entry}" == */ ]]; then
			if [[ -e "${repo}/${entry%/}" && ! -d "${repo}/${entry%/}" ]]; then
				conflicts+="${entry} (target has a file where the template has a directory) "
			fi
		elif [[ -e "${repo}/${entry}" && ! -f "${repo}/${entry}" ]]; then
			conflicts+="${entry} (target has a directory where the template has a file) "
		fi
	done
	printf '%s' "${conflicts}"
}

# An ignored manifest path can never converge: the mirror writes the file, git
# refuses to track it, and --check reports the same drift forever.
ignored_paths() {
	local repo="$1" entry listing
	listing=$(
		for entry in "${paths[@]}"; do
			if [[ "${entry}" == */ ]]; then
				(cd "${template}" && find "${entry%/}" -type f 2>/dev/null || true)
			else
				printf '%s\n' "${entry}"
			fi
		done | git -C "${repo}" check-ignore --stdin 2>/dev/null || true
	)
	printf '%s' "${listing}"
}

# Reject an owned file that carries the target's own identity: after a mirror
# every target would receive the template's module path instead of its own.
assert_no_identity_leak() {
	local repo="$1" module entry hit
	[[ -f "${repo}/go.mod" ]] || return 0
	module=$(awk '$1 == "module" { print $2; exit }' "${repo}/go.mod")
	[[ -n "${module}" ]] || return 0
	for entry in "${paths[@]}"; do
		hit=$(grep -rlF -- "${module}" "${template}/${entry%/}" 2>/dev/null || true)
		[[ -z "${hit}" ]] || fail "owned path ${entry} contains the target module path ${module}; it carries repository-specific content and must leave the manifest"
	done
	return 0
}

drifted_targets=0
synced_targets=0
rejected_targets=0

for target in "${targets[@]}"; do
	repo=$(CDPATH= cd -- "${target}" 2>/dev/null && pwd) || fail "target directory not found: ${target}"
	printf '== %s\n' "${repo}"

	if [[ "${repo}" == "${template}" ]]; then
		printf '   skipped: target is the template itself\n'
		continue
	fi

	drift=0
	report=""
	for entry in "${paths[@]}"; do
		if ! entry_report=$(diff_entry "${repo}" "${entry}"); then
			drift=1
			report+="${entry_report}"$'\n'
		fi
	done

	if ((drift == 0)); then
		printf '   in sync with template %s\n' "${template_revision}"
		continue
	fi

	printf '%s' "${report}"

	if [[ "${mode}" == "check" ]]; then
		drifted_targets=$((drifted_targets + 1))
		continue
	fi

	# Every refusal happens here, before the first write, so a refused target is
	# left exactly as it was found.
	if ! git -C "${repo}" rev-parse --git-dir >/dev/null 2>&1; then
		reject "not a git repository"
		continue
	fi
	if ! git -C "${repo}" symbolic-ref -q HEAD >/dev/null; then
		reject "detached HEAD, so a sync commit would not belong to any branch"
		continue
	fi

	conflicts=$(type_conflicts "${repo}")
	if [[ -n "${conflicts}" ]]; then
		reject "path type conflict: ${conflicts}"
		continue
	fi

	ignored=$(ignored_paths "${repo}")
	if [[ -n "${ignored}" ]]; then
		reject "$(printf '%s\n' "${ignored}" | wc -l | tr -d ' ') manifest path(s) are gitignored here and could never converge; first: $(printf '%s' "${ignored}" | head -1)"
		continue
	fi

	collect_present "${repo}"
	if ((${#present[@]} > 0)); then
		dirty=$(git -C "${repo}" status --porcelain -- "${present[@]}")
		if [[ -n "${dirty}" ]]; then
			printf '   uncommitted changes inside the manifest:\n'
			printf '%s\n' "${dirty}" | sed 's/^/     /'
			reject "the sync would overwrite them. Commit or discard these paths yourself, then sync again"
			continue
		fi
	fi

	assert_no_identity_leak "${repo}"

	for entry in "${paths[@]}"; do apply_entry "${repo}" "${entry}"; done

	if [[ -f "${repo}/scripts/template-sync.sh" ]]; then
		chmod +x "${repo}/scripts/template-sync.sh" "${repo}/scripts/ci/template-owned-purity-check.sh"
	fi
	if [[ -f "${repo}/Makefile" ]] && grep -q '^claude-skills-sync:' "${repo}/Makefile"; then
		if ! make -C "${repo}" --no-print-directory claude-skills-sync; then
			reject "claude-skills-sync failed; the mirror is in the working tree and was not committed"
			continue
		fi
	fi
	printf 'template %s\n' "$(git -C "${template}" rev-parse HEAD)" >"${repo}/.template-sync"

	if [[ "${commit}" == false ]]; then
		printf '   synced into the working tree, not committed\n'
		synced_targets=$((synced_targets + 1))
		continue
	fi

	# The manifest was clean before the mirror ran, so everything staged here was
	# produced by this sync. No work in progress can enter this commit.
	collect_present "${repo}"
	present+=(.template-sync)
	git -C "${repo}" add -A -- "${present[@]}"
	if [[ -n "$(git -C "${repo}" diff --cached --name-only -- "${present[@]}")" ]]; then
		git -C "${repo}" commit -q \
			-m "Sync template-owned instructions to ${template_revision}" \
			-- "${present[@]}"
		printf '   synced and committed at template %s\n' "${template_revision}"
	fi
	synced_targets=$((synced_targets + 1))
done

if [[ "${mode}" == "check" ]]; then
	if ((drifted_targets > 0)); then
		printf 'template-sync: %d target(s) drifted from template %s\n' \
			"${drifted_targets}" "${template_revision}" >&2
		exit 1
	fi
	echo "template-owned instructions are current"
	exit 0
fi

printf 'template-sync: %d target(s) synced to template %s' "${synced_targets}" "${template_revision}"
if ((rejected_targets > 0)); then
	printf ', %d refused\n' "${rejected_targets}"
	exit 1
fi
printf '\n'
