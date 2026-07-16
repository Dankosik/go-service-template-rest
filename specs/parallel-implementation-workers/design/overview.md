# Parallel implementation Worker design

status: ready

## Baseline replaced by this design

- `docs/spec-first-workflow.md` routed implementation with one ready ledger task at a time.
- `docs/spec-first-workflow/phases/planning.md` already recorded true dependencies and one independently reviewable outcome; this change makes writable ownership, exclusive resources, and exact handoffs structural while `task-review-readiness.md` rejects artificial dependency chains and hidden overlaps.
- `docs/spec-first-workflow/phases/implementation-validation-closeout.md` assigned one task per native App Worker in a dedicated managed worktree, but permitted only one active write Worker and required acceptance before dispatching the next task.
- The existing ledger, Worker brief, transient execution record, root acceptance, same-task correction, replacement, and dirty-Local recovery mechanisms are sufficient building blocks. No scheduler service, new dependency, or second state artifact exists or is needed.

## Selected mechanism

### Planned waves and lightweight dispatch

Planning places every task in exactly one earliest-safe planned wave. `Depends on` remains the execution/proof authority; the wave section precomputes the recommended concurrency schedule and records a short positive independence basis for each multi-task wave. Direct work remains one sequential outcome.

At implementation entry and after every accepted wave, the root reads the next planned wave and performs one lightweight check only on facts that may have changed: dependencies are accepted, every member starts from the same authoritative integration base, ownership and exclusive-resource boundaries remain disjoint, and no new canonical/generated, interface, migration, rollout, or proof coupling invalidates the recorded basis. `Depends on: none` alone never establishes independence.

The root clips the wave to current native App task capacity and narrows or serializes only members whose current check fails or remains uncertain. It records the adjusted wave in transient execution context or the existing ledger and continues unaffected work. Capacity, completion order, drift, or an implementation overlap changes the schedule locally; it does not by itself reopen planning or create a second scheduling artifact.

The root dispatches every task in the selected wave before waiting for a wave result. Each task gets one native App task, one managed worktree based on the same authoritative integration state, one outcome-first brief, and its own explicit model/effort selection. The root records the task-to-App-task-to-worktree mapping in transient execution context or the existing ledger evidence. No parallel status field or scheduler artifact is added.

### Execution and fan-in state

There are four transient states:

1. **Authoritative integration base**: the accepted repository state from which the wave starts.
2. **Worker candidates**: isolated task deltas in separate managed worktrees; none is accepted or authoritative.
3. **Disposable wave candidate**: a clean integration worktree or task-owned integration branch where the root mechanically applies provisionally suitable bounded Worker deltas in controlled order. It may contain unaccepted deltas and is never ledger completion.
4. **Authoritative integrated state**: the wave candidate only after every wave task passes root review and its declared proof on the combined candidate; the bounded wave delta is then promoted and every task's evidence is recorded.

The root may review returned results as they arrive and return ordinary task-local findings to the same owning App task immediately. It does not mark any wave task accepted while another wave member is unresolved. Before return, each Worker rereads its outcome and constraints, inspects its bounded diff and cleanup, runs focused proof, and reports gaps; this remains evidence rather than acceptance. Once every locally suitable result is present, the root freezes the disposable wave candidate, maps claims to exact commands, and runs each identical command once. One result may satisfy multiple explicit claim mappings, but commands with different arguments, environment, state preconditions, or observables remain distinct.

### Failure and recovery

- A Worker-local defect returns to the same task and worktree under the existing correction/replacement rule.
- Repeated material findings from one causal class stop symptom-level patch requests. The root names the violated invariant and owning surface, groups the evidence, and requires a materially different route; the same Worker continues unless the evidence-backed replacement rule also applies.
- A mechanical apply conflict, semantic overlap, invalidated assumption, or combined-proof failure holds the whole wave. No member is accepted or promoted.
- The root preserves unaffected results as provisional, supplies the current disposable wave candidate and concrete fan-in finding to the owning Worker, and has that Worker repair only its task. The root may mechanically update the task worktree with the current wave base; it does not author the repair.
- If the conflict only disproves the planned concurrency grouping, the root adjusts the wave and recovery order locally, then reassembles and re-proves it. Reopen the smallest planning, design, specification, or evidence owner only when recovery requires a genuinely missing or changed behavior, ownership, proof, or rollout decision.
- If App capacity is one, concurrency cannot be observed reliably, or any independence condition is uncertain, the existing sequential Worker loop is the complete fallback.

### Convergence-first correction

Root and Worker optimize jointly for the fastest safe path from bounded candidate to proven acceptance. The root's first acceptance inspection covers the full task claim and affected lenses, then batches every currently detectable compatible finding into the existing outcome-first brief with causal owner, evidence, required end state, affected scope, and proof. There is no correction artifact or acknowledgement turn.

The Worker treats a suggested patch as a hypothesis unless accepted design fixes the method. It validates the finding, may return counter-evidence, repairs the violated contract at the narrowest owner, checks sibling and transitive effects, and returns a candidate intended for acceptance. The root re-reviews the complete correction set, correction delta, affected proof, and invalidated lenses only; once they pass, it accepts without another review lane. A later finding is classified as a prior omission, correction regression, repeated causal class, genuinely new surface, or upstream decision gap. Each turn must shrink the evidence frontier or materially change the causal model or route; no fixed retry limit or unchanged loop is added.

