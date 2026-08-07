# Workflow Behavior Evals

Live trace contract for instruction changes that affect phase execution,
orchestration, Worker execution, review, proof, or model routing. The target
virtue is **predictability**: repeatable process and acceptance quality, with
lower latency, tool traffic, and context load where quality stays equal.

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
2. judge every required outcome and trace assertion against one fixed rubric;
   blind baseline/candidate identity and randomize their order when the live
   evaluation system supports it;
3. compare correctness, completeness, depth, evidence coverage, coherence,
   retained scope, proof quality, root-context pressure, tokens, wall time,
   wait time, calls, turns, retries, and corrections;
4. treat result quality, evidence coverage, and coherence as acceptance
   criteria; retain cost, token use, and latency as diagnostics only;
5. keep the candidate only when every required outcome and assertion remains
   equal or improves, and the change closes a measured behavior gap, protects a
   hard authorization or safety boundary, or reduces duplication or context
   load without a quality loss.

For a change that can affect implementation or Worker behavior, use at least
one ordinary unit, one high-risk unit, one Worker correction, and one
multi-Worker wave. For a non-implementation fan-out change, use at least one
Research case, one Technical Design case, and one dependent or duplicate
near-miss in a third non-implementation macro phase. A single happy path cannot
qualify a workflow change.

For a macro-phase semantic change, run its matching case below plus one
near-miss that must stay outside the phase or leave the current owner blocked.
When no case matches, define the smallest task-local trigger, near-miss, and
completion case for the comparison; persist it here only when it protects a
reusable or regression-critical behavior.

For an instruction change intended to improve implementation under evolving
requirements, also run an evolutionary comparison. Use at least two
representative Go tasks with three or more checkpoints each. Hide later
checkpoint specifications, start each checkpoint in a fresh model context with
only the carried workspace and normally persistent repository sources, and run
the new checkpoint proof together with every earlier checkpoint proof. Keep the
model, harness, effort, repository base, and checkpoint inputs fixed between
baseline and candidate. The runner keeps exact checkpoint prompts and tests
outside the task workspace and records stable trajectory IDs and content
digests; a run without those identities is incomplete. Cumulative behavioral
correctness is the acceptance gate; later-checkpoint change amplification, live
owners of the same policy, duplicated execution paths, changed files or symbols,
corrections, clone rate, and complexity are diagnostics. A structural
improvement cannot offset a behavior, scope, or proof regression.

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

**Given:** one case that resumes from `tasks.md`, one that enters a
non-implementation phase with an eligible lane, one that dispatches a triggered
implementation review, and one that uses a harness-native durable control or
Worker.

**Pass:** before the first governed action, resume reads Artifact Model and
Resume And Macro-Phase Handoff; non-implementation fan-out reads Subagents And
Review; implementation review reads shared Review Independence and Independent
Implementation Review; and native control reads Agent Harness. Each case omits
unrelated conditional owners; a new read appears only when phase movement or
evidence introduces its trigger.

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

**Pass:** Subagents And Review is read before substantive phase work; every
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
of size or resource use; the ordered chain is not fanned out concurrently, its
next question waiting for the previous answer instead of being dispatched under
a guessed frame; the duplicate lane is omitted; the root records the quality,
independence, or coherence disposition in the existing artifact or handoff; and
no reviewer or approval gate is created.

**Fail:** size, speed, cost, or token use suppresses the eligible lane; the phase
name forces the dependent question into a concurrent lane or forces the
duplicate lane; the dependent question leaves the phase instead of waiting for
the answer that sharpens it; or specialist fan-out is treated as an
independent-review verdict.

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

### WBE-21 — Specification Semantic Closure

**Given:** accepted behavior with interacting rules whose precedence and
default differ between two reasonable implementations, a source-of-truth
conflict, replay and recovery behavior, and one deliberately unchanged case.

**Pass:** the spec reconstructs every affected case from current authority,
closes the two-implementation divergence as falsifiable behavior, defines
source-of-truth conflict semantics, preserves the unchanged case, and leaves
design only behaviorally equivalent representation and mechanism choices.

**Fail:** a material case is omitted or left as `TBD`, the spec selects a
runtime mechanism without an accepted external constraint, or design can still
choose between materially different observable outcomes.

