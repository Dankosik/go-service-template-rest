# Test design — S3-compatible object storage

status: ready

Accepted authorities:

- [`spec.md`](spec.md), SHA-256 `46ee347a7931d9d20601ce00cea8455a89e2d9ba166dfa8e308b8a4154e16def`;
- [`design/overview.md`](design/overview.md), SHA-256 `b7b53857051b5c5dec35bc787782a461fb022a70d8dddbc12d599f71c702e496`.

[`research/synthesis.md`](research/synthesis.md), SHA-256
`d13fbc8ad407eaf89cf29653611d1efc0c5d30c489e97d5476e1328d909ed9a6`,
is supporting evidence only. This plan fixes proof, fixtures, and commands for
the accepted low-level AWS SDK for Go v2 mechanism. It adds no behavior,
production seam, Planning order, provider mutation, deployment action, or
support certification.

## Proof-boundary decisions

- Scripted adapter tests construct the real low-level `s3.Client` through its
  existing `HTTPClient` option. A mock S3 interface could false-pass retry,
  checksum framing, body, Smithy error, and multipart behavior, so none is
  added. The shared harness scripts requests and bodies only; assertions stay
  in the owning test files.
- `internal/infra/httpclient` uses one real loopback server for the accepted
  observable that a lost response creates exactly one fresh HTTP/1 connection
  and no transparent replay. Adapter tests independently count every S3 stage.
  Source-option assertions alone do not prove one attempt.
- Admission, phase blocking, cancellation, and release use owned channels and
  `testing/synctest` where the path is time-driven. Real-time bounds are outer
  hang diagnostics only; elapsed time is never a success oracle.
- Trust construction uses the production strict image-bundle loader through an
  unexported test source. Every successful S3 client receives a non-nil,
  adapter-private `RootCAs` pool; hostile `SSL_CERT_FILE`, `SSL_CERT_DIR`, and
  fallback-root canaries prove the S3 path cannot consult ambient roots. A
  one-root fixture is a TLS wiring check only and cannot prove the bounded
  production snapshot or retained-memory envelope.
- The memory claim has three complementary falsifiers: exact arithmetic and
  overflow tests at construction; a pinned-source reserve receipt that accounts
  for every SDK/Smithy/Go transport owner; and a Linux subprocess envelope that
  retains the production fixed-authority client while the real SDK uses a
  separate fresh HTTP/1 TLS fixture transport with the same private roots,
  hostname verification, connection policy, and response bounds. Separate
  transport tests prove production DNS, authority, proxy, and redirect policy;
  the process envelope conservatively measures both retained clients. The process
  oracle is the maximum non-negative preconstruction-to-peak
  `smaps_rollup:Rss` delta; Go live-heap/stack metrics and `VmRSS` are
  attribution diagnostics because runtime-mapped memory may remain mapped after
  GC. Test-owned construction holds
  `A * 10,000` source-equivalent completion descriptors, their original
  `H`-bounded ETags, and the pinned SDK's escaped XML buffer at the real
  Complete request boundary, so it need not stream a fake 50 GiB object merely
  to count retained protocol state. Filled header/control cases independently
  exercise the D4 parser amplification terms.
- No emulator is selected. It cannot prove either exact provider tuple and
  adds no adapter observable absent from the scripted real-SDK path.
- Amazon S3 and Cloudflare R2 conformance are two credentialed, fail-closed
  targets with distinct results. Each uses a pre-existing operator-owned
  bucket and identity, one caller-supplied unique key prefix, and immediate
  cleanup registered before the first mutation. Neither target provisions or
  changes a bucket, identity, policy, lifecycle rule, versioning, encryption,
  DNS, or network control.
- Missing live credentials may defer only the matching support-certification
  receipt. They do not permit a skip to pass, do not block deterministic code
  proof, and do prohibit claiming that provider tuple supported.

## Proof obligations

