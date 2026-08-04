# Boundary decision verifier

Use this after a draft exists. Its job is to catch an unsafe omission or unsupported claim, not to redesign the integration or add report sections.

Give a fresh reviewer the exact user request and draft when a reviewer or subagent is available; otherwise apply the same review yourself. The reviewer does not inspect or change repository or provider state.

Check only these five surfaces:

1. **Contract fidelity:** preserve every named deadline, completion promise, partial-progress rule, key scope/window, and output limit. Provider evidence is operation-specific; a sibling endpoint inherits nothing. Lookup absence resolves ambiguity only when visibility/completeness is documented.
2. **Identity, owner, authority:** every side effect has a durably accepted local operation identity before provider I/O, distinct from provider key/reference. Every rendered row names its confirmed local owner or an owner gap; a proposed owner slot is not an assignment. Design, repository changes, local tests, sandbox calls, compensation, and production activation have separate authority.
3. **Recovery coverage:** every affected caller/effect and distinct callback, poll, or reconciliation path is represented. Each named ambiguity or failure class has an authoritative resolver, bounded signal, and owned unresolved state.
4. **Executable proof:** runnable code or an existing runnable command crosses each failure boundary requested by the user. When no repository harness is available in a read-only task, include a self-contained inline test using the language standard library or a fake; never label a proposed file path or future command executable. Lost-response proof commits the provider effect before losing the response and asserts the external effect count. Crash/restart proof injects the crash at the named boundary and asserts both visibility and checkpoint behavior.
5. **Scope and usability:** preserve requested item/test counts and word limit, remove repeated ledger prose, and add nothing that cannot change the verdict or safety.

Return only:

- `blocking omissions`
- `unsupported claims`
- `scope or format excess`

Write `clear` when all three are empty. The drafting agent revises once from concrete findings, then rechecks the requested shape; it does not copy the verifier output into the answer.
