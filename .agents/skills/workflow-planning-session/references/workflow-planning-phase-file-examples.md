# Workflow-Planning Phase File Examples

## When To Load
Load this when writing or repairing `workflow-plans/workflow-planning.md`.

The phase file owns only this workflow-planning session's local orchestration: lane table, order/parallelism, fan-in or challenge path, status, completion marker, blockers, next action, and stop rule. Do not let it replace the master `workflow-plan.md` or any later artifact.

## Direct / Lightweight / Full Examples

### Direct path example

```markdown
Phase file: not expected
Reason: tiny direct-path work uses an inline skip rationale; creating `workflow-plans/workflow-planning.md` would add ceremony.
Stop rule: no phase-local workflow file is written.
```

### Lightweight local example

```markdown
# Workflow Planning Phase

## Status
Phase: workflow-planning
Status: in_progress
Execution shape: lightweight local
Research mode for next session: local

## Local Orchestration
- Order: write/repair master first, then this phase file, then run or record adequacy challenge.
- Planned lanes: none for research; local research starts only in the next session.
- Parallelizable work: none during this phase.
- Fan-in path: not applicable unless workflow planning escalates to fan-out.

## Completion Marker
The phase is complete when master and phase file agree on execution shape, research mode, artifact expectations, blockers, adequacy challenge status, and next-session start.

## Stop Rule
Stop after workflow-control handoff. Do not begin the local research read in this session.
```

### Full orchestrated example

```markdown
# Workflow Planning Phase

## Status
Phase: workflow-planning
Status: in_progress
Execution shape: full orchestrated
Research mode for next session: fan-out

## Planned Lanes For Next Research Session
| Lane | Role | Owned Question | Skill | Parallel? |
| --- | --- | --- | --- | --- |
| L1 | `api-agent` | API/resource contract questions that could change scope or validation. | `api-contract-designer-spec` | yes |
| L2 | `data-agent` | Job-state, tenant isolation, migration, and retention questions. | `go-data-architect-spec` | yes |
| L3 | `security-agent` | Admin auth, signed URL trust boundary, and abuse resistance. | `go-security-spec` | yes |
| L4 | `reliability-agent` | Async retry, timeout, cancellation, and degradation questions. | `go-reliability-spec` | yes |
| L5 | `qa-agent` | Test strategy and proof-obligation questions. | `go-qa-tester-spec` | yes |

## Adequacy Challenge
Required before handoff. Challenger lane uses `workflow-plan-adequacy-challenge` only and stays read-only.

## Completion Marker
The phase is complete when blocking adequacy findings are reconciled, artifact expectations are explicit, and `workflow-plan.md` marks the session boundary.

## Stop Rule
Stop before starting L1-L5 research.
```

## Good / Bad Lane Rows

Good phase-file lane rows:

| Lane | Role | Owned Question | Skill | Parallel? |
| --- | --- | --- | --- | --- |
| L1 | `domain-agent` | What business invariants or state-transition questions must research answer? | `go-domain-invariant-spec` | yes |
| L2 | `reliability-agent` | Which timeout, retry, shutdown, and degradation questions are planning-critical? | `go-reliability-spec` | yes |
| L3 | `challenger-agent` | Are workflow-control artifacts sufficient for this task and execution shape? | `workflow-plan-adequacy-challenge` | no; after draft artifacts |

Bad phase-file lane rows:

| Lane | Role | Owned Question | Skill | Why It Fails |
| --- | --- | --- | --- | --- |
| Lx | `default` | Look at the repo and figure it out. | `no-skill` | No owned question, no bounded scope, not synthesis-ready. |
| Ly | `qa-agent` | Own later validation and test-output work from this phase. | `go-qa-tester` | Starts later planning/test implementation from workflow planning. |
| Lz | `challenger-agent` | Approve the workflow plan. | `workflow-plan-adequacy-challenge` | The challenger returns advisory findings; the orchestrator owns reconciliation. |

## Handoff Examples

Good phase-file handoff:

```markdown
## Handoff
Phase status: complete
Blocking findings: none open
Next action: next session starts with research fan-out using L1-L5
Stop boundary: yes; no research has started
```

Bad phase-file handoff:

```markdown
## Handoff
Phase status: complete
Next action: mixed later-phase work already underway
```

Why it is bad: it crosses the session boundary and allows research/specification during workflow planning.

## Exa Source Links
Exa MCP was attempted before these examples were authored on 2026-04-11, but the service returned a 402 credit-limit error. The links below were gathered through fallback web search and are only calibration sources; the repo-local `AGENTS.md` and `docs/spec-first-workflow.md` remain authoritative.

- Anthropic, "Building effective agents": https://www.anthropic.com/engineering/building-effective-agents
- Claude Code skills supporting-files guidance: https://code.claude.com/docs/en/skills
- Anthropic, "Equipping agents for the real world with Agent Skills": https://claude.com/blog/equipping-agents-for-the-real-world-with-agent-skills
- arc42 architecture decisions: https://docs.arc42.org/section-9/
