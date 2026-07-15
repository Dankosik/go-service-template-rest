# Implementation / Validation / Closeout

On entering this macro phase, and only then, the root establishes the single Codex Goal required by `AGENTS.md`, launches the external CLI Worker that produces the requested change, reviews the returned diff, and proves the accepted outcome.

## Read When

- The request authorizes change/build/fix and required decisions are ready.
- Direct work has a clear inline outcome and proof, or structured/orchestrated work has a ready independently reviewed ledger.
- Existing implementation needs repair, review, validation, or closeout.

## Inputs

- Accepted direct outcome or current reviewed `tasks.md`.
- Required spec, design, test, and rollout decisions named by the work.
- Current repository state, including pre-existing user changes.
- Repository-owned generation and validation commands.

## Outputs

- In-scope code, tests, config, migrations, generated output, and docs.
- Updated task progress when a ledger exists.
- Review findings and repairs proportionate to risk.
- Fresh validation evidence and an evidence-clamped final claim.

## Implement

1. Inspect the owning code, callers, siblings, tests, and generated/manual boundary before editing.
2. For a defect, fix the narrowest owning surface whose contract the reproducer proves is violated; do not patch only the reported entry point when sibling callers share that contract.
3. Preserve accepted behavior, ownership, failure, cleanup, rollout, and proof decisions.
4. Prefer stdlib and existing repository patterns. Do not add a dependency, interface, helper layer, or architectural pattern without a present need and an owner.
5. Remove replaced paths and adjacent stale artifacts unless current compatibility evidence requires retention.
6. Keep changes reviewable and avoid unrelated cleanup.
7. If implementation exposes a missing product, contract, source-of-truth, ownership, test-strategy, or rollout decision, stop and reopen that owner instead of inventing it.

Every authorized implementation change is produced in an isolated Git worktree by an external Codex CLI Worker. Direct work assigns its one accepted outcome to one Worker. Ledger work assigns exactly one ready task to one Worker; only one write Worker runs at a time. The Worker owns that outcome or task until root acceptance or a genuine upstream blocker. If anything required is missing, the root resumes the same Worker session in the same worktree with concrete bounded gaps and does not author the repair. After acceptance, the root records task evidence and launches a fresh Worker/session for the next ready task. Worker checking is task-local deterministic implementation feedback, not acceptance: the Worker runs relevant automated checks, including behavior tests where relevant, and reports commands and raw results; its criteria mapping is traceability only. The root independently judges business completeness, code quality, test adequacy, scope, and final acceptance.

### Transient Worker Model Catalog

The exact allowlisted Worker models are `gpt-5.6-terra` and `gpt-5.6-sol`, validated against the OpenAI latest-model guide and native Codex catalog on 2026-07-15. Before launch, the root sets and records `WORKER_MODEL` from observable characteristics of the accepted outcome; never inherit a default, use the floating `gpt-5.6` alias, or select another model.

Terra is the normal efficiency choice only when the outcome is clear, bounded and local, low consequence, free of unresolved design or ownership judgment, covered by relevant automated proof, and readily inspectable and falsifiable by the root. Routine test implementation qualifies only when its behavior and oracle are already accepted; the word `test` alone is not a routing rule. Sol is required for material uncertainty or an ambiguous, open-ended, difficult-to-debug, cross-boundary or cross-cutting outcome; material architecture or product judgment; a protected or high-consequence domain named by repository authority; a large evidence, tool, or context load; or a result that is difficult for the root to falsify locally.

Before the next Worker launch after either a Codex CLI upgrade or a change in the latest-model guide, revalidate and coordinate updates to the exact model allowlist, catalog transformation, and instruction checks. If the selected model is unavailable, freshness validation fails, or the effective model differs, stop; never silently fall back or substitute another model.

Before first launch, derive one full transient catalog from native `codex debug models`. Fail unless `WORKER_MODEL` is allowlisted and the catalog contains exactly one matching model, preserve every model and all other metadata, and change only that selected model's `multi_agent_version` to JSON `null`:

```bash
set -euo pipefail
WORKER_MODEL_CATALOG="$(mktemp -t codex-worker-model-catalog.XXXXXX.json)"
/opt/homebrew/bin/rtk proxy codex debug models | /opt/homebrew/bin/rtk proxy jq -e --arg worker_model "$WORKER_MODEL" '
  (["gpt-5.6-terra", "gpt-5.6-sol"] | index($worker_model)) as $allowlisted
  | ([.models[] | select(.slug == $worker_model)] | length) as $matches
  | if $allowlisted == null then
      error("WORKER_MODEL is not allowlisted")
    elif $matches != 1 then
      error("expected exactly one selected Worker model")
    else
      .models |= map(
        if .slug == $worker_model then .multi_agent_version = null else . end
      )
    end
' > "$WORKER_MODEL_CATALOG"
/opt/homebrew/bin/rtk proxy jq -e 'type == "object" and (.models | type == "array")' \
  "$WORKER_MODEL_CATALOG" > /dev/null
```

RTK filtering corrupts this large JSON stream; both commands must use raw proxy mode. Do not launch if derivation or raw JSON validation fails. Retain this exact file unchanged for the whole Worker session, pass it again on every resume, and delete it only after root acceptance or abandonment. Do not regenerate it between launch and resume, persist it as a workflow artifact, copy a pinned catalog into the repository, or mutate user/global Codex configuration.

### CLI Worker Launch And Resume

The root launches the Worker with the native CLI; do not add a launcher framework. The command snippets are exact raw RTK proxy invocations; execute them as shown. Before launch, define these transient shell variables:

- `WORKTREE`: absolute path to the isolated Git worktree;
- `WORKER_MODEL`: the recorded exact allowlisted model selected for this outcome;
- `WORKER_BRIEF`: task prompt file;
- `WORKER_SCHEMA`: structured-output JSON schema;
- `WORKER_FINAL`: structured final-result path;
- `WORKER_EVENTS`: JSONL event-stream path;
- `WORKER_STDERR`: stderr evidence path;
- `WORKER_CAPABILITY_CONFIG`: task-gated optional-capability baseline.

Use distinct `WORKER_EVENTS`, `WORKER_FINAL`, and `WORKER_STDERR` paths for the initial launch and every resume attempt; never overwrite prior Worker evidence.

Choose `WORKER_REASONING_EFFORT` before launch rather than inheriting it: `low` for mechanical, well-specified, low-risk work; `medium` as the ordinary implementation baseline; `high` or `xhigh` for complex or ambiguous work when representative evals show a material quality gain; and `max` only for the hardest quality-first work. Both allowlisted models permit only `low`, `medium`, `high`, `xhigh`, or `max`; reject `ultra` because it requests automatic task delegation. Select model and reasoning effort separately: do not compensate for a Sol-shaped task by maximizing Terra. Choose effort from task complexity, ambiguity, consequence of error, evidence/tool load, and latency/cost, and calibrate both routing and effort on representative local evals; do not assume the highest effort is best. Use this exact locally validated runtime contract:

Before launch, run `codegraph status`. If it fails or reports `Not initialized`, set `WORKER_CODEGRAPH_CONFIG=(-c 'mcp_servers.codegraph.enabled=false')`, record the raw-navigation fallback in the Worker brief, and use that array on this launch and every resume. Otherwise set `WORKER_CODEGRAPH_CONFIG=()`. Worker setup never initializes or reindexes CodeGraph.

Set these lean code defaults before launch:

```bash
WORKER_CAPABILITY_CONFIG=(
  -c web_search=disabled
  -c 'sandbox_workspace_write.network_access=false'
  -c 'mcp_servers.context7.enabled=false'
  -c 'mcp_servers.headroom.enabled=false'
  -c 'mcp_servers.node_repl.enabled=false'
  -c 'mcp_servers.openaiDeveloperDocs.enabled=false'
  --disable plugins
  --disable apps
)
```

The root removes or replaces only the matching baseline override when the accepted task or its automated proof requires that capability (including Headroom compression/retrieval markers), records the grant and reason in the Worker brief, and preserves the same array on resume; prefer a specific documentation MCP over broad web or shell access.

Hooks are a Worker readiness dependency: before launch, root verifies active command-hook definitions are reviewed and trusted; a changed or untrusted hook stops for review instead of using `--dangerously-bypass-hook-trust`.

