# Rollout Compatibility And Risk Questions

## Behavior Change Thesis
When loaded for rollout or compatibility claims, this file makes the model test mixed-version and rollback state instead of forcing release ceremony or ignoring hard-to-reverse deployment risk.

## When To Load
Load this when the candidate synthesis mentions rollout, migration, feature flags, canary, backward compatibility, mixed versions, destructive state changes, data backfill, or rollback. Do not use it to force rollout ceremony onto tiny reversible local work.

## Decision Rubric
- Skip rollout questions for tiny, local, easily reverted work with no persisted state or external behavior.
- Challenge mixed-version safety when old and new binaries, clients, workers, or schema may coexist.
- Challenge rollback when new writes, cache entries, artifacts, side effects, or irreversible state can remain after disabling the feature.
- Challenge global rollout when fallback can hide rather than solve load, tenant-isolation, or correctness failures.
- Ask for a canary or flag only if it changes implementation order, observability, blast radius, or cleanup obligations.
- Do not ask for a rollout percentage unless the representative population and stopping signal matter to planning.

## Imitate
- "Can old and new binaries read the same persisted state during the rollout, and what breaks if rollback happens after new writes?"
  - Copy the mixed-version plus rollback-after-write shape.
- "If Redis fallback sends all traffic back to Postgres during global enablement, what guardrail prevents a load spike from becoming the rollback trigger too late?"
  - Copy the test that fallback may preserve availability while hiding capacity risk.
- "If the feature flag is turned off after partial exposure, which data or async side effects remain and who cleans them up?"
  - Copy the distinction between disabling code paths and undoing durable effects.
- "Does the migration require expand-contract ordering, or can code and schema deploy together safely?"
  - Copy the question that can change implementation phasing.

## Reject
- "Should we canary?"
  - Fails because it adds ceremony without naming the failure mode or guardrail.
- "Is rollback safe?"
  - Fails because it does not identify the state that survives rollback.
- "Use a feature flag to make it safe."
  - Fails because a flag may not undo writes, artifacts, cache entries, or external side effects.
- "What percentage should launch first?"
  - Fails unless cohort size affects the ability to detect the specific risk.

## Agent Traps
- Treating fallback as rollback when durable state or load shape has already changed.
- Treating a feature flag as cleanup.
- Requiring a canary for every change rather than only when it reduces planning risk.
- Ignoring old-client or old-worker behavior because the candidate path only describes new code.
