# Assumptions And Uncertainties

- Assumed webhook delivery uses a pre-registered partner endpoint referenced by `webhook_endpoint_id`; arbitrary callback URLs were intentionally excluded from `v1`.
- Assumed `POST /v1/catalog-import-jobs` can safely replay to recover an expired unused upload session while keeping the same `job_id`.
- Exact freshness SLA for the cache-backed job view is unknown, so the contract discloses eventual consistency with `version` and `consistency.as_of` instead of promising a hard max lag.
- The CSV business schema and any `schema_version` field were left out because the prompt focused on the transport and lifecycle contract, not row schema design.
- Retention period for completed jobs and row-level failure records is still open and may affect whether a downloadable failure artifact is needed.