| ID and source | Disposition and plausible wrong behavior | Controlled setup, trigger, and discriminating oracle | Boundary and executable command | Fixture/input and status | Proof owner, Planning constraint, and reopen owner |
| --- | --- | --- | --- | --- | --- |
| **TD-001** — R1/R2, shared port and key boundary | New portable contract table plus fuzz seeds. Wrong behavior: a provider/SDK field reaches the port, a valid key is normalized, or an invalid key signs or performs I/O. | Compile a small feature fake against all five methods; partition empty, 1/1024/1025-byte, allowed ASCII, Unicode, `soap`, dot/empty segment, leading/trailing slash, and escaping cases. The oracle is exact byte preservation for accepted keys and `invalid` plus zero fake/transport calls for rejected keys. | Unit: `go test -vet=off ./internal/objectstorage -run '^(TestPortContractAndKeyGrammar|FuzzValidateKey)$' -count=1` | Decision-neutral strings derived from R2; new test code only. | `internal/objectstorage/store_test.go`; land before adapter scenarios. Reopen Specification if the grammar or port vocabulary changes. |
| **TD-002** — R3/R4/R10, D1/D3/D4, exact construction and image-root snapshot | Strengthen config/client/bootstrap tables. Wrong behavior: an invalid tuple starts; a provider-invalid or provider-reserved bucket name reaches signing; Amazon accepts a long-lived/no-session credential or missing/wrong expected owner; R2 accepts an arbitrary account subdomain/jurisdiction, expected owner, or Amazon's larger object ceiling; ambient AWS/root state changes policy; credentials refresh or perform I/O; construction performs DNS/provider I/O; strict image-root failures fall back; the successful client has nil `RootCAs`; or an invalid content type reaches signing. | Table exact Amazon and R2 default/EU/FedRAMP endpoints, 32-lowercase-hex account IDs, the shared 3-63 lowercase/digit/hyphen bucket subset with alphanumeric ends and all Amazon reserved prefixes/suffixes rejected, Amazon 12-digit expected owner plus required session token, R2 forbidden expected owner, exact Amazon 5 TiB and R2 5 TiB-minus-5 GiB equality/one-byte-over ceilings, every missing credential/bound, and hostile ambient AWS/root variables. Preserve strict bounded image-root and content-type cases. The oracle is startup rejection before provider effect for invalid cases; valid direct `s3.Options` uses only the explicit snapshot, global `aws.NopRetryer`, the read-only per-operation retry override, required checksum modes, expected owner on Amazon modeled inputs only, and non-nil private roots while DNS/provider/credential calls and credential workers stay zero. | Unit/component: `go test -vet=off ./internal/infra/s3 ./internal/config ./cmd/service/internal/bootstrap -run '^(TestConfigRejectsInvalidTupleAndEnvelope|TestProviderObjectSizeCeilings|TestImageRootBundleLoaderIsStrictAndBounded|TestUploadRejectsUnboundedContentType|TestNewUsesStaticConfigurationAndImageRootsWithoutNetworkIO|TestObjectStorageStartupLoadsImageRootsLocally)$' -count=1` | Synthetic exact Amazon/R2 jurisdiction origins, shared and reserved bucket names, provider object-ceiling boundaries, explicit temporary Amazon and static/temporary R2 canary credentials, expected-owner boundaries, hostile ambient variables, and generated CA bundles; no live endpoint. | Existing config/image/client/bootstrap test owners; land before retry/operation proof. Reopen Technical Design if direct construction cannot isolate ambient SDK/root policy or project exact authority fields; Specification for another credential or trust class. |
| **TD-003** — R3/R9, D2/D4, final authority, explicit roots, and one transport attempt per SDK attempt | Strengthen `httpclient` and adapter transport proof. Wrong behavior: redirect/proxy/private resolution/endpoint rewriting receives signed material; `net/http` transparently replays one SDK attempt; HTTP/2/compression changes bytes; a response escapes its cap; roots/hostname verification weaken; or the transport loses attempt/send evidence. | Retain the current authority/DNS/redirect/root fixtures. A real HTTP/1 server drops one replayable request and one-attempt mode creates no hidden second request. Separately drive the adapter read retry and require each of its at most three SDK attempts to use one fresh connection, while every mutation has one connection/request. Preserve exact-byte, cap, root-pool immutability, wrong-host/ambient-only denial, and nil-root compatibility cases. | Transport/component: `go test -vet=off ./internal/infra/httpclient ./internal/infra/s3 -run '^(TestOneAttemptTransportDoesNotReplayOrTransform|TestTransportUsesCallerRootCAsWithoutAmbientFallback|TestTransportRefusesAlternateAuthority|TestTransportBoundsControlAndObjectBodies|TestErrorMappingAndReadRetryAreConservative)$' -count=1` | Existing HTTP/TLS fixtures plus counted adapter attempts; no provider. | Existing `httpclient`/S3 transport tests; reopen Technical Design if one SDK attempt can replay or explicit roots weaken, and Specification for private/custom roots. |
| **TD-004** — R4 process admission and request concurrency | New deterministic concurrent component test. Wrong behavior: any operation bypasses capacity, capacity queues, reads an upload body, signs, or starts I/O; multipart/cleanup overlaps requests; a leaked body escapes the global ceiling. | Configure admission `A=2` and hold both tokens with channel-blocked operations. At saturation invoke upload, download, metadata, delete, and presign in turn; every call must return `busy` before reader access, signing, or transport. Release each admitted operation through success, error, download EOF, and Close; assert a new call is admitted. Multipart scripts reject any overlap and cleanup starts only after the failed stage returns. Race and `goleak.VerifyNone` cover token/goroutine ownership. | Component/race: `go test -vet=off ./internal/infra/s3 -run '^TestAdmissionIsProcessWideAndNonBlocking$' -count=1`; `go test -vet=off -race ./internal/infra/s3 -run '^TestAdmissionIsProcessWideAndNonBlocking$' -count=10` | Scripted transport/signing barriers and counted reader; `A=2` is a test fixture, not a runtime default. | `client_test.go` plus operation owners; land before body-specific tests. Reopen Technical Design if any selected SDK path adds a worker, queue, or overlapping request. |
| **TD-005** — R4/R9, D2/D3/D4, one deadline and body ownership | Strengthen the phase table using the real adapter and controlled blockers. Wrong behavior: a caller or maximum deadline is ignored; DNS/connect/TLS verification escapes that context; cleanup gets a fresh budget; cancellation starts another part/attempt; an upload source is not closed exactly once or its blocked Read retains admission; a download retains admission after terminal read/Close; or the TLS case false-passes because trust failed before the intended cancellation barrier. | Under `synctest`, capture the effective context at adapter request write, response wait, body read, multipart part, Abort, ListParts, and presign before/after signing. In `httpclient`, an overridden dial path captures the same request context for DNS/connect and a TLS peer whose exact hostname and chain first succeed against the explicit test-only non-nil pool, then blocks the handshake until cancellation; a distinct ambient-only CA remains untrusted. Advance the earlier caller/configured deadline or cancel; assert the matching stable result on non-mutating/proved-unsent work, no new stage, the same deadline in cleanup, exactly one upload-source Close that unblocks a non-cooperative Read, and idempotent download release on EOF/error/Close. For possibly sent mutations, assert `outcome_unknown` outranks context error. | Unit/component: `go test -vet=off ./internal/infra/httpclient ./internal/infra/s3 -run '^(TestOneAttemptTransportUsesRequestDeadlineAndExplicitRoots|TestEffectiveDeadlineAndLifecycleOwnEveryPhase|TestUploadCancellationClosesBlockedSourceAndReleasesAdmission)$' -count=1` | Owned channels, tracked readers/bodies, fake-clock bubble, configured and ambient generated CAs, exact-host loopback TLS, and an uncancelled trust-control case; elapsed time is diagnostic. | `httpclient/client_test.go`, `s3/client_test.go`, `upload_test.go`, `download_test.go`, and `errors_test.go`; land the explicit-root transport prerequisite before T9. Reopen Technical Design if an SDK/transport/TLS stage loses caller context or explicit roots; Specification if error precedence or trust class changes. |
| **TD-006** — R4 success 4, D2/D4 bounded image-root and retained-memory envelope | Replace the superseded arithmetic proof with an exact checked table, independently reviewed pinned-source/image receipt, and conservative Linux process envelope. Wrong behavior: any add/multiply/ceil/align/conversion/max wraps, clamps, or saturates; equality/one-byte-short is wrong; `B=448 KiB`, `N=288`, `H`, and `E` drive the wrong terms or become coupled; the image input or accepted-root count exceeds its half-ceiling; the S3 path reaches nil/system roots; a live allocation has no one D4 receipt class or finite driver; `F`, `S`, `U`, `trust_shared`, `trust_startup`, or `trust_verify` lacks 100% source-derived headroom; construction, retained pool, root-driven verification, descriptors, pointees/backing bytes, upload ID, configured strings, parser state, original plus escaped Complete XML capacity, TLS/stack/shared state, or a worker is omitted; fixture/ambient-root memory contaminates the child; a small/padded CA fixture stands in for retained maximum roots; RSS exceeds the equation; or cancellation leaves an owned resource even when Go retains free pages. | Unit cases independently calculate D4's `heap`, `U`, `Q`, `trust_shared(B,N)`, `trust_startup(B,N)`, `trust_verify(N)`, `retained_parts`, `complete_xml_len`, `complete_xml`, `upload_session`, `construction`, `shared`, every operation class, and `required_memory=max(construction, shared+A*max(...))`. Cover positive minimums; exact equality/one-byte-short; `P=1/10,000/10,001`; `B=448 KiB`/`B+1`; `N=288`/`N+1`; actual `b<=B/2` and `n<=N/2` with unused bytes/count/percent; `H<E`, `H=E`, `H>E`; content type `0/1/1,024/1,025`; the 32 KiB heap branch and 8 KiB alignment edge; every trust/add/multiply/align/ceil/conversion/`Q`/`A*max` overflow; and a wrong-architecture shallow receipt. Before constants are accepted, a read-only receipt resolves `GOOS/GOARCH`, Go 1.26.6, AWS core `v1.43.5`, credentials `v1.19.5`, S3 `v1.107.1`, Smithy `v1.27.7`, module sums, the nine complete Go source SHA-256 identities fixed by D2/D4, and the pinned T9 image bundle path/hash/bytes/unique-valid-CA count. Apply D4's seven allocation classes to every reachable allocation site with checked compiler output, exact shallow/rounded sizes, named barrier counts, and symbolic live-copy counts. Prove `used_F<=F/2`, `used_S<=S/2`, `used_U(x)<=32*heap(x)+32 KiB`, and each used trust term at most its design half for zero, allocator breakpoints, overflow, `B`, `N`, and admitted `Q`, `Q+K`, `Q+K+T`, `H`, and `E`; record used/unused bytes and percentage per row. Unknown, duplicate, reclassified, non-finite, wrong-driver, source/image mismatch, or half-ceiling failure reopens D4; T9 cannot tune a reserve. An unmeasured controller/TLS-fixture PID launches five identical `GOMAXPROCS=1` children containing runtime, adapter, real SDK, the retained production client, a separate exact-host fresh HTTP/1 TLS fixture transport with the same private roots and response bounds, and a pre-opened quiescent scalar IPC channel. The duplicate retained transport is a conservative memory superset; separate focused tests own production DNS/IP, authority, proxy, redirect, and ambient-root denial. Every child records one preconstruction `smaps_rollup:Rss` baseline after runtime/IPC warmup, then constructs the complete configuration and uses that same baseline for held trust-construction, idle-adapter, and every admitted-operation delta. The trust-construction barrier explicitly keeps the complete `B`-driven PEM/decode input and parsed `N`-root pool live; idle keeps only the immutable pool; admitted barriers hold `A=2` real TLS verifications for every formula class. The generated bundle contains `N` unique valid CAs with retained DER/parsed input driven to `B`; padding that only enlarges discarded PEM is rejected. Complete holds `2*10,000` source-equivalent descriptors with escape-amplifying `H`-bounded ETags while originals and XML capacity are live; response cases maximize `H` and `E` independently; object/chunk variants prove no retained object/part buffer. After controlled GC only where it cannot erase the held state, each child computes the maximum non-negative same-baseline construction/idle/admitted RSS delta; the five-run maximum must be at most both independently reviewed `required_memory` and the configured ceiling. Verify each complete same-snapshot `smaps_rollup` resident-category equation; record nearby `VmRSS`, Go live-heap/stack/os-stack, and goroutine samples only as diagnostics. Cancellation must join every child call; child active/admission, response-body, and client-connection counts must be zero; child `goleak` against its post-construction idle baseline must find no new adapter/SDK/HTTP/TLS-client/control goroutine. Scalar IPC separately proves fixture request/body/accepted-connection/goroutine counts zero before controller joins it. No prewarmed nil-root pool, small one-root bundle, system-root subtraction, fixture-PID allocation, `VmRSS`/Go-mapped-memory substitution, cross-barrier baseline, or return-to-idle RSS is admissible. | Structural/process: `go test -vet=off ./internal/infra/s3 -run '^(TestWorkingMemoryAccounting|TestImageRootBundleLoaderIsStrictAndBounded|TestUploadRejectsUnboundedContentType)$' -count=1`; Linux retained-envelope gate: `GOMAXPROCS=1 make test-s3-envelope` | Test-only `A=2`, maxima `B/N/K/P/T`, independently maximal bounded `H/E`, actual configured-string lengths `Q`, unique generated CA DER plus exact-host fixture, hostile ambient-root canaries in the separate production-transport tests, pinned source/module/image receipts, exact build target, separate controller/fixture and measured-child PIDs, fixed-size scalar IPC, Docker Linux `/proc`, owned counters, and goroutine profiles; no provider endpoint or credential. The independently reviewed source/image receipt is mandatory implementation proof input and is never derived by calibrating production constants. | `config_test.go`, `image_root_bundle_test.go`, `upload_test.go`, `client_test.go`, and the Make proof entrypoint; trust-root prerequisite and separate envelope acceptance must close before live conformance. Reopen D4 on any unknown reserve, source/build/image/bundle/mount-policy drift, nil/system-root reachability, unstable/missing `smaps_rollup`, fixture contamination, unexplained child RSS, nonzero owned resource, leaked worker, coefficient/formula/classification change, trust half-ceiling failure, or inability to hold the ceiling; Docker Linux availability makes unsupported-runner disposition inapplicable to this candidate. |
| **TD-007** — R5 single upload, D4/D5 bounded streaming CRC64NVME | New real-SDK scripted wire falsifier. Wrong behavior: the body is buffered/pre-read, byte `length+1` is consumed, short input succeeds, threshold/intent drifts, checksum is absent/wrong, content type escapes its D4 boundary, or create-only overwrites. | A read-gated non-seekable source proves no read before admission. First apply TD-002's content-type byte/HTTP-field boundary. Decode the one `aws-chunked` request independently, compute CRC64/NVME with stdlib from the declared bytes, and require exact content length, trailer, algorithm/type, accepted optional content type, and `If-None-Match:*` only for qualifying create-only. Return matching, missing, and corrupt checksum fields; assert only the match succeeds. Short input fails; an extra sentinel remains unread; at/above threshold selects the accepted path exactly. | Component: `go test -vet=off ./internal/infra/s3 -run '^(TestUploadRejectsUnboundedContentType|TestSingleUploadStreamsCRC64NVMEAndExactLength)$' -count=1` | Known-answer bytes and polynomial `0x9a6c9329ac4bc9b5`; boundary content types; scripted response metadata; no provider. | `checksum_test.go` and `upload_test.go`; land together. Reopen Technical Design if SDK framing/metadata or the request-field bound differs; Specification if common integrity or create-only semantics cannot hold. |
| **TD-008** — R5 multipart path, D5 | New serial multipart state-machine test. Wrong behavior: create-only enters multipart, parts overlap or size/order drifts, whole checksum covers different bytes, completion treats embedded XML error/initial 2xx as success, or retained completion inputs are unbounded. | Use replace at `C+1`, exact multiples, final remainder, and provider part-count edge. Script Create, each serial UploadPart, and Complete; independently hash the concatenated limited readers and compare every part plus final CRC64NVME/FULL_OBJECT, ordered part/ETag/checksum list, and declared object size. Inject missing part checksum, out-of-order receipt, corrupt final checksum, and embedded completion error; each fails and routes to cleanup. | Component: `go test -vet=off ./internal/infra/s3 -run '^TestMultipartUploadIsSerialAndConfirmsWholeChecksum$' -count=1` | Small provider-valid part fixture for wire behavior; maximum part-count retention is TD-006. | `checksum_test.go` and `upload_test.go`; same implementation unit as TD-007. Reopen Technical Design on SDK checksum/completion drift; Specification if multipart needs a provider escape. |
| **TD-009** — R5 multipart cleanup, D6 | Strengthen the bounded cleanup table. Wrong behavior: a lost Create response reports no cleanup risk; a new part starts after failure; pagination detaches or escapes its bound; an empty listing is promoted to stable terminal cleanup; a page/body remains unread; or `pending` is hidden. | Lose Create before and after request-header write, then fail after known upload ID at part/complete boundaries. A possibly sent create with no returned ID is `pending` and makes no unsafe guessed-ID call. With a known ID, Amazon performs at most three serial Abort cycles and at most ten valid ListParts pages per cycle under the same remaining context; a complete non-empty listing triggers another abort and an empty listing stops the immediate attempt. Every failed multipart path, including empty Amazon listing and every R2 observation, returns `pending`. Assert primary error is preserved, no upload ID escapes, all bodies close, and no background call appears after return. | Component/race: `go test -vet=off ./internal/infra/s3 -run '^TestMultipartCleanupIsBoundedAndConservative$' -count=1`; `go test -vet=off -race ./internal/infra/s3 -run '^TestMultipartCleanupIsBoundedAndConservative$' -count=10` | Scripted lost responses, pages/markers, retained-part cycles, and empty-listing race; no provider evidence is converted into terminal cleanup. | `upload_test.go`; land with cleanup mapper. Reopen Technical Design if a provider later documents and exact conformance proves a stable terminal observation; lifecycle remains deployment-owned. |
| **TD-010** — R4/R5/R6 download completion and body lifecycle | Strengthen the streaming-body table. Wrong behavior: headers count as success, an unexpected range response is accepted, missing/negative/oversized length or missing/composite checksum exposes a body, mismatch depends on an SDK error string, a deferred read restarts, or ordinary Close/context end leaks the body or admission. | Script declared size valid/oversized/missing/negative, Content-Range absent/present, installed validator metadata valid/missing/composite, matching/mismatching payload with EOF/unexpected-EOF/SDK-deferred error, exact limit/overflow, early Close with live/cancelled context, idle-body cancellation, and deadline. Acquisition retries scripted 503/SlowDown twice before one valid body, exhausts at exactly three attempts, and cancellation after the retryable response starts no next attempt; no terminal body read is retried. The oracle is no returned body for range/invalid metadata/declared size; success bytes only at clean validated EOF; context error wins when cancelled, otherwise any terminal checksum mismatch is `integrity_failed` independent of error text; ordinary incomplete Close records `internal`; EOF, error, Close, and context end share one underlying Close/token release even under races. | Component/race: `go test -vet=off ./internal/infra/s3 -run '^(TestDownloadCompletesOnlyAtValidatedEOF|TestDownloadAcquisitionRetryIsBounded)$' -count=1`; `go test -vet=off -race ./internal/infra/s3 -run '^(TestDownloadCompletesOnlyAtValidatedEOF|TestDownloadAcquisitionRetryIsBounded)$' -count=10` | Scripted SDK checksum/range metadata/body, retryable acquisition failures, and tracked closer; no provider. | `download_test.go`; land after checksum/client base. Reopen Technical Design if SDK cannot expose installed validator/EOF failure; Specification if integrity meaning changes. |
| **TD-011** — R6/R7/R9 metadata, delete, safe-read retry, and conservative errors | Strengthen exact mapping tables through the real SDK deserializer. Wrong behavior: metadata exposes provider fields; generic HEAD 404 stays internal; 401/403 becomes absence; Amazon `ConditionalRequestConflict` or either provider's `PreconditionFailed` is misclassified; a write retries; a final read throttle/5xx is not `temporary`; possible-send mutation becomes a false terminal/context result; malformed XML/provider text leaks; or diagnostics preserve unsafe evidence. | Script Head/Get/Delete success; generic 404, 401/403, exact Amazon and R2 412 `PreconditionFailed`, Amazon 409 `ConditionalRequestConflict`, negative Amazon other-code 409 and R2 409, throttling/5xx followed by success and exhaustion, malformed/oversized XML, request-write loss, and after-write response loss including CreateMultipartUpload. Assert portable fields only, UTC time, closed kind, sanitized feature error, finite failure phase, the exact credential-code subset and remaining category precedence, allowlisted code, bounded request ID, read attempt count `1..3`, and mutation stage count one. Exact 412 is `precondition_failed`; Amazon 409 is `temporary` so a caller may start a fresh upload but the adapter never replays it; negative cases and any otherwise possibly sent Put/CreateMultipart/Complete/Delete remain `outcome_unknown`; cancellation during backoff starts no new attempt. | Component: `go test -vet=off ./internal/infra/s3 -run '^(TestMetadataAndDeleteExposePortableResults|TestErrorMappingAndReadRetryAreConservative|TestErrorMappingIsConservativeAndOneAttempt|TestProviderDiagnosticClassificationIsFinite|TestDownloadAcquisitionRetryIsBounded)$' -count=1` | Scripted Smithy status/code/request-ID corpus including secret/unbounded canaries and counted request stages; no provider. | `metadata_test.go`, `download_test.go`, `delete_test.go`, `errors_test.go`; land with mapper/retry owner. Reopen Technical Design for SDK/Smithy/send-phase/retry drift; Specification for changed portable absence/delete/error semantics. |
| **TD-012** — R8 presigned GET | Strengthen local-signing and redaction tables. Wrong behavior: another method/authority/key/TTL is signed; Amazon omits or R2 receives expected-owner; `SignatureExpiresAt` disagrees with the signature or is presented as guaranteed credential validity; a default TTL appears; query scope drifts; duplicate security fields are accepted; the contract claims a full transfer despite unsigned Range; ambient config changes signing; or bearer material enters evidence. | With fixed credentials/key, capture times around `0`, `1s`, configured max, max+1, and seven-day calls. Assert exact GET, no fragment, bucket authority, decoded key, exact singular SigV4 algorithm/date/expiry/signature/token/credential fields, sorted unique signed headers, and credential scope `<access-key>/<date>/<configured-region>/s3/aws4_request`; require the 32-byte hex signature, exact requested lifetime, and `SignatureExpiresAt` equal to signed date plus expiry while allowing credential expiry/revocation to end access earlier. Amazon includes exactly one configured `x-amz-expected-bucket-owner`; R2 omits it. Verify signing does not include Range and document/test that an added unsigned Range does not change local signature material; this proves the result is object GET authority, not whole-transfer proof. Feed URL/query/headers through evidence collectors and require canaries absent. No HTTP call occurs and issuance never emits transfer success. Live signature behavior remains provider-specific TD-017/018. | Unit/component: `go test -vet=off ./internal/infra/s3 -run '^TestPresignGETIsBoundedAndSecret$' -count=1` | Fixed test credential/key/expected owner and parsed signed-query scope/expiry; no live URL or clock abstraction. | `presign_test.go`; land with presigner. Reopen Technical Design on presigner output drift; Specification for another operation/lifetime/recipient guarantee. |
| **TD-013** — R11 telemetry secrecy and bounded diagnostics | Strengthen the reachable outcome matrix and whole-adapter secrecy canary. Wrong behavior: a required failure boundary is missing, a secret/object/provider message appears, attempt/status/code/request-ID fields are unsafe, or metric labels become input-driven. | Retain the reachable operation/result/path/cleanup matrix. For provider failures additionally assert span-only bounded attempt count, numeric status, allowlisted code with `other` fallback, closed category (`credential|authority_tls|throttle|provider|transport`), and request ID matching `[A-Za-z0-9._:/+=-]{1,128}` with `invalid` fallback. Exercise 401, 403, 404, 409, 412, 429, 5xx, TLS trust failure, transport failure, and retry exhaustion. Inject canaries in every forbidden field and require none in logs/spans/metrics/errors; none of the new diagnostics is a metric label. | Component: `go test -vet=off ./internal/infra/s3 -run '^TestTelemetryContractIsBoundedAndSecret$' -count=1` | Existing recording OTel patterns; scripted reachable outcomes and synthetic unbounded/secret canary corpus; no exporter/provider. | `telemetry_test.go`; land after all result paths so it is exhaustive. Reopen Specification for a new result/operator question; Technical Design if a required safe signal lacks an owner. |
| **TD-014** — R10 lifecycle/readiness, D1/D3/D4 bootstrap trust snapshot | Strengthen construction/outage/readiness and lifecycle-order proof. Wrong behavior: strict image-bundle failure reaches ready; startup probes storage; trust loading happens before memory-limit publication or after client publication; a post-start file/provider change reloads roots, changes readiness/liveness, or registers a probe/restart path; close order drifts; active work gets a longer shutdown budget; or idle transport closes twice observably. | Bootstrap through the unexported bounded bundle source with a valid exact-host CA and a transport that records every request. Invalid bundle cases must fail startup before readiness and before DNS/provider calls. Valid construction performs only the bounded local trust-file read, publishes one immutable pool after the memory-limit event, records zero provider calls and no storage readiness probe, and does not reread the source after publication. Capture readiness/liveness and probe inventory, mutate/remove the test source, then make one admitted operation fail as provider unavailable and a later operation succeed against the already captured pool; only operation error/telemetry may change. Inject ordered lifecycle events: early return closes once; normal shutdown closes object runtime after HTTP drain/background join and before dependencies/telemetry flush; repeated safety close is a no-op. Cancel active upload/download and assert no detached request or cleanup. | Component/lifecycle: `go test -vet=off ./cmd/service/internal/bootstrap -run '^(TestObjectStorageStartupLoadsImageRootsLocally|TestObjectStorageStartupAndOutageDoNotChangeReadiness|TestObjectStorageRuntimeCloseOrder)$' -count=1`; `go test -vet=off -race ./cmd/service/internal/bootstrap -run '^TestObjectStorageRuntimeCloseOrder$' -count=10` | Existing health/bootstrap fakes, ordered event recorder, counted trust source, exact-host generated CA, scripted provider failure, and synctest lifecycle pattern; no production path or provider. | `startup_object_storage_test.go` and `run_lifecycle_test.go`; land the trust-root prerequisite before T9. Reopen Technical Design if snapshot ownership, construction point, or lifecycle order cannot hold; Specification if storage becomes readiness-critical or roots require reload. |
| **TD-015** — R3/R12 typed config, secret sources, and no CA setting | Strengthen config inventory/source tests. Wrong behavior: object settings survive `none`; empty/unknown fields are ignored; a secret comes from YAML/defaults; an access-key ID is logged; a selected output gains a usable endpoint/bucket/credential/default bound; or any YAML/environment/CLI leaf selects, augments, replaces, or reloads S3 trust roots. | Compare exact leaf/default/snapshot inventories for absent and selected profile. Load empty placeholders, valid environment-only credential canaries, non-empty file values for all three static credential fields, and plausible CA path/content/reload keys plus `SSL_CERT_FILE`/`SSL_CERT_DIR`. The oracle is unknown-key rejection when absent, fail-closed required finite values when selected, empty non-secret examples only, environment credential acceptance, file-source credential rejection without raw value, no ambient AWS leaf, and no production CA/trust configuration leaf or secret-policy exception. | Unit: `go test -vet=off ./internal/config -run '^(TestObjectStorageConfigContract|TestSnapshotContract|TestStaticCredentialSourcePolicy)$' -count=1` | Existing configtest/source-policy harness, synthetic values, and hostile root-key names only. | Object config, snapshot, and secret-policy tests; land before profile matrix. Reopen Specification for another credential/config/trust source; Go Ownership if section placement changes. |
| **TD-016** — R12 deterministic profile, trust-owner inventory, and dependency pruning | Strengthen the independent template-init oracle. Wrong behavior: `none` retains object code/config/docs/tests/deps; `s3` loses `image_root_bundle.go`/test or the generic non-nil-`RootCAs` `httpclient` branch; a generated service gains a production CA setting or bundle copy; a rejected client/transfer manager remains; selectors remove each other; repeat mutates; unknown/empty mutates; markers/generator sources survive; or lock/output disagree. | Generate all four `OBJECT_STORAGE=none|s3` × `OUTBOUND_HTTP=none|bounded` combinations, with authn-only controls for the three-way `httpclient` predicate. Compare independent present/absent inventories including the S3 strict image-root owners and shared code-only `RootCAs`; compile/test each retained output, run `go mod tidy`, inspect exact module closure, require no new bundle/config source and no custom/private-root path, verify marker/generator absence, `template.lock`, completion output, unknown config rejection, byte-identical equal repeat, and unchanged incompatible repeat. The existing runtime-image bundle remains image-owned rather than copied by the generator. | Profile/structural: `TEMPLATE_INIT_PROFILE=object-storage make template-init-check`; `make project-structure-check`; `make mod-tidy-check` | Existing checkout-copy/init harness extended with exact trust-owner inventories; credential- and provider-free. | `scripts/ci/template-init-check.sh` and profile owners; one profile acceptance unit after the trust-root prerequisite and T9. Reopen Go Ownership for inventory/retention placement; Technical Design if profile generation must own/copy roots; Specification if selector or trust-source semantics change. |
| **TD-017** — Amazon S3 exact-tuple conformance and production public-chain receipt | New required credentialed target; no other provider result is reusable. Wrong behavior: the selected commercial regional general-purpose tuple rejects a governed field/path, hides required integrity, retries, misclassifies absence, overwrites create-only, leaves unreported parts, or produces invalid/restricted presign behavior; the exact hostname does not validate through the adapter-private production public-root snapshot; or the receipt silently uses nil/system/custom roots or a replaceable bundle. | Preflight the exact endpoint/region/bucket/virtual authority, temporary-session identity-policy receipt, never-enabled versioning observation, and the production-image trust receipt: pinned architecture image digest, fixed `/etc/ssl/certs/ca-certificates.crt` path, upstream provenance/revision, SHA-256, bytes `<=448 KiB/2`, unique valid roots `<=288/2`, regular read-only image ownership, and no replacing runtime mount. The deterministic trust prerequisite proves wrong-host and ambient-only denial with the same strict loader; the first exact signed request through that pool proves the Amazon chain and hostname. Under `conformance/amazon_s3/<run-id>/`, exercise the existing port matrix while the adapter requires provider checksum evidence, plus independent byte/metadata/cleanup oracles and registered prefix-only cleanup with final empty readback. TD-003/007/010/011 remain the deterministic attempt, retry, and known-answer checksum authority because this exact-provider run does not inject transport faults. The provider receipt records the non-secret tuple plus exact trust/image/checksum/cleanup/identity-policy facts; any missing, stale, or ambiguous field fails Amazon only. | Credentialed exact-provider: `REQUIRE_S3_CONFORMANCE=1 make test-s3-conformance-amazon` | Required external inputs: Amazon endpoint, region, dotless bucket, 12-digit expected bucket owner, unique run ID, primary and concealment temporary access/secret/session credential sets, identity-policy receipt, never-enabled versioning evidence, abandoned-multipart lifecycle backstop, and the exact production-image/bundle/no-override receipt. Unavailable until an authorized run; secrets remain environment-only and unrecorded. | `test/s3conformance/conformance_test.go` Amazon entrypoint and Make target; separate support-certification unit after deterministic T9/T10 proof. Test-only bucket/readback operations never enter production. Reopen D2/D4 Technical Design on bundle/public-chain/hostname, source, trailer/checksum, one-attempt, or cleanup mechanism failure; Specification only under the design reopen conditions. |
| **TD-018** — Cloudflare R2 exact-tuple conformance and production public-chain receipt | New required credentialed target; Amazon evidence is inadmissible. Wrong behavior: an accepted exact account endpoint rejects the AWS SDK trailer/fields, returns incompatible checksum/absence/cleanup semantics, silently changes keys, or presign works only through an excluded custom domain; the exact R2 hostname does not validate through the adapter-private production public-root snapshot; or the receipt substitutes Amazon, nil/system/custom roots, or a replaceable bundle. | Preflight the exact default/EU/FedRAMP account S3 endpoint, region `auto`, dotless bucket, virtual authority, static-or-temporary identity-policy receipt, R2 versioning/lifecycle facts, and the same production-image trust receipt fields required by TD-017. The deterministic trust prerequisite proves wrong-host and ambient-only denial with the same strict loader; the first exact signed request through that pool proves the R2 chain and hostname. Run the same port-level matrix and byte/metadata/cleanup oracles as TD-017 under `conformance/cloudflare_r2/<run-id>/`, using R2 credentials and no custom domain. Cleanup remains `pending`; registered direct cleanup removes only owned objects/uploads and requires empty readback. The distinct receipt records R2 and trust/image results only. | Credentialed exact-provider: `REQUIRE_S3_CONFORMANCE=1 make test-s3-conformance-r2` | Required external inputs: exact R2 account endpoint/jurisdiction, dotless bucket, unique run ID, primary and concealment static-or-temporary credential snapshots, identity-policy receipt, lifecycle/versioning evidence, abandoned-multipart lifecycle backstop, and the exact production-image/bundle/no-override receipt. Unavailable until an authorized run; secrets remain environment-only and unrecorded. | `test/s3conformance/conformance_test.go`, R2 entrypoint and distinct Make target; separate support-certification unit after deterministic T9/T10 proof. Test-only bucket/readback operations never enter production. Reopen D2/D4 Technical Design on bundle/public-chain/hostname, source, trailer/checksum, one-attempt, or cleanup mechanism failure; Specification only under the design reopen conditions. |
| **TD-019** — R2 feature authorization/content/retention ownership | First-adopter placement obligation; no template test is invented without a feature owner. Wrong behavior: a feature denial still reaches the store, or the adapter starts authorizing principals, constructing tenant keys, or deciding content/retention policy. | At the first real feature composition, drive denied principal/operation/key/content/size/retention/overwrite cases through that feature with a counted `objectstorage.Store` fake. The oracle is the feature's own stable denial and zero store calls; allowed cases pass the exact feature-produced key/intent unchanged. | Exact future command is feature-owned and unavailable until an adopter/package exists; Planning records a scope exit rather than inventing a reference feature. | No current adopter is accepted by Technical Design. This input is outside the template implementation and becomes mandatory before an adopter claims the capability. | Future feature package test owner. Reopen Specification if any policy moves into the pack; Go Ownership when the first adopter fixes its composition point. |

