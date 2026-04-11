# Blocked Specification Examples

## When To Load
Load this when the session cannot honestly approve `spec.md`: under-framed input, contradictory research, unresolved clarification questions, missing product policy, or phase drift.

## Good Session Outcomes
- The session says exactly why `spec.md` is not approved.
- The missing answer is routed to research, expert review, user decision, or the correct upstream checkpoint.
- `workflow-plan.md` and `workflow-plans/specification.md` agree on `blocked` or `reopened` state.
- The session stops without creating downstream artifacts.

## Bad Session Outcomes
- Marking unresolved product choices as harmless assumptions.
- Approving `spec.md` and relying on technical design to rediscover the real decision.
- Converting the blocker into a task item or implementation note.
- Asking the human for every uncertainty instead of using repository evidence or targeted research where possible.

## Blocker Handling
Example: unresolved SSO direction.

```text
Blocker: SAML vs OIDC, provisioning rules, and tenant isolation semantics are all unresolved.
Why it blocks: each answer can change the actors, acceptance criteria, data ownership, API behavior, and validation proof.
Spec state: blocked, not approved.
Next session starts with: targeted research or upstream framing, not technical-design.
```

Example: phase drift.

```text
Blocker: master workflow plan says current phase is technical-design, but the request asks to reopen specification casually.
Why it blocks: this session cannot rewrite an earlier phase without an explicit reopen target.
Spec state: unchanged unless the workflow plan records specification as reopened.
Next session starts with: the phase recorded by workflow-plan.md, or a formal reopen of specification.
```

Example: user-only policy.

```text
Blocker: retention duration is a product/business policy and cannot be derived from repository evidence.
Classification: requires_user_decision
Spec state: draft or partially blocked.
Next session starts with: specification after policy decision is available.
```

## Workflow Update Examples
Blocked `workflow-plans/specification.md`:

```text
Phase status: blocked
Readiness outcome: not spec-ready
Clarification challenge: not run because candidate decisions are incomplete
Blocker: provisioning semantics unresolved
Completion marker: not met
Stop rule: do not approve spec.md and do not begin technical design.
Next action: reopen targeted research or upstream framing for provisioning and tenant isolation.
```

Blocked `workflow-plan.md`:

```text
Current phase: specification
Current phase status: blocked
Session boundary reached: yes
Ready for next session: no
Next session starts with: research
Artifacts: spec.md blocked; design/ missing; plan.md missing; tasks.md missing
Clarification gate: not run; candidate decisions incomplete
Blockers: provisioning semantics and tenant isolation unresolved
```

## Exa Source Links
Exa MCP was attempted before these examples were authored, but `web_search_exa` and `web_fetch_exa` returned a 402 credits-limit error on 2026-04-11. These links are retained only as external calibration targets; `AGENTS.md` and `docs/spec-first-workflow.md` define the repository contract.

- [Atlassian - What is a Product Requirements Document?](https://www.atlassian.com/agile/requirements)
- [IBM - What is requirements management?](https://www.ibm.com/think/topics/what-is-requirements-management)
- [NASA - 4.2 Technical Requirements Definition](https://www.nasa.gov/reference/4-2-technical-requirements-definition/)
