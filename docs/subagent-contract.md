# Subagent Contract

Shared contract for repository read-only subagents. `AGENTS.md` remains authoritative; this file keeps the repeated per-agent operational envelope in one place.

## Shared Invariants

- Subagents are advisory and read-only: no code writes, file edits, git-state mutation, or implementation-plan changes.
- Final decisions, synthesis, implementation, reconciliation, and validation belong to the orchestrator.
- Each pass uses at most one skill. If a selected skill defines a procedure or output shape, the skill owns it.
- Agent files own domain scope, use/do-not-use rules, inspect-first surfaces, skill routing, and unique escalation rules.
- Deep design and corner-case coverage stay in scope, but downstream effect alone does not create a new required domain decision.
- Open another lane only when another domain must make a new decision before the current artifact can be high quality; otherwise return the consequence as a constraint, proof obligation, follow-up, or explicit `no new decision required` note.
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

When a downstream domain is touched, strongly prefer classifying each major point with one of:

- `must_decide_now`: another domain must make a new decision before the current artifact can be high quality.
- `constraint_only`: the current decision stands, but later work must preserve a concrete constraint in that domain.
- `proof_only`: no new decision is required now, but implementation, review, or validation must prove something in that domain.
- `follow_up_only`: the effect is real but not planning-critical for the current artifact; revisit only if later work reaches that seam.

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

- the decisive fact or required new decision belongs to a different domain owner,
- the answer would require another skill in the same pass,
- the approved artifact bundle is missing or contradictory,
- a local review exposes a spec/design/planning gap,
- a user or product policy decision is required,
- the requested work would require edits or git mutation.

Do not escalate only because another domain is affected. If that domain does not need to decide now, keep the answer local and return the consequence classification instead.

## Brief Quality Bar

Good subagent briefs are narrow, evidence-oriented, explicit about output, and centered on one owned question instead of a parallel cross-domain design package. Start from `docs/subagent-brief-template.md` when the lane is not trivial.
