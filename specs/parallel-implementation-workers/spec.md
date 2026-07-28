# Parallel execution of independent implementation tasks

status: ready

## Scope and non-goals

- The implementation/validation/closeout phase may execute multiple ready ledger tasks concurrently when their independence is positively established.
- Each concurrent task remains one task owned by one native Codex App Worker in its own Codex-managed worktree.
- Planning records a concurrent wave only when multiple ready tasks have proven independent ownership and resources; sequential tasks rely on dependencies without wave bookkeeping.
- Direct work remains one accepted outcome and does not gain artificial task splitting solely to create parallelism.
- This change does not authorize built-in subagents to implement, remove root acceptance, weaken per-task proof, or introduce a second ledger, phase, or parallel status field.
- This change does not target maximum fan-out. It targets the fastest safe bounded execution available from the current App capacity.

## Behavior and contract delta

1. Planning records a compact wave only for ready tasks proposed for concurrent dispatch, with a short positive independence basis.
2. After entering implementation under the single root Goal, the root uses the next planned wave as the default route and performs only a lightweight current-state check on facts that may have changed.
3. The root dispatches ready wave members concurrently only when repository evidence and the reviewed ledger establish all of the following:
   - no task requires another task's unaccepted output or proof;
   - their writable ownership/discovery boundaries do not overlap;
   - they do not share a canonical/generated source pair, migration or rollout sequence, mutable external resource, or non-concurrent proof resource;
   - neither task can change an interface, invariant, or assumption consumed by the other;
   - each result can be reviewed and integrated without choosing behavior omitted from the accepted artifacts.
4. When any independence condition is unknown or false, the root narrows or serializes the affected wave, records the adjustment in the existing ledger or transient execution context, and continues unaffected work. Absence of a dependency edge is necessary but not sufficient proof of safe concurrency. Capacity, completion-order, drift, or implementation-overlap changes alone do not reopen planning.
5. Every dispatched task receives the accepted `tasks.md` path and task ID plus only current facts absent from that entry when the exact ledger revision is visible; otherwise it receives the full outcome-first brief inline. Each dispatch also selects explicit model and effort, App task identity, and dedicated managed worktree. One Worker still owns one task until root acceptance, upstream reopen, or an execution stall or invalidated base permits evidence-preserving replacement.
6. Worker results may return in any order. Every result passes the implementation phase's [Scope Lock](../../docs/spec-first-workflow/phases/implementation-validation-closeout.md#scope-lock) before wave assembly. The root assembles only scope-valid provisional deltas in a disposable wave candidate, in controlled order, without changing authoritative integration state. Findings become available only after the full adjusted wave is frozen, reviewed once, and exercised by its initial mapped proof. A wave task reaches acceptance and ledger completion only when that frozen finding set is empty and the combined proof passes; the root then promotes the bounded wave delta and records each task's evidence.
7. Before the finding set is frozen, an unexpected overlap, merge conflict, or invalidated assumption holds the whole adjusted wave: no member is accepted, completed, or promoted, and a bounded assembly repair returns to the owning Worker. Failure of the initial combined proof enters the frozen finding set. After that freeze, any correction conflict or proof failure rejects the entire correction delta and restores the preceding baseline instead of being fixed forward. The root reopens planning/design/specification only when recovery requires a genuinely new or changed accepted decision.
8. Each adjusted wave's single acceptance review and mapped proof cover the seams affected by its integration. After the wave is accepted and integrated, the root advances to the next planned wave. Completion reuses accepted review dispositions and runs claim-mapped terminal validation; it does not open a second whole-diff review or a new finding set. A terminal failure stops and reopens its narrowest owner instead of starting another implementation correction set.
9. The ledger carries exact cross-task constraints once, makes writable owners plus mutable or exclusive resources structural, and names exact cross-task handoffs. Task sizing remains outcome-based: split separable ownership, review, failure/recovery, rollback, or proof domains only when each can end in a valid provable state; never split by file count, estimated minutes, or desired fan-out.
10. Before returning, each Worker performs one compact task-local self-check of its assigned criteria, bounded diff, cleanup, and focused proof. The root still owns acceptance.
11. Worker liveness and correction recovery follow the implementation phase's [Progress](../../docs/spec-first-workflow/phases/implementation-validation-closeout.md#progress) and [Diagnostic Gate](../../docs/spec-first-workflow/phases/implementation-validation-closeout.md#diagnostic-gate). A wave adds no repair or replacement authority; replacement remains limited to an execution stall or invalidated base.
12. When active execution state may be lost, the root persists one compact `Active wave` block in `tasks.md`; it creates no scheduler artifact. Identical proof commands run once on the frozen combined candidate and may satisfy multiple explicitly mapped claims.
13. Each disposable wave candidate follows the implementation phase's [Monotonic Acceptance](../../docs/spec-first-workflow/phases/implementation-validation-closeout.md#monotonic-acceptance). The wave-specific delta is only that the frozen candidate and proof include every assembled member and affected integration seam.

