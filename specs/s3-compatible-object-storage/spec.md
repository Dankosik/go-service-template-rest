# Feature code can use one bounded S3-compatible object store without owning provider mechanics

status: ready

Problem: the template has no object-storage capability, so each feature that
needs objects must currently choose an SDK and independently solve credential
selection, endpoint trust, streaming limits, multipart cleanup, integrity,
retries, errors, presigning, telemetry, and profile pruning. A broad
"S3-compatible" label would make that duplication worse by projecting Amazon
S3 behavior onto providers that implement different operation and field
subsets.

This specification turns the accepted outcome and the
[research synthesis](research/synthesis.md) into one behavioral contract. The
research is evidence, not an approved design. SDK selection, package placement,
transport composition, concrete configuration types, and test implementation
remain Technical Design and later-phase decisions.

## Scope and non-goals

In scope:

- an optional template-init profile, provisionally
  `OBJECT_STORAGE=none|s3`, defaulting to `none`;
- one process-wide adapter bound at startup to one exact provider, endpoint,
  region, bucket, address style, trust class, and credential mode;
- a small provider-neutral feature port for mediated upload, download,
  metadata, delete, and presigned read access;
- finite object, operation, concurrency, working-memory, multipart, deadline,
  and presign-expiry bounds supplied through validated runtime configuration;
- local, credential-free preparation for an adopter to certify an Amazon S3
  general-purpose bucket or Cloudflare R2 tuple as the initial shared-provider
  hypothesis; and
- profile generation, dependency pruning, readiness, cleanup, and bounded
  operator evidence.

The port deliberately does not own or expose:

- object-key construction, namespaces, tenant partitioning, business
  authorization, retention intent, content type policy, malware/media
  validation, or a domain digest; those remain feature decisions;
- bucket creation, provider account selection, identity grants or rotation,
  network egress, DNS, TLS termination, at-rest encryption, KMS, versioning,
  Object Lock, legal hold, replication, lifecycle rules, provider access logs,
  quotas, or alerts; those remain deployment controls;
- list, copy, range read, batch delete, tagging, ACL, storage-class, cache
  control, user metadata, public-bucket, custom-domain, access-point, directory
  bucket, Outposts, acceleration, dual-stack, FIPS, requester-pays, SSE,
  SSE-C, KMS, client-side encryption, version-ID, or bucket-control operations;
- presigned upload, HEAD, DELETE, or POST. Mediated upload supplies the bounded
  write path; metadata and delete already have bounded port operations;
- a public REST/OpenAPI operation, a new binary, a durable transfer queue, a
  reconciliation store, a bucket provisioner, or a provider failover router;
- a generic blob abstraction or multiple adapters. A second provider-specific
  port is a failed shared-pack hypothesis, not an extension seam hidden in this
  contract.

No rule below promises Amazon S3 consistency, IAM, encryption, checksum,
versioning, delete, ETag, error, or presign-policy behavior for Cloudflare R2.
Each provider earns the common outcome from its own current contract and exact
conformance evidence.

## Behavior and contract delta

### R1 — Provider support is an exact adopter certification claim

The only initial adopter certification targets are exactly this pair:

| Provider target | Required target form | Credential modes | Provider-specific fixed value |
| --- | --- | --- | --- |
| Amazon S3 | Commercial-partition regional endpoint and a general-purpose bucket with versioning never enabled | `static` | the signing region equals the configured bucket region |
| Cloudflare R2 | Account S3 API endpoint and one R2 bucket | `static` | signing region is exactly `auto` |

An adopter may claim a target is "supported" only when that exact tuple of
provider target, endpoint, region,
bucket, virtual-hosted authority, credential mode, checksum path, bucket
versioning state, and identity policy passes success criterion 3's conformance
obligations. The template's local, credential-free completion does not earn
either provider claim. A provider brand, successful SDK construction, emulator
result, or marketing compatibility statement is not support. No other provider
is implied.

The common wire subset contains only:

| Port behavior | Provider operation/fields admitted by the common subset | Feature-visible result |
| --- | --- | --- |
| Upload at or below the multipart threshold | `PutObject`: bucket, validated common key, declared content length, optional content type, selected checksum fields, and `If-None-Match: *` only for create-only | completion, or one closed error kind |
| Upload above the threshold | `CreateMultipartUpload`, serial `UploadPart`, `CompleteMultipartUpload`, `AbortMultipartUpload`, and cleanup-only `ListParts`; bucket, validated common key, declared total size, optional content type, selected checksum fields, upload ID, part number, and internal part ETag | completion, or one closed error kind plus cleanup disposition |
| Download | whole-object `GetObject`: bucket, validated common key, checksum-validation request fields | a bounded body stream plus size, optional content type, and last-modified time |
| Metadata | `HeadObject`: bucket and validated common key | size, optional content type, and last-modified time |
| Delete | unversioned `DeleteObject`: bucket and validated common key | provider-operation completion only |
| Presigned access | SigV4 presign of whole-object `GetObject`: bucket, validated common key, expiry, and signer-selected headers | exact `GET` method, bearer URL, required headers, and signature expiry |

