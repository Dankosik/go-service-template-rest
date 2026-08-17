# S3-compatible object storage implementation ledger

status: done

Completion: T1-T10 are the accepted original deterministic implementation. T13,
T14, T15, and T16 are accepted audit repairs. Amazon S3 and Cloudflare R2
certification remain separate optional adopter-owned handoffs, each unverified
until its own exact-tuple external receipt.

Stop: T16 is accepted; stop without provider contact. A future certification
attempt requires the matching H11 or H12 external authority and inputs; without
them its provider claim remains unverified.

Global constraints:

- T1-T10, T9A, and T13-T16 are credential-free and make no provider request. Credential values are environment-only; no selected output contains a usable endpoint, bucket, credential, or finite workload default.
- One process fixes one exact provider, HTTPS endpoint, signing region, dotless bucket, virtual-hosted authority, explicit credential snapshot, expected-owner policy, checksum policy, strict bounded image-owned public-root snapshot, and finite resource envelope. Amazon requires temporary session credentials; R2 admits its documented static or temporary credentials. There is no ambient SDK/root policy, credential refresh, production CA setting or reload, provider fallback, queue, background cleanup worker, readiness probe, public API, bucket provisioner, or feature-policy owner. Only HeadObject and pre-body GetObject have the finite three-attempt retry policy fixed by R4/R9.
- Every task preserves the provider-neutral five-operation port, feature-owned authorization/key/content/retention/overwrite policy, exact-length streaming, one adapter-wide non-blocking admission bound, one effective context, serial multipart, conservative mutation ambiguity, bounded secret telemetry, and the profile independence fixed by R1-R12.
- Canonical source, its tests/fixtures, profile markers, generated inventory, dependencies, operator docs, and replacement cleanup remain in the task that makes their postcondition true. Rejected clients, transfer managers, stale markers, generator-only sources, and object-owned outputs are absent where R12 requires them.
- Deterministic tests prove only their exercised repository surface; T9 proves only the pinned Linux process envelope; H11 and H12 can prove only their recorded exact provider tuple after separate authorization. Deployment identity, DNS/TLS/egress, bucket policy, lifecycle backstop, quotas, telemetry delivery, and encryption still require a separate authorized deployed-path canary.
- Reopen Technical Design before implementation chooses another mechanism when fixed authority, zero replay, deadline propagation, streamed checksum/EOF validation, bounded retained memory, serial multipart, or immediate cleanup cannot hold; when the source reserve has an unknown term; when exact-provider evidence breaks the selected mechanism but a behaviorally equivalent mechanism may exist; or when an SDK, Smithy, Go, checksum, transport, endpoint-resolution, credential, retry, signing, presign, buffer/goroutine, embedded-error, or error-shape update changes the pinned evidence. Reopen Specification when a provider-specific feature field/result is needed; portable key, credential, absence, checksum, create-only, delete, or presign semantics cannot hold for both tuples; no mechanism can prove R4/R5/R9; or fewer than two real adopter/provider cases remain.

## Obligation reconciliation

| Test Design obligation | Ledger disposition |
| --- | --- |
| TD-001 | T1. |
| TD-002 | T3 owns tuple/client construction; T8 adds selected config/bootstrap; T9A owns the D4 image-root/content-type delta and reruns the complete oracle. |
| TD-003 | T2 owns the generic one-attempt transport; T3 owns the adapter authority/body/send-phase guard; T9A adds and proves the shared caller-owned non-nil-root seam while preserving both receipts. |
| TD-004 | T7, after all five operation paths exist. |
| TD-005 | T7 owns the accepted full-phase deadline/lifecycle behavior; T9A reruns the explicit-root DNS/connect/TLS delta without changing its policy. |
| TD-006 | T9, after T9A supplies the fixed trust/config/transport handoff. |
| TD-007, TD-008, TD-009 | T4. |
| TD-010 | T5. |
| TD-011, TD-012 | T6. |
| TD-013 | T7. |
| TD-014, TD-015 | T8 owns the accepted lifecycle/config behavior; T9A owns the image-root construction/no-production-CA delta and reruns both oracles. |
| TD-016 | T10, including strict image-root and shared non-nil-`RootCAs` inventory. |
| TD-017 | Scope-exit handoff H11: Amazon-only credentialed support certification remains an optional adopter-owned external receipt, excluded from local template completion. |
| TD-018 | Scope-exit handoff H12: R2-only credentialed support certification remains an optional adopter-owned external receipt, excluded from local template completion. |
| TD-019 | Scope exit: no accepted adopter or feature composition point exists, and the Specification excludes feature authorization, key construction, content, size, retention, and overwrite policy from the pack. A future adopter owns its stable denial plus zero-store-call oracle and exact feature command; its existence reopens Go Ownership for placement, and moving policy into the pack reopens Specification. It creates no template implementation task or completion dependency now. |

## Readiness review

Independent Task Review / Readiness returned **PASS** on candidate SHA-256
`9e0063de53b4a38f27624c709594e650ba11343e989037c91ae978a35760894e`.
That disposition remains valid for the accepted T1-T8 receipts and TD-019.
The D4 image-root repair materially invalidated T9 and the then-executable
provider-certification entries, so no T9-or-later
movement consumes that old receipt. Independent Task Review / Readiness
returned **FAIL** on candidate SHA-256
`aa28c97a3b7555cf79f5242518f54f8742ccdff33582ea1c851d6067e94338fe`:
T9A could publish retained trust state before D4's equation and exact startup
rejection, which were deferred to T9. Planning moved the complete checked
equation, startup rejection, and mandatory source/image receipt into T9A and
narrowed T9 to the environment-only Linux process proof. Focused fresh
re-review returned **PASS** on repaired candidate SHA-256
`87280e64e0ee85e26bb47ae7a951708ea1ebb3db37c3ccceb23ab9a8ad45bcd1`;
no finding survives. Changes after that candidate are this review receipt,
status closure, and the credential-free range clarification only; they alter no
task outcome, dependency, owner, proof, gate, handoff, or reopen condition.

Independent Task Review / Readiness returned **FAIL** on candidate SHA-256
`66eb16e8a5004d16e1de55e320e3063f0b30d5833979c90300678e544b2c10cd`:
the canonical T9A blocker still required the now-cleared D4 runtime-bundle
decision, and its writable/proof boundary omitted the S3-owned configured
fixtures consumed by bootstrap. The repaired T9A route makes TD-006's
platform-resolved final-image extraction and strict-loader byte agreement
executable, and couples only those reached fixture helpers to the D4 equality
oracle. Outbound-machine-auth T7 remains a read-only consumer: its independently
reproduced named command now reaches a close-error assertion rather than the
recorded S3 validation error, so outbound Planning owns any correction to that
blocker record. A focused fresh Task Review / Readiness must review this repaired
candidate before the ledger returns to `ready`.

Focused fresh Task Review / Readiness returned **PASS** on repaired candidate
SHA-256 `a6f0ca396ce33d408f0e3a512d71fac8c9095eb0082e03e07ade460f1ba9718f`:
the `linux/arm64` final-image receipt now matches D4/TD-006 and leaves deployed
mount proof with Delivery; the two reached S3 fixture owners now have an exact
T9A placement and equality/one-byte-short oracle; and outbound T7 remains a
notification-only consumer. Changes after that candidate are this review
receipt and status closure only; they alter no task outcome, dependency, owner,
proof, gate, handoff, or reopen condition.

Focused Task Review / Readiness returned **PASS** on the template
certification-boundary candidate SHA-256
`4a657df2b9cccf9c60425353f5c90acd10e4d8ece4fbd3a4ca03acf96ff985cc`:
T1-T10 close the local credential-free template ledger, H11/H12 preserve their
separate Amazon/R2 authorization, prerequisite, non-substitution, and
unverified-receipt boundaries, and neither handoff implies provider action or
blocks template completion. This disposition records the review result only; it
alters no task outcome, dependency, owner, proof, gate, handoff, or reopen
condition.

