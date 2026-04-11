# Helper Extraction And Package Ownership

## When To Load
Load this when an implementation is about to extract helpers, introduce interfaces, move code across packages, export symbols for tests, or create package-level ownership for repeated policy.

## Good/Bad Examples

Bad: a generic utility package erases ownership.

```go
package util

func Normalize(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}
```

Good: keep stable policy near the package that owns its meaning.

```go
package users

func canonicalEmail(raw string) (string, error) {
	email := strings.ToLower(strings.TrimSpace(raw))
	if email == "" || !strings.Contains(email, "@") {
		return "", ErrInvalidEmail
	}
	return email, nil
}
```

Bad: exporting just to test the helper from another package.

```go
func NormalizeEmailForTest(raw string) string {
	return strings.ToLower(strings.TrimSpace(raw))
}
```

Good: test through the exported behavior when practical; use a same-package test only when the unexported policy is the real unit.

```go
func TestCreateUserCanonicalizesEmail(t *testing.T) {
	got, err := CreateUser(context.Background(), CreateUserInput{
		Email: "  Ada@Example.COM ",
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.Email != "ada@example.com" {
		t.Fatalf("CreateUser().Email = %q, want %q", got.Email, "ada@example.com")
	}
}
```

Bad: a provider package exports an interface before any consumer has a substitution need.

```go
package users

type Repository interface {
	Create(context.Context, User) (User, error)
	Update(context.Context, User) error
	Delete(context.Context, UserID) error
	Find(context.Context, UserID) (User, error)
	List(context.Context, ListUsersFilter) ([]User, error)
}
```

Good: the consuming package defines the small seam it needs.

```go
package handlers

type userFinder interface {
	Find(context.Context, users.UserID) (users.User, error)
}

type UserHandler struct {
	users userFinder
}
```

Bad: a helper that hides the main control flow and is used once.

```go
func shouldCreate(input CreateInput) bool {
	return input.Email != "" && input.Name != "" && !input.Disabled
}
```

Good: inline simple one-use checks when they make the operation easier to read.

```go
if input.Email == "" || input.Name == "" || input.Disabled {
	return User{}, ErrInvalidCreateInput
}
```

## Common False Simplifications
- Extracting a helper because a function looks long, even though the helper forces readers to jump away from the main state transition.
- Merging related-looking policies behind booleans, callbacks, mode strings, or option bags.
- Creating `util`, `common`, `shared`, or `helpers` packages before identifying a single owner and stable contract.
- Exporting functions to satisfy tests instead of testing through behavior or using a focused same-package test.
- Introducing interfaces at the provider side before a consumer needs substitution.
- Moving a local package policy into `internal/` too early; `internal/` controls import scope, but it does not by itself create clearer ownership.

## Validation Or Test Patterns
- Search for near-duplicates with `rg` before extracting: repeated stable policy can justify a same-package owner file.
- Verify the helper name states policy, not mechanics: `canonicalEmail` is stronger than `normalizeString`.
- Prefer table tests for extracted policy only when several distinct cases share the same setup and assertions.
- Check package imports after a move. A good boundary should not create cycles or force unrelated packages to import a broader dependency.
- Run tests for both the old call sites and the package that now owns the helper.

## Source Links Gathered Through Exa
- [Go Code Review Comments](https://go.dev/wiki/CodeReviewComments)
- [Effective Go](https://go.dev/doc/effective_go) - useful for package names and interface idioms, with the official caveat that it is not actively updated.
- [The Go Programming Language Specification](https://go.dev/doc/go_spec.html)
- [Go FAQ](https://go.dev/doc/faq)
