# AGENTS.md

<!-- codebase-memory-binding:start -->
## Codebase Memory MCP

This repository is indexed as `Users-daniil-Projects-Opensource-go-service-template-rest`.
Use that project directly. Check `index_status` when freshness matters. If the MCP is unavailable, fall back to CodeGraph, then `rg` or direct reads, and state the fallback when it affects confidence.
<!-- codebase-memory-binding:end -->

Repository-wide contract for producing reliable Go-service changes with the least workflow needed.

## Authority And Loading

- Explicit user, system, and developer instructions win.
- This file owns request authorization and repository-wide invariants.
- [docs/spec-first-workflow.md](docs/spec-first-workflow.md) is the workflow router. Read only the current phase file and any shared file needed for the decision at hand.
- Task-local artifacts own accepted task decisions. Runtime and generated-source authorities named by those artifacts still win over derived prose.
- `SOUL.md` is lower-precedence engineering and communication guidance. Skills provide methods; neither overrides this contract or task-local decisions.

## Authorization

- `answer`, `explain`, `review`, `diagnose`, and `plan` authorize inspection and reporting, not implementation.
- `change`, `build`, and `fix` authorize in-scope local edits and relevant non-destructive validation.
- Ask before external writes, destructive actions, purchases, or material scope expansion. Do not ask before ordinary repository reads, in-scope edits, or tests.
- Respect an explicit boundary such as `read-only`, `docs-only`, `research only`, or a named phase.

## Explicit Workflow Opt-Out

- Honor a workflow opt-out when the implementation request is clear enough to act on and the user explicitly says the workflow may be skipped or bypassed and asks to proceed to implementation.
- A valid opt-out overrides the normal path and phase routing for that request. Proceed directly to implementation; do not first require or run workflow-start checks, phase/readiness gates, workflow artifacts, or workflow-only delegation and review. Do not create a record merely to document the opt-out.
- The opt-out waives process, not scope, safety, permission, or proof. Preserve explicit boundaries, ask before external or destructive actions, inspect the affected code before editing, do not invent a materially unresolved behavior or ownership decision, and run fresh validation proportionate to the change.
- If implementation exposes a genuinely blocking decision, ask only for that decision. After it is resolved, continue directly unless the user withdraws the opt-out or requests workflow artifacts.

## Working Contract

1. Reconstruct the intended outcome before acting. Inspect repository facts instead of asking the user for facts the repository can answer. Ask only for a decision that would materially change scope, behavior, ownership, safety, or proof; otherwise state a bounded assumption and continue.
2. Choose the smallest execution path that preserves correctness. Process is proportional to uncertainty, blast radius, reversibility, and proof difficulty—not task size labels or the number of domains mentioned.
3. Describe outcomes, constraints, success criteria, and stop conditions. Do not prescribe steps the model can choose reliably, repeat the same rule across files, or create artifacts solely to prove that a phase happened.
4. `spec.md`, when present, is the decision record. `tasks.md`, when present and ready, is the implementation ledger. Later phases consume those decisions; they do not silently invent missing product, contract, ownership, or proof policy.
5. Public contracts, persisted data, security, money, concurrency/lifecycle, deployment, and cross-service ownership require explicit relevant decisions and proof. They do not automatically require every workflow phase or artifact.
6. Research, design, test design, and planning are conditional. Use them when they reduce a real uncertainty or make implementation/proof executable. Keep the result inline when it stays compact; persist it only when another session or actor must consume it.
7. Review is risk-based. Use an independent read-only reviewer for hard-to-reverse, high-impact, ambiguous, or explicitly review-required work. Otherwise a focused self-review plus fresh validation is sufficient. A reviewer never edits the artifact it judges. When review belongs to an active macro phase, the root runs review, repair, and any fresh re-review in the same session; an internal checkpoint never emits a next-session prompt. The exact handoff gate and standalone-review exception live in [Subagents And Handoff](docs/spec-first-workflow/shared/subagents-and-handoff.md#handoff).
8. Use subagents only for concrete, independent, bounded questions where separate context improves speed or quality. Keep sequential or tightly coupled work local. The root owns scope, synthesis, decisions, edits, and completion claims. Default to at most three concurrent lanes and no nested delegation.
9. Delegated output is evidence. The root verifies it against the current repository and accepted task before using it. Implementation may be local or delegated; isolation is required only when it materially reduces contention or risk.
10. Prefer current Go stdlib and established repository patterns. Add dependencies, helpers, interfaces, or architectural patterns only when they solve a present requirement better than the simpler option.
11. Keep ownership explicit. Put substantial code in the narrow owning package/file, preserve generated-source discipline, and remove replaced code and adjacent stale artifacts unless current compatibility evidence justifies retention.
12. Do not claim ready, complete, fixed, or covered without fresh evidence matched to the claim. Report unavailable or narrower proof honestly and name the next useful check.

## Execution Paths

| Path | Use when | Durable artifacts |
| --- | --- | --- |
| `direct` | The request is clear, the change is small and reversible, ownership is obvious, and proof is bounded. | None required. A short inline plan is enough. |
| `structured` | The work is non-trivial but bounded; one or more decisions or steps must survive implementation. | Create only the useful subset of `spec.md`, `design/`, `test-plan.md`, and `tasks.md`. |
| `orchestrated` | The work is broad, hard to reverse, multi-owner, evidence-heavy, explicitly multi-agent, or likely to span sessions. | Add `workflow-plan.md` and durable research only when coordination or resume requires them. |

Escalate the path when new evidence invalidates an assumption, reveals an owner conflict, or makes proof materially harder. Do not backfill ceremony for completed safe work; record only the decision or blocker the remaining work needs.

## Phases

The normal dependency order is:

`intake -> research? -> specification? -> design? -> test design? -> planning? -> implementation and verification`

Question marks mean conditional. Review belongs to the artifact it evaluates, not to a mandatory user-started session. A single user request may cross several phases when it authorizes end-to-end work and no explicit boundary says to stop. When the user names a phase, stay inside it and complete its internal review/repair loop before offering a next macro phase.

Use the phase files for their unique questions:

- intake: outcome, scope, constraints, success, blocking ambiguity;
- research: evidence that can change a decision;
- specification: observable behavior, invariants, decisions, non-goals, proof expectations;
- system/integration design: runtime mechanism, contracts, data, failures, rollout;
- Go ownership design: package/file responsibility, dependency direction, cleanup, test owner;
- test design: risky scenarios and the smallest convincing proof levels;
- planning: executable order, ownership, evidence, and completion condition;
- implementation/validation: make the change, review the diff, run fresh proof, and report the evidence boundary.

## Instruction Ownership

- Keep global rules here.
- Keep phase-specific method in `docs/spec-first-workflow/phases/`.
- Keep artifact persistence and status rules in `docs/spec-first-workflow/shared/artifact-model.md`.
- Keep delegation, review independence, resume, and handoff rules in `docs/spec-first-workflow/shared/subagents-and-handoff.md`.
- Keep task-specific decisions in task-local artifacts.
- When two surfaces repeat a rule, retain the narrowest canonical owner and replace the other copy with a link.

@SOUL.md

@RTK.md
