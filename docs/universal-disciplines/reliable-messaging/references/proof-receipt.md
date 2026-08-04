# Proof Receipt

Use this workflow to make a complex artifact complete without making its final presentation larger. Keep the receipt private unless the user asks for a ledger, matrix, or review evidence.

## 1. Capture obligations before drafting

Extract only:

- each explicitly requested artifact and count;
- each named failure mode, guarantee, provider fact, and authority boundary;
- an implicit retry, concurrency, ordering, or ambiguity path only when it can falsify the selected guarantee.

Split obligations that can pass or fail independently. Do not turn a narrow request into a lifecycle checklist.

## 2. Build the receipt

Use one private row per obligation:

| Obligation | Claim or decision | Durable evidence or provider fact | Ambiguity and recovery | Falsifier or returned artifact | Status |
| --- | --- | --- | --- | --- | --- |

Classify evidence as a documented provider fact, local invariant or decision, inference, or unknown. Set status to `covered`, `gap`, or `out of scope`; never treat an inference or unknown as a guarantee.

For returned code, the artifact cell names the exact replacement API and the test that calls it. Use only helpers shown in the answer or already present in the repository. Label incomplete fragments as pseudocode or unrun.

## 3. Draft from covered rows

Lead with the verdict and current artifact state. Include only material supported by a `covered` row or a decision-changing `gap`. Preserve explicit authority and artifact-count limits. Do not reproduce the receipt headings in the answer unless they improve the requested format.

## 4. Challenge the draft once

Before finalizing, check every in-scope row:

- removing the claimed commit, identity, fence, or provider behavior makes its falsifier fail;
- a lost response or repeated attempt has one durable disposition and recovery owner;
- strict ordering does not silently promise progress past a poison or delayed earlier item;
- replay or redrive is not substituted with ordinary redelivery when explicitly requested;
- code tests invoke the returned API, exercise the decision-changing condition, and assert durable state plus delivery progress;
- operations stay inside the authorized environment, identities, destination, velocity, and stop conditions.

If a row fails this check, repair the draft or mark the exact gap. Do not claim completion.

## 5. Remove unearned detail

Delete topology, worker mechanics, locks, capacity work, tests, and rollout prose that no obligation or selected guarantee requires. The receipt improves coverage; it does not earn more scope.
