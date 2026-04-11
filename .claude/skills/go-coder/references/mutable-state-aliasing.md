# Mutable State And Aliasing

## Behavior Change Thesis
When loaded for mutable-boundary pressure, this file makes the model clone or preserve pointer identity at the ownership boundary instead of leaking aliases, copying synchronization state, or changing observable nil/empty semantics.

## When To Load
Load this when work touches slices, maps, `[]byte`, pointer receivers, returned snapshots, cache entries, JSON nil/empty behavior, structs with synchronization fields, or mutation after data crosses a boundary.

## Decision Rubric
- Decide who owns mutable data after each call returns.
- Clone at the ownership boundary, not repeatedly inside unrelated code.
- Treat `slices.Clone`, `maps.Clone`, and `bytes.Clone` as shallow top-level clones.
- Preserve nil versus non-nil empty shape when JSON, SQL, cache, or API behavior observes it.
- Use pointer receivers when identity matters or copying would duplicate locks, atomics, pools, builders, or large mutable state.
- Avoid `sync.Map` as a shortcut for unresolved ownership; it does not protect the mutability of stored values.

## Imitate
Copy both on write and on read when cache callers must not share buffers.

```go
type Cache struct {
	values map[string][]byte
}

func (c *Cache) Put(key string, value []byte) {
	c.values[key] = bytes.Clone(value)
}

func (c *Cache) Get(key string) ([]byte, bool) {
	value, ok := c.values[key]
	if !ok {
		return nil, false
	}
	return bytes.Clone(value), true
}
```

Use pointer receivers for types with synchronization state.

```go
type Registry struct {
	mu    sync.Mutex
	items map[string]Item
}

func (r *Registry) Get(name string) (Item, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	item, ok := r.items[name]
	return item, ok
}
```

Use `slices.Clone` when nilness preservation and shallow cloning are intended.

```go
func cloneIDs(ids []ID) []ID {
	return slices.Clone(ids)
}
```

## Reject
Reject retaining caller-owned mutable data.

```go
func (c *Cache) Put(key string, value []byte) {
	c.values[key] = value
}
```

Reject returning internal maps as snapshots.

```go
func (s *Settings) Labels() map[string]string {
	return s.labels
}
```

Reject value receivers that copy locks.

```go
func (r Registry) Get(name string) (Item, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.items[name], true
}
```

Reject clone idioms that change observable shape.

```go
func cloneIDs(ids []ID) []ID {
	return append([]ID(nil), ids...)
}
```

This collapses a non-nil empty slice to `nil`.

## Agent Traps
- Cloning the top-level slice while elements still point at mutable structs, maps, slices, or buffers.
- Returning internal maps or slices to "avoid allocation" when callers can mutate config, repository, or cache state.
- Copying structs with `sync.Mutex`, `sync.RWMutex`, `sync.Once`, `sync.WaitGroup`, atomics, pools, or builders.
- Replacing a nil-preserving clone with an append idiom that changes nil versus empty behavior.
- Capturing loop variables or receiver copies in callbacks or goroutines while trying to simplify code.

## Validation Shape
- Add two-way aliasing tests: mutate caller input after `Put`, and mutate returned data after `Get`.
- Cover nil and non-nil empty slices or maps when shape is observable.
- Run `go vet` or the repository lint target when receiver changes might copy locks.
- Run `go test -race` when mutation or synchronization changed.
- For intentional shallow clones, add a focused test or boundary comment if nested mutable values remain shared by contract.
