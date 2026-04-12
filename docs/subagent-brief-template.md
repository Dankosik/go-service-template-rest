# Subagent Brief Template

Use this template when the orchestrator asks a read-only specialist lane for research, review, adjudication, or challenge. Fill only the sections that matter; keep the brief compact.

```text
Goal:
- <one sentence describing what the lane must decide or check>

Scope:
- Agent: <agent-name>
- Mode: <research | review | adjudication | challenge>
- Skill: <skill-name | no-skill>
- Read-only boundary: do not edit files, mutate git state, or change implementation plans.

Context:
- Workflow phase: <phase or "none">
- Task artifacts: <paths to workflow-plan.md/spec.md/design/plan/tasks when present>
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
- Recommended handoff must use one classification:
  spawn_agent, reopen_phase, needs_user_decision, accept_risk, record_only, or no_action.
```

Short variant:

```text
Use <agent-name> in <mode> with <skill-name | no-skill>.
Read-only: no edits, no git mutation, no implementation-plan changes.
Question: <exact question>.
Inspect first: <paths>.
Evidence: cite concrete files/artifacts/commands; label assumptions.
Return: skill output shape, or docs/subagent-contract.md envelope with one handoff classification.
```
