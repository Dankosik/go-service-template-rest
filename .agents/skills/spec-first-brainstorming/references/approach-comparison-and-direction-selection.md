# Approach Comparison And Direction Selection

## Behavior Change Thesis
When loaded for the symptom "multiple plausible behavior-level frames remain, but the task is beyond raw ideation," this file makes the model compare framing approaches and recommend or block one direction instead of carrying every option forward or drifting into architecture design.

## When To Load
Load this when the user has a mostly chosen direction but the frame could still split into `2-3` plausible problem interpretations, and the choice affects downstream specialist lanes.

## The Move
Compare frames, not implementations: each option states a different behavior problem or success signal, and options that differ only by mechanism while solving the same problem are one frame. Keep trade-offs at the level of actor, behavior delta, constraint, and risk. Recommend the option the user's symptom actually supports; when no discriminator exists, mark readiness `fail` and name the missing one — indecision stated as "support both" hides the choice, so when both frames are real, split scope or fail readiness until a product owner chooses. When no direction is chosen at all, return to `idea-refine`.

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

## Reject
```markdown
Approaches
1. Use Redis for report cache.
2. Add a jobs table and polling endpoint.
3. Stream report progress over websockets.
```

Those are implementation designs, not competing problem frames.
