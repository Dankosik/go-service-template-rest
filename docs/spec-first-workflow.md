# Spec-First Workflow (Universal, Minimal)

## 1. Purpose

This workflow is intentionally one-size-fits-all.

The same process must work for:
- small fixes,
- medium product changes,
- large cross-cutting features.

No tiering, no mandatory phase matrix, no mandatory file fan-out.

## 2. Core Rules

1. One canonical decision artifact: `specs/<feature-id>/spec.md`.
2. For long/ambiguous work, validated research may be stored in separate `research/*.md` files.
3. `spec.md` contains final decisions; research files contain evidence and exploration history.
4. Keep process smaller than the change. Add structure only when it removes real risk.
5. Avoid duplicate text across artifacts. A decision is written once and referenced.
6. No mandatory "N files" policy. Create extra files only when readability or reuse requires it.
7. No mandatory "at least two options" for every decision. Use option comparison only for irreversible or high-cost choices.
8. No mandatory status boilerplate (`updated/no changes required`) per domain file.
9. Code and spec stay in sync continuously; do not postpone drift cleanup.
10. Any readiness claim must include fresh command evidence.

## 3. Default Artifact Layout

Default for any feature:

```text
specs/<feature-id>/
  spec.md
```

Optional only when needed:

```text
specs/<feature-id>/
  research/
    <topic>.md   # validated research memory; format is flexible
  plan.md        # when execution steps are too large for one section in spec.md
  test-plan.md   # when test strategy is large enough to hide in spec.md
```

Research files may be created before final decision text is written in `spec.md`.
There is no universal mandatory template for `research/*.md`.
Do not create optional files preemptively.

## 4. Canonical `spec.md` Structure

Use one document with these sections:

1. `Context`
- what changes and why.

2. `Scope`
- in-scope,
- out-of-scope.

3. `Constraints`
- technical/business constraints that cannot be violated.

4. `Decisions`
- short decision log with stable IDs (`D-001`, `D-002`, ...),
- each item: decision, reason, impact.

5. `Open Questions / Assumptions`
- only unresolved items,
- each item has owner and unblock condition.

6. `Implementation Plan`
- ordered, concrete steps.

7. `Validation`
- smallest command set proving correctness,
- expected evidence.

8. `Outcome`
- what was implemented,
- what remains.

The structure is fixed; depth is variable.
For a small feature each section can be 1-3 bullets. For a large feature the same sections can be expanded.

## 5. Single Universal Loop

1. Clarify request and create the feature folder.
2. Resolve critical unknowns; persist validated research into `research/*.md` when needed.
3. Add final decisions in `spec.md` `Decisions`, linking to relevant research files when useful.
4. Convert decisions into an explicit `Implementation Plan` (in `spec.md` or `plan.md`).
5. Implement in small slices.
6. Run minimal useful validation after each meaningful slice.
7. Update `Outcome` and close resolved open questions.

No extra phase machinery is required.

In agent-centric execution:
- planning is owned by the orchestrator in the main flow (optionally via planning skill, e.g. `go-coder-plan-spec`);
- coding starts after planning and may use implementation skill (e.g. `go-coder`).

## 6. Scaling Rule (Without Tiering)

Scale by detail, not by workflow type:

- If the change is small, keep sections short.
- If the change grows, add detail inside the same sections.
- Split into `plan.md` or `test-plan.md` only when readability degrades.

The process stays the same.

## 7. Split Trigger (Readability Only)

Split from `spec.md` only if at least one is true:

1. Implementation steps become hard to review in a single section.
2. Test obligations become too long/noisy for the `Validation` section.
3. Multiple contributors need independent parallel workstreams.
4. Research volume is high enough that decision-quality evidence should be preserved separately from `spec.md`.

When split happens:
- keep `spec.md` as source of truth,
- keep research history in `research/*.md` and final decisions in `spec.md`,
- keep only links/summaries in extra files,
- do not duplicate full decision text.

## 8. Review Model

Use one review objective: reduce delivery risk.

Review checklist (apply only relevant items):
1. behavior correctness,
2. invariant/contract preservation,
3. reliability and failure paths,
4. security boundaries,
5. test adequacy,
6. operational safety.

No mandatory reviewer-role choreography is required.

## 9. Definition Of Ready / Done

Definition of Ready:
1. `spec.md` has clear scope, constraints, and first implementation steps.
2. Critical blockers are either resolved or explicitly tracked.

Definition of Done:
1. Implementation matches `spec.md` decisions.
2. Validation commands were executed and passed.
3. `Outcome` is updated and open questions reflect reality.

## 10. Migration From Legacy Spec Packages

Existing multi-file packages (`00/10/15/.../90`) are valid historical artifacts.

For new work:
1. use `spec.md` by default,
2. do not backfill legacy templates,
3. if editing a legacy package, you may collapse it into `spec.md` when convenient.

## 11. Anti-Patterns

1. Creating many files before concrete need appears.
2. Copying the same rationale into multiple artifacts.
3. Filling mandatory-looking templates with low-value text.
4. Turning process compliance into the main output.
5. Deferring obvious design decisions into coding by default.
