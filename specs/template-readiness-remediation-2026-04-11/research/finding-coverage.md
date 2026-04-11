# Review Finding Coverage

This file maps every finding from the read-only template-readiness review into the remediation plan. It exists to prevent implementation from closing only the five inline findings while dropping Nice To Have or residual research items.

## Coverage Matrix

| Review item | Priority from review | Remediation mapping |
| --- | --- | --- |
| Add a worked feature path | P2 | T001 |
| Clarify domain type placement | Docs/onboarding finding | T002 |
| Add telemetry placement recipe | Docs/onboarding finding | T003 |
| Add test placement matrix | Docs/onboarding finding | T004 |
| Add online migration safety guidance | P2 | T005 |
| Add list-limit guidance for SQL `LIMIT` sample pattern | Data finding | T006 |
| Add transaction recipe and avoid generic helper too early | Data finding | T007 |
| Add DB-required feature bootstrap guard/wiring guidance | Data finding | T008 |
| Clarify Redis/Mongo as guard-only extension stubs | P2 | T009 |
| Add config-key onboarding recipe | Docs/onboarding finding | T010 |
| Add strict-server endpoint checklist | API/HTTP finding | T011 |
| Add future parameterized route-label proof note | API/HTTP residual risk | T012 |
| Correct `openapi-check` versus `openapi-breaking` wording | QA finding | T013 |
| Add README "start here for feature work" pointer | Docs/onboarding finding | T014 |
| Refresh stale project tree and classify `specs/` / generated outputs | Architecture Nice To Have | T015 |
| Prove config keys reach `buildSnapshot` | P2 | T016 |
| Remove or clarify ignored `enforceSecretSourcePolicy` parameter | Maintainability finding | T017 |
| Harden config test that writes under real repo path | QA finding | T018 |
| Widen API runtime contract gate | P2 | T019 |
| Tighten manual root-route policy for all manual root routes | API/HTTP finding | T020 |
| Harden HTTP server lifecycle test pattern | QA finding | T021 |
| Close `httpx` package-name convention finding | Maintainability design escalation | T022 |
| Strengthen Postgres sample payload/timestamp assertions | QA finding | T023 |
| Move test-only Postgres repository factory out of production helper surface | Maintainability Nice To Have | T024 |
| Reduce startup dependency label drift from same-typed strings | Maintainability finding | T025 |
| Resolve tracked `.artifacts/test/*` generated outputs | Architecture Nice To Have | T026 |
| Avoid generic `common`/`util`, generic repositories, migration frameworks, service registries, and Redis/Mongo adapters without real consumers | Avoid/over-abstraction guidance | Enforced by `spec.md` non-goals and `design/ownership-map.md`; verify during implementation closeout. |
| Do not wire `ping_history` into `ping` | Avoid/over-abstraction guidance | Enforced by `spec.md` constraints and task surfaces; verify during implementation closeout. |
| Consider data/security/reliability review if Redis/Mongo become real adapters | Handoff note | Out of scope because T009 keeps them stubs; future separate task if promoted. |

## Closeout Rule

Implementation is not complete until each task above is either checked in `tasks.md` with evidence, or explicitly closed as a no-op/accepted residual risk with rationale.

## Phase 1 Closeout Notes

- T001-T015 are completed in `tasks.md` with Phase 1 docs evidence.
- Remaining open implementation coverage starts at T016 for Phase 2 config and HTTP guardrails.

## Phase 2 Closeout Notes

- T016-T022 are completed in `tasks.md` with fresh config, HTTP, OpenAPI, and diff-check evidence.
- The OpenAPI drift check now scopes generated-code drift to `internal/api/openapi.gen.go`, so the Phase 1 hand-written `internal/api/README.md` guidance no longer blocks `make openapi-check` as false codegen drift.
- Remaining open implementation coverage starts at T023 for Phase 3 data, bootstrap, artifacts, and final validation.

## Phase 3 Closeout Notes

- T023-T028 are completed in `tasks.md` with fresh Postgres, bootstrap, integration, OpenAPI, broader test, lint, and diff-check evidence.
- The Postgres sample remains sample-only: `ping_history` was not wired into `ping`, and the test-only querier factory was moved out of production code.
- Startup dependency label cleanup stayed in `cmd/service/internal/bootstrap` as a small same-package label/spec shape; no lifecycle framework, service registry, generic repository, `common`/`util` package, Redis adapter, or Mongo adapter was added.
- Tracked generated `.artifacts/test/*` report outputs were deleted for untracking in the next commit, and `.gitignore` now ignores `.artifacts/test/`; no deliberate tracked-sample decision remains.
