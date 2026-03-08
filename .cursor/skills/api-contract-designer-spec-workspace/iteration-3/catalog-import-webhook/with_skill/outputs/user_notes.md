# User Notes

- Assumed `50 MB` means `50,000,000` bytes for the API limit. If product wants `50 MiB`, the examples and `413` boundary need to change.
- Assumed webhook delivery uses a partner's pre-registered default completion webhook target and active signing secret, not an arbitrary per-job callback URL.
- Assumed `failed` is a strong invariant meaning no catalog changes from that job were applied; partial row rejection is represented only by `succeeded_with_errors`.
- Assumed the job `GET` and `/failures` endpoints are intentionally cache-backed eventual-consistency views, so a terminal webhook can lead polling.
- If resumable uploads, multiple webhook targets, or a published freshness SLA are mandatory, this contract needs a follow-up revision before implementation.
