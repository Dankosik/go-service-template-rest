# Subagent Contract

Shared contract for repository read-only subagents. `AGENTS.md` remains authoritative; this file keeps the repeated per-agent operational envelope in one place.

## Shared Invariants

- Subagents are advisory and read-only: no code writes, file edits, git-state mutation, or implementation-plan changes.
- Final decisions, synthesis, implementation, reconciliation, and validation belong to the orchestrator.
- Each pass uses at most one skill. If a selected skill defines a procedure or output shape, the skill owns it.
- Agent files own domain scope, use/do-not-use rules, inspect-first surfaces, skill routing, and unique escalation rules.
- Do not invent missing artifacts, source facts, policy decisions, diffs, validation output, or skill results.
- If input is insufficient, return `Missing input`, `Why it blocks`, and `Smallest artifact/evidence needed`.
- If a bounded assumption is safe enough, label it and proceed.

## Required Input Bundle

Every handoff should include:

- goal and exact question,
- expected mode: research, review, adjudication, or challenge,
- current workflow phase and task-local artifact paths when present,
- relevant diff, source files, source-of-truth documents, or specialist outputs to inspect,
- constraints, risk hotspots, non-goals, and known blockers,
- chosen skill name or `no-skill`,
- explicit read-only boundary.

## Fan-In Envelope

When the chosen skill does not define a stricter shape, return:

- `Decision or findings`: the role-specific conclusion, recommendation, blocker call, or ordered findings.
- `Evidence`: tight references to files, artifacts, commands, contracts, or source facts.
- `Open risks/gaps`: unresolved assumptions, compatibility, ownership, test, validation, or rollout risks.
- `Recommended handoff`: one smallest next action with target owner or artifact.
- `Confidence`: high, medium, or low with the key uncertainty.

Recommended handoff classifications:

- `spawn_agent`
- `reopen_phase`
- `needs_user_decision`
- `accept_risk`
- `record_only`
- `no_action`

Pair the classification with the target owner or artifact and the smallest next step.

## Escalation Rules

Escalate instead of stretching the lane when:

- the decisive fact belongs to a different domain owner,
- the answer would require another skill in the same pass,
- the approved artifact bundle is missing or contradictory,
- a local review exposes a spec/design/planning gap,
- a user or product policy decision is required,
- the requested work would require edits or git mutation.

## Brief Quality Bar

Good subagent briefs are narrow, evidence-oriented, and explicit about output. Start from `docs/subagent-brief-template.md` when the lane is not trivial.
