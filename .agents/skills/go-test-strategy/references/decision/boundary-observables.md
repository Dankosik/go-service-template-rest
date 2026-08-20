# Boundary Observables

## Load When
Load this when the obligation touches REST/OpenAPI, auth or tenant boundaries, idempotent writes, async acceptance, SQL transactions, migrations, cache behavior, timeouts, retries, shutdown, or outbox/inbox replay.

## Decide
- Vary the caller-controlled dimension: credential, tenant, object identifier, idempotency key, cursor, request size, unknown field, operation identifier. A boundary exercised only by the authorized actor is untested.
- Prove idempotency by duplicate side-effect suppression. Equal responses are the cheap half; exactly one durable write is the claim.
- Keep key-mismatch and in-flight-concurrency expectations separate unless the approved contract collapses them.
- For async acceptance, prove what is still rejected *before* the `202` — accepting work is not permission to defer validation — and prove the operation identity a client can poll through to a terminal state.
- Prove transactions on durable state: every row present or every row absent after a representative mid-step failure, plus the error class the caller must still recognize.
- Prove cache behavior at the origin rather than in the response: origin call count, cache write/delete/bypass, tenant-scoped key, and the fallback signal when the cache is unreachable.
- Separate retry classes — transient-then-success, exhausted, non-retryable, and poison for async flows — because each has a different terminal state.
- Prove cancellation twice: the recognizable context-derived error, and the absence of a success side effect after cancellation.
- Prove shutdown with a lifecycle observable — drain marker, joined goroutine, flushed or abandoned work — not with elapsed time.
- Prove migrations on compatibility and resumability: the old application against the new schema, a backfill that resumes, and a destructive step that stays blocked until verified.

## Inspect
A cache read can return the correct value while the entry is stale, because the origin was consulted anyway. Assert the origin call count and the cache write; the returned value alone was never the evidence.

## Reject
- "The mock repository recorded a rollback." The durable boundary is the thing under test, and a mock proves only that the code called a method it would call with no transaction present.
- "The endpoint returns 200, so the cache works." A success response is compatible with stale reads, tenant key collisions, serialization loss, and origin stampede.
- "Test authorization" with only the authorized actor, no missing credential, and no wrong tenant.
- "The migration applies cleanly" as the whole migration claim, with no old/new compatibility and no backfill resumption.

## Reopen
- Durable proof here lives in `./test/...` behind `//go:build integration`. `pgtest.DSN(tb)` and `pgtest.Migrated(tb, …)` create a per-test database on one shared, digest-pinned container, so a real-database row costs little and a mock is rarely the honest choice.
- A fake cache proves serialization and TTL only if the fake implements them; connection-failure and eviction behavior need the real client.

## Prove
Name the observable, the controlled data or failure trigger, and the validation family that would execute it. [validation-commands.md](validation-commands.md) owns the command map.
