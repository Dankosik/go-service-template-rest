#!/usr/bin/env bash
set -euo pipefail

fail() {
	printf 'template sync behavior: %s\n' "$1" >&2
	exit 1
}

root=$(CDPATH='' cd -- "$(dirname -- "${BASH_SOURCE[0]}")/../.." && pwd)
fixture=$(mktemp -d "${TMPDIR:-/tmp}/template-sync-behavior.XXXXXX")
trap 'rm -rf -- "${fixture}"' EXIT
template="${fixture}/template"

# shellcheck source=scripts/lib/manifest.sh
source "${root}/scripts/lib/manifest.sh"
mkdir -p \
		"${template}/scripts/lib" \
		"${template}/scripts/ci" \
		"${template}/make" \
	"${template}/.agents/role-classes" \
	"${template}/.agents/roles" \
	"${template}/.agents/skills/fixture-skill" \
	"${template}/.opencode"
for file in template-sync.sh template-settings-sync.py agent-roles-sync.sh harness-skills-sync.sh codex-agents-sync.sh; do
	cp -p "${root}/scripts/${file}" "${template}/scripts/${file}"
done
cp -p "${root}/scripts/lib/manifest.sh" "${root}/scripts/lib/sync-cli.sh" "${template}/scripts/lib/"
cp -p "${root}/.agents/codex-project.toml" "${template}/.agents/codex-project.toml"
cp -p "${root}/.agents/roles/worker-agent.toml" "${template}/.agents/roles/worker-agent.toml"
cp -p "${root}/.agents/roles/reviewer-agent.toml" "${template}/.agents/roles/reviewer-agent.toml"
cp -p "${root}/.agents/role-classes/mutable-worker.md" \
	"${root}/.agents/role-classes/mutable-worker-fallback.md" \
	"${template}/.agents/role-classes/"
cp -p "${root}/.agents/role-classes/read-only-specialist.md" \
	"${root}/.agents/role-classes/read-only-specialist-fallback.md" \
	"${template}/.agents/role-classes/"
cp -p "${root}/.opencode/.gitignore" "${template}/.opencode/.gitignore"
printf '# fixture template v1\n' >"${template}/AGENTS.md"
printf '%s\n' 'include make/template.mk' '-include make/service.mk' >"${template}/Makefile"
printf 'standard-check:\n\t@printf '\''standard v1\\n'\''\n' >"${template}/make/template.mk"
printf '%s\n' '#!/usr/bin/env bash' 'printf '\''portable v1\n'\''' >"${template}/scripts/ci/portable-check.sh"
chmod +x "${template}/scripts/ci/portable-check.sh"
cat >"${template}/.agents/skills/fixture-skill/SKILL.md" <<'EOF'
---
name: fixture-skill
description: "Fixture method."
metadata:
  invocation: model
  kind: method
---

# Fixture Skill
EOF
cat >"${template}/template-owned.paths" <<'EOF'
AGENTS.md
Makefile
make/template.mk
.agents/codex-project.toml
.agents/role-classes/
.agents/roles/
.agents/skills/
.claude/agents/
.claude/settings.json
.codex/agents/
.cursor/agents/
.grok/agents/
.grok/roles/
.opencode/.gitignore
.opencode/agents/
.qwen/agents/
.qwen/settings.json
template-owned.paths
scripts/template-sync.sh
scripts/template-settings-sync.py
scripts/agent-roles-sync.sh
scripts/harness-skills-sync.sh
scripts/codex-agents-sync.sh
scripts/lib/manifest.sh
scripts/lib/sync-cli.sh
scripts/ci/portable-check.sh
EOF
for file in \
	docs/repo-architecture.md \
	docs/project-structure-and-module-organization.md \
	docs/build-test-and-development-commands.md \
	docs/ci-cd-production-ready.md \
	docs/railway-deployment-profile.md \
	test/README.md; do
	mkdir -p "${template}/$(dirname -- "${file}")"
	printf '# Fixture repository owner\n' >"${template}/${file}"
done
bash "${template}/scripts/agent-roles-sync.sh" --apply --repo "${template}" >/dev/null
for harness in claude qwen; do
	cp -p "${root}/.${harness}/settings.json" "${template}/.${harness}/settings.json"
	cp -p "${root}/.${harness}/agents/acceptance-unit-lead.md" "${template}/.${harness}/agents/acceptance-unit-lead.md"
