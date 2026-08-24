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
	if len(arguments) != 2 {
		fmt.Fprintln(os.Stderr, "usage: integration-record-grpc-check FILE AUTH")
		os.Exit(2)
	}
	filename, authMode := arguments[0], arguments[1]
	parsed, err := parser.ParseFile(token.NewFileSet(), filename, nil, 0)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: parse: %v\n", filename, err)
		os.Exit(1)
	}

	grpcAlias := importAlias(parsed, "/internal/infra/grpcclient")
	oauthAlias := importAlias(parsed, "/internal/infra/oauth2clientcredentials")
	credentialsAlias := importAlias(parsed, "google.golang.org/grpc/credentials")
	if grpcAlias == "" || credentialsAlias == "" {
		fmt.Fprintf(os.Stderr, "%s: missing gRPC owners\n", filename)
		os.Exit(1)
	}

	targetBindings, tlsBindings, connectionBindings := 0, 0, 0
	authConfigBindings, authConnectionBindings, returnedClients := 0, 0, 0
	for _, declaration := range parsed.Decls {
		function, ok := declaration.(*ast.FuncDecl)
		if !ok || function.Recv != nil || function.Name.Name != "New" || function.Body == nil {
			continue
		}
		for _, statement := range function.Body.List {
			assignment, ok := statement.(*ast.AssignStmt)
			if !ok || len(assignment.Rhs) != 1 {
				continue
			}
			if targetAssignment(assignment) {
				targetBindings++
			}
			assigned, ok := assignment.Lhs[0].(*ast.Ident)
			if !ok {
				continue
			}
			call, ok := assignment.Rhs[0].(*ast.CallExpr)
			if !ok {
				continue
			}
			switch assigned.Name {
			case "creds":
				if tlsCredentialCall(call, credentialsAlias) {
					tlsBindings++
				}
			case "conn":
				if authMode == "none" && grpcConnectionCall(call, grpcAlias) {
					connectionBindings++
				}
				if authMode == "oauth2-client-credentials" && authConnectionCall(call) {
					authConnectionBindings++
				}
			case "auth":
				if authMode == "oauth2-client-credentials" && oauthConfigCall(call, oauthAlias) {
					authConfigBindings++
				}
			}
		}
		for _, statement := range function.Body.List {
			returned, ok := statement.(*ast.ReturnStmt)
			if !ok || len(returned.Results) != 2 || !identifierIs(returned.Results[1], "nil") {
				continue
			}
			fields := returnedClientFields(returned.Results[0])
			if fields["conn"] != "conn" {
				continue
			}
			if authMode == "oauth2-client-credentials" && fields["auth"] != "auth" {
				continue
			}
			returnedClients++
		}
	}

	wantConnections, wantAuthConfig, wantAuthConnections := 1, 0, 0
	if authMode == "oauth2-client-credentials" {
		wantConnections, wantAuthConfig, wantAuthConnections = 0, 1, 1
	}
	closeOnce, authClose, connectionClose := closeFlow(parsed, authMode)
	wantAuthClose := 0
	if authMode == "oauth2-client-credentials" {
		wantAuthClose = 1
	}
	if targetBindings != 1 || tlsBindings != 1 || connectionBindings != wantConnections ||
		authConfigBindings != wantAuthConfig || authConnectionBindings != wantAuthConnections ||
		returnedClients != 1 || closeOnce != 1 || authClose != wantAuthClose || connectionClose != 1 {
		fmt.Fprintf(os.Stderr, "%s: target=%d tls=%d conn=%d authConfig=%d authConn=%d returned=%d once=%d authClose=%d connClose=%d\n",
			filename, targetBindings, tlsBindings, connectionBindings, authConfigBindings, authConnectionBindings,
			returnedClients, closeOnce, authClose, connectionClose)
		os.Exit(1)
	}
}

func targetAssignment(assignment *ast.AssignStmt) bool {
	if assignment.Tok != token.DEFINE || len(assignment.Lhs) != 3 || len(assignment.Rhs) != 1 ||
		!identifierIs(assignment.Lhs[0], "target") || !identifierIs(assignment.Lhs[1], "hostname") || !identifierIs(assignment.Lhs[2], "err") {
		return false
	}
	call, ok := assignment.Rhs[0].(*ast.CallExpr)
	if !ok {
		return false
	}
	function, functionOK := call.Fun.(*ast.Ident)
	return functionOK && function.Name == "parseGRPCTarget" && len(call.Args) == 1 && expressionPath(call.Args[0]) == "cfg.Target"
}

func tlsCredentialCall(call *ast.CallExpr, alias string) bool {
	selector, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || selector.Sel.Name != "NewTLS" || !ownedBy(selector, alias) || len(call.Args) != 1 {
		return false
	}
	pointer, ok := call.Args[0].(*ast.UnaryExpr)
	if !ok || pointer.Op != token.AND {
		return false
	}
	literal, ok := pointer.X.(*ast.CompositeLit)
	if !ok {
		return false
	}
	for _, element := range literal.Elts {
		pair, pairOK := element.(*ast.KeyValueExpr)
		if !pairOK {
			continue
		}
		key, keyOK := pair.Key.(*ast.Ident)
		if keyOK && key.Name == "ServerName" && identifierIs(pair.Value, "hostname") {
			return true
		}
	}
	return false
}

func grpcConnectionCall(call *ast.CallExpr, alias string) bool {
	selector, ok := call.Fun.(*ast.SelectorExpr)
	return ok && selector.Sel.Name == "New" && ownedBy(selector, alias) && connectionArguments(call.Args)
}

