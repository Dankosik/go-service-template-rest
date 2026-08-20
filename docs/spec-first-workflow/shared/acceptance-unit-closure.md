# Acceptance-Unit Closure

Read after one fixed implementation candidate satisfies its accepted criteria,
mapped proof, and any triggered independent review.

For ledger work, keep the recorded singleton or grouped acceptance-unit
boundary fixed through acceptance and the immediate persisted transition owned
by the [Planning Ledger
Contract](../phases/planning/ledger-contract.md#implementation-transitions).
That transition is the unit's completion criterion. Only after it lands may
task selection re-evaluate `Depends on` and start a dependent unit or new
planned wave; members already running inside the same accepted wave remain
provisional and may continue.

When the unit contains the final unchecked task, verify the ledger's global
`Completion` condition and every required task disposition before setting its
status to `done`.

When the transition cannot occur, the unit remains selected for phase-owned
correction, reopen, or blocker disposition. When later implementation work
remains across a session boundary, apply [Implementation
continuation](../phases/implementation.md#acceptance-and-continuation) after the
transition so the receiving session selects the next ready unit from current
evidence.

A fixed inline unit keeps the same boundary through implementation, proof,
review, and closeout and creates no synthetic ledger transition.
