# User Notes

- Assumed `GET /v1/payouts/{payout_id}` can provide a stronger canonical read than the eventual payout-history projection. If not, the timeout-recovery contract must be reopened.
- Assumed `POST /v1/payouts` requires `Idempotency-Key` on every call because network timeouts are common and duplicate payouts are unacceptable.
- Left webhook/callback delivery out of scope. This contract is poll-based only.
- Left handler behavior, storage design, PSP integration details, and rollout steps out of scope on purpose.
- Still open: operation-resource retention/expiry semantics, exact `422` validation rules for amount/currency/destination eligibility, and whether `client_reference` carries any uniqueness semantics beyond being an informational field.
