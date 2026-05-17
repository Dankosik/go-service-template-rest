# Technical Design Review Stage Tasks

## Implementation Handoff

Consumes: approved `spec.md` and this task ledger.
Implementation readiness: PASS.
First task: `T001`.
Accepted concerns: keep direct path and compact lean-local `spec.md` lightweight; prove the new mandatory gate is scoped to separate technical design depth.
Reopen target: specification if implementation would make subagents write-capable, make design reviewers final authority, or require formal lanes for every compact lean-local design answer.

## Tasks

- [x] T001 [Phase 1] Update authority workflow docs so technical design review is a mandatory pre-planning gate whenever separate design depth is triggered.
  Files: `AGENTS.md`, `docs/spec-first-workflow.md`
  Proof: targeted read shows direct path and compact lean-local design stay lightweight, while separate `design/overview.md` or split `design/` cannot route directly to planning.

- [x] T002 [Phase 1] Update subagent contract and brief templates so technical design review lanes have explicit scope, evidence, read-only boundaries, and orchestrator-owned reconciliation.
  Depends on: `T001`
  Files: `docs/subagent-contract.md`, `docs/subagent-brief-template.md`
  Proof: targeted read shows technical design review is distinct from post-code review and does not make lanes decision-authoritative.

- [x] T003 [Phase 2] Update canonical workflow/design/planning/status surfaces so technical design produces review-ready output, planning requires a reconciled review gate, and status reports missing review as a blocker.
  Depends on: `T001`, `T002`
  Files: `.agents/skills/technical-design-session/SKILL.md`, `.agents/skills/planning-session/SKILL.md`, `.agents/skills/planning-and-task-breakdown/SKILL.md`, `.agents/skills/go-design-spec/SKILL.md`, `.agents/skills/workflow-status/SKILL.md`, `.codex/agents/design-integrator-agent.toml`
  Proof: targeted read shows the required phase order is technical design -> technical design review -> planning.

- [x] T004 [Phase 2] Sync runtime skill and agent mirrors from canonical sources.
  Depends on: `T003`
  Proof: `rtk make agents-sync`, `rtk make skills-sync`

- [x] T005 [Phase 3] Validate and close the amendment.
  Depends on: `T001`, `T002`, `T003`, `T004`
  Proof: `rtk make agents-check`, `rtk make skills-check`, `rtk git diff --check`; closeout artifacts updated after fresh evidence.

- [x] T006 [Phase 3] Tighten the technical-design-review instructions so agents receive a concrete review packet, leave a review record, classify findings by planning impact, and carry `CONCERNS` into planning proof obligations.
  Depends on: `T005`
  Files: `AGENTS.md`, `docs/spec-first-workflow.md`, `docs/subagent-contract.md`, `docs/subagent-brief-template.md`, `.agents/skills/technical-design-session/SKILL.md`, `.agents/skills/planning-session/SKILL.md`, `.agents/skills/planning-and-task-breakdown/SKILL.md`, `.agents/skills/go-design-spec/SKILL.md`, `.codex/agents/design-integrator-agent.toml`
  Proof: `rtk make agents-sync`, `rtk make skills-sync`, `rtk make agents-check`, `rtk make skills-check`, `rtk git diff --check`
