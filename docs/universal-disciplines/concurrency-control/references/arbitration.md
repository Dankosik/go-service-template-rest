# Arbitration

Load when choosing how durable writers resolve a proven schedule.

Stop at the first mechanism that closes the schedule at the actual isolation
level:

1. one atomic conditional statement;
2. unique/exclusion constraint with a handled conflict;
3. optimistic compare-and-set/version or ETag;
4. pessimistic row lock with one acquisition order and no user/external wait;
5. serializable transaction with bounded retry for cross-row write skew;
6. advisory/application lock when no row can arbitrate;
7. distributed lease only when no transactional arbiter exists.

Name the isolation level and the anomaly the mechanism excludes. A lock that
suppresses duplicate work does not authorize a stale result; a distributed
lease needs a monotonic fencing token checked by the effect owner, or durable
idempotency and reconciliation.

The proof deliberately produces one winner and exercises the losing conflict
path at the real store. Reopen the schema owner for a new constraint and the
performance owner only when contention or lock cost becomes a separate pressure.
