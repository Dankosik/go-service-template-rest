# S3-compatible object storage

<!-- profile:object-storage:start -->
`OBJECT_STORAGE=s3` retains one fixed-authority adapter for a deliberately small
common surface: upload, download, metadata, unversioned delete, and presigned
GET. Amazon S3 and Cloudflare R2 are separate certified targets; one provider's
receipt never qualifies the other.

## Required immutable tuple

All fields use `APP__OBJECT_STORAGE__*`. Access key ID, secret access key, and
session token are credential material and must come from the environment; the
remaining fields may come from the normal non-secret configuration source.
Generated examples are deliberately empty and unusable.

| Provider | Required authority and identity |
| --- | --- |
| Amazon S3 | Commercial regional endpoint `https://s3.<region>.amazonaws.com`, matching region, dotless general-purpose bucket, 12-digit `EXPECTED_BUCKET_OWNER`, temporary access/secret/session credentials, and `MAX_OBJECT_BYTES` no greater than 5 TiB. |
| Cloudflare R2 | One 32-lowercase-hex account endpoint: default, `.eu.r2.cloudflarestorage.com`, or `.fedramp.r2.cloudflarestorage.com`; region `auto`, dotless bucket, static or temporary access/secret credentials, empty `EXPECTED_BUCKET_OWNER`, and `MAX_OBJECT_BYTES` no greater than 5 TiB minus 5 GiB. |

The shared bucket subset is 3–63 lowercase alphanumeric or hyphen characters,
starts and ends alphanumeric, and rejects Amazon's reserved prefixes and
suffixes (including account-regional `-an`). Supporting Amazon account-regional
names is a distinct addressing/ownership capability rather than an accidental
match on a broad DNS regex.

The SDK is constructed directly from that snapshot. It does not read AWS
profiles, ambient AWS variables, IMDS/ECS/web-identity providers, proxy settings,
endpoint overrides, retry mode, or a credential refresh chain. Rotation is an
owner-controlled rolling process replacement before a temporary credential
expires. Supporting SDK-refreshed workload identity is a separate capability,
not an implicit fallback.