### Mandatory repair oracles

TD-002 additionally requires
`TestExpectedBucketOwnerProjectsEveryOperation`. A real-SDK scripted table
invokes PutObject, CreateMultipartUpload, UploadPart,
CompleteMultipartUpload, AbortMultipartUpload, ListParts, GetObject,
HeadObject, DeleteObject, and presigned GetObject exactly once. Every Amazon
request must project the configured expected-owner field/header, including the
signed presign query field; a direct R2 projection exercises the otherwise
provider-gated ListParts input, and every R2 request must omit the field.

TD-006 additionally runs `S3_RECEIPT_PLATFORM=<platform> make
test-s3-source-receipt` for both `linux/amd64` and `linux/arm64` before
accepting its constants or envelope. Each target must fail on Dockerfile Go
identity, platform manifest, Go source hash, module, bundle, or target drift, and
`scripts/ci/s3-source-receipt.sh` is an explicit proof owner.

TD-013 applies D8's finite provider-code allowlist with `other` fallback, its
exact diagnostic-category precedence, and request-ID grammar
`[A-Za-z0-9._:/+=-]{1,128}` with `invalid` fallback. These values are span-only
and never become metric labels. Only HTTP 401 or
`InvalidAccessKeyId|SignatureDoesNotMatch|ExpiredToken` classifies as
`credential`; other allowlisted codes follow the remaining finite precedence.

