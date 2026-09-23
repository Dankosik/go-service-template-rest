# Whole-project Super Review follow-up

## Accepted outcome

Deliver the 19 accepted recommendations from the 2026-09-23 whole-project
Super Review in one pull request. The reviewed Go source is commit
`cdfb6a876f103a643d92da260aab974bd8a64f5a`; this candidate starts from
`origin/main` at `fe53ad277`, whose intervening commits changed no Go files.
Preserve runtime behavior, external contracts, error matching, template profile
slicing, generated-source ownership, and unrelated dirty work in the original
checkout. This PR does not merge or deploy itself.

The review selected all 233 eligible production Go files. Tests, generated
source, test-only helpers, and unsupported languages were outside its scope.
Implementation should reuse sufficient existing tests and add focused cases for
material changed observations. One assembled validation and independent review
cover the final candidate.

## Recommendation ledger

| ID | Accepted change | Status |
| --- | --- | --- |
| R-001 | Make WorkersRuntime.Bind receive its Workers target and describe post-pool registration | Implemented |
| R-002 | Share genuinely common AST checker helpers and OAuth mapping predicate, retaining transport-specific checks | Implemented |
| R-003 | Remove the forwarding inbound endpoint parser and use the canonical manifest owner | Implemented |
| R-004 | Distinguish unread migration versions from observed version zero in RunResult and terminal reporting | Implemented |
| R-005 | Expose and release cancel for each synchronous service shutdown stage | Implemented |
| R-006 | Narrow objectStorageRuntime to its lifecycle operation and retain separate Store conformance | Implemented |
| R-007 | Separate idempotency ordinary success from stored replay success | Implemented |
| R-008 | Centralize OTLP endpoint provenance and name the shared exporter config key accurately | Implemented |
| R-009 | Clone attributes retained for grouped log replay | Implemented |
| R-010 | Give outbound webhook signing-key length one pair of constants | Implemented |
| R-011 | Explain unknown commit outcome at InTx's caller-facing entry point | Implemented |
| R-012 | Name and document SetupTracing shutdown and partial error results | Implemented |
| R-013 | Explain ambiguous Upload/Delete outcomes at the objectstorage port | Implemented |
| R-014 | Separate constructor checker CLI handling from its AST decision | Implemented |
| R-015 | Name the intermediate River stop condition separately from final cleanup safety | Implemented |
| R-016 | Correct stale consumer counts in runtimeopts rationale comments | Implemented |
| R-017 | Use tracing-specific names for tracing-only bootstrap values | Implemented |
| R-018 | Name jobs.max_workers in the webhook prerequisite diagnostic | Implemented |
| R-019 | Name deliveryAttempt.DeliveryID explicitly | Implemented |

Two observations from the review were rejected: extracting the short readiness
state calculation and extracting the two adjacent gRPC credential guards. They
are outside the implementation list.

The additional `Prepared.Stage` observation is a technical decision to close
within this PR: its caller contract says it must be the final transaction
operation, but the reviewed source and documentation do not establish why.
Investigate the existing invariant, then either document a verified reason or
remove the unsupported ordering requirement while retaining atomic staging,
conflict rollback, and current runtime behavior. Do not invent a rationale.

**Disposition:** The ordering requirement was removed. `Stage` inserts jobs
through the caller's `pgx.Tx`; its code neither commits the transaction nor
depends on the position of later statements. The only Go caller already returns
the staging error to `InTx`, which rolls back on error. The contract now states
the supported invariant: stage jobs and business writes in one transaction and
roll back on any staging error, including a conflict. No runtime path changed.

## Completion

After all changes are assembled, run the matching executable builds and root
module unit suite under the repository validation policy. Resolve one independent
review of the fixed candidate. Create one PR with the exact head and report its
CI state separately from local proof. No merge or deployment is requested.
