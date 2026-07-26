#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "${ROOT_DIR}"

failed=0
fail() {
	echo "claude skills: $*"
	failed=1
}

# .agents/skills/ is the canonical skill set for every harness, but Claude Code
# only discovers skills under .claude/skills/, so `make claude-skills-sync`
# exposes each one there as a symlink. Drift is silent in the direction that
# matters: a skill without its link never loads, Claude Code reports nothing,
# and the only symptom is an agent that fails to apply a skill this repository
# owns. This check turns that into a failure.
linked=0
for skill_dir in .agents/skills/*/; do
	[[ -d "${skill_dir}" ]] || continue
	name="$(basename "${skill_dir}")"
	link=".claude/skills/${name}"
	# The path the sync generates. Relative, so it survives a checkout at any
	# location; an equivalent absolute link would not.
	expected="../../.agents/skills/${name}"

	if [[ -L "${link}" ]]; then
		actual="$(readlink "${link}")"
		if [[ "${actual}" != "${expected}" ]]; then
			fail "${link} points at ${actual}, expected ${expected}"
			continue
		fi
		if [[ ! -f "${link}/SKILL.md" ]]; then
			fail "${link} does not resolve to a readable SKILL.md"
			continue
		fi
		linked=$((linked + 1))
	elif [[ -e "${link}" ]]; then
		# A checkout without symlink support materializes the link as a regular
		# file holding the target path, and a hand-copied directory diverges
		# from the canonical skill with nothing to report it.
		fail "${link} is not a symlink; .agents/skills/${name} is the only copy"
	else
		fail "${link} is missing, so Claude Code cannot see the ${name} skill"
	fi
done

# A link left behind by a renamed or deleted skill dangles, and would offer
# Claude Code a skill this repository no longer owns.
for link in .claude/skills/*; do
	[[ -e "${link}" || -L "${link}" ]] || continue
	name="$(basename "${link}")"
	if [[ ! -d ".agents/skills/${name}" ]]; then
		fail "${link} has no owner in .agents/skills/"
	fi
done

if ((failed != 0)); then
	echo "claude skills: run 'make claude-skills-sync' to rebuild .claude/skills"
	exit 1
fi

echo "claude skills: ${linked} skill links current"
