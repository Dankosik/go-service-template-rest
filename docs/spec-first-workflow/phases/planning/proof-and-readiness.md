# Planning Proof And Readiness

Read before declaring an inline plan or `tasks.md` ready.

## Proof placement

Name the claim before its check. A command is proof only when its expected
observable can establish that claim. Prefer the smallest repository-native
automated check unless accepted proof strategy requires manual observation or
automation cannot establish the oracle.

Attach proof to the earliest unit whose completed output makes its claim true
and require it before acceptance. A later proof unit is valid only for a
cross-unit, deployed, migration, or environment claim that cannot exist
earlier; it names the accepted upstream outputs it consumes and proves only the
integrated claim.

Make these explicit when triggered:

- canonical source before generated or mirrored output;
- regression-proof order and accepted test-design scenario IDs;
- performance workload, scale, amplification or resource constraints, and the
  matching benchmark, load, profile, query-count, or claim-matched proof;
- migration, backfill, rollout, and rollback gates;
- cleanup of replaced code, tests, fixtures, config, docs, skills, or mirrors;
- fresh validation and negative proof for retired identifiers;
- packet mutable owners and exclusive locks sufficient to compute the ready
  frontier without a new concurrency decision, naming only surfaces this unit
  mutates;
- Integrated check omitted unless this unit's postcondition requires
  post-landing proof the focused check cannot give;
- one successful completion condition distinct from blocked stop.

Preserve an accepted example only when it defines behavior or proof. Use local
obligation keys only when dense inputs cannot otherwise be audited from narrow
anchors. A no-implementation disposition cites current authoritative evidence
or an accepted upstream decision plus its proving surface or objective recheck
condition. Keep reconciliation inline unless it is too dense to audit.

## Readiness dry run

Walk the next inline acceptance unit or ledger task packet through its proof
using current inputs. Resolve any later decision that could invalidate that
work. A later unavailable input remains owned and pending; it blocks now only
when the next accepted result would otherwise be unusable or final completion
is being claimed.

Readiness passes only when that rehearsal can reach acceptance using the fixed
plan, cited current inputs, and available mandatory gates without chat history,
unfinished companion work, or a new behavior, mechanism, placement, ownership,
proof, rollout, concurrency, or carrier decision. Do not persist waves. The
Orchestrator recomputes the unit frontier after each result or canonical
transition and immediately dispatches newly ready units; a Lead decides
intra-unit parallelism from current evidence.
