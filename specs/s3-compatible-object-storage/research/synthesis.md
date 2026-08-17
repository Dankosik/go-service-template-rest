# S3-Compatible Object Storage Research Synthesis

Status: Research complete; ready for Specification

Independent review: PASS on 2026-08-11 after focused transport-trust repair

Evidence snapshot: 2026-08-11

Repository baseline: `40e6d212799ae8677b675339929c559246536181` plus the current tracked tree

Provisional profile selector: `OBJECT_STORAGE=s3`

## Research boundary

This note researches an optional template-init-selected object-storage pack that
could give generated feature code bounded upload, download, metadata, delete,
and presigned-access operations. It does not define the behavioral contract,
choose an implementation, design packages or configuration, create tasks, or
change production code.

The intended ownership boundary is part of the research question:

- the reusable pack may own S3 transport mechanics and safe operational bounds;
- deployment may own the bucket, credentials, network path, encryption,
  lifecycle, and provider controls;
- each feature retains object keys, authorization, retention intent, content
  policy, and the business decision to create, replace, expose, or delete an
  object.

## Executive synthesis

1. The current template has no object-storage configuration, runtime client,
   profile selector, generated surface, or S3 dependency. This would be a new
   optional pack, not an extension of an established contract.
2. “S3-compatible” names a family of APIs, not a shared behavioral guarantee.
   AWS S3 consistency, encryption defaults, error meanings, checksum support,
   versioning, Object Lock, IAM, and presigned-policy controls apply elsewhere
   only when that provider documents and proves them.
3. The AWS SDK for Go v2 is the current official AWS client family and is
   actively maintained. Its defaults are not ready-made template policy:
   ambient credential discovery, endpoint rewriting, three-attempt retries,
   automatic checksum selection, and transfer-manager buffering must be made
   explicit or bounded. Its new transfer manager is still pre-v1.
4. A maintained S3-compatible client such as `minio-go/v7` is a real alternative,
   not proof of broader compatibility. Cross-provider abstractions are useful
   only if the accepted feature contract is their smaller common subset.
5. Bounded direct HTTP does not survive the requested surface: owning SigV4,
   XML errors, endpoint addressing, credential refresh, multipart completion and
   abort, checksums, presigning, and provider variants would be more protocol
   machinery than using a maintained client. It reopens only if the requested
   contract shrinks materially.
6. Presigned transfers bypass the service process. The pack can bound what it
   signs—operation, key, expiry, headers, and preconditions—but cannot enforce
   process body or concurrency limits on the later transfer. Portable one-time
   URLs and presigned PUT size enforcement were not found.
7. Local fakes and S3 emulators can prove feature and adapter mechanics, but only
   an exact-provider test proves credentials, addressing, signed headers,
   checksum paths, conditional writes, cleanup, and compatibility.

The strongest hypothesis for Specification is a small feature-facing object
port with one S3 adapter, explicit provider capability claims, and no bucket or
content-policy ownership. AWS SDK v2 and a purpose-built S3-compatible client
remain adapter candidates subject to a pinned transfer strategy and
exact-provider conformance proof. This is a downstream hypothesis, not a
Research selection.

## Open-item map

| Research question | Method and strongest evidence | Research disposition | Downstream owner |
| --- | --- | --- | --- |
| Is there an existing repository owner to extend? | Current-tree navigation of config, bootstrap, outbound HTTP, telemetry, readiness, profiles, manifests, and CI | No object owner exists. Reuse current extension seams and policies; do not infer an existing S3 contract. | Specification, then design |
| What does S3 compatibility guarantee? | AWS API/user guides plus current R2, DigitalOcean, Backblaze, GCS, OCI, and Railway provider documents | Only operation-by-operation and field-by-field claims are defensible. | Specification |
| Which credential behavior is safe and portable? | AWS SDK configuration source/guide and provider credential docs | Credential mode must be explicit. AWS workload identity and compatible-provider static keys are distinct capabilities. | Specification/security design |
| Can endpoint, region, bucket, and trust be separate? | SDK endpoint resolver and SigV4 flow; repository fixed-authority transport | No. Together they determine the signed authority and credential exposure boundary. | Specification/security design |
| What bounds are intrinsic to SDK upload/download? | SDK source and guides | Low-level and transfer clients do not provide a process-wide body, memory, goroutine, or concurrency bound. | Specification/performance design |
| Are retries and writes idempotent by default? | SDK retry guide plus PutObject and multipart contracts | No blanket rule. Retry safety depends on operation, precondition, body replayability, and multipart state. | Specification/reliability design |
| Are checksums and encryption portable? | AWS checksum/encryption contracts plus provider compatibility matrices | No. Separate wire verification, stored digest, opaque ETag/version, provider at-rest encryption, and request-selected encryption. | Specification/deployment design |
| Can presigned access retain service bounds and authorization? | AWS/R2 presign contracts and POST-policy behavior | Authorization is decided before signing; the resulting URL is a reusable bearer credential. Direct transfer is outside service runtime bounds. | Specification/security design |
| Which errors can be stable across providers? | AWS/Smithy errors, HEAD/DELETE behavior, provider matrices | Only a small conservative taxonomy is plausible; 403/404 and delete finality are not portable without evidence. | Specification/API design |
| What telemetry and readiness are reusable? | SDK observability guide and repository adapter/readiness rules | Instrument adapter operations with bounded dimensions. Presigned execution and provider control-plane health need separate evidence. | Specification/observability design |
| Which implementation families remain live? | Candidate sweep across SDKs, abstractions, direct protocol, managed facilities, and no pack | Six distinct families cover the live decision level; additional vendors instantiate an existing family. | Specification |
| How can local proof avoid weakening production semantics? | Emulator limitations, Testcontainers state, provider test paths, production incident evidence | Keep deterministic local proof and exact-provider conformance as separate layers; never supply externally usable defaults. | Test design/deployment design |

## Current repository authority

### No existing object-storage pack

The module targets Go 1.26.6 and currently depends on Testcontainers core and
PostgreSQL, but contains no AWS, MinIO, Go Cloud, Thanos, or other object-storage
runtime dependency ([go.mod](../../../go.mod)). `OBJECT_STORAGE` is absent from
the initializer and profile lock. The default outbound profile is also not an
object client ([README](../../../README.md)).

