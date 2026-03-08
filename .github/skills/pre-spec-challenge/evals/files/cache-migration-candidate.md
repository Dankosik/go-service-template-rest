# Scenario

We want to add Redis caching in front of a slow account-summary query.

## Problem Frame

- The account summary endpoint is read-heavy and currently stresses Postgres.
- We want to lower DB load quickly without redesigning the whole read model.

## Candidate Decisions

- Add read-through Redis cache with a 10-minute TTL.
- Cache key will be `account_summary:{account_id}`.
- We will skip invalidation for now because TTL should be enough.
- During rollout, we will enable cache for all tenants at once.
- If Redis is unavailable, we will just fall back to Postgres.
- We are not planning a backfill or warmup strategy.

## Constraints

- We cannot change the endpoint shape in this iteration.
- Summary data is user-visible and can affect support decisions.
- We do not want to add a separate projection service.

## Open Assumptions

- [assumption] Ten minutes of staleness is acceptable because the endpoint is informational.
- [assumption] `account_id` alone is a sufficient cache key.
- [assumption] Skipping invalidation is fine as long as Redis can expire data naturally.
- [assumption] Global rollout is safe because fallback to Postgres exists.

## Task

Run a pre-spec challenge pass on these candidate decisions. Focus on what could still change planning safely.
