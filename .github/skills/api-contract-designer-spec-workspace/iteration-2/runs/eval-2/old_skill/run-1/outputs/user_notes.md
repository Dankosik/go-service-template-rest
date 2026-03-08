# Assumptions And Uncertainties

- Assumed model: each authenticated customer has exactly one editable profile. If that is false, `/v1/profile` is too narrow and the URI needs an explicit identifier.
- Assumed consistency: profile edits can honor a strong read-after-write contract. If the backing write path is eventually consistent, the API contract needs explicit freshness disclosure or an async redesign.
- Main rollout uncertainty: existing `POST /v1/profile/update` consumers may not be able to send `If-Match` or `Idempotency-Key`; if so, the legacy endpoint needs a time-boxed compatibility exception with documented lost-update risk.
- Main compatibility uncertainty: the current legacy error body is unknown. If clients depend on that exact shape, the old endpoint should keep it during coexistence even though the new `/v1/profile` surface uses `application/problem+json`.
