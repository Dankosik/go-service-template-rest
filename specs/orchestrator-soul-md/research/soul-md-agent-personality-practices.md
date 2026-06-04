# SOUL.md Agent Personality Practices Research

Accessed: 2026-06-04

## Question

What should a repository-level `SOUL.md` own for this Go service template, and how should it integrate with `AGENTS.md` so future orchestrators build production-ready microservices without artificial overengineering?

## Scope

In scope:
- SOUL.md purpose in Hermes/OpenClaw-style agents.
- File-boundary patterns between SOUL.md and AGENTS.md.
- Practical content shape for a production-focused Go service orchestrator persona.
- Integration implications for this repository's spec-first workflow.

Out of scope:
- Final `SOUL.md` wording.
- Final `AGENTS.md` edits.
- Runtime implementation or tests.
- Changing repository workflow gates during research.

## Source Map

| Source | Why It Matters |
| --- | --- |
| Hermes user guide, `Use SOUL.md with Hermes`: https://hermes-agent.nousresearch.com/docs/guides/use-soul-with-hermes/ | Primary official source for SOUL.md purpose, boundary with AGENTS.md, structure, and troubleshooting. |
| Hermes developer prompt assembly docs: https://github.com/NousResearch/hermes-agent/blob/main/website/docs/developer-guide/prompt-assembly.md | Explains prompt ordering: SOUL.md as identity layer, project context files later, and supported customization surfaces. |
| Hermes personality docs in repo: https://github.com/NousResearch/hermes-agent/blob/main/website/docs/user-guide/features/personality.md | Confirms SOUL.md is durable identity and AGENTS.md owns project-specific rules. |
| OpenClaw persona repository: https://github.com/will-assistant/openclaw-agents | Public repository of SOUL.md/IDENTITY.md/AGENTS.md persona examples and template structure. |
| GLaDOS SOUL.md example: https://github.com/will-assistant/openclaw-agents/blob/main/agents/sci-fi/glados/SOUL.md | Shows concise sections for vibe, tone, personality rules, example dialogue, and boundaries. |
| Will SOUL.md example: https://github.com/will-assistant/openclaw-agents/blob/main/agents/humor/will/SOUL.md | Shows a more direct "persona and boundaries" shape with communication style and knowing when to drop the bit. |
| Local setup-repo AGENTS.md files: `/Users/daniil/Projects/Opensource/openclaw-setup/AGENTS.md`, `/Users/daniil/Projects/Opensource/hermes-agent-setup/AGENTS.md`, `/Users/daniil/Projects/Opensource/gonkagate-claude-code/AGENTS.md` | Local precedent for keeping product invariants and workflow truth in AGENTS.md, not in persona files. |
| This repository's `AGENTS.md` and `docs/spec-first-workflow.md` | Authority for workflow gates, orchestration, phase boundaries, and artifact ownership. |

## Findings

### F1. SOUL.md should own identity, not workflow authority.

Hermes states that `SOUL.md` is about who the agent is and how it speaks: tone, personality, communication style, directness, stylistic avoids, and behavior under uncertainty or ambiguity. It explicitly says not to use SOUL.md for repo-specific coding conventions, file paths, commands, service ports, architecture notes, or project workflow instructions; those belong in AGENTS.md.

Implication: for this repository, `SOUL.md` can say "be a pragmatic senior service orchestrator" and "prefer production-ready simplicity." It should not restate task-ledger gates, `rtk` command rules, artifact shapes, subagent protocol, Go package layout, or validation commands.

Confidence: high for Hermes-style hosts; medium for generic hosts because SOUL.md loading is host-dependent.

### F2. AGENTS.md must stay the precedence authority.

This repository's `AGENTS.md` already owns non-negotiable invariants: orchestrator authority, read-only subagents, task-ledger gating, phase boundaries, provider-contract verification, and validation evidence. If `SOUL.md` duplicates or softens those rules, future agents may choose whichever wording is easier.

Implication: `AGENTS.md` integration should include a short precedence rule: load/apply `SOUL.md` as orchestrator personality and communication defaults, but `AGENTS.md` and task-local artifacts override it for workflow, scope, gates, commands, and implementation authority.

Confidence: high from local repository contract and Hermes SOUL.md vs AGENTS.md guidance.

### F3. Repo-local SOUL.md is not automatically portable across hosts.

Hermes loads SOUL.md from `HERMES_HOME`, not from arbitrary current working directories, and its docs warn users not to confuse repo-local SOUL.md with the global instance file. Hermes prompt assembly places SOUL.md in the stable identity layer and project context later. OpenClaw persona repositories copy SOUL.md into an agent workspace. Codex-style repo loading is different again and already uses `AGENTS.md` as the repository instruction file.

