# Intake

Use when new structured work lacks a complete
[Intent](../interfaces/intent.md), or outcome, meaningful scope, authority,
observable success, or the first owner is unresolved.

Own only requester intent and the brief needed for routing. Reconstruct what
the requester established and what they delegated. Inspect bounded repository
facts. Technical choices and routing belong to the agent system. Route
evidence gaps that could change a decision to Research.

Check whether a plausible reading would fulfill the request as worded while
leaving the affected person's problem unresolved. Weigh that reading against
the requester's stated goal, prior decisions, and later corrections.

Before asking for clarification, identify which Intent section or authority
boundary depends on the answer. Ask only for an unresolved user-owned decision
or external input under [Decision Ownership](../../../AGENTS.md#decision-ownership).
Frame the question in the requester's terms around the missing fact, observable
outcome, or consequence of a tradeoff. Keep it open to an answer outside the
interpretations already considered. Batch independent questions and follow up
when one answer determines the next.

After a material clarification, summarize its consequence in one sentence the
requester can correct and carry it into the affected section. Record a success
threshold only when the request or an applicable constraint establishes it.
Label any material assumption where it applies and state what would reopen it.

When every material ambiguity is resolved by the request, evidence, or a
labeled assumption that leaves no requester meaning to invent, synthesize
`specs/<task>/intent.md` through [Artifacts](../shared/artifacts.md). The file is
the interview summary, not its transcript. If no question survives the initial
reconstruction, create it directly; if an unresolved user-owned question can
still change a required Intent section, return that blocker and do not create
the file.

Return the intent locator, first owner, and only the assumptions or open
questions that can change routing. Ready means the next owner can act without
inventing requester intent or authority. Reopen when later evidence changes a
material Intent field.