## Invariants and edge cases

- Exactly one root Goal spans the full implementation ledger, regardless of Worker count.
- A task is never assigned to more than one active write Worker; replacement first ends the superseded write lane.
- Concurrent Workers never share one writable checkout or branch.
- No unaccepted Worker delta enters the authoritative integration state; the disposable wave candidate is not task acceptance or ledger completion.
- Same-task corrections stay with that task's Worker; replacement is limited to an execution stall or invalidated base and continues the same exact brief from the frozen candidate.
- Accepted sibling tasks do not become dependencies retroactively merely because they completed earlier.
- A task that edits a canonical source and a task that edits its generated or mirrored output are not independent.
- Shared broad validation commands do not by themselves force sequential implementation when they can run after fan-in, but shared mutable test infrastructure or destructive fixtures do.
- Dirty Local state remains protected; concurrent fan-in integrates only bounded accepted Worker deltas through a safe integration worktree or branch.
- If only one task is safely ready, behavior is the current sequential Worker loop.

## Decisions, constraints, and authorities

- OpenAI's current Worktrees documentation is authoritative for managed-worktree isolation; it supports multiple independent tasks but does not waive integration checks.
- `specs/parallel-implementation-workers/research/references.md` owns the current external evidence boundary.
- Existing task right-sizing remains outcome-based: split separable, independently reviewable outcomes; keep coupled source/generated/test/doc work together. Do not adopt `superpowers`' 2-5 minute step format or make tasks smaller solely to increase Worker count.
- `Global constraints`, `Depends on`, `Owner/surface/resources`, optional exact `Handoff`, and an optional compact top-level `Planned waves` section are the canonical planning inputs. Do not add a wave for sequential tasks, per-task parallel status, conflict lists, or a second scheduling artifact.
- Existing accepted-only authoritative integration, root acceptance, same-task correction, model/effort selection, worktree isolation, and evidence-clamped closeout rules remain authoritative. This specification replaces only the single-active-Worker restriction and the requirement to accept one ready task before dispatching another task in the same reviewed safe wave; provisional wave assembly does not weaken accepted-only authoritative integration.

## Success criteria and proof expectations

- A reviewed plan dispatches two or more positively independent ready members concurrently only when it records a real wave; other tasks remain dependency-ordered and sequential.
- A plan with dependencies, overlapping ownership, canonical/generated coupling, shared mutable state, or uncertain independence remains sequential for the affected tasks.
- An initial fan-in conflict or combined-proof failure cannot be accepted as successful completion and is routed to the owning Worker or smallest upstream owner; the same failure on a correction delta rejects that delta and restores the preceding baseline.
- Hidden shared databases, ports, environments, migration targets, destructive fixtures, locks, generated pairs, and proof resources prevent unsafe wave membership; oversized tasks are split only across valid independent outcome boundaries.
- An interrupted active wave resumes from the compact ledger state without chat reconstruction; per-task correction uses the canonical Progress and Diagnostic Gate states; Worker self-check and proof-command deduplication reduce avoidable correction and validation work without weakening root acceptance.
- A bounded task with several compatible admissible gaps receives one complete frozen finding set; an evidence-backed Worker can reject a wrong patch hypothesis, and a successful correction reaches immediate root acceptance through delta-only correction verification without a new review lane.
- `git diff --check` passes on the frozen candidate. Instruction review covers consistency and stale references; it does not claim live model adherence without an external behavior evaluation.

## Risks, assumptions, and reopen conditions

- Assumption: native App capacity permits more than one managed-worktree task to run concurrently. Reopen implementation policy if the current App surface cannot launch or observe concurrent tasks reliably.
- Risk: apparently disjoint files may still share an interface or generated source. The positive independence test and combined validation own this risk; uncertainty defaults to sequential execution.
- Risk: too-small tasks increase dispatch, review, and integration overhead. Reopen planning guidance only if representative behavior evals show the existing outcome-sized boundary prevents useful concurrency or causes repeated fan-in failures.
- Reopen technical design if safe fan-in requires a new persistent scheduler, per-task parallel field, or integration artifact rather than the existing ledger's planned waves and transient execution context.