Implication: this template should not assume that creating root `SOUL.md` alone changes every agent runtime. The specification must decide the integration mechanism. For this repo, the practical path is likely:
- add root `SOUL.md`;
- reference it from `AGENTS.md` using the repository's existing include convention;
- document that hosts which support native SOUL.md may also load it directly, but AGENTS.md remains authoritative.

Confidence: high for Hermes; medium for OpenClaw because official docs were partially unreachable; high for this repo because `AGENTS.md` is definitely loaded in current context.

### F4. Effective SOUL.md examples are short, concrete, and behavioral.

The OpenClaw persona repository uses small SOUL.md files, often around 30-50 lines, with sections such as vibe, tone, personality rules, boundaries, and example dialogue. The examples are character-heavy, but the reusable pattern is structural: a compact identity, a few behavior rules, and explicit boundaries.

Implication: this repository should adapt the shape, not the theatricality. A production Go service template needs:
- identity: pragmatic senior Go service orchestrator;
- operating beliefs: correctness, evidence, maintainability, operational reality;
- engineering balance: production-ready default, risk-scaled depth, no accidental complexity;
- ambiguity behavior: inspect first, state assumptions, escalate only on real risk triggers;
- avoid list: sycophancy, hype, performative architecture, premature abstraction, false simplicity, fake certainty.

Confidence: medium-high. The examples are public open-source files, but many are entertainment/persona templates rather than production engineering templates.

### F5. The desired balance is "simple enough, not simplistic."

The user concern is correct: "do not overengineer" can cause models to under-design real reliability, security, data, or contract problems. The better instruction is not "avoid complex solutions"; it is "make complexity earn its keep." Use the simplest design that satisfies the accepted scope's invariants, failure modes, and validation obligations. Add architecture only when the risk or workload needs it.

Implication: SOUL.md should define a judgment posture:
- challenge both overengineering and underengineering;
- prefer standard-library/native Go and existing repo patterns first;
- add abstractions only for stable ownership, meaningful duplication, or real policy boundaries;
- make tradeoffs explicit instead of defaulting to "MVP now, hardening later";
- keep production-ready target state inside accepted scope, while avoiding speculative scale features.

Confidence: high from this repository's workflow contract and the user's stated goal.

### F6. Local setup repositories show AGENTS.md as product truth, not persona.

The sibling setup repositories inspected (`openclaw-setup`, `hermes-agent-setup`, `gonkagate-claude-code`) do not currently carry SOUL.md files. Their AGENTS.md files define product identity, fixed invariants, repository structure, and code ownership. That is useful contrast: they show how quickly AGENTS.md becomes a dense product/workflow authority.

Implication: adding SOUL.md to this Go service template should reduce personality drift and make AGENTS.md less tempted to accumulate tone/style instructions. It should not move product invariants out of AGENTS.md.

Confidence: high for current local checkouts; future sibling repos may change.

## Recommended Specification Direction

For the next phase, specify a root `SOUL.md` with this boundary:

- Purpose: stable orchestrator personality for this repository template.
- Role: pragmatic senior Go microservice orchestrator.
- Goal: maximize accurate, production-ready technical outcomes with risk-scaled rigor and minimal accidental complexity.
- Non-goals: workflow rules, commands, file paths, repo conventions, subagent protocol, validation matrix, or task ledgers.
- Precedence: `AGENTS.md` and task-local artifacts override `SOUL.md` for operational rules and decisions.
- Integration: explicit `AGENTS.md` reference/include, because root SOUL.md is not guaranteed to be loaded by every host.

Suggested SOUL.md sections for specification:

1. `Role`
2. `Core Operating Beliefs`
3. `Engineering Balance`
4. `Default Behavior Under Ambiguity`
5. `Communication Style`
6. `Avoid`
7. `Boundaries`

The specification should keep the file compact. A useful target is enough detail to shape model behavior, not a second manual.

## Conflicts Or Weak Evidence

- Some OpenClaw search results were SEO-style guides or community summaries, not upstream docs. They were used only when consistent with GitHub-hosted examples and Hermes official docs.
- OpenClaw examples are optimized for distinctive character voice. For this repository, distinctive technical judgment matters more than entertainment.
- Hermes native SOUL.md semantics are global to a Hermes instance, while this repository likely needs repo-local behavior. That is an integration decision, not something the research can assume away.

## Handoff Implications

The next `specification-session` can start from this evidence and decide:

- accepted SOUL.md role and non-goals;
- exact AGENTS.md precedence wording;
- whether to add only root `SOUL.md` or also host-specific references;
- whether technical design depth is unnecessary because the change is docs/instruction-only;
- validation obligations for docs/instruction artifacts.

This note does not approve final decisions or authorize implementation.
