# Cross-Service Fault Localization

## Behavior Change Thesis
When loaded for a symptom that crosses a boundary, this file makes the model localize the defect across the whole failing path from evidence on both sides of every hop, instead of assuming the repository that holds the stack trace holds the defect.

## When To Load
Load when the failing path leaves this process: a caller-supplied value, a rejected or malformed request, a 4xx/5xx from or to a neighbor, a missing or duplicated effect, a frontend-visible failure, or a symptom that appeared without a local change. Also load when the local repair would only be a guard against a value this repository does not own.

## Decision Rubric
- Reconstruct the path before reading code: every service, client, job, broker, and managed dependency between the trigger and the observed symptom, with its owner. Take the inventory from `docs/repo-architecture.md` System Neighbors; when a neighbor is missing there, record it after the diagnosis instead of guessing its role.
- Pull the same unit of work out of every hop by correlation identity — request id, trace id, message id, or business key — rather than comparing unrelated log windows. A hop where that identity is absent is itself evidence: the work never arrived, or the identity is not propagated.
- Read each neighbor's canonical contract from its own repository, generated contract, published spec, or live contract endpoint. Its client code, a vendored copy, and this repository's expectations are not its contract.
- Classify each hop against its contract as producer defect, local defect, consumer defect, contract disagreement, or transport/infrastructure between hops. Only a classified hop is cleared; an uninspected hop stays an open hypothesis.
- Separate one cause with several downstream symptoms from several independent defects. A cascade is proven by the correlation identity and ordering, not by plausibility.
- Fix at the earliest owner of the violated invariant. When that owner is another repository, return an external blocker with the owner and the evidence rather than compensating locally; add a local guard only when this repository owns the contract it fails to enforce.
- Carry the minimum sanitized field that supports the claim; secrets, tokens, credentials, and customer data never leave a neighbor's logs.

## Imitate

```text
Symptom: client shows "payment pending" indefinitely; this service logs 502 from billing.
Path: web client -> gateway -> this service -> billing-service -> ledger worker.
Correlation: X-Request-ID 8f3c... found in gateway, this service, billing; absent in ledger.
Evidence per hop (both sides):
  gateway   emitted amount_cents=1250            (gateway logs)
  this svc  forwarded amount_cents=1250, got 500 (local logs)
  billing   500 on nil *Money after decoding amount_cents as float64
                                                (billing repo source + billing logs)
  ledger    no record for this correlation id    (never reached)
First broken invariant: the shared contract declares amount_cents as integer;
billing's generated client decodes it as float64 since its regeneration on rev abc123.
Owner: billing-service repository, contract regeneration.
This repository: no defect. Mapping an upstream 500 to 502 is its accepted behavior.
Disposition: external blocker with named owner; a local change would only hide the cascade.
```

Copy the shape: evidence on both sides of every hop, one named first broken invariant, and an explicit verdict that this repository is not the owner.

## Reject

```go
if resp.StatusCode >= 500 {
	return nil, fmt.Errorf("upstream unavailable")
}
```

Collapsing a neighbor's contract violation into a generic local error destroys the evidence that localizes the defect and makes this repository look like the owner.

```text
Root cause: billing-service returns 500. Fix: retry here.
```

A neighbor's status code is a hop observation, not a root cause. Retrying a deterministic contract mismatch converts one failure into an amplified one.

## Agent Traps
- Treating the repository that holds the stack trace as the repository that holds the defect, because it is the only one with a local reproducer.
- Stopping at this service's ingress because the bad value came from outside it.

## Validation Shape
Record the reconstructed path with owners, the correlation identity used, the evidence read on each side of every hop with its source, the hop where the invariant first broke, the repository that owns it, the rejected hop hypotheses, and either the replayed proof or the named external owner, blocker, and earliest checkpoint.