The feature surface exposes no bucket, endpoint, region, provider request type,
HTTP status, provider code, upload ID, part ETag, object ETag, version ID,
checksum algorithm, credential, or SDK type. Size is bytes. Last-modified is a
UTC instant supplied by the provider and is not a business clock or ordering
token. Content type is optional and untrusted; the feature that consumes it
still owns content policy.

The adapter sends only fields named above. Adding a provider-only request or
response field, or requiring a provider-specific escape from the feature port,
reopens R1. Amazon and R2 may use different hidden checksum algorithms or
client mechanics only when both produce the same feature-visible integrity
outcome in R5 and the exact choice is recorded by conformance.

**Falsifier.** The hypothesis fails if either initial provider cannot pass all
five port behaviors with the same feature request/result vocabulary, if a
required provider-specific field reaches feature code, or if an adapter can
claim support without identifying the exact conformance tuple.

### R2 — The feature owns every object decision

Every call carries one feature-produced key in one closed common domain:

- 1 through 1,024 ASCII bytes;
- only letters, digits, `-`, `_`, `.`, `~`, and `/`;
- no leading or trailing `/`, no empty slash-delimited segment, and no segment
  equal to `.` or `..`;
- the complete key is not the case-sensitive value `soap`.

Anything else returns `invalid` before signing or provider I/O. Inside this
domain the adapter preserves the exact bytes and segment boundaries; it does
not Unicode-normalize, case-fold, path-clean, add prefixes, infer tenants,
choose names, or accept a bucket per call. The ASCII restriction prevents
Cloudflare R2's documented NFC normalization from collapsing two feature keys;
the `soap` exclusion closes Amazon S3's documented virtual-hosted exception.
Wire escaping is transport representation only and must decode to the same
accepted bytes at both providers.

Before calling the port, the feature authorizes the acting principal for the
specific operation and key and applies its own content, size, retention, and
overwrite policy. The port neither accepts a principal nor caches an
authorization decision. Presign issuance follows the same rule immediately
before generation because the returned URL delegates the signer identity's
permission.

The pack may reject a request that exceeds its configured runtime ceiling; that
ceiling does not authorize an otherwise disallowed feature request. Feature
limits may be lower. Deployment bucket and identity policy remain defense in
depth and do not replace feature authorization.

**Falsifier.** Cross-feature or cross-tenant attempts using a caller-chosen key
fail at the feature enforcement point without invoking the adapter. ASCII keys
that differ inside the accepted grammar remain different objects on both exact
providers; Unicode, `soap`, dot-segment, empty-segment, and overlong cases fail
before I/O. The same adapter accepts two independently valid feature key
schemes inside the grammar without knowing either scheme.

### R3 — Startup fixes one credential and authority boundary

The selected pack has no usable provider defaults. Startup requires explicit
provider, endpoint, signing region, bucket, credential mode, maximum object
bytes, global active-operation ceiling, maximum adapter working-memory bytes,
multipart threshold, maximum operation duration, and maximum presign expiry.
Every bound is positive and finite; the threshold is within the object ceiling
and provider multipart constraints. Missing, empty, inconsistent, or
unprovable values fail startup before network I/O.

The only initial credential mode is `static`: an explicitly injected
access-key identifier and secret, plus an optional session token. Secret values
come only from the repository's `APP__...` environment channel, never
non-empty YAML. The immutable process snapshot neither discovers nor refreshes
credentials. Rotation, replacement of an expired session credential, or
recovery from revocation requires a new process with a new validated snapshot.
An R2 temporary access key is still explicit static input with its external
expiry owner.

There is no anonymous, default-chain, shared-file, named-profile, web-identity,
container-task, instance-metadata, or fallback mode. AWS workload identity
reopens Specification because STS, ECS task credentials, and EC2 instance
metadata each introduce a distinct authority, trust class, input, refresh,
deadline, and failure contract that the single S3 authority cannot decide.

Ambient SDK variables or files must not change endpoint, region, address style,
retry, checksum, profile, or credential behavior. Every ambient value capable
of changing the signed target, credential, or operation policy is ignored or
rejected visibly at startup.

The endpoint is an absolute HTTPS origin with no user info, path, query, or
fragment. Address style is fixed to virtual-hosted. The bucket must be valid for
that style and contain no dot, so the only request authority is the validated
`<bucket>.<endpoint-host>` authority. TLS validates that host against normal
trusted roots. Redirects, proxies, region correction, alternate endpoints, and
authority rewriting are refused. DNS resolution must remain in the external
public-address class on every dial; deployment separately enforces network
egress.

