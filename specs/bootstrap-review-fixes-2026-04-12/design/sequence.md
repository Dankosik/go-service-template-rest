# Runtime And Implementation Sequence

## Runtime Behavior After Fix

### Degraded Dependency Startup

1. Bootstrap initializes Redis or Mongo dependency admission as it does today.
2. If a probe failure should abort startup, the existing rejection path records failure telemetry and returns an error.
3. If a probe failure is degraded-but-serving:
   - call the shared degraded-startup helper,
   - set the effective degraded mode in `startup_dependency_status` to `1`,
   - log `startup_dependency_degraded`,
   - return `nil` so startup can continue.
4. Redis cache uses mode `feature_off`.
5. Mongo degraded path uses mode `degraded_read_only_or_stale`.

### Network Policy Configuration

1. Bootstrap builds the typed `internal/config.Config` snapshot.
2. Bootstrap loads `NETWORK_*` from the process environment as a separate operator network-policy channel.
3. If network policy parsing fails:
   - preserve the original parse/config cause in the returned error,
   - keep wrapping with `config.ErrDependencyInit`,
   - preferred: include `policy.class` and `reason.class` in structured startup-blocked logs through production use of `networkPolicyErrorLabels`.
4. If network policy parses successfully, ingress and egress enforcement continue as today.

## Implementation Order

1. Add the degraded dependency startup helper and update Redis/Mongo degraded paths.
2. Add Mongo degraded metric coverage and keep Redis coverage passing.
3. Simplify `dependencyInitFailure`.
4. Document the `NETWORK_*` operator-policy channel.
5. Resolve `networkPolicyErrorLabels` ownership by production logging use or test-only relocation.
6. Run package and config validation commands.

## Failure Points To Preserve

- Public ingress remains fail-closed when declaration is missing for non-local wildcard binds.
- Expired ingress/egress exceptions still reject startup or runtime readiness as today.
- Egress target denial still rejects startup dependency admission.
- Context cancellation, deadline exceeded, and low remaining startup budget still abort degraded dependencies instead of continuing silently.
- Error wrapping remains inspectable through `errors.Is(err, config.ErrDependencyInit)`.
