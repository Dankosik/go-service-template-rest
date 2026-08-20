# Generative Candidate Lanes

Read only when one expensive-to-reverse decision still has a real open fork and
independent candidate construction can change the selection.

One decision slot may receive several generative lanes, each constructing a
materially distinct candidate without seeing the others. They are not duplicate
lanes because they differ in what they produce, not merely in confidence over
one answer. A second lane asked the same question against the same evidence
boundary remains duplicate confidence and should not run.

The root owns comparison, selection, and the rejected-candidate record. Each
lane still uses the shared brief and result interfaces, stays read-only, and
returns no acceptance or completion claim.