Amazon access points, directory buckets, Outposts, accelerated, dual-stack,
FIPS, China/GovCloud partition endpoints, and R2 custom domains are outside the
initial target because they change authority or field semantics. A local
emulator may use a test-only authority and trust class, but that evidence never
qualifies an external provider tuple or creates a production plaintext mode.

**Falsifier.** Startup or a dial refuses a mismatched provider/region, dotted or
invalid bucket, changed authority, redirect, proxy route, private or loopback
resolution, invalid certificate, unselected credential source, or ambient SDK
override before signed identity material reaches that target.

### R4 — Admission, streaming, and deadlines are process-wide

All five port operations share one adapter-wide, non-blocking admission
ceiling. At capacity a new call returns `busy` before body read, signing, or
provider I/O; the adapter owns no waiting queue. An
upload holds admission until its provider result and required cleanup
disposition are known. A download holds admission until its body reaches EOF or
is closed. Metadata, delete, and presign hold it until return.

At most one provider request is in flight for one admitted operation. Multipart
parts are uploaded serially. Cleanup replaces, rather than runs beside, the
failed transfer request. Therefore provider-request concurrency never exceeds
the configured active-operation ceiling. Static credential use creates no
secondary credential-network request or refresh worker.

The adapter streams mediated bodies and never buffers an entire object. The
implementation candidate must prove that the worst-case retained working set
for all admitted operations, including SDK buffers, signing, checksums,
multipart parts, and download validation, fits the configured adapter
working-memory ceiling. Startup fails if that calculation cannot be made from
the selected client behavior and configuration. Object bytes stored outside
the process are not counted as process memory.

Every call receives a caller context. Its effective deadline is the earlier of
the caller deadline and start time plus the configured maximum operation
duration. Admission, DNS, connect, TLS, request write, provider response, body
read, multipart steps, and any
cleanup attempt spend that single budget. The adapter never roots work in a
fresh longer context. Cancellation or expiry stops new parts and new attempts;
it cannot prove that already transmitted bytes or a provider commit did not
happen.

Upload requires an exact declared content length no greater than the object
ceiling before reading the source. The adapter sends exactly that many bytes;
a short source is failure, and bytes after the declared length remain
caller-owned and unsent. The caller owns and closes the upload source.

Download rejects a declared provider size above the object ceiling before
returning a body. If size is absent or false, the returned stream delivers no
byte beyond the ceiling and then fails `too_large`. The feature owns closing the
body on every path. Receiving headers is not download completion: only EOF with
no deferred read or integrity error is success. Closing early releases
admission but proves no complete download. Reads after cancellation or deadline
fail with the matching stable error.

**Falsifier.** Concurrent calls cannot create more provider requests or
retained adapter memory than the configured ceilings; a saturated call causes
no effect; a leaking download consumes one visible slot rather than escaping
the bound; cancellation reaches I/O, body read, and multipart progress.

### R5 — Upload intent, multipart, and checksum outcomes are explicit

An upload chooses exactly one feature-owned intent:

- `create_only`: at or below the multipart threshold, the adapter sends
  `If-None-Match: *`. Existing or concurrently created content returns
  `precondition_failed` without overwrite. `create_only` above the threshold is
  rejected as `invalid` before reading the body because Cloudflare R2 does not
  currently document the completion-time conditional needed for an atomic
  multipart create.
- `replace`: the adapter may replace the current unversioned object. A
  concurrent writer may win before or after it; the pack provides no
  compare-and-swap, ordering, or last-writer identity.

An object at or below the configured threshold uses one `PutObject`; an object
above it uses multipart. The selected client must not silently choose a
different threshold. Multipart parts are serial, each part is within provider
constraints, and completion verifies the provider's final response rather than
treating the initial HTTP status alone as success.

On any multipart failure before confirmed completion, the adapter stops new
parts and attempts `AbortMultipartUpload` only within the operation's remaining
deadline. It returns one cleanup disposition:

- `complete`: the provider-specific bounded check proves no retained parts for
  that upload session;
- `pending`: abort or verification failed, timed out, was cancelled, or cannot
  rule out an in-flight/retained part.

The upload ID remains adapter diagnostic state and never reaches the feature.
The adapter starts no background reconciler. Deployment must configure an
exact-provider lifecycle rule for abandoned multipart uploads; that backstop
does not replace the immediate attempt. A `pending` disposition is observable
and is never reported as successful cleanup.

Every supported exact provider tuple selects and proves one checksum algorithm
and type for single uploads and one for multipart. Upload success means the
adapter calculated the checksum over the exact bytes sent and the provider
confirmed the matching accepted checksum path. Download success means the
provider supplied the accepted stored checksum and the adapter validated the
fully consumed body at EOF. An object without the accepted checksum, a mismatch,
or a deferred validation failure is `integrity_failed`, not a weaker success.

Checksum is transport/storage integrity evidence only. It is not a feature
digest, object identity, ETag, version, encryption, authenticity, or retention
proof. The port exposes none of those values. Presigned execution bypasses the
adapter and receives no adapter checksum-success claim.

