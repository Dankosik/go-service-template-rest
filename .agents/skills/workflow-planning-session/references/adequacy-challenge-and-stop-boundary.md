# Adequacy Challenge And Stop Boundary

## When To Load
Load this when a workflow-planning session needs examples for routing the workflow-plan adequacy challenge, reconciling findings, recording an eligible skip rationale, or marking the session boundary.

This reference only covers the workflow-planning handoff. It does not start the challenge lane itself unless the active session has already written the workflow-control artifacts and the repository policy allows that read-only gate.

## Direct / Lightweight / Full Examples

### Direct path example

```markdown
Adequacy challenge: skipped
Rationale: tiny direct-path task; no dedicated workflow-control artifacts created; inline skip rationale records why the full challenge would add ceremony.
Session boundary: inline note complete; no workflow-planning session artifact.
Next action: direct local work may continue only under the recorded tiny/direct-path waiver.
```

### Lightweight local example

```markdown
Adequacy challenge: skipped only if explicitly waived
Rationale: bounded single-domain work with local next-session research and no planned subagents; the waiver states why challenge overhead is not buying risk reduction.
Session boundary: mark yes only after master and phase file agree on artifact expectations, local research mode, blockers, and stop rule.
Next action: next session starts with local research; escalate if local reading exposes a cross-domain seam.
```

If the waiver is not recorded, run the read-only adequacy challenge before handoff.

### Full orchestrated example

```markdown
Adequacy challenge: required
Lane:
- Role: `challenger-agent`
- Skill: `workflow-plan-adequacy-challenge`
- Owned question: Are `workflow-plan.md` and `workflow-plans/workflow-planning.md` sufficient for this task's full-orchestrated research handoff?
- Scope: task frame, execution shape, planned lanes, artifact expectations, blockers, stop rules, completion marker, next-session start, master/phase consistency.
Reconciliation: repair blocking findings in workflow-control artifacts, or leave the phase blocked.
Session boundary: mark yes only after blocking findings are reconciled.
```

## Good / Bad Lane Rows

Good adequacy-challenge lane rows:

| Lane | Role | Owned Question | Skill | Timing |
| --- | --- | --- | --- | --- |
| A1 | `challenger-agent` | Are the workflow-control artifacts sufficient for the recorded execution shape and next-session handoff? | `workflow-plan-adequacy-challenge` | After draft master and phase file exist. |
| A2 | `challenger-agent` | Did a repaired workflow-control pair resolve the previous blocking handoff finding? | `workflow-plan-adequacy-challenge` | Only if material repairs changed the handoff. |

Bad adequacy-challenge lane rows:

| Lane | Role | Owned Question | Skill | Why It Fails |
| --- | --- | --- | --- | --- |
| Ax | `challenger-agent` | Approve the feature spec and design. | `workflow-plan-adequacy-challenge` | Wrong gate; it reviews workflow-control sufficiency only. |
| Ay | `challenger-agent` | Fix the workflow files directly. | `workflow-plan-adequacy-challenge` | The challenger is read-only and advisory; the orchestrator edits. |
| Az | `security-agent` | Run adequacy, security research, and spec clarification. | multiple | Multiple gates/domains collapsed into one lane. |

## Handoff Examples

Good reconciled handoff:

```markdown
Adequacy challenge status: complete
Blocking findings: none open
Non-blocking findings: recorded as accepted risk in `workflow-plans/workflow-planning.md`
Session boundary reached: yes
Ready for next session: yes
Next session starts with: research, fan-out mode
Stop rule: do not spawn research lanes in this workflow-planning session
```

Good blocked handoff:

```markdown
Adequacy challenge status: complete
Blocking findings: open
Phase status: blocked
Session boundary reached: no
Ready for next session: no
Next action: repair lane ownership and artifact expectations in workflow-control artifacts
Stop rule: do not proceed to research until the blocking finding is reconciled
```

Bad handoff:

```markdown
Adequacy challenge status: timed out after a short wait, treated as failed
Session boundary reached: yes
Next session starts with: research anyway
```

Why it is bad: the repository contract says short waits are not failure when the subagent result is required. Continue waiting unless the lane is clearly hung, superseded, canceled, or no longer needed.

## Exa Source Links
Exa MCP was attempted before these examples were authored on 2026-04-11, but the service returned a 402 credit-limit error. The links below were gathered through fallback web search and are only calibration sources; the repo-local `AGENTS.md` and `docs/spec-first-workflow.md` remain authoritative.

- Anthropic, "Building effective agents": https://www.anthropic.com/engineering/building-effective-agents
- Claude Skills overview: https://claude.com/docs/skills/overview
- Claude Code skills supporting-files guidance: https://code.claude.com/docs/en/skills
- Anthropic, "Equipping agents for the real world with Agent Skills": https://claude.com/blog/equipping-agents-for-the-real-world-with-agent-skills
- arc42 quality requirements: https://docs.arc42.org/section-10/
