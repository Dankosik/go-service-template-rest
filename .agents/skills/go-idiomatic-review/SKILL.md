---
name: go-idiomatic-review
description: "Use when changed Go may violate language or standard-library contracts for errors, context API or lifetime, nil and zero values, receivers, method sets, aliasing, resources, or exported APIs; Own Go-semantic correctness; Skip when behavior is correct but readability, whole-diff structure, or package ownership is the primary issue."
---

# Go Idiomatic Review

Load the [shared specialist contract](../specialist-contract.md) for common selection, scope, evidence, reference, return, and handoff mechanics; apply the domain-specific rules below.

## Target And Invariants

Review Go language, standard-library, and exported-surface semantics as correctness, not style. Read the changed code, affected tests, accepted intent, and effective Go version before version-sensitive findings.

- Preserve deliberate error identity and inspectability with sentinels, typed or joined errors, `%w`, `errors.Is`, and version-appropriate `errors.As`/`errors.AsType`; reject string matching, log-and-swallow, and hidden success.
- Keep caller-owned `context.Context` first and flowing through request work; reject nil or stored call-scoped contexts, unjustified replacement with `Background`, and derived contexts whose owner never cancels them.
- Match receivers and method sets to mutation, identity, interface satisfaction, and copy safety; reject copies of must-not-copy state and account for aliasing in buffers and slice-backed fields.
- Treat typed nil, nil map writes, nil channel blocking, zero-value usability, and observable nil-versus-empty behavior as contracts.
- Treat slices, maps, `[]byte`, buffers, headers, and URL values as mutable ownership surfaces; copy only where isolation is required and never assume map iteration order.
- Require correct resource lifetime and completion probes, including body/file/rows close, `rows.Err`, `scanner.Err`, cancel functions, and timer/ticker stop or reset behavior.
- Prefer current builtins and stdlib when a wrapper adds no compatibility, ownership, normalization, or domain contract. Keep exported APIs small, documented, compatible, consumer-oriented, and concrete unless an interface is a real behavior seam.
- For an approved pattern, verify explicit Go control flow, narrow interfaces, context-aware I/O, and simple composition. Treat framework layers, broad interfaces, hidden goroutine lifetimes, managers, or factories as idiomatic defects only when they violate a Go/stdlib contract; local readability and whole-diff overbuild remain neighboring axes.
- For custom infrastructure, runtime dependencies, or material abstractions, report missing live-choice evidence required by the [research method](../../../docs/spec-first-workflow/phases/research.md#method) as ownership/design risk, not Go style.

Use current release notes, the Go specification, pkg.go.dev, Go Code Review Comments, and official Go posts for version-sensitive claims; Effective Go is useful but not current release authority.

## Symptom-Driven References

| Pressure | Load |
| --- | --- |
| Error wrapping, identity, inspection, cancellation mapping, or hidden success. | [errors-and-contracts-review.md](references/errors-and-contracts-review.md) |
| Stored, nil, replaced, omitted, derived, or uncancelled context. | [context-and-lifetime-review.md](references/context-and-lifetime-review.md) |
| Receivers, method sets, interface satisfaction, value copies, sync fields, buffers, or pointer-to-container shapes. | [receivers-methodsets-and-copy-safety.md](references/receivers-methodsets-and-copy-safety.md) |
| Typed nil, nil containers/channels, constructors, zero values, or nil-versus-empty behavior. | [nil-zero-value-and-typed-nil.md](references/nil-zero-value-and-typed-nil.md) |
| Mutable containers, cloning, headers/URL values, aliasing, or map ordering. | [slices-maps-buffers-and-ownership.md](references/slices-maps-buffers-and-ownership.md) |
| Close/error probes, files, rows, scanner, body, cancel, timer/ticker, partial reads, or defer scope. | [resource-closure-and-iteration-probes.md](references/resource-closure-and-iteration-probes.md) |
| A local helper may duplicate current builtins or stdlib. | [stdlib-first-modern-go-review.md](references/stdlib-first-modern-go-review.md) |
| Exported names, docs, packages, constructors, interfaces, options, signatures, or compatibility. | [exported-api-and-interface-shape.md](references/exported-api-and-interface-shape.md) |

When pressures overlap, choose by the violated contract: context lifetime versus error inspectability, or mutable aliasing versus whether stdlib can replace the helper.

## Findings And Escalation

Each finding names the concrete Go/stdlib rule, observable correctness, diagnosability, compatibility, ownership, or maintenance impact, validation signal, and Go-version/source anchor when relevant. `critical` means confirmed Go-level panic, corruption, or operational failure; `high` means strong correctness, exported-contract, mutable-ownership, or must-not-copy risk.

Hand off concrete goroutine/channel/lock/atomic protocols to `go-concurrency-review`; transaction/query/cache semantics to `go-db-cache-review`; auth, tenant, injection, SSRF, secret, or abuse depth to `go-security-review`; benchmark/allocation/hot-path proof to `go-performance-review`; proof completeness to `go-test-review`; and package ownership to `go-implementation-ownership-review`. Escalate to the nearest specification owner when correction would define or change public/transport behavior, zero-value or compatibility promises, package ownership, or reliability/security/data/domain/concurrency/distributed policy.
