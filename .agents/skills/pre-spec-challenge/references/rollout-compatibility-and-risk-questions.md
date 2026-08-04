# Rollout Compatibility And Risk Questions

## Behavior Change Thesis
When loaded for rollout or compatibility claims, this file makes the model test mixed-version and rollback state instead of forcing release ceremony or ignoring hard-to-reverse deployment risk.

## When To Load
Load this when the candidate synthesis mentions rollout, migration, feature flags, canary, backward compatibility, mixed versions, destructive state changes, data backfill, or rollback. Tiny, local, easily reverted work with no persisted state or external behavior needs none of these questions.

## The Move
Challenge mixed-version safety wherever old and new binaries, clients, workers, or schema may coexist — including the old-client and old-worker behavior the candidate path leaves undescribed. Challenge rollback wherever new writes, cache entries, artifacts, or side effects can remain after disabling the feature: a flag disables code paths without undoing durable effects, and a fallback that preserves availability can hide the load, tenant-isolation, or correctness failure it should surface. Ask for a canary, flag, or rollout percentage only when it changes implementation order, observability, blast radius, or cleanup obligations, and only with a cohort able to detect the specific risk.

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
  - Ceremony without a named failure mode or guardrail.
- "Use a feature flag to make it safe."
  - A flag may not undo writes, artifacts, cache entries, or external side effects.