**Falsifier.** Single and multipart corruption fail, partial download never
claims checksum success, the exact threshold selects one path, create-only
never overwrites, and every failed multipart upload yields either proven
`complete` or visible `pending` cleanup.

### R6 — Metadata and download expose only portable observations

Metadata and download return exactly the feature-visible fields in R1. Metadata
success proves only that the exact provider returned those headers for the
configured key at that instant. Download metadata describes the representation
whose body was opened; it does not make a later metadata call atomic with the
stream.

The pack makes no cross-provider promise about read-after-write visibility,
cache/custom-domain freshness, snapshots, object generations, or list
consistency. A feature requiring those semantics must name one exact provider
contract and reopen Specification rather than infer them from this port.

An unambiguous provider absence returns `not_found`. A response that may hide
absence because the identity lacks disclosure permission returns `denied`; it
does not guess that an object exists or does not exist. Exact-provider
conformance must exercise both outcomes with the deployment's intended
identity policy.

**Falsifier.** Feature code cannot use metadata as a version token or checksum,
and a permission-concealed missing object never becomes `not_found` without
provider/identity evidence.

### R7 — Delete confirms one operation, not erasure

Delete accepts only the feature-owned key and sends unversioned `DeleteObject`.
Provider success, including a provider no-op for an absent key, returns success.
It means only that this operation completed for the configured bucket. It does
not promise that bytes, old versions, replicas, caches, legal holds, backups,
or independently retained copies are gone.

The initial Amazon conformance bucket must never have enabled or suspended
versioning, because a versionless delete would otherwise create a delete marker
rather than satisfy the common operation meaning. R2's lack of S3 versioning is
provider-specific evidence, not a generic compatible-provider guarantee.
Retention, legal hold, and lifecycle policy can still deny deletion; that is
`denied` and remains deployment/feature policy.

**Falsifier.** A versioning-enabled or previously versioned Amazon bucket
cannot be certified for this pack, and a successful delete receipt cannot be
used as proof of historical or physical erasure.

### R8 — Presigned access is read-only, bounded issuance of a bearer URL

The only presigned operation is whole-object `GET`. The feature authorizes the
recipient, operation, and key immediately before issuance and requests an
expiry in whole seconds. The request must be at least one second and no greater
than both the configured deployment maximum and the SigV4 ceiling of seven
days; there is no default TTL.

The result contains the exact method `GET`, query-bearing URL, exact headers the
recipient must send, and the signature expiry instant. The URL is tied to the
configured key and fixed virtual-hosted authority. R2 custom domains are not
eligible. A changed method, key, query value, authority, or signed header fails
provider signature validation.

The URL is a reusable bearer credential until the provider stops accepting it.
It has no one-time-use or revocation guarantee and may become unusable before
the returned signature expiry when its signing credential expires, is revoked,
or loses permission. A transfer that begins before expiry may continue after
it. The pack does not promise a minimum usable lifetime.

The pack never logs, traces, metric-labels, or persists the URL, query, signed
headers, or credential. The feature may transmit the result only to the
authorized recipient and must not persist it. Issuing a replacement creates a
second independently reusable bearer URL; it does not revoke the first.

Presign generation is bounded by R4, but the later client-to-provider transfer
bypasses the process. The service cannot enforce body bytes, concurrency,
deadline, checksum consumption, reuse count, completion, or recipient identity
on that transfer. The pack therefore makes no hard size guarantee for presigned
GET and offers no presigned PUT. A feature that requires service-enforced size,
integrity, completion, or one-time access uses the mediated path or reopens the
scope with a provider/deployment control that proves it.

**Falsifier.** Presign output contains only GET for one key and fixed authority;
secret-bearing values are absent from all service evidence; repeated use before
provider expiry is treated as expected; issuance telemetry cannot claim the
later transfer succeeded.

### R9 — Errors and retries do not invent provider guarantees

Feature code receives one closed error kind:

| Kind | Exact meaning |
| --- | --- |
| `invalid` | The request is empty, malformed, unsupported, or inconsistent before provider I/O. |
| `too_large` | Declared or observed object bytes exceed the configured ceiling. |
| `busy` | Adapter-wide admission refused the call before any effect. |
| `not_found` | Exact provider/identity evidence unambiguously proves current absence for this read. |
| `precondition_failed` | Create-only observed an existing or concurrent target; no overwrite is claimed. |
| `denied` | Provider policy denied the operation, including a response that deliberately conceals absence. It does not distinguish authentication from authorization. |
| `temporary` | A bounded non-mutating operation failed in a way exact provider evidence permits the feature to try again under a fresh business decision and deadline. |
| `integrity_failed` | Required upload or fully consumed download checksum evidence is missing or mismatched. |
| `cancelled` | Caller cancellation won before a more specific outcome was known. |
| `deadline_exceeded` | The effective operation deadline expired before a more specific outcome was known. |
| `outcome_unknown` | Upload, multipart completion, or delete may have committed but no authoritative result is known; cleanup disposition is included when multipart was involved. |
| `internal` | A sanitized unclassified adapter/client failure; no provider meaning is inferred. |