- [x] T1: The provider-neutral five-operation contract preserves every accepted key byte and rejects every non-common key before adapter work
  - Source: `spec.md` R1-R2 and R9; `design/overview.md` Feature-facing port and Go responsibility map; `test-plan.md` TD-001.
  - Owner/surface/resources: `internal/objectstorage/doc.go`, `store.go`, `errors.go`, and `store_test.go`; standard library only; no mutable resource.
  - Depends on: none.
  - Proof: compile a feature fake against upload, download, metadata, delete, and presigned GET; partition empty, 1/1024/1025-byte, allowed ASCII, Unicode, case-sensitive `soap`, dot/empty segments, leading/trailing slash, and wire-escaping cases. Accepted keys remain byte-exact; rejected keys return `invalid` with zero fake/transport calls. Run `go test -vet=off ./internal/objectstorage -run '^(TestPortContractAndKeyGrammar|FuzzValidateKey)$' -count=1`.
  - Reopen if: grammar or port vocabulary changes, or a provider/SDK field must cross the port — Specification.
  - Accepted: T1; evidence: `go test -vet=off ./internal/objectstorage -run '^(TestPortContractAndKeyGrammar|FuzzValidateKey)$' -count=1` PASS and fresh independent implementation review PASS; candidate: current bounded diff.
  - Accepted: T1 lint-repair; evidence: focused TD-001 command PASS (22), `go test -vet=off ./internal/objectstorage -count=1` PASS (22), scoped `golangci-lint` PASS (0 issues), `gofmt -d` and `git diff --check` clean, and fresh independent implementation review PASS; candidate: current bounded Local repair in `internal/objectstorage/store.go` and `store_test.go`. The repair only names existing interface parameters, uses equivalent empty-key and byte-index checks, and renames a test case; T10 `make lint` remains outside this receipt.

- [x] T2: The shared fixed-target HTTP client can make one fresh HTTP/1 attempt without replaying, transforming, proxying, redirecting, or changing authority
  - Source: `spec.md` R3/R9; `design/overview.md` D2; `test-plan.md` TD-003 generic transport delta.
  - Owner/surface/resources: `internal/infra/httpclient/config.go`, `client.go`, and `client_test.go`; remove OIDC-only marker ownership from the unconditional policy; loopback HTTP/TLS fixtures only.
  - Depends on: none.
  - Proof: retain current authority/DNS/redirect/body-cap checks. A real HTTP/1 server warms one connection, reads and drops a replayable request, and observes one fresh second connection with no third request; gzip response observes no `Accept-Encoding` and exact wire bytes. Run `go test -vet=off ./internal/infra/httpclient -run '^TestOneAttemptTransportDoesNotReplayOrTransform$' -count=1`.
  - Reopen if: the pinned Go transport can replay a fresh non-reused HTTP/1 request or exact-provider conformance requires HTTP/2 — Technical Design.
  - Accepted: T2; evidence: `go test -vet=off ./internal/infra/httpclient -run '^TestOneAttemptTransportDoesNotReplayOrTransform$' -count=1` PASS and fresh independent implementation review PASS; candidate: current bounded diff.
  - Accepted: T2 lint-repair; evidence: focused T2 proof PASS (3), `go test -vet=off ./internal/infra/httpclient -count=1` PASS (115), scoped `golangci-lint` PASS (0 issues), `gofmt -d` and scoped `git diff --check` clean, and fresh independent implementation review PASS; candidate: current bounded Local repair in `internal/infra/httpclient/client_test.go`. The repair only parallelizes subtests, omits an unused handler parameter, and checks the `http.Hijacker` assertion; T10 `make lint` remains outside this receipt.

- [x] T3: One explicit static Amazon-or-R2 tuple constructs a bounded local adapter core with fixed authority, no ambient credential/policy source, and no provider I/O
  - Source: `spec.md` R1/R3/R4/R9/R10; `design/overview.md` selected client, D1-D4 and D7, dependency ownership, and file map; `test-plan.md` TD-002 plus the adapter half of TD-003.
  - Owner/surface/resources: `internal/infra/s3/doc.go`, `config.go`, `client.go`, `transport.go`, `errors.go`, `harness_test.go`, `config_test.go`, `client_test.go`, `transport_test.go`, and `errors_test.go`; exact direct pins in `go.mod`/`go.sum` for AWS core `v1.43.5`, credentials `v1.19.5`, S3 `v1.107.1`, and Smithy `v1.27.7`; no provider or credential resource.
  - Depends on: T1 — accepted portable values/errors needed to start; T2 — accepted one-attempt transport policy needed to complete.
  - Proof: table both provider tuples and every missing/inconsistent authority, credential, part-count, duration, header/body, and memory field under hostile AWS region/profile/endpoint/retry/logger/proxy variables. Invalid input fails before transport; valid construction uses direct `s3.Options`, supplied static credentials, `aws.NopRetryer`, required checksum modes, no refresh worker, and zero I/O. Alternate authority and oversized control/object bodies fail at their exact caps and close. Run `go test -vet=off ./internal/infra/s3 -run '^(TestConfigRejectsInvalidTupleAndEnvelope|TestNewUsesOnlyStaticConfigurationAndPerformsNoIO|TestTransportRefusesAlternateAuthority|TestTransportBoundsControlAndObjectBodies)$' -count=1`, then the complete TD-003 command `go test -vet=off ./internal/infra/httpclient ./internal/infra/s3 -run '^(TestOneAttemptTransportDoesNotReplayOrTransform|TestTransportRefusesAlternateAuthority|TestTransportBoundsControlAndObjectBodies)$' -count=1`.
  - Reopen if: direct construction cannot isolate ambient SDK policy or the SDK resolver/transport cannot preserve fixed authority, single attempt, response caps, and send-phase evidence — Technical Design; another credential or authority class — Specification.
  - Accepted: T3; evidence: mandated T3 and combined TD-003 focused commands PASS, `go test -vet=off ./internal/infra/s3 -count=1` PASS, `git diff --check` PASS, and fresh independent implementation review PASS; candidate: current bounded diff.
  - Accepted: T3 lint-repair; evidence: focused T3 command PASS (28), combined TD-003 command PASS (7), retained T9A constructor selector PASS (3), and `go test -vet=off ./internal/infra/s3 -count=1` PASS (122); package-context `golangci-lint` reports no finding in the six T3 paths while its remaining findings are T4-T9-owned; `gofmt -d` and scoped whitespace checks clean; fresh independent implementation review PASS. Candidate: current bounded Local repair in `config.go`, `client.go`, `transport.go`, `config_test.go`, `client_test.go`, and `transport_test.go`. The repair rejects an empty endpoint query, preserves both existing constructor selectors, and keeps the static tuple, no-I/O construction, fixed authority, response bounds, and one-attempt transport unchanged; T10 `make lint` remains outside this receipt.

- [x] T4: Single and serial multipart replace uploads stream exactly the declared bytes, confirm CRC64NVME, and expose only conservative bounded cleanup
  - Source: `spec.md` R4-R5/R9; `design/overview.md` D5-D7 and upload/cleanup flows; `test-plan.md` TD-007, TD-008, TD-009.
  - Owner/surface/resources: `internal/infra/s3/checksum.go`, `upload.go`, and the reached `client.go`/`errors.go`; `checksum_test.go`, `upload_test.go`, `harness_test.go`, and reached client/error tests; scripted HTTP only.
  - Depends on: T3 — accepted direct client, authority, limits, and send-phase state needed to start.
  - Proof: for TD-007, gate a non-seekable source, independently decode `aws-chunked`, compute CRC64/NVME with stdlib polynomial `0x9a6c9329ac4bc9b5`, require exact declared length/trailer/algorithm/type/content type and qualifying `If-None-Match:*`, accept only matching provider confirmation, fail short input, leave byte `length+1` unread, and select single/multipart exactly at the threshold. Run `go test -vet=off ./internal/infra/s3 -run '^TestSingleUploadStreamsCRC64NVMEAndExactLength$' -count=1`.
  - Proof: for TD-008, exercise `C+1`, exact multiples, final remainder, and part-count edge; require serial Create/UploadPart/Complete, ordered bounded completion descriptors, independently checked part and whole CRC64NVME/FULL_OBJECT plus declared size, and failure on missing/out-of-order/corrupt/embedded-error results. Run `go test -vet=off ./internal/infra/s3 -run '^TestMultipartUploadIsSerialAndConfirmsWholeChecksum$' -count=1`.
  - Proof: for TD-009, after any created upload session stop new parts, make at most three Amazon Abort/List cycles with bounded pagination or one R2 Abort under the same remaining context, preserve the primary error, hide upload ID, close every body, and keep every failed multipart cleanup `pending`; no bounded empty observation becomes terminal proof. Run `go test -vet=off ./internal/infra/s3 -run '^TestMultipartCleanupIsBoundedAndConservative$' -count=1` and `go test -vet=off -race ./internal/infra/s3 -run '^TestMultipartCleanupIsBoundedAndConservative$' -count=10`.
  - Reopen if: SDK framing/checksum/completion metadata or a cleanup terminal observation breaks while an equivalent mechanism may exist — Technical Design; common integrity, create-only, cleanup outcome, or five-operation semantics require a provider escape — Specification.
  - Accepted: T4; evidence: TD-007, TD-008, TD-009, and TD-009 race x10 focused commands PASS; fresh independent implementation review PASS; candidate: current bounded diff.
  - Accepted: T4 lint-repair; evidence: TD-007 focused proof PASS (6), TD-008 PASS (5), TD-009 PASS (9), TD-009 race x10 PASS (90), and `go test -vet=off ./internal/infra/s3 -count=1` PASS (122); scoped `golangci-lint` is clean in `checksum.go`, `upload.go`, and `upload_test.go`; `gofmt`, scoped whitespace, and `git diff --check` are clean; fresh independent implementation review PASS. Candidate: current bounded Local repair in `checksum.go`, `upload.go`, and `upload_test.go`; it preserves closed object-storage errors, reader EOF identity, exact-length CRC64NVME, serial multipart, confirmed completion, and conservative cleanup. T10 whole-tree `make lint` remains outside this receipt.

