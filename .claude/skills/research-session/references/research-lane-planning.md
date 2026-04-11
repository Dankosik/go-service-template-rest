# Research Lane Planning

## Behavior Change Thesis
When loaded for a chosen or likely fan-out session whose lane plan is vague, this file makes the model assign one owned question, one role, one skill, one evidence target, and a fan-in path per lane instead of likely mistake: broad domain labels, multi-skill lanes, write-capable workers, or subagents asked to approve decisions.

## When To Load
Load when `workflow-plans/research.md` needs clearer lane ownership, role choice, evidence targets, order, parallelism, or fan-in mechanics.

Do not load to invent lanes for tiny local research. Use `local-vs-fanout-mode-selection.md` first if the mode itself is still undecided.

## Decision Rubric
A useful research lane names:
- lane ID
- execution type: local or read-only subagent
- role
- exactly one chosen skill, or explicit `no-skill`
- one owned question
- evidence target or source surface
- order or parallelism
- status

Keep fan-in local to the orchestrator. A lane can return evidence, conflicts, weak points, and handoff implications; it cannot edit files, approve `spec.md`, create design artifacts, or write a plan.

## Imitate
```markdown
## Research Lanes

Research mode: fan-out
Parallelism: L1, L2, and L3 can run in parallel. L4 waits for fan-in only if API/data conflict remains.

| Lane | Execution | Role | Skill | Owned question | Evidence target | Status |
| --- | --- | --- | --- | --- | --- | --- |
| L1 | subagent | api-agent | no-skill | What existing endpoint patterns apply to async job creation, lookup, and download handoff? | route files, OpenAPI inputs, handler tests | planned |
| L2 | subagent | data-agent | no-skill | What persisted state shape and tenant isolation patterns already exist for long-running work? | migrations, repository methods, transaction tests | planned |
| L3 | subagent | security-agent | no-skill | What trust-boundary risks exist for signed download URLs? | auth middleware, download handlers, token or signing helpers | planned |
| L4 | local | orchestrator | no-skill | Are lane outputs compatible enough for specification handoff? | L1-L3 summaries and preserved research notes | pending fan-in |
```

Copy the one-question ownership, read-only subagent roles, and explicit local fan-in lane.

## Reject
```markdown
## Research Lanes

- API/data/security/reliability: research export jobs.
- qa-agent: make the test plan.
- worker: try the implementation and report back.
- challenger-agent: approve the final spec.
```

Reject because the first lane has no owned question or evidence target, QA is asked to make a future planning artifact, worker execution is write-capable, and the challenger is asked to approve final spec.

## Agent Traps
- Letting role names stand in for questions.
- Combining evidence gathering and final decision writing in one lane.
- Assigning a specialist skill because the role sounds related, even when the question only needs repository reading.
- Forgetting to record which lanes can run in parallel and what waits for fan-in.
