# Rollout And Rollback Safety Review

## Behavior Change Thesis
When loaded for symptom `feature flag, config rollout, mixed-version compatibility, schema compatibility, canary signal, rollback behavior, or capacity-sensitive release path changed`, this file makes the model preserve safe partial rollout and rollback instead of likely mistake `assume all instances, config, data, and capacity change together`.

## When To Load
Load when a diff changes feature flags, config rollout, schema compatibility, startup behavior under new config, canary signals, metric labels needed for release comparison, progressive rollout assumptions, rollback behavior, migration sequencing, or capacity-sensitive release behavior.

Keep findings local: identify code paths that make safe rollout or rollback brittle. Hand off CI/CD gates, deployment policy, migration-runner ownership, and release governance to `go-devops-spec` or `go-design-spec` when the fix is broader than the changed code.

## Decision Rubric
- New code assumes all instances switch versions at the same time.
- New readers cannot tolerate old data; old readers cannot tolerate new data.
- Feature flag defaults fail open or enable expensive behavior when config is missing or malformed.
- Canary detection is impossible because errors/latency are not attributable to version, flag, or route at low cardinality.
- Rollback requires reversing data that may already have been written by the new version.
- Startup fails hard on optional new config and can crash-loop during partial rollout.
- A rollout increases per-request cost or fan-out without capacity guardrails.
- The change removes a fallback before the new path has soaked under real traffic.

## Imitate

Bad finding shape to copy: the changed reader assumes schema and binary advance atomically.

```go
func (s *Store) ScanUser(row *sql.Row) (User, error) {
	var user User
	if err := row.Scan(&user.ID, &user.Email, &user.NewRequiredTier); err != nil {
		return User{}, err
	}
	return user, nil
}
```

```text
[high] [go-reliability-review] internal/users/store.go:73
Issue: The changed reader requires the new column immediately, with no mixed-version or rollback tolerance.
Impact: During staged rollout or rollback, old and new binaries can disagree about the row shape, turning a deploy into a serving outage.
Suggested fix: Use an expand/contract-compatible read path, tolerate missing/default values until all versions are advanced, and keep rollback to the previous binary low-risk.
Reference: Google Cloud rollback guidance and Google SRE progressive rollout practices.
```

Good correction shape: new behavior is guarded and old data remains readable during rollout.

```go
func (s *Store) UserTier(ctx context.Context, id string) (Tier, error) {
	tier, err := s.queryOptionalTier(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return DefaultTier, nil
	}
	return tier, err
}
```

Bad finding shape to copy: malformed or missing config should not enable a risky new path by accident.

```go
enabled, _ := strconv.ParseBool(os.Getenv("ENABLE_NEW_AGGREGATOR"))
if enabled || os.Getenv("ENABLE_NEW_AGGREGATOR") == "" {
	return newAggregator.Handle(ctx, req)
}
```

Copy the review move: make invalid or missing config fail to the previous known-safe behavior, emit an operator-visible signal, and keep the old path until rollout evidence exists.

## Reject

```go
enabled, _ := strconv.ParseBool(os.Getenv("NEW_PATH"))
if enabled || os.Getenv("NEW_PATH") == "" {
	return newPath(ctx)
}
```

Reject because missing config enables the new path during partial rollout or config drift.

```go
return row.Scan(&u.ID, &u.Email, &u.NewRequiredField)
```

Reject when the deployment can include old rows, old migrations, old binaries, or rollback to a binary that cannot tolerate the new write shape.

```go
return s.newAggregator.Handle(ctx, req) // old path removed in same PR
```

Reject when rollback requires restoring deleted fallback code or reversing already-written data.

## Agent Traps
- Do not assume a deployment is all-at-once unless the repo's deployment contract says so.
- Do not turn every migration concern into DB ownership; reliability owns the rollback/mixed-version serving risk, while migration mechanics can hand off.
- Do not accept missing canary labels when the risky behavior is hidden inside aggregate error or latency metrics.
- Do not require heavy release process for a local fix if default config, compatibility reads, or guardrails solve the changed-code risk.
- Do not ignore capacity when a flag enables new fan-out, cache misses, warmup, or background jobs.

## Validation Shape
- `go test ./... -run 'Test.*(Rollback|Rollout|Canary|FeatureFlag|Config)'`
- `go test ./... -run 'Test.*(MixedVersion|Backward|Forward|Compatibility|Migration)'`
- `go test ./... -run 'Test.*(DefaultConfig|InvalidConfig|MissingConfig)'`
- `go test ./... -bench 'Benchmark.*(NewPath|Aggregator|Fanout)' -run '^$'` when the change affects request cost and benchmarks already exist.

For migrations, prefer repository-native migration tests or expand/contract validation over ad hoc review claims.
