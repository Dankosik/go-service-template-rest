//go:build ignore

package main

import (
	"fmt"
	"go/ast"
	"os"
)

func main() {
	exitCode, err := runConstructorCheck(os.Args[1:])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
	os.Exit(exitCode)
}

func runConstructorCheck(arguments []string) (int, error) {
	if len(arguments) > 0 && arguments[0] == "--" {
		arguments = arguments[1:]
	}
	if len(arguments) != 5 {
		return 2, fmt.Errorf("usage: integration-record-constructor-check FILE IMPORT_SUFFIX EXPECTED FORBIDDEN AUTH")
	}

	filename, importSuffix, expected, forbidden := arguments[0], arguments[1], arguments[2], arguments[3]
	auth, ok := constructorAuthModes[arguments[4]]
	if !ok {
		return 2, fmt.Errorf("unknown AUTH %q", arguments[4])
	}
	parsed, err := parseFile(filename)
	if err != nil {
		return 1, err
	}
	if err := checkConstructorAST(parsed, importSuffix, expected, forbidden, auth); err != nil {
		return 1, fmt.Errorf("%s: %w", filename, err)
	}
	return 0, nil
}

// constructorAuth is what one AUTH mode requires of the adapter constructor.
type constructorAuth struct {
	// oauth requires an OAuth client-credentials client between the transport
	// and the generated client.
	oauth bool
	// doer is the identifier the generated client must be built on.
	doer string
	// bindings is how many auth, authConfig, and doer bindings are required.
	bindings int
}

var constructorAuthModes = map[string]constructorAuth{
	"none":                      {doer: "transport"},
	"oauth2-client-credentials": {oauth: true, doer: "doer", bindings: 1},
}

// checkConstructorAST matches the expected constructor shape, not equivalent
// data flow in every form. Refactoring the adapter may require updating this check.
func checkConstructorAST(parsed *ast.File, importSuffix, expected, forbidden string, auth constructorAuth) error {
	alias := usableImportAlias(parsed, importSuffix)
	if alias == "" {
		return fmt.Errorf("missing usable import ending in %s", importSuffix)
	}

	expectedAssignments := 0
	forbiddenCalls := countForbiddenCalls(parsed, alias, forbidden)
	generatedBindings := 0
	authBindings := 0
	doerBindings := 0
	authConfigBindings := 0
	returnedClients := 0
	openapiAlias := usableImportAlias(parsed, "/internal/openapi")
	oauthAlias := usableImportAlias(parsed, "/internal/infra/oauth2clientcredentials")
	for _, declaration := range parsed.Decls {
		function, ok := declaration.(*ast.FuncDecl)
		if !ok || function.Recv != nil || function.Name.Name != "New" || function.Body == nil {
			continue
		}
		for _, statement := range function.Body.List {
			assignment, ok := statement.(*ast.AssignStmt)
			if !ok || len(assignment.Lhs) == 0 || len(assignment.Rhs) != 1 {
				continue
			}
			assigned, ok := assignment.Lhs[0].(*ast.Ident)
			if !ok {
				continue
			}
			if auth.oauth && assigned.Name == "doer" {
				if source, ok := assignment.Rhs[0].(*ast.Ident); ok && source.Name == "authenticated" {
					doerBindings++
				}
				continue
			}
			call, ok := assignment.Rhs[0].(*ast.CallExpr)
			if !ok {
				continue
			}
			if assigned.Name == "transport" && selectorNamed(call.Fun, alias, expected) && validArguments(expected, call.Args) {
				expectedAssignments++
			}

			if assigned.Name == "generated" && openapiAlias != "" && generatedClientCall(call, openapiAlias, auth.doer) {
				generatedBindings++
			}
			if auth.oauth && assigned.Name == "authenticated" && oauthHTTPCall(call) {
				authBindings++
			}
			if auth.oauth && assigned.Name == "auth" && oauthConfigMapping(call, oauthAlias) {
				authConfigBindings++
			}
		}
		for _, statement := range function.Body.List {
			returned, ok := statement.(*ast.ReturnStmt)
			if !ok || len(returned.Results) != 2 || !identifierIs(returned.Results[1], "nil") {
				continue
			}
			fields := returnedClientFields(returned.Results[0])
			if fields["generated"] != "generated" || fields["transport"] != "transport" {
				continue
			}
			if auth.oauth && fields["auth"] != "auth" {
				continue
			}
			returnedClients++
		}
	}

	if expectedAssignments != 1 || forbiddenCalls != 0 || generatedBindings != 1 || returnedClients != 1 ||
		authBindings != auth.bindings || doerBindings != auth.bindings || authConfigBindings != auth.bindings {
		return fmt.Errorf("constructor=%d forbidden=%d generated=%d returned=%d auth=%d authConfig=%d doer=%d",
			expectedAssignments, forbiddenCalls, generatedBindings, returnedClients, authBindings, authConfigBindings, doerBindings)
	}
	return nil
}

func countForbiddenCalls(file *ast.File, alias, forbidden string) int {
	count := 0
	ast.Inspect(file, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		if selectorNamed(call.Fun, alias, forbidden) {
			count++
		}
		return true
	})
	return count
}

func generatedClientCall(call *ast.CallExpr, openapiAlias, wantDoer string) bool {
	if !selectorNamed(call.Fun, openapiAlias, "NewClient") || len(call.Args) != 2 {
		return false
	}
	baseURL, ok := call.Args[0].(*ast.CallExpr)
	if !ok {
		return false
	}
	if !selectorNamed(baseURL.Fun, "transport", "BaseURL") {
		return false
	}
	option, ok := call.Args[1].(*ast.CallExpr)
	if !ok || len(option.Args) != 1 {
		return false
	}
	if !selectorNamed(option.Fun, openapiAlias, "WithHTTPClient") {
		return false
	}
	doer, ok := option.Args[0].(*ast.Ident)
	return ok && doer.Name == wantDoer
}

func oauthHTTPCall(call *ast.CallExpr) bool {
	return selectorNamed(call.Fun, "auth", "HTTP") && len(call.Args) == 1 && identifierIs(call.Args[0], "transport")
}

func returnedClientFields(expression ast.Expr) map[string]string {
	fields := map[string]string{}
	literal := pointerCompositeLiteral(expression)
	if literal == nil {
		return fields
	}
	typeName, ok := literal.Type.(*ast.Ident)
	if !ok || typeName.Name != "Client" {
		return fields
	}
	return literalIdentFields(literal)
}

func validArguments(constructor string, arguments []ast.Expr) bool {
	if constructor == "NewExternalHTTPS" {
		return len(arguments) == 2 && selectorNamed(arguments[0], "cfg", "BaseURL") &&
			selectorNamed(arguments[1], "cfg", "Limits")
	}
	return constructor == "NewPrivateHTTPS" && len(arguments) == 3 &&
		selectorNamed(arguments[0], "cfg", "BaseURL") && selectorNamed(arguments[1], "cfg", "PrivateDNSSuffix") &&
		selectorNamed(arguments[2], "cfg", "Limits")
}
