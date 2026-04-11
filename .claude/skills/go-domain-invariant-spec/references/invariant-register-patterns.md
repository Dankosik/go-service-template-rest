# Invariant Register Patterns

## Behavior Change Thesis
When loaded for symptom "rules are descriptive, ownerless, or missing enforcement points", this file makes the model write falsifiable owner-backed invariant rows instead of likely mistake "list broad business logic bullets with no source of truth, pass/fail signal, or violation outcome."

## When To Load
Load this when a domain spec needs invariant statements, owner assignment, source-of-truth authority, enforcement-point choices, or a review of hidden or descriptive-only business rules.

## Decision Rubric
- Write one invariant per condition that must remain true; split bundled "and" rules when failure outcomes differ.
- Use `local_hard_invariant` only when one owned boundary can prevent the violation synchronously. Use `cross_service_process_invariant` when correctness depends on async steps, derived surfaces, or reconciliation.
- Name an owner with authority to keep the rule true. "Database", "handler", and "tests" are mechanisms, not owners.
- State the enforcement point at the domain level first, such as transition guard, policy decision, persistence constraint, process contract, or reconciliation rule.
- Require an observable pass/fail signal and violation outcome before handing the rule to API, data, reliability, security, or QA work.
- If the rule depends on tenant, actor, object ownership, or source-of-truth authority, include that in the invariant instead of treating it as surrounding implementation detail.

## Imitate
```text
INV-SKILL-CANONICAL-001
Statement: A repository skill change is authoritative only when it is made under `.agents/skills`; runtime mirrors are derived surfaces and must not become competing sources of truth.
Type: local_hard_invariant
Owner: repository skill source-of-truth policy
Source of truth: specs/agents-skills-source-of-truth/spec.md
Enforcement point: skill sync/check workflow and reviewer discipline
Observable pass/fail: no active tooling or docs treat top-level `skills/` as canonical
Violation outcome: reject the change or repair mirror/source-of-truth drift before validation closes
Downstream handoff: tooling docs and sync validation must point at `.agents/skills`
```

Copy the shape: one falsifiable rule, one owner, one authority source, one pass/fail signal, and one consequence.

```text
INV-SUBAGENT-READONLY-001
Statement: Subagents may produce advisory research or review output but must not edit repository files, mutate git state, or change the implementation plan.
Type: local_hard_invariant
Owner: orchestrator workflow contract
Source of truth: AGENTS.md
Enforcement point: orchestration routing and fan-in reconciliation
Observable pass/fail: delegated lanes return read-only findings and no repository diff or git mutation is accepted from them
Violation outcome: reject the mutation as authoritative and reconcile in the orchestrator-owned flow
Downstream handoff: review and validation must not claim agent-backed coverage for write-capable delegated work
```

Copy the policy boundary: advisory authority and mutation authority are different business concepts.

## Reject
```text
Keep skills organized and up to date.
```

Failure: no owner, no boundary, no pass/fail signal, and no violation outcome.

```text
INV-001: The API validates the request and writes a row to the database.
```

Failure: transport and storage mechanics replaced the business rule. Rewrite as the allowed behavior and only then hand off implementation surfaces.

## Agent Traps
- Do not make every validation rule an invariant; reserve the register for rules whose violation changes correctness, authority, or accepted behavior.
- Do not hide tenant, actor, or object ownership inside a generic "authorized" note when it decides whether a transition is allowed.
- Do not assign ownership to the first component that detects a violation if another domain policy owns the rule.
- Do not use "eventual consistency" as an escape hatch for a hard invariant; either classify it as process-level with reconciliation or reject the design.
- Do not add downstream API status codes, table constraints, or retry budgets until the rule and violation outcome are stable.

## Validation Shape
Every critical invariant row should imply at least one positive proof, one negative proof, and one edge proof. If a reviewer cannot name those from the row, the invariant is still too vague.
