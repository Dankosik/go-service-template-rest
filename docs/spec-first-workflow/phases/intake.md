# Intake

Use when new structured work lacks a complete
[Intent](../interfaces/intent.md), or outcome, meaningful scope, authority,
observable success, or the first owner is unresolved.

Take responsibility for framing the request. Formulate the problem, desired
outcome, and observable success signal from the requester's words, prior
decisions, and relevant facts. Identify what the requester established and
what they delegated. Technical choices and routing belong to the agent system.
Route evidence gaps that could change a decision to Research.

Check whether a plausible reading would fulfill the request as worded while
leaving the affected person's problem unresolved. Weigh that reading against
the requester's stated goal, prior decisions, and later corrections.

Before asking for clarification, resolve the parts of the brief that fall
within the agent's responsibility. Ask only when an Intent section or the
authority to act still depends on an unresolved requester-owned choice, a
fact unavailable from accessible sources, or additional permission under
[Decision Ownership](../../../AGENTS.md#decision-ownership). Explain the
consequence in the requester's terms. When the requester must choose and the
evidence supports a recommendation, give it with the reason. Leave room for
the requester to correct the framing. Batch independent questions and follow
up when one answer determines the next.

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