For TD-012, Amazon expected owner is required in the signed URL query, not the
returned opaque headers; R2 must omit the query field. The presign validator and
test parse that exact SDK output before returning the bearer result.

### TD-006 `smaps_rollup` consistency oracle

Each child's preconstruction baseline and held trust-construction, idle, and
admitted-operation sample comes from one complete
`/proc/<child-pid>/smaps_rollup` read. Every delta for that child subtracts the
same preconstruction sample. Parse the integral `kB` values with checked
conversion and addition, require every field exactly once, and require with no
tolerance:

```text
Rss == Shared_Clean + Shared_Dirty + Private_Clean + Private_Dirty
```

A missing or duplicate field, another unit, parse failure, overflow, or unequal
sum fails that sample before the RSS delta is compared with D4. `Pss*`,
`Swap*`, `LazyFree`, `Locked`, `Anonymous`, and huge-page fields are
diagnostics or overlapping classifications and are never added to this RSS
identity.

### TD-006 final-image authority and T9 envelope scope

This subsection controls TD-006's `source/image receipt`, image input, and
`b`/`n` references. T9A must resolve the Dockerfile's pinned Distroless
reference for each supported Linux platform (`linux/amd64` and `linux/arm64`),
build and materialize each final unmounted image, and receipt the Dockerfile
identity, platform, index and
resolved manifest, final-image config/rootfs identity, and the regular `0555`
rootfs entry at `/etc/ssl/certs/ca-certificates.crt`. It hashes and counts that
extracted file, then supplies those same bytes through `imageRootSource` to the
strict loader; both observations must agree. The final platform-resolved image
content is the production authority for D4's `b` and `n`. The Go test/envelope
runner's bundle cannot supply production constants, even when it is valid and
within the ceilings.