- [x] T5: Download exposes no body before valid metadata and succeeds only at bounded checksum-validated EOF with one close and admission release
  - Source: `spec.md` R4-R6/R9; `design/overview.md` D3/D5/D7 and download flow; `test-plan.md` TD-010.
  - Owner/surface/resources: `internal/infra/s3/download.go` plus reached `client.go`, `checksum.go`, and `errors.go`; `download_test.go` and shared scripted fixture; no mutable resource.
  - Depends on: T4 — accepted checksum policy and shared scripted adapter state needed to start.
  - Proof: table valid/oversized/absent size, valid/missing/composite validator metadata, matching/mismatching payload, exact limit/overflow, terminal error, early Close, cancellation, and deadline. Invalid metadata or declared size exposes no body; no byte beyond `O` escapes; only clean validated EOF records success bytes; every other terminal path returns its exact stable read error, releases once, closes once, and makes no second GET. Run `go test -vet=off ./internal/infra/s3 -run '^TestDownloadCompletesOnlyAtValidatedEOF$' -count=1` and `go test -vet=off -race ./internal/infra/s3 -run '^TestDownloadCompletesOnlyAtValidatedEOF$' -count=10`.
  - Reopen if: the SDK cannot expose the installed validator or deferred EOF failure — Technical Design; common integrity success changes — Specification.
  - Accepted: T5; evidence: `go test -vet=off ./internal/infra/s3 -run '^TestDownloadCompletesOnlyAtValidatedEOF$' -count=1` PASS (17), `go test -vet=off -race ./internal/infra/s3 -run '^TestDownloadCompletesOnlyAtValidatedEOF$' -count=10` PASS (170), `go test -vet=off ./internal/infra/s3 -count=1` PASS (66), `git diff --check` PASS, and fresh independent implementation review PASS; candidate: current bounded diff.
  - Accepted: T5 lint-repair; evidence: TD-010 focused proof PASS (17), TD-010 race x10 PASS (170), and `go test -vet=off ./internal/infra/s3 -count=1` PASS (122); package-context `golangci-lint` reports no finding in `download.go` or `download_test.go` (the remaining 25 findings belong to T6/T7), `gofmt -d` and scoped `git diff --check` are clean, and fresh independent implementation review PASS. Candidate: current bounded Local repair in `download.go` and `download_test.go`; the repair only documents the retained context/release and closed error/EOF contracts, preserving validated-EOF behavior, one close, and one release. T10 whole-tree `make lint` remains outside this receipt.

- [x] T6: Metadata, delete, presigned GET, and the closed error mapper expose only portable conservative outcomes and no bearer/provider secret
  - Source: `spec.md` R6-R9; `design/overview.md` D7 and metadata/delete/presign flow; `test-plan.md` TD-011 and TD-012.
  - Owner/surface/resources: `internal/infra/s3/metadata.go`, `delete.go`, `presign.go`, and reached `errors.go`/`transport.go`; `metadata_test.go`, `delete_test.go`, `presign_test.go`, `errors_test.go`, and shared fixture; no live URL/provider.
  - Depends on: T5 — accepted body/checksum/error state and shared package safety gate needed to start.
  - Proof: script Head/Delete success, tuple-admitted absence, concealed absence, conditional conflicts, throttling, malformed/oversized XML, request-write loss, and after-write response loss. Require portable UTC fields, 403 never absence, only admitted 404 as `not_found`, absent delete as operation completion, one request per stage, sanitized closed errors/private diagnostics, and `outcome_unknown` for possibly sent Put/Complete/Delete. Run `go test -vet=off ./internal/infra/s3 -run '^(TestMetadataAndDeleteExposePortableResults|TestErrorMappingIsConservativeAndOneAttempt)$' -count=1`.
  - Proof: sign only GET for fixed credentials/key at `0`, `1s`, configured maximum, maximum+1, and seven-day boundary; parse `X-Amz-Date` and integer `X-Amz-Expires`, require `SignatureExpiresAt` to equal their UTC sum without claiming credential validity until that instant, exact bucket authority/decoded key/query/required headers, no HTTP call/default TTL/ambient drift, and no URL/query/header canary in errors or telemetry. Run `go test -vet=off ./internal/infra/s3 -run '^TestPresignGETIsBoundedAndSecret$' -count=1`.
  - Reopen if: SDK/Smithy/send-phase or presigner output changes — Technical Design; portable absence/delete/error semantics or presign operation/lifetime/recipient guarantees change — Specification.
  - Accepted: T6; evidence: mandated TD-011 and TD-012 focused commands PASS, `go test -vet=off ./internal/infra/s3 -count=1` PASS (83), `git diff --check` PASS, and fresh independent implementation review PASS; candidate: current bounded diff.
  - Accepted: T6 lint-repair; evidence: TD-011 PASS (15), TD-012 PASS (1), `go test -vet=off ./internal/infra/s3 -count=1` PASS (122), and fresh independent implementation review PASS; scoped `golangci-lint` has no finding in the T6 paths while four retained findings belong to T7 telemetry; `gofmt -d` and scoped whitespace checks are clean; candidate: current bounded Local repair in `metadata.go`, `delete.go`, `presign.go`, `errors.go`, and their direct tests. The repair preserves closed object-storage errors and the span rooted in the caller context; T10 whole-tree `make lint` remains outside this receipt.

- [x] T7: All five operations share one non-blocking admission/deadline/lifecycle policy and emit only the closed bounded telemetry matrix
  - Source: `spec.md` R4/R9/R11; `design/overview.md` D3/D7/D8; `test-plan.md` TD-004, TD-005, TD-013.
  - Owner/surface/resources: `internal/infra/s3/client.go`, `telemetry.go`, all five reached operation files, `errors.go`, and their existing tests including `client_test.go`, `telemetry_test.go`, upload/download/error tests; existing recording OTel/log patterns, `testing/synctest`, owned channels, loopback TLS, and `goleak`; no provider.
  - Depends on: T6 — complete accepted operation/result paths needed to start and prove the matrix.
  - Proof: with `A=2`, hold both tokens and saturate upload/download/metadata/delete/presign; each rejected call is `busy` before body read, signing, or I/O. Release through success, error, EOF, and Close; forbid multipart/cleanup overlap; prove no queue, token leak, or goroutine. Run `go test -vet=off ./internal/infra/s3 -run '^TestAdmissionIsProcessWideAndNonBlocking$' -count=1` and `go test -vet=off -race ./internal/infra/s3 -run '^TestAdmissionIsProcessWideAndNonBlocking$' -count=10`.
  - Proof: under `synctest`, propagate the earlier caller/configured deadline through DNS/connect, TLS, write, response, body, parts, Abort/ListParts, and presign before/after signing. Cancellation starts no new stage, cleanup keeps the same deadline, upload source remains caller-owned, EOF/error/Close release idempotently, and possibly sent mutations remain `outcome_unknown` ahead of context errors. Run `go test -vet=off ./internal/infra/httpclient ./internal/infra/s3 -run '^(TestOneAttemptTransportUsesRequestDeadline|TestEffectiveDeadlineAndLifecycleOwnEveryPhase)$' -count=1`.
  - Proof: drive only the reachable TD-013 pairs: upload `{success,invalid,too_large,busy,precondition_failed,denied,integrity_failed,cancelled,deadline_exceeded,outcome_unknown,internal}` with `single|multipart` and `none|pending`; download `{success,invalid,too_large,busy,not_found,denied,temporary,integrity_failed,cancelled,deadline_exceeded,internal}`; metadata `{success,invalid,busy,not_found,denied,temporary,cancelled,deadline_exceeded,internal}`; delete `{success,invalid,busy,denied,cancelled,deadline_exceeded,outcome_unknown,internal}`, with context kinds only when non-transmission is proved; presign `{success,invalid,busy,cancelled,deadline_exceeded,internal}`. Require admitted active `0->1->0`, admitted/rejected counts, result duration, failure phase, full-success bytes only, integrity count only on integrity failure, presign issuance without transfer success, impossible pairs absent, one `unknown` fallback, and zero leakage from every forbidden-field canary except sanitized request ID as a non-metric field. Run `go test -vet=off ./internal/infra/s3 -run '^TestTelemetryContractIsBoundedAndSecret$' -count=1`.
  - Reopen if: an SDK path adds a worker/queue/overlap, any stage loses the caller context, or a required safe signal has no owner — Technical Design; error precedence, result vocabulary, or operator question changes — Specification.
  - Accepted: T7; evidence: TD-004 focused command PASS and race x10 PASS; TD-005 focused command PASS; TD-013 focused command PASS (including real multipart `complete` and `pending` cleanup telemetry); affected `internal/infra/s3` and `internal/infra/httpclient` package and race gates PASS; `git diff --check` PASS; fresh independent implementation review PASS after one cleanup-telemetry repair; candidate: current bounded diff.
  - Accepted: T7 lint-repair; evidence: scoped `golangci-lint` PASS (0 issues); TD-004 PASS and race x10 PASS; current explicit-root TD-005 PASS (2); TD-013 PASS (3); `go test -vet=off ./internal/infra/s3 -count=1` and package race PASS (122 each); target `gofmt -d` and whitespace checks clean; fresh independent implementation review PASS. Candidate: current bounded Local repair in `telemetry.go` and `telemetry_test.go`; it only documents the existing caller-context/span-end ownership and makes the unreachable scripted-request fallback return an error. T10 whole-tree `make lint` remains outside this receipt.