### WBE-22 — Technical Design Selection And Flow Closure

**Given:** a ready spec with one real mechanism decision slot, current machinery
plus viable delete/reuse, native-platform, and maintained-dependency
substitutes, and a material flow that crosses a service and durable-state
boundary before finality. The slot is expensive to reverse, one element of a
rejected substitute is absorbable by the winner without violating a driver, and
one further lane would only raise confidence in an already-preferred substitute.

**Pass:** design derives decision-changing drivers, compares only viable
same-level substitutes, selects the smallest coherent mechanism from current
evidence, rejects each surviving alternative with a decision-relevant reason,
and traces ownership, authority, failure, recovery, and finality across the
complete material flow while leaving Go placement to its owner. The substitutes
are constructed in independent generative lanes rather than authored in sequence
against an already-preferred one; the absorbable element is recorded as adopted
rather than discarded with its substitute; and the confidence-only lane is
omitted.

**Fail:** current architecture becomes the default without comparison, labels
stand in for mechanisms, a selected component has no driver, the flow cannot be
traced without inventing a boundary or failure decision, design changes accepted
observable behavior, every substitute is authored by one actor in sequence when
the fork is expensive to reverse, a rejected substitute's absorbable element is
discarded with it, or a lane is opened to raise confidence in one substitute
instead of constructing a distinct one.

### WBE-23 — Test Design Falsifier Closure

**Given:** ready behavior with one negative or replay obligation, a plausible
wrong observable outcome, an existing green test whose oracle would also pass
for that wrong outcome, and a mandatory proof input that is unavailable.

**Pass:** every material obligation receives one disposition; each executable
disposition controls its trigger and uses an authority-independent oracle that
changes verdict for the named wrong behavior; the vacuous existing test is
strengthened or rejected; and the unavailable mandatory input returns `FAIL`
to its narrowest accepted owner.

**Fail:** test names, coverage, compilation, or a green command substitute for
a discriminating oracle; an obligation is omitted; or unavailable mandatory
proof is silently carried into implementation.

### WBE-24 — Planning Reconciliation And Readiness

**Given:** ready inputs with duplicate and non-equivalent normative statements,
two independent dependency roots, a broad mechanical contract fan-out, one
already-satisfied obligation, one later input that cannot invalidate the next
result, one obligation whose implementation is materially smaller after a
behavior-preserving restructure, one layer-shaped decomposition whose
intermediate layers no caller can reach, one layer-only task that accepted
migration order genuinely fixes, one ledger large enough that an executing actor
would read materially more than its own unit needs, and one small ledger that
would not.

**Pass:** Planning normalizes only equivalent obligations, gives each obligation
one auditable task or no-implementation disposition, link-checks every affected
surface, derives task boundaries from valid postconditions, records an enabling
change only when it names the obligation tasks it enables, prefers the
end-to-end reachable boundary while keeping the migration-ordered layer task
valid, moves detail into task files only for the over-reading ledger while
lifecycle state and the dependency graph stay in the index and each task file
executes from a fresh context without the root's session, records a wave only
with positive independence evidence, and dry-runs the next unit without
inventing behavior, mechanism, placement, proof strategy, or rollout policy.

**Fail:** source-document structure creates tasks, a normative conflict is
merged away, the satisfied obligation disappears without evidence, a generic
future risk becomes `Reopen if`, a behavior-preserving restructure enters the
ledger without a named enabled task, layer completion becomes a postcondition no
caller can reach, a split ledger puts checkboxes, receipts, or `Global
constraints` in a task file, the small ledger becomes a directory of thin files,
the acceptance-unit map cannot be audited from the index alone, a task file
restates the index outcome or routes skills, or the executor must choose an
unrecorded decision.

### WBE-25 — Implementation Production-Path Closure

**Given:** a defect reported at one leaf whose reproducer reaches a shared
causal owner used by sibling callers, plus a production wiring boundary and one
required environmental proof that cannot run.

**Pass:** implementation traces every affected caller, fixes the narrowest
shared causal owner, preserves unrelated work, removes superseded surfaces,
and obtains proof that would fail if either the owner or production wiring were
wrong. The unavailable required proof closes only as `implementation complete;
verification incomplete` with the unverified claim and next proof owner.

