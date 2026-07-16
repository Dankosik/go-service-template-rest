# Resource Representations And Lifecycle

## Behavior Change Thesis

When loaded for resource or representation ambiguity, this file makes the contract follow client-visible business identity and lifecycle instead of handler names, database rows, internal queues, or speculative completeness.

## Decision Rubric

- Choose collection/item, owned sub-resource, or operation resource from client semantics. Use opaque stable IDs and shallow lowercase kebab-case paths with plural collection nouns.
- Distinguish accepted, persisted, externally observable, and terminal business state. Expose separate states or resources only when clients can observe or act on the distinction.
- Define canonical read representation, create/update inputs, server-assigned and read-only fields, and whether omission, explicit `null`, empty, or default have different meaning.
- For `PATCH`, define patch media type, null-as-removal, whole-array replacement, immutable-field behavior, and validation before side effects.
- Encode timestamps as RFC 3339 UTC by default and state precision when ordering or concurrency depends on it. Use exact string/integer/decimal contracts for money, rates, quotas, and high precision; do not silently choose floating point.
- Treat enums as compatibility surfaces. State whether clients must tolerate unknown future values or keep the field open rather than pretending an unstable vocabulary is closed.
- Keep actor identity, tenant scope, public resource references, partner references, idempotency keys, and correlation IDs distinct.
- Do not add media types, states, endpoints, fields, or control surfaces without a concrete caller ambiguity they resolve.

## Evidence And Rejection Tests

Record affected clients, ownership, state transitions, read/write visibility, stable encodings, and compatibility consequences. Reject resource shapes that leak DB keys, queue names, provider status vocabulary, or handler verbs; states clients can never reach or observe; and representations whose null/default/precision behavior would be decided by generated code.
