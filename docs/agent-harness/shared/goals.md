# Goal Mechanics

Read before choosing a durable Goal control.

Use a Goal only for a genuinely long-running, multi-step, or resumable
Implementation outcome. The Codex Ledger Orchestrator is the routing-only
exception. One Goal owns one thread-local stage and names its observable end
state, proof surface, preserved constraints, and blocked stop condition. Never
use a turn or iteration count as completion.

The Goal directive starts work; do not send a second prompt that restates it.
An Acceptance-Unit Lead Goal ends only at its canonical receipt or blocker. A
Ledger Orchestrator Goal stays active through authorized agent-owned recovery.
The selected adapter owns evaluator visibility and exact controls.
