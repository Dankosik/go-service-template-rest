# Lead execution and scheduling refinements

Date: 2026-09-05. Scope: implement the user's three accepted instruction
refinements in the current template checkout. No publication or consumer sync.
This record is separate from the earlier six-change verification; its frozen
candidate and review are not reused as proof of these edits.

## Change boundary

| Pressure and owner | Accepted decision delta | Boundary retained |
| --- | --- | --- |
| Serial delegation duplicates a coherent Lead's context; `docs/spec-first-workflow/phases/implementation.md`. | Lead implements directly by default; delegation needs independent progress, a missing capability, or a context reason. | Independent disjoint lanes remain available; Lead acceptance and fresh review remain separate. |
| A lock mechanically forces strongest capability; `docs/agent-harness.md`. | Locks control scheduling/isolation. Semantic judgment determines capability; expected red tests and diagnosed mechanical errors alone do not escalate. | Weak proof, interacting invariants, conflicting capability signals, missed invariants, and unexplained/repeated causal failures still escalate under the adapter. |
| Saturation lacks a dependency tie-breaker; `docs/spec-first-workflow/phases/planning/ledger-contract.md`. | Reserve finishing lanes for active units; then prefer eligible outputs that unblock waiting work. | Readiness, authority, owner/lock exclusion, full dispatch when capacity permits, and serial landing remain unchanged. |

These are refinements of conditional methods. No role, phase, schema, scheduler,
model default, skill branch, or mandatory read owner was added. The three owners
grew from 2,299 to 2,380 whitespace-delimited words; bootstrap content did not
change. Word count measures instruction size, not loaded context or speed.

The [official Astra guidance](https://developers.openai.com/api/docs/guides/latest-model)
was fetched on 2026-09-05. It informs explicit delegation and scoped verification;
these edits preserve the local Codex adapter's Astra decision-owner policy.

## Fixed comparison and interpretation replay

The baseline is the dirty working tree immediately before this task's edits,
not HEAD. Temporary before/after snapshots, eight identical case inputs, a
manifest, and the task-only patch are in
`/var/folders/9r/ft1t72w13r765bpf61v9mly00000gn/T/lead-scheduling-qmdpfzva/`.
They are audit aids, not durable workflow dependencies.

Two fresh native evidence agents, `lead_policy_before` and `lead_policy_after`,
received the same brief and cases with different supplied policy roots. Both
spawns explicitly requested `gpt-6-astra`, `high`, and no inherited turns.
Available native state did not expose effective model/effort, so this establishes
matching requested settings, not independently verified runtime equality.
The agents had no expected answers or access to the other condition's result.

| Case | Before | After |
| --- | --- | --- |
| Coherent unit; redundant serial Worker | Lead implements directly. | Same result, explicit default and delegation justification. |
| Two independent writable subsets | Fan out; integrate serially; one unit review. | Same result. |
| Mechanical manifest edit with a shared lock | Serialize; generic policy forces strongest capability, with ambiguous Codex effort mapping. | Serialize; lock alone does not change capability; Codex decision ownership remains Astra. |
| Expected regression red / diagnosed syntax error versus causal or money-invariant failure | Ordinary repair is reasonable; escalation wording leaves an ambiguity. Real causal/invariant misses escalate. | Explicit ordinary-error exemption; real causal/invariant misses still escalate. |
| Last slot needed for active unit review; later two ready units fit | Finish review; later dispatch both ready units. | Same result. |
| One free slot, one ready output unlocks waiting work | Either ready unit is admissible; tie-breaker unspecified. | Prefer the ready output that unlocks waiting work. |
| Unlocking unit lacks production authority and overlaps a lock | Hold it; start unrelated eligible work. | Same result. |
| Missing required review/PostgreSQL proof or explicit Test Design-only limit | No acceptance without proof; no implementation across the phase limit. | Same result. |

A concurrent task changed the Codex adapter's Native Map between snapshot
captures. This fourth delta is context only and excluded from this task's patch.
Its cross-repository conditions do not trigger in these local cases, and its
Models section is unchanged. Its delegation wording can still affect model
salience; this is a confounder, not an isolated controlled experiment.
The snapshots also omit some linked owners; exact output-schema conformance was
not tested. One non-executing interpretation per condition cannot establish
delivery speed, statistical reliability, or equivalent code quality.

## Verification

- `make template-owned-purity-check`: PASS.
- `git diff --check`: PASS for the instruction candidate.
- Relative links resolve; all three task-owned files match their frozen hashes.
- Other pre-existing bytes were preserved, with the separately identified
  concurrent Codex adapter edit excluded from an unchanged-bytes claim.
- Fresh `lead_policy_review`, requested Astra/high: PASS, no findings. It
  independently checked candidate identity, eight static boundary falsifiers,
  and the concurrent adapter interaction. It did not rerun the structural gate
  or independently inspect the interpretation outputs; its verdict establishes
  static instruction consistency, not executed workflow performance.

Reviewed SHA-256 identities:

| File | SHA-256 |
| --- | --- |
| `docs/agent-harness.md` | `0b43910fe0ca6f58fa92a47ec3f3db823f2f7da6e5b22140191dcc3dcdd3c596` |
| `docs/spec-first-workflow/phases/implementation.md` | `f132aa39053461193faa91e97b492d67869285162e0b059f6064e044459fd001` |
| `docs/spec-first-workflow/phases/planning/ledger-contract.md` | `1a5634ef23c8f1f8a3979abd3bac3bc70f8ef8665b7a9fa4ffc2cb75b4fcba4f` |

Reopen empirical claims only with representative executed tasks, matched runtime
settings and inputs, elapsed time through acceptance, and independent defect
assessment. No runtime speedup, deployment, or consumer adoption is claimed.