- [x] T8: Selected object-storage configuration and bootstrap are fail-closed, readiness-neutral, locally constructed, and closed in the existing shutdown order
  - Source: `spec.md` R3/R10/R12; `design/overview.md` D1/D3/D9, startup/shutdown flow, and inverse map; `test-plan.md` TD-002, TD-014, TD-015.
  - Owner/surface/resources: `internal/config/object_storage_config.go`, `types.go`, `defaults.go`, `validate.go`, `secret_policy.go`, `object_storage_config_test.go`, `snapshot_contract_test.go`, `secret_policy_test.go`; `cmd/service/internal/bootstrap/startup_object_storage.go`, `run.go`, `startup_object_storage_test.go`, `run_lifecycle_test.go`; `env/.env.example`, `env/config/local.yaml`, and `docs/configuration-source-policy.md`; local fakes only. `env/docker-compose.yml` remains Postgres-only and is not an object-storage owner.
  - Depends on: T7 — complete `objectstorage.Store`, adapter telemetry, and lifecycle behavior needed to start composition.
  - Proof: rerun the complete TD-002 oracle so invalid tuple/bounds fail before transport, valid direct static construction performs no I/O, and bootstrap is local-only: `go test -vet=off ./internal/infra/s3 ./internal/config ./cmd/service/internal/bootstrap -run '^(TestConfigRejectsInvalidTupleAndEnvelope|TestNewUsesOnlyStaticConfigurationAndPerformsNoIO|TestObjectStorageStartupIsLocalOnly)$' -count=1`.
  - Proof: preserve readiness/liveness and probe inventory across a post-start scripted provider failure and later operation; construct after memory-limit publication; close once on early return and, normally, after HTTP drain/background join but before dependency close/telemetry flush; active work gets no detached or longer shutdown path. Run `go test -vet=off ./cmd/service/internal/bootstrap -run '^(TestObjectStorageStartupAndOutageDoNotChangeReadiness|TestObjectStorageRuntimeCloseOrder)$' -count=1` and `go test -vet=off -race ./cmd/service/internal/bootstrap -run '^TestObjectStorageRuntimeCloseOrder$' -count=10`.
  - Proof: exact absent/selected config leaf/default/snapshot inventories reject unknown absent keys, require finite selected values, accept empty placeholders, accept all three static credential values only from environment, reject non-empty file sources without leaking raw values, and expose no ambient AWS leaf or usable example. Run `go test -vet=off ./internal/config -run '^(TestObjectStorageConfigContract|TestSnapshotContract|TestStaticCredentialSourcePolicy)$' -count=1`.
  - Reopen if: lifecycle owner/order cannot hold — Technical Design; another credential/config source or storage-critical readiness is required — Specification; section/bootstrap placement changes — Go Ownership.
  - Accepted: T8; evidence: TD-002, TD-014 (including race x10), and TD-015 focused commands PASS; touched-package tests PASS; fresh independent implementation review PASS; candidate: current bounded diff.
  - Accepted: T8 lint-repair; evidence: legacy TD-002 command PASS (20), current local-construction oracle PASS (5), TD-014 focused and close-order race x10 PASS (5/30), TD-015 config command PASS (14), scoped bootstrap lint, `gofmt -d`, and untracked-file whitespace checks clean, and fresh independent implementation review PASS; whole-tree `make lint` remains T10 proof outside this receipt; candidate: current bounded Local repair in `startup_object_storage.go` and `startup_object_storage_test.go`.

