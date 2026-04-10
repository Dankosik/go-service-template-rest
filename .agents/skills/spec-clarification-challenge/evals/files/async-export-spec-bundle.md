# Scenario

The orchestrator is about to approve `spec.md` for tenant-aware async exports.

## Problem Frame

- Admin users need CSV exports of usage data.
- Synchronous export requests time out for larger tenants.
- The first version should keep the API responsive and avoid websocket or email delivery.

## Scope / Non-goals

- In scope: create export job, poll job status, download ready artifact.
- In scope: tenant-scoped access to every export and artifact.
- Out of scope: cancellation, email delivery, scheduled recurring exports.

## Candidate Decisions

- `POST /v1/exports` returns `202 Accepted` with an `export_id`.
- The UI disables the button after click, so backend idempotency is not needed for v1.
- Export artifacts live in object storage for 7 days.
- `GET /v1/exports/{id}` returns `queued`, `running`, `ready`, or `failed`.
- Failed jobs are retried by creating a new export.
- The generated artifact is downloaded through a signed URL.

## Constraints And Validation Expectations

- The API must preserve tenant isolation.
- Job status must survive service restarts.
- Validation should prove status transitions, scoped download access, and failed-job visibility.

## Known Assumptions / Open Questions

- [assumption] UUID export IDs plus tenant lookup are enough to protect export access.
- [assumption] Seven-day retention is acceptable for every tenant.
- [assumption] Duplicate clicks are rare enough to ignore at the backend.
- [open] Exact object-storage cleanup proof is not written yet.

## Research Links

- `research/object-storage.md`
- `research/async-job-status.md`
