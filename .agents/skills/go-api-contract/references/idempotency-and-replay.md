# Idempotency And Replay

## Load When

Load this when a `POST` or `PATCH` this service exposes can be retried, or when
a client timeout can hide whether a mutation happened.

## Decide

- Decide and publish five fields together — they are the clause: key scope
  (caller or tenant, plus operation and route), key syntax and entropy, TTL,
  what a mismatch answers, and which failures reserve the key. A clause missing
  any of them is not implementable.
- Bind the key to the authenticated caller. Scope is what makes a stored outcome
  unreachable by anyone else; a global key lookup is a cross-caller disclosure.
- Compare a retried payload at the normalized contract level, not by bytes, so
  irrelevant formatting, member order, or applied defaults do not read as a
  different request.
- Same key and equivalent payload after the durable boundary returns an
  equivalent outcome and the same resource identity — not a byte-identical
  response, which timestamps and correlation identifiers make impossible to
  promise.
- Same key and different payload is a stable caller-fixable problem; a retry
  arriving while the first attempt is still in flight is a conflict with retry
  guidance. `422` and `409` are the widely implemented split, and both codes are
  already in the `internal/problem` catalog for an operation that declares them.
- A missing required key is request validation: `400` with `code: bad_request`.
- Reserve the key from the durable boundary onward. A request rejected before it
  — strict decode, schema validation, authorization — leaves the key usable, and
  a key reused past its TTL starts new work.

## Reject

- Citing the `Idempotency-Key` header field as a standard: the IETF draft
  (`draft-ietf-httpapi-idempotency-key-header`) reached revision 07 on
  2025-10-15 and is now expired and archived without becoming an RFC. Write the
  syntax and semantics as this API's published rule; a client cannot look them
  up.
- Reusing the outbound key policy in `internal/infra/httpclient`: that key
  exists so this service may retry a non-idempotent request to a provider. It is
  the caller side of the boundary, owned by `external-api-integration`, and the
  inbound clause is a separate contract with its own scope and TTL.

## Prove

Four consumer-runnable replays: equivalent payload returns the same resource
identity; changed payload returns the mismatch status; a second request while
the first is in flight returns the conflict status; and the same key from a
second caller does not resolve to the first caller's outcome.