- [x] T9A: S3 construction owns one strict bounded image-root snapshot inside a final-image-proved startup memory ceiling without regressing accepted shared contracts
  - Source: `spec.md` R3/R4/R9/R10/R12; `design/overview.md` D1-D4, startup/shutdown flow, Go responsibility map, and D4 trust-root reopen; `test-plan.md` TD-002, TD-003, TD-005, TD-006, TD-014, and TD-015; `specs/outbound-machine-auth/tasks.md` T4 Accepted shared-HTTP receipt.
  - Owner/surface/resources: add the T9A-only source-receipt entry point under `scripts/ci/` and its `Makefile` target; add `internal/infra/s3/image_root_bundle.go` and `image_root_bundle_test.go`; change reached `internal/infra/s3/doc.go`, `config.go`, `client.go`, `upload.go`, `config_test.go`, `client_test.go`, and `upload_test.go`; change `internal/infra/httpclient/config.go`, `client.go`, and `client_test.go` only to add the generic code-only caller-owned `RootCAs` seam while preserving the current outbound-auth T4 attempt-authorization diff; change reached `cmd/service/internal/bootstrap/startup_object_storage.go`, `startup_object_storage_test.go`, `run_lifecycle_test.go`, and the S3-owned `run_test.go` bootstrap fixture; change only `internal/config/testhelpers_test.go`, `object_storage_config_test.go`, `snapshot_contract_test.go`, and `secret_policy_test.go` to keep the selected object-storage fixture and no-production-CA inventory aligned. The production source is exactly the image-owned regular `/etc/ssl/certs/ca-certificates.crt`; test roots use only an unexported source seam. The source-receipt target owns a temporary Docker image/container and extracted bundle file, removes each on exit, and never starts the container, mounts the bundle, contacts a provider, or retains a secret.
  - Depends on: T8 — accepted S3 config/bootstrap/lifecycle baseline needed to start; outbound-machine-auth T4 Accepted receipt — the current shared `httpclient` attempt-authorization surface is an immutable input needed to start and prove.
  - Handoff: T9A produces a strict `448 KiB`/288-unique-valid-CA loader, one immutable adapter-private non-nil pool applied with the validated hostname and no system-root fallback or mutation, the complete checked D4 `required_memory` equation with exact startup rejection, and a TD-006 receipt for `linux/arm64` that resolves the pinned Distroless index to its platform manifest, records the Dockerfile/final-image config/rootfs identities and unmounted regular `0555` bundle entry, and proves the extracted bundle's bytes/hash/count are exactly the bytes consumed by the strict loader. The S3-owned selected fixtures carry the reviewed equality value `62145920` and a one-byte-short control fails before root open. T9 consumes that fixed trust/config/transport/memory candidate only for the environment-level five-run Linux envelope; outbound T7 consumes no new S3 receipt and is notified only after this canonical transition.
  - Proof: prove strict bounded image loading, non-nil static S3 construction, pre-admission content-type rejection, and local bootstrap with `go test -vet=off ./internal/infra/s3 ./internal/config ./cmd/service/internal/bootstrap -run '^(TestConfigRejectsInvalidTupleAndEnvelope|TestImageRootBundleLoaderIsStrictAndBounded|TestUploadRejectsUnboundedContentType|TestNewUsesStaticConfigurationAndImageRootsWithoutNetworkIO|TestObjectStorageStartupLoadsImageRootsLocally)$' -count=1`.
  - Proof: prove caller-pool immutability, exact `ServerName`, verification enabled, configured-chain success, ambient-only/wrong-host denial, nil-caller compatibility, one attempt, and bounds with `go test -vet=off ./internal/infra/httpclient ./internal/infra/s3 -run '^(TestOneAttemptTransportDoesNotReplayOrTransform|TestTransportUsesCallerRootCAsWithoutAmbientFallback|TestTransportRefusesAlternateAuthority|TestTransportBoundsControlAndObjectBodies)$' -count=1`; prove deadline ownership reaches explicit-root DNS/connect/TLS and every adapter phase with `go test -vet=off ./internal/infra/httpclient ./internal/infra/s3 -run '^(TestOneAttemptTransportUsesRequestDeadlineAndExplicitRoots|TestEffectiveDeadlineAndLifecycleOwnEveryPhase)$' -count=1`.
  - Proof: prove trust construction after memory-limit publication, immutable snapshot/no reload, readiness neutrality, and close order with `go test -vet=off ./cmd/service/internal/bootstrap -run '^(TestObjectStorageStartupLoadsImageRootsLocally|TestObjectStorageStartupAndOutageDoNotChangeReadiness|TestObjectStorageRuntimeCloseOrder)$' -count=1`, `go test -vet=off -race ./cmd/service/internal/bootstrap -run '^TestObjectStorageRuntimeCloseOrder$' -count=10`, and prove no production CA/config source with `go test -vet=off ./internal/config -run '^(TestObjectStorageConfigContract|TestSnapshotContract|TestStaticCredentialSourcePolicy)$' -count=1`.
  - Proof: independently calculate every checked D4 component and `required_memory=max(construction, shared+A*max(...))`; cover equality/one-byte-short, every add/multiply/ceil/align/conversion/`Q`/trust/`A*max` overflow, `P=1/10,000/10,001`, `B/B+1`, `N/N+1`, actual half-ceiling headroom, independent `H<E`/`H=E`/`H>E`, content type `0/1/1,024/1,025`, allocator branches, every operation class, and wrong-architecture receipt. The two reached S3-owned fixture helpers must use the same reviewed equality value, and their focused test control must pass that value through selected config/bootstrap construction while a one-byte-short variant fails before root open. Run `go test -vet=off ./internal/config ./internal/infra/s3 ./cmd/service/internal/bootstrap -run '^(TestObjectStorageConfigContract|TestWorkingMemoryAccounting|TestObjectStorageConstructionFollowsMemoryPublication)$' -count=1`; exact maximum admission is accepted and one byte less rejects the complete trust/config/transport candidate before DNS, credential use, or provider I/O.
  - Proof: before accepting the production constants, `S3_RECEIPT_PLATFORM=linux/arm64 make test-s3-source-receipt` must build the final Dockerfile stage, resolve the pinned Distroless index and platform manifest, and materialize an otherwise unmounted stopped final-image container. It records Dockerfile identity, platform/index/resolved-manifest, final-image config/rootfs identity, and the fixed bundle entry's regular `0555` image ownership, then hashes/counts the extracted file and passes those identical bytes through the unexported strict-loader seam. It also records the fixed Go 1.26.5 and AWS/Smithy versions/sums, nine Go source SHA-256 identities, compiler escape/stack evidence, shallow/rounded sizes, simultaneous counts, and all seven D4 allocation classes. Every reachable allocation has one finite named driver; record used/unused bytes and percentage and require at least 100% source-derived headroom for `F`, `S`, `U`, `trust_shared`, `trust_startup`, and `trust_verify`. Any mismatch, unknown, duplicate/reclassified owner, wrong driver, non-finite count, wrong platform, non-regular/non-`0555` entry, failed byte agreement, or half-ceiling failure reopens D4 without tuning a reserve. This is an image-artifact receipt only: deployed mount/no-override proof remains Delivery-owned.
  - Proof: preserve outbound-machine-auth T4 on the same shared diff with its three accepted commands: `go test -vet=off ./internal/infra/httpclient -run '^TestAttemptAuthorizationPreservesRetryPolicy$' -count=1`; `go test -vet=off ./internal/infra/oauth2clientcredentials ./internal/infra/httpclient -run '^(TestHTTPClientResourceAuthorityIsFixed|TestHTTPClientAttachesOneOperationToken|TestHTTPClientRejectsCallerAuthorization|TestHTTPClientCallerCancellationStopsOnlyItsWait|TestHTTPClientPreservesDownstreamAuthResponses|TestGeneratedClientUsesAuthenticatedDoer)$' -count=1`; `go test -vet=off ./internal/infra/oauth2clientcredentials -run '^(TestOperationTokenCannotRenewAcrossExpiryMargin|TestHTTPRetryFixesOneTokenAndStopsAtMargin)$' -count=1`.
  - Reopen if: the fixed image path cannot be strictly byte/count bounded or supplied as one immutable non-nil caller pool; the platform-resolved final-image receipt cannot agree byte-for-byte with the strict loader; the pool can consult/mutate ambient roots; the shared edit cannot preserve outbound T4 or TD-005; the fixture equality cannot reach selected S3 construction; production gains a CA setting/reload; or bootstrap ordering/readiness cannot hold — D2/D4 Technical Design, Go Ownership, Test Design, or Specification according to the cited artifact's reopen condition.
  - Accepted: T9A; evidence: replacement receipt for the invalidated shared-bootstrap fixture: the S3-owned `62145920` fixture equality reaches selected bootstrap construction and a one-byte-short value rejects before root open; all listed T9A focused bootstrap/config/memory, explicit-root/transport, lifecycle race, and outbound-machine-auth T4 preservation commands PASS; `S3_RECEIPT_PLATFORM=linux/arm64 make test-s3-source-receipt` PASS with reviewed `linux/arm64/v8` Go manifest `sha256:145d3e4c318457af3040b2e575f3f511c7860054c277e4cb5de58c4fe913c3e7`, nine source identities, module pins, final-image bundle, and strict-loader agreement; outbound T3 inherited bootstrap command PASS; fresh independent implementation review PASS. Consumer-only outbound T7 reaches its separate profile/lint/full-suite failures and does not reopen T9A. Candidate: current bounded Local T9A implementation with the fixture reconciliation and final-image receipt entry point. T9, T10, provider, deployment, publication, and certification remain outside this receipt.
  - Accepted: T9A lint-repair; evidence: owned `golangci-lint` surfaces, `gofmt -d`, and scoped whitespace checks PASS; focused T9A bootstrap/config/memory (40 and 23), explicit-root/transport (12), deadline/lifecycle (2), lifecycle (5) and race x10 (30), config (14), and outbound T4 preservation (2, 12, 4) commands PASS; refreshed `S3_RECEIPT_PLATFORM=linux/arm64 make test-s3-source-receipt` PASS with final-image strict-loader byte/count agreement and required D4 headroom; fresh independent implementation review PASS. The remaining two-package lint findings are outside this unit and remain owned by T10 or their existing unit. Candidate: current bounded Local repair in `image_root_bundle.go`, `image_root_bundle_test.go`, the explicit-root `httpclient` branches, and the S3 source/process-envelope test portions. T10 whole-tree `make lint`, provider certification, deployment, publication, and H11/H12 remain outside this receipt.

- [x] T9: Five Linux process runs confirm the source-proved maximum-admission retained-memory ceiling and release every owned resource
  - Source: `spec.md` R4 success criterion 4; `design/overview.md` D4 and dependency reopen conditions; `test-plan.md` TD-006.
  - Owner/surface/resources: reached `internal/infra/s3/client_test.go` and `image_root_bundle_test.go` plus the `Makefile` `test-s3-envelope` entry point; one Docker Linux runner using the digest-pinned Go 1.26.5 image, `GOMAXPROCS=1`, separate unmeasured controller/TLS-fixture and measured-child PIDs, fixed-size scalar IPC, local fresh HTTP/1 TLS, and no provider credential.
  - Depends on: T9A — output handoff — needed to start.
  - Proof: `GOMAXPROCS=1 make test-s3-envelope` runs five identical measured children from one warmed preconstruction baseline per child and holds construction, idle, and every `A=2` formula-class peak over the real SDK and explicit-root fresh HTTP/1 TLS path. Drive retained `B/N`, `2*10,000` original plus escape-amplified completion descriptors, independent maximum `H/E`, and object/chunk variants; for every held sample require the exact same-snapshot `smaps_rollup` equation `Rss == Shared_Clean + Shared_Dirty + Private_Clean + Private_Dirty`, then require the five-run maximum non-negative delta at or below both reviewed `required_memory` and the configured ceiling. `VmRSS` and Go heap/stack/goroutine values are diagnostics only. Cancel/join all calls and the fixture process and require zero child admission/active/body/connection counters, zero fixture request/body/connection/goroutine counters, and no new child adapter/SDK/HTTP/TLS/control goroutine.
  - Reopen if: a source/build/image/bundle identity or mount policy drifts; a reserve, allocation class, driver, copy, buffer, goroutine, coefficient, or simultaneous count is unknown; any half-ceiling fails; `smaps_rollup` is missing/unstable or violates its identity; a retained byte/resource/worker or child RSS delta is unexplained; the configured ceiling cannot hold; or SDK/Smithy/Go/checksum/transport changes — D4 in Technical Design.
  - Accepted: T9; evidence: `GOMAXPROCS=1 make test-s3-envelope` PASS on the digest-pinned Linux/arm64 Go 1.26.5 runner with five fixed-root measured children; same-snapshot `smaps_rollup` deltas were `278962176`, `278990848`, `279240704`, `276086784`, and `269766656` bytes, so the maximum `279240704` is non-negative and below both reviewed `required_memory` and the configured `310087588`-byte ceiling. Every child holds the exact `448 KiB`/288-unique-valid-root fixture, construction/idle, all `A=2` formula classes, `2*10,000` escape-amplifying completion descriptors, exact accepted wire-`H=1024` and unended chunked `E=1024` control-error responses, maximum-object and `K` stream variants without response prefetch, and a `K` blocking upload source. Checked `smaps_rollup` requires all five resident categories exactly once with checked conversion/addition before comparison; child/fixture counters, connections, handlers, bodies, and new adapter/SDK/HTTP/TLS/control goroutines all return to zero. `go test -vet=off ./internal/infra/s3 -count=1` PASS (118), focused parser falsifier PASS, `gofmt -d` and scoped `git diff --check` PASS, and fresh independent implementation review PASS. Candidate: bounded Local T9 implementation only; T10, provider certification, deployment, and publication remain outside this receipt.
  - Accepted: T9 lint-repair; evidence: `GOMAXPROCS=1 make test-s3-envelope` PASS; `go test -vet=off ./internal/infra/s3 -run '^(TestAdmissionIsProcessWideAndNonBlocking|TestEffectiveDeadlineAndLifecycleOwnEveryPhase|TestSmapsRollupParserIsFailClosed|TestImageRootBundleLoaderIsStrictAndBounded)$' -count=1` PASS (17); package-context scoped lint reports no finding in `client_test.go` or `image_root_bundle_test.go`; `gofmt -d` and untracked-file whitespace checks clean; fresh independent implementation review PASS. Whole-tree `make lint` remains T10 proof outside this receipt. Candidate: current bounded Local repair in `internal/infra/s3/client_test.go`.

