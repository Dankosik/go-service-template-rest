## Context

The repository currently supports research fan-out, challenger patterns, and recheck, but it does not expose a dedicated pre-spec challenge checkpoint that pressure-tests candidate decisions before they solidify into `spec.md`.

The user approved integrating a risk-driven `pre-spec challenge` workflow step, plus any supporting agent and skill surfaces needed to make it executable rather than merely documented.

## Scope / Non-goals

In scope:
- update repository-wide workflow contracts to make `pre-spec challenge` explicit
- update overview docs so the workflow summary and agent portfolio stay accurate
- add a read-only project agent for pre-spec challenge
- add a reusable skill for discriminating pre-spec challenge passes
- update brainstorming guidance so framing can route into challenge rather than claiming readiness too early

Non-goals:
- changing implementation/review/validation semantics outside what the challenge checkpoint requires
- introducing a mandatory ritual phase for every small task
- adding code generation or evaluation harnesses for the new skill in this iteration
- rewriting legacy deep-dive docs unless they are directly needed to prevent active contradiction

## Constraints

- Final decisions must remain with the orchestrator.
- Subagents and the new challenger surface must stay read-only/advisory.
- The workflow must remain simple and risk-driven rather than turning into a mandatory linear chain.
- `spec.md` remains the canonical decision artifact; raw challenge transcripts must not become a second source of truth.
- Changes should cover the active repository surfaces: root contract docs, workflow docs, project agent registry, and mirrored skill instructions used in this repo.

## Decisions

1. Integrate `pre-spec challenge` as a named checkpoint inside the `synthesis` boundary, not as a heavyweight permanent authority phase.
2. Describe the recommended flow as `research -> candidate synthesis -> pre-spec challenge -> (re-research if needed) -> final synthesis/spec -> planning`.
3. Make the checkpoint default for medium/high-risk or ambiguous work and skippable for small/low-risk work with rationale.
4. Add a dedicated read-only `challenger-agent` surface for targeted questioning and candidate-decision pressure testing.
5. Add a dedicated `pre-spec-challenge` skill so the pattern is reusable across agents/runtimes instead of living only in prose.
6. Update `spec-first-brainstorming` so it frames the problem and explicitly routes to challenge when readiness depends on unresolved discriminating questions.
7. Keep challenge outputs compact and resolution-oriented: challenged assumption/question, why it matters now, what changes if answered differently, blocker level, and next action.
8. Keep `challenger-agent` intentionally thin: ownership, trigger rules, boundaries, and handoffs stay in the agent; protocol, output shape, stop condition, and anti-patterns live in the skill.

## Open Questions / Assumptions

- Corrected: active skill mirrors present in this repository for the affected surfaces are `skills/`, `.agents/skills/`, `.claude/skills/`, `.cursor/skills/`, `.gemini/skills/`, `.github/skills/`, and `.opencode/skills/`.
- Validated: adding the new agent to `.codex/config.toml` plus `.codex/agents/` and `.claude/agents/` is sufficient for the project-scoped agent surfaces visible in this repository.
- Resolved: instead of rewriting older `docs/skills/*` design notes, mark the directly affected `docs/skills/spec-first-brainstorming-spec.md` as a historical note so it no longer competes with the active runtime contract.

## Implementation Plan

1. Update `AGENTS.md`, `CLAUDE.md`, `README.md`, and `docs/spec-first-workflow.md` so the workflow explicitly includes the pre-spec challenge checkpoint, its risk-driven trigger rules, and resolution behavior.
   Completion criteria:
   - challenge is described as a checkpoint inside synthesis, not a new authority center
   - docs state when to run it and how it can loop back to research
   - docs clarify that `spec.md` stores resolutions, not raw transcripts

2. Add a new `challenger-agent` in project agent surfaces and register it in Codex configuration plus README portfolio summaries.
   Completion criteria:
   - `.codex/config.toml` includes the new agent
   - `.codex/agents/challenger-agent.toml` and `.claude/agents/challenger-agent.md` exist
   - agent instructions keep the role read-only, advisory, and pre-spec scoped

3. Add a reusable `pre-spec-challenge` skill in canonical and mirrored locations, then update `spec-first-brainstorming` to hand off to it when readiness depends on challenge.
   Completion criteria:
   - `skills/pre-spec-challenge/SKILL.md` exists
   - mirrored copies exist in `.agents/skills/` and `.claude/skills/`
   - brainstorming instructions mention candidate synthesis / challenge routing without taking final design ownership

4. Validate consistency with targeted searches and diffs, then update this spec `Validation` and `Outcome`.
   Completion criteria:
   - searches show the new agent/skill/workflow terms in all intended surfaces
   - any contradictions discovered during validation are resolved or called out explicitly

5. Run a blind A/B comparison between the current `pre-spec-challenge` skill and an alternative draft, then keep the stronger skill and sync mirrors.
   Completion criteria:
   - at least `2-3` realistic eval prompts exist for challenge behavior
   - both skill variants are executed against the same eval set
   - blind comparison produces a winner or justified tie with rationale
   - the chosen skill version becomes canonical and mirrors are resynced

## Validation

