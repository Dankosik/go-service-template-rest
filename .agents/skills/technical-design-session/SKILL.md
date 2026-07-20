---
name: technical-design-session
description: "Design closure: Use when the root technical-design phase needs end-to-end system/integration and Go-ownership decisions, risk-triggered review, repair, and its stop rule. Own phase orchestration and accepted design readiness; Skip when implementation or delegated placement-only authoring is requested."
---

# Technical Design Session

Must read [System / Integration Design](../../../docs/spec-first-workflow/phases/system-integration-design.md) and [Go Code / Ownership Design](../../../docs/spec-first-workflow/phases/go-code-ownership-design.md); together they own design decisions, review routing, repair, and stop rules. Drive design closure by reconstructing every material flow and changed responsibility from the accepted spec and affected runtime, then close mechanism and Go ownership together. Self-review is the default. Run exactly one independent read-only review only when explicitly requested or when the fixed decision is high-impact, hard to reverse, cross-owner, or weakly falsifiable. Repair findings or reopen their owner; obtain fresh review only when that triggered review occurred and a semantic repair changed material. Stop only when both owners permit test design or planning, or name the upstream owner to reopen.
