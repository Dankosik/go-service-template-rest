# Goal
status: done
Completion: T001 is accepted on one exact candidate; every local obligation and all 11 commands in the reviewed Test Design validation ladder are terminal-success for that candidate; the required Implementation Review passes; and no adopter/provider or deployment claim is made.
Global constraints: Preserve the hash-fixed Specification, reviewed Technical Design, and reviewed Test Design named by `transition.md`. Keep the complete removable `INBOUND_WEBHOOKS=none|standard-webhooks` profile as one acceptance unit, reuse the existing HTTP, PostgreSQL, SQLC, River, configuration, Problem, initializer, and Standard Webhooks owners, and add no dependency or parallel runtime mechanism. Treat every pre-existing dirty checkout byte as outside the inbound-webhooks candidate and preserve it; dirty-tree or unrelated proof is not acceptance evidence. Synthetic non-secret fixtures and isolated local PostgreSQL proof are allowed, but no real credential or secret acquisition, provider registration, network action, target migration, deployment, rollout, or other external effect is authorized.

## Tasks

- [x] T001: The source template and initialized outputs provide the complete removable durable Standard Webhooks ingress capability and satisfy every reviewed repository-local proof obligation.
  - Depends on: none
  - Provides: locally accepted inbound-webhooks source/profile implementation and exact local evidence for later adopter and deployment gates
  - Packet: tasks/T001-durable-standard-webhooks-ingress.md
Accepted: T001; evidence: focused inbound 566/race 327; Docker inbound 6; test-integration PASS; openapi/sqlc/migration-check PASS; template-init-check as union of profile runs; runtime-image-build 0; gosec 0; make check 0 lint/1273 tests; review PASS; TD-EXT-01/TD-REL-01 unclaimed; candidate: tree 8f470fa9face942bf5eeabad951a7babd9e3ef71 at /Users/daniil/Projects/Opensource/go-service-template-rest.impl-inbound-webhooks-t001

## No-implementation dispositions

- TD-EXT-01 remains an adopter/provider-owned compatibility and business-authority gate before endpoint enablement. Missing provider, typed decoder/handler, idempotent effect, payload classification, retention approval, and ordering/reconciliation inputs do not block T001 and cannot be manufactured by repository work.
- TD-REL-01 remains a deployment/database/security/operator-owned rollout and rollback gate. Missing target, immutable images and configuration generations, writable-primary read authority, ingress/TLS, secret delivery, capacity, stop/start, monitoring, and external-write authority do not block T001 and cannot be replaced by local proof.