No provider message, object key, bucket, endpoint, URL, query, credential
identifier, SDK type, HTTP status, provider code, ETag, version, checksum value,
or upload ID is feature-visible. The adapter may retain sanitized operation,
phase, status, provider code, and request ID as internal diagnostic evidence.

The pack performs no automatic storage-operation retry. SDK/client and
`OUTBOUND_HTTP` operation retries are disabled, so one port call makes at most
one attempt at each required protocol stage. Metadata and pre-body download may
return `temporary`; deferred download reads are never silently restarted. A
mutating transmission failure is
`outcome_unknown` unless exact evidence proves the effect was not sent.

Feature code may make a new call only under its own business semantics. The
pack never describes upload or delete as generically safe to retry, never turns
throttling alone into retry permission, and never converts absence after an
ambiguous write into proof that the write did not happen.

**Falsifier.** SDK defaults cannot produce a second operation attempt; an
after-possible-send loss is not `temporary`; 403/404 differences do not invent
absence; every feature-visible error is sanitized and belongs to the closed
table.

### R10 — Invalid configuration blocks startup; provider availability does not block readiness

Selecting the pack makes configuration, authority, credential,
resource-bound, and client construction validation startup-critical. Failure
there prevents the service from becoming ready because the selected capability
does not exist.

After successful construction, object-provider availability is not a startup
or readiness dependency. The shared pack registers no `HeadBucket`, list,
write, or per-request probe. Runtime storage failure is returned by the
operation and reported through R11; liveness remains process-only. A feature
whose accepted service outcome cannot operate without storage must reopen
Specification and define that feature's degradation/readiness policy rather
than switch on a generic bucket probe.

The adapter owns no durable background work. Shutdown closes its idle transport
inside the existing dependency-close stage and deadline. An active request or
download follows the service's existing drain and caller-context behavior;
shutdown does not extend its deadline or claim remote rollback. Multipart
`pending` cleanup remains for the deployment lifecycle rule.

**Falsifier.** A provider outage after startup does not evict every instance or
restart the process, invalid selected configuration never serves ready, and
shutdown leaves no adapter goroutine or unclosed idle transport while making no
claim about unresolved remote parts.

### R11 — Telemetry answers bounded operator questions without disclosing object identity

The adapter must let an operator answer:

- which of the five operations is succeeding, failing by stable error kind, or
  saturating, and how long admitted calls take;
- how many operations are active, admitted, and rejected against the configured
  ceiling;
- how many mediated bytes complete upload/download and how often integrity
  fails;
- whether an upload used single or multipart and whether multipart cleanup was
  `complete` or `pending`;
- whether presign generation succeeded, while explicitly not observing the
  direct transfer;
- whether credential rejection, endpoint trust, throttling, or the provider
  response phase is the current failure boundary.

Metric dimensions are closed, low-cardinality vocabularies: operation, stable
result kind, transfer path, and cleanup disposition. Provider, endpoint,
bucket, key, content type, account, tenant, user, credential mode, request ID,
provider code/message, status text, URL, and checksum are not metric labels.
The label product must remain within the repository's per-instrument
cardinality limit, with unknown values collapsed to one bounded fallback.

Structured logs and traces may carry operation, stable result, phase, attempt
count, byte count, duration, active count, cleanup disposition, and a sanitized
provider request ID as a non-metric diagnostic field. They never carry key,
bucket, endpoint, body, user metadata, content type, checksum value, access-key
identifier, credential material, authorization/signing headers, provider error
text, presigned URL/query, or returned signed headers. Existing context logging
supplies request and trace correlation; the adapter does not duplicate it.
Wire/body/signing debug logging is off and cannot be enabled by ambient SDK
configuration.

Deployment-owned provider access logs or audit signals are the only evidence
for presigned execution. Service telemetry describes issuance only and cannot
close direct-transfer completion, size, integrity, or recipient claims.

**Falsifier.** A corpus containing secret-like credentials, keys, URLs,
metadata, checksum values, and provider messages appears nowhere in emitted
logs, spans, metrics, or errors, while every closed result and cleanup outcome
still has a bounded diagnostic signal.

### R12 — The profile is orthogonal, deterministic, and dependency-clean

`OBJECT_STORAGE` accepts exactly `none` and `s3`; absent means `none`, while an
explicit empty or unknown value fails without mutation. Initialization records
the resolved value in `template.lock`, prints it in completion output, strips
all object-storage profile markers, removes generator-only profile sources,
and remains byte-idempotent for an identical repeat. A repeat with another
value fails unchanged.

With `OBJECT_STORAGE=none`:

- every object port, S3 adapter, object-specific bootstrap/config/runtimeopts,
  operator document, example, fixture, test, and profile CI lane is absent;
