#!/usr/bin/env bash
# Mirror the template-owned instruction surface between this template and the
# repositories derived from it. The manifest `template-owned.paths` owns copied
# content; generated Claude/Qwen skill links and the managed Codex
# agent-registry block are derived from that content and travel with the same
# sync.
#
# The sync never commits work it did not produce. A target whose manifest paths
# hold uncommitted changes is refused, except for a marked service-owned skill
# that the mirror and sync commit both exclude.
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

Uncommitted work outside the manifest and its generated `.claude/skills`,
`.qwen/skills`, and `.codex/config.toml` views is never staged or touched. Changes inside those
owned/generated paths refuse the target before any write, except a skill
directory with a `.service-owned` marker and its matching harness discovery
links. Commit or discard other owned paths yourself, then sync again.
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

script_dir=$(CDPATH='' cd -- "$(dirname -- "$0")" && pwd)
default_template=$(CDPATH='' cd -- "${script_dir}/.." && pwd)

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

template=$(CDPATH='' cd -- "${template}" 2>/dev/null && pwd) || fail "template directory not found"
manifest="${template}/template-owned.paths"
[[ -f "${manifest}" ]] || fail "manifest not found: ${manifest}"

if ((${#targets[@]} == 0)); then
	targets+=("${explicit_repo:-$PWD}")
fi

# shellcheck source=scripts/lib/manifest.sh
source "$(dirname -- "${BASH_SOURCE[0]}")/lib/manifest.sh"
manifest_paths "${manifest}"
((${#paths[@]} > 0)) || fail "manifest lists no paths"

template_pathspecs=()
for entry in "${paths[@]}"; do template_pathspecs+=("${entry%/}"); done

# Symlinks inside an owned path can redirect a read or write outside the
# repository. Check every path component and every nested entry before diffing.
first_manifest_symlink() {
	local root="$1" entry component candidate nested
	local -a components=()

	for entry in "${paths[@]}"; do
		components=()
		IFS='/' read -r -a components <<<"${entry%/}"
		candidate="${root}"
		for component in "${components[@]}"; do
			candidate="${candidate}/${component}"
			if [[ -L "${candidate}" ]]; then
				printf '%s' "${candidate#"${root}/"}"
				return
			fi
		done
		if [[ "${entry}" == */ && -d "${root}/${entry%/}" ]]; then
			nested=$(find "${root}/${entry%/}" -type l -print -quit 2>/dev/null || true)
			if [[ -n "${nested}" ]]; then
				printf '%s' "${nested#"${root}/"}"
				return
			fi
		fi
	done
}

# Tracked paths the sync regenerates from template-owned skills and roles. They
# are absent from the manifest because two are harness discovery views and the
# Codex config keeps repository-specific content outside its managed block.
generated_paths=(.claude/skills .qwen/skills .codex/config.toml)
# Historical generated receipts removed by the sync.
retired_paths=(.template-sync)
# These files carry service-specific decisions and therefore cannot be mirrored,
# but template-owned instructions depend on them as local authorities.
required_repo_owned_paths=(
	docs/repo-architecture.md
	docs/project-structure-and-module-organization.md
	docs/build-test-and-development-commands.md
	docs/ci-cd-production-ready.md
	docs/railway-deployment-profile.md
	test/README.md
)

template_revision=$(git -C "${template}" rev-parse --short HEAD 2>/dev/null || echo "")
source_root="${template}"
source_snapshot=""

cleanup() {
	[[ -z "${source_snapshot}" ]] || rm -rf -- "${source_snapshot}"
}
trap cleanup EXIT

source_symlink=$(first_manifest_symlink "${template}")
[[ -z "${source_symlink}" ]] ||
	fail "template manifest contains symlink ${source_symlink}; owned paths must not redirect outside the repository"

ignored_source=$(git -C "${template}" ls-files --others --ignored --exclude-standard -- "${template_pathspecs[@]}" 2>/dev/null || true)
[[ -z "${ignored_source}" ]] ||
	fail "template manifest contains ignored content; first: $(printf '%s' "${ignored_source}" | head -1)"

# Apply only a committed template snapshot. The sync commit names that revision,
# so uncommitted source edits must not enter the mirrored content.
if [[ "${mode}" == "apply" ]]; then
	[[ -n "${template_revision}" ]] ||
		fail "template is not a git repository, so no committed source revision is available: ${template}"
	if [[ -n "$(git -C "${template}" status --porcelain -- "${template_pathspecs[@]}")" ]]; then
		printf 'template-sync: the template has uncommitted changes inside its own manifest:\n' >&2
		git -C "${template}" status --porcelain -- "${template_pathspecs[@]}" | sed 's/^/  /' >&2
		fail "commit them first so targets receive one reviewable template revision"
	fi
	source_snapshot=$(mktemp -d "${TMPDIR:-/tmp}/template-sync.XXXXXX")
	git -C "${template}" archive HEAD -- "${template_pathspecs[@]}" |
		tar -xf - -C "${source_snapshot}"
	source_root="${source_snapshot}"
	source_symlink=$(first_manifest_symlink "${source_root}")
	[[ -z "${source_symlink}" ]] ||
		fail "template revision contains symlink ${source_symlink}; owned paths must not redirect outside the repository"
fi

claude_skills_helper="${source_root}/scripts/claude-skills-sync.sh"
[[ -f "${claude_skills_helper}" ]] ||
	fail "template-owned Claude skill helper is missing: scripts/claude-skills-sync.sh"
qwen_skills_helper="${source_root}/scripts/qwen-skills-sync.sh"
[[ -f "${qwen_skills_helper}" ]] ||
	fail "template-owned Qwen skill helper is missing: scripts/qwen-skills-sync.sh"
agent_roles_helper="${source_root}/scripts/agent-roles-sync.sh"
[[ -f "${agent_roles_helper}" ]] ||
	fail "template-owned role helper is missing: scripts/agent-roles-sync.sh"
codex_agents_helper="${source_root}/scripts/codex-agents-sync.sh"
[[ -f "${codex_agents_helper}" ]] ||
	fail "template-owned Codex agent helper is missing: scripts/codex-agents-sync.sh"

# Compare one manifest entry. Prints a human-readable delta, returns 1 on drift.
diff_entry() {
	local repo="$1" entry="$2" source="${source_root}/$2" destination
	destination="${repo}/${entry%/}"
	if [[ "${entry}" == */ ]]; then
		[[ -d "${source}" ]] || fail "manifest lists a missing template directory: ${entry}"
		if [[ ! -d "${destination}" ]]; then
			printf '  + %s (absent in target)\n' "${entry}"
			return 1
		fi
		local delta status=0
		if [[ "${entry}" == ".agents/skills/" ]]; then
			local name
			local -a excludes=()
			while IFS= read -r name; do
				[[ -n "${name}" ]] && excludes+=(--exclude "/${name}/")
			done < <(service_owned_skill_names "${repo}")
			if ((${#excludes[@]} > 0)); then
				delta=$(rsync -rni --checksum --no-times --delete --prune-empty-dirs \
					"${excludes[@]}" "${source%/}/" "${destination%/}/" 2>&1) || status=$?
			else
				delta=$(rsync -rni --checksum --no-times --delete --prune-empty-dirs \
					"${source%/}/" "${destination%/}/" 2>&1) || status=$?
			fi
			((status == 0)) || fail "could not compare ${entry}: ${delta}"
			delta=$(printf '%s\n' "${delta}" | sed '/^\.f\.\.T\.\.\.\. /d; /^$/d')
			[[ -z "${delta}" ]] && return 0
			printf '%s\n' "${delta}" | sed 's|^|  |'
			return 1
		fi
		delta=$(git diff --no-ext-diff --no-index --name-status -- "${source%/}" "${destination}" 2>&1) || status=$?
		((status == 0)) && return 0
		((status == 1)) || fail "could not compare ${entry}: ${delta}"
		printf '%s\n' "${delta}" | sed "s|${source_root}/||g; s|${repo}/||g; s|^|  |"
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
	local repo="$1" entry="$2" source="${source_root}/$2"
	if [[ "${entry}" == */ ]]; then
		local name
		local -a excludes=()
		if [[ "${entry}" == ".agents/skills/" ]]; then
			while IFS= read -r name; do
				[[ -n "${name}" ]] && excludes+=(--exclude "/${name}/")
			done < <(service_owned_skill_names "${repo}")
		fi
		mkdir -p "${repo}/${entry%/}"
		if ((${#excludes[@]} > 0)); then
			rsync -a --checksum --no-times --delete --prune-empty-dirs \
				"${excludes[@]}" "${source%/}/" "${repo}/${entry%/}/"
		else
			rsync -a --checksum --no-times --delete --prune-empty-dirs \
				"${source%/}/" "${repo}/${entry%/}/"
		fi
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
	for entry in "${paths[@]}" "${generated_paths[@]}" "${retired_paths[@]}"; do
		if [[ -e "${repo}/${entry%/}" || -L "${repo}/${entry%/}" ]]; then present+=("${entry%/}"); fi
	done
	# A trailing false test would return non-zero and `set -e` would end the run.
	return 0
}

service_owned_skill_names() {
	local repo="$1" marker
	local -a markers=()
	[[ -d "${repo}/.agents/skills" ]] || return 0
	shopt -s nullglob
	markers=("${repo}/.agents/skills/"*/.service-owned)
	shopt -u nullglob
	((${#markers[@]} > 0)) || return 0
	for marker in "${markers[@]}"; do
		basename -- "$(dirname -- "${marker}")"
	done
}

service_owned_skill_issue() {
	local repo="$1" marker name skill_dir
	local -a markers=()
	[[ -d "${repo}/.agents/skills" ]] || return 0
	shopt -s nullglob
	markers=("${repo}/.agents/skills/"*/.service-owned)
	shopt -u nullglob
	((${#markers[@]} > 0)) || return 0
	for marker in "${markers[@]}"; do
		skill_dir=$(dirname -- "${marker}")
		name=$(basename -- "${skill_dir}")
		if [[ ! -f "${marker}" || -L "${marker}" ]]; then
			printf '%s' "service-owned skill ${name} has an invalid .service-owned marker"
			return
		fi
		if [[ ! -f "${skill_dir}/SKILL.md" || -L "${skill_dir}/SKILL.md" ]]; then
			printf '%s' "service-owned skill ${name} has no real SKILL.md"
			return
		fi
		if [[ -e "${source_root}/.agents/skills/${name}" ]]; then
			printf '%s' "skill ${name} is marked service-owned but is also template-owned"
			return
		fi
	done
}

collect_service_skill_exclusions() {
	local repo="$1" name
	service_skill_exclusions=()
	while IFS= read -r name; do
		[[ -n "${name}" ]] && service_skill_exclusions+=(
			":(exclude).agents/skills/${name}"
			":(exclude).agents/skills/${name}/**"
			":(exclude).claude/skills/${name}"
			":(exclude).qwen/skills/${name}"
		)
	done < <(service_owned_skill_names "${repo}")
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

missing_required_repo_owned() {
	local repo="$1" entry
	for entry in "${required_repo_owned_paths[@]}"; do
		if [[ ! -f "${repo}/${entry}" || -L "${repo}/${entry}" ]]; then
			printf '%s\n' "${entry}"
		fi
	done
}

# An ignored manifest path can never converge: the mirror writes the file, git
# refuses to track it, and --check reports the same drift forever.
ignored_paths() {
	local repo="$1" entry listing
	listing=$(
		for entry in "${paths[@]}"; do
			if [[ "${entry}" == */ ]]; then
				(cd "${source_root}" && find "${entry%/}" -type f 2>/dev/null || true)
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
		hit=$(grep -rlF -- "${module}" "${source_root}/${entry%/}" 2>/dev/null || true)
		[[ -z "${hit}" ]] || fail "owned path ${entry} contains the target module path ${module}; it carries repository-specific content and must leave the manifest"
	done
	return 0
}

drifted_targets=0
synced_targets=0
rejected_targets=0

for target in "${targets[@]}"; do
	repo=$(CDPATH='' cd -- "${target}" 2>/dev/null && pwd) || fail "target directory not found: ${target}"
	printf '== %s\n' "${repo}"

	if [[ "${repo}" == "${template}" ]]; then
		printf '   skipped: target is the template itself\n'
		continue
	fi

	target_symlink=$(first_manifest_symlink "${repo}")
	if [[ -n "${target_symlink}" ]]; then
		if [[ "${mode}" == "check" ]]; then
			printf '  ! manifest symlink: %s\n' "${target_symlink}"
			drifted_targets=$((drifted_targets + 1))
		else
			reject "manifest contains symlink ${target_symlink}; refusing a copy that could leave the repository"
		fi
		continue
	fi

	target_ignored=""
	if git -C "${repo}" rev-parse --git-dir >/dev/null 2>&1; then
		target_ignored=$(git -C "${repo}" ls-files --others --ignored --exclude-standard -- "${template_pathspecs[@]}" 2>/dev/null || true)
	fi
	if [[ -n "${target_ignored}" ]]; then
		if [[ "${mode}" == "check" ]]; then
			printf '  ! ignored manifest content; first: %s\n' "$(printf '%s' "${target_ignored}" | head -1)"
			drifted_targets=$((drifted_targets + 1))
		else
			reject "ignored content inside the manifest could be deleted; first: $(printf '%s' "${target_ignored}" | head -1)"
		fi
		continue
	fi

	missing_repo_owned=$(missing_required_repo_owned "${repo}")
	if [[ -n "${missing_repo_owned}" ]]; then
		if [[ "${mode}" == "check" ]]; then
			while IFS= read -r entry; do
				printf '  ! required repository-owned instruction is missing: %s\n' "${entry}"
			done <<<"${missing_repo_owned}"
			drifted_targets=$((drifted_targets + 1))
		else
			reject "required repository-owned instruction is missing: $(printf '%s' "${missing_repo_owned}" | head -1)"
		fi
		continue
	fi
	service_skill_problem=$(service_owned_skill_issue "${repo}")
	if [[ -n "${service_skill_problem}" ]]; then
		if [[ "${mode}" == "check" ]]; then
			printf '  ! %s\n' "${service_skill_problem}"
			drifted_targets=$((drifted_targets + 1))
		else
			reject "${service_skill_problem}"
		fi
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
	for entry in "${retired_paths[@]}"; do
		if [[ -e "${repo}/${entry}" || -L "${repo}/${entry}" ]]; then
			drift=1
			report+="  - ${entry} (retired)"$'\n'
		fi
	done
	if ! generated_report=$(bash "${claude_skills_helper}" --check --repo "${repo}" 2>&1); then
		drift=1
		while IFS= read -r line; do
			[[ -n "${line}" ]] && report+="  ! ${line}"$'\n'
		done <<<"${generated_report}"
	fi
	if ! generated_report=$(bash "${qwen_skills_helper}" --check --repo "${repo}" 2>&1); then
		drift=1
		while IFS= read -r line; do
			[[ -n "${line}" ]] && report+="  ! ${line}"$'\n'
		done <<<"${generated_report}"
	fi
	if ! generated_report=$(bash "${agent_roles_helper}" --check --repo "${repo}" 2>&1); then
		drift=1
		while IFS= read -r line; do
			[[ -n "${line}" ]] && report+="  ! ${line}"$'\n'
		done <<<"${generated_report}"
	fi
	if ! generated_report=$(bash "${codex_agents_helper}" --check --repo "${repo}" 2>&1); then
		drift=1
		while IFS= read -r line; do
			[[ -n "${line}" ]] && report+="  ! ${line}"$'\n'
		done <<<"${generated_report}"
	fi

	if ((drift == 0)); then
		printf '   in sync with template %s\n' "${template_revision}"
		continue
	fi

	printf '%s' "${report}"

	if [[ "${mode}" == "check" ]]; then
		drifted_targets=$((drifted_targets + 1))
		continue
	fi

	# Resolve every deterministic target refusal before the first write. An
	# unexpected write or helper failure can leave only sync-produced dirt.
	if ! git -C "${repo}" rev-parse --git-dir >/dev/null 2>&1; then
		reject "not a git repository"
		continue
	fi
	if [[ "${commit}" == true ]] && ! git -C "${repo}" symbolic-ref -q HEAD >/dev/null; then
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
	collect_service_skill_exclusions "${repo}"
	if ((${#present[@]} > 0)); then
		if ((${#service_skill_exclusions[@]} > 0)); then
			dirty=$(git -C "${repo}" status --porcelain -- "${present[@]}" "${service_skill_exclusions[@]}")
		else
			dirty=$(git -C "${repo}" status --porcelain -- "${present[@]}")
		fi
		if [[ -n "${dirty}" ]]; then
			printf '   uncommitted changes inside the manifest:\n'
			printf '%s\n' "${dirty}" | sed 's/^/     /'
			reject "the sync would overwrite them. Commit or discard these paths yourself, then sync again"
			continue
		fi
	fi

	assert_no_identity_leak "${repo}"
	if ! preflight_report=$(bash "${claude_skills_helper}" --preflight --repo "${repo}" 2>&1); then
		[[ -z "${preflight_report}" ]] ||
			printf '%s\n' "${preflight_report}" | sed 's/^/   /'
		reject "generated Claude skill links cannot be rebuilt safely"
		continue
	fi
	if ! preflight_report=$(bash "${qwen_skills_helper}" --preflight --repo "${repo}" 2>&1); then
		[[ -z "${preflight_report}" ]] ||
			printf '%s\n' "${preflight_report}" | sed 's/^/   /'
		reject "generated Qwen skill links cannot be rebuilt safely"
		continue
	fi
	if ! preflight_report=$(bash "${agent_roles_helper}" --preflight --repo "${repo}" 2>&1); then
		[[ -z "${preflight_report}" ]] ||
			printf '%s\n' "${preflight_report}" | sed 's/^/   /'
		reject "generated agent roles cannot be rebuilt safely"
		continue
	fi
	if ! preflight_report=$(bash "${codex_agents_helper}" --preflight --repo "${repo}" 2>&1); then
		[[ -z "${preflight_report}" ]] ||
			printf '%s\n' "${preflight_report}" | sed 's/^/   /'
		reject "generated Codex project config cannot be rebuilt safely"
		continue
	fi

	for entry in "${paths[@]}"; do apply_entry "${repo}" "${entry}"; done
	for entry in "${retired_paths[@]}"; do rm -f -- "${repo}/${entry}"; done

	if [[ -f "${repo}/scripts/template-sync.sh" ]]; then
		chmod +x \
			"${repo}/scripts/template-sync.sh" \
			"${repo}/scripts/agent-roles-sync.sh" \
			"${repo}/scripts/harness-skills-sync.sh" \
			"${repo}/scripts/claude-skills-sync.sh" \
			"${repo}/scripts/qwen-skills-sync.sh" \
			"${repo}/scripts/codex-agents-sync.sh" \
			"${repo}/scripts/ci/claude-skills-check.sh" \
			"${repo}/scripts/ci/qwen-skills-check.sh" \
			"${repo}/scripts/ci/template-owned-purity-check.sh"
	fi
	if ! bash "${repo}/scripts/claude-skills-sync.sh" --apply --repo "${repo}"; then
		reject "Claude skill link rebuild failed; the mirror is in the working tree and was not committed"
		continue
	fi
	if ! bash "${repo}/scripts/qwen-skills-sync.sh" --apply --repo "${repo}"; then
		reject "Qwen skill link rebuild failed; the mirror is in the working tree and was not committed"
		continue
	fi
	if ! bash "${repo}/scripts/agent-roles-sync.sh" --apply --repo "${repo}"; then
		reject "agent role rebuild failed; the mirror is in the working tree and was not committed"
		continue
	fi
	if ! bash "${repo}/scripts/codex-agents-sync.sh" --apply --repo "${repo}"; then
		reject "Codex agent registry rebuild failed; the mirror is in the working tree and was not committed"
		continue
	fi
	if [[ "${commit}" == false ]]; then
		printf '   synced into the working tree, not committed\n'
		synced_targets=$((synced_targets + 1))
		continue
	fi

	# The manifest was clean before the mirror ran, so everything staged here was
	# produced by this sync. No work in progress can enter this commit.
	collect_present "${repo}"
	collect_service_skill_exclusions "${repo}"
	commit_paths=("${present[@]}")
	if ((${#service_skill_exclusions[@]} > 0)); then
		commit_paths+=("${service_skill_exclusions[@]}")
	fi
	for entry in "${retired_paths[@]}"; do
		if git -C "${repo}" ls-files --error-unmatch "${entry}" >/dev/null 2>&1; then
			commit_paths+=("${entry}")
		fi
	done
	git -C "${repo}" add -A -- "${commit_paths[@]}"
	if [[ -n "$(git -C "${repo}" diff --cached --name-only -- "${commit_paths[@]}")" ]]; then
		git -C "${repo}" commit -q \
			-m "Sync template-owned instructions to ${template_revision}" \
			-- "${commit_paths[@]}"
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
