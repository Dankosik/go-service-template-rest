# Test-Level Selection

## When To Load
Load this when the strategy must choose the smallest honest proving level, compare two or more test levels, justify rejecting broader tests, or decide whether fuzz or race evidence belongs in the plan.

## Source Grounding
- Use the repository's `docs/build-test-and-development-commands.md`, `Makefile`, CI workflows, and task artifacts before naming commands.
- Use Go's `testing` and `cmd/go` docs for what `go test`, subtests, fuzz targets, coverage, race, and focused runs can prove.
- Treat examples here as specification patterns only. Do not turn them into test code or implementation review notes.

## Selected/Rejected Level Examples
| Risk or obligation | Selected level | Rejected level | Why |
| --- | --- | --- | --- |
| Pure deterministic validation rule with no transport, DB, cache, clock, or goroutine boundary | Unit | Integration | The local invariant can be proven without a real external boundary; integration would slow feedback without adding evidence. |
| HTTP status, headers, request decoding, response shape, or OpenAPI drift | Contract | Unit-only | Unit-only tests can bypass generated bindings, middleware, or transport-visible behavior; contract proof checks the public boundary. |
| SQL transaction atomicity, rollback on failure, or query/migration compatibility | Integration | Mock-only unit | The obligation is the database behavior and transaction boundary, not just a branch in application code. |
| Object ownership or tenant mismatch at an API boundary | Contract or integration boundary test | Happy-path unit | The proof needs two actors or tenants and a boundary-visible deny outcome; a happy unit case cannot prove fail-closed behavior. |
| Parser, decoder, serializer, or validator with large input space and cheap deterministic invariants | Fuzz smoke plus seed corpus | Hand-picked unit cases only | Hand-picked cases remain useful, but fuzzing is better at exploring edge-case inputs and retaining regressions as seed corpus entries. |
| Goroutine, shared-state, worker-pool, shutdown, or cancellation behavior | Targeted test under race execution plus deterministic coordination | "Does not panic" e2e smoke | The race detector only sees executed paths, so the strategy must exercise the risky path under `-race`; broad smoke is too weak. |
| Critical composed service path after unit/contract/integration coverage exists | E2E smoke | E2E as primary proof | Smoke proves wiring and deployment assumptions; it should not replace smaller tests that localize exact failure modes. |

## Scenario Matrix Examples
| Obligation | Candidate levels | Selected proof | Matrix rows to require |
| --- | --- | --- | --- |
| New validation rule on request body | Unit, contract | Contract for boundary behavior plus unit only if the pure rule has edge complexity | accepted body, missing required field, malformed body, unknown field if contract says strict, oversized body when limits changed |
| Idempotent create command | Unit, integration, contract | Unit for idempotency policy if pure; contract/integration when persistence or public response semantics matter | first request, same key/same payload replay, same key/different payload conflict, concurrent same-key attempts, expired key if retention is specified |
| Cache fallback behavior | Unit with fake cache, integration with real cache | Integration when serialization/TTL/fallback matters; unit for pure fallback classifier | hit, miss, cache timeout, corrupt entry, stale entry, tenant key mismatch, origin failure while cache unavailable |
| Background worker shutdown | Unit or integration with deterministic fakes, race execution | Targeted worker test under `-race` when shared state or goroutine lifecycle changed | clean drain, cancel before work starts, cancel while blocked, worker error, no send after close, bounded wait |

## Pass/Fail Observables
- Selected level names the risk it proves and the boundary it exercises.
- At least one rejected level is named when the choice is nontrivial, with a specific evidence gap.
- Each level maps to an externally meaningful observable: returned error class, HTTP status/body/header, persisted state, emitted message, visible state transition, cache state, or command artifact.
- E2E smoke is justified by composed-runtime risk, not by avoidance of smaller proof.
- Fuzz is chosen only for deterministic input-heavy logic with a cheap invariant and seed/regression expectation.
- Race validation is paired with scenarios that execute the concurrency path; `-race` alone is not enough if the risky path is not covered.

## Exa Source Links
- [testing package](https://pkg.go.dev/testing)
- [go command test packages](https://pkg.go.dev/cmd/go#hdr-Test_packages)
- [go command testing flags](https://pkg.go.dev/cmd/go#hdr-Testing_flags)
- [Go Fuzzing](https://go.dev/doc/fuzz/)
- [Data Race Detector](https://go.dev/doc/articles/race_detector.html)