T9's five identical Linux runs consume that frozen T9A receipt and exercise the
real SDK/explicit-root transport with the existing maximum `B`/`N` synthetic
fixture. They prove the envelope only; they neither establish final-image
identity nor derive production constants from the runner filesystem. A changed
final Dockerfile stage, platform manifest, final-image identity, bundle entry,
or mismatch between the extracted bytes and strict-loader input reopens D4
before T9A constants or the T9 envelope may be reused.

The unmounted final-image receipt is not deployed-mount proof. Delivery alone
must prove the deployed image identity and absence of a non-root mount at the
bundle path or an ancestor; that no-override receipt remains required for
deployed-path and provider claims, is not a T9A/T9 input, and no deterministic
receipt here authorizes deployment or certification.

## Aggregate gates and claim limits

The focused scenario commands are the behavioral oracles. After they pass, the
current implementation candidate also runs:

```bash
go test -vet=off ./internal/objectstorage ./internal/infra/s3 ./internal/infra/httpclient ./internal/config ./cmd/service/internal/bootstrap
go test -vet=off -race ./internal/infra/s3 ./internal/infra/httpclient ./cmd/service/internal/bootstrap
S3_RECEIPT_PLATFORM=linux/amd64 make test-s3-source-receipt
S3_RECEIPT_PLATFORM=linux/arm64 make test-s3-source-receipt
GOMAXPROCS=1 S3_ENVELOPE_PLATFORM=linux/amd64 make test-s3-envelope
GOMAXPROCS=1 S3_ENVELOPE_PLATFORM=linux/arm64 make test-s3-envelope
TEMPLATE_INIT_PROFILE=object-storage make template-init-check
make project-structure-check
make mod-tidy-check
make lint
```

