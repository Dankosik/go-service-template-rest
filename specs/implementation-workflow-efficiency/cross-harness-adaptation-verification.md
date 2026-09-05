# Cross-harness implementation adaptation

Date: 2026-09-05. User approved adapting the completed Codex instruction work
to Claude Code, Qwen Code, Grok Build, Cursor, and OpenCode. This is local
template work, not publication or synchronization of real consumers.

## Changes

Context And Lifetime and Nested Execution now live in the existing shared
Agent Harness owner. Adapters translate fresh launch, continuation, nesting,
isolation, and depth into native controls. The worker role delegates only when
its selected adapter permits it. The orchestrator skill no longer contradicts
ledger-controlled Lead reuse by requiring every Lead to be fresh.

| Harness | Adaptation | Native boundary retained |
| --- | --- | --- |
| Codex | References the shared lifecycle/nested policy; native `fork_turns: none` and V2 remain. | Existing model authority, session ceiling, no App-chat fallback. |
| Claude | Ordinary named Lead carrier, mutable Agent/SendMessage/TaskOutput/TaskStop tools, portable project depth 3. | Fresh named contexts; no Agent.resume; Teams are optional and use root brokerage under this adapter. |
| Qwen | Ordinary named Lead carrier, mutable agent/list_agents/send_message/task_stop tools, portable project depth 5. | Teammates/forks/workflow agents cannot nest; nested agents are foreground-only. Root may broker background review without accepting the Lead's unit. |
| Cursor | Uses shared lifecycle, conditional review and Lead reuse, native resume mapping. | Native maximum two levels; deeper subsets return to the Lead. |
| OpenCode 1 | Project depth 3; workers may invoke only worker/evidence tasks. | Readonly roles retain task denial; background feature remains conditional; V2 schemas are not mixed in. |
| Grok | Corrects headless isolation to prepared worktree plus --cwd, explicit model-field limitations, and resume_from lineage/new ID handling. | Primary-session Leads retained until effective nested-depth support is proved; no claim that default depth 1 is a fixed product limit. |

Shared review triggers, progressive delta recheck, proof overlap, unit versus
delivery proof, Lead acceptance, root ledger ownership, and pre-Implementation
macro-phase boundaries remain authoritative. No new workflow phase, role type,
schema, scheduler, or lifecycle service was added.

Claude/Qwen depth-only settings are template-owned and selected by harness
profile. Their handwritten native Lead carriers follow the existing carrier
pattern; the generator preserves them and updates all six worker projections.
Credentials and machine settings remain outside the portable files.

## Evidence

The baseline is the dirty tree immediately before this turn. Frozen before,
initial after, repaired candidate, patches and hashes are local audit aids at
`/var/folders/9r/ft1t72w13r765bpf61v9mly00000gn/T/harness-portability-blxyf3k5/`.
Final 25-file manifest SHA-256:
`a26bb0de6679494ade72a307bbbedd9520b67f7b087807036848701fc7cbb28a`.
The changed-file set grew from 12,461 to 13,492 words including generated
copies, two native carriers, settings and focused fixture changes. Lifecycle
policy moved to an already-loaded owner; bootstrap and discovery catalog did
not gain another selector. These counts do not measure speed.

- `make template-owned-purity-check`: PASS, including carrier parity.
- `bash scripts/ci/template-sync-behavior-check.sh`: PASS. It checks native
  delegation permissions, preserved handwritten Leads, and real updates of
  stale consumer settings/carriers. Fixture depth values change only inside
  disposable repositories; real defaults remain Claude 3 and Qwen 5.
- Canonical ShellCheck on the two changed shell scripts: PASS; rerun on the
  fixture-only repair also PASS. `git diff --check`: PASS.
- All 75 relative links in changed instruction/carrier files resolve.
- Qwen 0.21.9 installed settings schema and SubagentManager carrier parser:
  PASS. Native tool names and foreground/nesting exclusions inspected.
- OpenCode 1.18.25 `--pure debug config` reads depth 3; native worker readback
  confirms task default-deny with worker/evidence exceptions and no Lead grant.
- Claude 2.1.227 installed tool/depth source and static carrier/settings shape
  checked; its native carrier parser was not independently exercised.

Primary evidence consulted in this conversation: [Claude subagents](https://code.claude.com/docs/en/sub-agents),
[Qwen agent tool](https://qwenlm.github.io/qwen-code-docs/en/developers/tools/task/),
[Cursor subagents](https://cursor.com/docs/subagents), [OpenCode schema](https://opencode.ai/config.json),
and [Grok's depth implementation](https://github.com/xai-org/grok-build/blob/72a61251fcffb464bcc687aeb5a998e5a98ec0c9/crates/codegen/xai-grok-tools/src/implementations/grok_build/task/mod.rs#L393),
plus installed native sources. Grok's available upstream source is not the exact
installed commit, so deeper Grok execution remains unclaimed.

## Review and limits

Fresh independent reviewer, requested Astra/high with no inherited turns:
initial FAIL on a propagation fixture whose clone already had matching inputs.
The repair makes both settings and Lead bodies stale before apply and asserts
inequality before/equality after. Bounded delta recheck: **PASS**, no findings.
The reviewer verified all final hashes and retained unaffected native/policy
reasoning without repeating checks. Its before/after interpretation covered
lifecycle, review, Lead reuse, native depth, and missing-capability recovery.

This is structural/native-parser proof and static instruction review, not a
blinded model experiment or live cross-harness coding benchmark. Full nested
execution, cancellation and resume were not exercised in all five harnesses;
no speed or defect-rate equivalence is claimed. Final candidate readback
matches the reviewed hashes and unrelated tracked changes are preserved.