done
# Regeneration must preserve the handwritten native Lead carriers.
bash "${template}/scripts/agent-roles-sync.sh" --apply --repo "${template}" >/dev/null
grep -Eq '^tools:.* Agent(,|$)' "${template}/.claude/agents/worker-agent.md" || fail "Claude worker cannot delegate"
if grep -Eq '^tools:.* Agent(,|$)' "${template}/.claude/agents/reviewer-agent.md"; then
	fail "Claude reviewer gained mutable delegation"
fi
grep -Fxq '  - agent' "${template}/.qwen/agents/worker-agent.md" || fail "Qwen worker cannot delegate"
if grep -Fxq '  - agent' "${template}/.qwen/agents/reviewer-agent.md"; then
	fail "Qwen reviewer gained mutable delegation"
fi
grep -Fxq '    "*": deny' "${template}/.opencode/agents/worker-agent.md" || fail "OpenCode worker lost its delegation default deny"
grep -Fxq '    worker-agent: allow' "${template}/.opencode/agents/worker-agent.md" || fail "OpenCode worker cannot delegate"
grep -Fxq '    evidence-agent: allow' "${template}/.opencode/agents/worker-agent.md" || fail "OpenCode worker cannot request evidence"
grep -Fxq '  task: deny' "${template}/.opencode/agents/reviewer-agent.md" || fail "OpenCode reviewer gained mutable delegation"
for harness in claude qwen; do
	cmp -s "${root}/.${harness}/agents/acceptance-unit-lead.md" "${template}/.${harness}/agents/acceptance-unit-lead.md" ||
		fail "${harness} Lead carrier was not preserved"
done
bash "${template}/scripts/harness-skills-sync.sh" claude --apply --repo "${template}" >/dev/null
bash "${template}/scripts/harness-skills-sync.sh" qwen --apply --repo "${template}" >/dev/null
bash "${template}/scripts/codex-agents-sync.sh" --apply --repo "${template}" >/dev/null
git -C "${template}" init -q
git -C "${template}" config user.name template-sync-behavior
git -C "${template}" config user.email template-sync-behavior@example.invalid
git -C "${template}" add -A
git -C "${template}" commit -qm v1

clone_target() {
	git clone -q "${template}" "$1"
	git -C "$1" config user.name template-sync-behavior
	git -C "$1" config user.email template-sync-behavior@example.invalid
}

expect_failure() {
	local output
	if output=$("${@:2}" 2>&1); then
		fail "$1 unexpectedly succeeded"
	fi
	printf '%s' "${output}"
}

purity="${fixture}/purity-routing"
mkdir -p "${purity}/scripts/ci"
cp "${root}/make/template.mk" "${purity}/Makefile"
printf '%s\n' 'module example.invalid/purity-routing' 'go 1.27.0' >"${purity}/go.mod"
printf '%s\n' 'echo "fixture purity validator invoked"' 'exit 42' >"${purity}/scripts/ci/template-owned-purity-check.sh"
report=$(expect_failure "source purity failure" \
	make -s -C "${purity}" --no-print-directory template-owned-purity-check)
grep -Fq 'fixture purity validator invoked' <<<"${report}" || fail "source purity validator was not invoked"
printf '%s\n' 'state = "complete"' 'agent_harness = "core"' >"${purity}/template.lock"
if ! report=$(make -s -C "${purity}" --no-print-directory template-owned-purity-check 2>&1); then
	fail "initialized consumer ran a stale source-only purity validator: ${report}"
fi
grep -Fq 'not applicable: template manifest purity is source-only' <<<"${report}" ||
	fail "consumer purity skip was not explicit"
[[ -f "${purity}/scripts/ci/template-owned-purity-check.sh" ]] || fail "consumer purity validator was removed"

assert_v1() {
	! grep -Fq 'fixture template v2' "$1/AGENTS.md" || fail "$2 changed before a safe apply"
}

target_direct="${fixture}/target-direct"
target_generated="${fixture}/target-generated"
target_pruned="${fixture}/target-pruned"
target_dirty="${fixture}/target-dirty"
target_invalid="${fixture}/target-invalid"
target_legacy="${fixture}/target-legacy"
for target in "${target_direct}" "${target_generated}" "${target_pruned}" "${target_dirty}" "${target_invalid}" "${target_legacy}"; do
	clone_target "${target}"
