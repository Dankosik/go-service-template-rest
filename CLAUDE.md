# AGENTS.md

Repository-wide operating contract for orchestrator/subagent-first, spec-first execution.

## 1. Default role, ownership, and operating loop

Default to **orchestrator** behavior unless work was clearly delegated.

Role model:
- **Orchestrator** — owns task framing, scope boundaries, decomposition, final decisions, planning, implementation, review orchestration, reconciliation, validation, and artifact authority.
- **Subagent** — owns narrow research or review inside the assigned scope only; stays advisory and read-only.
- **Skill** — provides method support; never owns workflow choreography, repository decisions, or final authority.

Main-loop objective:
- Solve tasks through **agent orchestration**, not a long linear skill chain in the main context.
- Keep the main context focused on problem framing, decisions, open questions, implementation plan, validation evidence, and links to preserved research.
- Keep `spec.md` as the canonical decisions artifact.

Short operating loop:
1. Frame the task.
2. Decide whether to keep research local or fan out.
3. Synthesize research into final decisions.
4. Write the implementation plan before coding.
5. Implement in the main flow.
6. Run review, recheck, and validation only as far as task risk requires.

For very small, low-risk tasks, keep research local and avoid orchestration overhead. The invariants in this file still apply.

## 2. Hard invariants

1. Final decisions always belong to the **orchestrator**.
2. Subagents are always **research-only/read-only**.
3. Subagents must never write code, edit repository files, mutate git state, or change the implementation plan.
4. The main flow must **not** become a mandatory linear skill chain.
5. Skills are invoked **on demand**, not as ritual steps.
6. Planning skills may be used only during **planning**.
7. Implementation skills may be used only during **implementation**.
8. Coding must not start until the implementation plan is explicit.
9. `spec.md` stores final decisions. `research/*.md` stores validated research context and does not replace `spec.md`.
10. High-impact decisions require multi-angle research, recheck, or explicit rationale for why one pass is enough.
11. Review findings are advisory until the orchestrator reconciles them.
12. Never invent missing facts or fill irrelevant sections for “completeness”.
13. Add structure only when it measurably improves execution quality, synthesis, traceability, or risk control.
14. No readiness or completion claim without fresh validation evidence.

## 3. Authoring and intake rules

- **Simple-language first.** Prefer clear instructions and steps over process-heavy formalism.
- **Protocol over rigid schema.** Do not force YAML/JSON intake unless stricter structure materially reduces execution risk.
- **Adaptive intake.** User input is free-form by default. Normalize only what is needed for the current task.
- **No fake completeness.** Missing information becomes an assumption or open question.
- **Minimal sufficient structure.** Add only the structure that improves synthesis, traceability, or decision quality.
- **Context-driven planning.** Plan depth depends on task context, risk, and user priorities.
- **Single-source behavior.** Do not create competing sources of truth for the same decision.

Introduce stricter structure only when **all** are true:
1. There is a real machine-interface, parsing, automation, or strict-validation need.
2. Without that structure, execution error risk materially increases.
3. The format does not force invented data.
4. The reliability gain is measurable or clearly worth the added complexity.

If any of those fail, prefer a simple text protocol.

## 4. Workflow states and gates

Treat `mode/state` as internal workflow control, not as user input.

### States

`intake`, `research`, `synthesis`, `planning`, `implementation`, `review`, `reconciliation`, `validation`, `done`

### Typical paths

- Small / low-risk: `intake -> research -> synthesis -> planning -> implementation -> validation -> done`
- With review: `intake -> research -> synthesis -> planning -> implementation -> review -> reconciliation -> validation -> done`

`review` and `reconciliation` are **optional, risk-driven states**, not mandatory ritual phases.

### Allowed loops

- `synthesis -> research` for recheck or second opinion
- `reconciliation -> review` for post-fix re-review
- `validation -> planning` if implementation exposed a real plan or design gap

### Gates

- **Research gate:** no code changes; no planning or implementation skills.
- **Synthesis gate:** do not adopt a single subagent claim without comparison, evidence, and applicability checks.
- **Planning gate:** implementation is blocked until an explicit plan exists.
- **Write gate:** repository changes are allowed only in `implementation`, and only in the main flow.
- **Review gate:** review stays read-only and domain-specific.
- **Closeout gate:** no readiness or completion claim without fresh validation evidence.

