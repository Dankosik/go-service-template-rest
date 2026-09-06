# Resume

Use after compaction, interruption, or an actor/session change.

For mid-task steering, reopen the earliest affected decision, invalidate only
its dependent work and proof, and continue unaffected work. A side question or
status request does not reset the task or require replaying completed phases.

1. Inspect the current workspace and Git status. Read `tasks.md` first when
   implementation or validation is active; for a split ledger, read the index
   and only the files of the current ready frontier.
2. Otherwise read `workflow-plan.md` when it exists for real multi-session
   coordination.
3. Read only the decision, design, research, or rollout inputs needed by the
   next action. Retire old Test Design phase state and its review waits. Reuse
   relevant expectations from an existing test-plan file as reference, but do
   not maintain, approve, or recreate it as a required input. The executor owns
   test choices while implementing; preserve actual product requirements.
4. Resolve conflict by reopening the narrowest authoritative owner; never merge
   competing decisions from chat.
5. Reconcile current execution policy in the active packet: remove superseded
   per-task check/review waits and repeated CPU-permit requests, move their required
   claims to Final validation, and keep genuine acceptance/effect gates. Replace
   old Accept when / Focused check / Integrated check sections with the current
   packet shape; move verification-only rows into Completion. Preserve all
   required claims without preserving their obsolete execution order. This
   timing correction does not reopen accepted pre-Implementation decisions or
   require another Planning review. Preserve explicit user-owned constraints;
   a pinned older workflow cannot override current user instructions. Normalize
   older completed task receipts to Implemented while retaining actual evidence
   for possible final reuse. Old Accepted task receipts do not enable a second
   execution mode or substitute for final Completion.
6. Resolve current files and writer identities before consuming Implemented
   outputs. Evaluate proof reuse only when final validation begins under the
   [Evidence Contract](evidence-contract.md). Resume the actual execution stage:
   coding uses Implementation's feedback boundary; final-validation repair
   includes its focused rerun under existing authority. Retire a superseded
   code-only repair brief without restarting the ledger or its review cycle.

Resolve the ledger's Execution locators against native task state before
resuming or redispatching; a stored locator alone does not prove work is active.
Use Resume notes only after checking them against current workspace and evidence.
Refresh large completed coordination from the canonical ledger, native task
status, and Git identities. Do not replay transcripts or duplicate native task
lifecycle state.
