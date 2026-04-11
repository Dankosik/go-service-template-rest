# Approach Comparison And Direction Selection

Behavior Change Thesis: When loaded for the symptom "multiple plausible behavior-level frames remain, but the task is beyond raw ideation," this file makes the model compare framing approaches and recommend or block one direction instead of the likely mistake of carrying every option forward or drifting into architecture design.

## When To Load
Load this when the user has a mostly chosen direction but the frame could still split into `2-3` plausible problem interpretations, and the choice affects downstream specialist lanes.

## Decision Rubric
- Compare frames, not implementations. Each option should state a different behavior problem or success signal.
- Keep trade-offs at the level of actor, behavior delta, constraint, and risk.
- Recommend one option only when evidence favors it; otherwise mark readiness `fail` and name the missing discriminator.
- Do not use this reference for raw product ideation. If no direction is chosen, return to `idea-refine`.
- Do not turn options into architecture proposals such as "Redis vs Postgres" or "polling vs webhooks."

## Imitate
Input:
```text
Users say exports are slow. Maybe make reports async or cache the generated files.
```

Output:
```markdown
Approaches
1. Async-progress frame: Users can submit a report request and see clear lifecycle status while generation completes later.
   Trade-off: Best fit if the pain is request timeout or lack of progress visibility; leaves worker, storage, and API contract design downstream.
2. Reuse/cache frame: Users repeatedly request the same report and wait for recomputation each time.
   Trade-off: Best fit if duplicate expensive generation is the dominant pain; leaves cache key, freshness, and invalidation design downstream.

Recommended Direction
fail - The prompt does not say whether the pain is first-run latency, repeated recomputation, or missing status after acceptance. Gather one workflow example or timing trace before deeper design.
```

Copy: both options are behavior frames, and the output refuses to pick without a discriminator.

Input:
```text
Admins need better audit visibility after account changes.
```

Output:
```markdown
Approaches
1. Investigation-history frame: Admins need to reconstruct who changed account-sensitive fields and when.
   Trade-off: Focuses downstream work on event completeness, redaction, retention, and tenant visibility.
2. Real-time-alerting frame: Admins need immediate notification when suspicious account changes happen.
   Trade-off: Focuses downstream work on detection rules, alert routing, and false-positive handling.

Recommended Direction
Choose the investigation-history frame unless the user specifically reports missed real-time intervention. The current wording says "visibility after account changes," which points to reconstruction rather than alerting.
```

Copy: the recommendation follows wording evidence and still leaves API/data/security decisions unmade.

## Reject
Bad:
```markdown
Approaches
1. Use Redis for report cache.
2. Add a jobs table and polling endpoint.
3. Stream report progress over websockets.
```

Why: those are implementation designs, not competing problem frames.

Bad:
```markdown
Recommended Direction
Do all of them so downstream design can decide later.
```

Why: carrying every plausible direction forward defeats the purpose of brainstorming; either pick a frame or mark the discriminator as blocking.

## Agent Traps
- Do not compare options that differ only by implementation mechanism while solving the same behavior problem.
- Do not recommend the option with the fanciest architecture. Recommend the option best supported by the user's symptom.
- Do not hide indecision as "support both." If both are real, split scope or fail readiness until a product owner chooses.
