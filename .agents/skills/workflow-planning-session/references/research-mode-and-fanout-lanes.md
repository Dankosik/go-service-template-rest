# Research Mode And Fan-Out Lanes

## Behavior Change Thesis
When loaded for symptom "I need to choose local research versus fan-out or write lane rows," this file makes the model plan read-only lanes by evidence question and one skill per lane instead of creating broad owner lanes, worker lanes, multi-skill lanes, or spawning research during workflow planning.

## When To Load
Load this when the execution shape already points toward a research-mode decision or a lane table. Do not load it just to choose direct/lightweight/full, and do not execute any lane from this reference.

## Decision Rubric
- Use `local` when a bounded read of task artifacts and nearby code can answer the next questions without losing important domain separation.
- Use `fan-out` when independent domain evidence would reduce risk: API, data, security, reliability, QA, performance, delivery, domain invariants, or architecture seams.
- A lane must have one role, one owned question family, one skill or explicit `no-skill`, and a read-only expected output.
- Duplicate roles are acceptable when the owned question differs; mixed domains in one lane are not.
- The adequacy challenger is a workflow-control lane, not a research lane; route it after draft workflow artifacts exist.
- The workflow-planning session records lanes for the next research session; it does not spawn them.

## Imitate

Local research handoff:

```markdown
Research mode: local in the next session
Reason: one bounded package surface; no visible API, data, auth, reliability, or rollout seam yet.
Planned lanes: none initially.
Escalation: if local reading finds public contract drift, persistent state, tenant isolation, auth policy, goroutine lifecycle, or rollout risk, stop and repair workflow planning before wider research.
```

What to copy: no fake lane table when local research is enough, but the escalation seam is explicit.

Fan-out lane table:

```markdown
| Lane | Role | Owned Question | Skill | Expected Output |
| --- | --- | --- | --- | --- |
| L1 | `api-agent` | What REST resources, status semantics, filtering, and error-contract questions must be answered before spec approval? | `api-contract-designer-spec` | API risks and decision options only; no OpenAPI edits. |
| L2 | `data-agent` | What source-of-truth, migration, tenant-isolation, and retention questions must be answered before spec approval? | `go-data-architect-spec` | Data decision options and blockers only; no migration files. |
| L3 | `security-agent` | What auth, trust-boundary, signed URL, and abuse-resistance seams affect scope or validation? | `go-security-spec` | Security questions and constraints only; no task breakdown. |
```

What to copy: each lane returns evidence for synthesis; none owns the final decision.

## Reject

```markdown
| Lx | `data-agent` | Own every export question and downstream planning decision. | `go-data-architect-spec` |
```

Failure: too broad; the lane owns planning instead of evidence.

```markdown
| Ly | `security-agent` | Security review, API design, data schema, and QA. | `go-security-spec` |
```

Failure: multiple domains collapsed into one lane and one skill.

```markdown
| Lz | `worker` | Implement Phase 1 after research. | `go-coder` |
```

Failure: write-capable implementation work is prohibited in workflow planning.

```markdown
| La | `api-agent` | Use `api-contract-designer-spec` and `go-reliability-spec` together. | multiple |
```

Failure: violates one-skill-per-pass; split it or keep synthesis local.

## Agent Traps
- Treating "fan-out planned" as permission to spawn lanes before the workflow-planning session closes.
- Asking one lane to own a whole feature because the agent role sounds senior.
- Forgetting that role reuse is fine when questions differ.
- Mixing the adequacy challenge with domain research, spec clarification, or review.
