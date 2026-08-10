// Proof that the middleware order Harden documents is the order Harden builds.
//
// The chain is the thing this package exists to give a service, and its order is
// the contract: every comment inside Harden explains why one position sits inside
// another. Yet the summary a reader actually navigates by is a nine-name arrow
// line above the function, and nothing relates it to the arguments below. A
// middleware inserted into newRootRouter without touching that line leaves the
// summary describing a chain the service no longer runs — and the summary is what
// a reader checks instead of counting closures.

package httpx

import (
	"go/ast"
	"go/parser"
	"go/token"
	"slices"
	"strings"
	"testing"
)

// documentedChainMarker is the arrow line's first name. The summary is an
// indented continuation of Harden's doc comment, so the scan starts at the line
// naming the outermost middleware and runs to the end of the indented block.
const documentedChainMarker = "RequestCorrelation"

// chainNameByIdentifier maps an identifier Harden passes to the name the summary
// gives that position. Only a value whose variable name is not the middleware's
// own name needs an entry.
var chainNameByIdentifier = map[string]string{
	"otelMiddleware": "OTel",
}

func TestDocumentedMiddlewareOrderMatchesHarden(t *testing.T) {
	t.Parallel()

	fileSet := token.NewFileSet()
	parsed, err := parser.ParseFile(fileSet, "harden.go", nil, parser.ParseComments)
	if err != nil {
		t.Fatalf("parse harden.go: %v", err)
	}
	harden := funcDecl(t, parsed, "Harden")

	built := builtChainOrder(t, harden)
	documented := documentedChainOrder(t, harden)
	if !slices.Equal(documented, built) {
		t.Fatalf(
			"Harden documents the chain\n\t%v\nbut builds\n\t%v",
			documented, built,
		)
	}
}

// builtChainOrder reads the chain Harden actually assembles, outermost first: the
// wrapper it returns, then newRootRouter's middlewares in argument order, then the
// subrouter they wrap, which newRootRouter takes first.
func builtChainOrder(t *testing.T, harden *ast.FuncDecl) []string {
	t.Helper()

	root := callTo(t, harden, "newRootRouter")
	if len(root.Args) < 2 {
		t.Fatalf("newRootRouter takes %d arguments, want a subrouter and at least one middleware", len(root.Args))
	}

	order := []string{chainName(t, returnedWrapper(t, harden))}
	for _, argument := range root.Args[1:] {
		order = append(order, chainName(t, argument))
	}
	return append(order, chainName(t, root.Args[0]))
}

// documentedChainOrder reads the arrow summary above Harden.
func documentedChainOrder(t *testing.T, harden *ast.FuncDecl) []string {
	t.Helper()

	if harden.Doc == nil {
		t.Fatal("Harden has no doc comment, and its middleware order is the contract")
	}
	summary := make([]string, 0)
	started := false
	for _, line := range harden.Doc.List {
		text := strings.TrimPrefix(line.Text, "//")
		if !started && !strings.Contains(text, documentedChainMarker) {
			continue
		}
		if started && !strings.HasPrefix(text, "\t") {
			break
		}
		started = true
		summary = append(summary, text)
	}
	if !started {
		t.Fatalf("Harden's doc comment has no line naming %s, so it publishes no chain order", documentedChainMarker)
	}

	order := make([]string, 0)
	for name := range strings.SplitSeq(strings.Join(summary, " "), "→") {
		if trimmed := strings.TrimSpace(name); trimmed != "" {
			order = append(order, trimmed)
		}
	}
	return order
}

// chainName is the summary's name for one value Harden passes: the identifier
// itself, or — for a middleware Harden adapts in place — the function that
// closure calls.
func chainName(t *testing.T, expression ast.Expr) string {
	t.Helper()

	switch node := expression.(type) {
	case *ast.Ident:
		if documented, aliased := chainNameByIdentifier[node.Name]; aliased {
			return documented
		}
		return node.Name
	case *ast.FuncLit:
		var called string
		ast.Inspect(node.Body, func(inner ast.Node) bool {
			if called != "" {
				return false
			}
			if call, isCall := inner.(*ast.CallExpr); isCall {
				if fun, isIdent := call.Fun.(*ast.Ident); isIdent {
					called = fun.Name
					return false
				}
			}
			return true
		})
		if called == "" {
			t.Fatal("a middleware closure in Harden calls no package-level function, so it has no name to publish")
		}
		return called
	default:
		t.Fatalf("Harden passes a %T the chain summary has no name for", expression)
		return ""
	}
}

// returnedWrapper is the call Harden returns, which wraps everything below it and
// is therefore the outermost position.
func returnedWrapper(t *testing.T, harden *ast.FuncDecl) ast.Expr {
	t.Helper()

	var wrapper ast.Expr
	ast.Inspect(harden.Body, func(node ast.Node) bool {
		returned, isReturn := node.(*ast.ReturnStmt)
		if !isReturn || len(returned.Results) == 0 {
			return true
		}
		if call, isCall := returned.Results[0].(*ast.CallExpr); isCall {
			if fun, isIdent := call.Fun.(*ast.Ident); isIdent {
				wrapper = fun
			}
		}
		return true
	})
	if wrapper == nil {
		t.Fatal("Harden returns no wrapping call, so it publishes no outermost middleware")
	}
	return wrapper
}

func callTo(t *testing.T, harden *ast.FuncDecl, name string) *ast.CallExpr {
	t.Helper()

	var found *ast.CallExpr
	ast.Inspect(harden.Body, func(node ast.Node) bool {
		call, isCall := node.(*ast.CallExpr)
		if !isCall {
			return true
		}
		if fun, isIdent := call.Fun.(*ast.Ident); isIdent && fun.Name == name {
			found = call
			return false
		}
		return true
	})
	if found == nil {
		t.Fatalf("Harden no longer calls %s, and this test reads the chain out of that call", name)
	}
	return found
}

func funcDecl(t *testing.T, parsed *ast.File, name string) *ast.FuncDecl {
	t.Helper()

	for _, declaration := range parsed.Decls {
		function, isFunc := declaration.(*ast.FuncDecl)
		if isFunc && function.Name.Name == name && function.Recv == nil {
			return function
		}
	}
	t.Fatalf("harden.go declares no function %s", name)
	return nil
}
