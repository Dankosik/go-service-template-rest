# Evidence Contract

Canonical proof semantics for correctness, readiness, and completion claims.

Evidence qualifies only when it is current, matches the claim's scope, and
would fail if the claimed behavior or required production wiring were absent or
wrong. Prefer one exercise of the observable path. Split proof qualifies only
when its owner and wiring checks together establish that path. Status, file or
symbol presence, unrelated checks, and implementation summaries do not qualify
alone.

Attach commit or tree identity only across a checkout or integration
boundary; the current bounded diff is enough for local work. Reuse a receipt
only while `candidate`, `scope`, `command`, and `environment` are unchanged and
the result is `pass`. Do not rerun unchanged evidence as ceremony.

Before execution, choose one minimal proof plan across all claims; one receipt
may support several. Do not run a leaf included in a selected aggregate or add
an aggregate whose additional surfaces are irrelevant. The
normal ladder is an optional focused falsifier, then one surface-aware
`make verify`; `make prove` is optional iteration, and a full-repository gate
is never automatic. `make plan` explains the route; it is not a gate.

Reuse existing proof when it would fail on the changed observable. Add a test or
fixture only for otherwise-unproved changed behavior; unrelated historical
coverage remains outside the unit.

The bounded-change actor owns iterative focused checks. The acceptance owner
assigns every deterministic gate, validates any reused receipt, and runs
`make verify` once on the integrated candidate. Rerun `make prove` only when
package-sized iteration is still needed and the candidate will change before
`make verify`. A reviewer runs only a missing or adversarial falsifier for its
independent question. Run `ALLOW_FULL=1 make check` only when the integrated claim spans its
full-repository evidence boundary. Heavy targets require `ALLOW_HEAVY=1` or CI.

Return [Evidence Result V1](../interfaces/evidence-result-v1.md) for each claim.
When required proof cannot run, stop as `implementation complete; verification
incomplete`; do not accept the unit or claim outcome completion or readiness.

Verification reports evidence and gaps. Repair, unknown-cause diagnosis, and
missing rollout policy return to their owning methods; rerun only evidence
invalidated by the resulting change.
