# Technical design — S3-compatible object storage

status: ready

Realizes: [../spec.md](../spec.md) at ready SHA-256
`46ee347a7931d9d20601ce00cea8455a89e2d9ba166dfa8e308b8a4154e16def`.
[Research synthesis](../research/synthesis.md), SHA-256
`d13fbc8ad407eaf89cf29653611d1efc0c5d30c489e97d5476e1328d909ed9a6`,
remains supporting evidence rather than contract authority.

Evidence was refreshed on 2026-08-14 against the pinned client and Go sources and
the provider sources linked below. Cloudflare's compatibility matrix is an
evolving statement of implemented fields, not an Amazon guarantee and not
exact-provider conformance.

## Outcome and retained hypothesis

Keep the shared-pack hypothesis. One provider-neutral object port and one S3
adapter can express upload, download, metadata, delete, and presigned GET for
the fixed Amazon S3 and Cloudflare R2 tuples without exposing a provider field.
The leading implementation is the low-level AWS SDK for Go v2 S3 client,
`service/s3 v1.107.1`; neither an SDK transfer manager nor a second provider
adapter is selected.

This is a conditional design decision, not a compatibility claim. Amazon and
R2 must each pass the complete exact-tuple conformance obligation. In
particular, current R2 documentation does not prove the SDK's one-pass
`aws-chunked` CRC64NVME trailer path, every returned checksum field, or a cleanup
postcondition strong enough to report `complete`. A failure there first reopens
this client decision. It reopens Specification if the repair needs a
provider-specific feature request/result or if no common five-operation subset
remains.

Feature code continues to own authorization, keys, overwrite and retention
intent, content policy, and any stricter size or lifetime limit. Deployment
continues to own the bucket, identity, endpoint/network path, encryption,
versioning state, and abandoned-multipart lifecycle rule. The adapter provisions
or changes none of them.

## Evidence-led client comparison

The maintained, materially distinct direct Go client families found were AWS
SDK v2, MinIO Go v7, SimpleS3, and kelindar/s3. SDK wrappers, MinIO forks,
servers, CLIs, emulators, and cross-cloud storage abstractions do not create a
different client mechanism.

| Candidate inspected on 2026-08-11 | Hard-contract fit | Required repository machinery | Disposition |
| --- | --- | --- | --- |
| AWS SDK Go v2 low-level `service/s3 v1.107.1`, core `v1.43.5`, credentials `v1.19.5`, Smithy `v1.27.7` | Direct explicit credentials, contexts, fixed base endpoint plus a final-authority HTTP guard, global `aws.NopRetryer` with per-operation `retry.Standard` only for HeadObject and pre-body GetObject, modeled conditions/checksums/multipart/ListParts/errors, and GET presign. The checksum middleware streams a known-length non-seekable body through an `aws-chunked` trailer instead of prebuffering it. | Adapter-owned admission/deadline, exact-length readers, whole-object multipart CRC, cleanup policy, conservative error mapping, response caps, and one transport attempt per SDK attempt. | **Selected.** It leaves the least protocol and checksum behavior for repository code while retaining all fixed decisions. |
| AWS Transfer Manager `v0.3.12` | Defaults include multipart workers, thresholds and buffers, body retries, ranged concurrent GET, and failure handling that can root work in a fresh background context. Abort has no ListParts proof. | Overriding or fighting mechanisms forbidden by R4, R5, and R9. | Rejected; it adds no required capability absent from the low-level client. |
| `minio-go/v7 v7.2.1` low-level `Core` | Viable static credentials, contexts, DNS-style bucket lookup, custom transport, serial raw multipart/ListParts/abort, structured errors, and presign. `MaxRetries=1` disables its operation retry. | Adapter-owned EOF checksum validation, lower-level multipart checksum assembly, active override of AWS dual-stack and path-style choices, and either a new `RoundTripper` seam or transport refactor. | Viable fallback, but not selected: it recreates more checksum behavior and fits the repository's HTTP `Doer` boundary less directly. |
| `rhnvrm/simples3 v0.11.1` | Custom endpoints are path-style; core calls lack caller contexts; signing and uploads buffer whole bodies; multipart retries cannot be disabled; checksum/error surfaces are insufficient. | Reimplementation of multiple hard requirements. | Rejected. |
| `kelindar/s3 v0.2.2` | Custom endpoints are path-style; reads lack contexts; transport replay is fixed; PUT accepts `[]byte`; multipart is parallel and lacks ListParts cleanup; checksum and bounded presign surfaces are absent. | Reimplementation of multiple hard requirements. | Rejected. |

The exact source decisions behind the selected family are:

- construct `s3.Options` directly; never call `config.LoadDefaultConfig` or
  consume the AWS shared environment/profile chain;
- supply `credentials.NewStaticCredentialsProvider` and `aws.NopRetryer`;
- set request checksum calculation and response checksum validation to
  `WhenRequired`, then request the exact algorithm on each governed operation;
- use only low-level `PutObject`, `GetObject`, `HeadObject`, `DeleteObject`,
  `CreateMultipartUpload`, `UploadPart`, `CompleteMultipartUpload`,
  `AbortMultipartUpload`, `ListParts`, and the low-level GET presigner;
- disable clock-skew correction, S3 Express session auth, multi-region access
  points, ARN region selection, accelerate, dual-stack, FIPS, path style, SDK
  wire/body logging, and any resolver-supplied alternate target;
- make the injected HTTP client, not `BaseEndpoint`, the final scheme and
  authority guard because the SDK resolver may modify its base endpoint.

Primary implementation evidence:

