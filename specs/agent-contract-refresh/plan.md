# Agent Contract Refresh Plan

Created: 2026-04-12
Status: approved for implementation handoff

## Execution Context

This plan consumes the approved `spec.md` and approved `design/` bundle for the instruction-only agent contract refresh.

Implementation should run as small, sequential sessions. The work changes project-scoped agent instructions and README inventory only; it must not touch Go runtime behavior, service contracts, migrations, generated code, or skill bodies unless the task is reopened upstream.

No dedicated `test-plan.md` or `rollout.md` is expected. Validation fits in the phase checkpoints below and the final validation phase.

No review phase-control file is created in this plan. Each implementation phase has local proof obligations, and a later dedicated review phase should be opened only by returning to planning if the user wants separate read-only review fan-out.

## Phase Plan

### Phase 1: Challenger Three-Mode Contract

Objective: fix the blocking runtime-contract drift for `challenger-agent`.

Depends On: approved `spec.md` and `design/`.

Task Ledger Link / IDs: T001-T004.

Acceptance Criteria:

- `.codex/agents/challenger-agent.toml` and `.claude/agents/challenger-agent.md` both describe `workflow-plan-adequacy-challenge`, `pre-spec-challenge`, and `spec-clarification-challenge`.
- The challenger role remains read-only, advisory, non-review, and one-skill-per-pass.
- Each challenge mode has clear use, inspect-first, return, and escalation guidance.

Change Surface:

- `.codex/agents/challenger-agent.toml`
- `.claude/agents/challenger-agent.md`
- README challenger wording only if the existing row conflicts with the fixed runtime contract.

Planned Verification:

- TOML parse check for `.codex/agents/challenger-agent.toml`.
- `rg` check for all three challenge skill names in both challenger runtime files.
- `rtk git diff --check`.

Review / Checkpoint: stop after this phase if the three-mode contract cannot be expressed equivalently in both runtime formats.

Exit Criteria: the highest-priority drift is fixed before broader portfolio cleanup begins.

### Phase 2: Observability Mirror And README Inventory

Objective: resolve the Codex/Claude mirror and README inventory drift around `observability-agent`.

Depends On: Phase 1.

Task Ledger Link / IDs: T010-T013.

Acceptance Criteria:

- `.claude/agents/observability-agent.md` exists unless implementation proves the Codex-only state is intentional.
- The Claude observability agent preserves the role semantics of `.codex/agents/observability-agent.toml`.
- README project-scoped agent inventory includes `observability-agent` when the Claude mirror exists.

Change Surface:

- `.codex/agents/observability-agent.toml` as source context only unless a small parity fix is needed.
- `.claude/agents/observability-agent.md`
- `README.md`

Planned Verification:

- File existence check for `.claude/agents/observability-agent.md`.
- README link check for `.claude/agents/observability-agent.md`.
- Agent inventory comparison between `.codex/agents` and `.claude/agents`.
- TOML parse check for Codex agent files if any Codex TOML changed.

Review / Checkpoint: reopen specification if implementation proves `observability-agent` is intentionally Codex-only.

Exit Criteria: runtime inventories and README no longer contradict the mirror policy approved for this task.

### Phase 3: Return Contracts For Review-Focused Agents

Objective: standardize fan-in return contracts for agents whose normal mode includes code or proof review.

Depends On: Phase 2.

Task Ledger Link / IDs: T020-T023.

Acceptance Criteria:

- Review-focused agent pairs use the agreed review return contract: `Findings by severity`, `Evidence`, `Why it matters`, `Validation gap`, `Handoff`, and `Confidence`.
- Role-specific return fields are kept only when they improve fan-in for that role.
- Style-only review guidance is not introduced unless it is tied to merge risk.

Change Surface:

- `.codex/agents/concurrency-agent.toml`
- `.codex/agents/data-agent.toml`
- `.codex/agents/domain-agent.toml`
- `.codex/agents/performance-agent.toml`
- `.codex/agents/qa-agent.toml`
- `.codex/agents/quality-agent.toml`
- `.codex/agents/reliability-agent.toml`
- `.codex/agents/security-agent.toml`
- Matching `.claude/agents/*.md` files for the same roles.

Planned Verification:

- `rg` check for the review return field names across the phase file set.
- TOML parse check for touched `.codex/agents/*.toml`.
- `rtk git diff --check`.

Review / Checkpoint: stop if a role cannot use the shared review shape without losing a role-specific safety boundary.