### Outbound trust and transport

The bounded HTTP client owns one fixed, credential-free authority, DNS/dial
target validation, redirects, proxy behavior, response limits, correlation,
connection pooling, and optional repeatable-request retries
([config.go](../../../internal/infra/httpclient/config.go),
[client.go](../../../internal/infra/httpclient/client.go),
[target_policy.go](../../../internal/infra/httpclient/target_policy.go),
[retry.go](../../../internal/infra/httpclient/retry.go)). Provider authentication,
credential acquisition, concrete trust, operation budgets, provider error
mapping, and provider readiness remain adapter concerns
([Repository Architecture](../../../docs/repo-architecture.md)).

That client cannot be reused wholesale merely because an SDK accepts an
`HTTPClient`:

- virtual-host S3 addressing derives a new authority by adding the bucket to the
  endpoint host, while the current transport rejects authority changes;
- its response-body cap is not by itself a streaming download policy;
- SDK retries plus transport retries would multiply attempts;
- the MinIO client accepts a `RoundTripper`, while the exported repository client
  is a Doer;
- AWS access points, directory buckets, acceleration, and region redirects can
  introduce authorities outside a single configured endpoint.

The reusable evidence is therefore the current trust policy and transport
components, not an automatic decision to wrap either client with the existing
high-level HTTP client. A later design must prove a fixed-authority composition
or use a provider-owned transport under the repository's platform-egress rule.

### Configuration and secrets

Configuration has typed defaults and validation; unknown YAML and `APP__...`
keys fail. Secret-like non-empty YAML values are rejected and secrets belong to
environment/file sources
([configuration source policy](../../../docs/configuration-source-policy.md),
[secret_policy.go](../../../internal/config/secret_policy.go),
[snapshot_contract_test.go](../../../internal/config/snapshot_contract_test.go)).

This makes these SDK behaviors policy decisions rather than harmless defaults:

- ambient AWS environment/shared-file/ECS/EC2 credential discovery;
- ambient region and checksum environment variables;
- a custom endpoint that receives signed identity material;
- static access key, secret key, and session token values;
- provider-specific path-style, signing-region, or TLS exceptions.

The smallest safe configuration is not yet a schema. Specification must decide
which credential modes and endpoint forms the pack promises before defaults can
be minimal and validated.

### Bootstrap, readiness, budgets, and telemetry

The closest optional runtime precedent is messaging: conditional construction,
cached readiness, supervised terminal failure, drain ordering, bounded close,
and adapter-owned telemetry
([startup_messaging.go](../../../cmd/service/internal/bootstrap/startup_messaging.go),
[natsjs client](../../../internal/infra/natsjs/client.go),
[natsjs telemetry](../../../internal/infra/natsjs/telemetry.go)). Readiness probes
run sequentially and the aggregate timeout must cover all enabled probes
([health service](../../../internal/health/service.go),
[configuration source policy](../../../docs/configuration-source-policy.md)).

No current authority says that object storage is startup- or readiness-critical.
`HeadBucket` is not a neutral answer: on AWS its generic 400/403/404 response can
mix wrong region, missing bucket, authentication, authorization, and policy; a
feature may also tolerate transient storage failure. Specification must name the
required availability semantics before a probe is chosen.

Current request, memory, connection, readiness, and shutdown budgets remain
upper bounds around any adapter. SDK timeouts, transfer worker counts, buffers,
and retries must fit inside them rather than create a second independent budget.
Dependency telemetry belongs at the adapter and must avoid object key, bucket,
URL, user metadata, or other unbounded/sensitive dimensions.

### Template profile and generated-output implications

A future `OBJECT_STORAGE=none|s3` selector would affect all existing generator
contracts, not only runtime files:

1. initializer usage, validation, idempotent repeat checks, `template.lock`, and
   completion output ([init-module.sh](../../../scripts/init-module.sh));
2. profile markers and explicit removal for pack-only files;
3. independent generator oracles for selected/absent files, marker removal,
   idempotence, incompatible repeats, and dependency retention
   ([template-init-check.sh](../../../scripts/ci/template-init-check.sh));
4. a CI profile lane ([ci.yml](../../../.github/workflows/ci.yml));
5. `go.mod`/`go.sum`, with the final generated-service `go mod tidy` removing the
   SDK when the pack is disabled;
6. `OUTBOUND_HTTP` retention if the chosen object adapter reuses that optional
   package while other selectors do not;
7. template-owned purity: profile-marked files cannot be mirrored instruction
   content ([template-owned.paths](../../../template-owned.paths)).

No public REST/OpenAPI surface or extra binary follows from the intended pack.
Those implications arise only if Specification later adds a public object API or
an independent transfer worker.

## External contract synthesis

### Compatibility boundary

