# Challenge Recommendation Examples

Behavior Change Thesis: When loaded for the symptom "the pre-spec challenge recommendation is hard to classify or needs sharper seams," this file makes the model tie `required`, `recommended`, or `skippable` to concrete risk seams instead of the likely mistake of recommending challenge as ritual or skipping it because the change sounds small.

## When To Load
Load this when deciding whether the pre-spec challenge pass is `required`, `recommended`, or `skippable`, or when choosing the seams a challenger should pressure-test.

## Decision Rubric
- `required`: hidden assumptions, edge semantics, ownership seams, identity, money, privacy, destructive action, long-running state, or failure behavior could materially change the spec.
- `recommended`: the frame is locally bounded but a second opinion would likely catch a meaningful ambiguity.
- `skippable`: the request is local, low-risk, already sharply bounded, and the skip rationale names what was checked.
- A useful recommendation names `1-3` seams to pressure-test. If no seam can be named, do not pretend the challenge is useful.
- The recommendation routes risk; it does not answer architecture, API, data, security, reliability, rollout, or implementation-plan questions.

## Imitate
Input:
```text
Make password reset tokens single-use and expire faster.
```

Output:
```markdown
Challenge Recommendation
required

Why
Password reset touches identity and account recovery. Hidden assumptions about token reuse, existing sessions, replay windows, audit needs, and user lockout could materially change the spec.

Seams To Pressure-Test
- Security semantics: What abuse case is being prevented, and what recovery behavior must remain usable?
- Domain semantics: What happens to already-issued tokens when expiration or single-use behavior changes?
- Observability and support: What signals are needed without leaking token or account details?
```

Copy: `required` is justified by identity and migration semantics, not by a generic security label.

Input:
```text
Add a query parameter to filter projects by archived=true.
```

Output:
```markdown
Challenge Recommendation
skippable

Why
The request is local and low risk if research confirms `archived` is already an existing project state and the filter does not change authorization, pagination, or default visibility semantics.

Seams To Pressure-Test If Skipping Is Reconsidered
- Whether archived projects have distinct authorization or tenant visibility behavior.
- Whether adding the filter changes default result ordering or pagination expectations.
```

Copy: the skip is conditional and names the small set of assumptions that would change the call.

Input:
```text
Queue invoice finalization so checkout responds faster.
```

Output:
```markdown
Challenge Recommendation
required

Why
Invoice finalization likely touches money, external side effects, consistency, retries, and user-visible checkout state. Queueing could change correctness even if it improves latency.

Seams To Pressure-Test
- Domain invariant: When is an invoice considered final from the customer's perspective?
- Distributed consistency: What side effects must be atomic, idempotent, or recoverable?
- Reliability: What happens if queued finalization fails after checkout responds?
```

Copy: "faster checkout" is treated as a correctness risk because money and async side effects are in scope.

## Reject
Bad:
```markdown
Challenge Recommendation
recommended

Why
It is always good to have a challenge pass.
```

Why: this creates review theater and gives the challenger no seam to attack.

Bad:
```markdown
Challenge Recommendation
skippable

Why
It is just adding a queue.
```

Why: queueing invoice finalization can change payment correctness and user-visible state.

## Agent Traps
- Do not let "small code change" override domain risk.
- Do not require challenge just because the workflow has one available; name the seam.
- Do not choose `skippable` unless the skip rationale would still make sense to a reviewer reading only the handoff.
