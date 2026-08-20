# Read-Only Lanes

Use built-in subagents only when they reduce a real context or coordination
problem.

## Read When

- Dispatching a non-implementation read-only lane.

## Delegation Decision

Skills define method; subagents provide separate context or independence. A
question is lane-eligible only when all are true:

- it is one concrete decision or falsification target with a checkable evidence
  boundary and read-only external-action limits;
- its answer can change a named criterion or downstream disposition;
- it is independent of mutable work and dependent reasoning in other lanes;
- separate context, specialist evidence, or required review independence can
  materially improve the answer; and
- the root can check and synthesize the result.

Keep ordered reasoning, user intent, authorization, synthesis, correction,
integration, acceptance, and completion in the root. Run one read-only lane per
eligible question; independent lanes may run concurrently within current
carrier capacity. A second lane over the same question and evidence is duplicate
confidence, not coverage.

A lane exists only after a native subagent control returns its identity. Before
reporting dispatch or synthesizing lane results, retain that identity; separate
root searches are not lanes. Call no wait control until at least one returned
identity exists; a wait with no receiver identity is a capability failure, not
progress. When the user explicitly requests lanes but no callable native
carrier returns an identity, state that capability gap before continuing
root-locally, and continue only when the requested outcome remains honest and
useful.

Evidence may sharpen the next question; dispatch again only when the revised
question independently passes the same test. Otherwise continue root-locally
without a gate record. A lane returns a newly exposed owner decision rather than
expanding scope. The harness-neutral lane maps through the [Read-Only Lane
Carrier](../../agent-harness.md#read-only-lane-carrier); Implementation Workers
use their separate [write boundary](../phases/implementation-worker-execution.md#implementation-write-boundary).

One open decision slot may instead receive several **generative** lanes, each
constructing a materially distinct candidate without seeing the others. These
are not duplicate lanes: they differ in what they produce rather than in the
question they answer, and their value is precisely that no candidate is authored
against an already-preferred one. Use them only where a real fork is still open
and reversing the choice later is expensive. A second lane asked the same
question against the same evidence boundary, to raise confidence in one answer,
remains a duplicate lane. The root still owns comparison, selection, and the
rejected-candidate record.

When a question map must survive the phase, persist only its current [open
decisions and frontier](resume-and-handoff.md#open-decisions-and-fog). Model and
effort follow the [harness map](../../agent-harness.md#model-and-effort-selection).

## Lane Brief

One lane owns one question, one evidence boundary, and one consuming
disposition. Keep the brief outcome-first:

```text
Question: <one decision or falsification target>
Context: <accepted facts and minimal artifact paths>
Evidence boundary: <what to inspect and what counts>
Constraints: <read-only boundary, non-goals, external-action limits>
Output: <finding/evidence/recommendation shape>
Stop: <missing input, conflict, or completion condition>
```

Do not copy the repository workflow, generic strictness language, or unrelated artifact summaries into every brief.

## Lane Result V1

Every material lane result returns this interface:

- conclusion and strongest evidence;
- uncertainty or conflict;
- consuming disposition: accept, reject, repair, carry as proof/risk, or reopen;
- destination artifact or owner.

A lane's summary is a secondary source: carry the locator it landed on, not its
restatement. A conclusion returned without one is an unknown, not a finding.

Do not paste raw transcripts into authoritative artifacts.
