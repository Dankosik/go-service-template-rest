#!/usr/bin/env bash
set -euo pipefail

manifest="template-owned.paths"
failed=0

fail() {
	printf 'template-owned purity: %s\n' "$1" >&2
	failed=$((failed + 1))
}

[[ -f "${manifest}" ]] || {
	echo "template-owned purity: ${manifest} is missing" >&2
	exit 1
}

# shellcheck source=scripts/lib/manifest.sh
source "$(dirname -- "${BASH_SOURCE[0]}")/../lib/manifest.sh"
manifest_paths "${manifest}"
((${#paths[@]} > 0)) || fail "${manifest} lists no paths"

contains_path() {
	local expected="$1" entry
	for entry in "${paths[@]}"; do
		[[ "${entry}" == "${expected}" ]] && return 0
	done
	return 1
}

for entry in "${paths[@]}"; do
	path="${entry%/}"
	if [[ "${entry}" == */ ]]; then
		[[ -d "${path}" ]] || fail "missing directory: ${entry}"
		[[ -n "$(find "${path}" -type f -print -quit 2>/dev/null)" ]] || fail "empty directory: ${entry}"
	else
		[[ -f "${entry}" ]] || fail "missing file: ${entry}"
	fi
	marker="$(grep -RIlE -- '<!--[[:space:]]*profile:[^:]+:(start|end)[[:space:]]*-->' "${path}" 2>/dev/null | head -1 || true)"
	[[ -z "${marker}" ]] || fail "${marker} contains an initialization profile marker"
done

for required in \
	"${manifest}" \
	scripts/template-sync.sh \
	scripts/agent-roles-sync.sh \
	scripts/harness-skills-sync.sh \
	scripts/codex-agents-sync.sh \
	scripts/lib/manifest.sh \
	scripts/lib/sync-cli.sh; do
	contains_path "${required}" || fail "${manifest} must list ${required}"
done

for reserved in \
	README.md \
	docs/build-test-and-development-commands.md \
	docs/ci-cd-production-ready.md \
	docs/first-production-feature.md \
	docs/project-structure-and-module-organization.md \
	docs/railway-deployment-profile.md \
	docs/repo-architecture.md \
	test/README.md; do
	contains_path "${reserved}" && fail "${reserved} is repository-owned"
done

service_marker="$(find .agents/skills -mindepth 2 -maxdepth 2 -name .service-owned -print -quit 2>/dev/null || true)"
[[ -z "${service_marker}" ]] || fail "${service_marker} marks a template skill as service-owned"

if ! report="$(bash scripts/agent-roles-sync.sh --check --repo . 2>&1)"; then
	fail "role carriers are stale: ${report}"
fi
for harness in claude qwen; do
	if ! report="$(bash scripts/harness-skills-sync.sh "${harness}" --check --repo . 2>&1)"; then
		fail "${harness} skill view is stale: ${report}"
	fi
done
if ! report="$(bash scripts/codex-agents-sync.sh --check --repo . 2>&1)"; then
	fail "Codex project config is stale: ${report}"
fi
if ! report="$(bash scripts/ci/template-sync-behavior-check.sh 2>&1)"; then
	fail "template sync behavior is unsafe: ${report}"
fi

if ((failed != 0)); then
	exit 1
fi

echo "template-owned manifest is safe"
