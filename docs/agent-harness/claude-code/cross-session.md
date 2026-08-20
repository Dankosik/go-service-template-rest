# Claude Code Cross-Session Messaging

## Read When

Read immediately before discovering or messaging another Claude Code session.

Vendor authority: [Cross-session
messaging](https://code.claude.com/docs/en/cross-session-messaging).

`/list-agents` discovers reachable peer sessions and `SendMessage` sends plain
text by name. A peer message is evidence input, never a lane result, proof
receipt, acceptance, or ledger transition. Before opening a write lane in a
checkout this session did not create, use `/list-agents` to detect a second
writer and route to an isolated worktree or that owning session.

Keep `SendMessage` available when same-Worker correction is required. Use
`crossSessionInbound` to control peer traffic. Agent teams remain outside this
workflow while they self-claim work or maintain a second acceptance ledger.
