# Chi Router Registration Hazards

## When To Read
Read this during review when a diff changes router construction order, moves `Use`, adds or reorders `Mount`, registers catch-all paths, mixes mounted subrouters with manual routes, or changes generated/manual path ownership. Focus on startup panic risk and on which router owns a method/path at runtime.

## Review Smell Patterns
- `Use(...)` is called on the same mux after the first route, `With`, `Group`, `Route`, `Mount`, `Handle`, or method registration has forced the mux handler to be built.
- Two `Mount(...)` calls claim the same path, or a manual wildcard such as `/api/*` is registered before a later mount on `/api`.
- A diff treats `Mount("/api", h)` as one exact path instead of subtree ownership through `/api`, `/api/`, and `/api/*`.
- A generated router and manual chi router both claim the same subtree, leaving route ownership dependent on registration order.
- A router constructor accepts a possibly nil handler or subrouter and passes it to `Mount` or `Route`.

## Minimal Examples

```diff
 func NewRouter() chi.Router {
   r := chi.NewRouter()
-  r.Get("/healthz", healthz)
   r.Use(requestID)
+  r.Get("/healthz", healthz)
   return r
 }
```

Review finding shape: this is a startup safety issue, not style. The smallest safe fix is to keep `Use` before the first route on that mux, or move the middleware into `With`/`Group` at the endpoint scope that needs it.

```diff
-r.Mount("/v1", publicRouter())
-r.Mount("/v1", adminRouter())
+v1 := chi.NewRouter()
+v1.Mount("/public", publicRouter())
+v1.Mount("/admin", adminRouter())
+r.Mount("/v1", v1)
```

Review finding shape: the conflict is subtree ownership. The smallest safe fix is one owner for `/v1` that delegates below it.

## Validation
- Add a constructor test that fails on startup panics:

```go
func TestNewRouterDoesNotPanic(t *testing.T) {
  defer func() {
    if got := recover(); got != nil {
      t.Fatalf("NewRouter panicked: %v", got)
    }
  }()
  _ = NewRouter()
}
```

- Add `httptest` cases for affected method/path pairs, including the mount root, trailing slash, and one child path:

```go
for _, tc := range []struct {
  method string
  path   string
  want   int
}{
  {http.MethodGet, "/v1/public/ping", http.StatusOK},
  {http.MethodGet, "/v1/admin/ping", http.StatusOK},
} {
  req := httptest.NewRequest(tc.method, tc.path, nil)
  rr := httptest.NewRecorder()
  NewRouter().ServeHTTP(rr, req)
  if rr.Code != tc.want {
    t.Fatalf("%s %s: got %d, want %d", tc.method, tc.path, rr.Code, tc.want)
  }
}
```

- Suggested validation command: `go test ./... -run 'TestNewRouter|TestRoutes'`.

## Sources Gathered With Exa
- [chi package docs on pkg.go.dev](https://pkg.go.dev/github.com/go-chi/chi/v5)
- [go-chi README router interface and examples](https://github.com/go-chi/chi/blob/master/README.md)
- [go-chi mux.go source for Use, Mount, Route, and constructor behavior](https://raw.githubusercontent.com/go-chi/chi/master/mux.go)
