# Scenario

We are designing tenant-aware async export jobs for a Go REST service.

## Problem Frame

- Tenants can request data exports from the dashboard.
- Exports may take from seconds to several minutes.
- We want the API to stay responsive and not block the request.

## Candidate Decisions

- `POST /v1/exports` returns `202 Accepted` with an `export_id`.
- If the same user clicks twice, the frontend should disable the button, so backend deduplication is probably unnecessary.
- Export artifacts will be stored in object storage for 7 days.
- Job status will be polled via `GET /v1/exports/{id}`.
- Failed jobs can be retried by creating a new export.
- Cancellation is out of scope for v1.

## Constraints

- Must support multi-tenant isolation.
- Should avoid expensive synchronous DB reads on the request path.
- We do not want to introduce websockets or email delivery in v1.

## Open Assumptions

- [assumption] Frontend button disabling is enough to prevent meaningful duplicates.
- [assumption] A new export on retry is simpler than preserving retry semantics on the original resource.
- [assumption] Tenant ownership checks are straightforward because export IDs are UUIDs.
- [assumption] Seven-day artifact retention is acceptable for all tenants.

## Task

Run a pre-spec challenge pass on these candidate decisions. Focus on what could still change planning safely.