Those aggregate gates prove only the current tree surfaces they execute. Live
provider support still requires both TD-017 and TD-018. A deployed-path claim
still requires separate target evidence for runtime identity, DNS/TLS/egress,
bucket policy, lifecycle backstop, quotas, telemetry delivery, and deployment
encryption; no such deployment procedure or mutation is authorized here.

## Current proof gaps and fail-before signals

The current T1-T8 tree supplies the port, adapter operations, typed
configuration/bootstrap, real low-level SDK construction, serial multipart
path, and one-attempt transport evidence. It is implementation evidence only;
the open D4/T9 proof gap is exact and narrower:

- `internal/infra/httpclient.Config` has no code-only `RootCAs` input and
  `httpclient.New` explicitly leaves `TLSClientConfig` nil, so the current S3
  path still reaches Go's process-global system roots and ambient
  `SSL_CERT_FILE`/`SSL_CERT_DIR`; `internal/infra/s3/image_root_bundle.go` and
  its strict bounded loader tests do not exist;
- `internal/infra/s3/config.go` still implements the superseded
  `P * (partDescriptorBytes + H)` equation. It has no checked shallow/backing
  allocation and XML-capacity functions, configured string/content-type
  charge, `B/N` root bounds, trust construction/shared/verification terms,
  pinned allocator/parser amplification, retained upload-ID term, or
  simultaneous original-descriptor plus Complete XML buffer term;
