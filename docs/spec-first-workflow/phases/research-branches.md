# Research Branches

Load only the branch selected by [Research](research.md).

## Current-State Or Semantic Baseline

Inspect only surfaces that can independently change the decision: repository
and generated authority, runtime behavior, callers/operators, persisted or
deployed state, external consumers, and mixed versions. Trace affected values
only as far as required, preserving grain, identifiers, units, absence/error
states, version authority, and lossy transformations. Matching names or
historical intent do not prove semantic equivalence or runtime use. Record an
unobservable required surface as an unknown with its decision effect.

## Current External Contract

Verify each decision-changing external claim against current primary authority:
official provider documentation or contract, the applicable standard, or
maintainer source. Resolve version and applicability. Add implementation or
operational evidence only when local fit or failure behavior remains uncertain.
For Go, toolchain, stdlib, runtime, or dependency behavior, establish the
repository version and inspect only the authority needed for that decision.

## Solution Discovery Evidence

Derive vendor-neutral terms from behavior, invariants, workload, failure and
recovery, constraints, and unacceptable trade-offs. Classify each candidate as
a substitute, prerequisite, complement, or defense-in-depth mechanism; compare
only same-level substitutes. Research may eliminate a candidate contradicted by
authority or a hard constraint, but System Design selects the mechanism.

Scan only relevant rungs: repository reuse, Go stdlib, native platform,
approved infrastructure, managed service, mature OSS, and custom code. For a
surviving external option record only evidence that can change approval:
ownership/support, availability/quota/SLA, compatibility, security and data
custody, pricing unit at accepted workload, lifecycle, migration, portability,
and exit/failback. For external code also cover maintenance, license,
vulnerabilities, API stability, and transitive cost. Stop when searches by
problem, force, failure mode, and known alias no longer expose a materially
different viable substitute.

## Empirical Claim Or Probe

Record version/configuration, command or method, environment, representative
workload or sample, timestamp, result, and limits. Performance, capacity,
reliability, cost, quality, or version claims also need a comparable baseline.
When authority cannot resolve applicability and an authorized safe surface
exists, run the smallest discriminating probe; otherwise carry the exact proof
gap.

## Conflict Or Freshness

For material conflict or a freshness-sensitive hard-to-reverse choice, obtain
an independent semantic challenge before downstream use. Unresolved authority
conflict blocks or returns to its evidence owner. Record `valid as of` plus an
objective refresh trigger.

## Downstream Input Closure

Record the input owner, authoritative source, required shape, availability, and
earliest checkpoint. For a proof implication, name the current observable and
setup availability or the missing-proof owner; Test Design still selects the
scenario and proof level.
