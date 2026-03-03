# 15 Domain Invariants And Acceptance

## Invariants
| ID | Type | Statement | Pass Condition | Fail Condition |
|---|---|---|---|---|
| DOM-001 | hard | Config snapshot is deterministic for identical inputs. | Repeated loads return identical `Config` and warnings. | Same input yields different snapshot. |
| DOM-002 | hard | Precedence is fixed: defaults < file < overlays < `APP__...`. | Effective value follows precedence in all tests. | Value resolved from lower-priority source. |
| DOM-003 | hard | Only namespace env keys participate in load. | Flat keys do not affect final config. | Flat key changes effective config. |
| DOM-004 | hard | Strict mode rejects unknown canonical keys. | `--config-strict=true` returns `ErrStrictUnknownKey` for unknown keys. | Unknown canonical key is silently accepted in strict mode. |
| DOM-005 | hard | Required-if-enabled secrets are enforced. | `postgres.enabled=true` requires `postgres.dsn`; `mongo.enabled=true` requires `mongo.uri`. | Service starts with enabled dependency and missing required secret. |
| DOM-006 | hard | Secret-like values are forbidden in YAML files. | Loader rejects YAML containing secret-like populated keys. | Secret-like config from file is accepted. |

## Acceptance Criteria
1. Namespace env keys configure all runtime fields used by service bootstrap.
2. Flat env keys outside `APP__` namespace are ignored and do not alter effective config.
3. Startup fails on parse/validate/secret-policy violations with typed errors.
4. Unknown-key behavior differs only by strict flag.
5. Security hardening for file source is enforced in non-local environments.
