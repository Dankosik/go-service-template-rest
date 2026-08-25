package failure_test

import (
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"net"
	"slices"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/example/go-service-template-rest/internal/failure"
)

func TestCodesAreStableAndTransportNeutral(t *testing.T) {
	t.Parallel()

	want := []failure.Code{
		"bad_request",
		"unauthorized",
		"forbidden",
		"not_found",
		"method_not_allowed",
		"already_exists",
		"request_entity_too_large",
		// profile:authn-bearer:start
		"request_header_fields_too_large",
		// profile:authn-bearer:end
		"unprocessable_content",
		"too_many_requests",
		// profile:http-idempotency-postgres:start
		"idempotency_key_mismatch",
		"idempotency_unavailable",
		"idempotency_outcome_unknown",
		// profile:http-idempotency-postgres:end
		"internal_error",
		"service_unavailable",
		"gateway_timeout",
	}
	// AllCodes rather than a second hand-built list: the wire strings above are
	// the pin, and walking the published enumeration means this test also fails
	// when AllCodes drifts from the constants every consumer resolves against.
	got := failure.AllCodes()
	if !slices.Equal(got, want) {
		t.Fatalf("published codes = %v, want %v", got, want)
	}
	if slices.Contains(got, "conflict") {
		t.Fatal("generic conflict became a transport-neutral domain identity")
	}
}

func TestAllCodesEnumeratesEveryDeclaredConstant(t *testing.T) {
	t.Parallel()

	syntax, err := parser.ParseFile(token.NewFileSet(), "failure.go", nil, 0)
	if err != nil {
		t.Fatalf("parse failure.go: %v", err)
	}

	declared := 0
	for _, declaration := range syntax.Decls {
		constants, ok := declaration.(*ast.GenDecl)
		if !ok || constants.Tok != token.CONST {
			continue
		}
		for _, specification := range constants.Specs {
			values, ok := specification.(*ast.ValueSpec)
			if !ok {
				continue
			}
			for _, name := range values.Names {
				if strings.HasPrefix(name.Name, "Code") {
					declared++
				}
			}
		}
	}

	if enumerated := len(failure.AllCodes()); enumerated != declared {
		t.Fatalf("AllCodes() enumerates %d codes, failure.go declares %d", enumerated, declared)
	}
}

// TestClassChainNamesWrappedTypesAndNoText pins both halves of the bargain the
// transports rely on: enough to route on, nothing that could be a secret.
func TestClassChainNamesWrappedTypesAndNoText(t *testing.T) {
	t.Parallel()

	const secret = "password=class-chain-secret"

	t.Run("names the type under the wrapper", func(t *testing.T) {
		t.Parallel()

		// The shape every repository failure has: fmt.Errorf over a typed error
		// from a dependency. %T alone would report only the wrapper.
		got := failure.ClassChain(fmt.Errorf("store article: %w", &net.OpError{Err: errors.New(secret)}))
		if !strings.Contains(got, "*net.OpError") {
			t.Fatalf("ClassChain() = %q, want the wrapped dependency type", got)
		}
		if strings.Contains(got, secret) {
			t.Fatalf("ClassChain() = %q, want no message text", got)
		}
	})

	t.Run("bounds a pathological chain", func(t *testing.T) {
		t.Parallel()

		err := errors.New("root")
		for range 50 {
			err = fmt.Errorf("layer: %w", err)
		}
		got := failure.ClassChain(err)
		if !strings.HasSuffix(got, "-> ...") {
			t.Fatalf("ClassChain() = %q, want a truncated chain", got)
		}
	})

	t.Run("nil is not a panic", func(t *testing.T) {
		t.Parallel()

		if got := failure.ClassChain(nil); got != "nil" {
			t.Fatalf("ClassChain(nil) = %q, want %q", got, "nil")
		}
	})
}

