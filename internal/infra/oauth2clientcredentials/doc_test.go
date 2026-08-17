package oauth2clientcredentials

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
	"testing"

	"github.com/example/go-service-template-rest/internal/packagetest"
)

func TestPackageContract(t *testing.T) {
	t.Parallel()
	allowed := map[string]bool{
		"Client": true, "Config": true, "FailureClass": true, "FailureClassOf": true,
		"GRPC": true, "HTTPClient": true, "New": true, "NewGRPC": true, "NewHTTPClient": true,
		"FailureInvalidConfiguration": true, "FailureEndpointTrust": true, "FailureCallerCanceled": true,
		"FailureProviderTimeout": true, "FailureProviderUnavailable": true, "FailureClientRejected": true,
		"FailureGrantRejected": true, "FailureUnsupportedResponse": true, "FailureTokenUnusable": true,
		"FailureDownstreamUnauthenticated": true, "FailureDownstreamForbidden": true,
	}
	packagetest.EachGoFile(t, ".", parser.Mode(0), func(filename string, file *ast.File, _ *token.FileSet) {
		if strings.HasSuffix(filename, "_test.go") {
			return
		}
		for _, declaration := range file.Decls {
			for _, name := range exportedDeclarationNames(declaration) {
				if !allowed[name] {
					t.Fatalf("unexpected package export %q in %s", name, filename)
				}
			}
		}
	})
}

func exportedDeclarationNames(declaration ast.Decl) []string {
	switch declaration := declaration.(type) {
	case *ast.FuncDecl:
		if declaration.Recv == nil && declaration.Name.IsExported() {
			return []string{declaration.Name.Name}
		}
	case *ast.GenDecl:
		var names []string
		for _, spec := range declaration.Specs {
			switch spec := spec.(type) {
			case *ast.TypeSpec:
				if spec.Name.IsExported() {
					names = append(names, spec.Name.Name)
				}
			case *ast.ValueSpec:
				for _, name := range spec.Names {
					if name.IsExported() {
						names = append(names, name.Name)
					}
				}
			}
		}
		return names
	}
	return nil
}