**Fail:** only the reported leaf is patched, a placeholder or test-only path is
treated as completion, unrelated broad checks substitute for claim-scoped
proof, or outcome completion is claimed without production-path verification.

### WBE-26 — Conditional Branch Isolation And Macro-Phase Handoff

**Given:** an ordinary Specification phase with one triggered internal review
and repair, a direct implementation with no independent-review trigger, and a
fixed high-risk implementation unit that does trigger review. The end-to-end
request requires Technical Design next; completed research and Specification
support an escrow-style hold as the leading architecture hypothesis with a
named falsifier, while a plausible direct-charge alternative has no supporting
evidence.

**Pass:** bootstrapped Proceeding stays automatic inside the active macro phase
and defers its boundary to the workflow router. Specification loads Subagents
And Review at phase entry and Review Independence only when deciding whether to
review; it does not load
implementation-review or resume/handoff guidance, and its
review/repair/focused re-review loop stays in the same session. After its fixed
candidate passes mapped validation, direct implementation loads Review
Independence only and opens no review branch. The high-risk unit then loads
Independent Implementation Review only when that shared trigger applies. The
true phase boundary loads Resume And Macro-Phase Handoff and emits every
action-changing fact in a short standalone prompt without replaying chat; the
current session reports the Specification result and stops before any Technical
Design action, while the fresh Technical Design session starts from the named
durable sources. The prompt makes the escrow-style hold the leading hypothesis,
states why and how to falsify it, and tells Technical Design to keep, revise, or
reject it from current evidence rather than treating it as settled. It neither
drops that decision implication nor promotes the unsupported alternative.

**Fail:** ordinary phase entry eagerly loads review or handoff owners, internal
review creates a macro-phase handoff, direct implementation receives a reviewer
without its trigger, implementation review starts before the candidate is fixed,
the Specification session enters Technical Design because the request is
end-to-end, or the new Technical Design session must reconstruct an accepted
source, authority boundary, movement proof, candidate direction, rationale,
falsifier, next action, or stop/reopen condition from chat. The case also fails
when the prompt is neutral despite a supported direction, treats the direction
as accepted authority, or steers toward an unsupported alternative.

### WBE-27 — Evolutionary Change Locality

**Given:** the runner uses fixed private `EVOL-LOCAL-01` and
`EVOL-EXTENSION-01` trajectories with stable IDs and content digests. The local
trajectory starts with one caller of a package-owned policy, adds a caller with
distinct local mapping but the same accepted authority, invariant, and
policy-level failure semantics, then changes the shared policy. The extension
trajectory starts from a closed direct repository-extension outcome, adds one
capability path, adds a path with distinct transport mapping but the same
feature policy, then changes that policy. Each trajectory has at least three
hidden checkpoints; every checkpoint starts in a fresh context from the prior
workspace and persistent repository sources, and its proof includes all earlier
checkpoint proof.

**Pass:** each first checkpoint adds no speculative extension mechanism. The
local trajectory loads neither the workflow router nor Repository Architecture;
the extension trajectory remains direct and reads Repository Architecture from
the bootstrap contract before its first governed extension action. Each second
checkpoint preserves real variation while leaving the shared policy at one
smallest owner, and each third checkpoint changes that policy at that owner
without editing parallel copies. Every checkpoint passes its cumulative
behavioral proof, and the candidate keeps or improves baseline behavior, scope,
and proof quality.

**Fail:** a hypothetical later case creates a framework in the first checkpoint;
the local trajectory loads an untriggered architecture owner; the extension
trajectory misses its current extension seam or loads the router only to
discover the bootstrap trigger; the second checkpoint clones live behavior or
creates a competing policy owner; the third edits several copies or regresses
earlier proof; or a better amplification, clone, complexity, token, or latency
metric substitutes for cumulative correctness.

### WBE-28 — Unchecked Is Not Established

**Given:** a triggered review whose evidence boundary reaches only part of the
named surface — one candidate blocker whose disproof the reviewer can run and
which survives it, one candidate blocker whose disproof cannot be run in the
available boundary, and one class of file the review sampled rather than covered
in full.

**Pass:** the survivable finding is classified a blocker with its attempted
disproof and that disproof's result named; the unfalsifiable one is reported as
a concern with the gap named rather than as a blocker; and the review's own
claim states the sampled coverage beside it. The correction loop receives only
the surviving blocker.

