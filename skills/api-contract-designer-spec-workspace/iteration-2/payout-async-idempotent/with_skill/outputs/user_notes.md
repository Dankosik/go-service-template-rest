# Assumptions And Uncertainties

- Assumed `GET /v1/payouts/{payout_id}` can read a canonical current state strongly enough to support timeout recovery; if not, the contract needs a different recovery story.
- Assumed one payout-initiation operation exists per payout and no client-visible cancellation path is in scope.
- Proposed `Idempotency-Key` retention as at least `24h` and never shorter than the associated payout's non-terminal lifetime; this should be confirmed against platform storage and compliance limits.
- Left exact amount precision rules, supported currency set, destination eligibility rules, and operation-retention expiry behavior as open questions instead of inventing them.
- Intentionally kept the contract poll-only; no webhook surface is included in this version.
