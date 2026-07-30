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

# Initialization profile markers encode target-specific retained or removed
# content. Verbatim mirroring cannot preserve that choice, so such content must
# live in a repository-owned/profile-owned file outside the manifest.
for entry in "${paths[@]}"; do
	profile_marker=$(
		grep -RIlE -- '<!--[[:space:]]*profile:[^:]+:(start|end)[[:space:]]*-->' \
			"${entry%/}" 2>/dev/null | head -1 || true
	)
	if [[ -n "${profile_marker}" ]]; then
		fail "${profile_marker} contains an initialization profile marker; verbatim template-owned paths must be valid for every derived repository"
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
for required in \
	"${manifest}" \
	scripts/template-sync.sh \
	scripts/claude-skills-sync.sh \
	scripts/codex-agents-sync.sh \
	scripts/ci/claude-skills-check.sh \
	scripts/ci/template-owned-purity-check.sh; do
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

# Instruction transport is part of template purity: a correctly written owner
# that the harness cannot discover is indistinguishable from a missing rule.
grep -Fxq '@AGENTS.md' CLAUDE.md ||
	fail "CLAUDE.md must import AGENTS.md with the native @AGENTS.md directive"
if grep -Fxq '@AGENTS.md' QWEN.md; then
	fail "QWEN.md must not re-import AGENTS.md; Qwen loads it natively"
fi