```bash
/opt/homebrew/bin/rtk proxy codex \
  --model "$WORKER_MODEL" \
  -c "model_catalog_json=\"$WORKER_MODEL_CATALOG\"" \
  -c "model_reasoning_effort=\"$WORKER_REASONING_EFFORT\"" \
  -c 'notify=[]' \
  -c 'check_for_update_on_startup=false' \
  -c 'features.multi_agent=false' \
  -c 'features.multi_agent_v2.enabled=false' \
  -c 'features.multi_agent_v2.max_concurrent_threads_per_session=1' \
  --disable chronicle \
  --disable goals \
  --disable memories \
  --enable hooks \
  "${WORKER_CAPABILITY_CONFIG[@]}" \
  "${WORKER_CODEGRAPH_CONFIG[@]}" \
  --cd "$WORKTREE" \
  --sandbox workspace-write \
  --ask-for-approval never \
  exec \
  --strict-config \
  --json \
  --output-schema "$WORKER_SCHEMA" \
  -o "$WORKER_FINAL" \
  - < "$WORKER_BRIEF" > "$WORKER_EVENTS" 2> "$WORKER_STDERR"
```

After launch, read `WORKER_SESSION_ID` from the first `thread.started.thread_id` event in `WORKER_EVENTS`. A missing, blank, or ambiguous ID is a launch/intake failure; do not start a replacement Worker.

An `item.completed` event with `item.type="error"` is non-terminal by shape: inspect its message rather than treating it as automatic failure or ignoring it. Do not globally filter stderr or warning events. Technical success requires exit status zero, exactly one session ID, a completed turn, a schema-valid structured final result, the selected model with no reroute or substitution, and no unresolved permission or contract violation. Do not trust model prose self-identification. Because `codex exec --json` may omit the model, verify the effective model from persisted session or turn metadata whenever the event stream does not establish it. Preserve and inspect `WORKER_STDERR`; benign diagnostics alone do not fail an otherwise complete run.

For bounded corrections, resume the same Worker with the same `WORKER_MODEL`, reasoning effort, unchanged catalog file, worktree, sandbox, approval policy, output schema, and multi-agent-disabled flags; never reroute the session or reselect its model:

```bash
/opt/homebrew/bin/rtk proxy codex \
  --model "$WORKER_MODEL" \
  -c "model_catalog_json=\"$WORKER_MODEL_CATALOG\"" \
  -c "model_reasoning_effort=\"$WORKER_REASONING_EFFORT\"" \
  -c 'notify=[]' \
  -c 'check_for_update_on_startup=false' \
  -c 'features.multi_agent=false' \
  -c 'features.multi_agent_v2.enabled=false' \
  -c 'features.multi_agent_v2.max_concurrent_threads_per_session=1' \
  --disable chronicle \
  --disable goals \
  --disable memories \
  --enable hooks \
  "${WORKER_CAPABILITY_CONFIG[@]}" \
  "${WORKER_CODEGRAPH_CONFIG[@]}" \
  --cd "$WORKTREE" \
  --sandbox workspace-write \
  --ask-for-approval never \
  exec resume "$WORKER_SESSION_ID" \
  --strict-config \
  --json \
  --output-schema "$WORKER_SCHEMA" \
  -o "$WORKER_FINAL" \
  - < "$WORKER_BRIEF" > "$WORKER_EVENTS" 2> "$WORKER_STDERR"
```

When a required accepted task-owned path is read-only under `workspace-write`, launch or resume the same Worker with top-level `--add-dir "$TASK_WRITABLE_PATH"` after `--cd` and before `exec`. Grant only that path, keep `workspace-write` and approval `never`, and never broaden the whole sandbox. Do not use `--ephemeral` or `--skip-git-repo-check`. The root records the direct outcome or task ID, session ID, model, catalog path, reasoning effort, worktree, JSONL event stream, structured final result, stderr evidence path, status, intake disposition, and proof in transient execution context or existing ledger evidence, never a second task ledger.

The outcome-first Worker brief contains: outcome; one direct outcome or one task ID; context and authorities; required repository skills when applicable; editable boundary; forbidden decisions and actions; acceptance criteria; required proof; structured return; and stop/blocker condition. It tells the Worker that it is not root and must not create, continue, or complete a Goal; delegate; self-accept; update task or workflow status; start another task; or claim repository completion. The structured return contains status, summary, changed files, criteria traceability, commands/results, and blockers.

