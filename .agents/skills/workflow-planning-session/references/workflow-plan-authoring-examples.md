# Workflow Plan Authoring Examples

## When To Load
Load this when writing or repairing the master `workflow-plan.md` during a workflow-planning session.

The master file owns cross-phase control. Keep the examples routing-focused; do not turn `workflow-plan.md` into `spec.md`, `design/`, `plan.md`, `tasks.md`, research notes, or implementation status.

## Direct / Lightweight / Full Examples

### Direct path example

For direct-path work, the best master-file example is usually no master file:

```markdown
Inline workflow-planning note:
Execution shape: direct path.
Rationale: one reversible local edit; no subagent, preserved research, design bundle, plan ledger, or multi-session resume need.
Artifact waiver: no dedicated `workflow-plan.md` or `workflow-plans/` expected.
Stop rule: no workflow-planning session artifact is created.
```

### Lightweight local example

```markdown
# Workflow Plan

## Routing
Current phase: workflow-planning
Phase status: in_progress
Execution shape: lightweight local
Research mode: local, next session
Session boundary reached: no
Ready for next session: no
Next session starts with: research

## Artifact Status
- `workflow-plan.md`: draft
- `workflow-plans/workflow-planning.md`: draft
- `spec.md`: missing, expected later
- `design/`: conditional; design-skip rationale may be recorded later only if the work stays local and unambiguous
- `plan.md`: conditional
- `tasks.md`: conditional, expected by default if `plan.md` is expected
- `test-plan.md`: not expected unless validation obligations grow
- `rollout.md`: not expected unless rollout sequencing is triggered

## Blockers / Assumptions
- Assumption: no persisted-state or public-contract change is visible yet.
- Reopen trigger: escalate to fan-out if local research finds data, security, API, reliability, or rollout ambiguity.

## Phase Workflow Plans
- `workflow-plans/workflow-planning.md`: active

## Adequacy Challenge
Status: pending unless the lightweight-local skip rationale is explicitly accepted.
```

### Full orchestrated example

```markdown
# Workflow Plan

## Routing
Current phase: workflow-planning
Phase status: in_progress
Execution shape: full orchestrated
Research mode: fan-out, next session
Session boundary reached: no
Ready for next session: no
Next session starts with: research

## Artifact Status
- `workflow-plan.md`: draft
- `workflow-plans/workflow-planning.md`: draft
- `research/*.md`: missing, expected for reusable fan-out evidence
- `spec.md`: missing, expected after research and synthesis
- `design/`: missing, expected after approved `spec.md`
- `plan.md`: missing, expected after approved `spec.md + design/`
- `tasks.md`: missing, expected with `plan.md`
- `test-plan.md`: conditional, trigger unknown
- `rollout.md`: conditional, trigger unknown
- Post-code phase workflow files: count unknown; planning must create any used files before implementation

## Planned Research Lanes
See `workflow-plans/workflow-planning.md` for phase-local lane detail.

## Adequacy Challenge
Status: required before workflow-planning handoff.
Resolution: pending.
```

## Good / Bad Lane Rows

Good master-file lane summary rows:

| Lane | Role | Owned Question | Skill | Master-File Detail |
| --- | --- | --- | --- | --- |
| L1 | `api-agent` | API research questions for next session. | `api-contract-designer-spec` | Link or summarize only; full lane detail belongs in the phase file. |
| L2 | `challenger-agent` | Workflow-control adequacy before handoff. | `workflow-plan-adequacy-challenge` | Record status and resolution, not raw transcript. |

Bad master-file lane rows:

| Lane | Role | Owned Question | Skill | Why It Fails |
| --- | --- | --- | --- | --- |
| Lx | `api-agent` | Write every REST decision into the master workflow plan. | `api-contract-designer-spec` | `workflow-plan.md` is not the decision record. |
| Ly | `worker` | Implement Phase 1 after the adequacy check. | `go-coder` | Implementation cannot be planned or started from workflow planning. |

## Handoff Examples

Good master-file complete handoff:

```markdown
Current phase: workflow-planning
Phase status: complete
Session boundary reached: yes
Ready for next session: yes
Next session starts with: research
Adequacy challenge: complete; blocking findings reconciled in `workflow-plans/workflow-planning.md`
Resume order: read `workflow-plan.md`, then `workflow-plans/workflow-planning.md`, then start the recorded research mode.
```

Bad handoff:

```markdown
Current phase: workflow-planning
Phase status: complete
Next session starts with: unspecified later work
Artifact status: everything else can be decided later.
```

Why it is bad: it bypasses research, synthesis, specification, design, and implementation planning.

## Exa Source Links
Exa MCP was attempted before these examples were authored on 2026-04-11, but the service returned a 402 credit-limit error. The links below were gathered through fallback web search and are only calibration sources; the repo-local `AGENTS.md` and `docs/spec-first-workflow.md` remain authoritative.

- Claude Skills overview: https://claude.com/docs/skills/overview
- Claude Code skills supporting-files guidance: https://code.claude.com/docs/en/skills
- Anthropic, "Equipping agents for the real world with Agent Skills": https://claude.com/blog/equipping-agents-for-the-real-world-with-agent-skills
- Michael Nygard, "Documenting Architecture Decisions": https://cognitect.com/blog/2011/11/15/documenting-architecture-decisions
- arc42 architecture decisions: https://docs.arc42.org/section-9/
