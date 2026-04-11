# Receivers, Method Sets, And Copy Safety

## Behavior Change Thesis
When loaded for receiver or copy-safety symptoms, this file makes the model tie findings to mutation, identity, method-set reachability, and must-not-copy state instead of likely mistake "use pointer receivers everywhere" or "make receivers consistent."

## When To Load
Load when a Go review touches method receivers, interface satisfaction, value vs pointer semantics, copying structs, `sync.Mutex`, `sync.RWMutex`, `sync.Once`, `sync.WaitGroup`, `sync.Map`, atomics, `strings.Builder`, `bytes.Buffer`, pointer-to-interface parameters, or method values/closures that capture receivers.

## Decision Rubric
- Use pointer receivers when a method mutates state, depends on identity, protects shared state, or must avoid copying copy-sensitive fields.
- Value receivers are fine for small immutable value types and methods that intentionally work on a copy.
- Mixed receiver types are a finding only when they confuse mutation visibility, method-set satisfaction, or copy expectations.
- Do not copy a type after first use if it contains locks, atomics, builders, buffers, wait groups, or other must-not-copy fields.
- Keep `sync` fields as values inside the struct unless a documented shared-lock indirection is required; pointer-to-lock often adds nil and ownership confusion.
- Check whether `T` or `*T` satisfies the interface actually used at the boundary; non-addressable values cannot call pointer receiver methods.
- Treat pointer-to-interface, pointer-to-map, and pointer-to-slice parameters as smells unless nilability or mutation of the header itself is the explicit contract.
- Inspect method values and closures when they capture a receiver copy that may diverge from the caller's object.

## Imitate
```text
[critical] [go-idiomatic-review] internal/cache/cache.go:52
Issue: Put has a value receiver on Cache, which copies the embedded sync.RWMutex while the map header still aliases the same backing map.
Impact: Concurrent callers can mutate the same map under different lock copies, so the lock no longer protects the state.
Suggested fix: Change Cache methods to pointer receivers and avoid passing Cache by value after first use.
Reference: sync must-not-copy contract
```

Copy the two-part proof: the lock copy and aliased map together create the defect.

```text
[high] [go-idiomatic-review] internal/report/builder.go:38
Issue: ReportBuilder is copied by value after writing to its strings.Builder field.
Impact: strings.Builder is copy-sensitive after first use; the copy can panic or corrupt builder assumptions when both values are used.
Suggested fix: Pass *ReportBuilder or construct a fresh builder per copy path.
Reference: strings.Builder copy contract
```

Copy the after-first-use condition: not every builder-containing value is already broken at declaration time.

```text
[medium] [go-idiomatic-review] internal/plugins/registry.go:91
Issue: Register stores a value of Plugin, but Plugin only implements io.Closer through a pointer receiver.
Impact: The stored value no longer satisfies the intended cleanup interface at the use site, so cleanup cannot be called through the registry contract.
Suggested fix: Store *Plugin or change the receiver only if close semantics do not mutate or depend on identity.
Reference: method-set rule
```

Copy the method-set boundary: the finding proves which type no longer satisfies which interface.

## Reject
```text
Use pointer receivers everywhere.
```

Reject because small immutable values can be clearer with value receivers. The finding must connect receiver choice to mutation, identity, method sets, or copy safety.

```text
This receiver is inconsistent, please clean it up.
```

Reject unless inconsistency affects interface satisfaction, copying, mutation visibility, or reader understanding of identity.

```text
Use *sync.Mutex in the struct so copies are safe.
```

Reject because pointer-to-lock usually introduces nil panics and unclear shared ownership. Prevent copying the containing type instead.

## Agent Traps
- Do not treat every value receiver as a bug. Check whether the type has identity, mutation, or copy-sensitive fields.
- Do not miss map/slice aliasing inside copied structs; copying the header can still share underlying mutable state.
- Do not use this file for deep race or lock-order analysis; hand off to concurrency review when needed.
- Do not recommend receiver changes on exported methods without considering public API compatibility.
- Do not flag pointer-to-map/slice/interface only because it looks odd; name the nilability or header-mutation confusion it creates.

## Validation Shape
- Run `go vet ./...` when copylock patterns are in scope.
- Add compile-time interface assertions for intended `T` vs `*T` satisfaction when that is the contract.
- Add tests that mutate through the method and assert the original object changed.
- Use `-race` only when the actual finding overlaps concurrent behavior; otherwise hand off rather than inflating validation.

## Handoffs
- Hand off lock-order, goroutine safety, and race-depth analysis to concurrency review.
- Hand off public API receiver/signature changes to design/API review.
- Hand off performance claims about receiver size or allocation to performance review.
