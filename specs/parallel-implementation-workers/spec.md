# Parallel execution of independent implementation tasks

status: ready

## Scope and non-goals

- The implementation/validation/closeout phase may execute multiple ready ledger tasks concurrently when their independence is positively established.
- Each concurrent task remains one task owned by one native Codex App Worker in its own Codex-managed worktree.
- Planning must precompute every task's earliest-safe planned wave from dependency, ownership, mutable-resource, and proof information so implementation does not rediscover the initial schedule.
- Direct work remains one accepted outcome and does not gain artificial task splitting solely to create parallelism.
- This change does not authorize built-in subagents to implement, remove root acceptance, weaken per-task proof, or introduce a second ledger, phase, or parallel status field.
- This change does not target maximum fan-out. It targets the fastest safe bounded execution available from the current App capacity.

## Behavior and contract delta

1. Planning records every ledger task in exactly one earliest-safe planned wave with a short positive independence basis for each multi-task wave.
2. After entering implementation under the single root Goal, the root uses the next planned wave as the default route and performs only a lightweight current-state check on facts that may have changed.
3. The root dispatches ready wave members concurrently only when repository evidence and the reviewed ledger establish all of the following:
   - no task requires another task's unaccepted output or proof;
   - their writable ownership/discovery boundaries do not overlap;
   - they do not share a canonical/generated source pair, migration or rollout sequence, mutable external resource, or non-concurrent proof resource;
   - neither task can change an interface, invariant, or assumption consumed by the other;
   - each result can be reviewed and integrated without choosing behavior omitted from the accepted artifacts.
4. When any independence condition is unknown or false, the root narrows or serializes the affected wave, records the adjustment in the existing ledger or transient execution context, and continues unaffected work. Absence of a dependency edge is necessary but not sufficient proof of safe concurrency. Capacity, completion-order, drift, or implementation-overlap changes alone do not reopen planning.
5. Every dispatched task receives its own outcome-first brief, explicit model and effort, App task identity, and dedicated managed worktree. One Worker still owns one task until root acceptance, upstream reopen, or evidence-backed replacement.
6. Worker results may return in any order. The root reviews every result independently and assembles provisionally suitable deltas in a disposable wave candidate, in a controlled order, without adding an unaccepted delta to the authoritative integration state. A wave task reaches root acceptance and ledger completion only after the full adjusted wave is assembled and every member's proof passes on the combined wave candidate. The root then promotes only the bounded accepted wave delta to the authoritative integration state and records every task's evidence.
7. An unexpected overlap, merge conflict, invalidated assumption, or combined-proof failure holds the whole adjusted wave: no member is accepted, completed, or promoted. Unaffected results remain provisional while the root returns an implementation-owned repair to the owning Worker against the current wave candidate. The wave is reassembled and re-proved before acceptance; the root reopens planning/design/specification only when recovery requires a genuinely new or changed accepted decision.
8. After each adjusted wave is accepted and integrated, the root advances to the next planned wave and rechecks only seams affected by integrated changes. Terminal validation and final integrated-diff review still cover the whole accepted outcome.
9. The ledger carries exact cross-task constraints once, makes writable owners plus mutable or exclusive resources structural, and names exact cross-task handoffs. Task sizing remains outcome-based: split separable ownership, review, failure/recovery, rollback, or proof domains only when each can end in a valid provable state; never split by file count, estimated minutes, or desired fan-out.
10. Before returning, each Worker performs one compact task-local self-check of its assigned criteria, bounded diff, cleanup, and focused proof. The root still owns acceptance.
11. Repeated findings from one causal class stop symptom-level correction: the root names the violated invariant and owner and requires a materially different route, preserving the cumulative evidence frontier and using replacement only under the existing evidence-backed rule.
12. When active execution state may be lost, the root persists one compact `Active wave` block in `tasks.md`; it creates no scheduler artifact. Identical proof commands run once on the frozen combined candidate and may satisfy multiple explicitly mapped claims.
13. Root and Worker share a convergence-first execution contract: the first acceptance inspection batches all currently detectable compatible findings, each correction aims to return an acceptance-ready candidate, the Worker may replace a wrong suggested patch with evidence for the correct owning route, and re-review covers only the correction plus invalidated surfaces. Review and correction are never treated as deliverables, no acknowledgement round or correction artifact is added, and a non-shrinking evidence frontier must change causal model or recovery route rather than repeat unchanged work.

