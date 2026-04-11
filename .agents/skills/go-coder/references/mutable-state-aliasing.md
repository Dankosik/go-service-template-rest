# Mutable State And Aliasing

## When To Load
Load this when work touches slices, maps, `[]byte`, pointer receivers, returned snapshots, cache entries, JSON nil/empty behavior, structs containing synchronization primitives, or any mutation after data crosses a boundary.

## Good/Bad Examples

Bad: caller-owned state is retained and later mutations leak in.

```go
type Cache struct {
	values map[string][]byte
}

func (c *Cache) Put(key string, value []byte) {
	c.values[key] = value
}
```

Good: copy at the ownership boundary.

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

Bad: exposing internal maps lets callers mutate package state.

```go
func (s *Settings) Labels() map[string]string {
	return s.labels
}
```

Good: return a snapshot. Remember this is a shallow clone.

```go
func (s *Settings) Labels() map[string]string {
	return maps.Clone(s.labels)
}
```

Bad: shallow-copying a type that contains a mutex.

```go
type Registry struct {
	mu    sync.Mutex
	items map[string]Item
}

func (r Registry) Get(name string) (Item, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	item, ok := r.items[name]
	return item, ok
}
```

Good: use pointer receivers when identity or synchronization state matters.

```go
func (r *Registry) Get(name string) (Item, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	item, ok := r.items[name]
	return item, ok
}
```

Bad: a clone idiom changes observable nil/empty semantics.

```go
func cloneIDs(ids []ID) []ID {
	return append([]ID(nil), ids...)
}
```

Good: use `slices.Clone` when top-level shallow clone and nilness preservation are the intended semantics.

```go
func cloneIDs(ids []ID) []ID {
	return slices.Clone(ids)
}
```

## Common False Simplifications
- Cloning the top-level slice while elements still point at mutable structs, maps, slices, or byte buffers.
- Treating `maps.Clone` or `slices.Clone` as deep copy operations.
- Copying structs with `sync.Mutex`, `sync.RWMutex`, `sync.Once`, `sync.WaitGroup`, atomics, pools, or builders.
- Returning internal maps or slices to "avoid allocation" when callers can mutate repository, cache, or config state.
- Replacing a nil-preserving clone with an append idiom that changes nil versus empty behavior.
- Adding `sync.Map` to avoid deciding ownership; it can help specific concurrent access patterns but does not fix aliasing of stored values.

## Validation Or Test Patterns
- Add two-way aliasing tests: mutate the caller's input after `Put`, and mutate the returned value after `Get`.
- Cover nil and non-nil empty slices when JSON, SQL, cache, or API shape observes the difference.
- Run `go vet` or the repository's lint target when receiver changes may copy locks.
- Run `go test -race` when mutation or synchronization changed.
- For shallow clones, add a test or comment at the boundary if nested mutable values remain intentionally shared.

## Source Links Gathered Through Exa
- [Go Slices: usage and internals](https://go.dev/doc/articles/slices_usage_and_internals.html)
- [The Go Programming Language Specification](https://go.dev/doc/go_spec.html)
- [slices package](https://pkg.go.dev/slices)
- [maps package](https://pkg.go.dev/maps)
- [bytes package](https://pkg.go.dev/bytes)
- [Go Code Review Comments](https://go.dev/wiki/CodeReviewComments)
- [Go FAQ](https://go.dev/doc/faq)
