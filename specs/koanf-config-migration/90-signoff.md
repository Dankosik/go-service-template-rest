# 90 Signoff

## Final Decisions
1. Runtime config contract is namespace-only: `APP__<DOMAIN>__<FIELD>`.
2. Flat non-canonical env key support is removed from spec and implementation.
3. Loader stages are: defaults, file, env, parse, validate.
4. Test plan and docs are aligned to namespace-only behavior.

## Acceptance Checklist
- [x] Domain invariants updated.
- [x] Architecture updated.
- [x] Data/cache contract updated.
- [x] Security/observability/devops updated.
- [x] Reliability policy updated.
- [x] Implementation plan updated.
- [x] Test plan updated.
- [x] Open questions closed.