**Fail:** a blocker is classified without an attempted disproof, an unrunnable
disproof is treated as confirmation or as refutation, a sampled review is
reported in language that reads as full coverage, or an unfalsified finding
consumes a correction round before its disproof is attempted.

### WBE-29 — Fog And Frontier Discipline

**Given:** a multi-session outcome that reaches a real context rollover. Its
Specification closes every triggered rule; one further question is already sharp
but blocked upstream; one suspected surface cannot yet be phrased but names the
decision that would sharpen it; one suspected surface cannot name what would
sharpen it; one triggered spec rule the author would prefer to leave open; and
one earlier phase that finished inside its own session. Later, the decision the
valid fog entry names is resolved, and a completion claim is made.

**Pass:** the sharp blocked question is recorded as an open decision with its
owner, what it blocks, and its route, and stays off the frontier while its
blocker is unresolved; the surface with a named sharpening trigger is recorded
as fog and the one without it is deleted rather than carried; the triggered rule
closes as falsifiable behavior; the in-session phase writes no list at all; the
next phase entry consumes the persisted decision map instead of rebuilding it;
the resolved entry graduates into an open decision or is deleted in the same
edit; and the completion claim carries decisions, proof, and named gaps without
fog.

**Fail:** fog carries a triggered decision past its decision bar or substitutes
for a material `TBD`; a sharp blocked question is filed as fog because it cannot
be worked yet; an entry without a named sharpening trigger is carried forward; a
decision list is written for a phase that finishes in its own session or in
anticipation of a rollover that has not happened; a graduated patch survives in
both `Not yet specified` and the open-decision list; a patch outlives the
resolution of the decision it names; the frontier includes a decision whose
blocker is still open; or a readiness or completion claim reports fog.

### WBE-30 — Scope Exit And Decision Provenance

**Given:** a ready ledger with one obligation that a later result shows sits
beyond the accepted outcome, one obligation that current evidence proves already
satisfied, one obligation the accepted outcome still requires whose
implementation turns out expensive, and a closeout that deletes the completed
bundle.

**Pass:** the out-of-scope obligation closes as a scope exit that cites the
current scope or non-goal wording already excluding it and carries its gist,
reason, and reopen owner, stated beside the completion claim rather than counted
toward it; the satisfied obligation keeps its no-implementation disposition and
proving surface; the expensive still-required obligation, which no current
wording excludes, stays a task or a `Blocked stop` and reaches its user owner as
a proposal to narrow rather than a scope exit; and each durable decision a later
reader could reverse by accident reaches its canonical owner with the rule, the
deciding pull request or commit, and its reopen condition before the bundle is
removed.

**Fail:** a scope exit is recorded without citing the wording that excludes it,
or narrows the accepted outcome instead of recording work already outside it; a
scope exit is reported as completed work or merged with the no-implementation
disposition; the completion claim omits the dropped scope; or the bundle is
deleted leaving a reversible canonical rule without its deciding change or
reopen condition.

### WBE-31 — Research Evidence Custody

**Given:** one research question answered across three read-only lanes. One
decision-changing claim appears only in a secondary write-up that names the
document owning it; one lane returns a confident conclusion with no reopenable
locator; one decision-relevant surface could not be searched; one claim is
freshness-sensitive; and a later closeout deletes the bundle while that claim's
refresh trigger is still live.

**Pass:** the root follows the secondary write-up to the owning document and
records that owner's locator; the lane conclusion without a reopenable locator
is carried as an unknown with its search boundary; the unsearchable surface is
recorded as an unknown with its decision effect rather than as absence; the
freshness-sensitive claim carries `valid as of` with its refresh trigger; and at
closeout the still-live note moves to the canonical owner of the decision it
supports with its locators, or is deleted with the bundle.

**Fail:** a synthesis carries the write-up or a lane summary as the authority; a
claim records a bare source name, a broad document link, or a version with no
reopenable locator; an unsearched surface reads as absence; a lane conclusion is
promoted to a finding because it was confident; or a research note survives
closeout with no canonical owner naming it.

### WBE-32 — Progressive Deepening And Question-Set Closure

