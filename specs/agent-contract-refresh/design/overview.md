# Agent Contract Refresh Technical Design

Status: approved for planning handoff

## Design Intent

Refresh the repository's project-scoped subagent instruction contracts without changing Go service runtime behavior.

The design keeps the work as an instruction/configuration refresh across these surfaces:

- Codex runtime agent files under `.codex/agents/*.toml`.
- Claude runtime agent files under `.claude/agents/*.md`.
- Agent inventory and usage documentation in `README.md`.
- Existing workflow authority in `AGENTS.md`, `docs/spec-first-workflow.md`, and `.agents/skills/*/SKILL.md` as source context, not as broad rewrite targets.

The first implementation priority remains the concrete `challenger-agent` contract drift. Later slices should standardize shared agent shape, add concise source-of-truth guidance, and reduce duplicated global policy only after the role-local contracts are explicit.

## Chosen Approach

Use incremental, semantics-preserving edits rather than a whole-portfolio rewrite:

1. Fix `challenger-agent` in both runtime formats so it explicitly supports `workflow-plan-adequacy-challenge`, `pre-spec-challenge`, and `spec-clarification-challenge`.
2. Repair inventory drift by adding the missing Claude `observability-agent` mirror and README table row unless implementation proves the Codex-only state is intentional.
3. Standardize return contracts and `Inspect first` blocks across agents in controlled follow-up slices.
4. Deduplicate repeated global policy only after the stricter role-local contracts are present.
5. Keep canonical-source generation, CI drift checks, missing review skills, nicknames, and model tuning as separate checkpoint or backlog work unless a later planning session explicitly reopens scope.

## Artifact Index

- `design/component-map.md`: affected files, stable files, and conditional artifacts.
- `design/sequence.md`: intended implementation order, validation checkpoints, and reopen triggers.
- `design/ownership-map.md`: source-of-truth boundaries and mirror semantics.

No conditional design artifacts are expected:

- `design/data-model.md`: not expected; no persisted state, schema, cache, projection, or migration changes.
- `design/dependency-graph.md`: not expected; no Go package or module dependency shape changes.
- `design/contracts/`: not expected; this task changes agent instructions and documentation, not API/event/generated/runtime subsystem contracts.
- `test-plan.md`: not expected; validation fits inside later `plan.md`.
- `rollout.md`: not expected; no deployment, migration, or mixed-version runtime choreography.

## Planning-Ready Summary

Planning can now decompose the work into small sessions around the approved slices:

- challenger three-mode contract fix,
- observability Claude mirror and README inventory repair,
- shared return-format standardization,
- role-specific `Inspect first` blocks,
- safe global-policy deduplication,
- drift-policy checkpoint,
- optional ergonomics/backlog checkpoint.

Planning must preserve the instruction-only constraint and avoid generating code, service tests, migrations, or Go runtime changes.

## Reopen Conditions

Return to specification or technical design before planning continues if any of these become true:

- `observability-agent` is intentionally Codex-only.
- Codex and Claude runtime formats cannot preserve equivalent semantics with hand-maintained mirrors.
- The user wants canonical-source generation, CI drift checks, new review skills, or model/reasoning policy in this same task cycle.
- Implementation finds that `AGENTS.md` or `docs/spec-first-workflow.md` must change to preserve the agent contract.