done

# Settings outside the two owned leaves are committed consumer data, not drift.
cat >"${target_dirty}/.claude/settings.json" <<'EOF'
{"hooks":{"Stop":[{"hooks":[{"type":"command","command":"echo local"}]}]},"env":{"LOCAL_ENV":"keep","CLAUDE_CODE_MAX_SUBAGENT_SPAWN_DEPTH":"3"},"permissions":{"deny":["Read(.env)"]},"precise":1.234567890123456789,"model":"consumer-model"}
EOF
cat >"${target_dirty}/.qwen/settings.json" <<'EOF'
{"env":{"LOCAL_ENV":"keep"},"hooks":{"AfterTool":[]},"permissions":{"allow":["Read"]},"model":{"name":"consumer-model","maxSubagentDepth":5,"generationConfig":{"temperature":0.1234567890123456789}}}
EOF
for harness in claude qwen; do
	cp "${target_dirty}/.${harness}/settings.json" "${fixture}/${harness}-before.json"
done
git -C "${target_dirty}" add .claude/settings.json .qwen/settings.json
git -C "${target_dirty}" commit -qm consumer-settings
if ! report=$(bash "${template}/scripts/template-sync.sh" --check --from "${template}" --repo "${target_dirty}" 2>&1); then
	fail "consumer-owned settings were treated as drift: ${report}"
fi

git -C "${target_legacy}" rm -q make/template.mk
printf 'legacy-local:\n\t@printf '\''legacy local\\n'\''\n' >"${target_legacy}/Makefile"
git -C "${target_legacy}" add Makefile
git -C "${target_legacy}" commit -qm legacy-makefile

printf '.claude/skills/local-note\n' >>"${target_direct}/.git/info/exclude"
printf 'keep direct helper content\n' >"${target_direct}/.claude/skills/local-note"
report=$(expect_failure "Claude helper with a real local file" \
	bash "${target_direct}/scripts/harness-skills-sync.sh" claude --apply --repo "${target_direct}")
grep -Fq 'is not a generated symlink' <<<"${report}" || fail "Claude helper refused for the wrong reason"
grep -Fxq 'keep direct helper content' "${target_direct}/.claude/skills/local-note" ||
	fail "Claude helper changed a local file"

printf '\nfixture template v2\n' >>"${template}/AGENTS.md"
sed -i.bak 's/standard v1/standard v2/' "${template}/make/template.mk"
rm -f "${template}/make/template.mk.bak"
sed -i.bak 's/portable v1/portable v2/' "${template}/scripts/ci/portable-check.sh"
rm -f "${template}/scripts/ci/portable-check.sh.bak"
# Make native settings and Lead carriers stale in every already-cloned consumer.
printf '%s\n' '{"env":{"CLAUDE_CODE_MAX_SUBAGENT_SPAWN_DEPTH":"4"}}' >"${template}/.claude/settings.json"
printf '%s\n' '{"model":{"maxSubagentDepth":6}}' >"${template}/.qwen/settings.json"
for harness in claude qwen; do
	printf '\nFixture Lead v2.\n' >>"${template}/.${harness}/agents/acceptance-unit-lead.md"
	if cmp -s "${template}/.${harness}/settings.json" "${target_dirty}/.${harness}/settings.json" ||
		cmp -s "${template}/.${harness}/agents/acceptance-unit-lead.md" "${target_dirty}/.${harness}/agents/acceptance-unit-lead.md"; then
		fail "${harness} propagation fixture did not start with stale consumer content"
	fi
done
git -C "${template}" add AGENTS.md make/template.mk scripts/ci/portable-check.sh \
	.claude/settings.json .qwen/settings.json \
	.claude/agents/acceptance-unit-lead.md .qwen/agents/acceptance-unit-lead.md
git -C "${template}" commit -qm v2

