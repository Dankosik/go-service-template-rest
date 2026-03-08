# User Notes

- Assumed `50 MB` means `50,000,000` bytes. If the product actually means `50 MiB`, the contract limit and examples need to change.
- Assumed the large-file path uses an opaque upload-session target instead of a direct API multipart upload because mandatory malware scanning makes direct synchronous acceptance misleading.
- Assumed signed completion webhooks use a partner account-level secret provisioned outside this contract; the per-job request only supplies the callback URL.
- Assumed `failed` is a strong client-visible invariant meaning the job applied no catalog changes at all. Partial row acceptance appears only under `completed_with_errors`.
- The cache-backed job view is intentionally eventual. A valid signed terminal webhook may arrive before `GET /v1/catalog-import-jobs/{job_id}` converges.
