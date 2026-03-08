# Assumptions And Uncertainties

- Assumed `POST /v1/payouts` may allocate a `payout_id` immediately while PSP submission continues asynchronously; if that is not feasible, the operation-resource design should be revisited.
- Assumed `GET /v1/payouts/{payout_id}` can be served strongly enough to support timeout recovery, while `GET /v1/payouts` remains projection-backed and eventual.
- Used `Idempotency-Key` as a required precondition because network timeouts are common; exact key format and maximum length are still unspecified.
- Left exact amount-scale limits, destination-type taxonomy, and operation-retention window as open questions rather than inventing values.
- No client-facing webhook contract was added; this draft intentionally keeps clients on polling only.