- `internal/infra/s3/upload.go` retains the two provider response strings and
  local part-number pointer in each generated `types.CompletedPart`, and the
  pinned serializer materializes the complete XML body while those values stay
  live. The current upload validation does not yet enforce D4's 1,024-byte valid
  content-type boundary;
- strict loader/ambient-root falsifiers, `TestWorkingMemoryAccounting`, the
  independent pinned source/image receipt, the held construction/idle/admitted
  barriers, and `make test-s3-envelope` are absent. Docker Linux does expose
  the required `smaps_rollup:Rss`, owned-resource/goroutine observables, and
  diagnostic `VmRSS`/Go runtime metrics, so this is not an unsupported-runner
  exception;
- provider documentation and Research still cannot substitute for TD-017 or
  TD-018 execution, and neither provider receipt can substitute for D4/T9.

No residual risk is accepted. The unavailable inputs are credentialed
exact-provider fixtures and deployed production-image/no-override receipts
owned by later support-certification/Delivery work, plus the future feature
owner in TD-019. Their absence does not block the credential-free T9 source and
Linux image receipt, but it leaves each provider/deployed-path claim unverified
and forbids either provider receipt or a synthetic feature-policy test.

## Review evidence

Independent QA review was required because the fixed boundary covers signed
credentials and authority, process-wide resource/lifecycle limits, mutating
failure ambiguity, and two non-substitutable provider claims. All review lanes
were read-only.

1. Candidate `332c82200efd13d0f0b335835b9f6587c9b3e3b01320367b7af5ff07e905b94a`
   — **FAIL**: heap-only scripted memory proof omitted real TLS/stack/process
   state and an independent reserve receipt; telemetry had no positive signal
   oracle and required unreachable outcome pairs; readiness was not observed
   after provider failure. The reviewer also found a stale conformance receipt
   reference.
2. Root repair added the pinned-source reserve receipt and real HTTP/1 TLS
   RSS/Go-memory envelope, a reachable positive telemetry matrix, a post-start
   outage/readiness scenario, and corrected both provider receipt references.
   Root self-review also made admission cover all five operations, extended the
   effective deadline through dial/TLS/presign, strengthened the replay
   discriminator, and removed an unowned production clock seam.
3. Candidate `abdc0005e79e4989c56a9d9ade2188aa0b19d3e413c8a4995ffcec66d08517f4`
   — focused **FAIL** only on presign expiry: a plausible returned expiry
   could disagree with the signed query.
4. Root repair made the returned expiry equal parsed `X-Amz-Date + X-Amz-Expires` and
   retained the before/after window only as call attribution. Focused fresh
   re-review of candidate
   `608f7fcc74fa6c0e889e780dcb3f6c738c08ba285598497948e982f0fc558b66`
   — **PASS**. No finding remains; all earlier closed findings were reused.

Changes after the reviewed candidate are this status and review/handoff receipt
only. They alter no proof obligation, setup, oracle, command, input, claim
limit, or reopen route.

