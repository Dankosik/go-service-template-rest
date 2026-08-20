# Domain Test Traceability

## Decide
- Missing proof is a finding only when a changed production or test line lets a named business regression pass unnoticed. Name that regression concretely; it is the whole finding.
- The strongest domain proof is usually a **negative** assertion — that the rejected path performed no effect, that the replay performed one. A test asserting only the returned error leaves the effect free to move.
- Anchor on the changed production line when the defect is missing proof, and on the changed test line when the defect is weakened proof — a renamed test, a dropped assertion, or a fake that no longer records the call.
- Ask for the smallest test that fails today on the regression and passes after the fix.
- Broad test strategy, fixture design, assertion style, and flake depth go to `go-test-strategy`; stay on the business rule that lacks proof.
- Behavior that did not change and is not adjacent to the diff's domain risk is out of scope, however thin its coverage.

## Reject
```text
Add more tests for edge cases.
```
Failure: no invariant, no failure mode, no business impact. Nothing tells the author which test would have caught it.

```text
Coverage on this package dropped.
```
Failure: a coverage delta is not a domain finding; it names no rule that a regression could break.

## Prove
One targeted proof is enough: the forbidden transition rejects, the rejected command triggers no effect, the duplicate is a no-op or a rejection by contract, stale input does not overwrite newer state, or the renamed test still asserts the accepted rule.
