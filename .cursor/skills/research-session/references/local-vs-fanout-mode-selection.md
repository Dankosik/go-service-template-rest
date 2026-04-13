# Local Vs Fan-Out Mode Selection

## Behavior Change Thesis
When loaded for uncertainty over `local` versus read-only `fan-out` research mode, this file makes the model choose mode by evidence surface, risk, and independent-question shape instead of likely mistake: choosing fan-out because "more agents is safer" or staying local to avoid recording real cross-domain uncertainty.

## When To Load
Load when the hard choice is the research mode for this session.

Do not load for tiny direct-path work where `research-session` itself should be skipped, or when mode is already chosen and only lane details need repair; use `research-lane-planning.md` for that.

## Decision Rubric
Choose `local` when:
- the uncertainty is bounded to one domain, artifact family, or repository surface
- the orchestrator can inspect the evidence directly without losing clarity
- the answer is low-risk or easily reversible
- a preserved note may help, but independent specialist lanes would not change the result

Choose read-only `fan-out` when:
- materially different domains can produce different evidence, for example API, data, security, reliability, or rollout
- a second opinion would reduce ambiguity or high-impact risk
- the questions can be split into independent lanes with one question each
- preserved specialist evidence will help later synthesis, challenge, or resume

Refuse both easy mistakes: do not fan out to write-capable or multi-skill lanes, and do not stay local just because fan-out would expose an unresolved seam.

## Imitate
```markdown
Research mode: local
Why: bounded HTTP semantics question for one GET endpoint and one repository method. Evidence targets are the existing handler, repository method, endpoint tests, and external HTTP precondition guidance.
Lanes:
- L1 local/no-skill: confirm current route and handler behavior.
- L2 local/no-skill: confirm repository version or timestamp source used for ETag material.
Preserved notes: one `research/http-etag-evidence.md` note only if external and repository findings need to be reused in specification.
Stop rule: update workflow handoff and stop before drafting `spec.md`.
```

Copy this when the work is narrow but still worth an explicit research checkpoint.

```markdown
Research mode: fan-out
Why: export jobs touch API semantics, persisted state, download security, retry behavior, and rollout risk. Independent read-only lanes reduce the chance of one research pass flattening those seams.
Lanes:
- L1 api-agent/no-skill: identify existing async job endpoint patterns and response/status conventions.
- L2 data-agent/no-skill: identify persisted job state ownership, tenant keys, and transaction boundaries.
- L3 security-agent/no-skill: inspect existing auth and download-link trust boundaries.
- L4 reliability-agent/no-skill: identify retry, timeout, and terminal failure evidence in current worker patterns.
Fan-in: orchestrator compares lane claims, records conflicts, and routes either to pre-spec challenge or future `specification-session`.
Stop rule: no `spec.md`, no `design/`, no implementation.
```

Copy this when the domains are genuinely separable and all lanes stay advisory.

## Reject
```markdown
Research mode: fan-out
Why: more agents is safer.
Lanes:
- worker: implement a prototype export job.
- api-agent with api-contract-designer-spec and go-security-spec: design API and security together.
- data-agent: decide final schema.
```

Reject because the rationale is ceremonial, one lane is write-capable, one lane uses multiple skills, and the lanes make final design decisions.

## Agent Traps
- Counting domains without checking whether they create independent evidence questions.
- Using fan-out as a substitute for orchestrator synthesis.
- Treating "local" as permission to keep blockers implicit in chat.
- Forgetting that subagent lanes are read-only even when the repository has worker agents available.
