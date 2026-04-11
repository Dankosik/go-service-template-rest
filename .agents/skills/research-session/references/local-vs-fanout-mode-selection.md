# Local Vs Fan-Out Mode Selection Examples

## When To Load
Load this file when the main uncertainty is whether the research session should stay `local` or use read-only subagent `fan-out`.

Do not load it for tiny direct-path work where `research-session` itself should be skipped.

## Selection Heuristics
Choose `local` when:
- the task is bounded to one domain or one repository surface
- the orchestrator can inspect sources directly without losing context
- the evidence target is narrow and low-risk
- a preserved note may be useful, but independent specialist lanes would not add much

Choose read-only `fan-out` when:
- materially different domains are involved
- a second opinion would reduce ambiguity or high-impact risk
- the research questions can be split into independent lanes
- preserved specialist evidence will help later synthesis, challenge, or resume

Never choose fan-out if the lane cannot stay read-only. Never choose local mode just to avoid recording a real cross-domain uncertainty.

## Good Local Research Artifact
```markdown
Research mode: local
Why: bounded HTTP semantics question for one GET endpoint and one repository method. Evidence targets are the existing handler, repository method, endpoint tests, and external HTTP precondition guidance.
Lanes:
- L1 local/no-skill: confirm current route and handler behavior.
- L2 local/no-skill: confirm repository version or timestamp source used for ETag material.
Preserved notes: one `research/http-etag-evidence.md` note only if external and repository findings need to be reused in specification.
Stop rule: update workflow handoff and stop before drafting `spec.md`.
```

## Good Fan-Out Research Artifact
```markdown
Research mode: fan-out
Why: export jobs touch API semantics, persisted state, download security, retry behavior, and rollout risk. Independent read-only lanes reduce the chance of one research pass flattening those seams.
Lanes:
- L1 api-agent/no-skill: identify existing async job endpoint patterns and response/status conventions.
- L2 data-agent/no-skill: identify persisted job state ownership, tenant keys, and transaction boundaries.
- L3 security-agent/no-skill: inspect existing auth and download-link trust boundaries. Read-only research only.
- L4 reliability-agent/no-skill: identify retry, timeout, and terminal failure evidence in current worker patterns.
Fan-in: orchestrator compares lane claims, records conflicts, and routes either to pre-spec challenge or future `specification-session`.
Stop rule: no `spec.md`, no `design/`, no `plan.md`, no implementation.
```

## Bad Research Artifact
```markdown
Research mode: fan-out
Why: more agents is safer.
Lanes:
- worker: implement a prototype export job.
- api-agent with api-contract-designer-spec and go-security-spec: design API and security together.
- data-agent: decide final schema.
```

Why this is bad:
- it uses a write-capable worker
- one lane uses multiple skills
- lanes are asked to make final design or schema decisions
- the mode choice is ceremonial instead of evidence-driven

## Evidence Vs Decision Separation
Good handoff:

```markdown
Evidence: API and data lanes disagree about whether job cancellation is already modeled. The API lane found a route naming precedent; the data lane found no terminal cancellation state in persisted jobs.
Handoff: future specification must decide whether cancellation is in scope or explicitly out of scope.
```

Bad handoff:

```markdown
Decision: cancellation is out of scope.
Plan: implement create/list/get first, then cancellation later.
```

The research session can describe the conflict and route it. It must not convert that conflict into final scope or an execution plan.

## Exa Source Links
These external method links were found through Exa before drafting the examples. Use them as selection and evidence-discipline inspiration only.

- [Penn Libraries, Developing the Research Question](https://guides.library.upenn.edu/c.php?g=475980&p=10914820)
- [University of California Santa Cruz, Creating the Search](https://guides.library.ucsc.edu/c.php?g=1440266&p=10697344)
- [Cochrane Handbook, Chapter 4: Searching for and selecting studies](http://training.cochrane.org/handbook/current/chapter-04)
- [University of Tasmania, Documenting Search Strategies](https://utas.libguides.com/SystematicReviews/Documenting)