- [x] T10: `OBJECT_STORAGE=none|s3` generation is deterministic, orthogonal, dependency-clean, credential-free, and leaves complete proof/operator entry points
  - Source: `spec.md` R12 and success criteria 5-6; `design/overview.md` D9, inverse map, and dependency ownership; `test-plan.md` TD-016 plus aggregate gates.
  - Owner/surface/resources: `scripts/init-module.sh`, `scripts/ci/template-init-check.sh`, `scripts/ci/ci-change-scope.sh` and its self-test cases, `.github/workflows/ci.yml`, `Makefile`, build-tagged external-package `test/s3conformance/conformance_test.go`, `go.mod`, `go.sum`, `env/.env.example`, `env/config/local.yaml`, `docs/configuration-source-policy.md`, `docs/s3-compatible-object-storage.md`, `docs/repo-architecture.md`, `docs/project-structure-and-module-organization.md`, `docs/build-test-and-development-commands.md`, `README.md`, and the exact object-profile markers/inventory across T1-T9A/T9-owned files, including `image_root_bundle.go`/test and the generic non-nil-`RootCAs` `httpclient` branch; checkout-copy generator fixtures only. The existing runtime-image bundle remains image-owned and is never copied or exposed as generated config.
  - Depends on: T9 — complete deterministic capability and accepted process envelope needed to prove retained/absent inventories and aggregate closure.
  - Proof: generate all four object/outbound combinations plus authn controls for the three-way `httpclient` predicate; compare independent present/absent inventories, including the strict S3 image-root owners and shared code-only `RootCAs` branch; compile/test each retained output, run `go mod tidy`, require only the selected AWS family and no rejected client/transfer manager or generated CA source/setting, strip markers/generator sources, check lock/completion values, unknown/empty no-mutation, byte-identical equal repeat, and unchanged incompatible repeat. The CI scope self-test must classify every object-profile source and the S3 conformance integration file as template-required. Run `TEMPLATE_INIT_PROFILE=object-storage make template-init-check`, `make ci-change-scope-check`, `make project-structure-check`, and `make mod-tidy-check`.
  - Proof: on the same bounded candidate run `go test -vet=off ./internal/objectstorage ./internal/infra/s3 ./internal/infra/httpclient ./internal/config ./cmd/service/internal/bootstrap`, `go test -vet=off -race ./internal/infra/s3 ./internal/infra/httpclient ./cmd/service/internal/bootstrap`, `TEMPLATE_INIT_PROFILE=object-storage make template-init-check`, `make project-structure-check`, `make mod-tidy-check`, and `make lint`. These are current-tree aggregate evidence only; the credentialed entrypoints compile/fail closed but establish no provider support.
  - Reopen if: inventory/retention placement changes — Go Ownership; selector semantics change — Specification; a dependency/source update hits the design source-ownership triggers — Technical Design before updating.
  - Accepted: T10; evidence: repaired profile-boundary inventory, then `TEMPLATE_INIT_PROFILE=object-storage make template-init-check` PASS for every object/outbound/authn generated fixture; five-package aggregate PASS (797); three-package race PASS (427); `make ci-change-scope-check`, `make project-structure-check`, `make mod-tidy-check`, `bash -n scripts/ci/template-init-check.sh`, and `git diff --check` PASS; the released whole-current-tree `make lint` PASS (0 issues); fresh independent implementation review PASS, including an adversarial fail-closed inventory check. Candidate: current bounded Local T10 diff; no provider-support claim and no provider request, credential use, deployment, publication, or H11/H12 activity occurred.

- [x] T13: Audit hardening closes provider-specific trust, retry, cleanup,
  integrity, error, presign, portability, and proof gaps without widening the
  five-operation port
  - Source: repaired `spec.md` R3-R11; `design/overview.md` D1/D3/D5-D8;
    `test-plan.md` TD-002/003/006/009-013/015-018; the reference audit of
    immutable candidate `282b15e007f95ab0feaec530308570185ad58d0e`.
  - Owner/surface/resources: existing object-storage config/bootstrap,
    `internal/infra/s3`, their direct tests, operator/config/architecture docs,
    `env` examples, `scripts/ci/s3-source-receipt.sh`, and the existing profile
    inventory. No new package, dependency, goroutine, credential source,
    provider call, bucket, or deployment resource.
  - Depends on: T1-T10 accepted original capability.
  - Postcondition: Amazon accepts only a commercial regional endpoint,
    temporary access/secret/session snapshot, and 12-digit expected owner on
    every operation; R2 accepts only exact 32-hex default/EU/FedRAMP account
    endpoints and omits expected owner. Both remain fixed-authority,
    virtual-host, explicit-root, no-ambient clients; snapshot rotation is
    process replacement and refreshed workload identity remains out of scope.
  - Postcondition: only HeadObject and pre-body GetObject use the exact bounded
    three-attempt transient set. Head 404 is `not_found` only for pinned
    `NotFound`/`NoSuchKey`; Get 404 only for exact `NoSuchKey`; 401/403 is
    `denied`, exhausted read 429/5xx is `temporary`, Amazon create-only 409/412
    is `precondition_failed`, and every possibly sent mutation remains
    `outcome_unknown`. Span-only provider diagnostics use the finite code,
    category, request-ID, status, and attempt policy without exposing an SDK
    cause or input-driven metric label.
  - Postcondition: a lost possibly sent CreateMultipartUpload reports cleanup
    `pending`; Amazon known-ID cleanup performs at most three serial Abort plus
    ten 1000-part ListParts pages per cycle; every failed multipart upload
    remains `pending`, including after an empty listing. Upload oversize is
    `too_large`; mediated Get rejects range responses before body exposure,
    owns terminal close, maps any non-context terminal checksum mismatch to
    `integrity_failed`, and does not label ordinary early Close cancellation.
    Presign validates Amazon expected owner in the signed query, R2 omission,
    GET/authority/TTL, and documents unsigned Range and bearer limits.
  - Proof: run the focused TD-002/003/009-013 commands, complete S3/config/
    bootstrap packages, and their race surfaces. Run
    `S3_RECEIPT_PLATFORM=linux/arm64 make test-s3-source-receipt` against Go
    1.26.6 index `sha256:116d58cbd88c1297624acc6e967a060012422bacf9930927e23fb719189c6f36`
    and arm64 manifest
    `sha256:7939e2c75db3d059fc944bb6464a916d0fa64bd5a3bd7b3528f2a1ac7673a0eb`,
    then `GOMAXPROCS=1 make test-s3-envelope`. Run
    `TEMPLATE_INIT_PROFILE=object-storage make template-init-check`,
    `make ci-change-scope-check`, `make project-structure-check`,
    `make mod-tidy-check`, the claim-scoped linter, and `git diff --check`.
  - Reopen if: an exact provider rejects the local mechanism, the fixed
    credential/authority class must expand, a new provider-specific port field
    is required, the source/process envelope fails, or generated `none|s3`
    purity cannot preserve the repaired owners — route to the owner named by
    the repaired spec/design/test-plan, never to the other provider's receipt.
  - Accepted: T13; evidence: ready Specification
    `46ee347a7931d9d20601ce00cea8455a89e2d9ba166dfa8e308b8a4154e16def`,
    ready Technical Design
    `b7b53857051b5c5dec35bc787782a461fb022a70d8dddbc12d599f71c702e496`,
    and ready Test Design
    `2c709fdc8ec6841945df320a33bc3f8d1665c101d4752d6e705af23dfc6be147`
    each received fresh independent PASS. The five-package aggregate passed 860
    tests and the three-package race aggregate passed 475; the claim-scoped
    linter reported 0 issues. The complete object-storage template profile
    matrix passed after proving that `none` removes the S3 secret-policy block;
    CI change-scope, project structure, module tidy, shell syntax, and full
    `git diff --check` passed. The final Linux/arm64 process envelope passed five
    runs with deltas `279367680`, `279015424`, `279265280`, `279240704`, and
    `279408640`, all below the computed `310099108`-byte ceiling with zero
    retained tokens/connections/goroutine growth. On a clean projection of
    immutable base `282b15e007f95ab0feaec530308570185ad58d0e` plus only the T13
    S3 production delta, the Go 1.26.6 source/image receipt passed with the
    pinned index/manifest above, final image config
    `sha256:1cf6e7019cd8accc6eafc4b5e56532355204f919b7bcf55209c91528cdba0588`,
    and bundle `216591` bytes / 142 roots / SHA-256
    `a3413a37a8e09cc21b2c11c9ffb23d92d2fc9d1933c9e7617f5c4fba4f72d37d`.
    The mixed working-tree receipt is excluded because unrelated in-progress
    webhook sources do not compile; none was copied into or repaired by the
    T13 projection. Fresh implementation re-review PASS after the final
    Amazon-only 409/412 mapper repair. No provider request, credential use,
    bucket mutation, deployment, publication, or H11/H12 activity occurred.

