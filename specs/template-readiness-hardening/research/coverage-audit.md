# Template Readiness Research Coverage Audit

## Purpose

This audit checks that every actionable point surfaced during the template readiness research is either planned, explicitly deferred, already covered, or rejected as over-abstraction.

## Planned For Implementation

| Research point | Coverage |
| --- | --- |
| OpenAPI runtime-contract target misses security-decision guard. | `tasks.md` T001. |
| Protected endpoint auth placement is underspecified. | `tasks.md` T003 and T015. |
| Protected-operation 401/403 Problem response helper can accept any inline schema. | `tasks.md` T002. |
| Ping history sample does not model bounded SQL `LIMIT`. | `tasks.md` T004-T006. |
| Ping history sample can imply infra-owned records are production business types. | `tasks.md` T007. |
| Redis store-mode readiness policy has config/bootstrap duplicate ownership. | `tasks.md` T008-T010. |
| README placement guide link is too late for production-code authors. | `tasks.md` T011. |
| CONTRIBUTING boundary summary omits placement-guide link and non-HTTP/data/config/test surfaces. | `tasks.md` T012. |
| Build/test command docs list feature checks but do not point back to placement or warn about validation scope. | `tasks.md` T013. |
| Architecture baseline wording for `internal/domain` is looser than the consumer-owned-first rule. | `tasks.md` T014. |
| Missing app-only feature recipe. | `tasks.md` T015. |
| Missing first real SQL feature replacement recipe for `ping_history`. | `tasks.md` T015. |
| Missing existing examples to inspect. | `tasks.md` T015. |
| Missing DB-required feature bootstrap sketch. | `tasks.md` T015. |
| Test placement guidance is split and `test/README.md` omits some feature author surfaces. | `tasks.md` T016. |
| README quality gates omit migration rehearsal shortcut and skip-output caveat. | `tasks.md` T017. |

## Explicitly Deferred

| Research point | Rationale |
| --- | --- |
| Add a bootstrap-local app/handler assembly helper before `Run` grows. | Good future cleanup, but not required to fix the guardrail/docs/copy-path defects. Defer until a real multi-feature bootstrap change or an implementation session explicitly accepts it. |
| Extract a startup failure recorder for repeated span/metric rejection recording. | Broader than this template-readiness pass and may need observability review. |
| Split `startupLogComponentStartupProbes` into more precise component labels. | Operational label taxonomy change should be handled with observability intent, not silently inside this hardening pass. |
| Extract degraded dependency logging helper. | Consider only if startup failure recorder work is accepted; avoid helper churn now. |
| Add reusable OpenAPI `Unauthorized` and `Forbidden` components. | Defer until a protected endpoint or guardrail change needs concrete contract components. Do not add unused components for completeness. |
| Add unique constraint/conflict mapping, transaction isolation, idempotency, and cursor pagination examples. | Useful for the first production data feature, but too broad for this template-hardening pass. |
| Add a real Redis adapter. | Rejected for this pass; Redis remains a guard-only extension stub until a real app feature needs runtime behavior. |

## Already Covered / Leave Stable

| Research point | Existing coverage |
| --- | --- |
| Keep generated API output as derived and never hand-edit `internal/api/openapi.gen.go`. | Existing docs and `internal/api/README.md`; T003 reinforces it. |
| Keep manual `/api/...` routes out of chi root router. | Existing router tests and docs; T003/T015 reinforce it. |
| Keep route-label proof for parameterized endpoints. | Existing `internal/api/README.md` already states this; T003 should preserve or reword it. |
| Keep Redis/Mongo guard-only guidance consistent. | Existing docs are consistent; T015 may add links but should not duplicate the full policy. |
| Avoid `common`, generic repositories, transaction managers, DI containers, broad `domain` buckets, or generic auth framework. | Captured in `spec.md` non-goals and `design/ownership-map.md`. |

## Audit Conclusion

All research points are accounted for as planned, deferred, already covered, or rejected as over-abstraction. Implementation should use `tasks.md` as the executable scope and should not treat deferred items as implicit coding work.
