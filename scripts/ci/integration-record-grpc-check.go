//go:build ignore

package main

import (
	"errors"
	"fmt"
	"go/ast"
	"go/token"
	"os"
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
	filename := arguments[0]
	auth, ok := grpcAuthModes[arguments[1]]
	if !ok {
		fmt.Fprintf(os.Stderr, "unknown AUTH %q\n", arguments[1])
		os.Exit(2)
	}
	parsed, err := parseFile(filename)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := checkGRPCClient(parsed, auth); err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", filename, err)
		os.Exit(1)
	}
}

// grpcAuth is what one AUTH mode requires of the adapter: whether the
// connection is opened through an OAuth client, and how many of each
// mode-specific binding must be present.
type grpcAuth struct {
	oauth           bool
	connections     int
	authConfig      int
	authConnections int
	authClose       int
}

var grpcAuthModes = map[string]grpcAuth{
	"none":                      {connections: 1},
	"oauth2-client-credentials": {oauth: true, authConfig: 1, authConnections: 1, authClose: 1},
}

// checkGRPCClient matches the expected constructor shape, not equivalent data
// flow in every form. Refactoring the adapter may require updating this check.
func checkGRPCClient(file *ast.File, auth grpcAuth) error {
	grpcAlias := usableImportAlias(file, "/internal/infra/grpcclient")
	oauthAlias := usableImportAlias(file, "/internal/infra/oauth2clientcredentials")
	credentialsAlias := usableImportAlias(file, "google.golang.org/grpc/credentials")
	if grpcAlias == "" || credentialsAlias == "" {
		return errors.New("missing gRPC owners")
	}

	targetBindings, tlsBindings, connectionBindings := 0, 0, 0
	authConfigBindings, authConnectionBindings, returnedClients := 0, 0, 0
	for _, declaration := range file.Decls {
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
				if !auth.oauth && grpcConnectionCall(call, grpcAlias) {
					connectionBindings++
				}
				if auth.oauth && authConnectionCall(call) {
					authConnectionBindings++
				}
			case "auth":
				if auth.oauth && oauthConfigMapping(call, oauthAlias) {
					authConfigBindings++
				}
			}
		}
		for _, statement := range function.Body.List {
			returned, ok := statement.(*ast.ReturnStmt)
			if !ok || len(returned.Results) != 2 || !identifierIs(returned.Results[1], "nil") {
				continue
			}
			fields := clientLiteralFields(returned.Results[0])
			if fields["conn"] != "conn" {
				continue
			}
			if auth.oauth && fields["auth"] != "auth" {
				continue
			}
			returnedClients++
		}
	}

	closeOnce, authClose, connectionClose := closeFlow(file, auth.oauth)
	if targetBindings != 1 || tlsBindings != 1 || connectionBindings != auth.connections ||
		authConfigBindings != auth.authConfig || authConnectionBindings != auth.authConnections ||
		returnedClients != 1 || closeOnce != 1 || authClose != auth.authClose || connectionClose != 1 {
		return fmt.Errorf("target=%d tls=%d conn=%d authConfig=%d authConn=%d returned=%d once=%d authClose=%d connClose=%d",
			targetBindings, tlsBindings, connectionBindings, authConfigBindings, authConnectionBindings,
			returnedClients, closeOnce, authClose, connectionClose)
	}
	return nil
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
	if !selectorNamed(call.Fun, alias, "NewTLS") || len(call.Args) != 1 {
		return false
	}
	return identifierIs(keyedFields(pointerCompositeLiteral(call.Args[0]))["ServerName"], "hostname")
}

func grpcConnectionCall(call *ast.CallExpr, alias string) bool {
	return selectorNamed(call.Fun, alias, "New") && connectionArguments(call.Args)
}

func authConnectionCall(call *ast.CallExpr) bool {
	return selectorNamed(call.Fun, "auth", "GRPC") && connectionArguments(call.Args)
}

func connectionArguments(arguments []ast.Expr) bool {
	if len(arguments) != 2 || !identifierIs(arguments[0], "target") {
		return false
	}
	literal, ok := arguments[1].(*ast.CompositeLit)
	return ok && identifierIs(keyedFields(literal)["TransportCredentials"], "creds")
}

// closeFlow counts closeOnce.Do callbacks in Close methods, and within them the
// guarded auth and connection closes the mode requires.
func closeFlow(file *ast.File, oauth bool) (once, authClose, connectionClose int) {
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
			if oauth {
				if len(statements) == 2 && guardedAuthClose(statements[0]) {
					authClose++
				}
				if len(statements) == 2 && guardedConnectionClose(statements[1]) {
					connectionClose++
				}
			} else if len(statements) == 1 && guardedConnectionClose(statements[0]) {
				connectionClose++
			}
		}
	}
	return once, authClose, connectionClose
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