## Invariants and edge cases

- Exactly one root Goal spans the full implementation ledger, regardless of Worker count.
- A task is never assigned to more than one active write Worker; replacement first ends the superseded write lane.
- Concurrent Workers never share one writable checkout or branch.
- No unaccepted Worker delta enters the authoritative integration state; the disposable wave candidate is not task acceptance or ledger completion.
- Same-task corrections stay with that task's Worker unless the existing evidence-backed replacement rule applies.
- Accepted sibling tasks do not become dependencies retroactively merely because they completed earlier.
- A task that edits a canonical source and a task that edits its generated or mirrored output are not independent.
- Shared broad validation commands do not by themselves force sequential implementation when they can run after fan-in, but shared mutable test infrastructure or destructive fixtures do.
- Dirty Local state remains protected; concurrent fan-in integrates only bounded accepted Worker deltas through a safe integration worktree or branch.
- If only one task is safely ready, behavior is the current sequential Worker loop.

## Decisions, constraints, and authorities

- OpenAI's current Worktrees documentation is authoritative for managed-worktree isolation; it supports multiple independent tasks but does not waive integration checks.
- `specs/parallel-implementation-workers/research/references.md` owns the current external evidence boundary.
- Existing task right-sizing remains outcome-based: split separable, independently reviewable outcomes; keep coupled source/generated/test/doc work together. Do not adopt `superpowers`' 2-5 minute step format or make tasks smaller solely to increase Worker count.
- `Global constraints`, `Depends on`, `Owner/surface/resources`, optional exact `Handoff`, and one compact top-level `Planned waves` section are the canonical planning inputs. Do not add per-task parallel status, conflict lists, or a second scheduling artifact.
- Existing accepted-only authoritative integration, root acceptance, same-task correction, model/effort selection, worktree isolation, and evidence-clamped closeout rules remain authoritative. This specification replaces only the single-active-Worker restriction and the requirement to accept one ready task before dispatching another task in the same reviewed safe wave; provisional wave assembly does not weaken accepted-only authoritative integration.

## Success criteria and proof expectations

- A reviewed plan places every task in an earliest-safe planned wave; two or more positively independent ready members cause concurrent native App Worker dispatches in separate managed worktrees, followed by atomic wave acceptance, controlled fan-in, and combined validation.
- A plan with dependencies, overlapping ownership, canonical/generated coupling, shared mutable state, or uncertain independence remains sequential for the affected tasks.
- Unexpected fan-in conflict or combined-proof failure cannot be accepted as successful completion and is routed to the owning Worker or smallest upstream owner.
- Hidden shared databases, ports, environments, migration targets, destructive fixtures, locks, generated pairs, and proof resources prevent unsafe wave membership; oversized tasks are split only across valid independent outcome boundaries.
- An interrupted active wave resumes from the compact ledger state without chat reconstruction; repeated causal classes change the recovery route; Worker self-check and proof-command deduplication reduce avoidable correction and validation work without weakening root acceptance.
- A bounded task with several compatible gaps receives one complete correction package; an evidence-backed Worker can reject a wrong patch hypothesis, and a successful correction reaches immediate root acceptance through delta-aware re-review without a new review lane.
- Workflow evals cover at least one positive concurrent wave, one false-independence case, and preservation of the same-task correction and single-Goal invariants.
- Existing repository instruction, skill-sync, agent, workflow-routing, behavior-manifest, and diff checks pass on the frozen candidate; live behavior claims remain bounded when live eval adapters are unavailable.

## Risks, assumptions, and reopen conditions

- Assumption: native App capacity permits more than one managed-worktree task to run concurrently. Reopen implementation policy if the current App surface cannot launch or observe concurrent tasks reliably.
- Risk: apparently disjoint files may still share an interface or generated source. The positive independence test and combined validation own this risk; uncertainty defaults to sequential execution.
- Risk: too-small tasks increase dispatch, review, and integration overhead. Reopen planning guidance only if representative behavior evals show the existing outcome-sized boundary prevents useful concurrency or causes repeated fan-in failures.
- Reopen technical design if safe fan-in requires a new persistent scheduler, per-task parallel field, or integration artifact rather than the existing ledger's planned waves and transient execution context.
