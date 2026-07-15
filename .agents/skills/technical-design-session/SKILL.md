---
name: technical-design-session
description: "Own the root technical-design macro phase: complete system/integration and Go-ownership design, run technical-design review, repair findings, and reach the phase stop rule. Use for end-to-end phase orchestration; use go-implementation-ownership-spec for design authoring without session ownership."
---

# Technical Design Session

Use when a ready spec still leaves runtime mechanism or Go ownership decisions open.

Read [system/integration design](../../../docs/spec-first-workflow/phases/system-integration-design.md), then [Go ownership design](../../../docs/spec-first-workflow/phases/go-code-ownership-design.md). When a durable design artifact is triggered, keep both in one compact `design/overview.md` unless a focused split materially improves ownership or review.

Follow the system/integration phase's fan-out and review rules.

Treat review as an internal checkpoint: the owning root repairs findings, re-reviews to the shared convergence condition, and continues in the same authorized request. Stop after findings only when the user explicitly requested a standalone read-only design review.

Before review, apply the router's [implementation-input closure](../../../docs/spec-first-workflow.md#implementation-input-closure) rule; prose promises or future fixtures do not make a design executable.

Success means test design or planning can proceed without reopening accepted behavior, source of truth, system mechanism, or Go ownership. Reopen the narrowest upstream owner when that condition is not met.