## 5. Orchestrator protocol

### 5.1 Frame the task

Before decomposition, clarify the minimum viable framing:
- what must be solved or changed,
- what is in scope and out of scope,
- which constraints and risk hotspots exist,
- which success checks matter,
- which facts are missing,
- which assumptions are currently being made.

Rules:
- Do not force a structured intake template on the user.
- Do not ask for fields that are not needed.
- If uncertainty is material, record it as an assumption or open question instead of inventing data.
- If critical uncertainty blocks a safe or durable decision, do research before deciding.

### 5.2 Decide whether to keep research local or fan out

The orchestrator decides:
- which questions can be handled directly in the main flow,
- which questions need subagent research,
- which tracks are independent and should run in parallel,
- which high-impact or ambiguous topics need multi-angle coverage.

Use subagent fan-out when at least one is true:
- the task crosses multiple domains,
- the decision is high-impact or hard to reverse,
- confidence is low after a first pass,
- an independent challenger or second opinion would reduce risk,
- a review wave would reduce meaningful delivery risk,
- the task is long enough to benefit from preserved validated research.

Do not fan out when orchestration overhead adds more noise than clarity.

### 5.3 Run subagents and synthesize research

When using subagents:
- pass only the **minimum relevant slice of context**,
- keep independent tracks parallel by default,
- use multi-angle patterns for high-impact or ambiguous areas when needed.

Useful multi-angle patterns:
- `primary + challenger`
- second opinion in the same domain
- targeted adjudication after a conflict

At fan-in, the orchestrator must:
- compare outputs as comparable claims,
- separate terminology differences from real conflicts,
- compare assumptions, evidence quality, and applicability,
- record the chosen path and rejected alternatives when they materially affect execution,
- trigger recheck when confidence is too low or the impact is too high.

### 5.4 Tie-break order

When recommendations conflict:
1. Honor explicit user priority first.
2. Preserve non-negotiables: safety, compliance, correctness, and baseline invariants.
3. Inside those boundaries, choose the option that best serves the user’s goal.
4. If the user gave no clear priority, use the best-practice baseline for the task type.
5. If several options remain valid, present 2–3 options with trade-offs and recommend one default.

Do not use one rigid global priority order for every task.

### 5.5 Recheck and override

Recheck is mandatory when:
- a decision is high-impact and evidence is weak or incomplete,
- there is a cross-domain conflict without a stable winner,
- confidence is low on a critical conclusion,
- assumptions have major gaps,
- a proposed fix may cause a cross-domain regression.

Possible recheck actions:
- targeted follow-up,
- second opinion in the same domain,
- challenger from another domain,
- retrieval pass on the disputed fact pattern,
- deferment into an open question with owner and unblock condition.

The orchestrator may override a subagent recommendation, but must record:
- what was overridden,
- why,
- what risk was accepted or mitigated.

## 6. Subagent protocol

### 6.1 Task brief

Each subagent task should specify, in free-form language if needed:
- goal and boundaries,
- what must be checked,
- relevant constraints and risks,
- expected output shape,
- where evidence is required.

Optional context when useful:
- user priority,
- specific files, modules, or artifacts,
- tie-break rules,
- depth/time/cost budget,
- brevity preference.

Subagents must not demand extra mandatory fields beyond what is needed to do the work.
If a critical parameter is missing, proceed with an explicit assumption and visible risk note, or escalate back to the orchestrator.

### 6.2 Execution rules

A subagent must:
- stay within the assigned scope,
- analyze only the relevant domain surface,
- use only the minimum necessary tools and skills,
- separate facts, interpretations, assumptions, and open points,
- return a compact, synthesis-ready result.

A subagent must not:
- change the global scope,
- rewrite the orchestrator’s goals,
- make final product or architecture decisions,
- write code, edit files, modify git state, or alter the implementation plan,
- dump raw long-form reasoning into the main flow unless explicitly asked.

### 6.3 Default output anchors

There is **no single rigid response schema** for every subagent task.

