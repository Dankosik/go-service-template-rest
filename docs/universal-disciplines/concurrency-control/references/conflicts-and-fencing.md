# Conflicts And Fencing

Load when arbitration can lose, retry, expire, transfer ownership, or overlap.

Give every loser one bounded disposition: recompute from fresh state and retry
within attempt/time budgets, return a caller-visible conflict, merge under an
accepted domain rule, or serialize per key. Replaying stale intent recreates the
race. External effects retain the same operation identity across retries.

Leader election, singleton workers, and distributed cron permit overlap during
pause, partition, and failover. Treat the term as a lease and its generation as
a fencing token; every durable effect conditionally checks the current
generation. A scheduled occurrence also needs a stable identity and unique
claim.

Cover reachable windows: crash while holding, pause past expiry, arbiter
unavailability, deadlock, and hot-key retry amplification. Each has a guard and
operator signal. Fencing proof pauses the old holder through expiry, lets a new
holder commit, resumes the old holder, and observes its write rejected.
