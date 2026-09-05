# Codex ledger execution through nested subagents

Date: 2026-09-05. User authority: replace separate Codex App worker chats with
collaboration subagents throughout ledger execution, including nested
Orchestrator -> Lead -> worker -> further useful descendants, and enable the
required native settings. Scope is the source template's Codex harness.

## Changes and retained boundaries

- Codex's adapter uses the current root as Orchestrator and general-purpose
  subagents as unit Leads. App chat creation, App handoff, and App-message model
  escalation are removed from the execution route, including its fallback.
- Git worktree isolation uses ordinary repository lifecycle tools and explicit
  absolute checkout assignment; changing cwd does not reload native project
  configuration or grant missing write authority.
- Codex workers may delegate strict subsets. Each parent integrates and joins
  its subtree; writable scopes remain disjoint. Only the Lead accepts the unit
  and only the root writes the ledger. Parent replacement includes descendant
  reconciliation; interrupt alone is not claimed to terminate a subtree.
- `.agents/codex-project.toml` explicitly enables `features.multi_agent_v2` and
  `agents.enabled`; `.codex/config.toml` is regenerated. The existing ceiling of
  20 spawned agents is shared across all levels, excluding the root. No legacy
  `max_depth` is added. Default models and efforts remain Astra/high.
- The canonical worker role and its six generated carriers now contain the
  Codex-specific descendant allowance. Other harnesses retain their no-descendant
  policy and native restrictions. No generator code, role type, scheduler, or
  schema was added. Pre-Implementation macro-phase focus and review, protected
  invariants, proof requirements, and the earlier efficiency changes remain.

The dirty working tree immediately before this turn is the baseline. Eleven
files changed; all other captured instruction and earlier evidence bytes were
preserved. Before/after snapshots, task-only patch, manifest, and probe file are
local audit aids under
`/var/folders/9r/ft1t72w13r765bpf61v9mly00000gn/T/codex-nested-ledger-bbly382k/`.
Candidate manifest SHA-256:
`b7d021622320d7e56ca94c46585a24469c52a21669cfab7355ad717e41fc9e87`.
Changed-file word count is 3,564 -> 3,945 including generated copies. No new
mandatory instruction owner or catalog branch was introduced.

## Configuration and runtime proof

The current [OpenAI schema](https://developers.openai.com/codex/config-schema.json)
defines `max_depth` as V1-only and ignored by V2. The official
[subagent guide](https://learn.chatgpt.com/docs/agent-configuration/subagents)
documents enabling agents, root-excluding concurrency, defaults, and inherited
permissions. These sources were fetched on 2026-09-05; installed schemas and
native controls own the actual capability claim.

- `make template-owned-purity-check`: PASS, including generated carrier parity.
- `git diff --check`: PASS; all 20 relative links in the changed docs resolve.
- Both source/generated TOML parse with V2 and agents enabled, ceiling 20,
  unchanged model defaults, and no `max_depth`.
- Native `app-server --strict-config --stdio` followed by `initialize` and
  `config/read` for the repository cwd: PASS on CLI 0.147.0 and desktop bundled
  0.153.0-alpha.5. Filtered readback confirms V2, enabled agents, ceiling 20,
  `max_depth: null`, and root/default-child Astra/high. No inference calls or
  global configuration writes were used for this check.
- Live chain: `/root` -> `/root/nested_probe` -> `/root/nested_probe/branch` ->
  `/root/nested_probe/branch/leaf`. The leaf wrote exactly 23 bytes,
  `nested-worker-write-ok\n`; branch, Lead, and root independently verified the
  bytes and SHA-256
  `27dbf458e88ee9cf9eb288ee2c173478758e6090b7a95865ef9e4fdce09e1448`.
  Generated `worker-agent` descendants completed and were joined. The first
  attempt lacked the probe directory; root supplied it and the same identities
  completed the bounded retry. No repository code or App chat was created.

This proves recognized configuration in new native processes and live mutable
nesting through depth 3 in the current session. It does not prove unlimited
depth, capacity reclamation, automatic reload of the current session, or a
speed/quality improvement over App chats. No consumer synchronization or
publication was performed.

## Independent review

**PASS**, no findings, on the fixed candidate. The fresh reviewer received the
accepted scope and before/after snapshots, with Astra/high requested and no
inherited turns. It checked nested authority, write isolation, recovery,
configuration, generated ownership, and non-Codex preservation, and compared
the dispatch, nested subset, unavailable capability, capacity, parent-replacement,
and session-resume routes. This is static review plus the bounded runtime proof
above, not a comparative performance experiment.
The reviewer independently verified all eleven hashes and found no surviving
spawn blocker or authority regression. Runtime receipts were consumed without
duplicate execution. Final source readback matched the candidate; all other
captured instruction and prior evidence bytes remained unchanged.