# Instruction adoption does not migrate builds or consume target-side tooling.
for scenario in legacy ignored-tooling codex core cursor grok opencode; do
	instruction_target="${fixture}/instructions-${scenario}"
	git clone -q "${target_legacy}" "${instruction_target}"
	printf 'exit 91\n' >"${instruction_target}/scripts/harness-skills-sync.sh"
	chmod -x "${instruction_target}/scripts/template-sync.sh"
	printf 'older full-sync receipt\n' >"${instruction_target}/.template-sync"
	case "${scenario}" in
	ignored-tooling)
		mkdir -p "${instruction_target}/make"
		printf 'local ignored tooling\n' >"${instruction_target}/make/template.mk"
		printf '/make/template.mk\n' >>"${instruction_target}/.git/info/exclude"
		;;
	codex | core | cursor | grok | opencode)
		printf 'state = "complete"\nagent_harness = "%s"\n' "${scenario}" >"${instruction_target}/template.lock"
		mkdir -p "${instruction_target}/.claude/worktrees"
		printf 'local worktree content\n' >"${instruction_target}/.claude/worktrees/local"
		printf '/.claude/worktrees/\n' >>"${instruction_target}/.git/info/exclude"
		;;
	esac
	tooling_before=$(git -C "${instruction_target}" diff --binary HEAD -- Makefile make scripts template-owned.paths)
	if ! report=$(bash "${template}/scripts/template-sync.sh" --apply --instructions-only \
		--from "${template}" --repo "${instruction_target}" 2>&1); then
		fail "${scenario} instruction adoption failed: ${report}"
	fi
	cmp -s "${template}/AGENTS.md" "${instruction_target}/AGENTS.md" || fail "instruction bytes did not propagate"
	[[ "${tooling_before}" == "$(git -C "${instruction_target}" diff --binary HEAD -- Makefile make scripts template-owned.paths)" ]] ||
		fail "instruction adoption changed tooling or its local work"
	[[ ! -x "${instruction_target}/scripts/template-sync.sh" ]] || fail "instruction adoption changed tooling mode"
	grep -Fxq 'older full-sync receipt' "${instruction_target}/.template-sync" || fail "instruction adoption removed a full-sync receipt"
	case "${scenario}" in
	legacy) [[ ! -e "${instruction_target}/make/template.mk" ]] || fail "instruction adoption migrated the legacy build" ;;
	ignored-tooling) grep -Fxq 'local ignored tooling' "${instruction_target}/make/template.mk" || fail "ignored tooling was changed" ;;
	codex | core | cursor | grok | opencode)
		grep -Fxq 'local worktree content' "${instruction_target}/.claude/worktrees/local" || fail "unselected harness data was removed"
		cmp -s "${target_legacy}/.claude/settings.json" "${instruction_target}/.claude/settings.json" || fail "unselected settings changed"
		;;
	esac
	if ! report=$(bash "${template}/scripts/template-sync.sh" --check --instructions-only \
		--from "${template}" --repo "${instruction_target}" 2>&1); then
		fail "instruction parity did not converge: ${report}"
	fi
	grep -Fq 'template-owned agent instructions are current' <<<"${report}" || fail "instruction check claimed full parity"
	printf '\nlocal instruction work\n' >>"${instruction_target}/AGENTS.md"
	cp "${instruction_target}/AGENTS.md" "${fixture}/instruction-before.md"
	report=$(expect_failure "dirty selected instructions" bash "${template}/scripts/template-sync.sh" \
		--apply --instructions-only --from "${template}" --repo "${instruction_target}")
	grep -Fq 'the sync would overwrite them' <<<"${report}" || fail "dirty instruction refusal used the wrong reason"
	cmp -s "${fixture}/instruction-before.md" "${instruction_target}/AGENTS.md" || fail "dirty instructions were overwritten"
done