skill_metadata_bytes=$(
	for skill_file in .agents/skills/*/SKILL.md; do
		sed -n -e 's/^name: //p' -e 's/^description: //p' "${skill_file}"
	done | wc -c | tr -d ' '
)
if ((skill_metadata_bytes > 8000)); then
	fail "repo skill name/description metadata is ${skill_metadata_bytes} bytes; keep it at or below 8000 so discovery descriptions remain lean"
fi
for skill_file in .agents/skills/*/SKILL.md; do
	grep -q '^name: [^[:space:]]' "${skill_file}" ||
		fail "${skill_file} has no discoverable name"
	grep -q '^description: .[^[:space:]]' "${skill_file}" ||
		fail "${skill_file} has no discoverable description"
done

for role_file in .codex/agents/*.toml; do
	role=$(basename "${role_file}" .toml)
	grep -Fxq "[agents.${role}]" .codex/config.toml ||
		fail "${role_file} exists but .codex/config.toml does not register agents.${role}"
	[[ -f ".claude/agents/${role}.md" ]] ||
		fail "${role_file} has no Claude role mirror"
	[[ -f ".qwen/agents/${role}.md" ]] ||
		fail "${role_file} has no Qwen role mirror"
done
for role_file in .claude/agents/*.md .qwen/agents/*.md; do
	role=$(basename "${role_file}" .md)
	[[ "${role}" == worker-* ]] && continue
	[[ -f ".codex/agents/${role}.toml" ]] ||
		fail "${role_file} has no Codex role mirror"
done
while IFS= read -r role; do
	[[ -f ".codex/agents/${role}.toml" ]] ||
		fail ".codex/config.toml registers agents.${role}, but its role file is missing"
done < <(sed -n 's/^\[agents\.\([^]]*\)\]$/\1/p' .codex/config.toml)
if ! codex_registry_report=$(bash scripts/codex-agents-sync.sh --check --repo . 2>&1); then
	fail "Codex role registry is not the generated current block: ${codex_registry_report}"
fi

if ((failed != 0)); then
	exit 1
fi

template_sync_behavior_check() (
	local fixture template target target_with_directory target_with_link outside sync_script check_output
	fixture=$(mktemp -d "${TMPDIR:-/tmp}/template-sync-check.XXXXXX")
	trap 'rm -rf -- "${fixture}"' EXIT
	template="${fixture}/template"
	target="${fixture}/target"
	target_with_directory="${fixture}/target-with-directory"
	target_with_link="${fixture}/target-with-link"
	outside="${fixture}/outside"
	sync_script="$(pwd)/scripts/template-sync.sh"

	mkdir -p \
		"${template}/owned" \
		"${template}/.agents/skills/fixture-one" \
		"${template}/.codex/agents" \
		"${template}/scripts/ci" \
		"${outside}"
	printf '%s\n' \
		'owned/' \
		'.agents/skills/' \
		'.codex/agents/' \
		'scripts/claude-skills-sync.sh' \
		'scripts/codex-agents-sync.sh' \
		'scripts/ci/claude-skills-check.sh' \
		>"${template}/template-owned.paths"
	printf 'v1\n' >"${template}/owned/version"
	printf '%s\n' '---' 'name: fixture-one' 'description: fixture' '---' \
		>"${template}/.agents/skills/fixture-one/SKILL.md"
	printf '%s\n' 'name = "fixture-agent"' 'description = "fixture"' \
		>"${template}/.codex/agents/fixture-agent.toml"
	printf '%s\n' '[agents]' 'max_depth = 1' '' '[fixture]' 'retained = true' \
		>"${template}/.codex/config.toml"
	cp scripts/claude-skills-sync.sh "${template}/scripts/claude-skills-sync.sh"
	cp scripts/codex-agents-sync.sh "${template}/scripts/codex-agents-sync.sh"
	cp scripts/ci/claude-skills-check.sh "${template}/scripts/ci/claude-skills-check.sh"
	git -C "${template}" init -q
	git -C "${template}" add \
		template-owned.paths \
		owned/version \
		.agents/skills/fixture-one/SKILL.md \
		.codex/agents/fixture-agent.toml \
		.codex/config.toml \
		scripts/claude-skills-sync.sh \
		scripts/codex-agents-sync.sh \
		scripts/ci/claude-skills-check.sh
	git -C "${template}" -c user.name=template-sync-check -c user.email=template-sync-check@example.invalid commit -qm v1
	git clone -q "${template}" "${target}"
	git -C "${target}" config user.name template-sync-check
	git -C "${target}" config user.email template-sync-check@example.invalid
	printf 'template legacy\n' >"${target}/.template-sync"
	git -C "${target}" add .template-sync
	git -C "${target}" -c user.name=template-sync-check -c user.email=template-sync-check@example.invalid commit -qm legacy-receipt

	printf 'v2\n' >"${template}/owned/version"
	mkdir -p "${template}/.agents/skills/fixture-two"
	printf '%s\n' '---' 'name: fixture-two' 'description: fixture' '---' \
		>"${template}/.agents/skills/fixture-two/SKILL.md"
	git -C "${template}" add owned/version .agents/skills/fixture-two/SKILL.md
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

	bash "${sync_script}" --apply --from "${template}" --repo "${target}" >/dev/null
	grep -Fxq v2 "${target}/owned/version" || {
		echo "template-owned purity: committed sync did not apply" >&2
		return 1
	}
	[[ ! -e "${target}/.template-sync" ]] || {
		echo "template-owned purity: sync retained the retired .template-sync receipt" >&2
		return 1
	}
	for skill in fixture-one fixture-two; do
		link="${target}/.claude/skills/${skill}"
		[[ -L "${link}" ]] || {
			echo "template-owned purity: sync omitted generated Claude link ${skill}" >&2
			return 1
		}
		[[ "$(readlink "${link}")" == "../../.agents/skills/${skill}" ]] || {
			echo "template-owned purity: sync generated the wrong Claude link for ${skill}" >&2
			return 1
		}
	done
	bash "${target}/scripts/ci/claude-skills-check.sh" >/dev/null || {
		echo "template-owned purity: synced Claude link checker rejected generated links" >&2
		return 1
	}
	bash "${target}/scripts/codex-agents-sync.sh" --check --repo "${target}" >/dev/null || {
		echo "template-owned purity: synced Codex registry checker rejected generated roles" >&2
		return 1
	}
	grep -Fxq 'retained = true' "${target}/.codex/config.toml" || {
		echo "template-owned purity: Codex registry sync changed repository-specific config" >&2
		return 1
	}
	rm "${target}/.claude/skills/fixture-two"
	if bash "${sync_script}" --check --from "${template}" --repo "${target}" >/dev/null 2>&1; then
		echo "template-owned purity: sync check missed a generated Claude link gap" >&2
		return 1
	fi
	bash "${target}/scripts/claude-skills-sync.sh" --apply --repo "${target}" >/dev/null
	git -C "${target}" show --format= --name-status HEAD |
		grep -Eq '^D[[:space:]]+\.template-sync$' || {
		echo "template-owned purity: sync commit omitted the retired .template-sync receipt" >&2
		return 1
	}

	printf 'v3\n' >"${template}/owned/version"
	git -C "${template}" add owned/version
	git -C "${template}" -c user.name=template-sync-check -c user.email=template-sync-check@example.invalid commit -qm v3

	git -C "${target}" checkout -q --detach
	bash "${sync_script}" --apply --no-commit --from "${template}" --repo "${target}" >/dev/null
	grep -Fxq v3 "${target}/owned/version" || {
		echo "template-owned purity: detached --no-commit sync did not apply" >&2
		return 1
	}
	mkdir "${template}/owned/empty"
	if ! check_output=$(bash "${sync_script}" --check --from "${template}" --repo "${target}" 2>&1); then
		printf '%s\n' "${check_output}" >&2
		return 1
	fi

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

	git clone -q "${template}" "${target_with_directory}"
	git -C "${target_with_directory}" config user.name template-sync-check
	git -C "${target_with_directory}" config user.email template-sync-check@example.invalid
	mkdir -p "${target_with_directory}/.claude/skills/manual"
	printf 'only copy\n' >"${target_with_directory}/.claude/skills/manual/SKILL.md"
	git -C "${target_with_directory}" add .claude/skills/manual/SKILL.md
	git -C "${target_with_directory}" commit -qm manual-skill-directory
	printf 'v4\n' >"${template}/owned/version"
	git -C "${template}" add owned/version
	git -C "${template}" -c user.name=template-sync-check -c user.email=template-sync-check@example.invalid commit -qm v4
	if bash "${sync_script}" --apply --no-commit --from "${template}" --repo "${target_with_directory}" >/dev/null 2>&1; then
		echo "template-owned purity: sync replaced a real Claude skill directory" >&2
		return 1
	fi
	grep -Fxq v3 "${target_with_directory}/owned/version" || {
		echo "template-owned purity: Claude link preflight failed after manifest writes" >&2
		return 1
	}
	grep -Fxq 'only copy' "${target_with_directory}/.claude/skills/manual/SKILL.md" || {
		echo "template-owned purity: Claude link preflight changed a real skill directory" >&2
		return 1
	}
)

template_sync_behavior_check
echo "template-owned manifest and sync behavior are safe"