### Completion

After atomic wave acceptance, the root advances to the next planned or adjusted wave and rechecks only seams affected by integrated changes. One root Goal spans all waves. Final integrated-diff review and terminal validation still cover every accepted obligation and task; a wave's green focused proof is not repository completion.

### Resume state

Transient execution state remains sufficient during uninterrupted work. Before compaction, interruption, or session handoff can lose an active wave, the root writes one compact `Active wave` block into `tasks.md`: adjusted members, accepted integration base, task-to-App-task/worktree state, disposable candidate identity when present, and the next root action or open causal class. It updates the block only at material transitions and removes or collapses it into task evidence after atomic acceptance. No scheduler or progress file is added.

## Planning and task shape

Use one compact task contract plus one top-level `Planned waves` section. `Global constraints` owns only exact constraints shared by multiple tasks; `Depends on` owns real start/complete/proof gates; `Owner/surface/resources` owns the writable boundary plus mutable, exclusive, or non-concurrent resources; optional `Handoff` owns an exact consumed/produced cross-task contract; task outcome and `Proof` own reviewability. Planning adds only these rules:

- every task appears exactly once in its earliest safe wave, with a short positive independence basis for each multi-task wave;
- task boundaries should expose genuinely separable outcomes but must not be made smaller solely for concurrency;
- coupled source/generated/test/doc changes stay in one task when splitting would create an invalid intermediate state;
- exact ownership, canonical/generated relationships, shared mutable resources, and cross-task interfaces must be explicit enough that implementation can prove or reject pairwise independence;
- an oversized-task preflight splits separable ownership, review, failure/recovery, rollback, or proof domains only when each can end in a valid provable state; file count, estimated minutes, and desired Worker count are not sizing rules;
- uncertain independence produces a one-task wave.

Per-task `Parallel with`, lane, status, or conflict-list fields are rejected. They duplicate the reviewed top-level schedule, drift when tasks change, and create parallel state fields the artifact model avoids. Shared database, port, environment, migration target, destructive fixture, lock, generated-pair, and proof-resource conflicts belong in `Owner/surface/resources`, not prose or a second conflict list.

## Instruction ownership

Canonical changes belong in:

- `docs/spec-first-workflow.md`: replace the one-ready-task-at-a-time spine with one direct outcome or one safe ready frontier/wave at a time.
- `docs/spec-first-workflow/phases/planning.md`: preserve outcome-sized tasks while recording earliest-safe waves and enough dependency/ownership evidence to justify them without per-task parallel fields.
- `docs/spec-first-workflow/phases/task-review-readiness.md`: falsify false independence, hidden shared write/proof state, and micro-tasking for fan-out.
- `docs/spec-first-workflow/phases/implementation-validation-closeout.md`: own safe-wave dispatch, isolated App tasks, disposable fan-in, atomic wave acceptance, correction, fallback, and closeout.
- `docs/spec-first-workflow-evals.md`: revise sequential cases so only dependent or not-positively-independent tasks are forbidden to overlap; add positive safe-wave and false-independence/fan-in cases.
- `scripts/ci/workflow-instructions-check.sh` and `scripts/dev/workflow-behavior-evals.sh`: enforce the new canonical tokens and eval manifest/counts.
- `Makefile` and `docs/build-test-and-development-commands.md`: update only the eval count exposed by existing help/docs.

`AGENTS.md` already delegates Worker execution, acceptance, and integration mechanics to the implementation phase and needs no duplicate scheduling policy. Canonical skills already delegate to the planning and implementation phase files, so they need no semantic rewrite or new skill; mirror checks still verify there is no drift.

## Alternatives rejected

- **Copy `superpowers` task steps and 2-5 minute granularity**: increases dispatch/review overhead and conflicts with the repository's independently reviewable outcome boundary.
- **Copy `superpowers`' ban on parallel implementation**: that ban addresses agents sharing one branch; native App managed worktrees remove that specific interference while retained fan-in checks cover remaining risk.
- **Parallel Workers in one checkout or branch**: violates the isolation invariant and recreates the conflict mode the reference warns about.
- **Persistent scheduler, queue, or concurrency configuration**: no current consumer needs it; native App capacity and the existing ledger's planned waves are sufficient.
- **Partial acceptance of a failed wave**: complicates authority and recovery. Hold the wave and preserve provisional results; revisit only if representative evals show this bounded all-or-nothing rule is the throughput bottleneck.

## Proof and reopen conditions

- Structural workflow proof must cover safe parallel dispatch, sequential fallback for hidden coupling, disposable fan-in, atomic acceptance, same-task repair, one Goal, dirty-Local preservation, and terminal combined validation.
- Repository proof uses the existing workflow instruction, behavior-manifest, routing, skill/mirror, agent, and diff checks. Live behavior requires authorized eval adapters and is not implied by structural checks.
- Reopen planning ownership if the compact planned waves and existing task fields repeatedly fail to expose material conflicts or useful concurrency in representative evals.
- Reopen this design if native App tasks cannot share one authoritative starting state, the root cannot materialize bounded Worker deltas in a disposable integration worktree, or whole-wave recovery creates measured unacceptable delay.
