# User Notes

- Assumed `50 MB` means `50,000,000` bytes. If the product actually means `50 MiB`, update the limit and examples.
- Assumed the upload itself uses an opaque presigned `PUT` URL outside the versioned REST surface because a direct 50 MB API upload plus mandatory malware scanning would make acceptance semantics muddy.
- Assumed signed webhooks use a partner account-level secret configured outside this contract; the per-job request only supplies the callback URL.
- Assumed `failed` is a strong invariant meaning no catalog changes from that job were committed. Partial row acceptance appears only under `completed_with_errors`.
- The job `GET` and failure-list `GET` are intentionally eventual-consistency views; a signed terminal webhook may arrive before those cache-backed reads converge.
