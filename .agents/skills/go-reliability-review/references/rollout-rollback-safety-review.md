# Rollout And Rollback Safety Review

## When To Load
Load this reference when a diff changes feature flags, config rollout, schema compatibility, startup behavior under new config, canary signals, metric labels needed for release comparison, progressive rollout assumptions, rollback behavior, migration sequencing, or capacity-sensitive release behavior.

Keep findings local: identify code paths that make safe rollout or rollback brittle. Hand off CI/CD gates, deployment policy, migration-runner ownership, and release governance to `go-devops-spec` or `go-design-spec` when the fix is broader than the changed code.

## Review Smells
- New code assumes all instances switch versions at the same time.
- New readers cannot tolerate old data; old readers cannot tolerate new data.
- Feature flag defaults fail open or enable expensive behavior when config is missing or malformed.
- Canary detection is impossible because errors/latency are not attributable to version, flag, or route at low cardinality.
- Rollback requires reversing data that may already have been written by the new version.
- Startup fails hard on optional new config and can crash-loop during partial rollout.
- A rollout increases per-request cost or fan-out without capacity guardrails.
- The change removes a fallback before the new path has soaked under real traffic.

## Failure Modes
- A bad release affects most traffic before operators can see the signal.
- Rollback cannot return to the last known-good binary because data or config is no longer compatible.
- A config typo or partial deployment bricks all new instances.
- A canary hides errors in aggregate metrics until the rollout is already broad.
- A capacity-sensitive change triggers cascading failure during rollout or warmup.

## Review Examples

Bad: the new code requires the new schema immediately.

```go
func (s *Store) ScanUser(row *sql.Row) (User, error) {
	var user User
	if err := row.Scan(&user.ID, &user.Email, &user.NewRequiredTier); err != nil {
		return User{}, err
	}
	return user, nil
}
```

Review finding shape:

```text
[high] [go-reliability-review] internal/users/store.go:73
Issue: The changed reader requires the new column immediately, with no mixed-version or rollback tolerance.
Impact: During staged rollout or rollback, old and new binaries can disagree about the row shape, turning a deploy into a serving outage.
Suggested fix: Use an expand/contract-compatible read path, tolerate missing/default values until all versions are advanced, and keep rollback to the previous binary low-risk.
Reference: Google Cloud rollback guidance and Google SRE progressive rollout practices.
```

Good: new behavior is guarded and old data remains readable during rollout.

```go
func (s *Store) UserTier(ctx context.Context, id string) (Tier, error) {
	tier, err := s.queryOptionalTier(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return DefaultTier, nil
	}
	return tier, err
}
```

Bad: config parse failure enables the new expensive path.

```go
enabled, _ := strconv.ParseBool(os.Getenv("ENABLE_NEW_AGGREGATOR"))
if enabled || os.Getenv("ENABLE_NEW_AGGREGATOR") == "" {
	return newAggregator.Handle(ctx, req)
}
```

Smallest safe local fix: make invalid or missing config fail to the previous known-safe behavior, emit an operator-visible signal, and keep the old path until rollout evidence exists.

## Smallest Safe Fix
- Make missing or malformed new config fail to the previous safe behavior unless an approved policy says otherwise.
- Keep new writes backward-compatible until rollback to the previous binary is no longer required.
- Add local compatibility handling for old data, new data, and partial rollout.
- Preserve the old path behind a flag until canary and rollback evidence exists.
- Emit low-cardinality version/flag/path signals needed to compare canary behavior.
- Add capacity guardrails before enabling a new expensive dependency, fan-out, cache, or background job.
- Escalate when the safe fix requires migration sequencing, rollout orchestration, or deployment policy.

## Validation Commands
- `go test ./... -run 'Test.*(Rollback|Rollout|Canary|FeatureFlag|Config)'`
- `go test ./... -run 'Test.*(MixedVersion|Backward|Forward|Compatibility|Migration)'`
- `go test ./... -run 'Test.*(DefaultConfig|InvalidConfig|MissingConfig)'`
- `go test ./... -bench 'Benchmark.*(NewPath|Aggregator|Fanout)' -run '^$'` when the change affects request cost and benchmarks already exist.

For migrations, prefer repository-native migration tests or expand/contract validation over ad hoc review claims.

## Exa Source Links
- Google Cloud reliable releases and rollbacks: https://cloudplatform.googleblog.com/2017/03/reliable-releases-and-rollbacks-CRE-life-lessons.html
- Google SRE Production Services Best Practices, progressive rollouts and rollback-first response: https://sre.google/sre-book/service-best-practices/
- Google SRE Launch Coordination Checklist: https://sre.google/sre-book/launch-checklist/
- Azure Deployment Stamps pattern, including independent stamp updates and deployment rings: https://learn.microsoft.com/en-us/azure/architecture/patterns/deployment-stamp
- Google SRE Addressing Cascading Failures, process updates and new rollouts as triggers: https://sre.google/sre-book/addressing-cascading-failures/

