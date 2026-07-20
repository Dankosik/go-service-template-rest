# Spec-First Workflow

Choose the smallest path that lets the next actor change the right owner and prove the result. `AGENTS.md` owns authorization and global invariants; this file owns routing, optional artifacts, review, and movement.

## Choose A Path

### Direct

Use `direct` when the request is clear, local, reversible, single-owner, and has bounded proof with no unresolved protected-domain decision. Inspect the owner, edit the assigned checkout, review the bounded diff, and run focused proof. Keep the outcome and assumptions inline. Do not create a Goal, App Worker, worktree, artifact, independent review, or workflow opt-out merely because code changes.

### Structured

Use `structured` when a decision, evidence packet, proof design, or task order must survive into another phase, actor, or session. Create only the artifact that carries that information:

| Need | Owner | Persist only when needed |
| --- | --- | --- |
| Ambiguous input | Intake | Brief |
| Decision-changing evidence | Research | Evidence note |
| Behavior or authority | Specification | `spec.md` |
| Mechanism, contract, data flow, or Go placement | Technical design | `design/` |
| Non-obvious proof | Test design | `test-plan.md` |
| Multi-step order or resume | Planning | `tasks.md` |
| Deployment sequence | Delivery | `rollout.md` |

Self-review is the default. Use one independent read-only reviewer only when the user requests it or the fixed decision is high-impact, hard to reverse, cross-owner, or cannot be credibly falsified by its author. Review findings return to the owning author; re-review only the repaired and transitively affected surface. `PASS`, `CONCERNS`, and `FAIL` describe a review result, not a universal phase gate.

### Orchestrated

Use `orchestrated` only when coordination is a real problem: long-running or resumable work, parallel independent outcomes, a dirty-checkout conflict, isolation, separate context, or explicit delegation. Then use the smallest necessary Goal, App Worker/worktree, ledger, or wave. A path is not a quality tier; re-route only when evidence changes risk, ownership, reversibility, or proof.

## Protected Decisions

Public contracts, persisted data, security, money, concurrency/lifecycle, deployment, and cross-service ownership need explicit relevant decisions and proof. Activate only the affected lenses. A risk signal does not by itself require every phase, artifact, reviewer, worker, or broad test suite.

## Movement

Move when the next action can proceed without inventing behavior, ownership, mechanism, or proof. Inputs must be canonical, mechanically derivable, or explicitly external with owner, shape, and checkpoint. Reopen the narrowest owner when evidence breaks a decision.

An authorized end-to-end request may cross any needed steps in one session. Stop only for a named boundary, missing user or external authority, a decision-changing contradiction, or a real need for durable coordination not yet recorded. Use [Artifact Model](spec-first-workflow/shared/artifact-model.md) only for persistence or resume; use [Subagents And Handoff](spec-first-workflow/shared/subagents-and-handoff.md) only when a separate read-only lane or handoff is useful. Explicit user grilling uses the `grilling` skill; it is not a default review stage.

## Prompt Maintenance

Preserve outcome, authority, success criteria, proof, and stop conditions. Change one instruction group at a time and run the fast instruction checks. Run model or mutation harnesses only when their implementation or selected fixtures change.