# Invalid committed source depths must refuse before touching an older consumer.
for invalid_depth in qwen-zero qwen-over-limit claude-zero; do
	settings_source="${fixture}/${invalid_depth}"
	clone_target "${settings_source}"
	case "${invalid_depth}" in
	qwen-zero) printf '%s\n' '{"model":{"maxSubagentDepth":0}}' >"${settings_source}/.qwen/settings.json" ;;
	qwen-over-limit) printf '%s\n' '{"model":{"maxSubagentDepth":101}}' >"${settings_source}/.qwen/settings.json" ;;
	claude-zero) printf '%s\n' '{"env":{"CLAUDE_CODE_MAX_SUBAGENT_SPAWN_DEPTH":"0"}}' >"${settings_source}/.claude/settings.json" ;;
	esac
	git -C "${settings_source}" add .claude/settings.json .qwen/settings.json
	git -C "${settings_source}" commit -qm invalid-source-depth
	report=$(expect_failure "apply with ${invalid_depth} source depth" \
		bash "${settings_source}/scripts/template-sync.sh" --apply --from "${settings_source}" --repo "${target_invalid}")
	grep -Fq 'template revision has invalid managed settings' <<<"${report}" || fail "source depth refusal used the wrong reason"
	[[ -z "$(git -C "${target_invalid}" status --porcelain)" ]] || fail "invalid source depth changed consumer content"
	assert_v1 "${target_invalid}" "invalid-source-depth target"
done

# Each refusal must precede every target write, even when other files have drift.
for harness in claude qwen; do
	for malformed in '{"private":"fixture-secret",' '[]' '{"env":null,"model":[]}' '{"private":1,"private":2}' '{"private":NaN}'; do
		settings_target="${fixture}/settings-target"
		clone_target "${settings_target}"
		# Give the template a separate visible change so an early copy is detectable.
		printf '# fixture template v1\n' >"${settings_target}/AGENTS.md"
		printf '%s\n' "${malformed}" >"${settings_target}/.${harness}/settings.json"
		git -C "${settings_target}" add AGENTS.md ".${harness}/settings.json"
		git -C "${settings_target}" commit -qm malformed-settings
		report=$(expect_failure "apply with malformed ${harness} settings" \
			bash "${template}/scripts/template-sync.sh" --apply --from "${template}" --repo "${settings_target}")
		grep -Fq 'invalid managed settings' <<<"${report}" || fail "settings refusal used the wrong reason"
		if grep -Fq 'fixture-secret' <<<"${report}"; then fail "settings refusal exposed JSON content"; fi
		[[ -z "$(git -C "${settings_target}" status --porcelain)" ]] || fail "malformed settings changed target content"
		assert_v1 "${settings_target}" "malformed-settings target"
		rm -rf -- "${settings_target}"
	done
done

for missing in files parents; do
	settings_target="${fixture}/settings-${missing}"
	clone_target "${settings_target}"
	if [[ "${missing}" == files ]]; then
		git -C "${settings_target}" rm -q .claude/settings.json .qwen/settings.json
	else
		printf '%s\n' '{"permissions":{"deny":["Read(.env)"]}}' >"${settings_target}/.claude/settings.json"
		printf '%s\n' '{"model":{"name":"consumer-model"}}' >"${settings_target}/.qwen/settings.json"
		git -C "${settings_target}" add .claude/settings.json .qwen/settings.json
	fi
	git -C "${settings_target}" commit -qm missing-settings
	if ! report=$(bash "${template}/scripts/template-sync.sh" --apply --from "${template}" --repo "${settings_target}" 2>&1); then
		fail "missing settings ${missing} could not be created: ${report}"
	fi
	python3 - "${settings_target}" "${missing}" <<'PY'
import json
from pathlib import Path
import sys
root = Path(sys.argv[1])
claude = json.loads((root / ".claude/settings.json").read_text())
qwen = json.loads((root / ".qwen/settings.json").read_text())
assert claude["env"]["CLAUDE_CODE_MAX_SUBAGENT_SPAWN_DEPTH"] == "4"
assert qwen["model"]["maxSubagentDepth"] == 6
if sys.argv[2] == "parents":
    assert claude["permissions"] == {"deny": ["Read(.env)"]}
    assert qwen["model"]["name"] == "consumer-model"
PY
done

report=$(expect_failure "apply with a legacy Makefile" \
	bash "${template}/scripts/template-sync.sh" --apply --from "${template}" --repo "${target_legacy}")
grep -Fq 'legacy Makefile requires explicit service-target extraction' <<<"${report}" ||
	fail "legacy Makefile was refused for the wrong reason: ${report}"
test "$(make -s -C "${target_legacy}" --no-print-directory legacy-local)" = 'legacy local' ||
	fail "legacy Makefile changed before migration"

