# Autonomous instruction follow-through: verification

Date: 2026-09-04. Accepted outcome: users supply business intent; agents own
technical choices, consultation, recovery, and completion through the applicable
workflow. Explicit phase boundaries and actual authority limits remain intact.

## Candidate and source boundary

The [candidate manifest](candidate-manifest.json) binds 13 integrated instruction
files. Its SHA-256 is
`1a68f25851d6d390c80340651c50061365b85a0bf58ab787c1ed16b1e60e49f4`.
The [instruction delta](instruction-delta.patch) compares their frozen baseline
with the final candidate. The baseline was captured from the working tree at
HEAD `841fbcb75b922e4ac683a4a33ea7c6338e664c3e`, including then-existing edits.
Compatible concurrent instruction changes were preserved; the delta is an
integrated comparison, not attribution of every changed line to this task.

Current official [model guidance](https://developers.openai.com/api/docs/guides/latest-model)
resolved to GPT-6 Astra. Its [prompting guidance](https://developers.openai.com/api/docs/guides/latest-model/gpt-6-astra.md#prompting-best-practices)
informed persistence, instruction priority, delegation, and verification
calibration. Guidance and defaults for still-selected GPT-5.6 roles remain.

## Design decision

The existing root coordinates cross-phase continuation, with a fresh actor
owning each phase. It brokers bounded specialist/reviewer requests when native
nesting is unavailable; the phase actor keeps decision, repair, and transition
ownership. At ready Planning, Implementation selects the fixed-unit Lead or
the existing Ledger Orchestrator. No new role, skill, scheduler, journal, or
user-visible task is required.

Broadening the Implementation-only Orchestrator was rejected because it would
conflate continuation with canonical ledger authority. Independent specialist
consultation closed this choice. Technical objections must be resolved against
evidence and constraints; a vote or unanimity does not prove a decision.

## Comparative decision replays

[Eight cases](replay-cases.json) were presented to fresh, isolated-context native
subagents with the same read-only replay brief and `high` reasoning effort:

- `autonomy_probe_a`: Astra, frozen baseline.
- `autonomy_probe_b`: Astra, frozen instruction candidate.
- `autonomy_probe_c`: Sol, the same instruction candidate.

Each run interpreted simulated authoritative state and proposed next actions.
No expected answers or comparator outputs were supplied to the actors. These
are policy interpretation probes inside the current native harness, not fully
isolated application executions. Shared harness instructions remain a possible
confounder. One run per condition is insufficient for a performance estimate.

| Case | Baseline / Astra | Candidate / Astra | Candidate / Sol | Observed retention |
| --- | --- | --- | --- | --- |
| Full outcome before ledger | Continue | Continue | Continue | Fresh Technical Design actor; root retains continuation. |
| Technical disagreement | Continue | Continue | Continue | Agent selects existing PostgreSQL from constraints, preserves independent review. |
| Business ambiguity | Needs user | Needs user | Needs user | Refund entitlement goes to user; independent inspection continues. |
| Explicit phase-only request | Complete | Complete | Complete | Stop after reviewed Specification. |
| Documentation typo | Complete | Complete | Complete | No extra tests, review, or confirmation. |
| Mid-task steering | Continue | Continue | Continue | Reopen export contract/proof; preserve independent cancellation work. |
| Missing external authority | Continue preparation | Continue preparation | Continue preparation | Prepare concrete local candidate and costs before requesting launch authority. |
| Existing deployment authority | Continue | Continue | Continue | Reuse granted authority despite actor change and skill recommendation. |

All conditions preserved the expected broad decisions. There is no measured
behavioral improvement or evidence here to replace role-model defaults. The
candidate makes continuation and ownership explicit; the baseline also passed.
Architectural correctness of an implemented booking mechanism was not tested.

Replays used scoped candidate
`6b685b7e7330c03909e36f7c86fb32e177208c25ddb4ae16d0a43449c83cc967`.
The final candidate only removes a duplicated stop-attribution clause from the
specialist contract; the same requirement remains in AGENTS.md. Reuse is limited
to semantic retention, subject to independent delta review.

## Structural proof and independent review

- `git diff --check`: PASS before candidate freeze.
- `make template-owned-purity-check`: PASS on the replay candidate, including
  role/config/skill parity and template sync behavior. Make emitted an incidental
  missing-Go warning; the instruction gate completed successfully.
- Fresh `autonomy_candidate_review`: PASS on the fixed 13-file replay candidate;
  no findings. It checked hashes and traced continuation, review nesting,
  technical recovery, authority, and phase boundaries.
- Final clause-consolidation delta review: PASS, no findings; all final manifest
  hashes verified. The inherited stop-transparency and parent-recovery rules
  remain intact.
- `make template-owned-purity-check`: PASS again on the final snapshot after that
  delta; `git diff --check` also passed on the current working tree.

An attempted isolated `codex exec` replay with CLI `0.153.0-alpha.5`, Astra/high,
ephemeral state, and read-only sandbox failed before inference with HTTP 401
(missing authorization). No credentials or account configuration were changed.
No CLI model result, actual multi-phase delivery, database proof, deployment,
or spending is claimed. Reopen live verification when an authorized isolated
runner is available; compare repeated representative tasks with unchanged
model, effort, tools, and inputs before claiming a behavioral gain.
