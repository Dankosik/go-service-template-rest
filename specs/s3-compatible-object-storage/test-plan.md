# Test design — S3-compatible object storage

status: ready

Accepted authorities:

- [`spec.md`](spec.md), SHA-256 `e3f61138d2eb27cb19a35b32020d59c978a18a3686e90e75d8933104ebb1e43c`;
- [`design/overview.md`](design/overview.md), SHA-256 `844eb16dafd7a4ffe357820e1d7d01211e91ed36c3cfdabd102b14c30b6496f9`.

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
  holds the real SDK over the accepted fresh HTTP/1 TLS transport. The process
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
| **TD-002** — R3/R4/R10, D1/D3/D4, exact construction and image-root snapshot | Strengthen config/client/bootstrap tables and add the strict image-root loader table. Wrong behavior: an invalid tuple or bound starts; ambient AWS or root state changes signed/trust policy; credentials refresh or perform I/O; construction performs DNS/provider I/O; a missing, non-regular, unreadable, empty, oversized, malformed, header-bearing, wrong-type, duplicate, non-CA, zero-root, or over-288-root bundle is accepted or falls back; the successful S3 client has nil `RootCAs`; or an invalid/over-1,024-byte content type reaches admission/signing. | Table both exact providers and every missing/inconsistent authority, credential, part-count, duration, header/body, memory, and image-bundle condition; set hostile AWS region/profile/endpoint/retry/logger/proxy plus `SSL_CERT_FILE`, `SSL_CERT_DIR`, and fallback-root canaries. Through the unexported source seam, require one bounded read of at most `448 KiB + 1`, complete PEM consumption, individual `x509.ParseCertificate`, unique valid-CA counting, and no fallback. Exercise content type at empty, 1, 1,024, 1,025, and invalid HTTP field-value boundaries before a counted body/transport. Construct with a transport that fails on any call. The oracle is `invalid` or startup rejection before body/provider effect for invalid cases; for valid cases the immutable direct `s3.Options` uses the supplied static tuple, `aws.NopRetryer`, required checksum modes, and a non-nil adapter-private root pool while DNS/provider/credential calls and credential workers stay zero. | Unit/component: `go test -vet=off ./internal/infra/s3 ./internal/config ./cmd/service/internal/bootstrap -run '^(TestConfigRejectsInvalidTupleAndEnvelope|TestImageRootBundleLoaderIsStrictAndBounded|TestUploadRejectsUnboundedContentType|TestNewUsesStaticConfigurationAndImageRootsWithoutNetworkIO|TestObjectStorageStartupLoadsImageRootsLocally)$' -count=1` | Synthetic Amazon/R2 origins, dotless buckets, static canary credentials, hostile ambient variables; decision-neutral generated CA bundles at 0/1/288/289 roots and `448 KiB`/`448 KiB+1`; no live endpoint. | `config_test.go`, `image_root_bundle_test.go`, `upload_test.go`, `client_test.go`, object config tests, and `startup_object_storage_test.go`; land the trust-root prerequisite before T9 arithmetic/process proof. Reopen Technical Design if direct construction cannot isolate ambient SDK/root policy, strictly bound the fixed image file, or bound request fields; Specification for another credential or trust class. |
| **TD-003** — R3/R9, D2/D4, final authority, explicit roots, and one attempt | Strengthen `httpclient` and adapter transport proof. Wrong behavior: redirect/proxy/private resolution/SDK endpoint rewriting receives signed material; a lost response is replayed; HTTP/2 or compression changes bytes; a control/object body escapes its cap; S3 installs nil roots, accepts an ambient-only chain, changes `ServerName`, disables verification, or mutates the caller pool; or adding `RootCAs` changes existing nil-root callers. | Use the current authority/DNS/redirect fixtures. A real HTTP/1 server first completes a warm-up request, then reads and drops a replayable idempotent request before responding; one-attempt mode must use a fresh second connection and create no third connection/request. A second case returns gzip bytes and asserts no `Accept-Encoding` plus exact wire bytes. For trust, generate distinct configured and ambient CAs and clone the configured pool before construction. The exact-host server chaining to the configured pool must succeed, while a server chaining only to hostile `SSL_CERT_FILE`/`SSL_CERT_DIR`/fallback canaries or presenting the wrong hostname must fail before the fixture records a request. Inspect the applied transport for the same non-nil pool pointer, exact validated hostname, and `InsecureSkipVerify=false`; after construction, the successful handshake, and both denial cases, require the caller pool to remain `Equal` to its preconstruction clone so an in-place added root cannot false-pass. Separately prove nil preserves current non-S3 caller behavior. Script alternate authority and oversized control/object responses; all fail before the forbidden target/effect or after the exact cap and close the body. | Transport/component: `go test -vet=off ./internal/infra/httpclient ./internal/infra/s3 -run '^(TestOneAttemptTransportDoesNotReplayOrTransform|TestTransportUsesCallerRootCAsWithoutAmbientFallback|TestTransportRefusesAlternateAuthority|TestTransportBoundsControlAndObjectBodies)$' -count=1` | Existing `httptest`/dial override patterns, a preconstruction `CertPool.Clone`, two generated CA hierarchies, hostile ambient-root files/directories, and scripted SDK HTTP client; no provider. | `httpclient/client_test.go`, `s3/image_root_bundle_test.go`, and `s3/transport_test.go`; land with D2/trust prerequisite before T9. Reopen Technical Design if fresh HTTP/1 can replay, non-nil roots can still consult/mutate the system pool, or conformance requires HTTP/2; Specification for private/custom roots. |
| **TD-004** — R4 process admission and request concurrency | New deterministic concurrent component test. Wrong behavior: any operation bypasses capacity, capacity queues, reads an upload body, signs, or starts I/O; multipart/cleanup overlaps requests; a leaked body escapes the global ceiling. | Configure admission `A=2` and hold both tokens with channel-blocked operations. At saturation invoke upload, download, metadata, delete, and presign in turn; every call must return `busy` before reader access, signing, or transport. Release each admitted operation through success, error, download EOF, and Close; assert a new call is admitted. Multipart scripts reject any overlap and cleanup starts only after the failed stage returns. Race and `goleak.VerifyNone` cover token/goroutine ownership. | Component/race: `go test -vet=off ./internal/infra/s3 -run '^TestAdmissionIsProcessWideAndNonBlocking$' -count=1`; `go test -vet=off -race ./internal/infra/s3 -run '^TestAdmissionIsProcessWideAndNonBlocking$' -count=10` | Scripted transport/signing barriers and counted reader; `A=2` is a test fixture, not a runtime default. | `client_test.go` plus operation owners; land before body-specific tests. Reopen Technical Design if any selected SDK path adds a worker, queue, or overlapping request. |
| **TD-005** — R4/R9, D2/D3/D4, one deadline and body ownership | Strengthen the phase table using the real adapter and controlled blockers. Wrong behavior: a caller or maximum deadline is ignored; DNS/connect/TLS verification escapes that context; cleanup gets a fresh budget; cancellation starts another part/attempt; the adapter closes an upload source; a download retains admission after terminal read/Close; or the TLS case false-passes because trust failed before the intended cancellation barrier. | Under `synctest`, capture the effective context at adapter request write, response wait, body read, multipart part, Abort, ListParts, and presign before/after signing. In `httpclient`, an overridden dial path captures the same request context for DNS/connect and a TLS peer whose exact hostname and chain first succeed against the explicit test-only non-nil pool, then blocks the handshake until cancellation; a distinct ambient-only CA remains untrusted. Advance the earlier caller/configured deadline or cancel; assert the matching stable result on non-mutating/proved-unsent work, no new stage, the same deadline in cleanup, upload source still open, and idempotent download release on EOF/error/Close. For possibly sent mutations, assert `outcome_unknown` outranks context error. | Unit/component: `go test -vet=off ./internal/infra/httpclient ./internal/infra/s3 -run '^(TestOneAttemptTransportUsesRequestDeadlineAndExplicitRoots|TestEffectiveDeadlineAndLifecycleOwnEveryPhase)$' -count=1` | Owned channels, tracked readers/bodies, fake-clock bubble, configured and ambient generated CAs, exact-host loopback TLS, and an uncancelled trust-control case; elapsed time is diagnostic. | `httpclient/client_test.go`, `s3/client_test.go`, `upload_test.go`, `download_test.go`, and `errors_test.go`; land the explicit-root transport prerequisite before T9. Reopen Technical Design if an SDK/transport/TLS stage loses caller context or explicit roots; Specification if error precedence or trust class changes. |
| **TD-006** — R4 success 4, D2/D4 bounded image-root and retained-memory envelope | Replace the superseded arithmetic proof with an exact checked table, independently reviewed pinned-source/image receipt, and real-transport Linux process envelope. Wrong behavior: any add/multiply/ceil/align/conversion/max wraps, clamps, or saturates; equality/one-byte-short is wrong; `B=448 KiB`, `N=288`, `H`, and `E` drive the wrong terms or become coupled; the image input or accepted-root count exceeds its half-ceiling; the S3 path reaches nil/system roots; a live allocation has no one D4 receipt class or finite driver; `F`, `S`, `U`, `trust_shared`, `trust_startup`, or `trust_verify` lacks 100% source-derived headroom; construction, retained pool, root-driven verification, descriptors, pointees/backing bytes, upload ID, configured strings, parser state, original plus escaped Complete XML capacity, TLS/stack/shared state, or a worker is omitted; fixture/ambient-root memory contaminates the child; a small/padded CA fixture stands in for retained maximum roots; RSS exceeds the equation; or cancellation leaves an owned resource even when Go retains free pages. | Unit cases independently calculate D4's `heap`, `U`, `Q`, `trust_shared(B,N)`, `trust_startup(B,N)`, `trust_verify(N)`, `retained_parts`, `complete_xml_len`, `complete_xml`, `upload_session`, `construction`, `shared`, every operation class, and `required_memory=max(construction, shared+A*max(...))`. Cover positive minimums; exact equality/one-byte-short; `P=1/10,000/10,001`; `B=448 KiB`/`B+1`; `N=288`/`N+1`; actual `b<=B/2` and `n<=N/2` with unused bytes/count/percent; `H<E`, `H=E`, `H>E`; content type `0/1/1,024/1,025`; the 32 KiB heap branch and 8 KiB alignment edge; every trust/add/multiply/align/ceil/conversion/`Q`/`A*max` overflow; and a wrong-architecture shallow receipt. Before constants are accepted, a read-only receipt resolves `GOOS/GOARCH`, Go 1.26.5, AWS core `v1.43.5`, credentials `v1.19.5`, S3 `v1.107.1`, Smithy `v1.27.7`, module sums, the nine complete Go source SHA-256 identities fixed by D2/D4, and the pinned T9 image bundle path/hash/bytes/unique-valid-CA count. Apply D4's seven allocation classes to every reachable allocation site with compiler escape/stack evidence, exact shallow/rounded sizes, named barrier counts, and symbolic live-copy counts. Prove `used_F<=F/2`, `used_S<=S/2`, `used_U(x)<=32*heap(x)+32 KiB`, and each used trust term at most its design half for zero, allocator breakpoints, overflow, `B`, `N`, and admitted `Q`, `Q+K`, `Q+K+T`, `H`, and `E`; record used/unused bytes and percentage per row. Unknown, duplicate, reclassified, non-finite, wrong-driver, source/image mismatch, or half-ceiling failure reopens D4; T9 cannot tune a reserve. An unmeasured controller/TLS-fixture PID binds the test authority and launches five identical `GOMAXPROCS=1` children containing only runtime, adapter, real SDK, production fresh HTTP/1 TLS client transport with non-nil adapter-private roots, and a pre-opened quiescent scalar IPC channel. Every child records one preconstruction `smaps_rollup:Rss` baseline after runtime/IPC warmup, then uses that same baseline for held trust-construction, idle-adapter, and every admitted-operation delta. The trust-construction barrier keeps the complete `B`-driven PEM/decode input and parsed `N`-root pool live; idle keeps only the immutable pool; admitted barriers hold `A=2` real TLS verifications for every formula class. The generated bundle contains `N` unique valid CAs with retained DER/parsed input driven to `B`; padding that only enlarges discarded PEM is rejected. One fixture chain reaches a selected root; a distinct hostile ambient-only chain must fail before fixture request count changes. Complete holds `2*10,000` source-equivalent descriptors with escape-amplifying `H`-bounded ETags while originals and XML capacity are live; response cases maximize `H` and `E` independently; object/chunk variants prove no retained object/part buffer. After controlled GC only where it cannot erase the held state, each child computes the maximum non-negative same-baseline construction/idle/admitted RSS delta; the five-run maximum must be at most both independently reviewed `required_memory` and the configured ceiling. Verify each complete same-snapshot `smaps_rollup` resident-category equation; record nearby `VmRSS`, Go live-heap/stack/os-stack, and goroutine samples only as diagnostics. Cancellation must join every child call; child active/admission, response-body, and client-connection counts must be zero; child `goleak` against its post-construction idle baseline must find no new adapter/SDK/HTTP/TLS-client/control goroutine. Scalar IPC separately proves fixture request/body/accepted-connection/goroutine counts zero before controller joins it. No prewarmed nil-root pool, small one-root bundle, system-root subtraction, fixture-PID allocation, `VmRSS`/Go-mapped-memory substitution, cross-barrier baseline, or return-to-idle RSS is admissible. | Structural/process: `go test -vet=off ./internal/infra/s3 -run '^(TestWorkingMemoryAccounting|TestImageRootBundleLoaderIsStrictAndBounded|TestUploadRejectsUnboundedContentType)$' -count=1`; Linux real-transport gate: `GOMAXPROCS=1 make test-s3-envelope` | Test-only `A=2`, maxima `B/N/K/P/T`, independently maximal bounded `H/E`, actual configured-string lengths `Q`, unique generated CA DER plus exact-host fixture, hostile ambient-root canaries, pinned source/module/image receipts, exact build target, separate controller/fixture and measured-child PIDs, fixed-size scalar IPC, Docker Linux `/proc`, owned counters, and goroutine profiles; no provider endpoint or credential. The independently reviewed source/image receipt is mandatory implementation proof input and is never derived by calibrating production constants. | `config_test.go`, `image_root_bundle_test.go`, `upload_test.go`, `client_test.go`, and the Make proof entrypoint; trust-root prerequisite and separate envelope acceptance must close before live conformance. Reopen D4 on any unknown reserve, source/build/image/bundle/mount-policy drift, nil/system-root reachability, unstable/missing `smaps_rollup`, fixture contamination, unexplained child RSS, nonzero owned resource, leaked worker, coefficient/formula/classification change, trust half-ceiling failure, or inability to hold the ceiling; Docker Linux availability makes unsupported-runner disposition inapplicable to this candidate. |
| **TD-007** — R5 single upload, D4/D5 bounded streaming CRC64NVME | New real-SDK scripted wire falsifier. Wrong behavior: the body is buffered/pre-read, byte `length+1` is consumed, short input succeeds, threshold/intent drifts, checksum is absent/wrong, content type escapes its D4 boundary, or create-only overwrites. | A read-gated non-seekable source proves no read before admission. First apply TD-002's content-type byte/HTTP-field boundary. Decode the one `aws-chunked` request independently, compute CRC64/NVME with stdlib from the declared bytes, and require exact content length, trailer, algorithm/type, accepted optional content type, and `If-None-Match:*` only for qualifying create-only. Return matching, missing, and corrupt checksum fields; assert only the match succeeds. Short input fails; an extra sentinel remains unread; at/above threshold selects the accepted path exactly. | Component: `go test -vet=off ./internal/infra/s3 -run '^(TestUploadRejectsUnboundedContentType|TestSingleUploadStreamsCRC64NVMEAndExactLength)$' -count=1` | Known-answer bytes and polynomial `0x9a6c9329ac4bc9b5`; boundary content types; scripted response metadata; no provider. | `checksum_test.go` and `upload_test.go`; land together. Reopen Technical Design if SDK framing/metadata or the request-field bound differs; Specification if common integrity or create-only semantics cannot hold. |
| **TD-008** — R5 multipart path, D5 | New serial multipart state-machine test. Wrong behavior: create-only enters multipart, parts overlap or size/order drifts, whole checksum covers different bytes, completion treats embedded XML error/initial 2xx as success, or retained completion inputs are unbounded. | Use replace at `C+1`, exact multiples, final remainder, and provider part-count edge. Script Create, each serial UploadPart, and Complete; independently hash the concatenated limited readers and compare every part plus final CRC64NVME/FULL_OBJECT, ordered part/ETag/checksum list, and declared object size. Inject missing part checksum, out-of-order receipt, corrupt final checksum, and embedded completion error; each fails and routes to cleanup. | Component: `go test -vet=off ./internal/infra/s3 -run '^TestMultipartUploadIsSerialAndConfirmsWholeChecksum$' -count=1` | Small provider-valid part fixture for wire behavior; maximum part-count retention is TD-006. | `checksum_test.go` and `upload_test.go`; same implementation unit as TD-007. Reopen Technical Design on SDK checksum/completion drift; Specification if multipart needs a provider escape. |
| **TD-009** — R5 multipart cleanup, D6 | New bounded cleanup table. Wrong behavior: a new part starts after failure, Abort/ListParts retries or detaches, `complete` is guessed from status, a page/body remains unread, or `pending` is hidden. | Fail after upload ID at create/part/complete boundaries. Count exactly one Abort and bounded paginated ListParts under the same remaining context. Tuple tables admit `complete` only for the conformance-approved empty/terminal observation; non-empty, truncated, malformed, timeout, cancel, abort/list failure, or unknown R2 observation must be `pending`. Assert primary error is preserved, no upload ID escapes, all bodies close, and no background call appears after return. | Component/race: `go test -vet=off ./internal/infra/s3 -run '^TestMultipartCleanupIsBoundedAndConservative$' -count=1`; `go test -vet=off -race ./internal/infra/s3 -run '^TestMultipartCleanupIsBoundedAndConservative$' -count=10` | Scripted pages and tuple evidence table initially closed to only design-admitted observations; provider additions require TD-017/018 receipts. | `upload_test.go`; land with cleanup mapper. Reopen Technical Design on a failed terminal observation/mechanism; Specification if neither tuple can expose the required cleanup outcome. |
| **TD-010** — R4/R5/R6 download completion and body lifecycle | New streaming-body table. Wrong behavior: headers count as success, a missing/composite checksum body is exposed, overflow leaks byte `O+1`, mismatch is missed before/at EOF, deferred errors restart, or Close/cancel/deadline leaks admission/body. | Script declared size valid/oversized/absent, installed validator metadata valid/missing/composite, matching/mismatching payload, exact limit/overflow, terminal error, early Close, cancellation, and deadline. The oracle is no returned body for invalid metadata/declared size; no byte beyond `O`; success bytes counted only at clean validated EOF; exact stable read error otherwise; one underlying Close and one token release for EOF/error/Close races; no second GET. | Component/race: `go test -vet=off ./internal/infra/s3 -run '^TestDownloadCompletesOnlyAtValidatedEOF$' -count=1`; `go test -vet=off -race ./internal/infra/s3 -run '^TestDownloadCompletesOnlyAtValidatedEOF$' -count=10` | Scripted SDK checksum metadata/body and tracked closer; no provider. | `download_test.go`; land after checksum/client base. Reopen Technical Design if SDK cannot expose installed validator/EOF failure; Specification if integrity meaning changes. |
| **TD-011** — R6/R7/R9 metadata, delete, and conservative errors | New exact mapping tables through the real SDK deserializer. Wrong behavior: metadata exposes provider identity fields, 403 becomes absence, arbitrary 404 becomes `not_found`, delete claims erasure, possible-send mutation becomes `temporary`/context error, malformed XML/provider text leaks, or an SDK stage retries. | Script Head/Delete success, tuple-admitted absence, concealed absence, 412 outside/inside single create-only, throttling, malformed/oversized XML, request-write loss, and after-write response loss. Assert portable fields only, UTC time, closed kind, sanitized error, exact private diagnostic vocabulary, and stage request count one. Delete absent success remains operation completion only; possibly sent Put/Complete/Delete is `outcome_unknown`. | Component: `go test -vet=off ./internal/infra/s3 -run '^(TestMetadataAndDeleteExposePortableResults|TestErrorMappingIsConservativeAndOneAttempt)$' -count=1` | Scripted Smithy status/code/request-ID corpus including secret canaries; tuple-specific absence admissions remain gated by TD-017/018. | `metadata_test.go`, `delete_test.go`, `errors_test.go`; land with mapper. Reopen Technical Design for SDK/Smithy/send-phase drift; Specification for changed portable absence/delete/error semantics. |
| **TD-012** — R8 presigned GET | New local-signing table plus redaction checks. Wrong behavior: another method/authority/key/TTL is signed, returned expiry disagrees with the signed credential, a default TTL appears, returned headers drift, ambient config changes signing, or bearer material enters evidence. | With fixed credentials/key, capture times immediately before and after each call for `0`, `1s`, configured max, max+1, and seven-day boundary. Assert exact GET, bucket authority, decoded key, query-bearing SigV4 fields, and exact required headers. Independently parse second-precision `X-Amz-Date` and integer `X-Amz-Expires`; `ExpiresAt` must equal their UTC sum exactly, while the before/after window proves only that the signing instant belongs to this call. No production clock seam is added. Feed URL/query/headers through error/telemetry collectors and require canaries absent. No HTTP call occurs and issuance never emits transfer success. Live method/authority/key/query/header signature rejection belongs separately to TD-017/018. | Unit/component: `go test -vet=off ./internal/infra/s3 -run '^TestPresignGETIsBoundedAndSecret$' -count=1` | Fixed test credential/key and parsed signed-query expiry; no live URL or clock abstraction. | `presign_test.go`; land with presigner. Reopen Technical Design on presigner output drift; Specification for another operation/lifetime/recipient guarantee. |
| **TD-013** — R11 telemetry secrecy and bounded diagnostics | New reachable operation/outcome recording matrix plus whole-adapter secrecy canary. Wrong behavior: a required signal/value is missing, a secret/object/provider value appears in evidence, or labels become input-driven. | Drive only reachable pairs: upload `{success,invalid,too_large,busy,precondition_failed,denied,integrity_failed,cancelled,deadline_exceeded,outcome_unknown,internal}` with single/multipart and `none|complete|pending`; download `{success,invalid,too_large,busy,not_found,denied,temporary,integrity_failed,cancelled,deadline_exceeded,internal}`; metadata `{success,invalid,busy,not_found,denied,temporary,cancelled,deadline_exceeded,internal}`; delete `{success,invalid,busy,denied,cancelled,deadline_exceeded,outcome_unknown,internal}` with context kinds only on proved-unsent cases; presign `{success,invalid,busy,cancelled,deadline_exceeded,internal}`. For each row assert the exact operation-duration/result record, active `0->1->0` lifetime when admitted, admitted/rejected count, failure phase, and absence of impossible pairs. Assert completed upload/download bytes only on full success, integrity count only on integrity failure, exact transfer path, exact cleanup disposition, and presign issuance without transfer signal. Separately inject canaries in every forbidden field and require none in logs/spans/metrics/errors; only sanitized request ID may appear as a non-metric field. Enumerate the closed label product with one `unknown` fallback. | Component: `go test -vet=off ./internal/infra/s3 -run '^TestTelemetryContractIsBoundedAndSecret$' -count=1` | Existing recording OTel/log patterns; scripted reachable outcomes and synthetic canary corpus; no exporter/provider. | `telemetry_test.go`; land after all result paths so it is exhaustive. Reopen Specification for a new result/operator question; Technical Design if a required safe signal lacks an owner. |
| **TD-014** — R10 lifecycle/readiness, D1/D3/D4 bootstrap trust snapshot | Strengthen construction/outage/readiness and lifecycle-order proof. Wrong behavior: strict image-bundle failure reaches ready; startup probes storage; trust loading happens before memory-limit publication or after client publication; a post-start file/provider change reloads roots, changes readiness/liveness, or registers a probe/restart path; close order drifts; active work gets a longer shutdown budget; or idle transport closes twice observably. | Bootstrap through the unexported bounded bundle source with a valid exact-host CA and a transport that records every request. Invalid bundle cases must fail startup before readiness and before DNS/provider calls. Valid construction performs only the bounded local trust-file read, publishes one immutable pool after the memory-limit event, records zero provider calls and no storage readiness probe, and does not reread the source after publication. Capture readiness/liveness and probe inventory, mutate/remove the test source, then make one admitted operation fail as provider unavailable and a later operation succeed against the already captured pool; only operation error/telemetry may change. Inject ordered lifecycle events: early return closes once; normal shutdown closes object runtime after HTTP drain/background join and before dependencies/telemetry flush; repeated safety close is a no-op. Cancel active upload/download and assert no detached request or cleanup. | Component/lifecycle: `go test -vet=off ./cmd/service/internal/bootstrap -run '^(TestObjectStorageStartupLoadsImageRootsLocally|TestObjectStorageStartupAndOutageDoNotChangeReadiness|TestObjectStorageRuntimeCloseOrder)$' -count=1`; `go test -vet=off -race ./cmd/service/internal/bootstrap -run '^TestObjectStorageRuntimeCloseOrder$' -count=10` | Existing health/bootstrap fakes, ordered event recorder, counted trust source, exact-host generated CA, scripted provider failure, and synctest lifecycle pattern; no production path or provider. | `startup_object_storage_test.go` and `run_lifecycle_test.go`; land the trust-root prerequisite before T9. Reopen Technical Design if snapshot ownership, construction point, or lifecycle order cannot hold; Specification if storage becomes readiness-critical or roots require reload. |
| **TD-015** — R3/R12 typed config, secret sources, and no CA setting | Strengthen config inventory/source tests. Wrong behavior: object settings survive `none`; empty/unknown fields are ignored; a secret comes from YAML/defaults; an access-key ID is logged; a selected output gains a usable endpoint/bucket/credential/default bound; or any YAML/environment/CLI leaf selects, augments, replaces, or reloads S3 trust roots. | Compare exact leaf/default/snapshot inventories for absent and selected profile. Load empty placeholders, valid environment-only credential canaries, non-empty file values for all three static credential fields, and plausible CA path/content/reload keys plus `SSL_CERT_FILE`/`SSL_CERT_DIR`. The oracle is unknown-key rejection when absent, fail-closed required finite values when selected, empty non-secret examples only, environment credential acceptance, file-source credential rejection without raw value, no ambient AWS leaf, and no production CA/trust configuration leaf or secret-policy exception. | Unit: `go test -vet=off ./internal/config -run '^(TestObjectStorageConfigContract|TestSnapshotContract|TestStaticCredentialSourcePolicy)$' -count=1` | Existing configtest/source-policy harness, synthetic values, and hostile root-key names only. | Object config, snapshot, and secret-policy tests; land before profile matrix. Reopen Specification for another credential/config/trust source; Go Ownership if section placement changes. |
| **TD-016** — R12 deterministic profile, trust-owner inventory, and dependency pruning | Strengthen the independent template-init oracle. Wrong behavior: `none` retains object code/config/docs/tests/deps; `s3` loses `image_root_bundle.go`/test or the generic non-nil-`RootCAs` `httpclient` branch; a generated service gains a production CA setting or bundle copy; a rejected client/transfer manager remains; selectors remove each other; repeat mutates; unknown/empty mutates; markers/generator sources survive; or lock/output disagree. | Generate all four `OBJECT_STORAGE=none|s3` × `OUTBOUND_HTTP=none|bounded` combinations, with authn-only controls for the three-way `httpclient` predicate. Compare independent present/absent inventories including the S3 strict image-root owners and shared code-only `RootCAs`; compile/test each retained output, run `go mod tidy`, inspect exact module closure, require no new bundle/config source and no custom/private-root path, verify marker/generator absence, `template.lock`, completion output, unknown config rejection, byte-identical equal repeat, and unchanged incompatible repeat. The existing runtime-image bundle remains image-owned rather than copied by the generator. | Profile/structural: `TEMPLATE_INIT_PROFILE=object-storage make template-init-check`; `make project-structure-check`; `make mod-tidy-check` | Existing checkout-copy/init harness extended with exact trust-owner inventories; credential- and provider-free. | `scripts/ci/template-init-check.sh` and profile owners; one profile acceptance unit after the trust-root prerequisite and T9. Reopen Go Ownership for inventory/retention placement; Technical Design if profile generation must own/copy roots; Specification if selector or trust-source semantics change. |
| **TD-017** — Amazon S3 exact-tuple conformance and production public-chain receipt | New required credentialed target; no other provider result is reusable. Wrong behavior: the selected commercial regional general-purpose tuple rejects a governed field/path, hides required integrity, retries, misclassifies absence, overwrites create-only, leaves unreported parts, or produces invalid/restricted presign behavior; the exact hostname does not validate through the adapter-private production public-root snapshot; or the receipt silently uses nil/system/custom roots or a replaceable bundle. | Preflight the exact endpoint/region/bucket/virtual authority, static identity-policy receipt, never-enabled versioning observation, and the production-image trust receipt: pinned architecture image digest, fixed `/etc/ssl/certs/ca-certificates.crt` path, upstream provenance/revision, SHA-256, bytes `<=448 KiB/2`, unique valid roots `<=288/2`, regular read-only image ownership, and no replacing runtime mount. Build the adapter with that exact strict-loaded non-nil pool; require successful chain and hostname validation for the Amazon authority and failure for a wrong hostname/ambient-only CA before any signed request. Under `conformance/amazon_s3/<run-id>/`, exercise the existing port matrix, independent CRC/attempt/cleanup oracles, and registered prefix-only cleanup with final empty readback. The provider receipt records the non-secret tuple plus exact trust/image/checksum/cleanup/identity-policy facts; any missing, stale, or ambiguous field fails Amazon only. | Credentialed exact-provider: `REQUIRE_S3_CONFORMANCE=1 make test-s3-conformance-amazon` | Required external inputs: Amazon endpoint, region, dotless bucket, unique run ID, primary and concealment static credential sets, identity-policy receipt, never-enabled versioning evidence, abandoned-multipart lifecycle backstop, and the exact production-image/bundle/no-override receipt. Unavailable until an authorized run; secrets remain environment-only and unrecorded. | `test/s3_object_storage_conformance_integration_test.go` Amazon entrypoint and Make target; separate support-certification unit after deterministic T9/T10 proof. Test-only bucket/readback operations never enter production. Reopen D2/D4 Technical Design on bundle/public-chain/hostname, source, trailer/checksum, one-attempt, or cleanup mechanism failure; Specification only under the design reopen conditions. |
| **TD-018** — Cloudflare R2 exact-tuple conformance and production public-chain receipt | New required credentialed target; Amazon evidence is inadmissible. Wrong behavior: the `auto` account endpoint rejects the AWS SDK trailer/fields, returns incompatible checksum/absence/cleanup semantics, silently changes keys, or presign works only through an excluded custom domain; the exact R2 hostname does not validate through the adapter-private production public-root snapshot; or the receipt substitutes Amazon, nil/system/custom roots, or a replaceable bundle. | Preflight the exact account S3 endpoint, region `auto`, dotless bucket, virtual authority, static identity-policy receipt, R2 versioning/lifecycle facts, and the same production-image trust receipt fields required by TD-017. Build the adapter with that exact strict-loaded non-nil pool; require successful chain and hostname validation for the R2 authority and failure for a wrong hostname/ambient-only CA before any signed request. Run the same port-level matrix and independent CRC/attempt/cleanup oracles as TD-017 under `conformance/cloudflare_r2/<run-id>/`, using R2 credentials and no custom domain. Cleanup is `complete` only for a TD-018-proven terminal observation; otherwise report `pending`; registered direct cleanup removes only owned objects/uploads and requires empty readback. The distinct receipt records R2 and trust/image results only. | Credentialed exact-provider: `REQUIRE_S3_CONFORMANCE=1 make test-s3-conformance-r2` | Required external inputs: R2 account endpoint, dotless bucket, unique run ID, primary and concealment static credential sets, identity-policy receipt, lifecycle/versioning evidence, abandoned-multipart lifecycle backstop, and the exact production-image/bundle/no-override receipt. Unavailable until an authorized run; secrets remain environment-only and unrecorded. | Same integration file, R2 entrypoint and distinct Make target; separate support-certification unit after deterministic T9/T10 proof. Test-only bucket/readback operations never enter production. Reopen D2/D4 Technical Design on bundle/public-chain/hostname, source, trailer/checksum, one-attempt, or cleanup mechanism failure; Specification only under the design reopen conditions. |
| **TD-019** — R2 feature authorization/content/retention ownership | First-adopter placement obligation; no template test is invented without a feature owner. Wrong behavior: a feature denial still reaches the store, or the adapter starts authorizing principals, constructing tenant keys, or deciding content/retention policy. | At the first real feature composition, drive denied principal/operation/key/content/size/retention/overwrite cases through that feature with a counted `objectstorage.Store` fake. The oracle is the feature's own stable denial and zero store calls; allowed cases pass the exact feature-produced key/intent unchanged. | Exact future command is feature-owned and unavailable until an adopter/package exists; Planning records a scope exit rather than inventing a reference feature. | No current adopter is accepted by Technical Design. This input is outside the template implementation and becomes mandatory before an adopter claims the capability. | Future feature package test owner. Reopen Specification if any policy moves into the pack; Go Ownership when the first adopter fixes its composition point. |

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
reference for the reviewed Linux platform, build and materialize the final
unmounted image, and receipt the Dockerfile identity, platform, index and
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
   — focused **FAIL** only on presign expiry: a plausible returned `ExpiresAt`
   could disagree with the signed query.
4. Root repair made `ExpiresAt` equal parsed `X-Amz-Date + X-Amz-Expires` and
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
