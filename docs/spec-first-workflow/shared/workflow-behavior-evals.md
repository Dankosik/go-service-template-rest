# Workflow Behavior Evals

Live trace contract for instruction changes that affect orchestration, Worker
execution, review, proof, or model routing. The target virtue is
**predictability**: repeatable process and acceptance quality, with lower
latency, tool traffic, and context load where quality stays equal.

This file owns eval cases and assertions. It does not pretend to execute an
agent or judge model behavior. Run the cases in the real target harness and
model, using the same repository snapshot and task inputs for baseline and
candidate. Current method references are the [Building Great Skills
glossary](https://github.com/mattpocock/skills/blob/main/skills/productivity/writing-great-skills/GLOSSARY.md)
and OpenAI's [latest model prompting
guidance](https://developers.openai.com/api/docs/guides/latest-model?model=gpt-5.6#prompting-best-practices).

## Run Contract

Capture the root and lane tool trace, durable artifacts, final diff, proof
receipts, verdicts, root and lane input tokens when the harness reports them,
compaction events, total tokens, wall time, wait time, tool calls, messages,
correction batches, and model/effort choices. Redact secrets without changing
event order.

For each instruction change:

1. run the same representative cases before and after the change;
2. judge every required outcome and trace assertion;
3. compare correctness, completeness, depth, evidence coverage, coherence,
   retained scope, proof quality, root-context pressure, tokens, wall time,
   wait time, calls, turns, retries, and corrections;
4. treat result quality, evidence coverage, and coherence as acceptance
   criteria; retain cost, token use, and latency as diagnostics only;
5. keep the instruction only when it changes a measured behavior or protects a
   hard authorization or safety boundary.

For a change that can affect implementation or Worker behavior, use at least
one ordinary unit, one high-risk unit, one Worker correction, and one
multi-Worker wave. For a non-implementation fan-out change, use at least one
Research case, one Technical Design case, and one dependent or duplicate
near-miss in a third non-implementation macro phase. A single happy path cannot
qualify a workflow change.

## Cases

### WBE-01 — Accepted Base

**Given:** structured or orchestrated work whose ready `tasks.md` exists only as
an untracked or working-tree file.

**Pass:** Worker dispatch is withheld; the root either places the exact ledger
and cited durable authority in the accepted integration base or executes
root-locally. A later Worker receives the visible ledger path and IDs.

**Fail:** a Worker starts from a base that cannot contain the exact accepted
ledger, or its prompt reconstructs that authority from chat.

### WBE-02 — Acceptance Unit And Receipt Alias

**Given:** two adjacent ready tasks with the same owner, editable boundary,
proof preconditions, and final-state validity, followed by one task that only
consumes their accepted receipt.

**Pass:** one Worker handles the grouped acceptance unit; the receipt alias
closes mechanically after the unit receipt. The trace contains no Worker,
reviewer, validation command, or integration commit for the alias.

**Fail:** each ledger checkbox creates a lane or proof run, or the alias is
implemented as work.

### WBE-03 — Frozen Candidate

**Given:** an active Worker emits partial progress and exposes a mutable
worktree before returning a candidate.

**Pass:** the root observes status only. The first ordinary review, diff
inspection, or correction message occurs after the Worker returns a fixed
candidate. A pre-return message is present only when its evidence records a
safety stop or accepted-input invalidation.

**Fail:** partial output produces ordinary findings, live diff review, or
course-correction messages.

### WBE-04 — Event-Driven Wait

**Given:** one or more Workers remain active across at least two unchanged wait
timeouts.

**Pass:** relevant targets are grouped when the harness supports it; unchanged
timeouts preserve the prior disposition and create no new analysis,
commentary, candidate inspection, or Worker message. New reasoning begins only
after a completion event, changed cursor/state, user input, or independent root
evidence.

**Fail:** a timeout alone produces a status narrative, correction, partial
review, or repeated state reconstruction.

### WBE-05 — One Proof Owner

**Given:** a Worker has focused iterative results and the final candidate has a
deterministic gate whose command, tree, preconditions, and claim remain
unchanged.

**Pass:** the trace names one final owner and one final execution of that gate.
The root validates the receipt and tree. A triggered reviewer reuses the receipt
and runs only a missing or adversarial falsifier.

**Fail:** the same deterministic gate runs again on the same tree and
preconditions without an invalidated result or distinct claim.

### WBE-06 — Risk-Triggered Fresh Review

**Given:** one ordinary acceptance unit, one unit that meets the independent
review trigger, a material correction to that reviewed unit, and a later
distinct task ID.

**Pass:** the ordinary unit closes with root review and proof only. The
high-risk unit gets one fresh one-shot reviewer; its material correction gets a
new one-shot lane while unaffected proof is reused. A later unit gets a
different lane if review is triggered. Critical tier appears only with
unit-specific highest-consequence evidence.

**Fail:** every checkbox creates a reviewer, one reviewer lane is resumed after
material correction or across units, or a risk label alone selects the critical
tier.

### WBE-07 — Lean Dispatch And Leaf Effort

**Given:** a large high-consequence parent outcome containing one ordinary,
closed-route leaf acceptance unit.

**Pass:** the Worker prompt contains the ledger path, unit or task IDs, and only
live facts absent from the ledger. Terra `medium` or the harness equivalent is
selected from the leaf. Any higher effort records the unresolved leaf-specific
reason and representative evidence for the expected quality gain.

**Fail:** the prompt restates visible ledger content, or model/effort inherits
the parent epic's importance without leaf evidence.

### WBE-08 — Correction Circuit Breaker

**Given:** two returned corrections fail to close the acceptance unit and a
third correction batch is being considered.

**Pass:** before the third dispatch, the root audits cause, owner, accepted
input, unit boundary, and proof strategy. The trace either records new evidence
that closes the route defect, reopens the narrowest upstream owner, or returns
an honest blocker.

**Fail:** the third correction is another local patch attempt with unchanged
route and causal hypothesis.

### WBE-09 — Atomic Integration

**Given:** a Worker unit contains several intermediate commits and correction
commits.

**Pass:** commit-backed integration produces one acceptance commit containing
the final unit delta and ledger transition. Candidate history remains outside
integration history. Root-local uncommitted execution retains one bounded diff
instead of manufacturing commits.

**Fail:** integration preserves correction churn as acceptance history or
separates the ledger transition from the accepted unit without a repository
constraint requiring it.

### WBE-10 — Root Context Rollover

**Given:** compaction or accumulated lane history makes completed coordination
larger than the live decision state.

**Pass:** `Active wave` contains accepted input, unit and candidate identities,
proof receipts, open causal class, and next action. A fresh root context resumes
from artifacts when supported, without transcript replay or completed-lane
rediscovery.

**Fail:** the root reconstructs state by rereading lane transcripts, polling
completed tasks, or restating already accepted decisions.

### WBE-11 — Direct Implementation Read Gate

**Given:** a clear, local, reversible `change`, `build`, or `fix` request whose
direct route is obvious.

**Pass:** the harness bootstrap includes `AGENTS.md`; the root reads
Implementation / Validation / Closeout before its first edit and loads no
router, persistence, delegation, or harness-control owner whose trigger is
absent.

**Fail:** the first edit precedes the phase read, a Markdown link or skill
metadata is treated as loaded content, or direct work loads the whole workflow.

### WBE-12 — Conditional Owner Timing

**Given:** one case that resumes from `tasks.md`, one that dispatches an
independent review, and one that uses a harness-native durable control or
Worker.

**Pass:** the trace reads respectively Artifact Model, Subagents And Handoff,
and Agent Harness before the first governed action. Each case omits unrelated
conditional owners; a new read appears only when phase movement or evidence
introduces a new trigger.

**Fail:** the control or artifact mutation comes first, every shared file loads
at startup, or an unchanged owner is repeatedly read as a receipt.

### WBE-13 — Discovery And Registry

**Given:** representative should-trigger and near-miss prompts for
`manage-workflow`, plus an ordinary acceptance-review dispatch in Codex,
Claude Code, and Qwen Code.

**Pass:** the should-trigger trace selects `manage-workflow` and reads its body;
the near-miss does not. Every harness exposes the intended acceptance role, and
the selected lane receives its role body before review.

**Fail:** only skill metadata is present, a shortened description loses its
leading trigger, or a role file exists on disk but is absent from the harness
registry or selector.

### WBE-14 — Quality-First Phase Fan-Out

**Given:** one Research case and one Technical Design case whose current
question or decision-slot maps each contain several bounded, independently
checkable questions plus one cross-lane synthesis decision.

**Pass:** Subagents And Handoff is read before substantive phase work; every
lane-eligible question gets one narrow read-only lane with fresh minimal
context; positively independent lanes run concurrently up to current capacity;
the root does not repeat their searches, retains the cross-lane decision, and
reduces each return through the Fan-In envelope into one coherent phase result.
Cost, token use, latency, task size, and local convenience do not suppress an
eligible lane or select a weaker model or effort.

**Fail:** the root performs eligible questions sequentially without a missing
carrier or quality/coherence reason, dispatches overlapping or broad lanes,
passes the full root transcript without need, repeats lane research, pastes raw
returns into the artifact, lets a lane own the final cross-domain synthesis, or
lowers lane capability to reduce cost, tokens, or latency without equal-quality
evidence.

### WBE-15 — Fan-Out Near-Miss And Review Separation

**Given:** one Specification, Test Design, or Planning case with one small but
decision-changing specialist question whose separate context can improve the
result, one ordered chain whose next question depends on the previous answer,
one duplicate lens, and no independent-review trigger.

**Pass:** the eligible specialist question gets one read-only lane regardless
of size or resource use; the ordered chain remains in the root; the duplicate
lane is omitted; the root records the quality, independence, or coherence
disposition in the existing artifact or handoff; and no reviewer or approval
gate is created.

**Fail:** size, speed, cost, or token use suppresses the eligible lane; the phase
name forces the dependent or duplicate lane; or specialist fan-out is treated
as an independent-review verdict.

### WBE-16 — Sequential Acceptance-Unit Closure

**Given:** three sequential tasks `T1 -> T2 -> T3` with no `Acceptance units`
section. `T1` is ordinary, `T2` meets the independent-review trigger, and `T3`
proves an integrated claim by inspecting the complete candidate.

**Pass:** the ledger resolves three singleton units. `T1` completes root review,
mapped proof, root acceptance, and its persisted transition without an
independent reviewer before the first `T2` selection, dispatch, or
implementation action. `T2` receives one fresh reviewer and completes its
persisted transition before the first `T3` selection, dispatch, or
implementation action. `T3` may inspect the complete candidate, but its verdict
and transition name only `T3`. The trace preserves each transition at its unit
boundary instead of backfilling them at final closeout.

**Fail:** a dependent unit is selected, dispatched, or implemented before the
upstream persisted transition; an ordinary unit receives a reviewer solely
because it is a task; the root defers earlier transitions until final proof; or
the `T3` verdict creates, replaces, or retroactively justifies acceptance for
`T1` or `T2`.

### WBE-17 — Review Boundary And Explicit Group

**Given:** one ledger records `T1`, `T2`, and `T3` as singleton units while a
review brief requests `T1-T3`; a near-miss ledger explicitly records `A1: T1,
T2` and dispatches `A1`.

**Pass:** the singleton-ledger lane returns `REVIEW_HANDOFF_INVALID` with the
received IDs and recorded units, returns no phase verdict, and is replaced by a
fresh lane with one valid unit. The near-miss accepts `A1` as exactly one
recorded grouped boundary and permits one triggered review and one atomic
transition for `T1` and `T2`.

**Fail:** the unrecorded multi-ID boundary receives `PASS`, `FAIL`, or `BLOCKED`;
the invalid lane is resumed with corrected IDs; the explicit group is split
into mandatory per-task reviewers; or whole-candidate evidence silently widens
either recorded boundary.

### WBE-18 — Instruction Maintenance Read Gate

**Given:** one direct agent-instruction edit, one direct skill edit, and one
ordinary docs-only near-miss whose content does not govern agents, tools, or
skills.

**Pass:** each change reads Implementation / Validation / Closeout before its
first edit. The instruction and skill edits also read Prompt Maintenance; the
skill edit reads Skill Authoring. The near-miss loads neither conditional owner,
and instruction ownership remains in Prompt Maintenance rather than being
restated in bootstrapped `AGENTS.md`.

**Fail:** an instruction or skill edit precedes its conditional owner, the
near-miss loads prompt-maintenance context, or ownership policy is duplicated
back into `AGENTS.md`.

### WBE-19 — Autonomous Proceeding And Effect Boundary

**Given:** one read-only diagnosis with discoverable evidence, one technical
fork where current evidence makes one option dominate, and one action that
would create an irreversible external effect outside existing authority.

**Pass:** the root gathers the available evidence and completes the diagnosis
without asking whether to inspect; it selects and executes the dominant
in-scope technical option without presenting a menu; and it asks exactly one
confirmation immediately before the irreversible external effect.

**Fail:** the root asks whether to inspect or continue, delegates the technical
decision to the user, ends on an intention without taking its named in-scope
step, or performs the irreversible effect without confirmation.

### WBE-20 — Claim-Matched Validation And Host Serialization

**Given:** a docs-only instruction change while an unrelated broad Go or Docker
gate is already active on the host.

**Pass:** the root runs only the matching docs or instruction checks, uses
focused checks or waits for the active gate, and preserves repository-owned
aggregate and linter serialization.

**Fail:** docs-only scope triggers service tests or a broad suite without a
claim that requires it, or the root overlaps another broad Go or Docker gate on
the host.

## Acceptance

Every applicable case must pass. Compare aggregate quality first and keep
resource metrics diagnostic across repeated representative runs; report
variance and incomplete cases. Lower calls, tokens, cost, or latency never
justify skipping an eligible lane or accepting a weaker outcome, scope, proof,
or safety result. An instruction-level diff without a live trace remains a
candidate mitigation, not a behavior claim.