Exit Criteria: the main review-oriented agent lanes have predictable fan-in output.

### Phase 4: Return Contracts For Advisory And Mixed-Mode Agents

Objective: standardize fan-in return contracts for research/adjudication-only and mixed-mode advisory agents.

Depends On: Phase 3.

Task Ledger Link / IDs: T030-T033.

Acceptance Criteria:

- Advisory and mixed-mode agent pairs use the agreed research/adjudication return contract: `Conclusion`, `Evidence`, `Open risks`, `Recommended handoff`, and `Confidence`.
- Mixed-mode roles that also support targeted review keep their review nuance without becoming generic reviewers.
- `delivery-agent`, `distributed-agent`, and `observability-agent` continue to state their non-default-review limitations.

Change Surface:

- `.codex/agents/api-agent.toml`
- `.codex/agents/architecture-agent.toml`
- `.codex/agents/challenger-agent.toml`
- `.codex/agents/delivery-agent.toml`
- `.codex/agents/design-integrator-agent.toml`
- `.codex/agents/distributed-agent.toml`
- `.codex/agents/observability-agent.toml`
- Matching `.claude/agents/*.md` files for the same roles, including `.claude/agents/observability-agent.md` after Phase 2.

Planned Verification:

- `rg` check for the research/adjudication return field names across the phase file set.
- TOML parse check for touched `.codex/agents/*.toml`.
- `rtk git diff --check`.

Review / Checkpoint: stop if a mixed-mode role needs a missing review skill instead of a return-format edit.

Exit Criteria: all agent roles now expose predictable evidence-anchored output for orchestrator fan-in.

### Phase 5: Inspect-First Blocks For Runtime And Domain Roles

Objective: add concise source-of-truth lookup guidance to roles closest to service behavior and runtime risk.

Depends On: Phase 4.

Task Ledger Link / IDs: T040-T043.

Acceptance Criteria:

- Each role in this phase has a short `Inspect first` block with 3-6 repository surfaces or task artifacts.
- The blocks point to sources of truth and nearby consumers, not broad repo-scanning instructions.
- Codex and Claude mirrors preserve equivalent lookup intent.

Change Surface:

- `api-agent`
- `concurrency-agent`
- `data-agent`
- `domain-agent`
- `observability-agent`
- `performance-agent`
- `reliability-agent`
- `security-agent`
- Both `.codex/agents/*.toml` and `.claude/agents/*.md` runtime files for those roles.

Planned Verification:

- `rg` check that each touched role has an `Inspect first` section.
- Manual spot check that file/path suggestions match `docs/repo-architecture.md` and the approved spec examples.
- TOML parse check for touched Codex TOML.

Review / Checkpoint: stop if a role's inspect-first list would require a missing design decision about repository ownership.

Exit Criteria: runtime and domain roles have bounded, role-specific starting surfaces.

### Phase 6: Inspect-First Blocks For Workflow And Meta Roles

Objective: add concise source-of-truth lookup guidance to workflow, planning, integration, and proof-strategy roles.

Depends On: Phase 5.

Task Ledger Link / IDs: T050-T053.

Acceptance Criteria:

- Each role in this phase has a short `Inspect first` block with role-specific artifacts or repo surfaces.
- `challenger-agent` inspect-first guidance distinguishes the three challenge modes instead of forcing one generic scan path.
- Codex and Claude mirrors preserve equivalent lookup intent.

Change Surface:

- `architecture-agent`
- `challenger-agent`
- `delivery-agent`
- `design-integrator-agent`
- `distributed-agent`
- `qa-agent`
- `quality-agent`
- Both `.codex/agents/*.toml` and `.claude/agents/*.md` runtime files for those roles.

Planned Verification:

- `rg` check that each touched role has an `Inspect first` section.
- TOML parse check for touched Codex TOML.
- `rtk git diff --check`.

Review / Checkpoint: stop if the challenger inspect-first split exposes missing policy for a challenge gate.

Exit Criteria: all project-scoped agents have bounded lookup guidance.

### Phase 7: Safe Deduplication And Drift-Policy Checkpoint

Objective: reduce repeated global policy only after role-local contracts are explicit, and leave broader mirror tooling as a separate decision.

Depends On: Phase 6.

Task Ledger Link / IDs: T060-T064.

Acceptance Criteria:

- Repeated global policy is trimmed only where the role still has clear mission, boundaries, mode routing, return contract, and escalation rules.
- Agent files still preserve read-only, advisory, and one-skill-per-pass safety.
- README and runtime inventory remain consistent.
- Canonical-source generation, CI drift checks, new review skills, nickname additions, and model tuning are not silently implemented in this task.

