# Data, Cache, Security, And Distributed Test Obligations

## When To Load
Load this when test strategy must cover SQL transactions, migrations, query shape, cache key/fallback/staleness behavior, tenant or authorization boundaries, outbox/inbox, dedup, replay, ordering, compensation, reconciliation, or mixed-version distributed behavior.

## Source Grounding
- Use approved data, cache, security, and distributed specs as the source of behavior.
- Use Go database docs to calibrate transaction and context-proof obligations.
- Use OWASP only for security test objective patterns such as object-level authorization; do not import vulnerability examples as product requirements.

## Selected/Rejected Level Examples
| Obligation | Selected level | Rejected level | Why |
| --- | --- | --- | --- |
| Transaction commits all changes or rolls back all changes | Integration | Mock-only unit | The claim is atomic durable state, not just control flow. |
| Query pagination is deterministic under equal sort keys | Integration | API e2e only | The data ordering invariant belongs near the DB/query boundary and needs controlled data shape. |
| Cache key includes tenant/scope/version dimensions | Unit for pure key builder plus integration when Redis/TTL/serialization matters | Contract happy path | Happy response cannot prove isolation, stale prevention, or fallback semantics. |
| Cache outage degradation | Integration or component test with controlled cache failure | Live outage e2e | Controlled failure proves the exact fallback and origin-protection behavior. |
| Tenant mismatch or object ownership denial | Contract or integration with multiple actors | Unit-only auth helper | The proof must exercise caller identity and boundary-visible denial. |
| Outbox/inbox dedup or ack-after-durable-state | Integration or process-level component | Unit-only message handler | The durable ordering of state change, message write, ack, and replay behavior is the obligation. |
| Migration/backfill compatibility | Migration validation/integration rehearsal | Unit | The proof concerns schema/data compatibility, idempotence, resumability, and drift. |

## Scenario Matrix Examples
| Surface | Required rows | Selected proof | Pass/fail observable |
| --- | --- | --- | --- |
| SQL transaction | all statements succeed, mid-transaction failure, commit failure if representable, context cancellation | Integration | persisted rows all present or all absent, returned error class, no inconsistent read |
| Pagination/query | empty, first page, last page, invalid cursor, equal sort-key tie, concurrent insert if specified | Integration plus contract if client-visible | stable order, cursor validity, no duplicate/missing row across page boundary |
| Cache | hit, miss, stale entry, corrupt entry, Redis timeout, tenant key mismatch, stampede under parallel miss | Unit and/or integration | returned value, origin call count, cache write/delete, fallback/degraded signal |
| Security boundary | unauthenticated, invalid credential, wrong tenant, wrong object owner, allowed actor | Contract/integration | fail-closed status/error, no data leak, no side effect, audit/metric only if specified |
| Distributed flow | first delivery, duplicate delivery, out-of-order event, retryable error, poison message, replay after restart | Integration/process proof | durable state, inbox/outbox row, ack timing, retry counter, DLQ/escalation, reconciliation output |
| Migration/backfill | expand step, old app/new schema compatibility, resumable backfill, verification gate, contract step | Migration validation/integration | migration applies cleanly, generated SQL drift resolved, backfill idempotent, destructive step blocked until verified |

## Pass/Fail Observables
- Durable-state obligations name the persisted state that proves success and the absence of partial state that proves rollback.
- Cache obligations distinguish correctness, staleness, isolation, fallback, and origin protection.
- Security obligations include negative actor/scope rows and expected denial semantics.
- Distributed obligations include duplicate, replay, ordering, and ack/durable-state observables where applicable.
- Migration obligations include compatibility and verification gates before destructive or irreversible steps.
- Any behavior not already approved by the owning specialist remains a blocker, not a QA invention.

## Exa Source Links
- [Executing transactions](https://go.dev/doc/database/execute-transactions)
- [Canceling in-progress operations](https://go.dev/doc/database/cancel-operations)
- [OWASP WSTG API Broken Object Level Authorization](https://owasp.org/www-project-web-security-testing-guide/latest/4-Web_Application_Security_Testing/12-API_Testing/02-API_Broken_Object_Level_Authorization)
- [Go security best practices](https://go.dev/doc/security/best-practices)
- [OpenAPI Specification v3.0.4](https://spec.openapis.org/oas/v3.0.4.html)

