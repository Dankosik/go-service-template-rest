# Assumptions And Uncertainties

- Assumed v1 uses one async job resource plus a returned upload URL, rather than a separate upload-session resource.
- Assumed webhook delivery references a pre-registered `webhook_endpoint_id`; if partners need per-job arbitrary callback URLs, the contract should change.
- Assumed the `50 MiB` limit applies to the raw uploaded CSV and that only `text/csv` is accepted in v1.
- Assumed malware scan, CSV parsing, and item processing failures surface as terminal job state, not as late synchronous API errors.
- Assumed the cache-backed read model can document a `10s` freshness target, but the exact lag SLO still needs confirmation.
