# Execution Shape Selection

## When To Load
Load this when a workflow-planning session needs examples for choosing or checking `direct path`, `lightweight local`, or `full orchestrated`.

Use it only after reading `AGENTS.md` and `docs/spec-first-workflow.md`. This file calibrates routing examples; it does not authorize research, `spec.md`, `design/`, `plan.md`, `tasks.md`, tests, or implementation work.

## Direct / Lightweight / Full Examples

### Direct path example

```markdown
Execution shape: direct path
Why: one small reversible documentation typo in one skill reference; no domain behavior, runtime, data, contract, or multi-session risk.
Research mode: local; no subagent lanes.
Artifact expectations: no dedicated workflow artifacts; inline skip rationale is enough.
Adequacy challenge: skipped because the work is tiny/direct-path.
Stop rule: record the inline workflow-planning rationale, then proceed only if the user requested the tiny edit in the same direct-path pass.
```

Why it works: it uses the tiny exception instead of creating process files for ceremony.

### Lightweight local example

```markdown
Execution shape: lightweight local
Why: bounded single-domain update to one existing HTTP middleware behavior; no persisted-state change, public contract change, or cross-domain ownership seam visible after first read.
Research mode: local in the next session; no fan-out unless local reading exposes a domain seam.
Artifact expectations: `workflow-plan.md` and `workflow-plans/workflow-planning.md` expected for this dedicated session; later `spec.md` expected; `design/` may be skipped only if the specification records the local design-skip rationale; `plan.md` and `tasks.md` may be waived only if it remains tiny after specification.
Adequacy challenge: record skip only if the upfront lightweight-local waiver is explicit; otherwise run the read-only challenge before handoff.
Stop rule: stop after routing is written; do not begin the local research read.
```

Why it works: it preserves the explicit research-mode decision without pretending the next stage has already happened.

### Full orchestrated example

```markdown
Execution shape: full orchestrated
Why: tenant-scoped async exports touch API, data ownership, background work, signed URL security, admin authorization, reliability, observability, and rollout risk.
Research mode: fan-out in the next session.
Artifact expectations: `workflow-plan.md` and `workflow-plans/workflow-planning.md` required now; later `research/*.md`, `spec.md`, core `design/`, `plan.md`, and `tasks.md` expected; `test-plan.md` and `rollout.md` likely but still marked conditional until research confirms triggers.
Adequacy challenge: required before workflow-planning handoff.
Stop rule: stop after workflow-control artifacts and challenge reconciliation; next session starts with research fan-out.
```

Why it works: it plans orchestration and later artifacts without starting research or designing the feature.

## Good / Bad Lane Rows

Good lane rows for execution-shape calibration:

| Lane | Role | Owned Question | Skill | Why It Fits |
| --- | --- | --- | --- | --- |
| L1 | `data-agent` | Which persisted-state and tenant-isolation questions must research answer before specification? | `go-data-architect-spec` | One domain, one question family, read-only, useful only after full orchestration is chosen. |
| L2 | `security-agent` | What authorization, signed URL, and abuse-resistance seams need separate research? | `go-security-spec` | Separates security evidence from API/data assumptions. |

Bad lane rows to reject:

| Lane | Role | Owned Question | Skill | Why It Fails |
| --- | --- | --- | --- | --- |
| Lx | `worker` | Own a later code-change phase from this routing session. | `go-coder` | Pulls implementation ownership into workflow planning. |
| Ly | `architecture-agent` | Own final architecture decisions from the routing artifact. | `go-architect-spec` | Mixes research, final decisions, and later-phase authorship into the routing phase. |

## Handoff Examples

Good handoff for direct path:

```markdown
Workflow-planning result: inline direct-path skip rationale recorded. No dedicated workflow-control files expected.
Next action: continue only with the tiny local edit if the active task explicitly allows same-pass direct-path execution.
Stop boundary: no subagents, no research artifact, no spec/design/planning artifact.
```

Good handoff for lightweight local:

```markdown
Session boundary reached: yes
Ready for next session: yes
Next session starts with: research, local mode
Handoff note: if local research exposes data ownership, API contract, or security uncertainty, reopen workflow planning or escalate to fan-out before specification.
```

Good handoff for full orchestrated:

```markdown
Session boundary reached: yes
Ready for next session: yes
Next session starts with: research, fan-out mode
Handoff note: spawn only the planned read-only lanes in the research session; preserve evidence in `research/*.md` when it remains useful after fan-in.
```

Bad handoff:

```markdown
Next session starts with: unspecified later work
Reason: the workflow plan already lists the expected lanes.
```

Why it is bad: workflow planning does not replace research, synthesis, specification, design, or implementation planning.

## Exa Source Links
Exa MCP was attempted before these examples were authored on 2026-04-11, but the service returned a 402 credit-limit error. The links below were gathered through fallback web search and are only calibration sources; the repo-local `AGENTS.md` and `docs/spec-first-workflow.md` remain authoritative.

- Claude Skills overview: https://claude.com/docs/skills/overview
- Claude Code skills supporting-files guidance: https://code.claude.com/docs/en/skills
- Anthropic, "Equipping agents for the real world with Agent Skills": https://claude.com/blog/equipping-agents-for-the-real-world-with-agent-skills
- Anthropic, "Building effective agents": https://www.anthropic.com/engineering/building-effective-agents
- arc42 architecture decisions: https://docs.arc42.org/section-9/