- object-storage config inputs are unknown keys, not ignored disabled-feature
  settings;
- `go mod tidy` removes every S3/Smithy/compatible-client runtime dependency
  that no independently selected profile owns.

With `OBJECT_STORAGE=s3`:

- the provider-neutral feature contract, one S3 adapter, typed fail-closed
  configuration, bootstrap lifecycle, operator contract, deterministic fault
  proof, exact-provider conformance entrypoints, and one CI profile lane remain;
- no public API, bucket provisioner, local live credential, resolvable bucket,
  or externally usable endpoint is generated by default;
- generated non-secret config may show empty placeholders and finite-bound
  requirements; secrets appear only as empty environment examples;
- the one client family later selected by Technical Design remains in
  `go.mod`/`go.sum`, and the rejected client family does not.

`OBJECT_STORAGE` and `OUTBOUND_HTTP` are independent selectors. All four
combinations are valid. Selecting object storage neither selects nor requires
`OUTBOUND_HTTP=bounded`; selecting bounded outbound HTTP does not create object
storage. With both enabled, each keeps its feature-facing contract and there is
one owner for retries and one fixed authority per dependency. Shared lower-level
transport/trust code may be chosen in Technical Design only if removing either
profile still leaves the other generated service compiling and retaining only
the dependencies and user-visible configuration it owns.

The template-init oracle proves selected and absent paths from one independent
inventory, marker removal, lock/completion values, unknown-key behavior,
idempotent equal repeats, unchanged incompatible repeats, all four selector
combinations, and dependency pruning after `go mod tidy`.

**Falsifier.** Any `OBJECT_STORAGE=none` output retains S3 code/config/deps, any
selected output requires a source edit or live default, or either
`OUTBOUND_HTTP` choice silently changes object behavior.

### R13 — Template completion is local; provider certification is adopter-owned

The reusable template is complete for this capability when its selected `s3`
profile can be generated, configured fail-closed, constructed locally, and
validated by the credential-free deterministic evidence in success criteria
1, 2, 4, and 5. It must give an adopter the profile selector, empty examples,
typed required inputs, operator documentation, and fail-closed conformance
entrypoints without requiring a live endpoint, bucket, credential, provider
request, provider mutation, deployment, purchase, or publication.

An adopter that chooses to claim Amazon S3 or Cloudflare R2 support owns one
separate exact-provider tuple certification. That adopter supplies and
authorizes its account, bucket, identity, endpoint, region, lifecycle and
versioning evidence, runtime-image/no-override evidence, and prefix-scoped
test mutation. Until that provider's own receipt exists, its support claim is
`unverified`. An Amazon receipt never certifies R2, an R2 receipt never
certifies Amazon, and neither receipt certifies a deployed path or another
adopter tuple.

The template makes no provider-support claim merely because its local profile
or credentialed entrypoint exists. Missing or unauthorized provider inputs do
not block template completion; they block only the matching adopter claim.
Any template path that requires a live tuple for generation, startup
construction, deterministic proof, or local completion reopens this rule.

**Falsifier.** A selected generated service cannot be locally configured and
fail closed without source edits, its deterministic profile proof contacts a
provider or needs credentials, or a provider claim is emitted without that
provider's own exact-tuple receipt.

## Invariants and edge cases

- One process owns one configured provider and bucket. Multi-bucket,
  per-tenant-provider, dynamic-endpoint, failover, or dual-write behavior
  reopens the trust and consistency contract.
- Feature authorization and key/content/retention policy always precede adapter
  work. Deployment controls always remain required defense in depth.
- The port provides whole-object operations only. No range, resume, list,
  pagination, batch, copy, or client-managed multipart state is implied.
- A successful upload is one confirmed provider operation with required
  checksum evidence; it is not durability, replication, consistency,
  encryption, retention, or business-publication proof.
- A successful download requires EOF; metadata receipt and stream acquisition
  are not completion.
- A successful delete is neither historical erasure nor retention-policy proof.
- Presigned access delegates the signer identity through a reusable bearer URL
  and bypasses process bounds after issuance.
- ETag and provider version/generation are neither exposed nor used as portable
  identity. Multipart part ETags remain internal protocol receipts only.
- No automatic storage retry, background cleanup worker, hidden provider
  fallback, ambient SDK policy, or second admission queue exists.
- Errors, telemetry, and conformance claims remain provider-evidence-bounded;
  unsupported provider behavior is a failed target, not `internal` success.

## Decisions, constraints, and authorities

- **D1 — Initial provider set: Amazon S3 plus Cloudflare R2.** Both publish
  current field-level contracts for the requested operation family, including
  conditional `PutObject`, whole-object reads/metadata/deletes, multipart, and
  presigned GET. Rejected: a generic provider registry, because a brand name or
  S3 label does not prove fields or outcomes. Adding another provider requires
  the same exact contract and conformance, not an adapter escape.
