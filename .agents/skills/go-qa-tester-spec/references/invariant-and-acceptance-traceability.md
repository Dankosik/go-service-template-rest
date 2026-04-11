# Invariant And Acceptance Traceability

## When To Load
Load this when approved invariants, acceptance criteria, or state-transition rules need to become explicit QA obligations before coding.

## Source Grounding
- Start from approved domain/API/data/reliability/security decisions. QA strategy proves them; it does not invent them.
- Use this reference to keep traceability compact and falsifiable.
- If an invariant's owner, violation outcome, or external behavior is missing, route back to the owning spec instead of filling the gap in test strategy.

## Selected/Rejected Level Examples
| Invariant or acceptance claim | Selected level | Rejected level | Why |
| --- | --- | --- | --- |
| Pure local invariant with deterministic inputs | Unit | Integration | The proof target is the local rule and its invalid cases, not infrastructure. |
| API-visible acceptance criterion | Contract | Internal unit-only | The claim is what a client observes: status, payload, headers, validation details, or async acknowledgement. |
| Persistent invariant enforced by DB constraint or transaction | Integration | Mock-only unit | The proof must exercise source-of-truth state and conflict behavior. |
| Cross-service process invariant | Integration, contract, or reconciliation proof depending on the approved flow | Single local unit | The obligation is convergence, dedup, replay, compensation, or reconciliation, not only one branch. |
| Tenant or object-ownership invariant | Contract or integration with multiple actors/scopes | Same-actor happy path | The fail-closed behavior is visible only when identity/scope is varied. |

## Scenario Matrix Examples
| Invariant or acceptance ID | Proof level | Required rows | Pass/fail observable | Reopen trigger |
| --- | --- | --- | --- | --- |
| `OneActiveExportPerTenant` | Contract plus integration if durable lock/idempotency is storage-backed | first export accepted, duplicate same request returns equivalent response, different request conflicts, concurrent duplicate suppressed | one durable operation, stable operation resource, conflict status where specified | idempotency retention or conflict semantics are not approved |
| `NoCrossTenantRead` | Contract or integration | tenant A reads own object, tenant B attempts tenant A object, unauthenticated request, admin/internal actor if specified | deny/conceal status, no leaked fields, no side effect | tenant source or concealment policy is unresolved |
| `RollbackOnPartialFailure` | Integration | all steps succeed, middle step fails, commit fails if relevant, retry after rollback | no partial persisted state, recognizable error, retry eligibility as specified | transaction owner or failure class is not approved |
| `AsyncEventuallyTerminal` | Integration or component-level process test | accepted, retryable failure, non-retryable failure, poison message, replay after restart | terminal state, retry/DLQ/escalation signal, idempotent replay | terminal states or poison policy are missing |
| `BoundaryValueAcceptedExactly` | Unit or contract depending on visibility | min, max, just below, just above, malformed | accepted/rejected exactly as approved, stable error classification | boundary value is not specified |

## Pass/Fail Observables
- Every critical invariant maps to at least one selected proof level and one scenario row.
- Every acceptance criterion maps to a client-visible, state-visible, message-visible, or persisted observable.
- Every rejected level has an evidence reason, not a taste preference.
- The matrix distinguishes local hard invariants from cross-service process invariants.
- Deferred or residual risks name the missing upstream decision and the phase that must answer it.

## Exa Source Links
- [testing package](https://pkg.go.dev/testing)
- [OpenAPI Specification v3.0.4](https://spec.openapis.org/oas/v3.0.4.html)
- [OWASP WSTG API Broken Object Level Authorization](https://owasp.org/www-project-web-security-testing-guide/latest/4-Web_Application_Security_Testing/12-API_Testing/02-API_Broken_Object_Level_Authorization)
- [Executing transactions](https://go.dev/doc/database/execute-transactions)
- [Canceling in-progress operations](https://go.dev/doc/database/cancel-operations)

