# Assumptions And Uncertainties

- I assumed `POST /v1/payouts` belongs to the authenticated caller’s payout scope, so `vendor_id` is not part of the body or headers.
- I assumed a separate strong item read, `GET /v1/payouts/{payout_id}`, is acceptable even though payout-history collection reads are eventual.
- I assumed `destination_id` references an already verified payout destination resource and that raw bank-account entry is out of scope here.
- I assumed `Idempotency-Key` is mandatory for payout creation because retries after network timeouts are a core requirement.
- I left `client_reference` as an echoed field, not a uniqueness guarantee, because long-lived business dedup requirements were not specified.
