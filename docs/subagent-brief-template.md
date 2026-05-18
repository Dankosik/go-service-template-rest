# Subagent Brief Template

Use this template when the orchestrator asks a read-only specialist lane for research, review, adjudication, or challenge. Open a lane only for an unresolved owned question; do not use the template to make subagents default ceremony. Fill only the sections that matter and keep the brief compact.

```text
Goal:
- <one sentence describing what the lane must decide or check>

Scope:
- Agent: <agent-name>
- Mode: <research | review | adjudication | challenge>
- Skill: <skill-name | no-skill>
- Lens or specialist domain: <required for multi-lane fan-out; omit only when not applicable>
- Read-only boundary: do not edit files, mutate git state, or change task ledgers or implementation handoffs.

Context:
- Workflow phase: <phase or "none">
- Task artifacts: <paths to workflow-plan.md/spec.md/design/tasks and any triggered test-plan.md or rollout.md when relevant>
- Source-of-truth inputs: <contracts, docs, diffs, files, commands, specialist outputs>
- Constraints and non-goals: <short list>
- Known blockers or assumptions: <short list>

Inspect first:
- <small ordered list of files, directories, docs, or diffs>

Question:
- <the exact question this lane owns>

Evidence requirement:
- Cite exact files, artifact sections, commands, or source facts.
- Separate facts from assumptions and inferences.
- Do not invent missing artifacts or validation results.

Expected output:
- If the chosen skill defines an output shape, follow that shape.
- Otherwise use the shared envelope from docs/subagent-contract.md:
  Decision or findings / Evidence / Open risks or gaps / Recommended handoff / Confidence.
- When adjacent domains are touched, prefer classifying each major point as `must_decide_now`, `constraint_only`, `proof_only`, or `follow_up_only`.
- Recommended handoff must use one classification:
  spawn_agent, reopen_phase, needs_user_decision, accept_risk, record_only, or no_action.
```

Short variant:

```text
Use <agent-name> in <mode> with <skill-name | no-skill>.
Read-only: no edits, no git mutation, no task-ledger or handoff changes.
Question: <exact question>.
Inspect first: <paths>.
Evidence: cite concrete files/artifacts/commands; label assumptions.
Return: skill output shape, or docs/subagent-contract.md envelope with one handoff classification.
Prefer `must_decide_now` / `constraint_only` / `proof_only` / `follow_up_only` for adjacent-domain effects when relevant.
```

Short challenge/review variant:

```text
Use <agent-name> for read-only <challenge | review> with <skill-name | no-skill>.
Scope: <artifact, diff, or decision being challenged/reviewed>.
Lens: <specific approval-risk lens or specialist domain when part of multi-lane fan-out>.
Question: <one approval-, risk-, or correctness-critical question>.
Inspect first: <small path list>.
Evidence: cite exact files/artifacts/commands; separate facts from assumptions.
Return: findings only, ordered by impact, with one recommended handoff classification.
Do not edit files, mutate git state, approve decisions, or change task ledgers/handoffs.
```

Technical design review variant:

```text
Use <design-integrator-agent | specialist-agent> for read-only technical design review with <go-design-spec | specialist skill | no-skill>.
Gate: technical design review before planning.
Scope: <design bundle or design artifact set being reviewed>.
Question: Is this technical design coherent and safe enough for planning, or must technical design/specification reopen?
Inspect first:
- <task>/spec.md
- <task>/design/overview.md
- <task>/design/<triggered artifacts>
- <task>/workflow-plan.md and <task>/workflow-plans/technical-design*.md when present
- docs/repo-architecture.md when boundaries, ownership, dependency direction, or runtime flow matter
Evidence: cite concrete artifact sections and source facts; label assumptions.
Decision quality: for each material finding, state the planning decision at risk, the strongest counterargument or simpler alternative considered, and why the severity/gate result is not stronger or weaker.
Return: Findings classified as `blocks_planning`, `reopens_design`, `reopens_spec`, `accepted_risk_candidate`, `proof_obligation`, or `record_only`; required fixes or reopen targets; accepted-risk candidates; planning proof obligations; recommended gate result: PASS | CONCERNS | FAIL with status rationale.
Read-only: no edits, no git mutation, no approval authority, no task-ledger or implementation handoff changes.
```

Multi-challenger clarification variant:

```text
Shared candidate bundle:
- Workflow phase: <specification phase or equivalent>
- Candidate spec: <path>
- Related artifacts: <workflow-plan/design/research/tasks paths if relevant>
- Shared constraints/non-goals: <short list>
- Sibling lenses: <names of the other planned challenge lenses, so this lane avoids duplicate coverage>
- Fan-in owner: orchestrator reconciles all lane outputs; lanes do not approve decisions.
- Scoped-down rationale: <required when using fewer lanes than the broad formal default; omit for full lens set>

Lane:
- Agent: challenger-agent
- Mode: challenge
- Skill: spec-clarification-challenge
- Lens: <scope/spec coherence | domain invariants | architecture ownership | API/data/compatibility | security/reliability/delivery/validation | custom lens>
- Question: <one concrete approval-critical question for this lens; not a generic topic label>
- Inspect first: <small path list, plus lens-specific files only>
- Evidence: cite exact files/artifacts/commands; separate facts from assumptions.
- Return: spec-clarification-challenge shape, with the lens named and only the strongest non-duplicative findings/questions ordered by approval impact.
- Read-only: no edits, no git mutation, no task-ledger or handoff changes.
```
