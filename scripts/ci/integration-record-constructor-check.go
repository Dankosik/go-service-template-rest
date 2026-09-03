//go:build ignore

package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path"
	"strconv"
	"strings"
)

func main() {
	arguments := os.Args[1:]
	if len(arguments) > 0 && arguments[0] == "--" {
		arguments = arguments[1:]
	}
	if len(arguments) != 5 {
		fmt.Fprintln(os.Stderr, "usage: integration-record-constructor-check FILE IMPORT_SUFFIX EXPECTED FORBIDDEN AUTH")
		os.Exit(2)
	}

	filename, importSuffix, expected, forbidden, authMode := arguments[0], arguments[1], arguments[2], arguments[3], arguments[4]
	parsed, err := parser.ParseFile(token.NewFileSet(), filename, nil, 0)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: parse: %v\n", filename, err)
		os.Exit(1)
	}

	alias := importAlias(parsed, importSuffix)
	if alias == "" || alias == "." || alias == "_" {
		fmt.Fprintf(os.Stderr, "%s: missing usable import ending in %s\n", filename, importSuffix)
		os.Exit(1)
	}

	expectedAssignments := 0
	forbiddenCalls := 0
	generatedBindings := 0
	authBindings := 0
	doerBindings := 0
	authConfigBindings := 0
	returnedClients := 0
	openapiAlias := importAlias(parsed, "/internal/openapi")
	oauthAlias := importAlias(parsed, "/internal/infra/oauth2clientcredentials")
	ast.Inspect(parsed, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		selector, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || selector.Sel.Name != forbidden {
			return true
		}
		owner, ok := selector.X.(*ast.Ident)
		if ok && owner.Name == alias {
			forbiddenCalls++
		}
		return true
	})
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
			if authMode == "oauth2-client-credentials" && assigned.Name == "doer" {
				if source, ok := assignment.Rhs[0].(*ast.Ident); ok && source.Name == "authenticated" {
					doerBindings++
				}
				continue
			}
			call, ok := assignment.Rhs[0].(*ast.CallExpr)
			if !ok {
				continue
			}
			selector, ok := call.Fun.(*ast.SelectorExpr)
			if !ok {
				continue
			}
			if assigned.Name == "transport" && selector.Sel.Name == expected {
				owner, ownerOK := selector.X.(*ast.Ident)
				if ownerOK && owner.Name == alias && validArguments(expected, call.Args) {
					expectedAssignments++
				}
			}

			if assigned.Name == "generated" && openapiAlias != "" && generatedClientCall(call, openapiAlias, authMode) {
				generatedBindings++
			}
			if authMode == "oauth2-client-credentials" && assigned.Name == "authenticated" && oauthHTTPCall(call) {
				authBindings++
			}
			if authMode == "oauth2-client-credentials" && assigned.Name == "auth" && oauthConfigCall(call, oauthAlias) {
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
			if authMode == "oauth2-client-credentials" && fields["auth"] != "auth" {
				continue
			}
			returnedClients++
		}
	}

	wantAuthBindings, wantDoerBindings, wantAuthConfigBindings := 0, 0, 0
	if authMode == "oauth2-client-credentials" {
		wantAuthBindings, wantDoerBindings, wantAuthConfigBindings = 1, 1, 1
	}
	if expectedAssignments != 1 || forbiddenCalls != 0 || generatedBindings != 1 || returnedClients != 1 ||
		authBindings != wantAuthBindings || doerBindings != wantDoerBindings || authConfigBindings != wantAuthConfigBindings {
		fmt.Fprintf(os.Stderr, "%s: constructor=%d forbidden=%d generated=%d returned=%d auth=%d authConfig=%d doer=%d\n",
			filename, expectedAssignments, forbiddenCalls, generatedBindings, returnedClients, authBindings, authConfigBindings, doerBindings)
		os.Exit(1)
	}
}

func returnedClientFields(expression ast.Expr) map[string]string {
	fields := map[string]string{}
	pointer, ok := expression.(*ast.UnaryExpr)
	if !ok || pointer.Op != token.AND {
		return fields
	}
	literal, ok := pointer.X.(*ast.CompositeLit)
	if !ok {
		return fields
	}
	typeName, ok := literal.Type.(*ast.Ident)
	if !ok || typeName.Name != "Client" {
		return fields
	}
	for _, element := range literal.Elts {
		pair, ok := element.(*ast.KeyValueExpr)
		if !ok {
			continue
		}
		key, keyOK := pair.Key.(*ast.Ident)
		value, valueOK := pair.Value.(*ast.Ident)
		if keyOK && valueOK {
			fields[key.Name] = value.Name
		}
	}
	return fields
}

func importAlias(file *ast.File, suffix string) string {
	for _, spec := range file.Imports {
		importPath, err := strconv.Unquote(spec.Path.Value)
		if err != nil || !strings.HasSuffix(importPath, suffix) {
			continue
		}
		if spec.Name != nil {
			return spec.Name.Name
		}
		return path.Base(importPath)
	}
	return ""
}