- [x] T14: Reference-audit repair makes provider differences and certification
  proof executable without widening runtime authority
  - Source: current official Amazon S3 bucket, conditional PutObject,
    AbortMultipartUpload, and SigV4 query-auth contracts; current Cloudflare R2
    bucket, S3-extension, error, and presign contracts; the reference audit of
    immutable candidate `282b15e007f95ab0feaec530308570185ad58d0e`.
  - Owner/surface/resources: existing S3 config/error/upload/presign/telemetry
    owners, their tests, the isolated `test/s3conformance` package, Make/profile
    inventory, multi-architecture source/process receipts, and current S3
    operator/specification artifacts. No new dependency, runtime worker,
    provider call, credential use, bucket, or deployment resource.
  - Postcondition: the shared bucket subset rejects all current Amazon reserved
    namespaces; lost possibly-sent CreateMultipartUpload is
    `outcome_unknown`/cleanup `pending`; Amazon 409
    `ConditionalRequestConflict` is retryable only by a fresh caller operation,
    exact Amazon/R2 412 `PreconditionFailed` is `precondition_failed`, and all
    other possibly-sent mutations stay `outcome_unknown`. Provider diagnostics
    expose only finite phase/category/code/status/request-ID evidence.
  - Postcondition: presigned GET validates exact singular SigV4 credential
    scope, token, lifetime, signature, authority, signed-header ordering, and
    provider expected-owner policy. Source and process receipts cover both
    `linux/amd64` and `linux/arm64`. Separate fail-closed Amazon and R2 harnesses
    execute the real port matrix only after all provider/image/policy inputs are
    present and never substitute one provider's result for the other.
  - Proof: focused and package/race tests; integration-tag compile/skip; both
    fail-closed no-credential provider targets; both architecture source and
    process receipts; object-storage template/profile, change-scope, structure,
    module, shell, lint, and whitespace gates.
  - Reopen if: either exact provider rejects its mechanism, a current official
    contract changes, a supported architecture receipt fails, or profile purity
    loses an owner. H11 and H12 remain separately unverified until authorized
    provider-specific runs succeed.
  - Accepted: T14; evidence: the final five-package aggregate passed 886 tests;
    the three-package race aggregate passed 501; integration-tag compilation,
    the focused S3/conformance linter (`0 issues`), shell syntax, change-scope,
    project-structure, module-tidy, whitespace, and final `none|s3` profile
    matrix passed. Clean projection receipts passed for `linux/amd64` and
    `linux/arm64` with their pinned Go/Distroless manifests and identical
    `216591`-byte, 142-root bundle; process-envelope maxima were `283156480`
    and `279396352` bytes against `310099108`. The first amd64 envelope run
    exposed a one-second test-only operation deadline that could expire before
    maximum-part serialization reached the fixture; the repaired harness keeps
    the independent ten-second diagnostic guard and the same memory ceiling
    while giving the measured operation 30 seconds. Both provider targets failed
    closed before I/O on missing mutation authorization, as required. The
    whole dirty-tree linter is excluded from this receipt: its eight findings
    are confined to unrelated in-progress jobs/webhook files. Candidate:
    current bounded T14 S3 delta over immutable base
    `282b15e007f95ab0feaec530308570185ad58d0e`; no provider request,
    credential use, bucket mutation, deployment, or publication occurred. A
    fresh independent implementation review found no actionable finding and
    passed 43 targeted adversarial S3 tests. H11 and H12 remain separate,
    unverified external provider certifications.

- [x] T15: Remaining reference-audit repairs make upload cancellation,
  multipart cleanup, read absence, download length, and presign lifetime honest
  at the provider-neutral boundary
  - Source: repaired `spec.md` R4-R6/R8/R11; `design/overview.md` D5-D7;
    `test-plan.md` TD-005/009-013; the final focused audit findings against
    immutable base `282b15e007f95ab0feaec530308570185ad58d0e`.
  - Owner/surface/resources: `internal/objectstorage`, existing S3 upload,
    download, error, presign, telemetry owners and tests, bootstrap/conformance
    fakes, and current S3 docs/artifacts. No new dependency, worker, buffered
    body, provider call, credential, bucket, or deployment resource.
  - Postcondition: `Upload` owns an `io.ReadCloser`, closes it exactly once on
    every path, and cancellation/deadline closes a source whose contract
    promptly unblocks `Read`; admission cannot be retained by the prior bare
    `io.Reader` contract. Every failed multipart cleanup is `pending`, including
    after Amazon returns one empty ListParts traversal; abandoned-upload
    lifecycle remains the terminal reclamation boundary.
  - Postcondition: HeadObject 404 is `not_found` only for pinned SDK
    `NotFound`/`NoSuchKey`, GetObject 404 only for exact `NoSuchKey`, and generic
    404/`NoSuchBucket` stays `internal`. Download refuses missing, negative, or
    above-ceiling Content-Length before body exposure and closes/releases an
    idle returned body when its effective context ends. Presign returns
    `SignatureExpiresAt`, the SigV4 expiry rather than a guaranteed credential
    lifetime; credential expiry/revocation may end access earlier.
  - Proof: focused/package/race tests; integration-tag compile; both
    provider targets fail closed before I/O without mutation authorization;
    object-storage profile, CI-scope, structure, module-tidy, claim-scoped lint,
    whitespace, both architecture source receipts, and both architecture Linux
    process envelopes. Independent implementation review applies to this fixed
    unit. H11/H12 remain unverified and non-substitutable.
  - Reopen if: a caller cannot provide a Close-unblocks-Read source, an exact
    provider proves a different structured absence/error shape, a stable
    terminal multipart cleanup observation becomes available, or the public
    presign result needs a guaranteed minimum lifetime — route to Specification
    or Technical Design as named by the current artifacts, never infer the
    other provider's behavior.
  - Accepted: T15; candidate was bounded S3 diff SHA-256
    `459d612c7aa80d314538f9962043f2aeea91e747aadc9c5433c3f46ade58992e`
    over immutable base `282b15e007f95ab0feaec530308570185ad58d0e`.
    Focused object/S3/httpclient packages passed 334 tests; object-storage
    config oracles passed 15; bootstrap passed 6; the mapped race set passed
    142. Integration-tag conformance compiled, claim-scoped lint reported 0
    issues, targeted whitespace passed, and both Amazon/R2 targets failed
    closed before I/O without mutation authorization. The final object-storage
    profile, CI-scope, project-structure, and module-tidy gates passed; current
    linux/amd64 and linux/arm64 source/image receipts and process envelopes
    passed. The full `internal/config` aggregate is excluded because concurrent
    unrelated webhook work fails
    `TestWebhookWorkerLoaderIgnoresForeignProfiles`; the exact S3 config oracle
    passed and no webhook/jobs path was changed for T15. Fresh independent T15
    implementation review returned PASS after its own focused and adversarial
    race checks. No provider request, credential use, bucket mutation,
    deployment, publication, or certification occurred; H11 and H12 remain
    separate and unverified.

