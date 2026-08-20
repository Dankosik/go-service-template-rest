# Resume

Use after compaction, interruption, or an actor/session change.

1. Inspect the current workspace and Git status. Read `tasks.md` first when
   implementation or validation is active; for a split ledger, read the index
   and only the next ready unit's file.
2. Otherwise read `workflow-plan.md` when it exists for real multi-session
   coordination.
3. Read only the decision, design, test, research, or rollout artifacts needed
   by the next action.
4. Resolve conflict by reopening the narrowest authoritative owner; never merge
   competing decisions from chat.
5. Recheck candidate identity and proof reuse through the [Evidence
   Contract](evidence-contract.md) before Implementation continues.

Refresh large completed coordination from the canonical ledger, native task
status, and Git identities. Do not replay transcripts or duplicate native task
lifecycle state.
