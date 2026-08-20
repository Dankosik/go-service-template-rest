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

# shellcheck source=scripts/lib/manifest.sh
source "$(dirname -- "${BASH_SOURCE[0]}")/../lib/manifest.sh"
manifest_paths "${manifest}"

((${#paths[@]} > 0)) || fail "${manifest} lists no paths"

contains_path() {
	local expected="$1"
	local entry

	for entry in "${paths[@]}"; do
		[[ "${entry}" == "${expected}" ]] && return 0
	done
	return 1
}

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
	scripts/agent-roles-sync.sh \
	scripts/claude-skills-sync.sh \
	scripts/codex-agents-sync.sh \
	.agents/codex-project.toml \
	scripts/lib/manifest.sh \
	scripts/lib/sync-cli.sh \
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
	test/README.md
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
service_marker=$(find .agents/skills -mindepth 2 -maxdepth 2 -name .service-owned -print -quit 2>/dev/null || true)
[[ -z "${service_marker}" ]] ||
	fail "${service_marker} marks a template skill as service-owned; choose one owner"

if ! role_report=$(bash scripts/agent-roles-sync.sh --check --repo . 2>&1); then
	fail "harness role carriers are not generated from .agents/roles: ${role_report}"
fi

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
	fail "Codex project runtime or role registry is stale: ${codex_registry_report}"
fi

if ((failed != 0)); then
	exit 1
fi

template_sync_behavior_check() (
	local fixture template target target_dirty_local target_empty_claude target_invalid_codex target_missing_codex_source target_nonportable_codex target_secret_codex target_missing_owner target_without_local_skill target_with_directory target_with_link outside sync_script check_output failure_output
	fixture=$(mktemp -d "${TMPDIR:-/tmp}/template-sync-check.XXXXXX")
	trap 'rm -rf -- "${fixture}"' EXIT
	template="${fixture}/template"
	target="${fixture}/target"
	target_dirty_local="${fixture}/target-dirty-local"
	target_empty_claude="${fixture}/target-empty-claude"
	target_invalid_codex="${fixture}/target-invalid-codex"
	target_missing_codex_source="${fixture}/target-missing-codex-source"
	target_nonportable_codex="${fixture}/target-nonportable-codex"
	target_secret_codex="${fixture}/target-secret-codex"
	target_missing_owner="${fixture}/target-missing-owner"
	target_without_local_skill="${fixture}/target-without-local-skill"
	target_with_directory="${fixture}/target-with-directory"
	target_with_link="${fixture}/target-with-link"
	outside="${fixture}/outside"
	sync_script="$(pwd)/scripts/template-sync.sh"

	mkdir -p \
		"${template}/owned" \
		"${template}/.agents/role-classes" \
		"${template}/.agents/roles" \
		"${template}/.agents/skills/fixture-one" \
		"${template}/.claude/agents" \
		"${template}/.codex/agents" \
		"${template}/.qwen/agents" \
		"${template}/docs" \
		"${template}/docs/validation" \
		"${template}/scripts/ci" \
		"${template}/scripts/lib" \
		"${template}/test" \
		"${outside}"
	printf '%s\n' \
		'owned/' \
		'.agents/role-classes/' \
		'.agents/codex-project.toml' \
		'.agents/roles/' \
		'.agents/skills/' \
		'.claude/agents/' \
		'.codex/agents/' \
		'.qwen/agents/' \
		'docs/validation/' \
		'scripts/agent-roles-sync.sh' \
		'scripts/claude-skills-sync.sh' \
		'scripts/codex-agents-sync.sh' \
		'scripts/lib/sync-cli.sh' \
		'scripts/ci/claude-skills-check.sh' \
		>"${template}/template-owned.paths"
	printf 'v1\n' >"${template}/owned/version"
	printf '%s\n' '---' 'name: fixture-one' 'description: fixture' \
		'metadata:' '  invocation: model' '  kind: method' '---' \
		>"${template}/.agents/skills/fixture-one/SKILL.md"
	printf '%s\n' \
		"Apply \`docs/spec-first-workflow/shared/delegation.md\`." \
		>"${template}/.agents/role-classes/read-only-specialist.md"
	printf '%s\n' 'This lane is read-only.' \
		>"${template}/.agents/role-classes/read-only-specialist-fallback.md"
	printf '%s\n' \
		'name = "fixture-agent"' \
		'description = "fixture"' \
		'class = "read-only-specialist"' \
		'claude_model = "sonnet"' \
		'output_schema = "lane-result-v1"' \
		'' \
		'instructions = """' \
		'Own fixture evidence.' \
		'"""' \
		>"${template}/.agents/roles/fixture-agent.toml"
	printf '%s\n' '[agents]' 'max_depth = 4' \
		>"${template}/.codex/config.toml"
	cp .agents/codex-project.toml "${template}/.agents/codex-project.toml"
	cp docs/validation/instructions.md "${template}/docs/validation/instructions.md"
	for repo_owned in \
		docs/repo-architecture.md \
		docs/project-structure-and-module-organization.md \
		docs/build-test-and-development-commands.md \
		docs/ci-cd-production-ready.md \
		docs/railway-deployment-profile.md \
		test/README.md; do
		printf 'fixture repository owner\n' >"${template}/${repo_owned}"
	done
	cp scripts/agent-roles-sync.sh "${template}/scripts/agent-roles-sync.sh"
	cp scripts/claude-skills-sync.sh "${template}/scripts/claude-skills-sync.sh"
	cp scripts/codex-agents-sync.sh "${template}/scripts/codex-agents-sync.sh"
	# The synced sync scripts are executed from the target below, so the library
	# they source has to travel with them exactly as the manifest makes it.
	cp scripts/lib/sync-cli.sh "${template}/scripts/lib/sync-cli.sh"
	cp scripts/ci/claude-skills-check.sh "${template}/scripts/ci/claude-skills-check.sh"
	bash "${template}/scripts/agent-roles-sync.sh" --apply --repo "${template}" >/dev/null
	bash "${template}/scripts/codex-agents-sync.sh" --apply --repo "${template}" >/dev/null
	git -C "${template}" init -q
	git -C "${template}" add \
		template-owned.paths \
		owned/version \
		.agents/codex-project.toml \
		.agents/role-classes \
		.agents/roles \
		.agents/skills/fixture-one/SKILL.md \
		.claude/agents/fixture-agent.md \
		.codex/agents/fixture-agent.toml \
		.qwen/agents/fixture-agent.md \
		.codex/config.toml \
		docs/validation/instructions.md \
		docs/repo-architecture.md \
		docs/project-structure-and-module-organization.md \
		docs/build-test-and-development-commands.md \
		docs/ci-cd-production-ready.md \
		docs/railway-deployment-profile.md \
		scripts/agent-roles-sync.sh \
		scripts/claude-skills-sync.sh \
		scripts/codex-agents-sync.sh \
		scripts/lib/sync-cli.sh \
		scripts/ci/claude-skills-check.sh \
		test/README.md
	git -C "${template}" -c user.name=template-sync-check -c user.email=template-sync-check@example.invalid commit -qm v1
	git clone -q "${template}" "${target}"
	git clone -q "${template}" "${target_empty_claude}"
	git clone -q "${template}" "${target_without_local_skill}"
	git -C "${target}" config user.name template-sync-check
	git -C "${target}" config user.email template-sync-check@example.invalid
	mkdir -p "${target}/.agents/skills/service-local"
	printf '%s\n' '---' 'name: service-local' 'description: local fixture' \
		'metadata:' '  invocation: model' '  kind: method' '---' \
		>"${target}/.agents/skills/service-local/SKILL.md"
	printf 'owned by fixture service\n' >"${target}/.agents/skills/service-local/.service-owned"
	git -C "${target}" add .agents/skills/service-local
	git -C "${target}" commit -qm service-owned-skill
	printf 'template legacy\n' >"${target}/.template-sync"
	git -C "${target}" add .template-sync
	git -C "${target}" -c user.name=template-sync-check -c user.email=template-sync-check@example.invalid commit -qm legacy-receipt
	git clone -q "${target}" "${target_dirty_local}"
	git -C "${target_dirty_local}" config user.name template-sync-check
	git -C "${target_dirty_local}" config user.email template-sync-check@example.invalid

	printf 'v2\n' >"${template}/owned/version"
	mkdir -p "${template}/.agents/skills/fixture-two"
	printf '%s\n' '---' 'name: fixture-two' 'description: fixture' \
		'metadata:' '  invocation: model' '  kind: method' '---' \
		>"${template}/.agents/skills/fixture-two/SKILL.md"
	git -C "${template}" add owned/version .agents/skills/fixture-two/SKILL.md
	git -C "${template}" -c user.name=template-sync-check -c user.email=template-sync-check@example.invalid commit -qm v2

	mkdir -p "${target_empty_claude}/.claude/skills"
	if failure_output=$(bash "${target_empty_claude}/scripts/claude-skills-sync.sh" --check --repo "${target_empty_claude}" 2>&1); then
		echo "template-owned purity: Claude check accepted an empty generated view" >&2
		return 1
	fi
	grep -Fq 'claude skills: .claude/skills/fixture-one is missing' <<<"${failure_output}" || {
		echo "template-owned purity: Claude check rejected an empty generated view for the wrong reason" >&2
		return 1
	}
	bash "${target_empty_claude}/scripts/claude-skills-sync.sh" --apply --repo "${target_empty_claude}" >/dev/null
	[[ -L "${target_empty_claude}/.claude/skills/fixture-one" ]] || {
		echo "template-owned purity: Claude sync did not repair an empty generated view" >&2
		return 1
	}
	bash "${sync_script}" --apply --no-commit --from "${template}" --repo "${target_without_local_skill}" >/dev/null
	grep -Fxq v2 "${target_without_local_skill}/owned/version" || {
		echo "template-owned purity: sync failed for a target without service-owned skills" >&2
		return 1
	}

	printf 'dirty local work\n' >>"${target_dirty_local}/.agents/skills/service-local/SKILL.md"
	git -C "${target_dirty_local}" add .agents/skills/service-local/SKILL.md
	mkdir -p "${target_dirty_local}/.agents/skills/untracked-local"
	printf '%s\n' '---' 'name: untracked-local' 'description: untracked fixture' \
		'metadata:' '  invocation: model' '  kind: method' '---' \
		>"${target_dirty_local}/.agents/skills/untracked-local/SKILL.md"
	printf 'owned by fixture service\n' >"${target_dirty_local}/.agents/skills/untracked-local/.service-owned"
	bash "${sync_script}" --apply --from "${template}" --repo "${target_dirty_local}" >/dev/null
	grep -Fxq v2 "${target_dirty_local}/owned/version" || {
		echo "template-owned purity: dirty service-owned skill blocked the sync" >&2
		return 1
	}
	grep -Fq 'dirty local work' "${target_dirty_local}/.agents/skills/service-local/SKILL.md" || {
		echo "template-owned purity: sync changed a dirty service-owned skill" >&2
		return 1
	}
	git -C "${target_dirty_local}" diff --cached --name-only |
		grep -Fxq '.agents/skills/service-local/SKILL.md' || {
		echo "template-owned purity: sync consumed staged service-owned work" >&2
		return 1
	}
	[[ -f "${target_dirty_local}/.agents/skills/untracked-local/SKILL.md" ]] || {
		echo "template-owned purity: sync deleted an untracked service-owned skill" >&2
		return 1
	}
	[[ -L "${target_dirty_local}/.claude/skills/untracked-local" ]] || {
		echo "template-owned purity: sync omitted an untracked service-owned skill link" >&2
		return 1
	}
	git -C "${target_dirty_local}" status --porcelain -- .agents/skills/untracked-local |
		grep -Fq '?? .agents/skills/untracked-local/' || {
		echo "template-owned purity: sync changed untracked service-owned Git status" >&2
		return 1
	}
	git -C "${target_dirty_local}" status --porcelain -- .claude/skills/untracked-local |
		grep -Fq '?? .claude/skills/untracked-local' || {
		echo "template-owned purity: sync staged an untracked service-owned skill link" >&2
		return 1
	}
	if git -C "${target_dirty_local}" show --format= --name-only HEAD |
		grep -Eq '(^|/)(service-local|untracked-local)(/|$)'; then
		echo "template-owned purity: sync commit included service-owned work" >&2
		return 1
	fi

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
	[[ -f "${target}/.agents/skills/service-local/SKILL.md" ]] || {
		echo "template-owned purity: sync deleted a marked service-owned skill" >&2
		return 1
	}
	[[ -L "${target}/.claude/skills/service-local" ]] || {
		echo "template-owned purity: sync omitted the service-owned skill discovery link" >&2
		return 1
	}
	bash "${target}/scripts/ci/claude-skills-check.sh" >/dev/null || {
		echo "template-owned purity: synced Claude link checker rejected generated links" >&2
		return 1
	}
	bash "${target}/scripts/agent-roles-sync.sh" --check --repo "${target}" >/dev/null || {
		echo "template-owned purity: synced role checker rejected generated carriers" >&2
		return 1
	}
	grep -Fq 'bash scripts/agent-roles-sync.sh --check --repo .' \
		"${target}/docs/validation/instructions.md" || {
		echo "template-owned purity: derived validation does not use the synced role checker" >&2
		return 1
	}
	if grep -Eq "\`make (agent-roles-check|codex-agents-check|claude-skills-check)\`" \
		"${target}/docs/validation/instructions.md"; then
		echo "template-owned purity: derived validation depends on a repository-owned Make target" >&2
		return 1
	fi
	grep -Fq 'template checkout only' "${target}/docs/validation/instructions.md" || {
		echo "template-owned purity: derived validation does not bound the template-only purity gate" >&2
		return 1
	}
	printf '\n# manual drift\n' >>"${target}/.codex/agents/fixture-agent.toml"
	if bash "${target}/scripts/agent-roles-sync.sh" --check --repo "${target}" >/dev/null 2>&1; then
		echo "template-owned purity: role checker accepted a manually changed carrier" >&2
		return 1
	fi
	bash "${target}/scripts/agent-roles-sync.sh" --apply --repo "${target}" >/dev/null
	bash "${target}/scripts/codex-agents-sync.sh" --check --repo "${target}" >/dev/null || {
		echo "template-owned purity: synced Codex project checker rejected runtime or roles" >&2
		return 1
	}
	grep -Fxq 'max_concurrent_threads_per_session = 20' "${target}/.codex/config.toml" || {
		echo "template-owned purity: Codex project sync omitted portable concurrency" >&2
		return 1
	}
	if grep -Eq '^max_depth[[:space:]]*=' "${target}/.codex/config.toml"; then
		echo "template-owned purity: Codex project sync retained unsupported max_depth" >&2
		return 1
	fi
	grep -Fq 'do not emit unchanged heartbeats' "${target}/.codex/config.toml" || {
		echo "template-owned purity: Codex project sync omitted portable wait policy" >&2
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

	git clone -q "${template}" "${target_missing_owner}"
	git -C "${target_missing_owner}" config user.name template-sync-check
	git -C "${target_missing_owner}" config user.email template-sync-check@example.invalid
	git -C "${target_missing_owner}" rm -q docs/repo-architecture.md
	git -C "${target_missing_owner}" commit -qm missing-repository-owner

	git clone -q "${template}" "${target_invalid_codex}"
	git -C "${target_invalid_codex}" config user.name template-sync-check
	git -C "${target_invalid_codex}" config user.email template-sync-check@example.invalid
	printf '%s\n' '# template-owned-codex-agents:start' >"${target_invalid_codex}/.codex/config.toml"
	git -C "${target_invalid_codex}" add .codex/config.toml
	git -C "${target_invalid_codex}" commit -qm malformed-codex-registry

	git clone -q "${template}" "${target_nonportable_codex}"
	git -C "${target_nonportable_codex}" config user.name template-sync-check
	git -C "${target_nonportable_codex}" config user.email template-sync-check@example.invalid
	printf '%s\n' '' '[mcp_servers.local-tool]' \
		"command = '/private/var/run/local-tool'" \
		>>"${target_nonportable_codex}/.codex/config.toml"
	git -C "${target_nonportable_codex}" add .codex/config.toml
	git -C "${target_nonportable_codex}" commit -qm nonportable-codex-config

	git clone -q "${template}" "${target_secret_codex}"
	git -C "${target_secret_codex}" config user.name template-sync-check
	git -C "${target_secret_codex}" config user.email template-sync-check@example.invalid
	printf '%s\n' '' '[mcp_servers.literal-secret]' 'bearer_token = ""' \
		>>"${target_secret_codex}/.codex/config.toml"
	git -C "${target_secret_codex}" add .codex/config.toml
	git -C "${target_secret_codex}" commit -qm literal-secret-codex-config

	git clone -q "${template}" "${target_missing_codex_source}"
	git -C "${target_missing_codex_source}" config user.name template-sync-check
	git -C "${target_missing_codex_source}" config user.email template-sync-check@example.invalid
	git -C "${target_missing_codex_source}" rm -q .agents/codex-project.toml
	git -C "${target_missing_codex_source}" commit -qm missing-codex-runtime-source

	printf 'v4\n' >"${template}/owned/version"
	git -C "${template}" add owned/version
	git -C "${template}" -c user.name=template-sync-check -c user.email=template-sync-check@example.invalid commit -qm v4
	if failure_output=$(bash "${sync_script}" --apply --no-commit --from "${template}" --repo "${target_with_directory}" 2>&1); then
		echo "template-owned purity: sync replaced a real Claude skill directory" >&2
		return 1
	fi
	grep -Fq 'refused: generated Claude skill links cannot be rebuilt safely' <<<"${failure_output}" || {
		echo "template-owned purity: Claude preflight refused for the wrong reason" >&2
		return 1
	}
	grep -Fxq v3 "${target_with_directory}/owned/version" || {
		echo "template-owned purity: Claude link preflight failed after manifest writes" >&2
		return 1
	}
	grep -Fxq 'only copy' "${target_with_directory}/.claude/skills/manual/SKILL.md" || {
		echo "template-owned purity: Claude link preflight changed a real skill directory" >&2
		return 1
	}
	if failure_output=$(bash "${sync_script}" --apply --no-commit --from "${template}" --repo "${target_missing_owner}" 2>&1); then
		echo "template-owned purity: sync accepted a missing repository-owned instruction" >&2
		return 1
	fi
	grep -Fq 'refused: required repository-owned instruction is missing' <<<"${failure_output}" || {
		echo "template-owned purity: repository-owner preflight refused for the wrong reason" >&2
		return 1
	}
	grep -Fxq v3 "${target_missing_owner}/owned/version" || {
		echo "template-owned purity: repository-owner refusal happened after manifest writes" >&2
		return 1
	}
	if failure_output=$(bash "${sync_script}" --apply --no-commit --from "${template}" --repo "${target_invalid_codex}" 2>&1); then
		echo "template-owned purity: sync accepted malformed Codex registry markers" >&2
		return 1
	fi
	grep -Fq 'refused: generated Codex project config cannot be rebuilt safely' <<<"${failure_output}" || {
		echo "template-owned purity: Codex preflight refused for the wrong reason" >&2
		return 1
	}
	grep -Fxq v3 "${target_invalid_codex}/owned/version" || {
		echo "template-owned purity: Codex registry refusal happened after manifest writes" >&2
		return 1
	}
	if ! bash "${sync_script}" --apply --no-commit --from "${template}" --repo "${target_nonportable_codex}" >/dev/null; then
		echo "template-owned purity: sync could not replace machine-local Codex config" >&2
		return 1
	fi
	if grep -Fq '/private/var/run/local-tool' "${target_nonportable_codex}/.codex/config.toml"; then
		echo "template-owned purity: sync retained a host-absolute Codex path" >&2
		return 1
	fi
	grep -Fxq v4 "${target_nonportable_codex}/owned/version" || {
		echo "template-owned purity: machine-local Codex cleanup omitted manifest writes" >&2
		return 1
	}
	if ! bash "${sync_script}" --apply --no-commit --from "${template}" --repo "${target_secret_codex}" >/dev/null; then
		echo "template-owned purity: sync could not replace literal Codex credentials" >&2
		return 1
	fi
	if grep -Eq '^[[:space:]]*bearer_token[[:space:]]*=' "${target_secret_codex}/.codex/config.toml"; then
		echo "template-owned purity: sync retained a literal Codex credential" >&2
		return 1
	fi
	grep -Fxq v4 "${target_secret_codex}/owned/version" || {
		echo "template-owned purity: literal Codex cleanup omitted manifest writes" >&2
		return 1
	}
	if ! bash "${sync_script}" --apply --no-commit --from "${template}" --repo "${target_missing_codex_source}" >/dev/null; then
		echo "template-owned purity: sync could not upgrade a consumer without the new Codex runtime source" >&2
		return 1
	fi
	[[ -f "${target_missing_codex_source}/.agents/codex-project.toml" ]] || {
		echo "template-owned purity: sync omitted the new Codex runtime source" >&2
		return 1
	}
	grep -Fxq 'max_concurrent_threads_per_session = 20' "${target_missing_codex_source}/.codex/config.toml" || {
		echo "template-owned purity: upgraded consumer omitted generated Codex runtime" >&2
		return 1
	}
	grep -Fxq v4 "${target_missing_codex_source}/owned/version" || {
		echo "template-owned purity: Codex runtime-source upgrade omitted manifest writes" >&2
		return 1
	}
)

template_sync_behavior_check
echo "template-owned manifest and sync behavior are safe"