- [x] T16: Provider ceilings, partition validation, source receipts, process
  envelopes, and CI claims match the final reference-audit boundary
  - Source: current official Amazon commercial regional endpoint contract and
    Cloudflare R2 limits; `spec.md` R3/R4/R12; `design/overview.md` D1/D4/D9;
    `test-plan.md` TD-002/006/016; the reference audit of immutable base
    `282b15e007f95ab0feaec530308570185ad58d0e`.
  - Owner/surface/resources: existing S3 config and resource-envelope tests,
    source-receipt script, object-storage CI profile, and the matching operator,
    Specification, Technical Design, and Test Design text. No new package,
    dependency, runtime worker, provider call, credential, bucket, deployment,
    or publication resource.
  - Postcondition: Amazon accepts only the selected commercial partition and a
    maximum 5 TiB object envelope; R2 accepts at most its documented 5
    TiB-minus-5 GiB object envelope. Invalid partition or provider ceiling fails
    during local construction before signing or I/O.
  - Postcondition: the source receipt runs the strict root-bundle loader and
    checked compiler predicates inside the selected pinned Linux Go image. The
    process baseline precedes config, PEM input, root parsing, and client
    construction; the construction barrier keeps the raw bundle live. The
    measured child conservatively retains the production client plus a separate
    TLS-fixture client, while focused tests—not the fixture—own production
    DNS/IP, authority, proxy, redirect, hostname, and ambient-root denial.
  - Postcondition: the generated object-storage CI profile runs the amd64 source
    and process-envelope gates; arm64 remains a separate mandatory local receipt.
    Amazon and R2 conformance remain distinct fail-closed external handoffs.
  - Accepted: T16; fixed nine-file content candidate SHA-256
    `ea20e459b3167a838f67741bc031d338eefaf8b073fe8b404e12ce080ff2d773`.
    The five-package aggregate passed 897 tests; the three-package race aggregate
    passed 511; integration-tag conformance compiled; scoped lint reported 0
    issues; actionlint, targeted ShellCheck/shell syntax, CI change-scope,
    project structure, module tidy, profile generation, and whitespace passed.
    Both provider targets failed closed before I/O on absent mutation authority.
    Fresh independent implementation review verified the same content hash,
    passed 43 focused adversarial S3 tests, and returned PASS.
    Current pinned source/image receipts passed on linux/amd64 and linux/arm64
    with checked compiler evidence and identical `216591`-byte, 142-root bundle;
    process-envelope maxima were `283365376` and `279633920` bytes against the
    `310099108`-byte ceiling. No provider request, credential use, bucket
    mutation, deployment, publication, or certification occurred; H11 and H12
    remain separate and unverified.

The prior T9A/T9 Go 1.26.5 source/process receipts above are historical inputs
only and are superseded for the current candidate by T14's Go 1.26.6 receipts.

## Optional adopter-owned certification handoffs

These handoffs are outside the local template implementation ledger. They are
unverified and require separate external authorization; no provider request,
credential use, mutation, deployment, purchase, publication, or certification
is implied by this completed ledger.

### H11: Amazon S3 exact-tuple support certification

- Status: optional adopter-owned handoff; unverified until its own Amazon-only receipt.
- Recipient instructions: only an authorized adopter may run this handoff after every external input below is available. Its receipt cannot certify R2, a deployed path, or another adopter.
  - Source: `spec.md` R1 and success criterion 3; `design/overview.md` provider matrix and reopen conditions; `test-plan.md` TD-017.
  - Owner/surface/resources: fixed `//go:build integration`, external-package `test/s3conformance/conformance_test.go` Amazon entrypoint and `Makefile` target that supplies `-tags=integration`; one pre-existing operator-owned commercial regional general-purpose bucket, primary and concealment temporary-session identities, and unique `conformance/amazon_s3/<run-id>/` prefix/uploads. The test may mutate only that prefix/uploads and never provisions or changes bucket, identity, policy, lifecycle, versioning, encryption, DNS, or network controls.
  - Local prerequisite: T1-T10 accepted deterministic template capability, conformance entrypoint, aggregate gates, and profile output.
  - External input/gate: explicit provider-mutation authorization plus Amazon endpoint, matching region, dotless bucket, 12-digit expected bucket owner, unique run ID, both environment-only temporary access/secret/session credential sets, identity-policy receipt, never-enabled versioning evidence, abandoned-multipart lifecycle backstop, and the exact production-architecture image digest with fixed bundle path/provenance/revision/hash/bytes/unique-valid-root count, regular read-only image ownership, and no replacing runtime mount. Every item must be present before the first request; secrets are never recorded; bundle bytes and roots must satisfy D4's half-ceilings.
  - Proof: the deterministic strict-loader prerequisite proves wrong-host and ambient-only denial; the first exact signed request through the same production non-nil pool proves the Amazon public chain/hostname. Register direct test cleanup before the first mutation. Under the owned prefix exercise single create-only collision/replace, serial multipart replace and forced cleanup, validated-EOF download, metadata with unambiguous and concealed absence identities, absent/existing delete, and presigned GET twice plus method/key/query/header mutations; independently verify exact bytes, portable metadata, and cleanup while the adapter requires provider checksum evidence. TD-003/007/010/011 remain the separate deterministic attempt, retry, and known-answer checksum authority. After adapter assertions, the test-only direct SDK path lists/removes only owned objects/uploads and requires final empty readback. Run `REQUIRE_S3_CONFORMANCE=1 make test-s3-conformance-amazon`; a missing/stale image or no-override input, skipped case, field mismatch, or ambiguous result fails Amazon only.
  - Reopen if: trailer/checksum/one-attempt/cleanup mechanics fail while an equivalent mechanism may exist — Technical Design first; only the design-wide Specification conditions may reopen the common contract. Provider/client/tuple drift requires a fresh H11 receipt.

### H12: Cloudflare R2 exact-tuple support certification

- Status: optional adopter-owned handoff; unverified until its own R2-only receipt.
- Recipient instructions: only an authorized adopter may run this handoff after every external input below is available. Its receipt cannot certify Amazon, a deployed path, or another adopter.
  - Source: `spec.md` R1 and success criterion 3; `design/overview.md` provider matrix and reopen conditions; `test-plan.md` TD-018.
  - Owner/surface/resources: fixed `//go:build integration`, external-package `test/s3conformance/conformance_test.go` R2 entrypoint and distinct `Makefile` target that supplies `-tags=integration`; one pre-existing operator-owned R2 bucket, primary and concealment static-or-temporary identities, and unique `conformance/cloudflare_r2/<run-id>/` prefix/uploads. The test may mutate only that prefix/uploads and never provisions or changes provider controls or uses a custom domain.
  - Local prerequisite: T1-T10 accepted deterministic template capability, conformance entrypoint, aggregate gates, and profile output.
  - External input/gate: explicit provider-mutation authorization plus exact 32-hex default/EU/FedRAMP R2 account S3 endpoint, region `auto`, dotless bucket, unique run ID, both environment-only static-or-temporary credential sets, identity-policy receipt, lifecycle/versioning evidence, abandoned-multipart lifecycle backstop, and the exact production-architecture image digest with fixed bundle path/provenance/revision/hash/bytes/unique-valid-root count, regular read-only image ownership, and no replacing runtime mount. Every item must be present before the first request; secrets are never recorded; bundle bytes and roots must satisfy D4's half-ceilings.
  - Proof: the deterministic strict-loader prerequisite proves wrong-host and ambient-only denial; the first exact signed request through the same production non-nil pool proves the R2 public chain/hostname. Register direct test cleanup before the first mutation. Under the R2 prefix exercise byte-exact accepted keys, single create-only collision and replace, serial multipart replace and forced cleanup, validated-EOF download, metadata with unambiguous and concealed absence identities, absent/existing delete, and presigned GET twice plus method/key/query/header mutations through the account S3 authority; independently verify exact bytes, portable metadata, and cleanup while the adapter requires provider checksum evidence. TD-003/007/010/011 remain the separate deterministic attempt, retry, and known-answer checksum authority. Immediate cleanup remains visible `pending`; the test-only direct path removes only owned objects/uploads and requires final empty readback. Run `REQUIRE_S3_CONFORMANCE=1 make test-s3-conformance-r2`; a missing/stale image or no-override input, skipped case, field mismatch, or ambiguous result fails R2 only and cannot consume H11.
  - Reopen if: trailer/checksum/one-attempt/cleanup mechanics fail while an equivalent mechanism may exist — Technical Design first; only the design-wide Specification conditions may reopen the common contract. Provider/client/tuple drift requires a fresh H12 receipt.
