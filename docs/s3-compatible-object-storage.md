# S3-compatible object storage

<!-- profile:object-storage:start -->
`OBJECT_STORAGE=s3` retains a small `internal/objectstorage.Store` port for
upload, download, metadata, delete, and presigned GET. Business code owns keys,
authorization, content policy, retention, object size, create-only intent, and
presign recipients. It never signs requests, parses provider errors, manages
multipart state, or calculates checksums.

`internal/infra/s3` is a thin adapter over AWS SDK for Go v2. The SDK owns
SigV4, HTTP/XML, endpoint resolution, credential retrieval, retry execution,
checksums, presigning, and provider error decoding. Its transfer manager owns
multipart parts, completion, and the immediate abort attempt.

## Deployment configuration

All application fields use `APP__OBJECT_STORAGE__*`:

| Field | Amazon S3 | Cloudflare R2 |
| --- | --- | --- |
| `PROVIDER` | `amazon_s3` | `cloudflare_r2` |
| `BUCKET` | Required dotless DNS bucket | Required dotless DNS bucket |
| `REGION` | Required commercial AWS region | `auto` (defaulted when empty) |
| `ENDPOINT` | Empty; the SDK resolves the regional endpoint | Exact `https://<account>[.eu|.fedramp].r2.cloudflarestorage.com` origin |
| `EXPECTED_BUCKET_OWNER` | Required 12-digit account ID | Empty; R2 does not support the field |
| `CREDENTIAL_SOURCE` | `aws_default` or `static` | `static` |
| `MAX_OBJECT_BYTES` | Positive, at most 80 GiB | Positive, at most 80 GiB |

`credential_source=aws_default` delegates to the AWS SDK credential chain.
Production should prefer workload identity. `static` reads the standard
`AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`, and optional `AWS_SESSION_TOKEN`
environment variables. No credential belongs in YAML, logs, errors, metrics,
or traces.

The 80 GiB product ceiling follows the fixed 8 MiB part size and S3's 10,000
part limit. Raise it only with a measured buffer policy for a real adopter.

## Runtime policy

- Small uploads use SDK `PutObject` and remain streaming. Uploads above 8 MiB
  use transfer manager with one part worker, an 8 MiB threshold/part size, and
  a five-second failure-cleanup timeout.
- The adapter computes no checksum. It explicitly requests CRC64NVME
  `FULL_OBJECT`; the SDK calculates uploads and validates fully consumed
  downloads.
- Create-only is supported only for small `PutObject` uploads. Portable
  conditional multipart completion is not claimed.
- The SDK standard retryer owns at most three attempts under the caller's
  context. A failed mutation is conservatively `ErrOutcomeUnknown`, except a
  documented `412 PreconditionFailed`, which is `ErrAlreadyExists`.
- One process-local four-operation admission ceiling rejects excess work with
  `ErrBusy`. A retained download holds its slot until EOF, error, or `Close`.
- The caller owns the upload reader and operation context. The caller must close
  every successful download body.
- The HTTP transport rejects redirects, disables proxies, uses public system
  roots with TLS 1.2+, and caps connections and response headers. Amazon uses
  the SDK's default endpoint resolver; R2 uses `BaseEndpoint`.
- Object storage is not a readiness dependency. Startup constructs the client
  without probing or mutating the provider.

An immediate multipart abort is best effort. Every deployment must also own an
exact-provider lifecycle rule for abandoned multipart uploads. No synchronous
result promises that all uploaded parts have disappeared.

## Local proof and provider certification

Credential-free local proof is part of ordinary Go and profile checks:

```bash
go test ./internal/objectstorage ./internal/infra/s3
go test ./internal/objectstorage ./internal/infra/s3
```

Provider certification remains two distinct, mutation-authorized gates:

```bash
make test-s3-conformance-amazon
make test-s3-conformance-r2
```

Amazon certification requires its exact region, bucket owner, credential mode,
bucket versioning/lifecycle policy, TLS/egress path, checksum, multipart,
conditional write, delete, and presign evidence. R2 certification independently
requires its account/jurisdiction endpoint, region `auto`, credential and
lifecycle facts, checksum/multipart compatibility, conditional write, delete,
and presign evidence. One provider's receipt never qualifies the other, and no
fixture, emulator, SDK unit test, or documentation review substitutes for a
live exact-provider receipt.
<!-- profile:object-storage:end -->
