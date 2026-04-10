# Scenario

The orchestrator is about to approve `spec.md` for an account-summary Redis cache.

## Problem Frame

- The account summary endpoint is read-heavy and stresses Postgres.
- The first version should reduce DB load without changing the public endpoint.

## Scope / Non-goals

- In scope: read-through Redis cache for account summary responses.
- Out of scope: separate projection service, endpoint response changes, write-model redesign.

## Candidate Decisions

- Cache summary responses for 10 minutes.
- Use cache key `account_summary:{account_id}`.
- Skip invalidation for v1 because TTL is simpler.
- If Redis is unavailable, fall back to Postgres.
- Roll out globally once tests pass.

## Constraints And Validation Expectations

- Summary data is user-visible and used by support.
- The API response shape cannot change in this iteration.
- Validation should prove fallback behavior and cache-hit behavior.

## Known Assumptions / Open Questions

- [assumption] Ten minutes of stale summary data is acceptable.
- [assumption] `account_id` is globally unique and does not need tenant prefixing.
- [assumption] Redis fallback cannot overload Postgres during an outage.
- [open] No rollout cohort or kill-switch expectation is recorded yet.

## Research Links

- `research/account-summary-query.md`