mkdir -p "${target_direct}/.agents/skills/local-note"
printf 'local skill\n' >"${target_direct}/.agents/skills/local-note/SKILL.md"
printf 'service owned\n' >"${target_direct}/.agents/skills/local-note/.service-owned"
git -C "${target_direct}" add .agents/skills/local-note
git -C "${target_direct}" commit -qm service-owned-local-note
report=$(expect_failure "sync with a non-link service-owned discovery path" \
	bash "${template}/scripts/template-sync.sh" --apply --from "${template}" --repo "${target_direct}")
grep -Fq 'service-owned skill local-note has a non-symlink Claude discovery path' <<<"${report}" ||
	fail "service-owned discovery path was refused for the wrong reason: ${report}"
grep -Fxq 'keep direct helper content' "${target_direct}/.claude/skills/local-note" ||
	fail "sync changed a service-owned discovery file"
assert_v1 "${target_direct}" "service-owned discovery target"

git -C "${target_generated}" rm -q .codex/config.toml
git -C "${target_generated}" commit -qm local-config-untracked
printf '%s\n' \
	'.claude/skills/local-note' \
	'.qwen/skills/local-note' \
	'.codex/config.toml' >>"${target_generated}/.git/info/exclude"
printf 'keep claude content\n' >"${target_generated}/.claude/skills/local-note"
printf 'keep qwen content\n' >"${target_generated}/.qwen/skills/local-note"
printf 'keep codex content\n' >"${target_generated}/.codex/config.toml"
report=$(expect_failure "sync with ignored generated content" \
	bash "${template}/scripts/template-sync.sh" --apply --from "${template}" --repo "${target_generated}")
grep -Fq 'ignored generated or pruned content could be overwritten' <<<"${report}" ||
	fail "ignored generated content was refused for the wrong reason: ${report}"
grep -Fxq 'keep claude content' "${target_generated}/.claude/skills/local-note" || fail "sync changed ignored Claude content"
grep -Fxq 'keep qwen content' "${target_generated}/.qwen/skills/local-note" || fail "sync changed ignored Qwen content"
grep -Fxq 'keep codex content' "${target_generated}/.codex/config.toml" || fail "sync changed ignored Codex content"
assert_v1 "${target_generated}" "ignored generated target"

printf '%s\n' 'state = "complete"' 'agent_harness = "core"' >"${target_pruned}/template.lock"
git -C "${target_pruned}" add template.lock
git -C "${target_pruned}" commit -qm core-harness
mkdir -p "${target_pruned}/.opencode/node_modules"
printf 'keep pruned content\n' >"${target_pruned}/.opencode/node_modules/local-note"
report=$(expect_failure "sync with ignored pruned content" \
	bash "${template}/scripts/template-sync.sh" --apply --from "${template}" --repo "${target_pruned}")
grep -Fq 'ignored generated or pruned content could be overwritten' <<<"${report}" ||
	fail "ignored pruned content was refused for the wrong reason: ${report}"
grep -Fxq 'keep pruned content' "${target_pruned}/.opencode/node_modules/local-note" ||
	fail "sync changed ignored pruned content"
assert_v1 "${target_pruned}" "ignored pruned target"

printf '\nfixture dirty source\n' >>"${template}/AGENTS.md"
report=$(expect_failure "check with dirty source" \
	bash "${template}/scripts/template-sync.sh" --check --from "${template}" --repo "${target_dirty}")
grep -Fq 'template has uncommitted changes inside its own manifest' <<<"${report}" || fail "dirty check used the wrong candidate"
report=$(expect_failure "apply with dirty source" \
	bash "${template}/scripts/template-sync.sh" --apply --from "${template}" --repo "${target_dirty}")
grep -Fq 'template has uncommitted changes inside its own manifest' <<<"${report}" || fail "dirty apply refused for the wrong reason"
assert_v1 "${target_dirty}" "dirty-source target"
git -C "${template}" restore AGENTS.md

mkdir -p "${target_dirty}/make" "${target_dirty}/scripts/ci"
printf 'local-check:\n\t@printf '\''local service\\n'\''\n' >"${target_dirty}/make/service.mk"
printf '%s\n' '#!/usr/bin/env bash' 'printf '\''local script\n'\''' >"${target_dirty}/scripts/ci/local-check.sh"
chmod +x "${target_dirty}/scripts/ci/local-check.sh"
git -C "${target_dirty}" add make/service.mk scripts/ci/local-check.sh
git -C "${target_dirty}" commit -qm local-tooling

