# Research Mode And Fan-Out Lanes

## When To Load
Load this when a workflow-planning session needs examples for choosing `local` versus `fan-out`, enumerating read-only subagent lanes, assigning exactly one skill per lane, or checking whether lane rows are too broad.

Use it only for planning the next research session. Do not spawn the lanes, run local research, write `research/*.md`, draft `spec.md`, or create implementation artifacts from this reference.

## Direct / Lightweight / Full Examples

### Direct path example

```markdown
Research mode: local, inline only
Reason: the task is tiny and reversible; the first read is enough to confirm no domain seam.
Planned lanes: none.
Challenge path: none; direct-path skip rationale recorded.
Stop rule: no fan-out table; no subagent call.
```

### Lightweight local example

```markdown
Research mode: local in the next session
Reason: one bounded package surface; low confidence cost for a local read; no visible cross-domain risk yet.
Planned lanes: none initially.
Escalation trigger: if local reading finds API contract drift, persistent state, auth policy, goroutine lifecycle, or rollout risk, stop and repair workflow planning before wider research.
Challenge path: adequacy challenge may be skipped only with explicit lightweight-local waiver; otherwise route it before handoff.
```

### Full orchestrated example

```markdown
Research mode: fan-out in the next session
Reason: the task crosses API, data, background job, security, reliability, observability, and delivery seams.
Planned lanes:
- L1 `api-agent`: client-visible REST/resource model questions; skill `api-contract-designer-spec`.
- L2 `data-agent`: job state, tenant isolation, migration/backfill triggers; skill `go-data-architect-spec`.
- L3 `security-agent`: admin-only control surface and signed URL trust boundary; skill `go-security-spec`.
- L4 `reliability-agent`: async job timeouts, retries, cancellation, degradation; skill `go-reliability-spec`.
- L5 `qa-agent`: proof obligations and deterministic test layers; skill `go-qa-tester-spec`.
Fan-in: compare claims, record conflicts, then run pre-spec challenge only after research synthesis.
Stop rule: the workflow-planning session stops before spawning L1-L5.
```

## Good / Bad Lane Rows

Good lane rows:

| Lane | Role | Owned Question | Skill | Expected Output |
| --- | --- | --- | --- | --- |
| L1 | `api-agent` | What REST resources, status semantics, pagination/filtering, and error contracts need specification evidence? | `api-contract-designer-spec` | Candidate API questions and risks only; no OpenAPI edits. |
| L2 | `data-agent` | What source-of-truth, migration, tenant isolation, and retention questions must be answered before spec approval? | `go-data-architect-spec` | Data decision options and blockers only; no migration files. |
| L3 | `security-agent` | What auth, trust-boundary, signed URL, and abuse-resistance seams affect scope or validation? | `go-security-spec` | Security questions and constraints only; no implementation plan. |
| L4 | `challenger-agent` | Are the candidate workflow-control artifacts sufficient for handoff? | `workflow-plan-adequacy-challenge` | Blocking or non-blocking adequacy findings only. |

Bad lane rows:

| Lane | Role | Owned Question | Skill | Why It Fails |
| --- | --- | --- | --- | --- |
| Lx | `data-agent` | Own every export question and downstream planning decision. | `go-data-architect-spec` | Too broad and asks a research lane to own planning. |
| Ly | `security-agent` | Security review, API design, data schema, and QA. | `go-security-spec` | Multiple domains collapsed into one lane and one skill. |
| Lz | `worker` | Own downstream code changes from this routing phase. | `go-coder` | Write-capable implementation work is prohibited in workflow planning. |
| La | `api-agent` | Use `api-contract-designer-spec` and `go-reliability-spec` together. | multiple | Violates one-skill-per-pass. Split into two lanes or keep synthesis local. |

## Handoff Examples

Good local-research handoff:

```markdown
Next session starts with: research, local mode.
Local research scope: inspect only the relevant package and existing task artifacts needed to confirm behavior and artifact triggers.
Escalation: if cross-domain or high-impact ambiguity appears, stop local research and reopen workflow planning for fan-out.
```

Good fan-out handoff:

```markdown
Next session starts with: research, fan-out mode.
Planned parallel lanes: L1 API, L2 data, L3 security, L4 reliability, L5 QA.
Fan-in rule: compare assumptions and evidence quality before candidate synthesis; do not treat any single lane as final authority.
Stop rule: this workflow-planning session ends before lane execution.
```

Bad handoff:

```markdown
Next session starts with: mixed research/specification work.
Lanes may own later-phase outputs as they finish.
```

Why it is bad: research lanes are advisory/read-only, and specification is a later phase with its own gate.

## Exa Source Links
Exa MCP was attempted before these examples were authored on 2026-04-11, but the service returned a 402 credit-limit error. The links below were gathered through fallback web search and are only calibration sources; the repo-local `AGENTS.md` and `docs/spec-first-workflow.md` remain authoritative.

- Anthropic, "Building effective agents": https://www.anthropic.com/engineering/building-effective-agents
- Claude Skills overview: https://claude.com/docs/skills/overview
- Claude Code skills supporting-files guidance: https://code.claude.com/docs/en/skills
- Anthropic, "Equipping agents for the real world with Agent Skills": https://claude.com/blog/equipping-agents-for-the-real-world-with-agent-skills
