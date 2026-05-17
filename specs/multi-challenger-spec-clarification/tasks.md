# Multi-Challenger Spec Clarification Tasks

## Implementation Handoff

Consumes: approved `spec.md` and this task ledger.
Implementation readiness: PASS.
First task: `T001`.
Accepted concerns: keep direct path and ordinary lean-local work lightweight; prove the multi-challenger default is trigger-scoped.
Reopen target: specification if implementation would make challenger lanes write-capable, decision-authoritative, mandatory for all lean work, or not lens-specific.

## Tasks

- [x] T001 [Phase 1] Update authority docs so formal spec clarification supports lens-based multi-challenger fan-out while preserving direct/lean/full routing and read-only advisory subagents.
  Files: `AGENTS.md`, `docs/spec-first-workflow.md`
  Proof: targeted read shows five-lens default is scoped to broad or multi-domain formal challenge, and new single-lane challenge requires scoped-down rationale.

- [x] T002 [Phase 1] Update subagent contract and brief templates so multi-challenger lanes have distinct lenses, one owned question, one skill or `no-skill`, explicit read-only boundaries, and orchestrator-owned fan-in.
  Depends on: `T001`
  Files: `docs/subagent-contract.md`, `docs/subagent-brief-template.md`
  Proof: targeted read shows the template can brief a shared candidate spec plus per-lane lenses without making subagents authoritative.

- [x] T003 [Phase 2] Update canonical specification/challenge skills and challenger agent routing so `spec-document-designer` and `specification-session` can route multi-lens clarification, `spec-clarification-challenge` can run as one lens in a fan-out, workflow adequacy checks require lane ownership and fan-in clarity, and `challenger-agent` respects assigned lenses.
  Depends on: `T001`, `T002`
  Files: `.agents/skills/spec-document-designer/SKILL.md`, `.agents/skills/specification-session/SKILL.md`, `.agents/skills/specification-session/references/spec-clarification-gate-flow.md`, `.agents/skills/spec-clarification-challenge/SKILL.md`, `.agents/skills/workflow-plan-adequacy-challenge/SKILL.md`, `.codex/agents/challenger-agent.toml`
  Proof: targeted read shows formal broad specs no longer imply exactly one challenger lane, each lane still uses one skill, and `Lens` remains metadata rather than a new classification vocabulary.

- [x] T004 [Phase 2] Sync runtime skill and agent mirrors from canonical `.agents/skills` and `.codex/agents`.
  Depends on: `T003`
  Proof: `rtk make skills-sync`, `rtk make agents-sync`; follow-up direct sync scripts aligned generated mirrors after concurrent source-managed instruction drift was detected.

- [x] T005 [Phase 3] Validate the docs/skills change and update closeout artifacts.
  Depends on: `T001`, `T002`, `T003`, `T004`
  Proof: `rtk git status --short`, `rtk make agents-check`, `rtk make skills-check`, final `sync-skills.sh --check` stability loop, `rtk git diff --check`; closeout artifacts updated after fresh evidence.
