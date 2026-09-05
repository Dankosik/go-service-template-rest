# Review Result V1

Every independent review returns:

```text
candidate: <fixed artifact revision, diff, tree, or commit>
verdict: <phase adapter value>
findings: <anchored surviving findings or none>
evidence_boundary: <what was independently checked>
reopen_owner: <owner or none>
```

Each surviving finding names its anchor, outcome impact, classification,
smallest repair/reopen owner, and evidence boundary. Record the attempted
falsifier and result, or why it is unavailable. The phase adapter determines
whether the gap blocks the result; missing mandatory proof is not downgraded
to a concern.

Implementation finding classification is closed:

| Kind | Meaning | Routing |
| --- | --- | --- |
| `TASK_DEFECT` | The candidate falsifies the current task packet. | `FAIL`; repair the same task. |
| `UPSTREAM_GAP` | An accepted input or dependency output is missing or invalid. | `NEEDS_PARENT`; reopen the smallest upstream owner. |
| `INTEGRATION_DEFECT` | A seam or assembly between accepted outputs fails. | `FAIL`; route repair under [Integrated Candidate](../phases/implementation-review.md#integrated-candidate) to the smallest affected existing unit, preserving unaffected acceptance. Reopen Planning only when its accepted boundary cannot cover the repair. |
| `FOLLOW_UP` | An improvement is outside the current packet and does not falsify it. | It cannot block `PASS`; route separately through Planning if accepted. |

Non-implementation adapters use `PASS | CONCERNS | FAIL`. Implementation uses
`PASS | FAIL | NEEDS_PARENT`; `NEEDS_PARENT` also names the unverified claim and
unavailable proof/action owner. An unresolved implementation boundary, or a
review that still must accept more than one unit, returns
`REVIEW_HANDOFF_INVALID` without a verdict and names the acceptance owner that
must repair the handoff.
