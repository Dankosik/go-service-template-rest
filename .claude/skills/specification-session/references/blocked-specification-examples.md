# Blocked Specification Examples

## Behavior Change Thesis
When loaded for a specification session with under-framed input, contradictory evidence, unresolved clarification questions, product-only policy, or phase drift, this file makes the model choose an explicit blocked or reopened state instead of the likely mistake of approving `spec.md` with invented decisions or deferring approval-changing questions to technical design.

## When To Load
Load this when `spec.md` cannot honestly be approved and the session needs concrete blocked-state language or routing examples.

## Decision Rubric
- A blocker is not "missing information"; it is a missing answer that can change scope, correctness, ownership, rollout, acceptance, or validation proof.
- Classify the reopen target: upstream framing, targeted research, expert subagent work, `requires_user_decision`, or formal phase reopen.
- Ask the user only for product or business policy that cannot be derived from repository evidence or safe assumptions.
- Keep `spec.md` draft, partially blocked, or unchanged; do not mark it approved and hope later phases discover the gap.
- Make `workflow-plan.md` and `workflow-plans/specification.md` agree on blocked or reopened state before stopping.

## Imitate
Unresolved domain direction:

```text
Blocker: SAML vs OIDC, provisioning rules, and tenant isolation semantics are unresolved.
Why it blocks: each answer can change actors, acceptance criteria, data ownership, API behavior, and validation proof.
Spec state: blocked, not approved.
Next session starts with: targeted research or upstream framing, not technical-design.
```

Phase drift:

```text
Blocker: master workflow plan says current phase is technical-design, but the request asks to reopen specification casually.
Why it blocks: this session cannot rewrite an earlier phase without an explicit reopen target.
Spec state: unchanged unless workflow-plan.md records specification as reopened.
Next session starts with: the phase recorded by workflow-plan.md, or a formal reopen of specification.
```

User-only policy:

```text
Blocker: retention duration is a product/business policy and cannot be derived from repository evidence.
Classification: requires_user_decision
Spec state: draft or partially blocked.
Next session starts with: specification after the policy decision is available.
```

Copy the "why it blocks" line: it should name the decision surface that could change, not simply say more research is needed.

## Reject
False unblock:

```text
Assumption: retention lasts 90 days unless design says otherwise.
Spec status: approved.
```

This fails because a business-policy guess changes acceptance and validation.

Phase leak:

```text
Added a task for technical design to decide tenant visibility later.
```

This fails because technical design cannot own a missing specification decision.

## Agent Traps
- Treating a planning-critical blocker as a harmless assumption because the wording sounds narrow.
- Creating `tasks.md` or a design note to park the blocker.
- Asking the user for repository-discoverable facts instead of reopening targeted research.
- Leaving `workflow-plan.md` ready for `technical-design` while `workflow-plans/specification.md` says blocked.