Change Surface:

- `.codex/agents/*.toml`
- `.claude/agents/*.md`
- `README.md` only if the drift-policy checkpoint needs a short existing-section note.

Planned Verification:

- TOML parse check for all `.codex/agents/*.toml` and `.codex/config.toml`.
- Inventory comparison between `.codex/agents` and `.claude/agents`.
- `rg` checks that read-only/advisory/one-skill-per-pass guardrails still exist.
- `rtk git diff --check`.

Review / Checkpoint: reopen specification if the user wants canonical-source generation, CI drift checks, new review skills, or model/reasoning policy in the same task cycle.

Exit Criteria: the portfolio cleanup is complete without broadening into separate tooling or skill-design work.

### Validation Phase 1: Final Proof And Closeout

Objective: prove the instruction-only refresh is complete and update existing closeout surfaces.

Depends On: Phase 7.

Task Ledger Link / IDs: T900-T903.

Acceptance Criteria:

- All planned validation checks pass or any failure is recorded with a reopen target.
- `spec.md` `Validation` / `Outcome` are updated with fresh evidence.
- `workflow-plan.md`, `tasks.md`, and `workflow-plans/validation-phase-1.md` reflect final status.

Change Surface:

- Existing task-local control and closeout artifacts only.

Planned Verification:

- Full TOML parse check for `.codex/config.toml` and `.codex/agents/*.toml`.
- Inventory/link checks for `.codex/agents`, `.claude/agents`, and README.
- Return-contract and inspect-first `rg` checks.
- `rtk git diff --check`.

Review / Checkpoint: if validation exposes a missing spec or design decision, stop and reopen the named earlier phase.

Exit Criteria: the task can be marked done with fresh validation evidence.

## Cross-Phase Validation Plan

Each implementation phase should run focused proof for its touched files before moving to the next phase. The final validation phase should then run the cross-portfolio checks once across all agent files.

Recommended final proof commands:

```bash
rtk python3 -c 'import pathlib,tomllib; tomllib.loads(pathlib.Path(".codex/config.toml").read_text()); [tomllib.loads(p.read_text()) for p in pathlib.Path(".codex/agents").glob("*.toml")]'
rtk rg -n "workflow-plan-adequacy-challenge|pre-spec-challenge|spec-clarification-challenge" .codex/agents/challenger-agent.toml .claude/agents/challenger-agent.md
rtk test -f .claude/agents/observability-agent.md
rtk rg -n "observability-agent" README.md .claude/agents/observability-agent.md .codex/config.toml
rtk rg -n "Conclusion|Evidence|Open risks|Recommended handoff|Confidence|Findings by severity|Why it matters|Validation gap|Handoff" .codex/agents .claude/agents
rtk rg -n "Inspect first" .codex/agents .claude/agents
rtk git diff --check
```

No Go test command is expected while the task remains instruction-only.

## Implementation Readiness

Status: PASS.

Gate result: implementation may start with `implementation-phase-1` in a later session.

Proof path: task-level proof is listed in `tasks.md`; phase checkpoint proof is listed in this `plan.md`; final closeout proof is routed through `workflow-plans/validation-phase-1.md`.

Adequacy challenge status: waived by lightweight-local exception for this planning handoff. Rationale: the task remains instruction-only, approved design inputs exist, no subagent fan-out was requested for this planning session, and the first implementation phase is narrow and reversible. Reopen planning and run the challenge if the work expands into generated mirror tooling, CI drift checks, new skills, model policy, or separate review fan-out.

## Blockers / Assumptions

No active blockers.

Assumptions carried from `spec.md`:

- The observed `observability-agent` Codex-only state is accidental drift.
- The current task cycle remains instruction-only.
- Codex and Claude runtime files can preserve equivalent role semantics with hand-maintained mirrors.

## Handoffs / Reopen Conditions

Next session starts with: `implementation-phase-1`.

Reopen specification or technical design before continuing if:

- `observability-agent` is intentionally Codex-only.
- Codex and Claude runtime formats cannot preserve equivalent semantics with hand-maintained mirrors.
- The user wants canonical-source generation, CI drift checks, new review skills, model/reasoning policy, or workflow-document rewrites in this same task cycle.
- Implementation finds that `AGENTS.md`, `docs/spec-first-workflow.md`, or skill bodies must change to preserve the approved agent contract.