printf 'dirty portable target\n' >"${target_dirty}/scripts/ci/portable-check.sh"
report=$(expect_failure "apply with dirty portable target" \
	bash "${template}/scripts/template-sync.sh" --apply --from "${template}" --repo "${target_dirty}")
grep -Fq 'uncommitted changes inside the manifest' <<<"${report}" ||
	fail "dirty portable target was refused for the wrong reason: ${report}"
grep -Fxq 'dirty portable target' "${target_dirty}/scripts/ci/portable-check.sh" ||
	fail "dirty portable target changed before refusal"
git -C "${target_dirty}" restore scripts/ci/portable-check.sh

if ! report=$(bash "${template}/scripts/template-sync.sh" --apply --from "${template}" --repo "${target_dirty}" 2>&1); then
	fail "valid apply failed: ${report}"
fi
if ! report=$(bash "${template}/scripts/template-sync.sh" --check --from "${template}" --repo "${target_dirty}" 2>&1); then
	fail "repeat check failed: ${report}"
fi
grep -Fq 'fixture template v2' "${target_dirty}/AGENTS.md" || fail "valid apply omitted committed content"
for harness in claude qwen; do
	cmp -s "${template}/.${harness}/agents/acceptance-unit-lead.md" "${target_dirty}/.${harness}/agents/acceptance-unit-lead.md" ||
		fail "${harness} Lead carrier was not propagated"
done
python3 - "${fixture}" "${target_dirty}" <<'PY'
from pathlib import Path
import sys
fixture, target = map(Path, sys.argv[1:])
for harness, before, after in (
    ("claude", '"CLAUDE_CODE_MAX_SUBAGENT_SPAWN_DEPTH":"3"', '"CLAUDE_CODE_MAX_SUBAGENT_SPAWN_DEPTH":"4"'),
    ("qwen", '"maxSubagentDepth":5', '"maxSubagentDepth":6'),
):
    original = (fixture / (harness + "-before.json")).read_bytes()
    expected = original.replace(before.encode(), after.encode())
    assert expected != original, "fixture did not start with a stale owned depth"
    assert (target / ("." + harness) / "settings.json").read_bytes() == expected, "consumer settings lost bytes or owned depth did not update"
PY
test "$(make -s -C "${target_dirty}" --no-print-directory standard-check)" = 'standard v2' ||
	fail "standard Make target was not updated"
test "$(make -s -C "${target_dirty}" --no-print-directory local-check)" = 'local service' ||
	fail "service Make target was not preserved"
test "$(bash "${target_dirty}/scripts/ci/portable-check.sh")" = 'portable v2' ||
	fail "portable script was not updated"
test "$(bash "${target_dirty}/scripts/ci/local-check.sh")" = 'local script' ||
	fail "service script was not preserved"

role="${template}/.agents/roles/worker-agent.toml"
sed 's/^description = .*/description = "unsafe "quoted" description"/' "${role}" >"${role}.tmp"
mv "${role}.tmp" "${role}"
git -C "${template}" add "${role#"${template}/"}"
git -C "${template}" commit -qm invalid-role-description
report=$(expect_failure "apply with invalid committed role metadata" \
	bash "${template}/scripts/template-sync.sh" --apply --from "${template}" --repo "${target_invalid}")
grep -Fq 'description contains unsupported quoting' <<<"${report}" || fail "invalid role metadata was refused for the wrong reason"
[[ -z "$(git -C "${target_invalid}" status --porcelain)" ]] || fail "invalid committed source changed the target"
assert_v1 "${target_invalid}" "invalid-source target"

expect_bad_manifest() {
	local name="$1"
	shift
	printf '%s\n' "$@" >"${fixture}/${name}.paths"
	if (manifest_paths "${fixture}/${name}.paths") >/dev/null 2>&1; then
		fail "${name} manifest unexpectedly passed"
	fi
}
expect_bad_manifest duplicate alpha alpha
expect_bad_manifest overlap alpha/ alpha/file
expect_bad_manifest alias ./alpha
expect_bad_manifest pathspec ':(glob)alpha'
expect_bad_manifest wildcard 'alpha*'

echo "template sync behavior is safe"
