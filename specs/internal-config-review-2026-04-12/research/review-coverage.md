# Review coverage audit

This file preserves the full fan-in checklist from the read-only review lanes so the implementation session can verify that every researched point was either accepted into the plan, intentionally filtered, or explicitly deferred.

## Coverage matrix

| ID | Source | Point | Disposition | Covered by |
| --- | --- | --- | --- | --- |
| R001 | workflow adequacy challenge | Review workflow-control artifacts were sufficient; no blocking adequacy findings. | Closed. | `workflow-plan.md` challenge status and `workflow-plans/review-phase-1.md` reconciliation status. |
| R002 | `go-idiomatic-review` | `intFromFloat64` and `int64FromFloat64` compare against imprecise `float64(math.MaxInt*)` bounds before integer conversion. | Accepted. | `spec.md` Decisions; `design/component-map.md` parse helpers; `plan.md` validation plan; `tasks.md` `T001`. |
| R003 | `go-idiomatic-review` | `validateSampler` allows `NaN` because ordered comparisons with `NaN` are false. | Accepted, with stricter classification decision. | `spec.md` Decisions requires `parseFloat64` rejection and `ErrParse`; `design/component-map.md`; `design/sequence.md`; `plan.md`; `tasks.md` `T002`. |
| R004 | `go-idiomatic-review` | `net.SplitHostPort` splits host and port but does not validate numeric TCP ports for Redis and Mongo probe addresses. | Accepted. | `spec.md` Decisions chooses deterministic `strconv.ParseUint` and port range `1..65535`; `design/component-map.md`; `design/sequence.md`; `plan.md`; `tasks.md` `T003`. |
| R005 | `go-idiomatic-review` | `ErrorType(nil)` returns `"load"`. | Accepted. | `spec.md` Decisions; `design/component-map.md`; `plan.md`; `tasks.md` `T007`. |
| R006 | `go-idiomatic-review` handoff | `os.Lstat` plus `Mode().IsRegular()` makes some later symlink checks partly unreachable; changing behavior needs security/source-policy review. | Deferred, not a coding task. | `spec.md` Non-goals and Open Questions / Assumptions; `workflow-plan.md` deferred handoff; `design/overview.md`; `design/ownership-map.md`; `plan.md` accepted risks and reopen conditions. |
| R007 | `go-language-simplifier-review` | `loadConfigFile` calls `filepath.Clean(strings.TrimSpace(path))`, so whitespace-only input becomes `"."` and the empty-path branch is ineffective. | Accepted. | `spec.md` Decisions; `design/component-map.md`; `design/sequence.md`; `plan.md`; `tasks.md` `T004`. |
| R008 | `go-language-simplifier-review` | Redis mode normalization is split between `buildSnapshot` and `RedisConfig.ModeValue`. | Accepted. | `spec.md` Decisions; `design/component-map.md`; `plan.md`; `tasks.md` `T006`. |
| R009 | `go-language-simplifier-review` | `APP__APP__ENV` is hardcoded instead of derived from the namespace/key mapping rule. | Accepted. | `spec.md` Decisions; `design/component-map.md`; `design/sequence.md`; `plan.md`; `tasks.md` `T005`. |
| R010 | `go-language-simplifier-review` handoff | Empty-path guard touches config file policy if severity is treated as security-sensitive. | Accepted as readability/source-of-truth fix, not escalated to security. | `tasks.md` `T004`; `spec.md` keeps symlink/source-policy changes out of scope. |
| R011 | `go-language-simplifier-review` residual | Review was package-state, not diff-specific; domain/security correctness of allowed roots, secret-key classification, context budgets, and dependency behavior were not fully reviewed. | Recorded as residual scope boundary, not implementation work. | This file; `spec.md` Non-goals preserves source policy and config precedence; `plan.md` reopen conditions cover source-policy drift. |
| R012 | `go-design-review` | `internal/config` exports `ErrDependencyInit` and classifies it in `ErrorType`, but bootstrap owns dependency/network-policy failures. | Accepted. | `spec.md` Decisions; `design/component-map.md` bootstrap section; `design/ownership-map.md`; `plan.md`; `tasks.md` `T007` and `T008`. |
| R013 | `go-design-review` | `MongoProbeAddress` exported from config could be viewed as probe-target behavior that belongs near bootstrap. | Filtered. Repository docs explicitly assign this guard-only helper to `internal/config` for now. | `workflow-plan.md` filtered finding; `workflow-plans/review-phase-1.md` reconciliation; `spec.md` Non-goals; `design/overview.md`; `design/ownership-map.md`; `plan.md` assumptions/reopen conditions. |
| R014 | `go-design-review` residual | Config key ownership is spread across tags, defaults, manual snapshot reads, validation strings, and tests; existing drift tests reduce the risk. | Recorded as residual, not a new refactor. | `spec.md` Non-goals says not to replace the manual snapshot builder or validation file with a reflection/generic framework. |

## Audit result

Every review-lane point is represented in the handoff:

- accepted implementation work is in `tasks.md`;
- intentionally filtered work is named and justified;
- deferred or residual concerns are documented as non-goals, accepted risks, or reopen conditions.

No production code was edited during this audit.
