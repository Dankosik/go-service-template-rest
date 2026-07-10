# Subagent Brief Template

Use this compact brief after the root has applied the fan-out decision in `AGENTS.md`. Do not repeat the shared contract from `docs/subagent-contract.md`.

## Default Brief

```text
Lane: <id and agent>
Question: <one concrete independent bounded question>
Why separate context helps: <speed or quality reason>
Mode: <research | review | adjudication | challenge>
Inspect first: <smallest artifact/source list>
Evidence boundary: <what counts; relevant constraints and non-goals>
Skill: <one skill | no-skill>
Model route: <agent profile; exact selected model/reasoning effort; complexity rationale; enforcing launch surface>
Return: <specialist-specific fields plus compact conclusion, evidence, open gap, escalation>
Read-only enforcement: <actual read-only execution choice>
```

If the question cannot be stated independently, keep it in the root flow. If two briefs ask materially the same question, merge them.

## Approval Or Review Gate Variant

Use only when the lane can change a specification, technical-design, task-review, or validation verdict.

```text
Lane: <id and agent>
Question: <one approval-critical falsification question>
Reviewed artifact: <path, exact revision/content anchor, current routing identity when durable, and review-cycle attempt>
Prior finding closure: <finding ids and repair/evidence anchors to recheck | first review>
Inspect first: <artifact plus named evidence sources>
Skill: <one review/challenge skill | no-skill>
Model route: <agent profile; exact selected model/reasoning effort; approval-risk rationale; enforcing launch surface>
Finding format: <artifact anchor; evidence; impact; classification; owner/reopen target; why not stronger/weaker>
Allowed recommendation: <gate-specific verdict vocabulary>
Return: <findings-first compact result; fresh-context attestation; no artifact edits, repair, or approval claim>
Read-only enforcement: <actual read-only execution choice>
```

The root owns the final verdict and records `procedural_gate_state`, `review_verdict`, `record_validity`, accepted risks, proof obligations, and reopen routing.

## Workflow Adequacy Challenge Variant

Use only for a triggered `workflow-plan-adequacy-challenge`, because routing identity can change the finding.

```text
Question: Can the recorded route and handoff be falsified against the canonical rule IDs?
Inspect first: <accepted brief; FULL/DIRECT/LEAN audit; matched SHAPE rule; agent_request; routing scope/revision/validity; transition record; artifact consequences; handoff>
Skill: workflow-plan-adequacy-challenge
Return: <falsification findings, evidence, affected rule/state, readiness consequence, reopen target>
Read-only enforcement: <actual read-only execution choice>
```