- **D2 — One small port and one adapter remain a hypothesis.** The five
  operations share one ownership and failure policy and need no provider type.
  Rejected: Go Cloud/Thanos-style abstraction now, because no second non-S3
  transport or list/copy surface exists and those abstractions may hide the
  required precondition, checksum, cleanup, and presign semantics.
- **D3 — No SDK is selected here.** Technical Design must compare the current
  AWS SDK for Go v2 family and a maintained purpose-built S3-compatible family
  against this fixed behavior. Direct SigV4/XML ownership and cross-provider
  abstraction remain rejected unless that comparison proves the requested
  contract smaller and safer through them.
- **D4 — Resource numbers are explicit deployment inputs, not invented
  template defaults.** Repository evidence supplies no representative object
  workload from which to choose bytes, concurrency, memory, threshold, or
  operation duration. Startup therefore requires finite values and validates
  their relations. Reopen defaults only from a named reusable workload and
  measured envelope.
- **D5 — Multipart is serial and presign is GET-only.** Serial parts make the
  global request bound independent of a client's per-call worker defaults.
  Mediated PUT already supplies bounded writes; portable presigned PUT cannot
  enforce the process body/concurrency contract. Reopen only from measured
  throughput need plus exact-provider size/integrity/cleanup proof.
- **D6 — No automatic operation retry.** Side-effect ambiguity and incompatible
  client defaults make a generic retry policy unsafe. Reopen per operation only
  with exact provider evidence, immutable replayable intent, one attempt owner,
  and a total budget that preserves the same feature-visible outcome.
- **D7 — Provider availability is non-critical after construction.** A generic
  bucket probe cannot decide feature degradation policy and can cause fleet
  eviction during a dependency outage. Reopen only for a feature whose accepted
  outcome makes storage readiness-critical and names a non-mutating capability
  that proves it.
- **D8 — Encryption is deployment evidence.** External TLS is required, but
  the pack sends no SSE/KMS/SSE-C fields and claims no at-rest or client-side
  encryption. Amazon and R2 provider-managed encryption statements remain
  provider-specific; deployment proves its chosen bucket/account policy.