func generatedClientCall(call *ast.CallExpr, openapiAlias, authMode string) bool {
	selector, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || selector.Sel.Name != "NewClient" || !ownedBy(selector, openapiAlias) || len(call.Args) != 2 {
		return false
	}
	baseURL, ok := call.Args[0].(*ast.CallExpr)
	if !ok {
		return false
	}
	baseSelector, ok := baseURL.Fun.(*ast.SelectorExpr)
	if !ok || baseSelector.Sel.Name != "BaseURL" || !ownedBy(baseSelector, "transport") {
		return false
	}
	option, ok := call.Args[1].(*ast.CallExpr)
	if !ok || len(option.Args) != 1 {
		return false
	}
	optionSelector, ok := option.Fun.(*ast.SelectorExpr)
	if !ok || optionSelector.Sel.Name != "WithHTTPClient" || !ownedBy(optionSelector, openapiAlias) {
		return false
	}
	wantDoer := "transport"
	if authMode == "oauth2-client-credentials" {
		wantDoer = "doer"
	}
	doer, ok := option.Args[0].(*ast.Ident)
	return ok && doer.Name == wantDoer
}

func oauthHTTPCall(call *ast.CallExpr) bool {
	selector, ok := call.Fun.(*ast.SelectorExpr)
	return ok && selector.Sel.Name == "HTTP" && ownedBy(selector, "auth") && len(call.Args) == 1 && identifierIs(call.Args[0], "transport")
}

func oauthConfigCall(call *ast.CallExpr, oauthAlias string) bool {
	selector, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || selector.Sel.Name != "New" || !ownedBy(selector, oauthAlias) || len(call.Args) != 1 {
		return false
	}
	literal, ok := call.Args[0].(*ast.CompositeLit)
	if !ok || !selectorTypeIs(literal.Type, oauthAlias, "Config") {
		return false
	}
	fields := map[string]ast.Expr{}
	for _, element := range literal.Elts {
		pair, pairOK := element.(*ast.KeyValueExpr)
		if !pairOK {
			continue
		}
		key, keyOK := pair.Key.(*ast.Ident)
		if !keyOK {
			continue
		}
		fields[key.Name] = pair.Value
	}
	return selectorPathIs(fields["TokenURL"], "cfg.OAuth.TokenURL") &&
		selectorPathIs(fields["ClientID"], "cfg.OAuth.ClientID") &&
		selectorPathIs(fields["ClientSecret"], "cfg.OAuth.ClientSecret") &&
		stringsFieldsCall(fields["Scopes"], "cfg.OAuth.Scopes")
}

func selectorTypeIs(expression ast.Expr, ownerName, typeName string) bool {
	selector, ok := expression.(*ast.SelectorExpr)
	return ok && selector.Sel.Name == typeName && ownedBy(selector, ownerName)
}

func selectorPathIs(expression ast.Expr, want string) bool {
	return expressionPath(expression) == want
}

func stringsFieldsCall(expression ast.Expr, argument string) bool {
	call, ok := expression.(*ast.CallExpr)
	if !ok || len(call.Args) != 1 {
		return false
	}
	selector, ok := call.Fun.(*ast.SelectorExpr)
	return ok && selector.Sel.Name == "Fields" && ownedBy(selector, "strings") && selectorPathIs(call.Args[0], argument)
}

func ownedBy(selector *ast.SelectorExpr, ownerName string) bool {
	owner, ok := selector.X.(*ast.Ident)
	return ok && owner.Name == ownerName
}

func identifierIs(expression ast.Expr, name string) bool {
	identifier, ok := expression.(*ast.Ident)
	return ok && identifier.Name == name
}

func expressionPath(expression ast.Expr) string {
	switch value := expression.(type) {
	case *ast.Ident:
		return value.Name
	case *ast.SelectorExpr:
		prefix := expressionPath(value.X)
		if prefix == "" {
			return ""
		}
		return prefix + "." + value.Sel.Name
	default:
		return ""
	}
}

func validArguments(constructor string, arguments []ast.Expr) bool {
	if constructor == "NewExternalHTTPS" {
		return len(arguments) == 2 && selectorIs(arguments[0], "cfg", "BaseURL") &&
			selectorIs(arguments[1], "cfg", "Limits")
	}
	return constructor == "NewPrivateHTTPS" && len(arguments) == 3 &&
		selectorIs(arguments[0], "cfg", "BaseURL") && selectorIs(arguments[1], "cfg", "PrivateDNSSuffix") &&
		selectorIs(arguments[2], "cfg", "Limits")
}

func selectorIs(expression ast.Expr, ownerName, fieldName string) bool {
	selector, ok := expression.(*ast.SelectorExpr)
	if !ok || selector.Sel.Name != fieldName {
		return false
	}
	owner, ok := selector.X.(*ast.Ident)
	return ok && owner.Name == ownerName
}