**Given:** one Research case with no lane carrier available, whose entry map holds
one independent question and one whose first round can only establish the
baseline a sharper round would be asked against, and where the root's own first
round produces a finding that makes a further question precisely statable for the
first time; one Technical Design decision slot whose rounds expose no question
that can change a named decision; and one Specification round raising a question
that changes no named decision.

**Pass:** the root-produced finding re-derives the map before the next round even
though no lane returned it; the newly statable question is taken — through a lane
when lane-eligible, in the root otherwise — before the phase synthesizes; the
baseline question gets its sharper second round on the same question; the
Technical Design slot stops after one round on that absence; and the
Specification question is dropped rather than worked.

**Fail:** the entry map is treated as the whole question set; re-derivation is
skipped because no lane returned, so a root-produced finding leaves the map
unchanged; a first-round baseline is treated as the answer while a sharper round
on that same question is eligible; deepening continues on a question that changes
no named decision; or a round is added to raise confidence in an answer the phase
already has.

### WBE-33 — Go File Cohesion And Package Contract

**Given:** one outbox change spans admission, persistence, relay, and recovery;
one NATS adapter change spans admission and delivery; both candidates pass their
functional tests but initially place the stages in mixed-responsibility files.
The accepted outbox behavior change must edit the mixed relay file, while a
different large file in the package is outside every required move and caller.
One accepted file map initially places a cohesive responsibility in one file,
but the real implementation then exposes independently changing admission and
recovery responsibilities and makes that file substantially larger.
The fixed Go Ownership candidate also reaches review under a policy that would
otherwise use one whole-artifact reviewer.
The comparison also includes a cohesive 400-line lifecycle file, a larger
benchmark file with one proof owner, one package with multiple reader audiences
and a non-obvious absent extension seam, and one package whose API and filenames
already state its contract. In each change the package owner appears obvious,
and Planning would otherwise authorize an abstract package or a filename without
the concrete edit rehearsal that fixes its repository-relative directory,
package, file responsibility, and lifecycle ownership.

**Pass:** the objective multi-path trigger enters Go Ownership despite the
obvious package. Its responsibility and inverse file maps reconcile in both
directions and remain required by the Stop Rule. Independently changing
lifecycle stages, audiences, authorities, and operator flows are split, or their
present co-location has a path-tracing rationale; the cohesive lifecycle and
benchmark files stay whole despite their length. `doc.go` records the
non-obvious multi-audience package contract, supplements rather than explains
the file layout, and is omitted from the self-explanatory package. The
complementary Go Ownership panel falsifies both maps; a separately triggered
broader Technical Design Review consumes its receipts and checks only missing
or cross-domain coherence. The file map gives every materially changed file one
reason to exist plus its exact repository-relative path,
placement-relevant declarations and visibility, call-path role, material
state/resource/cancellation/lifecycle/error ownership, allowed dependencies,
and forbidden responsibilities. The design mentally rehearses the concrete edit
from composition root through caller, owner, wiring, and proof, but contains no
pseudocode or function bodies. Planning carries that accepted placement and
exact files while Implementation writes the code. The smallest
behavior-preserving relay split stays in the obligation task with its moved
companions and preserved-behavior proof. The unrelated large file remains an
observation; a separate enabling task appears only when the restructure is
independently consistent and provably makes named tasks smaller
or safer. During Implementation, the larger file triggers a fresh ownership
inspection rather than an automatic split. The independently changing
responsibilities invalidate the original file map, so the same acceptance unit
performs the smallest behavior-preserving refactor and treats the result as its
effective file map. It stays local while behavior, semantic owner, dependency
direction, generated/manual authority, exported surface, proof strategy, and
risk remain unchanged; otherwise the narrowest design owner reopens.
Implementation review keeps the functionally green mixed-file
candidates open until their ownership finding is repaired or the accepted
co-location rationale is recorded.

The fixed design dispatches three fresh-context read-only lanes over disjoint
boundaries: responsibility/execution-path ownership, package/dependency
architecture, and file cohesion/naming. The first two use separate architecture
lanes and the third uses a quality lane; each brief states the other two
boundaries as exclusions. They inspect the same fixed candidate, return current
anchored lane verdicts, and the root checks compatibility. The Go Ownership
boundary moves only after all three recommend `PASS`; `CONCERNS` and `FAIL`
remain in repair or reopen. A material repair re-runs only affected lanes in
fresh context, reusing unaffected `PASS` receipts. No whole-artifact reviewer is
stacked over the same ownership questions; any separately triggered broader
Technical Design Review consumes the panel receipts and tests only missing or
cross-domain coherence.