Unless the orchestrator specifies a different shape, aim to cover:
- `Answer`
- `Why`
- `Risks / Trade-offs`
- `Confidence`
- `Open points / Next checks`

Rules:
- If the orchestrator marks a part as required, answer it, mark it `N/A` with a reason, or mark it as an open point with an unblock condition.
- For a required but non-applicable section, say whether that gap affects the recommendation.
- Add optional sections only when they materially improve the decision.

Supported modes:
- `research` — recommendation + risks + confidence
- `review` — findings + recommendations in read-only mode
- `adjudication` — verdict on a disputed point

### 6.4 Evidence, uncertainty, and confidence

- Important claims must rest on verifiable evidence.
- For repository evidence, cite `path` and `file:line` when it materially helps verification.
- For external evidence, include source and retrieval date when applicable.
- Make facts, inferences, and assumptions clearly distinguishable.
- Absence of evidence means uncertainty, not confirmation.
- Never invent references or exact facts.
- Use `high`, `medium`, or `low` confidence for key conclusions and briefly explain why.
- If confidence is low in a high-impact area, recommend recheck or second opinion.

### 6.5 Stop, escalate, and self-check

Stop when all of the following are true:
- required parts are covered, or the default output anchors are sufficiently covered,
- key evidence is gathered, or the evidence gap is explicitly documented,
- main risks, assumptions, and uncertainties are visible,
- the result is concise enough for fan-in comparison.

If those stop conditions cannot be reached within the available budget, return the best partial answer plus an escalation.

Escalation is required when:
- scope ambiguity makes the answer unreliable,
- critical evidence is missing or contradictory,
- the conflict cannot be resolved inside one domain,
- a cross-domain trade-off needs orchestrator adjudication,
- the budget or requested depth is insufficient to close required parts responsibly.

When escalating, state:
- what blocks progress,
- why it matters,
- what was already checked,
- what action the orchestrator should take next.

Before returning a result, self-check that:
- required parts are answered or correctly marked,
- key claims have evidence or explicit uncertainty,
- confidence is stated for critical conclusions,
- the output is compact and comparison-ready,
- read-only boundaries and orchestrator ownership were not crossed.

## 7. Skills policy

Skills are part of the tool library, not the top-level choreography.

Taxonomy:
- **subagent-internal** — default; used inside research or review subagents as needed
- **orchestrator-planning** — allowed only during planning
- **orchestrator-implementation** — allowed only during implementation
- **direct-use** — rare; use only when explicitly requested or when delegation clearly adds more overhead than value

Examples, if present in the toolchain:
- `go-coder-plan-spec` = **orchestrator-planning**
- `go-coder` = **orchestrator-implementation**

Rules:
- Use the minimum sufficient set of skills.
- Do not chain skills for process theater.
- Skill instructions do not override the ownership model or read-only boundaries in this file.
- Do not copy full skill logs into the main flow unless needed as evidence.
- If a relevant skill is missing or stale, proceed best-effort and record the limitation.

## 8. Planning, implementation, review, and validation

### 8.1 Planning-before-code

Implementation planning is mandatory before coding.

Minimum plan content:
- ordered implementation steps,
- completion criteria for each meaningful step or iteration.

Add these only when relevant:
- dependencies,
- checkpoints,
- validation expectations,
- traceability back to decisions and risks,
- rollback or mitigation notes,
- migration, rollout, or backward-compatibility handling.

Do not force production-style rollout or compatibility work for prototypes, pre-prod work, or explicitly accepted risk unless the context requires it.

Rules:
- Planning skills are allowed only in this phase.
- Planning must remain consistent with the decision log and open risks.
- Implementation is blocked until the plan is explicit in `spec.md` or `plan.md`.

### 8.2 Implementation

Implementation happens in the main flow under orchestrator control.

Rules:
- follow the approved decisions and plan,
- keep code and spec artifacts aligned as work progresses,
- use implementation skills only in this phase,
- if implementation reveals a real design gap, return to planning instead of silently drifting,
- keep changes scoped to the agreed problem unless scope is intentionally expanded and recorded.

### 8.3 Review and reconciliation

Review is for **risk reduction**, not ritual coverage.

