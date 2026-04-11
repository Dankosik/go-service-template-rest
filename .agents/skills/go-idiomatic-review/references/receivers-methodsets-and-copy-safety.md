# Receivers, Method Sets, And Copy Safety

## When To Load It
Load this reference when a Go review touches method receivers, interface satisfaction, value vs pointer semantics, copying structs, `sync.Mutex`, `sync.RWMutex`, `sync.Once`, `sync.WaitGroup`, `sync.Map`, atomics, `strings.Builder`, `bytes.Buffer`, pointer-to-interface parameters, or method values/closures that capture receivers.

## Exa Source Links
- [Go specification: Method sets](https://go.dev/ref/spec#Method_sets)
- [Go Code Review Comments: Receiver Type, Copying, Pass Values](https://go.dev/wiki/CodeReviewComments)
- [sync package](https://pkg.go.dev/sync)
- [strings.Builder](https://pkg.go.dev/strings#Builder)
- [Go FAQ: methods on values or pointers](https://go.dev/doc/faq)
- [Go Wiki: MethodSets](https://go.dev/wiki/MethodSets)

## Review Cues
- A value receiver is added to a type containing a lock, atomic field, builder, buffer, or map/slice that represents protected state.
- Receiver types are mixed and that changes interface satisfaction or copy expectations.
- A method with a pointer receiver is expected to satisfy an interface with a non-addressable value.
- A struct containing copy-sensitive fields is passed by value, returned by value, embedded by value, or placed in a slice/map as a mutable object.
- A pointer to a map, slice, string, or interface appears without a nilability or mutation contract that requires it.

## Bad Review Examples
Bad review:

```text
Use pointer receivers everywhere.
```

Why it is bad: small immutable value types can be clearer with value receivers. The finding should connect receiver choice to mutation, method sets, copy safety, or identity.

Bad review:

```text
This receiver is inconsistent, please clean it up.
```

Why it is bad: inconsistent receivers are merge risk only when they affect interface satisfaction, copying, mutation visibility, or reader understanding.

Bad review:

```text
Use *sync.Mutex in the struct so copies are safe.
```

Why it is bad: pointer-to-lock can introduce nil panics and unclear shared ownership. Usually the safer fix is preventing the containing struct from being copied and using pointer receivers.

## Good Review Examples
Good finding:

```text
[critical] [go-idiomatic-review] internal/cache/cache.go:52
Issue: Put has a value receiver on Cache, which copies the embedded sync.RWMutex while the map header still aliases the same backing map.
Impact: Concurrent callers can mutate the same map under different lock copies, so the lock no longer protects the state.
Suggested fix: Change Cache methods to pointer receivers and avoid passing Cache by value after first use.
Reference: https://pkg.go.dev/sync
```

Good finding:

```text
[high] [go-idiomatic-review] internal/report/builder.go:38
Issue: ReportBuilder is copied by value after writing to its strings.Builder field.
Impact: strings.Builder is copy-sensitive after first use; the copy can panic or corrupt builder assumptions when both values are used.
Suggested fix: Pass *ReportBuilder or construct a fresh builder per copy path.
Reference: https://pkg.go.dev/strings#Builder
```

Good finding:

```text
[medium] [go-idiomatic-review] internal/plugins/registry.go:91
Issue: Register stores a value of Plugin, but Plugin only implements io.Closer through a pointer receiver.
Impact: The stored value no longer satisfies the intended interface at the use site, so cleanup cannot be called through the registry contract.
Suggested fix: Store *Plugin or change the receiver only if close semantics do not mutate or depend on identity.
Reference: https://go.dev/ref/spec#Method_sets
```

## Real Merge-Risk Impact
- Lock copies can create real data races while tests appear green.
- Copying builders and buffers can produce aliasing, panics, or surprising output.
- Method-set mistakes can silently remove interface satisfaction at package boundaries.
- Value receivers on identity-bearing types can mutate a copy and leave the original unchanged.
- Pointer-to-interface APIs can hide nil and method-set confusion while adding no useful ownership semantics.

## Smallest Safe Correction
- Use pointer receivers for methods that mutate, expose identity, or protect shared state.
- Keep receiver choice consistent when any method needs pointer semantics.
- Prevent value copying by changing function signatures, map/slice storage, or constructor returns to use pointers where identity matters.
- Keep `sync` fields as values inside the struct unless a documented shared-lock indirection is required.
- Replace `*interface{}` or `*io.Reader` parameters with the interface value unless nilability of the interface variable itself is the explicit contract.

## Validation Ideas
- Run `go vet ./...` to catch copylock patterns.
- Add compile-time interface assertions for intended `T` vs `*T` satisfaction.
- Add tests that mutate through the method and assert the original object changed.
- Add race tests only when concurrency behavior is actually in scope; otherwise hand off.

## Handoffs
- Hand off lock-order, goroutine safety, and race-depth analysis to concurrency review.
- Hand off public API receiver/signature changes to design/API review.
- Hand off performance claims about receiver size or allocation to performance review.
