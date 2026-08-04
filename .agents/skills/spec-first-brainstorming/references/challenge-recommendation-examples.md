# Challenge Recommendation Examples

## Behavior Change Thesis
When loaded for the symptom "the pre-spec challenge recommendation is hard to classify or needs sharper seams," this file makes the model tie `required`, `recommended`, or `skippable` to concrete risk seams instead of recommending challenge as ritual or skipping it because the change sounds small.

## When To Load
Load this when deciding whether the pre-spec challenge pass is `required`, `recommended`, or `skippable`, or when choosing the seams a challenger should pressure-test.

## The Move
`required` when hidden assumptions, edge semantics, ownership seams, identity, money, privacy, destructive action, long-running state, or failure behavior could materially change the spec — domain risk decides this, and it outranks how small the code change sounds. `recommended` when the frame is locally bounded but a second opinion would likely catch a meaningful ambiguity. `skippable` when the request is local, low-risk, and sharply bounded, with a skip rationale that still makes sense to a reviewer reading only the handoff. A useful recommendation names `1-3` seams to pressure-test; when no seam can be named, the challenge is not useful and the recommendation says so. The recommendation routes risk; architecture, API, data, security, reliability, rollout, and implementation-plan answers stay downstream.

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

## Reject
```markdown
Challenge Recommendation
recommended

Why
It is always good to have a challenge pass.
```

Review theater: it gives the challenger no seam to attack.