// TestOpNamesTheStepWithoutPublishingText pins what Op buys over fmt.Errorf: the
// step reaches the record, and nothing the error carries does.
func TestOpNamesTheStepWithoutPublishingText(t *testing.T) {
	t.Parallel()

	const secret = "password=op-secret"

	t.Run("names each step and still names the dependency", func(t *testing.T) {
		t.Parallel()

		err := failure.Op("create article", failure.Op("store row", &net.OpError{Err: errors.New(secret)}))
		got := failure.ClassChain(err)
		if got != "create article -> store row -> *net.OpError -> *errors.errorString" {
			t.Fatalf("ClassChain() = %q, want both steps then the dependency type", got)
		}
		if strings.Contains(got, secret) {
			t.Fatalf("ClassChain() = %q, want no message text", got)
		}
	})

	// The reason chainLink asserts on one layer rather than calling errors.As: As
	// searches the rest of the chain, so every fmt.Errorf layer above an Op would
	// render as that Op's name.
	t.Run("an outer wrapper does not inherit an inner name", func(t *testing.T) {
		t.Parallel()

		err := fmt.Errorf("outer: %w", failure.Op("inner step", errors.New("root")))
		got := failure.ClassChain(err)
		if got != "*fmt.wrapError -> inner step -> *errors.errorString" {
			t.Fatalf("ClassChain() = %q, want the wrapper rendered as its type", got)
		}
	})

	t.Run("preserves the identity callers match on", func(t *testing.T) {
		t.Parallel()

		sentinel := errors.New("sentinel")
		err := failure.Op("create article", fmt.Errorf("store row: %w", sentinel))
		if !errors.Is(err, sentinel) {
			t.Fatal("Op() broke errors.Is on the wrapped sentinel")
		}
		if want := "create article: store row: sentinel"; err.Error() != want {
			t.Fatalf("Error() = %q, want %q", err.Error(), want)
		}
	})

	t.Run("adds nothing when there is nothing to name", func(t *testing.T) {
		t.Parallel()

		if got := failure.Op("create article", nil); got != nil {
			t.Fatalf("Op(_, nil) = %v, want nil", got)
		}
		// Asserted through the chain rather than by comparing error values: a
		// blank name must add no layer, and the rendered chain is what says so.
		if got := failure.ClassChain(failure.Op("   ", errors.New("root"))); got != "*errors.errorString" {
			t.Fatalf("ClassChain(Op(blank, err)) = %q, want the error unwrapped and unchanged", got)
		}
	})

	// classChainDepth bounds the number of links; without this the text in one
	// link would still be unbounded.
	t.Run("bounds one name without splitting a rune", func(t *testing.T) {
		t.Parallel()

		got := failure.ClassChain(failure.Op(strings.Repeat("щ", 100), errors.New("root")))
		name, _, found := strings.Cut(got, " -> ")
		if !found {
			t.Fatalf("ClassChain() = %q, want a chain", got)
		}
		if !strings.HasSuffix(name, "…") || utf8.RuneCountInString(name) != 65 {
			t.Fatalf("rendered name = %q, want 64 runes and an ellipsis", name)
		}
		if !utf8.ValidString(name) {
			t.Fatalf("rendered name = %q, want valid UTF-8", name)
		}
	})

	t.Run("keeps a multibyte name shorter than the rune limit", func(t *testing.T) {
		t.Parallel()

		name := strings.Repeat("界", 22) // 66 bytes, 22 runes.
		got := failure.ClassChain(failure.Op(name, errors.New("root")))
		if want := name + " -> *errors.errorString"; got != want {
			t.Fatalf("ClassChain() = %q, want %q", got, want)
		}
	})
}

func TestClassifySkipsNilAndUsesFirstMatch(t *testing.T) {
	t.Parallel()

	target := errors.New("target")
	broad := func(error) (failure.Classification, bool) {
		return failure.Classification{Code: failure.CodeBadRequest}, true
	}
	nonMatching := func(error) (failure.Classification, bool) {
		return failure.Classification{}, false
	}
	specific := func(err error) (failure.Classification, bool) {
		if !errors.Is(err, target) {
			return failure.Classification{}, false
		}
		return failure.Classification{Code: failure.CodeAlreadyExists, Detail: "already exists"}, true
	}

	got, ok := failure.Classify(target, []failure.Mapper{nil, nonMatching, specific, broad})
	if !ok {
		t.Fatal("Classify() reported no match")
	}
	want := failure.Classification{Code: failure.CodeAlreadyExists, Detail: "already exists"}
	if got != want {
		t.Fatalf("Classify() = %+v, want %+v", got, want)
	}
}

func TestClassifyRefusesUnknownError(t *testing.T) {
	t.Parallel()

	got, ok := failure.Classify(errors.New("unknown"), []failure.Mapper{nil})
	if ok || got != (failure.Classification{}) {
		t.Fatalf("Classify() = (%+v, %t), want (zero, false)", got, ok)
	}
}

func TestClassifyRefusesUnpublishedClassification(t *testing.T) {
	t.Parallel()

	invalid := func(error) (failure.Classification, bool) {
		return failure.Classification{Code: "not_published", Detail: "must not escape"}, true
	}
	broad := func(error) (failure.Classification, bool) {
		return failure.Classification{Code: failure.CodeInternalError}, true
	}
	got, ok := failure.Classify(errors.New("unknown"), []failure.Mapper{invalid, broad})
	if ok || got != (failure.Classification{}) {
		t.Fatalf("Classify() = (%+v, %t), want unpublished code rejected", got, ok)
	}
}