Virtual-host addressing is mandatory. Redirects, alternate authorities,
private/loopback DNS results, path-style access, custom R2 domains, and TLS
verification against anything other than the image-owned public root snapshot
are rejected. Amazon sends `x-amz-expected-bucket-owner` on every mediated
request. The AWS presigner places it in the signed URL query; R2 receives no
such field. See the provider contracts for [Amazon expected bucket
owner](https://docs.aws.amazon.com/AmazonS3/latest/userguide/bucket-owner-condition.html)
and [R2 endpoint jurisdictions](https://developers.cloudflare.com/r2/reference/data-location/).

## Resource and failure policy

Deployment must supply positive finite object, multipart chunk, active-operation,
operation-duration, presign-lifetime, response-header, control-response, and
working-memory bounds. One adapter-wide non-blocking admission ceiling covers
all operations and retained download bodies. Uploads and downloads stream;
multipart parts and cleanup requests are serial. No operation spools an object
to disk or starts a background worker.

The object ceiling is provider-specific rather than nominally
“S3-compatible”: Amazon admits 5 TiB, while R2 documents an effective maximum
of 5 TiB minus 5 GiB. Startup rejects a larger R2 envelope before signing or
network I/O. See [R2 limits](https://developers.cloudflare.com/r2/platform/limits/).

`Upload` takes ownership of an `io.ReadCloser` and closes it exactly once. The
source's `Close` must promptly unblock `Read`; cancellation and deadline use
that property to release the operation slot without leaking a goroutine.

HeadObject and only the request-to-headers phase of GetObject may make three
attempts. Retry eligibility is limited to non-TLS network errors, HTTP
`429|500|502|503|504`, and the codes `RequestTimeout`,
`RequestTimeoutException`, `SlowDown`, and `TooManyRequests`. Mutations,
multipart stages, returned bodies, and presigning
remain one-attempt. A possibly transmitted upload, completion, or delete returns
`outcome_unknown`; the adapter never hides that ambiguity behind a generic
retry.

Create-only collision evidence stays provider-specific: Amazon/R2 documented
`412 PreconditionFailed` is `precondition_failed`; Amazon-only `409
ConditionalRequestConflict` is `temporary`, allowing a feature to make a fresh
call with a fresh source but never causing an adapter body replay. An R2 409 or
an unrecognized 412 remains conservative.

Amazon multipart cleanup performs at most three Abort/ListParts cycles and ten
1000-part pages per cycle. Every failed multipart upload remains `pending`,
including after an empty listing, because an in-flight part may finish after
abort. R2 likewise remains `pending`. Each deployment must also own an
exact-provider abandoned-multipart lifecycle rule. Amazon's API explicitly
notes that in-flight parts can finish after abort and repeated abort may be
needed: [AbortMultipartUpload](https://docs.aws.amazon.com/AmazonS3/latest/API/API_AbortMultipartUpload.html).

Mediated data uses CRC64NVME with `FULL_OBJECT` evidence. A download sends no
Range, rejects `206`/`Content-Range`, exposes no body without accepted checksum
metadata and a non-negative bounded `Content-Length`, and succeeds only at
clean checksum-matching EOF. ETag is never an
integrity or object-identity assumption. Closing early releases resources but
does not claim completion or cancellation unless the context was cancelled.
EOF, terminal error, explicit Close, cancellation, and deadline share one
exactly-once provider-body close and admission release.

Presigned output is GET-only, one second through the configured maximum and the
SigV4 seven-day ceiling. It is a reusable bearer URL: do not log, persist, or
send it to anyone other than the authorized recipient. An unsigned client
`Range` header can request a partial representation, so issuance authorizes
GetObject and does not prove a full transfer. R2 documents the same bearer and
seven-day boundary for its S3 API: [R2 presigned URLs](https://developers.cloudflare.com/r2/api/s3/presigned-urls/).
Before returning a URL, the adapter validates a single SigV4 algorithm,
signature, timestamp, exact requested lifetime, access-key/date/region/service
credential scope, session-token presence, signed-header set, key, authority,
and provider-specific expected-owner field.
`SignatureExpiresAt` is the expiry encoded in SigV4, not a guaranteed minimum
lifetime; credential expiry, revocation, or permission change may end access
earlier.

For absence, HeadObject maps only HTTP 404 with the pinned SDK's `NotFound` or
provider `NoSuchKey`; GetObject maps only HTTP 404 with exact `NoSuchKey`.
Generic 404/`NoSuchBucket` stays `internal`, while 401/403 stays `denied`.

Feature-visible errors stay in the closed object-storage vocabulary. Internal
spans may retain only bounded attempt count, numeric status, an allowlisted code,
closed failure category, and a sanitized provider request ID. Keys, bucket,
endpoint, credentials, provider text, checksum values, signed headers, and URL
query never enter errors, logs, metrics, or traces.

## Local proof and separate certification

Credential-free source and Linux process-envelope checks are:

```bash
S3_RECEIPT_PLATFORM=linux/amd64 make test-s3-source-receipt
S3_RECEIPT_PLATFORM=linux/arm64 make test-s3-source-receipt
GOMAXPROCS=1 S3_ENVELOPE_PLATFORM=linux/amd64 make test-s3-envelope
GOMAXPROCS=1 S3_ENVELOPE_PLATFORM=linux/arm64 make test-s3-envelope
```

The source receipt binds the Dockerfile's Go 1.26.6 and Distroless indexes, the
selected Linux/amd64 or Linux/arm64 manifests, nine Go source identities,
module pins, final-image root bundle, strict loader, and checked compiler escape
evidence for every retained D4 owner. Both the bundle parser and compiler proof
run inside the selected pinned Linux Go image. CI runs the amd64 source and
process-envelope gates; the arm64 receipts remain a separate required local
architecture check. A receipt applies only to the architecture it names and
does not prove a deployed image or runtime mount.

The process envelope retains the production fixed-authority client and a
separate local TLS-fixture client in the measured child, so it is a conservative
memory superset rather than a claim that the fixture is the production
transport. Focused tests separately prove production DNS/IP, authority, proxy,
redirect, hostname, and ambient-root denial.

`make test-s3-conformance-amazon` and `make test-s3-conformance-r2` are distinct,
fail-closed credentialed certification entrypoints. They require separately
authorized buckets, identities, lifecycle/versioning policy, TLS/DNS/egress,
cleanup, integrity, error, and presign receipts. Missing credentials leave only
that provider unverified; no local fixture, emulator, documentation review, or
other-provider result substitutes for it. The adapter does not provision,
change, probe, or certify a bucket during startup or readiness.

The isolated `test/s3conformance` package requires
`S3_CONFORMANCE_MUTATION_AUTHORIZED=1`, a unique lowercase run ID, the exact
provider tuple, primary and concealment credentials, provider-specific
versioning evidence, verified identity-policy/lifecycle receipts, and the
running image platform/digest plus live root-bundle hash/bytes/root count and
no-override attestation. It registers prefix-only cleanup before its first
mutation and exercises create-only/replace, multipart/abort, validated download,
metadata, delete, presigned execution and mutation rejection. Omitted or stale
inputs fail before client construction or provider I/O.
<!-- profile:object-storage:end -->