func authConnectionCall(call *ast.CallExpr) bool {
	selector, ok := call.Fun.(*ast.SelectorExpr)
	return ok && selector.Sel.Name == "GRPC" && ownedBy(selector, "auth") && connectionArguments(call.Args)
}

func connectionArguments(arguments []ast.Expr) bool {
	if len(arguments) != 2 || !identifierIs(arguments[0], "target") {
		return false
	}
	literal, ok := arguments[1].(*ast.CompositeLit)
	if !ok {
		return false
	}
	for _, element := range literal.Elts {
		pair, pairOK := element.(*ast.KeyValueExpr)
		if !pairOK {
			continue
		}
		key, keyOK := pair.Key.(*ast.Ident)
		if keyOK && key.Name == "TransportCredentials" && identifierIs(pair.Value, "creds") {
			return true
		}
	}
	return false
}

func oauthConfigCall(call *ast.CallExpr, alias string) bool {
	selector, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || selector.Sel.Name != "New" || !ownedBy(selector, alias) || len(call.Args) != 1 {
		return false
	}
	literal, ok := call.Args[0].(*ast.CompositeLit)
	if !ok {
		return false
	}
	fields := map[string]ast.Expr{}
	for _, element := range literal.Elts {
		pair, pairOK := element.(*ast.KeyValueExpr)
		if !pairOK {
			continue
		}
		key, keyOK := pair.Key.(*ast.Ident)
		if keyOK {
			fields[key.Name] = pair.Value
		}
	}
	return expressionPath(fields["TokenURL"]) == "cfg.OAuth.TokenURL" &&
		expressionPath(fields["ClientID"]) == "cfg.OAuth.ClientID" &&
		expressionPath(fields["ClientSecret"]) == "cfg.OAuth.ClientSecret" && stringsFieldsCall(fields["Scopes"])
}

func stringsFieldsCall(expression ast.Expr) bool {
	call, ok := expression.(*ast.CallExpr)
	if !ok || len(call.Args) != 1 || expressionPath(call.Args[0]) != "cfg.OAuth.Scopes" {
		return false
	}
	selector, ok := call.Fun.(*ast.SelectorExpr)
	return ok && selector.Sel.Name == "Fields" && ownedBy(selector, "strings")
}

func closeFlow(file *ast.File, authMode string) (int, int, int) {
	once, auth, connection := 0, 0, 0
	for _, declaration := range file.Decls {
		function, ok := declaration.(*ast.FuncDecl)
		if !ok || function.Recv == nil || function.Name.Name != "Close" || function.Body == nil {
			continue
		}
		for _, statement := range function.Body.List {
			expression, ok := statement.(*ast.ExprStmt)
			if !ok {
				continue
			}
			call, ok := expression.X.(*ast.CallExpr)
			if !ok || len(call.Args) != 1 {
				continue
			}
			selector, ok := call.Fun.(*ast.SelectorExpr)
			callback, callbackOK := call.Args[0].(*ast.FuncLit)
			if !ok || selector.Sel.Name != "Do" || expressionPath(selector.X) != "c.closeOnce" || !callbackOK {
				continue
			}
			once++
			statements := callback.Body.List
			if authMode == "oauth2-client-credentials" {
				if len(statements) == 2 && guardedAuthClose(statements[0]) {
					auth++
				}
				if len(statements) == 2 && guardedConnectionClose(statements[1]) {
					connection++
				}
			} else if len(statements) == 1 && guardedConnectionClose(statements[0]) {
				connection++
			}
		}
	}
	return once, auth, connection
}

func guardedAuthClose(statement ast.Stmt) bool {
	body := nonNilGuardBody(statement, "c.auth")
	if len(body) != 1 {
		return false
	}
	expression, ok := body[0].(*ast.ExprStmt)
	if !ok {
		return false
	}
	call, ok := expression.X.(*ast.CallExpr)
	if !ok || len(call.Args) != 0 {
		return false
	}
	selector, ok := call.Fun.(*ast.SelectorExpr)
	return ok && selector.Sel.Name == "Close" && expressionPath(selector.X) == "c.auth"
}

func guardedConnectionClose(statement ast.Stmt) bool {
	body := nonNilGuardBody(statement, "c.conn")
	if len(body) != 1 {
		return false
	}
	assignment, ok := body[0].(*ast.AssignStmt)
	if !ok || assignment.Tok != token.ASSIGN || len(assignment.Lhs) != 1 || len(assignment.Rhs) != 1 ||
		expressionPath(assignment.Lhs[0]) != "c.closeErr" {
		return false
	}
	call, ok := assignment.Rhs[0].(*ast.CallExpr)
	if !ok || len(call.Args) != 0 {
		return false
	}
	selector, ok := call.Fun.(*ast.SelectorExpr)
	return ok && selector.Sel.Name == "Close" && expressionPath(selector.X) == "c.conn"
}

func nonNilGuardBody(statement ast.Stmt, owner string) []ast.Stmt {
	guard, ok := statement.(*ast.IfStmt)
	if !ok || guard.Init != nil || guard.Else != nil {
		return nil
	}
	condition, ok := guard.Cond.(*ast.BinaryExpr)
	if !ok || condition.Op != token.NEQ || expressionPath(condition.X) != owner || !identifierIs(condition.Y, "nil") {
		return nil
	}
	return guard.Body.List
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
	for _, element := range literal.Elts {
		pair, pairOK := element.(*ast.KeyValueExpr)
		if !pairOK {
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