Rerun relevant proof in the integration workspace when worker evidence does not already establish the integrated state. Do not rerun an unchanged command without a changed risk surface. Before an intentional stop, record the blocker and next executable task; on resume, follow the shared [Resume Order](../shared/artifact-model.md#resume-order) instead of reconstructing progress from chat.

## Review

Always inspect the final diff for:

- correctness against accepted behavior and invariants;
- error, cancellation, retry, concurrency, transaction, and resource-lifetime behavior where relevant;
- contract/generated-source drift;
- security, privacy, money, data, and rollout risk in scope;
- ownership, unnecessary abstraction/dependencies, and stale replacement surfaces;
- tests that prove behavior rather than implementation detail.

The root reviews every external Worker result and its proof before acceptance. Apply every matching review skill locally and evaluate all materially affected risk and specialist lenses in one coherent root inspection. A review skill supplies a method, not a subagent lane. Never launch a built-in subagent, reviewer, specialist, or re-review lane anywhere inside implementation/validation/closeout.

Every ledger task receives root acceptance review before the next task starts. After every task is accepted, generated outputs and task evidence are current, and terminal validation is complete, the root reviews the final integrated diff against the accepted outcome and every affected lens. An explicitly requested independent review of completed implementation starts only as a separate read-only boundary outside this macro phase; it is not an implementation gate.

Return every implementation-owned finding to the Worker session that owns the affected direct outcome or task. Collect compatible findings for that Worker as one bounded correction, resume its session, rerun affected proof, and have the root re-inspect the correction and every transitively affected lens. Reopen upstream decisions rather than broadening implementation. A passing command is insufficient when its implemented fixtures or assertions do not exercise the binding proof obligation; a matching selector name alone is not evidence.

Treat edits to tests, fixtures, golden files, skip or exclusion settings, lint/build configuration, and proof commands as proof-surface changes. They require an accepted task or behavior reason; a green result obtained by weakening or removing an oracle or bypassing a triggered gate is invalid.

Validation, in-scope Worker repair, root re-inspection, revalidation, and closeout run automatically in the same root session. An implementation-owned failure never produces a next-session prompt.

## Validate

Run focused proof while implementing, then one terminal fresh evidence set for the frozen candidate. Do not rerun an unchanged command unless a new patch, finding, or required final bundle changes what it proves. The terminal set covers the claim with:

- targeted tests for changed behavior;
- build, type, lint, race, integration, or repository gates relevant to affected packages;
- contract, migration, generation, or mirror drift checks when their source changes;
- integrated target-environment proof across the affected deployment graph when the accepted outcome is system-wide; provider deployment status or one component's readiness alone is insufficient;
- a smoke/manual check when automated proof is unavailable or insufficient;
- targeted negative searches for identifiers and references that should be gone.

Worker output, cached results, unrelated green checks, skipped commands, and too-narrow tests do not prove the claim. When a required check cannot run, record the command, reason, narrower evidence, and unverified remainder.

Reconcile both directions: every accepted obligation and every ledger task on the current completion path maps to its implementation or an already accepted evidence-backed no-implementation disposition, and to adequate proof; every material change maps back to accepted scope. Keep this reconciliation inline unless an existing ledger owns it. Preserve unrelated pre-existing changes.

## Close Out

Mark a task complete only after its proof passes. The final response states:

- what changed;
- the most important design/behavior consequence;
- validation actually run and result;
- remaining risk, unavailable proof, or blocker;
- the exact reopen owner when unfinished.

Use `complete`, `fixed`, `ready`, or equivalent only when fresh evidence supports the full claim. A blocker is a valid outcome, not successful completion.

## Stop Rule

Finish when the direct outcome or every ledger task has passed root acceptance, the root has reviewed the final integrated diff and affected lenses, the accepted completion condition is met, and relevant proof passes. Return implementation-owned gaps to their Worker session. Stop and reopen planning, test design, technical design, specification, research, or user/external authority only when that owner must change a decision or supply unavailable evidence.
