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
	"${template}/.agents/role-classes" \
	"${template}/.agents/roles" \
	"${template}/.agents/skills/fixture-skill" \
	"${template}/.opencode"
for file in template-sync.sh agent-roles-sync.sh harness-skills-sync.sh codex-agents-sync.sh; do
	cp -p "${root}/scripts/${file}" "${template}/scripts/${file}"
done
cp -p "${root}/scripts/lib/manifest.sh" "${root}/scripts/lib/sync-cli.sh" "${template}/scripts/lib/"
cp -p "${root}/.agents/codex-project.toml" "${template}/.agents/codex-project.toml"
cp -p "${root}/.agents/roles/worker-agent.toml" "${template}/.agents/roles/worker-agent.toml"
cp -p "${root}/.agents/role-classes/mutable-worker.md" \
	"${root}/.agents/role-classes/mutable-worker-fallback.md" \
	"${template}/.agents/role-classes/"
cp -p "${root}/.opencode/.gitignore" "${template}/.opencode/.gitignore"
printf '# fixture template v1\n' >"${template}/AGENTS.md"
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
.agents/codex-project.toml
.agents/role-classes/
.agents/roles/
.agents/skills/
.claude/agents/
.codex/agents/
.cursor/agents/
.grok/agents/
.grok/roles/
.opencode/.gitignore
.opencode/agents/
.qwen/agents/
template-owned.paths
scripts/template-sync.sh
scripts/agent-roles-sync.sh
scripts/harness-skills-sync.sh
scripts/codex-agents-sync.sh
scripts/lib/manifest.sh
scripts/lib/sync-cli.sh
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

assert_v1() {
	! grep -Fq 'fixture template v2' "$1/AGENTS.md" || fail "$2 changed before a safe apply"
}

target_direct="${fixture}/target-direct"
target_generated="${fixture}/target-generated"
target_dirty="${fixture}/target-dirty"
target_invalid="${fixture}/target-invalid"
for target in "${target_direct}" "${target_generated}" "${target_dirty}" "${target_invalid}"; do
	clone_target "${target}"
done

printf '.claude/skills/local-note\n' >>"${target_direct}/.git/info/exclude"
printf 'keep direct helper content\n' >"${target_direct}/.claude/skills/local-note"
report=$(expect_failure "Claude helper with a real local file" \
	bash "${target_direct}/scripts/harness-skills-sync.sh" claude --apply --repo "${target_direct}")
grep -Fq 'is not a generated symlink' <<<"${report}" || fail "Claude helper refused for the wrong reason"
grep -Fxq 'keep direct helper content' "${target_direct}/.claude/skills/local-note" ||
	fail "Claude helper changed a local file"

printf '\nfixture template v2\n' >>"${template}/AGENTS.md"
git -C "${template}" add AGENTS.md
git -C "${template}" commit -qm v2

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
grep -Fq 'ignored generated content could be overwritten' <<<"${report}" ||
	fail "ignored generated content was refused for the wrong reason: ${report}"
grep -Fxq 'keep claude content' "${target_generated}/.claude/skills/local-note" || fail "sync changed ignored Claude content"
grep -Fxq 'keep qwen content' "${target_generated}/.qwen/skills/local-note" || fail "sync changed ignored Qwen content"
grep -Fxq 'keep codex content' "${target_generated}/.codex/config.toml" || fail "sync changed ignored Codex content"
assert_v1 "${target_generated}" "ignored generated target"

printf '\nfixture dirty source\n' >>"${template}/AGENTS.md"
report=$(expect_failure "check with dirty source" \
	bash "${template}/scripts/template-sync.sh" --check --from "${template}" --repo "${target_dirty}")
grep -Fq 'template has uncommitted changes inside its own manifest' <<<"${report}" || fail "dirty check used the wrong candidate"
report=$(expect_failure "apply with dirty source" \
	bash "${template}/scripts/template-sync.sh" --apply --from "${template}" --repo "${target_dirty}")
grep -Fq 'template has uncommitted changes inside its own manifest' <<<"${report}" || fail "dirty apply refused for the wrong reason"
assert_v1 "${target_dirty}" "dirty-source target"
git -C "${template}" restore AGENTS.md

if ! report=$(bash "${template}/scripts/template-sync.sh" --apply --from "${template}" --repo "${target_dirty}" 2>&1); then
	fail "valid apply failed: ${report}"
fi
if ! report=$(bash "${template}/scripts/template-sync.sh" --check --from "${template}" --repo "${target_dirty}" 2>&1); then
	fail "repeat check failed: ${report}"
fi
grep -Fq 'fixture template v2' "${target_dirty}/AGENTS.md" || fail "valid apply omitted committed content"

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