- [AWS SDK v2 S3 options](https://github.com/aws/aws-sdk-go-v2/blob/service/s3/v1.107.1/service/s3/options.go)
  and [`aws.NopRetryer`](https://github.com/aws/aws-sdk-go-v2/blob/v1.43.5/aws/retryer.go);
- [static credentials provider](https://github.com/aws/aws-sdk-go-v2/blob/credentials/v1.19.5/credentials/static_provider.go);
- [streaming checksum middleware](https://github.com/aws/aws-sdk-go-v2/blob/service/internal/checksum/v1.9.29/service/internal/checksum/middleware_compute_input_checksum.go)
  and [output validation](https://github.com/aws/aws-sdk-go-v2/blob/service/internal/checksum/v1.9.29/service/internal/checksum/middleware_validate_output.go);
- [AWS checksum contract](https://docs.aws.amazon.com/AmazonS3/latest/userguide/checking-object-integrity-upload.html),
  [abort contract](https://docs.aws.amazon.com/AmazonS3/latest/API/API_AbortMultipartUpload.html),
  and [ListParts contract](https://docs.aws.amazon.com/AmazonS3/latest/API/API_ListParts.html);
- [R2 S3 matrix and checksum types](https://developers.cloudflare.com/r2/api/s3/api/),
  [R2 presigned URLs](https://developers.cloudflare.com/r2/api/s3/presigned-urls/),
  and [R2 lifecycle rules](https://developers.cloudflare.com/r2/buckets/object-lifecycles/).

## Component and authority boundaries

```text
feature package
  owns principal authorization, key construction, content/retention/overwrite policy
        |
        v
internal/objectstorage
  owns the five-operation port, portable values, key grammar, stable errors
        |
        v
internal/infra/s3
  owns the one S3 policy, AWS SDK translation, admission, integrity, cleanup,
  deadlines, the bounded image-root snapshot, provider evidence mapping,
  and safe telemetry
        |
        +--> internal/infra/httpclient
        |      owns fixed public HTTPS authority, post-DNS address gate,
        |      caller-selected TLS-pool application, redirect/proxy refusal,
        |      body/header limits, and transport lifecycle
        |
        +--> AWS SDK Go v2 low-level S3 client
               owns SigV4 wire construction, modeled S3 XML, and streamed checksum I/O

cmd/service/internal/bootstrap
  owns selected-profile construction and dependency close registration

internal/config + env inputs
  own fail-closed deployment-supplied tuple and finite resource bounds

deployment
  owns bucket/identity/network/encryption/versioning/lifecycle controls and
  image-bundle provenance, placement, rotation, and no-override mounts
```

There is no public HTTP or gRPC resource, bucket repository, general filesystem,
blob abstraction, provider registry, factory, background worker, provisioner,
readiness probe, or runtime provider switch. There is one shared port because
the Specification requires reusable feature substitution and at least two
adopters; there is one concrete adapter because the selected wire subset is one
policy.

## Feature-facing port

`internal/objectstorage` imports only the standard library. Its single `Store`
interface is the complete shared surface:

```go
type Store interface {
	Upload(context.Context, string, io.ReadCloser, UploadOptions) (UploadResult, error)
	Download(context.Context, string) (Download, error)
	Metadata(context.Context, string) (Metadata, error)
	Delete(context.Context, string) error
	PresignGET(context.Context, string, time.Duration) (PresignedGET, error)
}
```

The package owns these provider-neutral values and nothing more:

- `UploadOptions { ContentLength int64; ContentType string; Intent UploadIntent }`,
  with closed intents `create_only` and `replace`;
- `UploadResult { Cleanup CleanupDisposition }`, where cleanup is `none` for a
  successful request and `pending` when failed multipart work may have created
  or retained an upload session;
- `Download { Body io.ReadCloser; Size int64; ContentType string; LastModified time.Time }`;
- `Metadata { Size int64; ContentType string; LastModified time.Time }`;
- `PresignedGET { Method string; URL string; Headers http.Header; SignatureExpiresAt time.Time }`;
- the exact R9 error kinds and a `Kind(error)` accessor that never exposes an
  SDK value or provider text.

An empty content type means absent. Last-modified is normalized to UTC but is
not reinterpreted. The package's key validator owns the fixed ASCII grammar;
every adapter entry point calls it before admission side effects, signing, or
provider I/O. The feature remains responsible for producing and authorizing the
string. No port value names bucket, provider, region, checksum, ETag, upload ID,
credential, HTTP status, or provider code.

Feature tests use a small fake implementing this interface. Features do not
import `internal/infra/s3` or the AWS SDK. The adapter's compile-time interface
assertion is the only reverse connection.

## System decisions

### D1 — One validated exact provider tuple constructs one adapter

The selected `s3` profile has no `enabled` field. It requires:

- provider: exactly `amazon_s3` or `cloudflare_r2`;
- HTTPS base endpoint without user info, path, query, fragment, or non-default
  port; region; DNS-compatible dotless bucket; explicit access-key identifier,
  secret, and optional session token; plus Amazon-only expected bucket owner;
- maximum object bytes, multipart chunk bytes, maximum active operations,
  maximum operation duration, maximum presign lifetime, maximum response-header
  bytes, maximum control-response bytes, and maximum adapter working-memory
  bytes.

The object-size validator is provider-specific: Amazon permits at most 5 TiB,
while R2 permits at most 5 TiB minus 5 GiB. The lower R2 limit is enforced at
construction rather than presented as an Amazon-compatible 5 TiB ceiling.

Production trust is not another deployment-configurable tuple field. The S3
constructor reads exactly `/etc/ssl/certs/ca-certificates.crt`, the public-WebPKI
bundle owned by the pinned runtime image, under D4's compile-time byte and
certificate-count ceilings. There is no production custom/private CA path,
system-root fallback, root augmentation, environment override, or hot reload.

For Amazon, startup accepts only a current commercial-partition regional
general-purpose endpoint whose SDK-resolved region and configured endpoint are
the same authority; the region cannot be `auto`, the expected owner is exactly
12 decimal digits, and the session token is non-empty. Every modeled mediated
input receives that owner; the pinned presigner serializes it into the signed
query rather than the returned opaque headers. For R2,
the endpoint host is exactly a 32-lowercase-hex account ID followed by
`.r2.cloudflarestorage.com`, `.eu.r2.cloudflarestorage.com`, or
`.fedramp.r2.cloudflarestorage.com`; region is `auto` and expected owner is
empty because R2 marks that field unsupported. The final request
authority is precomputed as `<bucket>.<endpoint-host>` and independently given
to the fixed-authority HTTP client. The common bucket grammar is 3–63 lowercase
alphanumeric or hyphen characters with alphanumeric ends and rejects Amazon's
reserved prefixes/suffixes, including account-regional `-an`. Dotted buckets are rejected because normal
public TLS wildcard validation would not cover that final host.

Construction validates the tuple, credential presence, key-independent field
bounds, multipart provider constraints, maximum part count, and memory equation,
then loads and strictly parses the bounded image bundle before creating the
client. It performs bounded local file I/O but no DNS lookup, credential-provider
I/O, or provider request. A successful constructor therefore means locally
valid capability and trust wiring, not bucket existence, identity authorization,
provider availability, or support certification.

The SDK is built directly from `s3.Options` with explicit snapshot credentials. No ambient
AWS region, endpoint, profile, web identity, instance/container metadata,
credential process, SSO, environment SDK flag, retry mode, logger, proxy, or
defaults mode is consulted. Session credentials remain a process snapshot and
create no refresh path. Amazon's required session token prevents accidental
long-lived IAM-user certification but still requires deployment-owned rolling
replacement before expiry; supporting one SDK-refreshed IAM-role source remains
a separate Specification decision because it adds another network authority.

### D2 — The existing fixed-target HTTP owner gains explicit one-attempt and trust inputs

`internal/infra/httpclient` remains the owner of public-address validation,
per-dial DNS result enforcement, exact request authority, applying caller-owned
TLS roots, redirect refusal, proxy refusal, response-header/body caps, and
idle-transport closing.
The S3 adapter constructs a private instance whose base URL is the final bucket
authority, propagation is `none`, retries are zero, and instrumentation is
disabled so the key-bearing URL never enters generic HTTP spans or metrics.

The current transport alone is insufficient for R9: Go may replay a replayable
request on a reused HTTP/1 connection, and HTTP/2 has its own stream replay
paths below the SDK retryer. Add one generic code-only `OneAttempt` transport
mode to `httpclient.Config`. In that mode it:

- enables HTTP/1 only;
- disables keep-alives so every request owns a fresh connection and cannot use
  net/http's stale-idle replay path;
- disables transparent compression so downloaded bytes reach checksum
  validation unchanged;
- retains the existing exact-authority, public-address, timeout, header/body,
  redirect, and proxy controls.

This mode and `DisableInstrumentation` are unconditional `httpclient` transport
options, not runtime `OUTBOUND_HTTP` or object-storage configuration. The
implementation removes the current `profile:authn-oidc-jwt` markers around the
field and branch, rewrites their OIDC-only comments as fixed-target secrecy
policy, and places the S3-independent proof in `httpclient/client_test.go`.
They carry no SDK dependency and remain valid when any profile retains
`httpclient`. The generic outbound client keeps its current pooled behavior
when `OneAttempt` is false.

Add one generic code-only `RootCAs *x509.CertPool` input to
`httpclient.Config`. Nil preserves every existing caller. When non-nil,
`httpclient` installs a `tls.Config` whose `RootCAs` is that pool, whose
`ServerName` is the already-validated base authority hostname, and whose
`InsecureSkipVerify` remains false. The S3 constructor is the only owner that
selects and builds its public-root pool; `httpclient` neither reads a trust
source nor mutates the pool. The pool is a startup snapshot and is never
modified after client publication.

Go 1.26.6 `net/http` shallow-clones that TLS config per connection and preserves
the pool pointer; it fills `ServerName` from the connection host only when the
field is empty. `crypto/tls` passes the non-nil pool and exact name into
`x509.Verify`, and `x509.Verify` calls the process-global `systemRootsPool` only
when `VerifyOptions.Roots` is nil. Consequently this S3 path cannot consult
`SSL_CERT_FILE`, `SSL_CERT_DIR`, or `GODEBUG=x509usefallbackroots`, and an
unrelated package's earlier or later system-root initialization neither changes
the S3 trust set nor becomes S3-retained state. The process-global pool remains
outside the S3 envelope only under that source-proven non-reachability; another
client that uses it owns its service-envelope charge.

The exact per-architecture source receipt resolves the Dockerfile build-stage
`golang:1.26.6-bookworm@sha256:116d58cbd88c1297624acc6e967a060012422bacf9930927e23fb719189c6f36`
index to `sha256:433f9dc4f8ea3a1ce4e28f9f15d0f7c056b10475307f886d6f1ac1ccc4abd976`
for `linux/amd64` and
`sha256:7939e2c75db3d059fc944bb6464a916d0fa64bd5a3bd7b3528f2a1ac7673a0eb`
for `linux/arm64`; it separately pins each Distroless platform manifest
and uses that platform image's source tree. It binds `net/http/transport.go` (`1c170e...`),
`crypto/tls/common.go` (`d2837f...`) and `handshake_client.go` (`7d6210...`),
and `crypto/x509/root.go` (`813fae...`), `root_unix.go` (`421c06...`),
`cert_pool.go` (`d995d6...`), and `verify.go` (`3fbc65...`) to their complete
SHA-256 values in the T9 receipt. The governing upstream source locations are
[`net/http.Transport.addTLS`](https://go.dev/src/net/http/transport.go),
[`tls.Config.RootCAs`](https://pkg.go.dev/crypto/tls),
[`x509.Verify`](https://go.dev/src/crypto/x509/verify.go), and the
[Unix system-root loader](https://go.dev/src/crypto/x509/root_unix.go).

The S3 loader reads at most D4's byte ceiling plus one byte, requires a regular
file, decodes the complete input with `encoding/pem`, and individually parses
each headerless `CERTIFICATE` block with `x509.ParseCertificate`. Missing,
unreadable, empty, oversized, trailing non-PEM data, a wrong block type or PEM
header, malformed or duplicate certificate, a certificate without valid CA
basic constraints, zero roots, or more than D4's certificate-count ceiling
fails startup without fallback. `CertPool.AppendCertsFromPEM` is deliberately
not the validator because it silently skips malformed blocks after accepting
another certificate. Tests may supply a bounded test-only root bundle through
an unexported construction seam; that class never reaches production config or
proves provider support.

An adapter-local HTTP `Doer` wrapper narrows every non-success and non-object
response body to the configured control-response limit, while a successful
whole-object GET may stream only through `max_object_bytes + 1`. It also attaches
a standard-library `httptrace.ClientTrace` and records whether request headers
could have been written. This is internal phase evidence only: a mutating error
after that point is conservatively `outcome_unknown`. It never records a URL,
header value, or key.

**Rejected.** The SDK default client, because it accepts ambient proxy/transport
behavior and supplies neither the repository's DNS trust gate nor a proven
single attempt. A second S3-specific dialer is also rejected because it would
duplicate the current trust owner.

Nil system roots plus image pinning or environment rejection is also rejected:
the Unix loader scans deployment-replaceable files/directories and initializes
one process-global pool, so neither image identity nor a small fixture bounds
the S3-retained trust path. Embedding a root bundle is finite but would move
public-root provenance, licensing, and update cadence into this repository.
`x509.SetFallbackRoots` is process-global and still does not create a private
S3 trust owner. The fixed, bounded image file is the smaller equivalent
mechanism while Delivery can prove that runtime mounts cannot replace it.

**Reopen.** If the pinned Go transport can still replay an HTTP/1 request on a
fresh non-reused connection, or if exact-provider conformance requires HTTP/2,
return to Technical Design before changing R9. Also reopen D2/D4 if Delivery
cannot prove the image-owned path is immutable, if production requires a
private/enterprise root, or if trust must rotate faster than image rollout and
process restart; the first condition compares an embedded bundle, while the
latter two reopen Specification's trust class or lifecycle.

### D3 — Admission and one effective context bound every path

One buffered-channel token set, sized at construction, is the complete
adapter-wide admission owner. Every public method attempts a non-blocking token
send. Failure returns `busy` before reading a body, signing, or provider I/O.
The token is released on method return except for a successful download, where
the returned body owns it until EOF or first Close. A guarded release makes EOF,
Close, error, and cancellation idempotent.

At admission the adapter derives one context whose deadline is the earlier of
the caller deadline and `start + max_operation_duration`. Every SDK call,
request write, body read, multipart stage, Abort, and ListParts page receives
that context. No cleanup or close path creates `context.Background`, a longer
deadline, a retry context, or a goroutine. Presign has no provider request but
still holds admission and checks the effective context before and after
signing.

Each admitted operation is synchronous and has at most one provider request in
flight. Multipart reads and uploads one part at a time; cleanup starts only
after the failed request has returned. The client default is `aws.NopRetryer`.
HeadObject and the request-to-headers GetObject phase override it with a fresh
`retry.Standard` instance capped at three attempts and one-second maximum
backoff. Its only retry predicate admits non-TLS `net.Error`, HTTP `429`, `500`,
`502`, `503`, or `504`, and provider code `RequestTimeout`,
`RequestTimeoutException`, `SlowDown`, or `TooManyRequests`; target denial,
certificate verification, context, and every other response are terminal. The
same effective context bounds attempts and jitter. Every mutation,
cleanup stage, and returned download body remains one-attempt. The transport's
`OneAttempt` mode forbids transparent `net/http` replay inside each SDK attempt,
and explicit credentials perform no I/O. Consequently provider request
concurrency cannot exceed the active token count and there is no internal
waiting queue.

The service's existing drain controls caller contexts. Shutdown closes the S3
HTTP client's idle transport in the dependency-close stage. One-attempt mode
normally has no idle connection, but the close remains the lifecycle invariant.
Shutdown does not extend active operations or reinterpret remote cleanup.

### D4 — The memory envelope is structural and startup-checked

No operation buffers an object or a multipart chunk. A single upload streams
through the SDK trailer checksum. Multipart streams each exact-length part
directly from the source through the SDK's part checksum while an adapter-owned
stdlib CRC64 hash observes the same bytes for final whole-object completion.
There is no prefetch and no second part buffer. A download streams through the
SDK EOF validator and the object-limit reader.

The S3 trust path is also structural. Production reads one fixed image file at
startup, parses every accepted root eagerly, adds the parsed certificates to a
private `x509.CertPool`, discards the PEM input, and thereafter shares that
immutable pool across the adapter's TLS configs. Eager `ParseCertificate` plus
`CertPool.AddCert` is selected over `AppendCertsFromPEM`: it makes malformed
input fail closed and places all root parsing/retention in the startup/shared
owner instead of allowing a first handshake to materialize lazy certificate
state. Per-handshake parent candidates and verified chains still scale with the
bounded root count and receive their own per-admission term below. The S3 path
neither creates nor retains Go's process-global system pool.

The pinned `UploadPart` success path does not deserialize a control body. S3
`v1.107.1` copies `ETag` and `x-amz-checksum-crc64nvme` response headers into
`UploadPartOutput`; the adapter retains only those two pointers plus its local
part number in each `types.CompletedPart`. Go 1.26.6 applies
`http.Transport.MaxResponseHeaderBytes` to the complete response header, so the
two provider strings are jointly `H`-bounded. `E > H` therefore does not make
the current deserializer's retained strings larger than `H`.

That correction does not close the original equation. During
`CompleteMultipartUpload`, the pinned generated serializer keeps the completed
part slice and strings live while it emits every part into a new
`bytes.Buffer`. Smithy XML escaping can expand one opaque byte to five bytes,
and the pinned Go buffer retains capacity above the logical XML length. Create
also leaves the `E`-bounded upload ID live for the rest of the multipart
operation; error deserializers copy the current `E`-bounded body before parsing.
Those are workload-growing terms and cannot hide inside a fixed `R_sdk`.

The exact pinned evidence is:

- `service/s3@v1.107.1/deserializers.go`,
  `awsRestxml_deserializeOpUploadPart` and
  `awsRestxml_deserializeOpHttpBindingsUploadPartOutput`: successful part
  values come from response headers; error bodies alone are copied and parsed;
- `service/s3@v1.107.1/types/types.go`, `types.CompletedPart`, plus
  `internal/infra/s3/upload.go`: the generated value has twelve pointer fields,
  while this adapter retains only `ETag`, `ChecksumCRC64NVME`, and a local
  `PartNumber`;
- `service/s3@v1.107.1/serializers.go`,
  `awsRestxml_serializeOpCompleteMultipartUpload` and
  `awsRestxml_serializeDocumentCompletedPart`: Complete is materialized in a
  `bytes.Buffer` while all retained descriptors remain reachable;
- `smithy-go@v1.27.7/encoding/xml/escape.go`, `escapeString`: every selected
  replacement is at most five output bytes per input byte;
- Go 1.26.6 `net/http/transport.go`, `bytes/buffer.go`,
  `runtime/slice.go`, and `runtime/msize.go`: `Transport.maxHeaderResponseSize`,
  `persistConn.Read`, and `persistConn.readLoop` make `H` bound the whole HTTP/1
  wire header; `Buffer.grow`/`growSlice` request no more than twice the final
  logical length; and `roundupsize` rounds large byte allocations by less than
  one 8 KiB page;
- `internal/infra/s3/checksum.go` and `upload.go`: accepted CRC64NVME text is
  the standard Base64 encoding of eight bytes, exactly twelve bytes, before a
  descriptor is retained.

The current tree resolves those sources as follows; the Linux implementation
receipt must reproduce these content identities rather than trusting version
labels alone:

```text
github.com/aws/aws-sdk-go-v2            v1.43.5  h1:yKT5GYnFWhuDo+DqKvE5ZPwVn3RjC4MAeBtZGlh6AVM=
github.com/aws/aws-sdk-go-v2/credentials v1.19.5 h1:xMo63RlqP3ZZydpJDMBsH9uJ10hgHYfQFIk1cHDXrR4=
github.com/aws/aws-sdk-go-v2/service/s3 v1.107.1 h1:VUTtUJMuRNMkb/7NIKmd8NQaeQLPGCMoTJxkYKre4qM=
github.com/aws/smithy-go                 v1.27.7 h1:Zgj5z4LfcDYoQIVk+n/yGdTkP/2y6ZT5vYxe0fp7bqE=
Go toolchain                             go1.26.6
```

For the repaired trust path the same Linux receipt additionally binds complete
SHA-256 identities for `encoding/pem/pem.go`
(`536954f803f79d8972e8f86b792b25d4ac83167fbe4c3117954a4378e60521da`),
`crypto/x509/parser.go`
(`70ae7c65f68d17a59e7bacb0d1cc520e8e96e8eafee2b7224465180437ab3a35`),
`cert_pool.go`
(`d995d6e88af70f36a345185420bac88c58e86ebed1b3eea8e087228eaa7da03b`),
`verify.go`
(`3fbc65e9ba1a710f1276d6b9e36483bbc9dd98f48817ed0991ca0894505783f9`),
`crypto/tls/common.go`
(`d2837fbe55a398c7362b3ff8ffe43c06d0832df2305f45f32aeaebadc784486d`),
`handshake_client.go`
(`7d6210c69ff9bf0d8506a8f2f59bf33a6132d0a3574487bcc356f858e50c6fab`),
`net/http/transport.go`
(`1c170ec3581321bd19ccd1b15863f46b16dc749ac6d54d00f5e97f7ffa2ccb5f`),
`crypto/x509/root.go`
(`813faea4e4990c9760b5fe99d8ae91c8f26c4d3b5dd3c4a6d7bef7b0f11dbae7`),
and `root_unix.go`
(`421c066f193250dc1adaf13fce4779fc3dc576e20faa86e43bc22af6c9536a14`).
Any identity mismatch fails the receipt rather than falling back to a version
label.

Let:

- `A` be maximum admitted operations;
- `O` be maximum object bytes;
- `C` be multipart chunk bytes and `P = ceil(O/C)`, with `P <= 10,000`;
- `H` be maximum response-header bytes;
- `E` be maximum control-response bytes;
- `K` be the fixed maximum common key bytes from R2;
- `Q` be the checked sum of the actual startup byte lengths of provider,
  endpoint, derived final authority, region, bucket, access-key identifier,
  secret, and session token;
- `B = 448 KiB` be the compile-time maximum bytes read from the fixed image CA
  bundle, with byte `B+1` used only to reject oversize input;
- `N = 288` be the compile-time maximum accepted unique CA certificates in that
  bundle;
- `T = 1,024` be the compile-time maximum content-type bytes. Empty remains
  absent; a longer or invalid HTTP field value returns `invalid` before
  admission, body read, signing, or provider I/O;
- `F = 2 MiB` be the design ceiling for fixed per-admitted-operation state;
- `S = 64 KiB` be the design ceiling for fixed once-per-adapter state;
- `U(x) = 64 * heap(x) + 64 KiB` be the design ceiling for one variable-input
  owner of at most `x` bytes. The factor pays for every simultaneously live
  backing copy and allocation rounding; the additive term pays for bounded
  maps, slices, structs, fixed-field XML tokens, and scratch;
- `trust_shared(B,N) = U(B + 128*N)` cover the eagerly parsed immutable root
  certificates, their DER and decoded fields, `CertPool` maps/slices/keys, and
  function/interface backing;
- `trust_startup(B,N) = trust_shared(B,N) + U(B+1)` cover the final pool while
  the bounded PEM read/decode input, including the oversize discriminator byte,
  is still live;
- `trust_verify(N) = 2 * 101 * heap(80*N)` cover per-handshake parent
  candidate slices, root-driven chain state, and retained verified-root
  references. Go's 100-signature-check limit plus the initial frame gives 101;
  `80*N` is five simultaneously reachable candidate-slice backing sets of
  16-byte `potentialParent` entries, and the factor two is the required 100%
  source headroom;
- `heap(0) = 0`; for positive `x <= 32 KiB`,
  `heap(x) = 2*x + 8`; otherwise `heap(x) = align_up(x, 8 KiB)`. This is a
  conservative upper bound over the pinned Go 1.26.6 allocator size-class and
  page rounding, not a language-version promise.

These are fail-closed ceilings, not values that Implementation may calibrate.
The pinned production-image receipt must also establish actual bundle bytes
`b <= B/2` and accepted unique roots `n <= N/2`, recording `B-b`, `N-n`, and
their unused percentages. Thus both external input cardinalities retain at
least 100% headroom before the allocation factors apply. If the pinned image
does not meet either half-ceiling, D4 reopens; Implementation may not raise it.
The pinned T9 Linux image currently supplies a separate source fact at the same
fixed path: 219,109 bytes, 144 certificate blocks, SHA-256
`27c6ae455d9ac17e4f86aaf1e72b1fe6850033f0f73e478232055b4719af2d90`.
That is 109% byte headroom and exactly 100% count headroom against `B/N`; it does
not substitute for the production image. The digest-pinned Distroless image
named below was re-inspected on 2026-08-17 after its base digest moved: its
regular image file is 224,449 bytes, contains 150 certificate blocks, and has
SHA-256
`714d457d580922dbf1d0be8bd35ba236a842b50b0072ae791582a19adef772a5`,
giving 104% byte and 92% count headroom. That count headroom no longer meets
`N`'s half-ceiling, so D4 is reopened to re-derive `N`; the ceiling itself is
unchanged at 288 and the reserves below, which are charged against it rather
than against the observed count, are unaffected. The previous digest carried
216,591 bytes and 142 blocks at 112% and 103%. Delivery still must prove the
deployed architecture image and no-override mount condition. At the design
maxima, `trust_shared = 32,047,104`,
`trust_verify = 9,309,776`, and `trust_startup = 61,997,056` bytes before the
other existing terms. These reserves are intentionally charged to the
configured process envelope; a small test CA does not reduce them.

#### D4 runtime-bundle identity and mount policy

For the T9A source receipt, the canonical image-root source is the final stage
of `build/docker/Dockerfile`, not the Go test/envelope runner. For the exact
Linux `GOARCH` whose 64-bit receipt is under review, resolve the Dockerfile's
`gcr.io/distroless/static-debian12:nonroot@sha256:1b7b9f0f0e0a1d2155f531db587cc48ec26aaf97ab64364225f5bf18a054e66a`
reference to its platform manifest, build the final image, and materialize an
otherwise unmounted stopped container from that exact image. The receipt records
the Dockerfile source identity, platform, index and resolved manifest digest,
final-image config/rootfs identity, and the final rootfs entry at exactly
`/etc/ssl/certs/ca-certificates.crt`. It fails if that entry is not the regular
`0555` image-owned file, if the final Dockerfile stage adds/replaces that path,
or if the no-mount materialization cannot reproduce it.

The receipt hashes and counts that extracted final-image file, then supplies
those same bytes through the unexported `imageRootSource` seam to the production
strict loader. Both observations must agree. The currently accepted final-image
facts are 224,449 bytes, SHA-256
`714d457d580922dbf1d0be8bd35ba236a842b50b0072ae791582a19adef772a5`, and
150 unique valid roots; they, and not the 219,109-byte/144-root Go runner file,
set D4's `b` and `n` headroom inputs. Running `productionImageRootSource` in the
Go compiler/test image is therefore a receipt failure, even if its file is
strictly valid and below the ceilings.

The final image's inherited `SSL_CERT_FILE` may name that path but never selects
the S3 source: `productionImageRootSource` remains the fixed-path open. The
local source receipt proves only the image artifact and its unmounted container
view. Delivery separately proves the deployed image identity and rejects every
non-root runtime mount whose destination is the bundle path or an ancestor;
that later no-override receipt is required for deployed and provider claims,
but is neither a T9A input nor a substitute for this source receipt. A changed
Dockerfile final stage, platform manifest, bundle entry, or mount policy
reopens D4 before the constants or T9 envelope may be reused.

The source receipt applies the following complete classification; an allocation
site may appear in exactly one row, and its variable driver may not be moved to
a larger cap to make the arithmetic pass:

| Receipt class | Exact current owners and source anchors | Charged term |
| --- | --- | --- |
| Shared fixed | Adapter/client/config shallow values, the admission channel header, resolver/dialer/transport/client structs, fixed SDK client and middleware registration, telemetry instrument handles, and fixed close bookkeeping in `internal/infra/s3/client.go`, `telemetry.go`, `internal/infra/httpclient/client.go`, AWS `service/s3/api_client.go`, and Smithy middleware stack construction. | `S`; the `A` channel cells are charged one per admitted operation under `F`. |
| Shared variable | The immutable config strings named by `Q`, derived authority/URL copies, credential-provider copies, and client/middleware values that retain or canonicalize those strings; plus the eagerly parsed adapter-private public-root pool, its maps/slices/DER/decoded fields, and the construction-only bounded PEM/decode input in `internal/infra/s3/image_root_bundle.go`, Go `encoding/pem`, `crypto/x509/parser.go`, and `cert_pool.go`. | Steady state is `U(Q) + trust_shared(B,N)`; the construction barrier substitutes `trust_startup(B,N)`. Original config strings are included, not caller-excluded. |
| Operation fixed | Admission/context/timer and telemetry-call state; one SDK input/output and middleware values; request/response wrappers; CRC/hash values; HTTP/1 request, TCP/TLS connection, fixed buffers, and read/write goroutines; bounded DNS result; and fixed error/signing/checksum/serializer scratch in the operation, Smithy, `net/http`, `crypto/tls`, and `crypto/x509` paths. Go 1.26.6 `crypto/tls` fixes ordinary handshake messages at 64 KiB and the certificate message at 256 KiB; those bytes and their fixed-path parse copies are included here because they do not scale with a D4 runtime input. Root-count-driven candidate/chain state in `crypto/x509/verify.go` is separated from this fixed set. | `F + trust_verify(N)`; one complete set per admission while a provider request is active. |
| Request variable | Copies and canonical forms driven by the current immutable tuple plus key and, for upload, content type. | `request_state(Q,K)` or `request_state(Q,K+T)`. |
| Response header | HTTP/1 wire bytes, textproto strings, header map/slices, selected output strings, and header-deserializer scratch driven by `H`. | `response_headers(H)`; retained multipart strings are removed from this transient owner and charged below. |
| Control response | Limited body copy, XML tokens/field strings, generated error/output values, and error formatting driven by `E`. | `control_response(E)`; a surviving upload ID is removed from this transient owner and charged below. |
| Multipart retained | CompletedPart slice/descriptors, part-number/string pointees, exact checksum backing, H-bounded ETag backing, E-bounded upload-ID backing, and simultaneous Complete XML backing. | `retained_parts(P,H) + upload_session(E) + complete_xml(P,H)`. |

For the pinned build, the receipt derives `used_F`, `used_S`, the piecewise
`used_U(x)`, `used_trust_shared(B,N)`, `used_trust_startup(B,N)`, and
`used_trust_verify(N)` from that table, every reachable allocation site, the
compiler escape/stack report, exact shallow sizes, pinned allocator rounding,
and maximum simultaneous counts at each named barrier. Acceptance requires
`used_F <= F/2`, `used_S <= S/2`,
`used_U(x) <= 32*heap(x) + 32 KiB`,
`used_trust_shared <= trust_shared/2`,
`used_trust_startup <= trust_startup/2`, and
`used_trust_verify <= trust_verify/2` for every input domain for which the
checked D4 arithmetic succeeds. The symbolic proof fixes the maximum live copy
count and container overhead; executable receipt cases cover `x=0`, every
allocator breakpoint, the overflow edge, `B`, `N`, and the actual `Q`,
`Q+K`, `Q+K+T`, `H`, and `E` inputs used by each envelope child.
Thus each design ceiling retains at least 100% source-derived headroom. The
receipt records used bytes, unused bytes, and unused percentage for every row.
A missing allocation site, an ambiguous class, a variable not dominated by its
named input, a non-finite simultaneous count, or a half-ceiling failure reopens
D4; T9 may not raise a ceiling, reclassify an owner, or reduce headroom.

The 64-bit build receipt fixes these additional constants from the generated
types and wire shape:

```text
completed_part_shallow = 96       # twelve 8-byte pointer fields
part_number_pointee    = 8        # allocator-rounded *int32
string_pointees        = 32       # two allocator-rounded *string headers
crc64_text             = heap(12) # exact Base64 CRC64NVME backing bytes
complete_root_bytes    = 101      # root open/namespace/close
complete_part_bytes    = 107      # tags + 12-byte CRC + 5-digit part number
complete_escape_factor = 5
complete_buffer_slack  = 8 KiB
```

For another `GOARCH`, the source receipt must derive these shallow/allocator
values from that exact build and the arithmetic test must use them; a silent
reuse of the 64-bit constants is startup rejection, not a portability claim.

The checked component functions are:

```text
retained_parts(P,H) =
    heap(96*P) + P * (8 + 32 + heap(12) + heap(H))

complete_xml_len(P,H) = 101 + P * (107 + 5*H)
complete_xml(P,H)     = 2*complete_xml_len(P,H) + 8 KiB
upload_session(E)     = 16 + heap(E) # retained *string header + upload-ID bytes
request_state(Q,X)    = U(Q + X)
response_headers(H)   = U(H)
control_response(E)   = U(E)
```

The conservative construction and admitted-operation reservation is:

```text
construction          = S + U(Q) + trust_startup(B,N)
shared                = S + U(Q) + trust_shared(B,N)

multipart_operation = F + trust_verify(N) + request_state(Q,K+T)
                        + response_headers(H) + control_response(E)
                        + K + T + upload_session(E)
                        + retained_parts(P,H) + complete_xml(P,H)
single_upload        = F + trust_verify(N) + request_state(Q,K+T)
                        + response_headers(H) + control_response(E) + K + T
download             = F + trust_verify(N) + request_state(Q,K)
                        + response_headers(H) + K
control_operation    = F + trust_verify(N) + request_state(Q,K)
                        + response_headers(H) + control_response(E) + K

required_memory = max(
    construction,
    shared + A * max(
        multipart_operation,
        single_upload,
        download,
        control_operation,
    ),
)
```

`max_object_bytes + 1` is a streamed byte limit, not a retained buffer, so it is
not added as object memory. Caller-owned source bytes are not charged until the
adapter/SDK copies or retains them. The formula intentionally sums `H` and `E`
parser reserves even when one response cannot exercise both. `H` alone owns the
retained success ETag and therefore its escaped XML representation; `E` owns
control/error body state and the retained upload ID. The original descriptors
remain charged beside the complete XML capacity. This conservative separation
proves maximum admission without a provider-specific ETag claim or a competing
bound authority.

Every addition, multiplication, ceiling division, `max`, `align_up`, integer
conversion, `Q` sum, trust term, and `A * per_operation` is checked before
client construction. Zero/negative values, `P` outside `1..10,000`, a non-64-bit
build using the 64-bit receipt, a trust file outside `B/N`, or any overflow
returns one startup configuration error before DNS, credential use, or provider
I/O. Arithmetic never wraps, saturates, or clamps. `MaxWorkingMemoryBytes` must
be at least `required_memory` exactly; equality is admitted and one byte less is
rejected.

`F`, `S`, `U`, `heap`, `B`, `N`, the three trust functions, the generated-type
sizes, the XML constants, the receipt classes, and the 100% minimum source
headroom are D4-owned accounting policy, not deployment defaults. T9's
read-only receipt materializes that fixed policy; it does not choose or tune it.
The receipt records `GOOS/GOARCH`, Go version, module versions and sums, exact
files/symbols, compiler escape/stack evidence, shallow and rounded allocation
sizes, bundle path/hash/bytes/count, maximum simultaneous counts, and used,
unused, and percentage headroom for every row. An unowned value, collection,
goroutine, buffer, copy, coefficient, root, or failed half-ceiling reopens D4.

The complementary measurable proof is credential-free. In the pinned Docker
Linux Go 1.26.6 environment, `GOMAXPROCS=1 make test-s3-envelope` runs five
identical staged children through the real low-level SDK. Each child retains
the production fixed-authority client and drives the SDK through a separate
fresh HTTP/1 TLS fixture transport with the same private roots, hostname
verification, connection policy, and response bounds. Retaining both clients
is a conservative memory superset; focused transport tests separately own the
production DNS/IP, authority, proxy, redirect, and ambient-root deny paths. The
measured child PID contains the runtime, adapter, SDK, both client transports,
and a pre-opened scalar barrier/control channel.
The controller and TLS fixture run in a separate unmeasured PID; IPC carries
only fixed-size stage/counter messages and never response payloads. The
controller makes the fixture ready before launching the child; each child then
warms only its runtime and quiescent IPC, samples before configuration and
adapter construction, and samples again at the held trust-construction barrier while the complete PEM
input and parsed pool are live, after idle adapter construction, at the held
operation peak, and after cancellation/join. The test bundle contains `N`
unique valid CA certificates with aggregate retained certificate input driven
to `B`; the fixture authority chains to one of them. Padding that only enlarges
the discarded PEM input cannot stand in for maximum retained DER/parsed state.
The fixture transport dials only the controller-owned local listener; it does
not claim to exercise the production DNS gate. The separate production-client
tests prove hostile `SSL_CERT_FILE`/`SSL_CERT_DIR`, ambient-only CA, private DNS,
wrong-host, proxy, redirect, and alternate-authority denial. No provider
endpoint or credential is used. Peak
cases hold `A=2` operations at each formula class, including
`2*10,000` source-equivalent descriptors with escape-amplifying `H`-bounded
ETags at the real Complete request barrier, filled `H` and `E` responses, and
object/chunk variants that prove no retained object or part buffer.

At each held barrier the controller reads `/proc/<pid>/smaps_rollup` `Rss`; the
sum of that file's resident categories must equal its `Rss` value. A nearby
`/proc/<pid>/status` `VmRSS` sample and Go `/gc/heap/live:bytes`,
`/memory/classes/heap/stacks:bytes`, `/memory/classes/os-stacks:bytes`, and
`/sched/goroutines:goroutines` samples are attribution diagnostics, not a
second pass/fail memory oracle. Go total mapped memory is deliberately excluded:
it may remain mapped after GC and is not current retained working set.

After controlled GC at barriers where GC cannot erase the state being proved,
repetition `i` computes deltas from the same preconstruction RSS for the held
trust-construction, idle adapter, and each held operation sample. The controller
sets `observed_i` to their maximum non-negative delta and
`observed = max_i(observed_i)`. The construction sample proves
`trust_startup`; the idle sample proves the retained adapter-private public-root
pool; and every
operation sample includes `shared`, all `A` held operations, and root-driven
verification state. `observed` must be at most both the independently reviewed
`required_memory` and the configured ceiling. `VmRSS` disagreement is reported
with both raw samples and timestamps but cannot overturn the `smaps_rollup`
result.

Cleanup is reclamation-independent. Before each child exits, cancellation must
join every child call; child adapter active/admission counts and open response
bodies/client connections must each be exactly zero; and a child `goleak`
comparison against its post-construction idle baseline must find no new
adapter, SDK, HTTP, TLS-client, or control goroutine. Through scalar IPC, the
unmeasured controller separately requires fixture in-flight requests, response
bodies, accepted connections, and fixture goroutines to reach zero and joins
the fixture process. Final child RSS and Go memory samples are diagnostic only
because the runtime may retain free pages. Fresh child exit prevents one
repetition's address space from entering another. Missing/unstable
`smaps_rollup`, an unexplained child RSS delta, a nonzero owner-side counter, or
an unjoined goroutine/process fails T9; Docker Linux availability makes none an
unsupported-runner exception.

**Rejected.** Substituting `E` or `max(H,E)` for the header-owned retained ETag,
dropping either independent header/control parser reserve, or hiding XML
buffer/parser amplification inside `F`: each duplicates an unrelated authority
or loses a live term when Complete holds both representations. Transfer-manager
part pools, a disk spool, an arena, a background
buffer recycler, and a process-wide Go memory-limit change. None is necessary
for one-pass low-level streaming, and the last would consume memory owned by the
rest of the service. A one-certificate fixture, prewarmed nil-root process pool,
or subtraction of system roots from the baseline is likewise rejected: none
bounds the trust state retained because of this adapter.

**Reopen.** Any SDK/Smithy/Go/transport/checksum version or build-target change;
a PEM/X.509 pool/parser/verification or TLS-config-clone change; image trust
path, bundle provenance/hash/bytes/count, or runtime-mount policy drift; a
second pool or reload path;
a field moving between header and body; another retained completion field;
changed XML tags/escaping/buffering; a content type above `T` becoming required;
receipt evidence exceeding `F`, `S`, `U`, `heap`, a trust half-ceiling, or a
shallow constant;
unknown/unbounded state; an unexplained process delta or worker; or inability to
hold the configured ceiling reopens D4 before constants, arithmetic proof, or
the Linux receipt can be accepted.

### D5 — Exact-length streaming and one common checksum policy

For both initial tuples the leading checksum policy is
`CRC64NVME/FULL_OBJECT`. R2 documents CRC64NVME as its only full-object checksum
type common to single and multipart objects; AWS documents it as a full-object
algorithm. That common choice is hidden from the port and earns support only
after each exact tuple passes conformance.

For single upload the adapter:

1. validates key, intent, declared length, content type, and ceilings before
   reading;
2. rejects multipart-sized `create_only`, and sets `If-None-Match: *` only on a
   qualifying single PutObject;
3. wraps the caller source in an exact-length reader and supplies the declared
   content length plus explicit CRC64NVME algorithm;
4. lets the pinned SDK hash the one-pass stream and emit its known-length HTTPS
   `aws-chunked` checksum trailer;
5. requires the SDK's computed-input-checksum metadata and the provider's
   returned CRC64NVME/FULL_OBJECT value to exist and match before success.

The exact-length reader never reads byte `length+1`, so extra bytes remain with
the caller. A short source is an error. The adapter owns the source after the
call begins and closes it exactly once. A `context.AfterFunc` closes it on
cancellation or deadline, so the required `io.ReadCloser.Close` must promptly
unblock an in-flight `Read`; the adapter adds no goroutine around an
uninterruptible reader. If any bytes could have been transmitted, conservative mutation-phase
mapping applies; local knowledge about the reader does not invent a provider
rollback.

For multipart replace the adapter:

1. creates one upload with CRC64NVME/FULL_OBJECT, total content type, and no
   create-only condition;
2. streams `C` bytes per non-final part and the exact remainder for the final
   part, serially, while one stdlib CRC64/NVME hash observes the complete source;
3. requests and verifies the CRC64NVME checksum for every UploadPart, retaining
   only the bounded completion descriptor;
4. completes with ordered part numbers/ETags/checksums, the computed whole-object
   CRC64NVME, `FULL_OBJECT`, and the declared multipart object size;
5. accepts success only after the SDK's embedded-error handling has consumed the
   final XML and the returned whole-object checksum/type match.

The stdlib table uses the pinned inverted NVME polynomial
`0x9a6c9329ac4bc9b5`; a known-answer check guards the wire encoding. The adapter
does not import AWS internal checksum code. SDK and adapter calculation observe
the same limited readers, so neither can silently choose another byte range.

For download the adapter sends checksum mode `ENABLED` and no Range, rejects an
unexpected Content-Range response, requires a non-negative declared size no
greater than `O` before exposing the body, and requires SDK result metadata to show that a
CRC64NVME validator was installed. Missing or composite/hyphenated checksum
metadata fails `integrity_failed` before returning a body. The returned wrapper
limits observed bytes to `O` and compares its own checksum at terminal Read.
Caller cancellation/deadline wins first; otherwise any terminal mismatch is
`integrity_failed`, so the adapter does not match an unexported SDK error string.
It releases admission on EOF, terminal read failure, or Close. Early Close
records cancellation only when the retained context is cancelled; an ordinary
incomplete Close records sanitized `internal` and is not integrity success. A
context callback closes an otherwise-idle body and releases its token at the
effective deadline; one joined finish path closes the provider body exactly
once at EOF, error, cancellation, or explicit Close.

The current R2 contract does not explicitly promise all of these trailer and
response fields. That is a conformance obligation, not a license to fall back
to ETag, MD5, an Amazon guarantee, buffering, or a provider escape.

### D6 — Multipart cleanup is immediate, serial, and conservative

Any failure after CreateMultipartUpload and before confirmed completion stops
new parts. A failed create whose request headers were written but whose upload
ID was not returned immediately reports `pending`; no safe direct cleanup is
possible without that identifier. When the ID is known and the same effective
context has time remaining, Amazon cleanup performs at most three serial cycles
of one AbortMultipartUpload followed by at most ten ListParts pages. Every page
sets `MaxParts=1000`; a truncated page supplies the next request's
`PartNumberMarker`, and a missing/non-advancing marker is invalid. A complete
empty traversal stops the immediate attempt; a complete traversal containing
any part starts the next abort cycle. Invalid continuation state, excess
pages/cycles, or exhausted context also stop the attempt. There is no
background reconciler or detached cleanup context.

Every failed multipart upload returns `pending`. Amazon explicitly allows an
in-flight part to finish after `AbortMultipartUpload`, so a bounded empty
`ListParts` observation is useful best-effort evidence but not a stable terminal
proof. R2 receives one bounded abort and also remains `pending`.

Cleanup disposition accompanies the primary stable upload error. It never
exposes the upload ID and never changes an ambiguous possible object commit
into a known rollback. Deployment's exact-provider abandoned-multipart
lifecycle rule is mandatory backstop evidence and is the terminal reclamation
boundary.

### D7 — One conservative error mapper owns provider and send-phase evidence

`internal/infra/s3/errors.go` maps from operation, phase, effective context,
whether headers could have been written, HTTP status, and structured Smithy API
code into the exact R9 kinds. It returns only sanitized `internal/objectstorage`
errors. Rules are ordered:

1. locally proven request/key/length/intent/expiry invalidity, admission, and
   configured size violations;
2. a more specific provider-confirmed precondition, denial, unambiguous absence,
   or integrity result admitted by that tuple's conformance table;
3. `outcome_unknown` for PutObject, CreateMultipartUpload,
   CompleteMultipartUpload, or Delete after
   request headers could have left the process and no authoritative result is
   available, even when caller cancellation or the effective deadline is the
   local error that ended the wait;
4. caller cancellation or effective deadline only for a non-mutating path, a
   pre-completion multipart stage that cannot commit the object, or a mutating
   request whose non-transmission was proved;
5. `temporary` only for a conformance-admitted failure before a non-mutating
   Metadata or pre-body Download result, or Amazon's documented create-only
   `409 ConditionalRequestConflict` without an automatic body replay;
6. sanitized `internal` for every remaining case.

401 and 403 become `denied` and never become absence. A 404 from GetObject or
HeadObject becomes `not_found`; this uses HTTP status for HEAD because Amazon
documents that exact exceptions are unavailable there. Amazon and R2
`412 PreconditionFailed` become `precondition_failed` only on single
create-only. Amazon-only `409 ConditionalRequestConflict` becomes `temporary`;
R2 409 and unrelated 412 evidence do not inherit that rule. For Head/Get only, 429 and 5xx remaining after bounded SDK
retry become `temporary`; the exact retry eligibility is D3's finite predicate.
Throttling does not make a mutation retry-safe.
Deferred download errors are never restarted.

Provider code, status, request ID, and phase may remain in a private diagnostic
record long enough to emit sanitized telemetry; the feature error never wraps
or formats the SDK error. No provider error text crosses the mapper.

This precedence is intentionally not ordinary context-first error handling. A
deadline explains why the process stopped waiting; it does not prove a
possibly-sent mutation failed to commit. The caller-visible result therefore
remains `outcome_unknown`, while safe diagnostics may separately record the
local terminal context phase.

### D8 — Adapter telemetry replaces generic URL instrumentation

The adapter emits one bounded operation-duration/result instrument, active and
rejected admission counts, completed mediated bytes, integrity failures,
single/multipart path, cleanup disposition, and presign issuance. Closed labels
are exactly operation, stable result kind, transfer path, and cleanup
disposition, with one `unknown` fallback. The design adds no provider label.

Safe spans may carry operation, phase, bounded attempt count, numeric HTTP
status, an allowlisted provider code, one closed failure category, bytes,
duration, active count, cleanup disposition, and a sanitized request ID. The
finite provider-code set is `AccessDenied`, `AuthorizationHeaderMalformed`,
`ConditionalRequestConflict`, `ExpiredToken`, `InternalError`,
`InvalidAccessKeyId`, `InvalidArgument`, `InvalidRequest`, `NoSuchBucket`,
`NoSuchKey`, `PreconditionFailed`, `RequestTimeTooSkewed`, `RequestTimeout`,
`RequestTimeoutException`, `ServiceUnavailable`, `SignatureDoesNotMatch`,
`SlowDown`, and `TooManyRequests`; another non-empty code becomes `other`.
Request IDs must match `[A-Za-z0-9._:/+=-]{1,128}`; another non-empty value
becomes `invalid`.

Classification precedence is exact: HTTP 401 or
`InvalidAccessKeyId|SignatureDoesNotMatch|ExpiredToken` is `credential`;
`httpclient.ErrTargetDenied` or a TLS/X.509 verification error is
`authority_tls`; HTTP 429 or `SlowDown|TooManyRequests` is `throttle`; a
no-response `net.Error` is `transport`; every other provider response is
`provider`; a local failure emits no category. They never carry key, bucket,
endpoint, provider/account, URL,
query, headers, content type, checksum, ETag, upload ID, access-key identifier,
credential, SDK error/type, provider message, or body. SDK client logging and
the generic HTTP URL instrumentation are disabled. Presign telemetry ends at
issuance and never claims direct transfer.

There is no storage readiness signal. Runtime failures are operation results;
selected local construction failure blocks startup.

### D9 — `OBJECT_STORAGE` is an independent compile-time profile selector

`scripts/init-module.sh` adds exactly `OBJECT_STORAGE=none|s3`, defaulting an
absent value to `none` and rejecting explicit empty/unknown values before
mutation. It records `object_storage` in `template.lock`, completion output, and
the idempotency comparison.

The `s3` inventory contains the port, adapter, bootstrap/config, operator doc,
bounded CA loader/tests, deterministic adapter proof, exact-provider
conformance entry point, profile CI lane, and AWS runtime dependencies. The
`none` path removes that inventory and `go mod tidy` removes AWS/Smithy modules
unless an independent future profile owns them. The runtime image's existing
public bundle is platform-owned and already present for other HTTPS clients; it
is not copied or generated by this profile.

`OBJECT_STORAGE` does not select `OUTBOUND_HTTP`. The lower-level `httpclient`
package is retained when any of object storage, bounded outbound HTTP, or OIDC
authentication needs it, but each selected dependency constructs its own fixed
authority and owns its own retry policy. All four object/outbound selector
combinations compile. The one-attempt, no-instrumentation, and `RootCAs`
transport fields are code policy, not object config keys, so retaining
`httpclient` alone does not retain object-storage user-visible configuration, a
trust-source path, or an AWS dependency.

Concretely, initialization changes the current removal predicate from
`outbound_http == none && authn == none` to
`outbound_http == none && authn == none && object_storage == none`. The object
profile oracle independently proves that `OBJECT_STORAGE=s3`, `AUTHN=none`,
`OUTBOUND_HTTP=none` retains `httpclient`, its unconditional one-attempt and
no-instrumentation/`RootCAs` source, and no generic outbound user configuration.
The
inverse `OBJECT_STORAGE=none`, `AUTHN=none`, `OUTBOUND_HTTP=none` output proves
the package absent; the other selector combinations prove neither capability
removes code owned by the other.

No selected output contains a usable endpoint, bucket, credential, or live
default. Non-secret config examples contain empty placeholders and required
finite bounds; secret environment examples are empty. No local emulator is a
runtime dependency or support claim.

#### Template closeout and provider-certification boundary

Template completion is the credential-free local closure of this selected
profile: deterministic `OBJECT_STORAGE=none|s3` generation and pruning, the
validated configuration/bootstrap/integration seams, and the T1-T10 local proof
surfaces. It establishes a fail-closed reusable capability; it does not
establish that either external provider is supported.

Amazon S3 and Cloudflare R2 certification are separate adopter-owned optional
handoffs. Each needs its own explicitly authorized, exact provider tuple and
receipt under the existing safety constraints: environment-only explicit
credentials (temporary session credentials for Amazon), a pre-existing
operator-owned bucket, mutations limited to the
owned conformance prefix, and no provisioning or change to bucket, identity,
network, encryption, versioning, lifecycle, or deployment controls. An Amazon
receipt never substitutes for an R2 receipt, and the absence, failure, or later
expiry of either receipt blocks only that provider-support claim. It neither
blocks nor reopens completed template closure.

No template generator, CI, aggregate gate, or completion disposition depends on
live provider credentials, a provider request or mutation, deployment, purchase,
or publication. The existing provider entry points remain fail-closed until an
adopter supplies the separately owned authorization and exact receipt inputs.

## Material flows

### Startup and shutdown

```text
load selected config
  -> validate exact provider tuple + finite bounds + memory equation
  -> publish the existing GC memory limit
  -> startup_object_storage reads and strictly parses the fixed bounded image CA
     bundle, then builds fixed-authority HTTP + explicit-credential S3 client
     and returns one objectStorageRuntime (store + idempotent close; no network)
  -> immediately defer its early-return CloseIdleConnections safety close
  -> construct the existing runtimeDependencies
  -> serve ready without a bucket probe

after HTTP drain and background join
  -> objectStorageRuntime.CloseIdleConnections in the existing
     dependency-close window
  -> runtimeDependencies.Close under the same bounded shutdown stage
  -> deferred idempotent safety closes are no-ops
  -> telemetry flush remains last
```

`run.go` calls `wiring.initObjectStorage(startupCtx, bootstrap)` immediately
after `applyMemoryLimit`/request-budget reporting and before
`wiring.dependencies`. `runtimeWiring` gains that exact profile-owned function;
its signature uses `startupBootstrap`, so it does not make the existing
`config` import survive an unrelated profile marker. `startup_object_storage.go`
owns the concrete unexported `objectStorageRuntime`, which retains the
`objectstorage.Store` and its idempotent `CloseIdleConnections`. There is no
generic close registry and it does not join DATABASE-owned
`runtimeDependencies`.

No current feature is injected with the store: the template profile constructs
and validates the capability, while a later adopter wires the retained port at
its feature composition point. `run.go` retains the runtime only for lifecycle.
On early return its immediate defer closes it. On ordered teardown, after HTTP
drain and background join, `run.go` closes it before
`runtimeDependencies.Close` in the existing dependency-close window and before
the already-last telemetry flush. Both closes are idempotent, so their deferred
safety nets cannot duplicate an effect.

Trust rotation is an image release and process restart, never an in-process
file refresh. Delivery publishes a bundle containing the old and new provider
chains, binds its path/provenance/SHA-256/bytes/count and read-only no-override
mount policy to each architecture image, runs both exact-provider TLS receipts,
deploys new processes, and only then retires the old chain/processes. Old and
new binaries may coexist because there is no shared state or wire-contract
change. Rollback to the nil-root predecessor restores ambient process-global
trust and is a security regression unless that predecessor receives its own
closed trust proof; the default recovery is roll-forward with the last accepted
bounded bundle.

### Single upload

```text
feature authorizes operation/key and chooses intent
  -> port validates key/request
  -> non-blocking admission + one effective context
  -> exact-length source + explicit CRC64NVME PutObject
  -> fixed authority / one transport attempt / streamed trailer
  -> compare calculated and provider-confirmed full checksum
  -> sanitized completion or stable error
  -> release admission
```

### Multipart upload and cleanup

```text
validated replace above C
  -> admission/context
  -> CreateMultipartUpload(CRC64NVME, FULL_OBJECT)
  -> [limited source -> whole-object CRC -> serial UploadPart -> verify part] x P
  -> CompleteMultipartUpload(parts, whole CRC, FULL_OBJECT, declared size)
  -> verify embedded result + final checksum
  -> success

any pre-completion failure after upload ID
  -> stop parts
  -> Amazon: at most 3 serial Abort + bounded paginated ListParts cycles
  -> R2: one Abort and pending
  -> complete only on Amazon's full empty traversal; otherwise pending
  -> return primary stable error + disposition
```

### Download body lifetime

```text
feature authorizes key
  -> admission/context
  -> one checksum-enabled GetObject
  -> validate metadata, object ceiling, accepted checksum validator
  -> return bounded body while retaining token/context
  -> reads hash through EOF
     -> EOF match: success + byte metric + release
     -> mismatch/deferred/cancel/deadline/too-large: stable read error + release
     -> early Close: incomplete, release
```

### Metadata, delete, and presign

- Metadata is one HeadObject and exposes only size, optional content type, and
  UTC last-modified. It is not a version token.
- Delete is one unversioned DeleteObject. Success means only operation
  completion; a send-phase loss is `outcome_unknown`.
- Presign performs local SigV4 GET issuance after authorization and admission,
  validates `1s <= requested <= configured <= 7d`, rechecks method/final
  authority/query-bearing URL/returned headers, single query values, exact
  access-key/date/region/`s3`/`aws4_request` credential scope, session-token
  presence, provider-specific expected owner, and requested lifetime, then
  exposes `SignatureExpiresAt`, the exact expiry encoded by the signature. A
  session credential may expire or be revoked earlier. No provider call or
  mediated-transfer claim occurs.

## Provider operation and conformance matrix

| Common behavior | Amazon S3 current contract | Cloudflare R2 current contract | Design consequence |
| --- | --- | --- | --- |
| Authority/credential identity | Regional virtual-hosted SigV4 with matching region, expected 12-digit bucket owner, and temporary session credential | Exact default/EU/FedRAMP account S3 endpoint, virtual host, region `auto`, explicit long-lived or temporary R2 credentials, and no expected-owner field | One hidden provider tuple validates different fixed region/endpoint/credential rules; required presign headers stay opaque to the port. |
| Public TLS trust | Exact virtual authority chains to the image-owned public-WebPKI snapshot and passes hostname verification | Same, independently, for the exact R2 account authority | Each provider receipt records the image bundle provenance/path/SHA-256/bytes/count plus its own successful chain and hostname; neither provider result substitutes for the other or for the production-image receipt. |
| Single create/replace | PutObject supports checksum fields, Content-Type, `If-None-Match:*`, and returned checksum/type | Matrix lists PutObject, conditional header, Content-Type, CRC64NVME/FULL_OBJECT | Same request vocabulary; R2 trailer acceptance and returned fields require exact proof. |
| Multipart replace | Create/part/complete, full CRC64NVME, object size, 2xx embedded-error handling | Operations and CRC64NVME/FULL_OBJECT are listed | Same hidden path; every R2 create/part/complete checksum field requires exact proof. Multipart create-only stays invalid. |
| Download | GetObject checksum mode returns stored checksum; SDK validates at EOF | GetObject and checksum mode are documented, but ordinary GET response fields are not fully enumerated | Require CRC64NVME validator before exposing body; R2 must prove it. |
| Metadata/absence | Head fields documented; HEAD exposes generic HTTP status and 403/404 disclosure depends on ListBucket permission | Head implemented; NoSuchKey, Unauthorized, and AccessDenied documented | Head 404 maps only pinned `NotFound`/`NoSuchKey`, Get 404 only exact `NoSuchKey`; 401/403 maps to denied; each provider receipt proves both with its intended identity policy. |
| Delete | Versionless delete has accepted meaning only if versioning was never enabled | Delete implemented; S3 versioning is unsupported | Deployment precondition differs; port meaning remains operation completion only. |
| Presigned GET | SigV4 GET, 1 second through 7 days; expected owner is a signed query field; unsigned Range may alter the representation | Same documented expiry range and virtual-host GET, excluding custom domains; expected owner is absent | Same result vocabulary; exact method/authority/query/opaque headers/reuse remain per-tuple proof, and neither result claims a full transfer. |
| Abort proof | In-flight parts may finish after abort; repeated abort and ListParts emptiness may be required | Abort/ListParts implemented, default lifecycle documented, terminal cleanup semantics incomplete | Amazon gets bounded repeated-abort/paginated best effort; both providers remain `pending` and require their own lifecycle receipt. |

One provider passing never covers the other. A local scripted transport proves
adapter policy only. An emulator, if later chosen by Test Design, proves only
its exercised wire subset. Deployment-path identity, DNS/TLS/egress, bucket
policy, versioning, encryption, lifecycle, quota, telemetry delivery, and the
image bundle's read-only no-override mount policy remain separate deployment
evidence. A private fixture CA proves only the bounded loader/TLS mechanism and
memory envelope.

## Go responsibility and file map

### Production owners

| Path | Single reason to change |
| --- | --- |
| `internal/objectstorage/doc.go` | Shared feature/adapter contract and deliberately absent provider-policy seam. |
| `internal/objectstorage/store.go` | Provider-neutral port, request/result values, and common key grammar. |
| `internal/objectstorage/errors.go` | Closed stable error and cleanup vocabularies. |
| `internal/infra/s3/doc.go` | Multi-stage adapter contract, explicit immutable public-root snapshot, and deliberately absent provider registry/factory/trust reload. |
| `internal/infra/s3/config.go` | Exact provider-tuple, expected-owner, R2 jurisdiction-host, and credential-shape validation; pinned D4 constants including `B/N` trust reserves, checked amplification/serialization arithmetic, and startup memory/resource rejection. |
| `internal/infra/s3/image_root_bundle.go` | Fixed-image-path, byte/count-bounded, strict PEM/X.509 public-root loading into one adapter-private immutable pool. |
| `internal/infra/s3/client.go` | Bounded trust loading, direct SDK construction, default mutation non-retry plus bounded read retry, expected-owner projection, admission, effective context, and lifecycle. |
| `internal/infra/s3/transport.go` | SDK HTTP response narrowing plus attempt and possible-send phase evidence. |
| `internal/infra/s3/checksum.go` | Hidden CRC64NVME policy, wire encoding, and checksum metadata comparison. |
| `internal/infra/s3/upload.go` | Exact-length single/multipart upload, the 1,024-byte valid content-type boundary, bounded completion state, and immediate cleanup state machine. |
| `internal/infra/s3/download.go` | Bounded-retry GetObject acquisition, range refusal, metadata-checked streaming body, terminal integrity, ceiling, deadline, close classification, and token release. |
| `internal/infra/s3/metadata.go` | Bounded-retry HeadObject projection and HTTP-status absence semantics. |
| `internal/infra/s3/delete.go` | Unversioned mutating delete and possible-send outcome. |
| `internal/infra/s3/presign.go` | Local bounded SigV4 GET issuance and bearer-result validation. |
| `internal/infra/s3/errors.go` | Conservative tuple/send-phase classification plus bounded private provider diagnostics and sanitization. |
| `internal/infra/s3/telemetry.go` | Closed safe adapter signals, diagnostic span projection, and redaction boundary. |
| `internal/infra/httpclient/config.go` | Declare/validate unconditional one-attempt, no-instrumentation, and code-only caller-selected `RootCAs` transport policy. |
| `internal/infra/httpclient/client.go` | Apply HTTP/1 fresh-connection exact-byte mode and a non-nil pool with the validated base hostname; remove OIDC-only markers from instrumentation bypass. |
| `internal/config/object_storage_config.go` | Object section type, empty/zero profile defaults, canonicalization, and fail-closed section validation. |
| `internal/config/types.go` | Register the profile-marked object-storage section. |
| `internal/config/defaults.go` | Merge the selected section's deliberately non-usable defaults. |
| `internal/config/validate.go` | Invoke selected object-storage validation in root validation order. |
| `internal/config/secret_policy.go` | Treat all three static credential inputs, including the access-key identifier, as environment-only credential material. |
| `cmd/service/internal/bootstrap/startup_object_storage.go` | Construct the concrete profile runtime and own idempotent idle-transport close. |
| `cmd/service/internal/bootstrap/run.go` | Wire exact post-memory-limit construction, early close, and ordered dependency-window close. |

`internal/infra/s3` imports `internal/objectstorage`, `internal/infra/httpclient`,
the repository observability owners, AWS SDK core/credentials/S3, Smithy error
interfaces, and the standard library. `internal/objectstorage` imports none of
those runtime packages. Feature packages import only `internal/objectstorage`.
Bootstrap is the only owner importing config and the concrete adapter together.
No import reaches from either shared package into a feature or composition root.

### Proof, fixtures, docs, and profile owners

| Path | Ownership |
| --- | --- |
| `internal/objectstorage/store_test.go` | Portable key/value/error contract and small feature fake compatibility. |
| `internal/infra/s3/harness_test.go` | Scripted HTTP fixture shared by two or more adapter owner tests; no assertions or policy. |
| `internal/infra/s3/config_test.go` | Exact tuple, provider constraints, part count, every D4 component/boundary/overflow including trust terms, and startup memory rejection. |
| `internal/infra/s3/image_root_bundle_test.go` | Fixed image source, exact `B/N` and invalid-input boundaries, eager adapter-private public-root pool, ambient-root non-reachability, and hostname/chain denial. |
| `internal/infra/s3/client_test.go` | Direct static client, admission, effective context, no provider/network work, lifecycle contract, and the source-receipt/process-envelope child modes. |
| `internal/infra/s3/transport_test.go` | Response limits, final authority, possible-send phase, and adapter Doer behavior. |
| `internal/infra/s3/checksum_test.go` | CRC64NVME known-answer and SDK metadata/EOF integrity boundary. |
| `internal/infra/s3/upload_test.go` | Exact-length/content-type boundaries, single/multipart selection, serial parts, bounded retained descriptors, confirmation, and immediate cleanup. |
| `internal/infra/s3/download_test.go` | Body ceiling, deadline, close/release, missing checksum, and deferred EOF result. |
| `internal/infra/s3/metadata_test.go` | Portable metadata and exact absence/denial projection. |
| `internal/infra/s3/delete_test.go` | Unversioned success and after-possible-send ambiguity. |
| `internal/infra/s3/presign_test.go` | GET-only TTL, fixed authority, required headers, and bearer redaction. |
| `internal/infra/s3/errors_test.go` | Closed conservative provider/context/send-phase error table. |
| `internal/infra/s3/telemetry_test.go` | Closed label product and secret/key/URL/provider-text corpus redaction. |
| `internal/infra/httpclient/client_test.go` | Generic one-attempt HTTP/1/no-keepalive/no-compression behavior, unconditional instrumentation bypass, and non-nil/nil `RootCAs` transport behavior. |
| `internal/config/object_storage_config_test.go` | Selected required fields, tuple canonicalization, and finite bound validation. |
| `internal/config/snapshot_contract_test.go` | Exact object section leaf/default/snapshot inventory, including intentional zero/empty required values. |
| `internal/config/secret_policy_test.go` | File-source rejection and empty-placeholder allowance for all static credential fields. |
| `cmd/service/internal/bootstrap/startup_object_storage_test.go` | Bounded local trust-source construction with no DNS/provider I/O, selected startup failure, no readiness probe, and idempotent close. |
| `cmd/service/internal/bootstrap/run_lifecycle_test.go` | Construction after GC limit and object close after drain/background but before dependency close/telemetry flush. |
| `test/s3conformance/conformance_test.go` | Credentialed Amazon and R2 tuple entry points isolated from unrelated integration fixtures; results remain provider-specific. |
| `docs/s3-compatible-object-storage.md` | Operator inputs, image-bundle provenance/rotation/receipt, exact support/evidence boundary, feature/deployment ownership, lifecycle and versioning preconditions. |
| `build/docker/Dockerfile` | Existing digest-pinned runtime-image owner; no new copy or path is added, and Delivery receipts the bundle already present in that image. |
| `env/.env.example` | Empty environment-only static credential inputs; no usable endpoint, bucket, or credential. |
| `env/config/local.yaml` | Non-secret selected placeholders and finite-bound requirements; no secret value or usable provider target. |
| `docs/configuration-source-policy.md` | Canonical YAML-versus-environment ownership for object settings, including all three environment-only static credential fields. |
| `scripts/init-module.sh` | Selector validation, exact profile removal/retention markers, lock/output values, and three-way `httpclient` retention predicate. |
| `scripts/ci/template-init-check.sh` | Independent selected/absent inventories, all four object/outbound combinations, marker survival, idempotency, and dependency pruning. |
| `scripts/ci/ci-change-scope.sh` | Route every object-profile owner, including the build-tagged conformance file, through the template-required CI lane and self-test that classification. |
| `scripts/ci/s3-source-receipt.sh` | Fail-closed Dockerfile-derived Go and Distroless platform manifests, source identities, fixed root-bundle receipt, and separate `linux/amd64` or `linux/arm64` target proof. |
| `.github/workflows/ci.yml` | One credential-free object-profile generator/compile/image lane. |
| `Makefile` | Discoverable deterministic and credentialed S3 proof entry points without making live conformance a merge prerequisite. |
| `go.mod` | Direct selected AWS module requirements and no rejected client/transfer manager. |
| `go.sum` | Checksums for that exact selected dependency closure. |
| `docs/repo-architecture.md` | Register the reusable port and one outbound S3 adapter seam. |
| `docs/project-structure-and-module-organization.md` | Register package/file placement and the independent object profile. |
| `docs/build-test-and-development-commands.md` | Document deterministic and exact-provider proof entry points and claim limits. |
| `README.md` | Advertise the optional capability and its exact-provider evidence boundary. |

This inverse map is the implementation owner map; changing a listed production
or proof owner requires a focused Go Ownership repair rather than an
Implementation-time file choice. The scripted HTTP seam is the AWS client's
existing `HTTPClient` option and an unexported constructor input, not a
production provider interface. No protocol stage receives another interface or
factory.

### T10 lint-repair routing

The current-tree `make lint` gate is a T10 aggregate proof failure, not an
ownership or mechanism failure. The fixed map above already gives every
reported path one semantic and proof owner; therefore no package, file,
dependency, generated-source, or selector decision changes before a narrow
implementation repair. The repair preserves the existing return/error, context,
lifecycle, and image-root contracts; it may use behavior-preserving local code
or a precisely justified local suppression, but it may not silence an
ownership, dependency, or security finding by weakening `.golangci.yml`.

The 2026-08-13 current-tree reproduction (`make lint`, exit 2) reports exactly
169 findings. Every finding has one existing repair owner below; zero current
findings belong to T1 or T9. Each named owner reruns its existing focused
proof, then the exact whole-tree lint gate remains T10's proof:

| Findings | Current lint surface | Repair owner and bounded route |
| --- | --- | --- |
| 3 | `internal/infra/httpclient/client_test.go:156,162,167` | Reopen S3 T2 only: generic one-attempt transport proof; this is shared infrastructure, not outbound OAuth behavior. |
| 8 | `internal/infra/s3/{config,client,transport}.go` and `transport_test.go` | Reopen S3 T3 only: exact static tuple, direct construction, and one-attempt adapter boundary. |
| 27 | `internal/infra/s3/{checksum,upload}.go` and `upload_test.go` | Reopen S3 T4 only: exact-length/checksum/multipart/cleanup behavior. |
| 15 | `internal/infra/s3/download.go` and `download_test.go` | Reopen S3 T5 only: validated-EOF body lifetime and release. |
| 21 | `internal/infra/s3/{metadata,delete,presign,errors}.go` and their direct tests | Reopen S3 T6 only: portable projections, possible-send classification, and stable sanitized errors. |
| 4 | `internal/infra/s3/telemetry.go` and `telemetry_test.go` | Reopen S3 T7 only: closed signals, spans, context propagation, and token release. |
| 2 | `cmd/service/internal/bootstrap/startup_object_storage.go` | Reopen S3 T8 only: local construction, config, readiness neutrality, and close ownership. |
| 1 | `cmd/service/internal/bootstrap/run.go` | Reopen outbound-machine-auth T7-R0 only: the shared visible bootstrap lifecycle sequence; both its existing outbound T3 and S3 T8 lifecycle proofs must pass. |
| 22 | `internal/httpidempotency/{contract,result}.go`, `internal/infra/http/idempotency*`, `internal/infra/http/router.go`, their direct tests, and `internal/problem/problem.go` | Reopen HTTP-idempotency T1 only: declaration, envelope, rendering, and closed Problem ownership. |
| 15 | `internal/infra/postgresidempotency/{store_acquire,store_reserve,store_epoch,store_reconcile,telemetry}.go` | Reopen HTTP-idempotency T2 only: writer-authoritative Store arbitration, outcome mapping, and exact epoch ownership. |
| 1 | `internal/config/snapshot_contract_test.go:95` | Reopen HTTP-idempotency T3 only: the active-only idempotency configuration inventory. |
| 12 | `internal/infra/postgresjobs/{store,store_operation,store_rows,store_schema}.go` and direct tests | Reopen durable-jobs T2 only: canonical schema/generated boundary, reserved Session, and jobs-config ownership. |
| 5 | `internal/infra/postgresjobs/{store_accept,store_producer_probe}.go` | Reopen durable-jobs T3 only: producer admission and caller-owned staging. |
| 12 | `internal/infra/postgresjobs/{store_claim,store_finalize,store_observe,store_renew,store_rescue}.go` | Reopen durable-jobs T4 only: fenced persisted transitions. |
| 10 | `internal/infra/postgresjobs/{engine,engine_claim,engine_attempt}.go` and direct tests | Reopen durable-jobs T6 only: coordinator claim/attempt ownership. |
| 1 | `internal/infra/postgresjobs/engine_renew.go` | Reopen durable-jobs T7 only: lease-renewal ownership. |
| 9 | `internal/infra/postgresjobs/engine_drain.go`, `cmd/jobs-worker/internal/bootstrap/lifecycle{,_test}.go`, and `internal/config/config.go` | Reopen durable-jobs T8 only: worker drain/readiness/exit and worker-scoped configuration. |
| 1 | `cmd/outbox-relay/main.go` | Reopen outbox-production-closure T4 only: profile-selected relay publisher construction. |

The current S3 `internal/infra/s3` inventory is 75 findings; the three generic
`httpclient` findings are T2, and the two object-bootstrap findings are T8.
There is no current explicit-root `httpclient`, image-root, source-receipt, or
Linux-envelope report, so T9A/T9 transfer their accepted receipt but do not
open a repair. There is likewise no current outbound OAuth package report;
T7-R0 is the only current outbound cross-ledger repair.

Run the shared dirty-tree recovery serially: S3 T2 through T8; outbound
T7-R0 after S3 T8; retain T9A/T9's existing receipt without a repair; then
HTTP-idempotency T1 through T3; durable-jobs T2, T3, T4, T6, T7, and T8 with
its accepted T5 receipt as the dependency; and outbox-production-closure T4.
Each repair returns its unchanged focused proof, candidate identity, and
lint-clean surface to the T10 Lead. Only after every return may T10 rerun
`make lint` and obtain fresh implementation review. HTTP-idempotency,
durable-jobs, and outbox-production-closure owners need this route notification
before dispatch; outbound needs none because its existing T7-R0 handoff already
names the S3 serial recovery lead.

Reopen this Go Ownership decision only if a repair needs a new file/package,
an import or generated/manual change, a different shared transport boundary,
or a changed observable/error/lifecycle policy. Otherwise the named accepted
unit or shared-bootstrap prerequisite is implementation-ready; after every
repair, rerun its existing focused proof and finish with T10's exact `make
lint` and fresh implementation review. Provider certification remains outside
this route.

## Dependency and source ownership

The selected runtime source is pinned as direct dependencies where imported:

```text
github.com/aws/aws-sdk-go-v2                         v1.43.5
github.com/aws/aws-sdk-go-v2/credentials             v1.19.5
github.com/aws/aws-sdk-go-v2/service/s3              v1.107.1
github.com/aws/smithy-go                             v1.27.7
```

`feature/s3/transfermanager`, MinIO, SimpleS3, and kelindar/s3 do not enter the
module graph. Go's `hash/crc64` owns CRC computation; the adapter owns only the
fixed NVME table selection, base64 wire representation, and comparison policy.
AWS generated models remain dependency authority and are never copied or
edited.

D4 additionally pins the Go toolchain source at the Dockerfile's `go 1.26.6`. Its source receipt
must resolve the selected module versions through `go list -m`, verify their
`go.sum` identities, and cite the generated/deserializer/serializer, Smithy XML,
Go transport/allocator, PEM/X.509 pool/parser/verifier, and TLS clone/handshake
symbols named in D4. The production-image receipt separately binds
`gcr.io/distroless/static-debian12:nonroot@sha256:1b7b9f0f0e0a1d2155f531db587cc48ec26aaf97ab64364225f5bf18a054e66a`
for each architecture to the fixed public bundle path,
upstream provenance/revision, SHA-256, bytes, accepted unique count, regular
read-only ownership, and absence of a replacing runtime mount. A source link,
module/image version, or small fixture without the local resolved file, build
target, and bundle identity is not a reserve or trust receipt.

The pin reopens on any Go or dependency update that changes endpoint resolution,
credential retrieval, retry, signing, checksum framing/metadata, response body
validation, UploadPart header/body ownership, completed-part shape,
CompleteMultipartUpload serialization/escaping/buffering or embedded-error
handling, allocator rounding, goroutine/stack/buffer ownership, presign output,
Smithy error shape, PEM/X.509 parsing/pool/verification, TLS config cloning, the
bundle path/content/provenance, or the image/mount contract. The same Technical
Design comparison and D4 receipt must be refreshed before substituting another
family or accepting new constants.

## Reopen and stop conditions

Return to **Technical Design** when:

- the selected SDK or one-attempt transport cannot preserve fixed authority,
  zero replay, deadline propagation, streaming checksums, bounded retained
  memory, EOF validation, serial multipart, or immediate cleanup;
- the source-reviewed memory reserve or process-envelope evidence cannot bound
  a pinned SDK/Go behavior, opaque completion field, XML/control/header
  amplification, upload-session value, configured-string input, public-root
  pool, root-count-driven verification state, construction peak, or build
  target;
- the fixed bundle is replaceable at runtime, exceeds the `B/N` half-ceilings,
  cannot retain public-root provenance, or requires another pool/reload owner;
- exact-provider evidence fails the selected trailer/checksum/cleanup mechanics
  but another mechanism may realize the same feature contract;
- a dependency update triggers one of the source-ownership conditions above.

Return to **Specification** when:

- any of the five operations needs a provider-specific feature field or result;
- portable key, credential, absence, checksum, create-only, delete, or presign
  semantics cannot hold for both exact tuples;
- production private/enterprise trust, operator-configurable roots, or trust
  refresh faster than image rollout and process restart becomes required;
- no mechanism can prove process-wide admission/memory/deadline or multipart
  cleanup under R4/R5/R9;
- fewer than two real feature/adopter cases remain, or the reusable shared pack
  otherwise collapses.

The D4 system-root blocker is closed in Technical Design. This repair changes
existing Test Design inputs, so the exact next owner is **Test Design** for a
focused reopen of TD-002, TD-003, TD-005, TD-006, TD-014, TD-015, TD-016,
TD-017, and TD-018: construction now performs bounded local trust-file I/O;
TLS fixtures must exercise the explicit test-only pool and hostile ambient-root
denials; the source/formula/process oracle must cover `B/N`, construction,
retained pool, and verification state; config/profile oracles must prove no new
production CA setting and inventory the new S3-owned files; and each provider
receipt must bind the production bundle and successful public chain/hostname.
TD-001, TD-004, TD-007 through TD-013, and TD-019 retain their accepted inputs.
Test Design decides its own repaired artifact and downstream
invalidations. This phase does not edit `test-plan.md`, the ledger, code, tests,
scripts, config, credentials, provider state, deployment, T1-T8, T9's recorded
state, T10-T12, or TD-019, and stops before any of them.

## Review evidence

### Go Ownership panel — first fixed candidate

Candidate SHA-256:
`7a58201090abb75d34851ab2554a27ac777925175964460352e5b36df6f065ad`.
All reviewers were read-only.

| Required lens | Result | Material finding and disposition |
| --- | --- | --- |
| Responsibility and execution path | **FAIL** | `DisableInstrumentation` was still OIDC-marker-owned despite S3 use, and bootstrap named a nonexistent close registry instead of the current `run.go` lifecycle. Repaired by unconditional `httpclient` ownership, an exact profile oracle, and concrete post-memory-limit construction plus early/ordered close. |
| Package and dependency architecture | **FAIL** | The same marker/retention gap broke `s3 + no authn + no outbound`, and object runtime close was not assigned outside DATABASE-owned `runtimeDependencies`. Repaired with the exact three-selector retention predicate and a separate concrete `objectStorageRuntime`. Port/import direction and exact dependency pins had no finding. |
| File cohesion and naming quality | **FAIL** | Grouped inverse-map rows, `operations.go`, a generic adapter test dump, missing `doc.go`, and `Get` rather than `GET` initialism left ownership choices open. Repaired with one exact row per changed owner, split metadata/delete/presign paths and matching tests, fixture-only `harness_test.go`, explicit package docs, and `PresignGET`/`PresignedGET`. |

The repair also corrected the conservative memory equation to include response
headers in single and multipart upload reservations. A focused panel re-review
must PASS the repaired fixed candidate before the broader Technical Design
review begins.

### Go Ownership panel — focused repaired candidate

Candidate SHA-256:
`57cbcc73c3177932fa5895a37be2759333c3f88a2199dc7bfc304fba9e939bd2`.

| Required lens | Result | Receipt |
| --- | --- | --- |
| Responsibility and execution path | **PASS** | Unconditional shared transport ownership, exact profile oracle, post-memory-limit construction, and early/ordered close all match current lifecycle seams; no repair regression survived. |
| Package and dependency architecture | **PASS** | The three-selector retention predicate, separate concrete object runtime, marker-safe bootstrap signature, import direction, source pins, and SDK HTTP test seam are closed. |
| File cohesion and naming quality | **PASS** | Every production/proof owner has an exact row; split operation paths, fixture-only harness, package docs, and GET initialism resolve the prior findings. |

All three required lenses PASS the same repaired revision. The remaining review
is the broader Technical Design integration review; it consumes these receipts
and must not repeat their lane scope.

### Technical Design integration review — first fixed candidate

Candidate SHA-256:
`7d3af71e345e26efc65cadaad2f109e582da5275093bb8ade6a2929929e8fd98`.
The independent reviewer returned **FAIL** on one System / Integration Design
edge: D7 let caller cancellation/deadline outrank `outcome_unknown` after a
PutObject, CompleteMultipartUpload, or Delete request could have been sent. That
contradicted R9 and could authorize a duplicate business mutation.

D7 is repaired without changing Specification: definitive local/provider
evidence remains first; possibly-sent mutation ambiguity now outranks the local
context error; cancellation/deadline applies only to non-mutating work,
pre-completion multipart stages, or proved non-transmission. Safe diagnostics
may still record that the context ended the wait. Only this precedence and its
repair-induced regressions require focused integration re-review.

### Technical Design integration review — focused repaired candidate

Candidate SHA-256:
`1b95540d558d9a4e450af4e36a521f8b7d66e7ab76d053c4d38a05c55f40b903`.
The independent reviewer returned **PASS**. Definitive local/provider evidence
remains first; possibly-sent PutObject, CompleteMultipartUpload, and Delete map
to `outcome_unknown` even when cancellation/deadline ended the wait; context
kinds remain limited to non-mutating, pre-completion multipart, and
proved-unsent paths. No repair-induced regression survived.

The artifact is ready. Changes after the reviewed candidate are receipt and
status closure only; they do not alter a design decision, owner, flow, or reopen
condition.

### Go Ownership focused current-tree owner-map repair

Planning's current-tree link-check reopened only the inverse owner map: two
named config-example paths did not exist, `env/docker-compose.yml` is
Postgres-only, the canonical config-source policy was omitted, and the CI scope
classifier would treat the profile-owned conformance test as
template-independent. The repaired candidate SHA-256 is
`b216972ea49fc5716be8c29f9658f7a9da230860bdf834e2c68476504c65415a`.

The required focused Go Ownership panel returned **PASS** in all three lenses
on that exact candidate: responsibility/execution-path ownership,
package/dependency/generated authority, and file cohesion/naming. The repaired
map now assigns `env/.env.example`, `env/config/local.yaml`,
`docs/configuration-source-policy.md`, and `scripts/ci/ci-change-scope.sh` to
their current responsibilities and excludes the Postgres-only Compose file.
No behavior, system mechanism, dependency pin, proof oracle, provider claim, or
reopen condition changed, so the prior Technical Design integration PASS and
unaffected Go Ownership panel receipts remain current.

### D4 retained-memory reopen — Go Ownership panel

The first D4 candidate, SHA-256
`c0d220019480c53b72c6fd07abe973f7d49a0ba5d6511cdff8a194a931d0431d`,
returned **FAIL** in all three required read-only lenses. Although its source
trace proved successful `UploadPart` completion values were header-owned, it
used `B=max(H,E)` for retained ETags and Complete XML. That gave the same value
two bound authorities, fabricated an unreachable `E`-sized success field in
proof, and could multiply an unrelated control-body cap across 10,000 parts.

The smallest repair removed `B`. `H` now exclusively owns retained ETags and
their escaped XML representation; CRC64NVME text remains fixed at twelve bytes;
`E` exclusively owns control/error body state and the retained upload ID.
TD-006 varies `H<E`, `H=E`, and `H>E` to prove those effects remain independent.

Fresh focused review of repaired design candidate SHA-256
`c27282a190ff7f7500f3993dd4106d73831b3d9abfe8095ee8e1133f2dff8d3d`
and paired test-plan candidate SHA-256
`528cd53c1e6680b3c9c6a0fc21780fb215970c2f931ff6b7cc895e9d8d236ddc`
returned **PASS** in all three lenses:

| Required lens | Result | Receipt |
| --- | --- | --- |
| Responsibility and execution path | **PASS** | Startup arithmetic, content-type rejection, retained descriptors/upload ID, Complete XML, source receipt, and staged Linux proof each have one semantic owner; no repair regression survived. |
| Package and dependency architecture | **PASS** | Arithmetic remains adapter-config-owned, retention remains upload-owned, bootstrap remains the only composition point, and pinned AWS/Smithy generated authority introduces no new package or export seam. |
| File cohesion and naming | **PASS** | `config.go`/`config_test.go`, `upload.go`/`upload_test.go`, `client_test.go`, fixture-only `harness_test.go`, and the Make entrypoint retain one coherent present responsibility each. |

The broader Technical Design review remains required for D4 numeric/source
sufficiency, performance amplification, cross-flow coherence, and Linux proof
feasibility; it consumes this panel receipt and does not repeat its lanes.

### D4 retained-memory reopen — Technical Design integration review

Independent review of candidate SHA-256
`808a51e1db3498301e861e2364a35e731f8d9ac8ca635c23c077c14ce5dc70a3`
returned **FAIL** on two D4-owned edges. `F`, `S`, and `U` were fixed numbers
without a fixed allocation classification or minimum headroom rule, leaving T9
to choose the derivation. The Linux oracle also combined RSS with Go total
mapped memory and required an undefined return-to-idle memory condition even
though Go may retain free pages.

D4 repaired both edges: seven allocation classes now give every fixed and
variable owner one charge and variable driver; a symbolic receipt must retain
at least 100% headroom and cannot tune or reclassify the design ceilings; and
only the preconstruction-to-peak `smaps_rollup:Rss` delta is the memory oracle.
Cleanup now uses exact owned-resource counters and idle-baseline goroutine
comparison independently of page reclamation.

Focused review of that repair returned **FAIL** only because controller/TLS
fixture memory could still be read as part of the measured child. The final
repair fixed an unmeasured controller/fixture PID, an adapter/SDK/TLS-client-only
measured PID, payload-free fixed-size scalar IPC, child-only RSS sampling, and
separate cleanup/join oracles.

Fresh focused review of final design candidate SHA-256
`bc36b137d080f7f72bcd6a1711abe96035d59a929e3a4b0b69e9833966364563`
and paired test-plan candidate SHA-256
`4a2f564a9b113acb37b63738e3174e03ccee7089cf9fc46ab4de2b57deb310ae`
returned **PASS**. No numeric/source, amplification, cross-flow, process
attribution, cleanup, or proof-feasibility finding survives. Changes after the
reviewed design candidate are review/status receipts and authority-hash refresh
only. Test Design later made its own QA-owned `smaps_rollup` consistency oracle
exact; that changed no D4 decision, formula, allocation class, owner map, or
reopen condition. The numeric/process repairs likewise added no package, file,
dependency, composition, or proof owner, so the current three-lens Go Ownership
PASS remains applicable.

### D4 system-root trust-pool reopen — Go Ownership panel

The three fresh read-only lenses reviewed fixed candidate SHA-256
`1c19a55ae6439357238bf5513fdcde69b6e151cf1e0e6ea9cdbe3d14ee7852e6`.
Responsibility/execution path and package/dependency architecture returned
**PASS**: S3 alone selects, loads, retains, and accounts for the explicit pool;
`httpclient` only applies a caller-owned non-nil pool while nil preserves other
consumers; bootstrap only composes and closes; Delivery owns image provenance
and mount immutability; and the import/profile/dependency directions remain
closed. File cohesion returned **FAIL** only because `ca_bundle.go` and
“private CA pool” obscured the fixed-image public-root policy and could invite
custom-CA behavior.

The focused repair renamed that owner pair to
`image_root_bundle.go`/`image_root_bundle_test.go` and consistently names the
retained value an adapter-private public-root pool; the only remaining
“private CA” phrase prohibits that production class. Fresh focused review of
repaired candidate SHA-256
`098cef8f9b391bdba452978cc286346b84c2b2595c847c6f12f2b994896d616a`
returned **PASS** with no regression. The two unaffected PASS receipts remain
valid, so all three Go Ownership lenses approve the repaired fixed mechanism.
The broader Technical Design integration review remains required for trust,
numeric/source, process-envelope, rollout, and downstream-reopen coherence; it
consumes this panel receipt rather than repeating those ownership lenses.

### D4 system-root trust-pool reopen — Technical Design integration review

The fresh independent reviewer consumed the current Go Ownership receipts and
returned **PASS** on fixed candidate SHA-256
`18f768f32b597a6667f56eaf28b92c402a65f674501aa9d17a8b401ef6f39748`.
The nine pinned Go source identities and explicit-root non-reachability of the
process-global pool were exact; the fixed image-owned public bundle was the
smallest behaviorally equivalent mechanism; `B/N`, trust formulas, image facts,
100% source headroom, checked process equation, and three measured barriers
were coherent; hostile ambient-root denial and PID isolation closed fixture
contamination; rotation/rollback preserved provider neutrality; and the exact
Test Design reopen preserved Specification and phase boundaries. No trust,
performance/capacity, source, process-envelope, rollout, cross-flow, or
proof-feasibility finding survives.

The reviewer did not execute the downstream Linux envelope, deployed-image
mount receipt, or Amazon/R2 conformance. Those are deliberately Test Design,
Delivery, and provider-proof obligations rather than evidence claimed by this
Technical Design result. Changes after the reviewed candidate are this receipt
and final artifact-hash reporting only; no design decision, formula, owner,
proof input, or reopen route changed.

### D4 runtime-bundle identity/mount-policy reopen — Technical Design integration review

Independent review of fixed design candidate SHA-256
`510c2e7e9ef38e5cf743c0b26c67725afefebcb4115b070c5c54980e6d104f71`
returned **PASS**. The final Dockerfile stage and its platform-resolved manifest
now supply the only T9A image-root identity; the Go compiler/test image cannot
set D4's bundle inputs. The source receipt compares the extracted final-image
file with the same strict loader seam, while Delivery retains the distinct
deployed-image/no-override-mount receipt. No downstream owner must choose a
root identity or mount policy, and no provider, Docker, deployment, or runtime
receipt was claimed by this review.

The next owner is the Ledger Orchestrator. It re-reads this design with the
canonical T9A blocker and, only after native resume/replacement routing, returns
the preserved T9A candidate to its complete source/allocation receipt and
acceptance proof. This review record changes no D4 decision, formula, owner,
test-plan obligation, task, or implementation candidate.

### D4 arm64 Go source-identity reopen — Technical Design integration review

Independent review of fixed design candidate SHA-256
`cf3757a9662eb81d261f5e76c8d5b98f761157024d9ad3d0662d22e07ad6bc6b`
returned **PASS**. The Dockerfile build-stage Go index resolves for
`linux/arm64/v8` to `sha256:7939e2c75db3d059fc944bb6464a916d0fa64bd5a3bd7b3528f2a1ac7673a0eb`;
the repaired source-receipt input now carries that exact identity. Final-image
bundle `b`/`n`, D4 accounting and headroom, and Delivery's deployed-image
no-override receipt remain unchanged. No provider, deployment, certification,
test-plan, task, or T9A candidate claim changed.

The next owner is the Ledger Orchestrator, which routes a fresh T9A Lead from
the preserved candidate to update and rerun the source receipt. This D4 review
does not resume or replace T9A.

### Template certification-boundary reopen — Technical Design review

The Technical Design self-review refreshed the Specification authority to
`8b4000e19c3bd560809ef5beecf8a313582b7cfdbf65c3748822c08d42e0e0ca` and
reconciled D9 with the existing deterministic profile/configuration/bootstrap
seams, TD-016, TD-017, TD-018, and T1-T12 boundaries. The only repair is the
completion disposition: T1-T10 close the credential-free template capability;
Amazon and R2 remain individually authorized adopter certifications. No D1-D9
mechanism, D4 constraint, source owner, file map, task, test-plan scenario,
provider entrypoint, or no-provider-action policy changed.

An independent Technical Design review of candidate SHA-256
`0b226a25cc7af4e2c9ba41c6195addfa9fa3a51453a96fa9c99f196d829b42a0`
returned **PASS**. It found no unsupported flow, authority, ownership, or proof
edge: the fixed static tuple and local-only bootstrap remain intact; one
provider receipt cannot establish the other; and TD-016/T1-T10 already own
template closure while TD-017/T11 and TD-018/T12 own their separate authorized
provider evidence. The review executed no provider action, credentials,
deployment, purchase, publication, or test.

**Disposition:** Technical Design is ready. The existing Go Ownership panel
remains applicable because no Go responsibility, package, file, dependency, or
generated/manual boundary changed. Test Design and Planning are not invalidated:
their current proof and ledger boundaries already express this separation. The
next owner is the Ledger Orchestrator; it re-reads the review-cleared design and
resumes only the preserved implementation route that its canonical state makes
ready. This reopen neither authorizes nor schedules provider certification.