Executed:
- `rg -n "pre-spec challenge|pre-spec-challenge|challenger-agent|candidate synthesis|ritualized coverage|challenge-handoff" AGENTS.md CLAUDE.md README.md docs/spec-first-workflow.md .codex/config.toml .codex/agents .claude/agents skills .agents/skills .claude/skills specs/pre-spec-challenge-workflow/spec.md`
- `rg -n "whether candidate synthesis|open-ended redesign|challenge resolutions or skip rationale|historical design note|pre-spec-challenge" AGENTS.md CLAUDE.md docs/spec-first-workflow.md README.md docs/skills/spec-first-brainstorming-spec.md skills/spec-first-brainstorming/SKILL.md .agents/skills/spec-first-brainstorming/SKILL.md .claude/skills/spec-first-brainstorming/SKILL.md skills/pre-spec-challenge/SKILL.md .agents/skills/pre-spec-challenge/SKILL.md .claude/skills/pre-spec-challenge/SKILL.md .codex/config.toml .codex/agents/challenger-agent.toml .claude/agents/challenger-agent.md`
- `git diff -- AGENTS.md CLAUDE.md README.md docs/spec-first-workflow.md .codex/config.toml .codex/agents/challenger-agent.toml .claude/agents/challenger-agent.md skills/pre-spec-challenge/SKILL.md .agents/skills/pre-spec-challenge/SKILL.md .claude/skills/pre-spec-challenge/SKILL.md skills/spec-first-brainstorming/SKILL.md .agents/skills/spec-first-brainstorming/SKILL.md .claude/skills/spec-first-brainstorming/SKILL.md specs/pre-spec-challenge-workflow/spec.md`
- `git diff --stat -- AGENTS.md CLAUDE.md README.md docs/spec-first-workflow.md docs/skills/spec-first-brainstorming-spec.md .codex/config.toml .codex/agents/challenger-agent.toml .claude/agents/challenger-agent.md skills/pre-spec-challenge/SKILL.md .agents/skills/pre-spec-challenge/SKILL.md .claude/skills/pre-spec-challenge/SKILL.md skills/spec-first-brainstorming/SKILL.md .agents/skills/spec-first-brainstorming/SKILL.md .claude/skills/spec-first-brainstorming/SKILL.md specs/pre-spec-challenge-workflow/spec.md`
- `git status --short -- AGENTS.md CLAUDE.md README.md docs/spec-first-workflow.md docs/skills/spec-first-brainstorming-spec.md .codex/config.toml .codex/agents/challenger-agent.toml .claude/agents/challenger-agent.md skills/pre-spec-challenge/SKILL.md .agents/skills/pre-spec-challenge/SKILL.md .claude/skills/pre-spec-challenge/SKILL.md skills/spec-first-brainstorming/SKILL.md .agents/skills/spec-first-brainstorming/SKILL.md .claude/skills/spec-first-brainstorming/SKILL.md specs/pre-spec-challenge-workflow/spec.md`
- `git diff --check -- AGENTS.md CLAUDE.md README.md docs/spec-first-workflow.md docs/skills/spec-first-brainstorming-spec.md .codex/config.toml skills/spec-first-brainstorming/SKILL.md .agents/skills/spec-first-brainstorming/SKILL.md .claude/skills/spec-first-brainstorming/SKILL.md`
- `skills/pre-spec-challenge/evals/evals.json` plus `evals/files/*.md` created with three challenge scenarios: async export, cache migration, and admin deactivation
- blind A/B run executed in `skills/pre-spec-challenge-workspace/iteration-1/` against `variant-a` and `variant-b`, with blind comparator fan-in over all three evals
- blind comparison result: `variant-b` won `eval-1` and `eval-2`, `variant-a` won `eval-3`; overall winner `variant-b` by `2:1`
- canonical `skills/pre-spec-challenge/SKILL.md` replaced with the winning `variant-b` and mirrored copies re-synced to `.agents/skills/` and `.claude/skills/`
- canonical `pre-spec-challenge` eval fixtures were mirrored into `.agents/skills/pre-spec-challenge/evals/` and `.claude/skills/pre-spec-challenge/evals/` so neighboring skill surfaces now carry the same test inputs
- additional runtime skill mirrors discovered during follow-up review were synced too: `.cursor/skills/pre-spec-challenge/`, `.gemini/skills/pre-spec-challenge/`, `.github/skills/pre-spec-challenge/`, and `.opencode/skills/pre-spec-challenge/`, each including the same `SKILL.md` and `evals/` bundle

## Outcome

Completed repository integration for the risk-driven `pre-spec challenge` checkpoint:
- workflow contracts and overview docs now describe challenge as a checkpoint inside synthesis
- a new read-only `challenger-agent` exists for Codex and Claude project surfaces
- a reusable `pre-spec-challenge` skill exists in canonical and mirrored skill locations
- the agent/skill split is explicit: the agent owns role boundaries and the skill owns challenge behavior
- `spec-first-brainstorming` now routes to challenge when framing is clear but candidate decisions are still fragile
- the directly affected legacy design note now explicitly defers to the active runtime contract
- blind A/B evaluation of two prompt variants is complete; `variant-b` won `2:1` and is now the canonical skill text
- the promoted skill version tightens the question budget, makes falsification and question filtering more explicit, and keeps blocker/next-action guidance sharper
- mirror skill copies were re-synced after promotion so project runtimes now point at the same winning instructions
- neighboring skill mirrors now also include the same `evals/` bundle as the canonical skill, removing the remaining duplication gap
- follow-up audit corrected the runtime surface inventory and confirmed that `.cursor`, `.gemini`, `.github`, and `.opencode` now carry the same `pre-spec-challenge` skill bundle as the canonical directory