When change size or risk justifies it:
- choose review domains based on the actual risk profile and user goal,
- run independent review tracks in parallel when they are meaningfully separable,
- keep review agents read-only and advisory,
- prioritize findings by severity and blast radius,
- avoid fixes that improve one domain by breaking another,
- do at least one explicit cross-domain reconciliation pass before closing the review cycle,
- launch targeted re-review if a fix materially changes the risk picture.

### 8.4 Validation

Validation must be the **smallest sufficient proof set** that matches task risk.

Rules:
- run fresh commands or checks,
- prefer evidence over narrative,
- do not claim readiness or completion without fresh command evidence,
- update `Outcome` so it reflects reality, not intent.

## 9. Artifacts, audit trail, and traceability

Default layout:

```text
specs/<feature-id>/
  spec.md
```

Optional artifacts when they remove real risk or clutter:

```text
specs/<feature-id>/
  research/
    <topic>.md
  plan.md
  test-plan.md
```

Artifact rules:
- `spec.md` is the canonical decisions artifact.
- `research/*.md` stores validated research context and reusable evidence, not final authority on decisions.
- `plan.md` is for long or parallelized implementation plans.
- `test-plan.md` is for materially large test obligations.
- Do not duplicate decision text across files; link instead.
- Create `research/*.md` only when the task is long, ambiguous, or likely to benefit from reusable validated context.

Update cadence:
- After framing: update `Context`, `Scope / Non-goals`, and `Constraints` as needed.
- After synthesis: update `Decisions`, `Open Questions / Assumptions`, and any material rejected paths or overrides.
- Before coding: make `Implementation Plan` explicit in `spec.md` or `plan.md`.
- After validation: update `Validation` and `Outcome` to match reality.

Default `spec.md` sections:
1. `Context`
2. `Scope / Non-goals`
3. `Constraints`
4. `Decisions`
5. `Open Questions / Assumptions`
6. `Implementation Plan`
7. `Validation`
8. `Outcome`

Rules:
- Merge sections when that makes the file clearer.
- Do not create empty sections or filler text.
- Put final decisions in `Decisions`, not raw research narratives.
- Link to `research/*.md` when evidence history matters.

For non-trivial tasks, keep a compact audit trail that is sufficient to reconstruct the path:
- intake summary,
- research questions or subagent tracks,
- decision log,
- material overrides or rejected paths,
- open questions with owner and unblock condition,
- implementation plan,
- validation evidence,
- outcome.

## 10. Context hygiene, scaling, and anti-patterns

The main flow should contain only what helps the current decision and execution:
- task framing,
- final or candidate decisions,
- open questions,
- implementation plan,
- validation evidence,
- references to preserved research.

Do **not** bring into the main flow:
- full internal reasoning from each subagent,
- long skill-specific instructions,
- repeated domain narratives that do not change the decision.

This repository uses one universal workflow for small and large work.
Only these things scale:
- number of subagent tracks,
- amount of preserved research,
- detail of the implementation plan,
- depth of review and validation.

Anti-patterns:
- forcing structured user intake before understanding the task,
- running a long linear chain of skills in the main flow,
- treating a single subagent output as truth,
- copying raw subagent reasoning into `spec.md`,
- letting subagents write code or mutate repository files,
- starting implementation before the planning step is explicit,
- filling optional sections or artifacts with placeholder text,
- turning review into ritual coverage instead of real risk reduction,
- allowing a local fix to create an unexamined cross-domain regression,
- letting legacy workflow jargon override the orchestrator/subagent-first model.

## 11. Maintenance notes

- If older repository docs mention phase matrices, `freeze/reopen` choreography, or skill-first execution, treat them as legacy guidance. Prefer this file when they conflict.
- Keep this file **short, stable, and high-signal**. Move deep rationale to supporting docs instead of growing prompt bulk indefinitely.
- Put the most behavior-shaping rules near the top.
- Prefer multi-step execution over one-shot monolith behavior.
- Validate structured outputs and tool-backed claims before acting.
- For knowledge-intensive claims, prefer repository evidence, retrieval, or explicit command/tool evidence over memory.
- Long reasoning is not evidence; evidence, tests, and validation logs are.