- **D9 — Exact provider contracts are external authorities.** Amazon's
  [S3 API](https://docs.aws.amazon.com/AmazonS3/latest/API/Welcome.html) and
  [presigned URL contract](https://docs.aws.amazon.com/AmazonS3/latest/userguide/using-presigned-url.html)
  govern Amazon behavior; Cloudflare's current
  [R2 S3 API compatibility](https://developers.cloudflare.com/r2/api/s3/api/)
  and [presign](https://developers.cloudflare.com/r2/api/s3/presigned-urls/)
  matrices govern R2. The repository configuration
  source policy, fixed-authority egress policy, runtime budgets, readiness
  service, telemetry cardinality/privacy rules, architecture seams, and
  template-init contract remain local authorities.
- **D10 — Initial credentials are static-only.** The explicit snapshot keeps
  all signed requests inside the one accepted S3 authority boundary. Rejected:
  labeling AWS's default/workload chain as one mode, because web identity adds
  STS while ECS and EC2 add separate link-local metadata authorities with
  different trust, refresh, deadline, and failure semantics. Add one workload
  source only through a fresh Specification decision that closes those inputs.

## Success criteria and proof expectations

1. **Shared behavior.** The same feature fake and port contract exercises
   upload, download-through-EOF, metadata, delete, and presigned GET without an
   SDK/provider type or field. Authorization, key, retention, and content policy
   tests cause no storage call when they deny.
2. **Deterministic adapter boundaries.** Scripted transport/client fault proof
   covers authority refusal, admission, finite memory/body bounds, caller
   deadline/cancellation at every blocking phase, single/multipart threshold,
   serial parts, short body, oversized response, deferred body failure,
   checksum mismatch/missing checksum, create-only collision, no operation
   retry, after-possible-send ambiguity, abort `complete|pending`, body close,
   and telemetry redaction.
3. **Exact-provider conformance.** A real adopter may separately run
   credentialed tests against one Amazon S3 general-purpose bucket and one
   Cloudflare R2 bucket to prove the exact R1 operation/field matrix, static
   credential mode, endpoint/region, virtual-hosted authority, checksum choices
   for single/multipart/download, create-only precondition, unambiguous/hidden
   absence, delete on the accepted versioning state, multipart abort/list-parts
   cleanup, presigned GET method/headers/expiry/reuse, and sanitized provider
   evidence. Each receipt supports only its own tuple; neither is required for
   the template's local completion or covers the other.
4. **Process envelope.** Under the configured maximum admitted operations and
   object paths, observed provider request concurrency and retained adapter
   working memory do not exceed their declared ceilings; cancellation and
   saturation release every owned resource. A client whose peak or worker
   behavior cannot be bounded fails the hypothesis.
5. **Profile outputs.** Independent generator oracles prove every R12
   postcondition for `none`, `s3`, both `OUTBOUND_HTTP` choices, equal repeat,
   incompatible repeat, marker removal, unknown config keys, `template.lock`,
   completion output, compilation, and `go mod tidy` dependency closure.
6. **Layered claims.** A feature fake proves feature policy only; scripted
   faults prove adapter behavior only; an optional emulator proves only its
   exercised subset; exact-provider tests prove only their exact tuple; a
   deployed-path canary separately proves runtime identity, DNS/TLS/egress,
   bucket policy, lifecycle backstop, quotas, telemetry delivery, and
   deployment encryption controls. No lower layer is promoted to a higher
   claim.
7. **Template completion boundary.** The selected profile's local evidence
   proves adopter-ready setup and fail-closed behavior only. It records no
   provider support, deployment, or adopter-owned tuple result without the
   separately authorized receipt required by R13.

These are proof expectations, not a Test Design or command plan. Technical
Design must first produce one mechanism capable of them without changing their
meaning.

## Risks, assumptions, and reopen conditions

- **Shared-pack assumption.** Amazon S3 and R2 can satisfy the stable subset
  with one feature port and one adapter policy. Reopen owner: Specification.
  Reopen when exact conformance needs a provider escape, either provider cannot
  cover a requested operation/field, checksum/precondition/error behavior
  differs materially, or fewer than two reusable adopter/provider cases remain.
  If no smaller stable subset still covers all five requested operations,
  remove the shared pack.
- **Bounded-client assumption.** At least one live client family can expose or
  constrain every worker, buffer, retry, body, checksum, cancellation, and
  cleanup path strongly enough to prove R4/R5. Reopen owner: Technical Design
  first, then Specification if no behaviorally equivalent mechanism exists.
- **Versioning precondition.** The initial Amazon target has never enabled
  versioning. Reopen R7 for any versioned/suspended bucket or requirement to
  target an exact version; do not reinterpret versionless delete.
- **Create-only ceiling.** Atomic create-only is limited to single PUT until
  both provider contracts and exact conformance prove completion-time multipart
  `If-None-Match: *`. Reopen R5 on that evidence; a HEAD-then-write check is not
  sufficient.
- **Presign limitation.** GET issuance cannot observe or enforce direct
  transfer size, checksum consumption, completion, recipient, reuse, or
  revocation. Reopen only with an exact external control and keep its proof with
  deployment/feature ownership.
- **No numeric defaults.** A later adopter may show a reusable workload. Reopen
  D4 only with object-size distribution, concurrency, memory/cgroup, latency,
  and multipart evidence that supports defaults with headroom.
- **Workload identity is not silently deferred to design.** Reopen R3 only for
  one named AWS source after Specification fixes its secondary authority,
  allowed inputs, trust class, refresh/expiry/rotation semantics, deadline and
  cancellation behavior, startup/runtime failure outcomes, and proof boundary.
- **Provider drift.** Refresh the exact-provider matrix and conformance when a
  provider contract, selected client, checksum, retry, endpoint, credential,
  or presign behavior changes, or when deployment changes versioning, Object
  Lock, lifecycle, encryption, identity, CDN/custom-domain, or bucket policy.
- **Template/adopter boundary.** Template completion depends only on the local
  behavior named by R13. Reopen Specification if a template claim requires a
  live provider tuple, or if a local setup seam cannot remain credential-free
  and fail closed; reopen the matching adopter certification when its tuple,
  provider contract, identity policy, or provider receipt changes.

## Review disposition

Independent Specification review is required because this artifact fixes a
security-sensitive signed authority and credential boundary, a cross-provider
write/delete contract with ambiguous outcomes, process-wide resource limits,
and a template profile consumed by generated services.

1. Independent whole-artifact review — `FAIL`: the draft allowed a
   provider-dependent key domain despite R2 Unicode normalization and Amazon's
   virtual-hosted `soap` exception, and advertised AWS workload sources without
   defining their STS/ECS/IMDS trust boundaries.
2. Root repair — R2 now fixes one ASCII path grammar with exact rejection and
   preservation semantics; R1/R3 and every affected deadline, retry, telemetry,
   proof, and risk clause now make the initial credential mode static-only and
   give workload identity an explicit Specification reopen contract.
3. Focused fresh re-review — `CONCERNS`: the repair closed both original
   blockers, but R4 retained one impossible `credential wait` proof obligation.
4. Root repair removed that stale phrase. Final focused confirmation — `PASS`:
   no finding remains, and the prior unaffected whole-artifact conclusions
   remain valid.
5. Template-completion-boundary reopen — independent whole-artifact review of
   candidate SHA-256 `a5b9113281080cfa750de74f85b4a58bd6b8d36142f491108144e5be655219a2`
   — `PASS`: local selector, fail-closed examples, and deterministic profile
   proof establish adopter-ready template completion; T11 and T12 remain
   separate, optional provider receipts with no cross-provider substitution.
