# Evidence Contract

Canonical proof semantics for correctness, readiness, and completion claims.

Evidence qualifies only when it is current, matches the claim's scope, and
would fail if the claimed behavior or required production wiring were absent or
wrong. Prefer one exercise of the observable path. Split proof qualifies only
when its owner and wiring checks together establish that path. Status, file or
symbol presence, unrelated checks, and implementation summaries do not qualify
alone.

Record the command or procedure, relevant environment and preconditions,
result, and gaps. Attach commit or tree identity only across a checkout or
integration boundary; the current bounded diff is enough for local work. Reuse
proof only while its claim, content, provenance, preconditions, and risk surface
remain unchanged. Do not rerun unchanged evidence as ceremony.

The bounded-change actor owns iterative focused checks. The acceptance owner
assigns every deterministic gate, validates any reused receipt, and runs it on
the integrated tree when identity or preconditions changed. A reviewer runs
only a missing or adversarial falsifier for its independent question.

Return [Evidence Result V1](../interfaces/evidence-result-v1.md) for each claim.
When required proof cannot run, stop as `implementation complete; verification
incomplete`; do not accept the unit or claim outcome completion or readiness.

Verification reports evidence and gaps. Repair, unknown-cause diagnosis, and
missing rollout policy return to their owning methods; rerun only evidence
invalidated by the resulting change.