Amazon S3 publishes strong read-after-write consistency for successful object
writes and deletes, including subsequent reads, metadata, and lists
([AWS S3 consistency](https://docs.aws.amazon.com/AmazonS3/latest/userguide/Welcome.html)).
That is an AWS guarantee. Cloudflare R2 separately publishes its own strong
consistency contract and explicitly narrows cached custom-domain behavior
([R2 consistency](https://developers.cloudflare.com/r2/reference/consistency/)).
Neither statement establishes a generic S3 guarantee.

Current providers expose material differences:

- [R2's matrix](https://developers.cloudflare.com/r2/api/s3/api/) marks specific
  operations and fields unsupported or partial, including AWS KMS, ACL, Object
  Lock, tagging, expected owner, and checksum variants; its region is `auto`.
- [DigitalOcean Spaces](https://docs.digitalocean.com/products/spaces/reference/s3-compatibility/)
  calls compatibility partial and documents different encryption, multipart,
  lifecycle, tagging, endpoint, and region behavior.
- [Backblaze B2](https://www.backblaze.com/docs/cloud-storage-s3-compatible-api)
  omits or changes IAM, ACL, tagging, and browser POST features.
- [GCS XML interoperability](https://cloud.google.com/storage/docs/interoperability)
  and [OCI's S3-compatible API](https://docs.oracle.com/en-us/iaas/Content/Object/Tasks/s3compatibleapi.htm)
  document their own subsets rather than S3 parity.
- [Railway Buckets](https://docs.railway.com/storage-buckets) provide a private
  S3-compatible deployment facility and injected endpoint/bucket/key/region
  values. Its broad compatibility wording has no field-level matrix, so it is
  deployment evidence, not proof of AWS checksum, condition, consistency,
  encryption, versioning, or error guarantees.

The reusable claim must therefore name operations, request/response fields, and
outcomes. Provider names alone are insufficient.

### Credentials and authority

`config.LoadDefaultConfig` resolves AWS credentials from environment static or
web-identity values, shared files, ECS task roles, then EC2 instance roles.
Explicit credentials replace that chain. Providers returned through the loader
are cached for concurrent use; directly assigned providers require
`aws.NewCredentialsCache`
([AWS SDK configuration](https://docs.aws.amazon.com/sdk-for-go/v2/developer-guide/configure-gosdk.html)).

This creates at least two distinct capability modes:

- AWS ambient/workload identity, including short-lived refreshable credentials;
- explicitly injected S3 access key, secret key, and optional session token,
  which is common for compatible providers.

Treating them as one silent fallback chain would undermine fail-fast typed
configuration and could select a developer's ambient AWS credentials. The
credential mode, endpoint, signing region, address style, and bucket jointly
form a trust boundary. SigV4 resolves an identity and endpoint before attaching
the access-key identifier, session token when present, and signature to the
request ([SDK authentication workflow](https://docs.aws.amazon.com/sdk-for-go/v2/developer-guide/configure-auth.html)).
By inference, an attacker-controlled endpoint can receive signed identity
material even though the secret key itself is never transmitted.

AWS recommends `BaseEndpoint` plus the S3 endpoint resolver rather than replacing
the resolver because S3 may incorporate region, bucket, virtual-host/path-style,
dual-stack, FIPS, access-point, and other rules
([endpoint resolution](https://docs.aws.amazon.com/sdk-for-go/v2/developer-guide/configure-endpoints.html)).
Those AWS rules are not a portable endpoint contract. Exact-provider evidence
must decide whether virtual-host or path-style addressing is supported and
whether a fixed-host repository trust policy can hold.

### Upload, multipart, streaming, cancellation, and cleanup

The low-level AWS S3 client's `PutObject` accepts `io.Reader`, but the default
signing path expects a seekable body to determine length and hash. Unknown-length
or non-seekable input requires explicit length/unsigned-payload treatment or an
upload manager
([unseekable streaming input](https://docs.aws.amazon.com/sdk-for-go/v2/developer-guide/sdk-utilities-s3.html#unseekable-streaming-input)).

The current AWS transfer-manager successor is
`feature/s3/transfermanager` v0.3.12. It is pre-v1 and replaces the deprecated
`feature/s3/manager` uploader/downloader
([transfer manager](https://pkg.go.dev/github.com/aws/aws-sdk-go-v2/feature/s3/transfermanager@v0.3.12),
[deprecated manager](https://pkg.go.dev/github.com/aws/aws-sdk-go-v2/feature/s3/manager)).
Its current source defaults are 8 MiB parts, a 16 MiB multipart threshold,
concurrency 5, at most 10,000 upload parts, a 50 MiB GetObject buffer, and three
part-body retries
([source defaults](https://github.com/aws/aws-sdk-go-v2/blob/feature/s3/transfermanager/v0.3.12/feature/s3/transfermanager/api_client.go#L5-L22)).
Concurrency is a per-call worker pool; simultaneous calls multiply workers and
buffers. Exact peak memory across seekable, streaming, checksum, and concurrent
paths is not a published invariant.

Consequences for a bounded pack:

- admit work before calling the SDK; SDK per-call concurrency is not a global
  process bound;
- state a maximum accepted object size independent of provider maxima;
- treat seekability, content length, buffering, and multipart threshold as
  observable execution paths;
- use caller context as the outer operation budget; cancellation stops further
  retries but does not prove remote non-commit;
- preserve multipart upload IDs on failure and distinguish abort attempt,
  uncertain in-flight parts, reconciliation, and deployment lifecycle cleanup;
- do not claim abort completion merely because `AbortMultipartUpload` returned:
  AWS says in-flight parts can still succeed and `ListParts` may need rechecking
  ([AbortMultipartUpload](https://docs.aws.amazon.com/AmazonS3/latest/API/API_AbortMultipartUpload.html));
- always close response bodies and regard deferred read/checksum errors as
  operation failures.

AWS charges incomplete multipart parts until completion or abort and recommends
`AbortIncompleteMultipartUpload` lifecycle cleanup
([multipart overview](https://docs.aws.amazon.com/AmazonS3/latest/userguide/mpuoverview.html),
[lifecycle cleanup](https://docs.aws.amazon.com/AmazonS3/latest/userguide/mpu-abort-incomplete-mpu-lifecycle-config.html)).
That lifecycle is deployment-owned and provider-specific; it backs up, but does
not replace, immediate adapter cleanup.

### Download and metadata

`GetObject` returns an `io.ReadCloser`. The caller must close it, and connection
or integrity errors may arrive while reading after the SDK call returned
([response ownership](https://docs.aws.amazon.com/sdk-for-go/v2/developer-guide/using.html#responses-with-ioreadcloser),
[GetObject FAQ](https://docs.aws.amazon.com/sdk-for-go/v2/developer-guide/faq-gosdk.html)).
A trustworthy download contract therefore cannot equate “headers received” with
“download succeeded.” A declared byte limit must constrain the stream even when
`Content-Length` is absent or false, and checksum success requires consuming the
body to EOF.

AWS `HeadObject` can return 404 for a missing object only when the caller also
has `s3:ListBucket`; otherwise the same missing object yields 403
([HeadObject](https://docs.aws.amazon.com/AmazonS3/latest/API/API_HeadObject.html)).
This blocks a universal 403-to-not-found mapping. AWS user metadata is carried
in bounded HTTP headers, and the Go SDK normalizes returned metadata keys to
lowercase ([AWS object metadata](https://docs.aws.amazon.com/AmazonS3/latest/userguide/UsingMetadata.html)).
Specification must define the small metadata subset feature code may rely on;
arbitrary user metadata must not become telemetry dimensions.

### Checksums and object identity

Since S3 module v1.74.1, the AWS Go v2 SDK automatically calculates CRC32 for
uploads when no algorithm is selected. Transfer-manager uploads also default to
CRC32. `GetObject` validates only with `ChecksumMode=Enabled`, only when stored
checksum metadata exists, and only as the body is fully read
([Go SDK S3 checksums](https://docs.aws.amazon.com/sdk-for-go/v2/developer-guide/s3-checksums.html)).

The following must remain separate:

- in-transit request/response checksum verification;
- a stored full-object digest the feature can use;
- a multipart composite checksum;
- ETag, which AWS documents as opaque and which changes with multipart and
  encryption;
- a provider version or generation token.

R2's current matrix supports different algorithm/type combinations, so the AWS
SDK's automatic CRC32 path cannot be projected onto all compatible providers.
Checksum algorithm, multipart path, returned fields, and EOF verification need
per-provider proof. Ambient `AWS_REQUEST_CHECKSUM_CALCULATION` and
`AWS_RESPONSE_CHECKSUM_VALIDATION` settings must not silently change a generated
service's integrity policy
([shared SDK checksum settings](https://docs.aws.amazon.com/sdkref/latest/guide/feature-dataintegrity.html)).

### Retries, idempotency, and overwrite protection

The AWS Standard retryer defaults to three attempts, up to 20 seconds backoff,
and a client-side rate-limit token bucket. Cancellation prevents further retry
([retry behavior](https://docs.aws.amazon.com/sdk-for-go/v2/developer-guide/configure-retries-timeouts.html)).
Only one layer may own retries, and its complete attempt/backoff budget must fit
inside the caller context.

Operation names alone do not prove semantic idempotency:

- GET and HEAD can be replayed, but deferred body reads are outside the SDK's
  operation retry loop;
- unconditional PutObject can overwrite a concurrent writer even when replaying
  identical bytes;
- AWS `If-None-Match: *` protects create-only writes with 412 and can return 409
  on a concurrent conflict, but support and exact outcomes require provider
  evidence ([PutObject](https://docs.aws.amazon.com/AmazonS3/latest/API/API_PutObject.html));
- multipart completion has upload-session state; AWS documents conflict cases
  that require restarting the multipart upload rather than replaying completion
  ([conditional writes](https://docs.aws.amazon.com/AmazonS3/latest/userguide/conditional-writes.html));
- delete may be repeatable at the HTTP level while its meaning changes under
  versioning or delete markers.

Feature policy must choose create-only, replace, compare-and-swap, or delete
intent. A reusable pack may expose only the precondition mechanism its supported
providers prove; it must not choose whether overwriting is allowed.

### Encryption

TLS is a transport requirement for external/public targets. A private HTTP
target remains possible only when Specification accepts the repository's
`PrivateHTTP` trust class and deployment enforces its private network boundary.
At-rest encryption is a deployment/provider capability, not an S3-compatible
default:

- AWS S3 encrypts new uploads with SSE-S3 by default
  ([AWS default encryption](https://docs.aws.amazon.com/AmazonS3/latest/userguide/default-encryption-faq.html));
- AWS SSE-KMS adds KMS permissions and request behavior
  ([SSE-KMS](https://docs.aws.amazon.com/AmazonS3/latest/userguide/UsingKMSEncryption.html));
- R2 provides provider-managed encryption but rejects AWS SSE/SSE-KMS fields;
- DigitalOcean documents a different, limited encryption subset.

SSE-C and client-side encryption are separate security designs with key
lifecycle, retry, presign, and browser implications. They must not be pulled
into a minimal shared pack by implication. Deployment should enforce the chosen
at-rest policy through bucket/provider controls; an adapter sends encryption
headers only when the accepted provider capability requires them.

### Presigned access

AWS explicitly treats presigned URLs as bearer tokens. They are reusable until
expiry, may expire earlier with temporary credentials or revocation, and are
checked when a request starts; an established transfer can continue past expiry
([AWS presigned URLs](https://docs.aws.amazon.com/AmazonS3/latest/userguide/using-presigned-url.html)).
The Go v2 SDK's current default presign expiry is 15 minutes, but a template must
not inherit that as business policy
([PresignOptions](https://pkg.go.dev/github.com/aws/aws-sdk-go-v2/service/s3@v1.107.1#PresignOptions)).

The safe reusable mechanics are candidates, not yet a contract:

- sign only an explicitly requested operation and feature-owned key;
- require the feature to authorize immediately before issuance;
- cap expiry and return the exact method and signed headers with the URL;
- never log, metric-label, trace-attribute, or persist the query-bearing URL;
- bind content type, checksum, encryption, and overwrite preconditions when the
  provider supports and the feature requires them;
- state that issuance telemetry does not observe the later client transfer;
- treat revocation and one-time-use as unsupported unless an external control
  proves them.

AWS SigV4 POST policy can enforce predicates including `content-length-range`
([POST policy](https://docs.aws.amazon.com/AmazonS3/latest/API/sigv4-HTTPPOSTConstructPolicy.html)),
but R2 supports presigned GET/PUT/HEAD/DELETE and not POST
([R2 presigning](https://developers.cloudflare.com/r2/api/s3/presigned-urls/)).
Therefore a portable presigned PUT cannot promise service-side body limits. If
hard upload-size enforcement is required, Specification must narrow providers,
select a supported policy mechanism, or keep the upload on the service path.

### Delete semantics

AWS DeleteObject permanently removes an unversioned object, but an ordinary
delete in a versioned bucket can create a delete marker while older versions
remain ([DeleteObject](https://docs.aws.amazon.com/AmazonS3/latest/API/API_DeleteObject.html)).
R2 currently lacks bucket versioning while other compatible providers differ.
Thus a stable delete result can mean only that the requested provider operation
completed; it cannot universally mean all bytes, versions, legal holds, or
replicas are gone. Retention, Object Lock, legal hold, version cleanup, and
proof of erasure remain feature/deployment policy outside a minimal pack.

### Error mapping

AWS SDK errors are wrapped in `smithy.OperationError`; modeled errors support
`errors.As`; service errors expose `smithy.APIError`; S3 response errors may
include AWS request and host IDs
([Go SDK error handling](https://docs.aws.amazon.com/sdk-for-go/v2/developer-guide/handle-errors.html)).
Compatible providers may return different codes, bodies, or headers.

A downstream stable taxonomy should be deliberately small and preserve the
underlying operation, HTTP status, provider code, request ID when safe, retry
classification, and deferred stream error for diagnostics. Candidate categories
are invalid input/configuration, not found when unambiguous, precondition failed,
access denied without inventing authentication detail, throttled/temporary,
integrity failure, cancelled/deadline, and internal/unknown. Specification must
define exact caller-visible meanings; this Research does not finalize them.

### Observability and readiness

AWS SDK tracing, metrics, and logs are no-op by default. Smithy adapters can
emit operation spans plus call/attempt/error, endpoint, signing, credential,
DNS/TLS/connection, and time-to-first-byte measurements
([SDK observability](https://docs.aws.amazon.com/sdk-for-go/v2/developer-guide/configure-observability.html)).
Debug logging can include signing and HTTP request/response details; no current
authoritative safe-redaction guarantee was found for presigned query values,
encryption headers, metadata, or bodies
([SDK logging](https://docs.aws.amazon.com/sdk-for-go/v2/developer-guide/configure-logging.html)).

Reusable low-cardinality signals may include operation, result class, provider
family configured at startup, duration, attempt count, bytes, active/admitted
transfers, throttling, integrity failures, multipart abort outcomes, and
readiness outcome. Keys, bucket names, URLs, query strings, user metadata,
provider messages, and credential identifiers must stay out of dimensions.
Request IDs may be sanitized log fields, never metric labels.

Presigned generation and presigned execution are different paths. The SDK can
observe generation; the later caller-to-provider transfer needs provider access
logs/metrics or feature-level completion proof. Readiness should test only the
capability the service requires and should not create, list, or expose objects.
Its criticality, cache interval, and timeout remain downstream decisions.

## Candidate families

| Candidate family | Distinct relationship to the outcome | Current strengths | Owned cost / risk | Flip conditions |
| --- | --- | --- | --- | --- |
| AWS SDK for Go v2 S3 client, optionally with the new transfer manager | Official AWS S3 implementation used against AWS or explicitly tested compatible endpoints | Active Apache-2.0 implementation; SigV4, refreshable credential providers, endpoint resolver, modeled/Smithy errors, presigning, checksums, multipart, OTel adapters | New Smithy/AWS module graph; ambient defaults; AWS-specific features; endpoint-host interaction with fixed trust; pre-v1 transfer manager; explicit process bounds still required | Exact providers and address style pass conformance; accepted credential modes map cleanly; pinned transfer behavior meets memory/cancellation/cleanup budgets |
| Purpose-built S3-compatible SDK, represented by `minio-go/v7` | Client centered on common S3-compatible endpoints rather than AWS service families | Active Apache-2.0 client; endpoint, path/virtual lookup, credential providers, streaming/multipart, presigning, custom transport | Its compatibility claim is not a provider guarantee; distinct retries/errors/checksum behavior; `RoundTripper` integration; MinIO server is separately archived | Accepted surface matches client semantics on every target; maintenance and proof remain acceptable |
| Cross-provider object abstraction, such as [Go Cloud `blob`](https://gocloud.dev/howto/blob/) or a [Thanos-style client](https://github.com/thanos-io/objstore) | Common object API above multiple provider drivers | Can centralize a genuinely provider-neutral feature port; maintained source and production precedent | Common denominator may omit S3 preconditions, checksums, multipart control, presign details, and exact errors; still pulls provider drivers. Go Cloud's URL opener must explicitly select AWS SDK v2 because its default remains v1 | A real multi-cloud requirement exists and the accepted operations fit the abstraction without provider escapes |
| Bounded direct HTTP, with hand-written SigV4 or only the AWS signer | Repository owns the wire contract directly | Small dependency surface for a truly tiny fixed-provider operation set; direct transport control | Must own signing canonicalization, refresh, XML, addressing, redirects, clock behavior, checksums, multipart state/abort, presign, retries, and compatibility. AWS itself recommends SDKs | Reopens only if the contract drops multipart and credential-chain breadth, targets one fixed authority, and the remaining wire code is demonstrably smaller and safer |
| Provider-specific managed/native facility | Deployment/runtime binding supplies bucket and sometimes identity or native object calls | Can remove static credentials and provisioning burden on a fixed platform; Railway Buckets directly match the repository's current deployment profile | Ties generated runtime semantics to one platform; native APIs are not a reusable S3 contract; provider compatibility remains partial | Product explicitly fixes a provider/runtime and accepts its native contract; otherwise use the facility only as deployment backing for the portable adapter |
| No shared pack | Each adopter owns feature port, provider adapter, config, and proof | Zero template dependency or implied portability contract; current state | Does not meet the stated template-ready outcome and repeats security/reliability mechanics across adopters | Remains valid if evidence shows only one specialized adopter, incompatible feature policies, or no reusable stable subset |

Current module snapshots, not version selections:

- `github.com/aws/aws-sdk-go-v2/service/s3` v1.107.1, Go 1.24 minimum,
  published 2026-08-10;
- `github.com/aws/aws-sdk-go-v2/feature/s3/transfermanager` v0.3.12,
  Go 1.24 minimum, published 2026-08-10;
- `github.com/aws/smithy-go` v1.27.7;
- `github.com/minio/minio-go/v7` v7.2.1, Go 1.25 minimum,
  published 2026-06-26;
- this repository uses Go 1.26.6, so minimum Go versions do not currently block
  either client family.

AWS SDK for Go v1 is not a current candidate because support ended on
2025-07-31
([AWS v1 end of support](https://aws.amazon.com/blogs/developer/announcing-end-of-support-for-aws-sdk-for-go-v1-on-july-31-2025/)).

### Candidate saturation

The search covered official direct clients, S3-compatible direct clients,
cross-provider object abstractions, repository-owned protocol, provider-native
facilities/gateways, the null/no-pack option, and test-only substitutes. Each is
a different relationship to the same live decision: who owns the object API and
S3 wire semantics. Additional storage vendors instantiate the provider-specific
family; additional emulators instantiate a proof layer. No new family emerged
from searches across credentials, addressing, multipart, checksums, presigning,
consistency, encryption, or errors, so family-level saturation is reached.

## Candidate ownership boundary

| Stable reusable behavior candidate | Deployment-owned control | Feature-owned policy |
| --- | --- | --- |
| Construct one validated S3 authority from endpoint, region, bucket, address style, and explicit credential mode | Provision bucket/account/region and private or public network path | Generate and validate object keys and namespaces |
| Acquire and refresh credentials through an accepted mode; never expose them to feature code | Least-privilege identity, key injection/rotation, workload identity, bucket policy | Authorize every upload, download, metadata read, delete, and presign issuance |
| Stream bounded requests/responses; own body close, cancellation, and operation deadlines | Provider quotas, capacity, egress, service SLOs | Content type, allowed content, feature-level size limits, malware/media validation |
| Bound process admission, multipart workers/buffers, SDK retries, and cleanup attempts | Lifecycle rule for incomplete multipart uploads and abandoned objects | Retention intent, deletion schedule, business reconciliation |
| Verify a supported checksum path and surface integrity failure | Provider checksum/encryption capabilities | Decide whether a domain digest is stored and what it means |
| Expose supported conditional-write/precondition mechanics | Versioning, Object Lock, legal hold, replication policy | Choose create-only, replace, expected-version, or conflict behavior |
| Generate presigned requests with capped mechanics and safe returned headers | Provider/IAM/bucket policy restrictions, access logging, CDN/custom-domain controls | Decide recipient, operation, key, business TTL, content constraints, and revocation workflow |
| Map provider failures conservatively and instrument bounded outcomes | Provider dashboards, audit logs, alerts, incident response | Translate stable pack outcomes into feature/API behavior |
| Close idle transports and expose a narrowly justified readiness probe | Deployment health, credentials, endpoint/DNS/TLS | Decide whether a feature can degrade when storage is unavailable |

The pack must not create buckets, invent key formats, authorize callers, choose
retention, inspect content, promise erasure, or select a provider's KMS/lifecycle
policy. Those responsibilities have different authorities and change rates.

## Local development and proof ladder

| Layer | Useful proof | Explicitly not proved |
| --- | --- | --- |
| Feature fake/in-memory port | Feature authorization, key ownership, retention decisions, error handling, and no storage dependency in business tests | S3 signing, streaming, provider errors, checksums, cleanup, compatibility |
| Scripted `httptest` or custom transport | Adapter mapping for statuses/XML, cancellation, deadlines, response over-limit, body close, attempt budget, malformed responses, and telemetry redaction | Real SigV4 acceptance, DNS/TLS authority, provider behavior, multipart races |
| Local S3 emulator/gateway | Client wiring and a selected operation subset without cloud cost | Target-provider credentials, endpoint rules, consistency, policy, encryption, presign validation unless explicitly implemented |
| Exact target-provider integration | Credentials, addressing, signed headers, operation/field matrix, checksum paths, conditional writes, multipart abort/reconciliation, and presign execution | Production network policy, quotas, load, KMS/bucket policy, and operational alerts unless the same deployed path is exercised |
| Deployed-path canary | Runtime identity, DNS/TLS/egress, bucket policy, readiness, telemetry, limits, and cleanup in the target environment | Guarantees for another provider or a different bucket policy |

Local implementation candidates have important limits:

- Adobe S3Mock is active and Apache-2.0, but is path-style only and explicitly
  does not validate presigned signature, expiry, or HTTP verb
  ([S3Mock limitations](https://github.com/adobe/S3Mock#important-limitations)).
  The repository's Testcontainers v0.43.0 does not yet include the documented
  S3Mock module; a generic container would be required.
- The MinIO Go client remains active, but the separate `minio/minio` server
  repository is archived and source-only. A pinned historical image may exercise
  a subset; it is weak evidence for a new default local contract
  ([MinIO server](https://github.com/minio/minio)).
- LocalStack is broader and heavier than the requested operation subset; its S3
  behavior is emulator evidence, not AWS or compatible-provider proof
  ([LocalStack S3](https://docs.localstack.cloud/aws/services/s3/)).
- SeaweedFS offers a maintained S3 gateway, but proves that gateway rather than
  the chosen production provider ([SeaweedFS](https://github.com/seaweedfs/seaweedfs)).

Local convenience must not weaken production semantics:

- local HTTP, if supported, is an explicit private/local target class;
  Specification must choose production trust, with external HTTPS as the safe
  default and private production HTTP reopened only for an accepted private
  target and deployment-enforced network boundary;
- endpoint, bucket, and credentials have no externally usable defaults;
- local address style is explicit rather than silently switching production;
- every integration run owns a unique prefix or disposable bucket, closes
  bodies, aborts multipart uploads, removes objects, and registers container
  cleanup immediately;
- emulator-green claims remain labeled as adapter/subset proof.

Grafana's published Loki incident—local MinIO defaults led to attempted writes to
real AWS buckets—shows why resolvable default bucket/endpoint combinations are
unsafe ([incident report](https://grafana.com/blog/grafana-security-update-grafana-loki-and-unintended-data-write-attempts-to-amazon-s3-buckets/)).
It is incident evidence for fail-closed defaults, not a universal S3 contract.

## Risks, conflicts, and refresh triggers

| Risk or conflict | Consequence | Current disposition / refresh trigger |
| --- | --- | --- |
| Marketing-level “fully S3 compatible” vs field-level matrices | Accidental projection of AWS guarantees | Require an operation/field/provider matrix; refresh when a provider compatibility page changes |
| AWS ambient credentials vs repository typed configuration | Wrong account or local credentials may be selected | Credential mode is an explicit downstream decision; no silent chain equivalence |
| SDK endpoint rewriting vs fixed-authority trust | Signed identity material may reach an unapproved host or requests may fail closed | Prove the exact resolved authority set for endpoint/bucket/address style |
| SDK retry plus repository retry | Attempt amplification and budget exhaustion | One retry owner; operation- and precondition-aware policy |
| New transfer manager pre-v1 and per-call concurrency | Unstable API and unbounded aggregate memory/workers | Pin source and prove peak/admission/cancel/cleanup before process-wide claims; refresh at v1 or default changes |
| Automatic CRC32 vs provider checksum matrix | Upload rejection or false integrity claim | Pin/test exact algorithm and single/multipart/download paths per provider |
| Presigned direct transfer | Service body/concurrency/telemetry controls do not execute | Bound issuance only; use provider policies/logs or service-mediated transfer for stronger controls |
| HEAD 403/404 ambiguity | Authorization may be mistaken for absence | Conservative error mapping; provider-specific evidence before not-found |
| Versioned delete and retention controls | “Deleted” may leave prior bytes | Keep retention/erasure outside generic delete result |
| Multipart cancel/abort race | Orphaned billed parts | Immediate cleanup plus reconciliation and deployment lifecycle backstop |
| Debug logging and URL query credentials | Secret leakage | Wire/body/signing debug off; explicit redaction tests before any enablement |
| Local emulator drift | False compatibility confidence | Maintain layer-scoped claims and exact-provider integration |

Refresh the research when the pinned client, transfer manager, checksum, retry,
endpoint, or presign defaults change; a target provider or access path is added;
or the deployment changes versioning, Object Lock, lifecycle, encryption/KMS,
CDN/cache, identity, or bucket policy.

## Downstream decision inputs

Specification must resolve behavior before design selects packages or knobs:

1. Define what “S3-compatible” promises: one provider, a named provider set, or
   a core operation/field matrix with explicit exclusions.
2. Define the feature-facing outcome for upload, download, metadata, delete, and
   presigned access, including stream ownership, success point, maximum bytes,
   cancellation, and partial work.
3. Choose supported credential modes and their fail-fast behavior: static
   injected keys, AWS workload identity/default chain, or both as explicit modes.
4. Define endpoint, region/signing-region, bucket, address-style, TLS, and
   resolved-authority invariants. Decide whether AWS-specific access points,
   acceleration, directory buckets, and region redirects are outside the pack.
5. Define overwrite semantics and which provider-tested preconditions are stable.
6. Define checksum meaning for upload and fully consumed download; state whether
   ETag/version values are opaque.
7. Define when multipart begins, which outer object/concurrency/memory bounds
   hold, what cancellation means, and how cleanup uncertainty is surfaced.
8. Define retryable operations and provider outcomes under one total deadline and
   one attempt owner.
9. Define encryption claims without projecting AWS SSE/KMS defaults; keep bucket
   policy and key lifecycle with deployment unless the accepted behavior says
   otherwise.
10. Define presign operation set, expiry ceiling, returned signed headers,
    overwrite/content constraints, URL secrecy, reuse/revocation statement, and
    how direct-transfer completion is observed.
11. Define a conservative stable error vocabulary, including ambiguous HEAD and
    deferred body/checksum failures.
12. Decide whether storage participates in startup/readiness and what exact
    non-mutating capability the probe proves.
13. Define bounded telemetry and redaction, separating SDK operation,
    presign generation, and direct presigned transfer.
14. Define the proof matrix: deterministic feature tests, adapter fault tests,
    optional emulator subset, exact-provider conformance, and deployed-path
    obligations.
15. Define the profile postcondition for `OBJECT_STORAGE=none|s3`, including
    dependency pruning and interaction with `OUTBOUND_HTTP`.

The leading hypothesis is falsified if the stable cross-provider subset cannot
cover bounded upload/download/metadata/delete/presign without provider escapes;
if exact-provider tests show incompatible checksum, condition, or error behavior;
if a process-wide transfer bound cannot be proven with the chosen client; or if
only one specialized adopter owns the capability and no template-level reuse
remains.

## Evidence register

### Authoritative specifications, provider contracts, and source

| Source | Authority and use | Limit |
| --- | --- | --- |
| [AWS SDK Go v2 configuration](https://docs.aws.amazon.com/sdk-for-go/v2/developer-guide/configure-gosdk.html), [endpoints](https://docs.aws.amazon.com/sdk-for-go/v2/developer-guide/configure-endpoints.html), [authentication](https://docs.aws.amazon.com/sdk-for-go/v2/developer-guide/configure-auth.html) | Current SDK credential, endpoint, and signing behavior | AWS SDK behavior, not compatible-provider guarantee |
| [AWS retries/timeouts](https://docs.aws.amazon.com/sdk-for-go/v2/developer-guide/configure-retries-timeouts.html), [HTTP client](https://docs.aws.amazon.com/sdk-for-go/v2/developer-guide/configure-http.html) | Retry defaults, context cancellation, transport defaults | Requires adapter-level total budgets |
| [AWS transfer manager v0.3.12](https://pkg.go.dev/github.com/aws/aws-sdk-go-v2/feature/s3/transfermanager@v0.3.12), [source defaults](https://github.com/aws/aws-sdk-go-v2/blob/feature/s3/transfermanager/v0.3.12/feature/s3/transfermanager/api_client.go#L5-L22) | Current transfer API/status/defaults | Pre-v1; exact peak memory and abort outcome are not published invariants |
| [AWS PutObject](https://docs.aws.amazon.com/AmazonS3/latest/API/API_PutObject.html), [multipart overview](https://docs.aws.amazon.com/AmazonS3/latest/userguide/mpuoverview.html), [AbortMultipartUpload](https://docs.aws.amazon.com/AmazonS3/latest/API/API_AbortMultipartUpload.html) | Write, condition, multipart, and cleanup semantics | AWS only |
| [AWS checksum guide](https://docs.aws.amazon.com/sdk-for-go/v2/developer-guide/s3-checksums.html), [checksum API](https://docs.aws.amazon.com/AmazonS3/latest/API/API_Checksum.html) | Go SDK automatic checksums and AWS stored checksum types | Provider support differs |
| [AWS HeadObject](https://docs.aws.amazon.com/AmazonS3/latest/API/API_HeadObject.html), [DeleteObject](https://docs.aws.amazon.com/AmazonS3/latest/API/API_DeleteObject.html), [SDK errors](https://docs.aws.amazon.com/sdk-for-go/v2/developer-guide/handle-errors.html) | Metadata/delete/error ambiguity | AWS/Smithy details are not portable taxonomy |
| [AWS default encryption](https://docs.aws.amazon.com/AmazonS3/latest/userguide/default-encryption-faq.html), [SSE-KMS](https://docs.aws.amazon.com/AmazonS3/latest/userguide/UsingKMSEncryption.html) | AWS at-rest and KMS behavior | AWS only |
| [AWS presigned URL guide](https://docs.aws.amazon.com/AmazonS3/latest/userguide/using-presigned-url.html), [POST policy](https://docs.aws.amazon.com/AmazonS3/latest/API/sigv4-HTTPPOSTConstructPolicy.html) | Bearer/reuse/expiry and AWS upload-policy controls | Provider presign operations differ |
| [AWS SDK observability](https://docs.aws.amazon.com/sdk-for-go/v2/developer-guide/configure-observability.html), [logging](https://docs.aws.amazon.com/sdk-for-go/v2/developer-guide/configure-logging.html) | Smithy signals and debug modes | No established complete secret-redaction guarantee |
| [AWS SDK Go v2 repository](https://github.com/aws/aws-sdk-go-v2), [MinIO Go client](https://github.com/minio/minio-go) | Current maintenance, license, and source behavior | Client maintenance does not prove provider compatibility |
| [Go Cloud blob guide](https://gocloud.dev/howto/blob/), [s3blob API](https://pkg.go.dev/gocloud.dev/blob/s3blob), [Thanos objstore](https://github.com/thanos-io/objstore) | Cross-provider abstraction surfaces and current source | Common API does not prove fit for the requested S3-specific behavior |
| [R2 compatibility matrix](https://developers.cloudflare.com/r2/api/s3/api/), [presigning](https://developers.cloudflare.com/r2/api/s3/presigned-urls/), [consistency](https://developers.cloudflare.com/r2/reference/consistency/) | Current provider operation/field and access-path contracts | R2 only |
| [DigitalOcean Spaces matrix](https://docs.digitalocean.com/products/spaces/reference/s3-compatibility/), [Backblaze B2](https://www.backblaze.com/docs/cloud-storage-s3-compatible-api), [GCS](https://cloud.google.com/storage/docs/interoperability), [OCI](https://docs.oracle.com/en-us/iaas/Content/Object/Tasks/s3compatibleapi.htm) | Independent evidence of compatibility divergence | Each provider only; refresh before selection |
| [Railway Buckets](https://docs.railway.com/storage-buckets) | Current managed deployment option relevant to this template | Broad compatibility claim lacks field-level conformance matrix |

### Recent operational articles and conference evidence

| Source | Practical signal | Evidence limit |
| --- | --- | --- |
| [AWS Compute/Security guidance on presigned sharing, 2025](https://aws.amazon.com/blogs/security/how-to-securely-transfer-files-with-presigned-urls/) | Short expiry, IAM boundaries, audit, and bearer-URL operational trade-offs | First-party AWS practice, not a portable provider contract |
| [AWS re:Invent 2024: new default S3 data-integrity protections](https://reinvent.awsevents.com/content/dam/reinvent/2024/slides/stg/STG354-NEW_New-default-Amazon-S3-data-integrity-protections.pdf) | Production motivation for checksum defaults and end-to-end integrity | AWS talk and AWS behavior only |
| [SNIA SDC25 compatibility testing talk](https://www.snia.org/sites/default/files/2025-10/SNIA-SDC25-Terrace-Borich-SNIA-Community-Driven-S3-Compatibility-Testing.pdf), [Q&A](https://www.snia.org/blog/2025/s3-compatibility-testing-qa-expert-insights) | Independent field-by-field compatibility testing found divergent calls, headers, and ACL behavior | Sampled products and 2025 snapshot, not a universal conformance standard |
| [Grafana Loki unintended S3 write incident](https://grafana.com/blog/grafana-security-update-grafana-loki-and-unintended-data-write-attempts-to-amazon-s3-buckets/) | Externally usable local defaults can cause real data writes | One product incident; supports fail-closed defaults only |
| [Grafana Loki 3.4 storage-client consolidation](https://grafana.com/blog/grafana-loki-3-4-standardized-storage-config-sizing-guidance-and-promtail-merging-into-alloy/) | A large production system found value in a cross-provider object client | Observability-storage scope does not establish template fit |
| [USENIX FAST '25 Cloudscape study](https://www.usenix.org/conference/fast25/presentation/satija) | S3 appeared in 68% of roughly 400 sampled deployed AWS architectures | Prevalence evidence, not interoperability or SDK selection evidence |
| [ClickHouse production object-cache article, 2025](https://clickhouse.com/blog/building-a-distributed-cache-for-s3) | Object storage is high-latency and benefits from measured concurrency/prefetch rather than assumed tuning | Analytics workload and reported benchmarks do not set this service's budgets |

## Research stop rationale

Current repository authority is mapped, candidate families are saturated, major
provider conflicts are explicit, and unresolved items are behavior decisions
owned by Specification. More Research would add vendor instances or tuning data
without changing the live decision level. Research therefore stops here and does
not choose an SDK, define a config schema, or create design/tasks/code.

## Standalone prompt for Specification

```text
Continue the S3-Compatible Object Storage capability in the Specification macro phase for /Users/daniil/Projects/Opensource/go-service-template-rest.

Start from specs/s3-compatible-object-storage/research/synthesis.md. Treat it as Research evidence, not an approved design. Define a falsifiable behavioral contract for an optional template-init-selected pack, provisionally OBJECT_STORAGE=s3, that gives feature code bounded upload, download, metadata, delete, and presigned-access operations while leaving object keys, authorization, retention, and content policy with the feature and bucket/identity/network/encryption/lifecycle controls with deployment.

Resolve the research's downstream inputs: the exact S3-compatible provider/operation/field promise; credential modes; endpoint, region, bucket, address-style, TLS, and authority invariants; streaming/body/concurrency/deadline semantics; multipart threshold and cleanup outcomes; checksum meaning; retry/idempotency and overwrite preconditions; encryption claim; presigned operation, expiry, signed-header, size, reuse, secrecy, and observability semantics; conservative error vocabulary; readiness criticality; telemetry/redaction; proof layers; and OBJECT_STORAGE profile postconditions including dependency pruning and OUTBOUND_HTTP interaction.

Use a small feature-facing object port with one S3 adapter and exact-provider conformance as the shared-pack hypothesis only. Compare the still-live AWS SDK for Go v2 and purpose-built S3-compatible client families after behavior is fixed; do not select an SDK in Specification. Falsify the shared-pack hypothesis if the stable subset cannot cover the five requested operations without provider escapes, if no candidate can prove process-wide bounds and cleanup, or if template-level reuse is absent. Do not project AWS consistency, encryption, checksum, IAM, versioning, delete, or error guarantees onto compatible providers.

Produce only the Specification artifact and its required Specification review/repair evidence. Do not enter design, test design, planning, implementation, or deployment. Stop at the Specification macro-phase boundary with the standalone prompt for the next required phase.
```