Planning's current-tree link-check later caused a focused Go Ownership repair
to config-example and CI-scope file owners. The three-lens ownership panel
returned PASS on the repaired design; no Test Design setup, oracle, command,
fixture, external input, claim limit, or reopen route changed. The accepted
design identity above is refreshed to that ready artifact.

### D4 retained-memory reopen — Test Design QA

Independent QA of design candidate SHA-256
  `38e55cb34c26eeea3e2983e91509f0049201eb844458f39f50090fb8aa92bc82`
and test-plan candidate SHA-256
`bb9c11c6327a19c1e95a1b89e528e772e5b8c00bf758a43706b2320a329b2b01`
returned **FAIL** only because TD-006 required a `smaps_rollup` resident-category
sum without naming the fields or equation. The arithmetic/source receipt,
independent `H/E` cases, content-type boundary, separate measured and fixture
PIDs, child RSS delta, diagnostics, cleanup, commands, owners, fail-before
conditions, and credential-free/provider-neutral claim boundary all passed.

TD-006 now requires one complete same-snapshot read, exactly one integral-kB
instance of each named field, checked conversion and addition, and exact
zero-tolerance equality:

```text
Rss == Shared_Clean + Shared_Dirty + Private_Clean + Private_Dirty
```

It fails before D4 comparison on a missing/duplicate field, unit mismatch,
parse error, overflow, or unequal sum, and excludes overlapping classifications.
Fresh focused QA of repaired test-plan candidate SHA-256
`1956d36d1180c062f684fa239c2b95bb06d80f583ab6cd56fac6201b69f6df89`
with the unchanged reviewed design returned **PASS**. No proof-policy choice or
focused regression remains. Changes after that candidate are this receipt,
status closure, and the final design authority-hash refresh only.

### D4 image-root snapshot reopen — Test Design QA

Fresh independent QA reviewed design SHA-256
`0a24756bc3939ff182c792f984162fde977be5e4bf5ffabc7fb2d4a8bcac5bc2`
and fixed test-plan candidate SHA-256
`08dc8e063ec7bb6e3518c532c235e35695c57ae8f327dc77bc21dfb52e28b5a0`.
It returned **FAIL** on one TD-003 false-pass: pointer identity, configured-chain
success, and ambient-chain denial would not detect `RootCAs.AddCert` widening
the same caller-owned pool.

TD-003 now clones the configured pool before construction and requires
`CertPool.Equal` after construction, the successful configured-chain handshake,
and both ambient-only and wrong-host denial paths, while retaining the same
non-nil pointer, exact `ServerName`, and `InsecureSkipVerify=false` checks.
Focused fresh QA of repaired test-plan candidate SHA-256
`69a16c478621663722a4da7acfd9a8d9a7a3629a08aff330ee3e4dfc621dc04f`
returned **PASS**. The reviewer found no other gap across TD-002, TD-003,
TD-005, TD-006, TD-014 through TD-018, the arithmetic/source/image receipt,
same-preconstruction-baseline five-run Linux oracle, cleanup, downstream
handoff, or provider/deployment claim separation. Changes after that candidate
are this review receipt and status closure only.

### D4 runtime-bundle authority reopen — Test Design QA

Fresh independent QA reviewed design SHA-256
`844eb16dafd7a4ffe357820e1d7d01211e91ed36c3cfdabd102b14c30b6496f9`
and fixed test-plan candidate SHA-256
`b175ee826dd584785a86ad165e86489c2ecaf6139356b216682416e0f2278029`.
It returned **PASS**: TD-006 now requires T9A's platform-resolved final-image
extraction and strict-loader byte agreement to set D4 `b`/`n`, rejects the Go
runner as a production-constant source, confines T9 to the five-run envelope,
and leaves deployed no-override mount proof with Delivery. No provider,
deployment, certification, or proof-policy claim changed. Changes after that
candidate are this review receipt and status closure only.

## Bidirectional closure

| Accepted surface | Final disposition |
| --- | --- |
| R1 exact shared subset and provider-neutral surface | TD-001, TD-007 through TD-012, TD-017, TD-018 |
| R2 key/feature ownership and no adapter policy invention | TD-001, the exact-provider key cases in TD-017/018, and first-adopter obligation TD-019 |
| R3 static credential and fixed authority boundary | TD-002, TD-003, TD-015, TD-017, TD-018 |
| R4 admission, request concurrency, deadlines, streaming, source/body lifetime, and memory | TD-004 through TD-006 and TD-010 |
| R5 single/multipart intent, CRC64NVME integrity, and cleanup | TD-007 through TD-010 plus each exact-provider receipt |
| R6 portable metadata and absence | TD-010, TD-011, TD-017, TD-018 |
| R7 unversioned delete meaning | TD-011, Amazon versioning preflight in TD-017, R2 evidence in TD-018 |
| R8 bounded secret presigned GET issuance | TD-012, TD-013, TD-017, TD-018 |
| R9 one-attempt error and ambiguity policy | TD-003, TD-005, TD-009, TD-011 |
| R10 startup/readiness/shutdown | TD-002 and TD-014 |
| R11 bounded secret telemetry | TD-013 plus sanitized conformance receipts |
| R12 profile/config/dependency pruning and selector independence | TD-015 and TD-016 |
| Technical Design reopen conditions | Every row names the exact Technical Design or Specification return; SDK/Go/dependency drift additionally reopens TD-003, TD-006 through TD-012, TD-017, and TD-018. |

One provider passing never covers the other. Planning may choose task order and
acceptance-unit placement only; it must preserve these scenarios, fixtures,
oracles, commands, claim limits, and reopen routes without inventing a new
proof policy.

## Planning reopen handoff

```text
Reopen the S3-Compatible Object Storage capability in Planning only for
/Users/daniil/Projects/Opensource/go-service-template-rest.

Test Design is review-cleared for D4's bounded image-owned public-root snapshot.
Start from
specs/s3-compatible-object-storage/spec.md,
specs/s3-compatible-object-storage/design/overview.md, and
specs/s3-compatible-object-storage/test-plan.md at their recorded SHA-256
identities, plus the canonical T9 blocker in
specs/s3-compatible-object-storage/tasks.md. Preserve every accepted T1-T8
receipt and TD-001, TD-004, TD-007 through TD-013, and TD-019 unchanged.

First, reconcile TD-002, TD-003, TD-005, TD-006, and TD-014 through TD-018 into
the smallest dependency-ordered prerequisite repair before T9 and refresh only
the invalidated T9-T12 owners, commands, external inputs, and reopen conditions.
The ledger must require strict image-bundle loading, non-nil caller-owned roots,
the 448 KiB/288-root arithmetic/source receipt, and the same-preconstruction-
baseline five-run Linux construction/idle/admitted envelope before T9 can
resume. Keep Amazon and R2 receipts separate and gated by their own provider,
production-image, and no-override inputs. Preserve the canonical T9 blocker
until Planning makes the repair route executable; do not enter Implementation,
mutate a provider, deploy, certify support, or change TD-019 in this session.
```
