# Implementation Phase 1

## Goal

Close all audit findings for agent-instruction hygiene in one lightweight local implementation pass.

## Work Items

- Add a shared subagent contract and reusable brief template.
- Update Codex config with explicit fan-out/depth policy and compatibility notes.
- Add missing review skills for delivery, distributed systems, and observability.
- Update affected agent routing from "no dedicated review skill" to the new skills.
- Add agent mirror sync/check tooling and wire it into Makefile, Docker tooling, setup, and CI.
- Sync `.claude/agents` from `.codex/agents`.
- Update README and command docs to describe the new agent and skill hygiene model.

## Stop Rule

Stop after code/docs/tooling edits are complete and validation has fresh evidence, or record the validation gap in `workflow-plan.md` and `tasks.md`.

Status: completed; validation evidence is recorded in `../spec.md`.

## Allowed Writes

- `.codex/config.toml`
- `.codex/agents/*.toml`
- `.claude/agents/*.md`
- `.agents/skills/**`
- mirrored skill directories after `skills-sync`
- `scripts/dev/**`
- `Makefile`
- `.github/workflows/ci.yml`
- `README.md`
- `docs/**`
- `specs/agent-instruction-hygiene/**`

## Validation Handoff

Run the checks listed in `workflow-plans/validation-phase-1.md`.