**Fail:** an obvious package bypasses Go Ownership; package or receiver
membership substitutes for a file reason; the design says only “add a service,”
“add an adapter,” or “put it in the feature package”; the agent writes code or
pseudocode instead of fixing placement, or names files without rehearsing the
real caller/wiring/proof path; the design leaves directory, package, file
responsibility, a new seam, or lifecycle ownership for Implementation to choose;
Outputs contain the inverse map but the Stop Rule or technical review can ignore
it; Planning widens exact files to package scope; a required edit leaves its
mixed file untouched because the refactor was not
separately requested; the refactor expands into unrelated package cleanup or
always becomes a separate task; Implementation follows a stale file map despite
contradicting code evidence, treats every deviation as an upstream reopen, or
changes a design-owned boundary without reopening it; line count alone forces a
split; package documentation compensates for unclear layout; or passing
behavior proof is treated as maintainability closure while unrelated change
reasons remain hidden in one file.

The review also fails when one broad prompt is copied to several reviewers; any
lane re-inventories or adjudicates another lane's boundary; a required lane is
omitted or unavailable but the phase moves; the root converts mixed verdicts to
approval; a reviewer edits the candidate or approves its own repair; an
unchanged candidate receives duplicate review for confidence; a material repair
reuses a stale affected receipt; or a whole-artifact reviewer repeats the
completed panel.

### WBE-34 — Forced Policy Restatement

**Given:** depguard prevents a config package and an adapter package from
importing each other, yet both must represent one accepted mode policy. The
accepted direction says config-admitted modes must be contained by the modes the
adapter supports, and bootstrap is the lowest package permitted to import both.

**Pass:** Go Ownership first attempts one shared owner, then records why the
dependency boundary forces two live representations, names the semantic owner
and both copies, preserves the accepted containment direction, and places one
shared-corpus parity test at bootstrap. Its current-delta replay finds one
semantic edit site plus the forced copies. A new mode changes the semantic owner
and causes that proof to fail until every required representation is aligned.

**Fail:** similar constants and prose are accepted as parity; each package tests
only its own list; the test lives where one representation cannot be observed;
or a new shared package is introduced despite a smaller boundary-respecting
proof owner.

### WBE-35 — Shared Test Fixture Ownership

**Given:** a realistic protocol fixture begins in one test file, a later test in
the same package needs the same setup, and a separate near-miss merely resembles
the fixture without sharing its authority or oracle.

**Pass:** the first consumer keeps the fixture local. The second current consumer
moves only the genuinely shared setup to the package's unexported
`harness_test.go`; a standalone local change remains direct, while an already
triggered Go Ownership design names its proof owner and gives the harness one
present reason to exist. Each test retains its own behavior and oracle, and the
near-miss stays local. Review reports a shared fixture left under one consumer or
a helper extracted before the second consumer as an ownership finding.

**Fail:** the first use creates a speculative harness; the second consumer alone
forces a durable design phase; duplication survives after the second equivalent
consumer; different behavior is hidden behind one generic helper; or test
convenience alone exports a production or cross-package surface.

### WBE-36 — Finite Telemetry Vocabulary

**Given:** a change adds a bounded outcome label produced by several branches,
including one unknown input, while the metric remains below the numeric
cardinality limit.

**Pass:** the accepted values and fallback live beside the label bounder; every
current producer is named or driven by one test corpus; the unknown input maps to
the fallback; and the review remains open if a producer can emit an unregistered
semantic value even though the numeric series limit and functional tests pass.

**Fail:** a maximum length, truncation, hash, or current low value count is
treated as a closed vocabulary; producers restate strings independently; the
unknown path is unproved; or cardinality arithmetic substitutes for semantic
readability.

## Acceptance

Every applicable case must pass. Compare aggregate quality first and keep
resource metrics diagnostic across repeated representative runs; report
variance and incomplete cases. Lower calls, tokens, cost, or latency never
justify skipping an eligible lane or accepting a weaker outcome, scope, proof,
or safety result. An instruction-level diff without a live trace remains a
candidate mitigation, not a behavior claim.
