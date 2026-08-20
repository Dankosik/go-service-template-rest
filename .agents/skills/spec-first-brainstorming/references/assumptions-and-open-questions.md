# Assumptions And Open Questions

## Load When
Load this when the request says "should be fine," "obviously," "just," "we know," or similar, or when the open questions do not say who owns the answer and what answer would unblock design.

## The Move
Label every uncertainty and decide everything technical:

- `[assumption]` marks an uncertain fact, never disguised as a decision; each carries its risk and validation path, and one with no risk does not belong here.
- `[decided]` records an agent-owned architecture, API, data, security, reliability, or rollout branch chosen from current evidence, with its reopen condition. Mechanism is agent-owned: a technical menu never reaches the user.
- `[blocks design]` routes a branch a later phase owns; it stays agent-owned there and never becomes a user question in transit.
- `[user-owned]` marks only an item that survives [Decision Ownership](../../../../AGENTS.md#decision-ownership), with an owner and an unblock condition precise enough for the orchestrator to act on.
- `[nice to know]` holds anything that cannot block readiness.

Ask the questions that change framing or route the next spec lanes; everything else is a labeled assumption or a decision.

## Inspect
Input:
```text
Let's require email verification before users can create projects. That should stop abuse.
```

Output:
```markdown
Assumptions
- [assumption] Unverified email is a material abuse path.
  Risk: The change may add user friction without reducing the dominant abuse mode.
  Validation path: Check abuse reports or logs for correlation with unverified accounts.
- [assumption] Project creation is the right enforcement point.
  Risk: Abuse may happen before or after project creation, or require different controls.
  Validation path: Map the user action where abuse becomes costly before security design.

Decisions
- [decided] Existing unverified users keep their projects and are prompted, not blocked; enforcement applies to new project creation only.
  Reopen condition: The abuse evidence shows existing unverified accounts are the dominant source.

Open Questions
- [user-owned] What abuse behavior must change?
  Why it survives: Both "stop throwaway-account signup floods" and "stop shared-credential resale" are defensible, they lead to different controls, and neither can be assumed honestly from repository evidence.
  Owner: product/security framing.
  Unblock condition: A concrete abuse scenario and success signal.
```

Copy: the one branch that changes the accepted outcome reaches the user; the compatibility branch is decided with its reopen condition.

## Reject
```markdown
Open Questions
- [blocks design] Should dedupe live in middleware, the handler, or a unique index?
  Owner: user.
```

It hands the user a technical menu. Mechanism is agent-owned; decide it and record the reopen condition.
