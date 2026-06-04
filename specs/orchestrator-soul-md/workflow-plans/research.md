# Research Phase Plan

Phase: research
Phase status: complete
Research mode: local
Mode rationale: the evidence surfaces were bounded and directly inspectable. Subagent fan-out would normally be useful for a non-trivial instruction-surface change, but the available subagent tool permits spawn only when the user explicitly asks for subagents, delegation, or parallel agent work. This session therefore records a local-only rationale instead of opening lanes.

## Research Tracks

| Track | Question | Evidence Target | Status |
| --- | --- | --- | --- |
| Runtime semantics | What does SOUL.md own in Hermes/OpenClaw-style agents? | Hermes official docs, Hermes prompt assembly docs, OpenClaw persona repository | complete |
| File-boundary practice | What belongs in SOUL.md vs AGENTS.md? | Hermes docs, OpenClaw persona examples, third-party OpenClaw bootstrap guidance | complete |
| Local integration constraints | How should this repository integrate SOUL.md without weakening AGENTS.md? | `AGENTS.md`, `docs/spec-first-workflow.md`, sibling setup-repo AGENTS.md files | complete |
| Synthesis | What personality stance best supports production-ready Go microservices without overengineering? | Fan-in from external and local evidence | complete |

## Preserved Research Notes

- `research/soul-md-agent-personality-practices.md`: source-backed evidence and handoff implications.

## Fan-In Summary

Must-answer-now questions are handled:
- `SOUL.md` is best treated as identity/personality, not as a second workflow contract.
- The strongest external distinction is stable persona in `SOUL.md`, project and operational rules in `AGENTS.md`.
- OpenClaw-style examples use short files with sections such as vibe, tone, rules, boundaries, and examples. For this repository, those sections should be adapted into professional engineering identity rather than character cosplay.
- Hermes uses SOUL.md as the first identity layer for its own runtime, but repo-local SOUL.md is not universal across hosts. This repository must integrate it explicitly, most likely through `AGENTS.md` precedence wording and an include/reference.
- The right personality target is "pragmatic senior service orchestrator": evidence-driven, direct, production-ready by default, allergic to accidental complexity, but willing to use deeper design when the risk genuinely calls for it.

## Evidence Limits

- OpenClaw official docs were only partially reachable through search/open results, so GitHub-hosted OpenClaw persona examples were weighted more heavily than SEO-style guides.
- Sibling GonkaGate setup repositories do not currently contain `SOUL.md`, so their value is local AGENTS.md boundary comparison rather than SOUL.md precedent.
- No live model-behavior test was run. Specification should treat this research as instruction-design evidence, not proof that a specific model will always obey the file.

## Stop Rule

Do not create `SOUL.md`, edit `AGENTS.md`, write `spec.md`, assemble design, create `tasks.md`, or implement checks in this session. Stop after research evidence and workflow state are recorded.

## Next Action

Start specification. The specification phase should decide the accepted SOUL.md purpose, content shape, AGENTS.md integration and precedence, validation obligations, and whether any separate design or challenge gate is needed.
