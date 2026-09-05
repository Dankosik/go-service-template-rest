# Implementation autonomy: verification

Date: 2026-09-05. Authority: the user approved the Implementation-only changes
after explicitly retaining separate pre-Implementation macro phases. This
candidate changes execution of an accepted plan, not those phases or their
review requirements. No publication, consumer synchronization, active task
packet rewrite, model-policy change, or runtime code change is included.

## Instruction changes

| Observed instruction pressure | Disposition and expected change | Retained boundary |
| --- | --- | --- |
| Codex sends every delegated code defect or failed proof back to Astra for diagnosis. | Conditional method: executors diagnose and repair ordinary failures within the accepted contract. | Lead acceptance; unresolved judgment, invalid oracle, contract conflict, and stalled diagnosis return to the Lead. |
| Every Implementation unit requires a reviewer; the acceptance schema rejects an otherwise valid unreviewed result. | Conditional method: Lead self-review with `not_required` when the shared review trigger is absent. | Independent review for material protected-invariant changes, unresolved material correctness questions, or explicit requirements; outstanding findings cannot be waived. |
| Two accepted units automatically require another reviewer. | Remove the count trigger; review remaining cross-unit risks and invalidated reasoning. | Global Completion and its required proof remain mandatory. |
| Disjoint intra-unit work is phrased as mandatory fan-out. | Conditional method: delegate when independent progress outweighs dispatch and integration cost. | Disjoint owners and locks, stable contracts, serial integration, and one unit acceptance owner. |
| A newly found responsibility may reopen Planning despite serving the same accepted outcome. | Conditional method: ordinary callers, error handling, cleanup, and placement remain Implementation work. | Changed accepted behavior, architecture, task boundary, or proof criterion reopens its smallest owner. |
| Unit acceptance and final delivery share ambiguous aggregate-check ownership. | Clarify executor/Lead unit proof, registrar receipt handling, and one final delivery aggregate. | Required unit Integrated checks cannot be deferred; stale evidence is invalidated; incomplete final proof prevents delivery completion. |

## Candidate and structural proof

The baseline is the dirty working tree immediately before this task, not HEAD.
Eight instruction files changed. Pre-existing bytes outside those files,
including the workflow router and pre-Implementation phase owners, matched the
initial snapshot after editing. The task changes in the ledger contract are
confined to Implementation completion. No skill, carrier, schema file, or
mandatory read owner was added; the existing acceptance schema gained a review
disposition. The eight files grew from 5,113 to 5,491 words. Bootstrap and
catalog branches did not change; these counts do not measure execution speed.

Temporary before/after snapshots, exact task-only patch, ten case inputs, and
candidate hashes are under
`/var/folders/9r/ft1t72w13r765bpf61v9mly00000gn/T/implementation-autonomy-_ixu5sef/`.
These are local audit aids, not runtime or workflow dependencies. The candidate
manifest SHA-256 is
`60172a85498d1f2de1bb4371cb2244731d31f093dd08d5d81603d411daee0cb2`.

- `make template-owned-purity-check`: PASS.
- `git diff --check`: PASS.
- All 36 relative file links in the eight changed files resolve.

The [official OpenAI model guidance](https://developers.openai.com/api/docs/guides/latest-model)
was fetched in this conversation on 2026-09-05. Its scoped-verification and
delegation guidance informed the change; repository authority remains local.

## Interpretation comparison and independent review

Two fresh evidence agents received the same
ten hypothetical cases and brief, with only the supplied before/after policy
root changed. Both requested `gpt-6-astra`, `high`, with no inherited turns.
The tool response does not independently verify effective runtime settings.
A separate fresh reviewer inspected the candidate without the comparison results.
These are static policy interpretations and review, not executed coding tasks,
a speed benchmark, or evidence of unchanged production defect rates.

| Case | Before interpretation | After interpretation |
| --- | --- | --- |
| Executor has an ordinary parser-test failure. | Astra diagnoses and supplies a correction. | Executor diagnoses and repairs directly; Lead accepts. |
| Mechanical unit with sufficient proof and unchanged protected behavior. | Independent unit reviewer required. | Lead self-review and `not_required` permit acceptance after all packet checks. |
| Authorization and monetary replay change; tests pass. | Independent review required. | Same protection retained. |
| Two unrelated accepted units with complete delivery proof. | Additional integrated reviewer required by unit count. | No extra reviewer absent a specific remaining trigger. |
| Retry seam has unproved duplicate-effect safety. | Integrated review and missing proof required. | Same protection, with review targeting the seam. |
| Small versus large disjoint intra-unit edits. | Direct default competes with mandatory fan-out wording. | Lead chooses direct work for the small unit and profitable independent lanes for the large one. |
| Extra caller/fixture cleanup versus invalid contract/oracle. | Caller classification can conflict with the new-responsibility reopen rule. | Routine repair stays in the unit; invalid accepted inputs reopen their smallest owner. |
| T1 has current focused proof; another unit is running. | Unit acceptance can precede final delivery, with mandatory unit review. | Same timing made explicit; review is conditional and final aggregate is not repeated per unit. |
| Missing PostgreSQL proof, production authority, or explicit Technical Design-only boundary. | No unsupported acceptance/effect/downstream phase. | Same boundaries retained. |
| Repaired candidate still changes authorization. | Retain required review/recheck. | Same protection retained. |

The comparison supports the intended routing interpretation and removes the
observed small-unit fan-out and new-responsibility ambiguities. It does not
establish elapsed-time improvement or equal defect rates. The cases did not
execute complete coding tasks or every possible contract transition.

Fresh `policy_review`, requested Astra/high with no inherited turns: **PASS**,
no findings. The reviewer verified the manifest and eight hashes and attempted
protected-risk, acceptance-schema, proof-timing, repair-authority, and integration
review bypasses. Its scope was static consistency; it did not rerun structural
checks or consume the interpretation results. Final source readback matched the
reviewed candidate. Reopen empirical claims only with executed representative
tasks and comparable runtime settings, delivery time, and defect assessment.
