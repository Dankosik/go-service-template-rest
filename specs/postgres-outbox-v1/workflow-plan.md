# PostgreSQL Transactional Outbox V1

status: accepted

## Goal

Add an optional `OUTBOX=postgres` template pack that atomically persists a
domain mutation and outbound-event intent in PostgreSQL, then publishes it with
an independently deployable, bounded, at-least-once relay. `OUTBOX=none`
physically removes the pack.

## Authority and boundaries

- Base: `origin/main` at `d8b3ee2`; isolated branch
  `codex/postgres-outbox-v1` in this worktree.
- Local edits, non-destructive proof, research, local commits, and workflow
  artifacts are authorized.
- No push, PR, merge, deploy, managed infrastructure, purchase, or other
  external write.
- The pack owns neither broker adapters, inbox processing, nor domain-specific
  events. A relay-owned publisher interface may consume an independently
  selected messaging adapter.
- Implementation may not fork migration mechanics. The current repository
  migration runner is an input whose canonical status must be proven first.

## Observable completion

- `OUTBOX=postgres` is valid only with canonical PostgreSQL persistence.
- A real-PostgreSQL example proves one transaction for state plus event intent.
- A separately runnable relay proves leases, concurrent replicas, retry,
  poison handling, crash recovery, backlog observability, drain, and explicit
  duplicate publication after the publish/mark crash window.
- `OUTBOX=none` purity and repeated initialization byte stability prove every
  outbox-owned source, migration, binary, config, doc, test, command, and
  dependency is absent.
- Canonical migration proof, independent reviews, the required validation
  matrix, one local commit, and a clean task-owned status are closed.

## Current phase

Implementation / Validation / Closeout. Specification, technical design, test
design, and task readiness have passed their required independent reviews.
T1-T6 record the accepted original candidate. T7 is selected after its deep
PostgreSQL/performance/architecture audit found actionable correctness and
scalability gaps. The earlier T6 review history follows: its first whole-candidate review found
duration-overflow, deterministic-time-proof, and workflow-state blockers, which
are repaired. A second review found and this candidate repairs the production
stderr privacy boundary plus the remaining workflow-state drift. A third review
found and this candidate repairs missing database enforcement for finite,
non-zero occurrence time plus invalid workflow status/verdict vocabulary. A
fourth review passed; the first final aggregate then found and this candidate
repairs two NilAway safety findings in the shared transaction receiver and
relay duration helper. A fifth review passed; the next aggregate exposed an
effective-coverage gap. The coverage repair kept the 80% threshold and all
production files in scope, but its fresh review found a design-record mismatch
and a timeout-versus-completion race. This candidate reopens Go ownership to
record the consumer-local fault seam, preserves concrete `*Store` production
composition, and treats every observed attempt timeout as publication failure
even when the publisher then returns `nil`. A deterministic cancellation test
covers that crash boundary. The next fresh review passed; its final aggregate
then rejected unsigned/signed jitter conversions and an unexplained non-security
PRNG. This candidate uses the signed standard-library API, preserves the
maximum-duration bound, and narrowly documents why retry jitter needs spread
rather than unpredictability. A fresh whole-candidate review passed on that
repair. The final serialized matrix then passed: 371 focused tests, 128 race
tests, 34 real PostgreSQL/NATS integration tests, canonical sqlc and migration
checks, migration/image rehearsal, the complete initialization/purity matrix,
`make check-full`, and `git diff --check`. T1-T6 are closed by the single local
commit containing this receipt and its clean post-commit status.

T7 now materializes the current ordered head, uses stable per-statement database
time, preserves SQLSTATE `08007`/`40003` ambiguity, aligns database JSON checks
with the Go envelope contract, and bounds observation freshness, cleanup catch-up,
redrive storage telemetry, and Publisher cleanup. The accepted polling topology,
one in-flight publication per relay, broker-neutral `Publisher`, durable leases,
and explicit at-least-once duplicate window remain unchanged.

## Active artifacts

- `research/`: current-state and primary-source evidence.
- `spec.md`: created after research challenge closes.
- `design/`: system and Go ownership decisions.
- `test-plan.md`: executable failure and concurrency proof.
- `tasks.md`: implementation acceptance ledger.

## Blockers and assumptions

- Migration authority is closed: merged commit `240de21` makes Goose v3.27.3,
  `cmd/migrate`, transactional six-digit SQL migrations, and the current
  source/history/image rehearsal the canonical base-template path.
- The accepted JetStream pack on this base is integration evidence only. The
  outbox pack will consume no concrete broker package and will not make a
  broker a prerequisite for local correctness proof.

## Next action

No in-scope implementation action remains. Keep push, PR, merge, deploy, schema
application, and other external writes outside this local acceptance boundary.

## Completion proof

T7 closes with the updated `tasks.md` receipt, frozen-diff independent PASS,
focused/race/full real-PostgreSQL proof, canonical sqlc/migration/image checks,
the complete profile-purity and `check-full` gates, exact candidate/diff identity,
and final Git status. Benchmark evidence remains a comparative local envelope;
it does not establish a production SLO or authorize rollout.
