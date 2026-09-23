//go:build ignore

// integration-record-bootstrap-check verifies the bootstrap wiring that
// scripts/integration-init.sh generates for one integration record.
//
// STARTUP must hold INIT_FUNC with adjacent statements
// `client, err := <alias>.New(<alias>.Config{...})`, an `if err != nil` guard
// returning nil and an error, and `return client, nil`; each FIELD=VALUE names
// one Config field path and the config path it must map. RUN must hold the
// construction `CLIENT_VAR, err := INIT_FUNC(CONFIG_PATH)` that wire_run_go
// inserts: an immediate `if err != nil { return err }`, a later
// `<name>Closed := false`, one assignment to CLIENT_VAR, and at least two Close
// calls on it.
package main

import (
	"fmt"
	"go/ast"
	"go/token"
	"os"
	"slices"
	"strings"
)

func main() {
	status, diagnostics := runBootstrapCheck(os.Args[1:])
	for _, diagnostic := range diagnostics {
		fmt.Fprintln(os.Stderr, diagnostic)
	}
	os.Exit(status)
}

func runBootstrapCheck(arguments []string) (int, []string) {
	if len(arguments) > 0 && arguments[0] == "--" {
		arguments = arguments[1:]
	}
	if len(arguments) < 7 {
		return 2, []string{"usage: integration-record-bootstrap-check STARTUP RUN IMPORT_SUFFIX INIT_FUNC CLIENT_VAR CONFIG_PATH FIELD=VALUE..."}
	}

	startupFile, runFile := arguments[0], arguments[1]
	importSuffix, initFunction := arguments[2], arguments[3]
	clientVariable, configPath := arguments[4], arguments[5]
	expected := make(map[string]string, len(arguments)-6)
	for _, raw := range arguments[6:] {
		field, value, ok := strings.Cut(raw, "=")
		if !ok || field == "" || value == "" {
			return 2, []string{fmt.Sprintf("invalid mapping %q", raw)}
		}
		expected[field] = value
	}

	startup, err := parseFile(startupFile)
	if err != nil {
		return 1, []string{err.Error()}
	}
	alias := usableImportAlias(startup, importSuffix)
	if alias == "" {
		return 1, []string{fmt.Sprintf("%s: missing usable import ending in %s", startupFile, importSuffix)}
	}
	if err := checkStartupMapping(startup, startupFile, alias, initFunction, expected); err != nil {
		return 1, []string{err.Error()}
	}

	run, err := parseFile(runFile)
	if err != nil {
		return 1, []string{err.Error()}
	}
	valid, diagnostics := checkRunLifecycle(run, runFile, initFunction, clientVariable, configPath)
	if !valid {
		return 1, diagnostics
	}
	return 0, diagnostics
}

func checkStartupMapping(startup *ast.File, startupFile, alias, initFunction string, expected map[string]string) error {
	actual := map[string]string{}
	startupFlows := 0
	for _, declaration := range startup.Decls {
		function, ok := declaration.(*ast.FuncDecl)
		if !ok || function.Recv != nil || function.Name.Name != initFunction || function.Body == nil {
			continue
		}
		// This is an AST shape check: construction, its error guard, and the
		// successful client return must be adjacent statements. An equivalent
		// refactoring to non-adjacent statements will not match.
		for index := 0; index+2 < len(function.Body.List); index++ {
			literal, ok := startupConstruction(function.Body.List[index], alias)
			if !ok || !returnsNilAndError(function.Body.List[index+1]) || !clientReturn(function.Body.List[index+2]) {
				continue
			}
			startupFlows++
			collectMappings("", literal, actual)
		}
	}

	if startupFlows != 1 {
		return fmt.Errorf("%s: canonical startup flows=%d, want 1", startupFile, startupFlows)
	}
	for field, want := range expected {
		if got := actual[field]; got != want {
			return fmt.Errorf("%s: mapping %s=%q, want %q", startupFile, field, got, want)
		}
	}
	return nil
}

func checkRunLifecycle(run *ast.File, runFile, initFunction, clientVariable, configPath string) (bool, []string) {
	var diagnostics []string
	runFlows := 0
	closedVariable := strings.TrimSuffix(clientVariable, "Client") + "Closed"
	for _, declaration := range run.Decls {
		function, ok := declaration.(*ast.FuncDecl)
		if !ok || function.Body == nil {
			continue
		}
		// This run-site AST shape requires construction followed immediately by
		// its error guard; the closed flag must appear later in the same body.
		for index := 0; index+2 < len(function.Body.List); index++ {
			constructionOK := runConstruction(function.Body.List[index], initFunction, clientVariable, configPath)
			if !constructionOK {
				continue
			}
			errorOK := returnsErr(function.Body.List[index+1])
			closedOK := slices.ContainsFunc(function.Body.List[index+2:], func(statement ast.Stmt) bool {
				return falseAssignment(statement, closedVariable)
			})
			assignments := assignmentsTo(function.Body, clientVariable)
			closes := closeCalls(function.Body, clientVariable)
			if !errorOK || !closedOK || assignments != 1 || closes < 2 {
				diagnostics = append(diagnostics, fmt.Sprintf("%s: %s flow error=%t closed=%t assignments=%d closes=%d", runFile, clientVariable, errorOK, closedOK, assignments, closes))
				continue
			}
			runFlows++
		}
	}
	if runFlows != 1 {
		diagnostics = append(diagnostics, fmt.Sprintf("%s: canonical %s lifecycle flows=%d, want 1", runFile, clientVariable, runFlows))
	}
	return runFlows == 1, diagnostics
}

func startupConstruction(statement ast.Stmt, alias string) (*ast.CompositeLit, bool) {
	assignment, ok := statement.(*ast.AssignStmt)
	if !ok || assignment.Tok != token.DEFINE || len(assignment.Lhs) != 2 || len(assignment.Rhs) != 1 ||
		!identifierIs(assignment.Lhs[0], "client") || !identifierIs(assignment.Lhs[1], "err") {
		return nil, false
	}
	call, ok := assignment.Rhs[0].(*ast.CallExpr)
	if !ok || len(call.Args) != 1 {
		return nil, false
	}
	if !selectorNamed(call.Fun, alias, "New") {
		return nil, false
	}
	literal, ok := call.Args[0].(*ast.CompositeLit)
	return literal, ok && selectorNamed(literal.Type, alias, "Config")
}

// returnsNilAndError matches `if err != nil { return nil, <non-nil> }`.
func returnsNilAndError(statement ast.Stmt) bool {
	returned := errGuardReturn(statement)
	return returned != nil && len(returned.Results) == 2 &&
		identifierIs(returned.Results[0], "nil") && !identifierIs(returned.Results[1], "nil")
}

// returnsErr matches `if err != nil { return err }`.
func returnsErr(statement ast.Stmt) bool {
	returned := errGuardReturn(statement)
	return returned != nil && len(returned.Results) == 1 && identifierIs(returned.Results[0], "err")
}

// errGuardReturn returns the sole return statement of an `if err != nil`
// guard, or nil when statement is not one.
func errGuardReturn(statement ast.Stmt) *ast.ReturnStmt {
	branch, ok := statement.(*ast.IfStmt)
	if !ok || !errNotNil(branch.Cond) || len(branch.Body.List) != 1 {
		return nil
	}
	returned, _ := branch.Body.List[0].(*ast.ReturnStmt)
	return returned
}

func errNotNil(expression ast.Expr) bool {
	comparison, ok := expression.(*ast.BinaryExpr)
	return ok && comparison.Op == token.NEQ && identifierIs(comparison.X, "err") && identifierIs(comparison.Y, "nil")
}

func clientReturn(statement ast.Stmt) bool {
	returned, ok := statement.(*ast.ReturnStmt)
	return ok && len(returned.Results) == 2 && identifierIs(returned.Results[0], "client") && identifierIs(returned.Results[1], "nil")
}

func runConstruction(statement ast.Stmt, initFunction, clientVariable, configPath string) bool {
	assignment, ok := statement.(*ast.AssignStmt)
	if !ok || assignment.Tok != token.DEFINE || len(assignment.Lhs) != 2 || len(assignment.Rhs) != 1 ||
		!identifierIs(assignment.Lhs[0], clientVariable) || !identifierIs(assignment.Lhs[1], "err") {
		return false
	}
	call, ok := assignment.Rhs[0].(*ast.CallExpr)
	if !ok || len(call.Args) != 1 || expressionPath(call.Args[0]) != configPath {
		return false
	}
	function, ok := call.Fun.(*ast.Ident)
	return ok && function.Name == initFunction
}

func falseAssignment(statement ast.Stmt, variable string) bool {
	assignment, ok := statement.(*ast.AssignStmt)
	return ok && assignment.Tok == token.DEFINE && len(assignment.Lhs) == 1 && len(assignment.Rhs) == 1 &&
		identifierIs(assignment.Lhs[0], variable) && identifierIs(assignment.Rhs[0], "false")
}

func assignmentsTo(body *ast.BlockStmt, variable string) int {
	count := 0
	ast.Inspect(body, func(node ast.Node) bool {
		assignment, ok := node.(*ast.AssignStmt)
		if !ok {
			return true
		}
		for _, target := range assignment.Lhs {
			if identifierIs(target, variable) {
				count++
			}
		}
		return true
	})
	return count
}

func closeCalls(body *ast.BlockStmt, variable string) int {
	count := 0
	ast.Inspect(body, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		if selectorNamed(call.Fun, variable, "Close") {
			count++
		}
		return true
	})
	return count
}

func collectMappings(prefix string, literal *ast.CompositeLit, mappings map[string]string) {
	for key, value := range keyedFields(literal) {
		field := prefix + key
		if nested, ok := value.(*ast.CompositeLit); ok {
			collectMappings(field+".", nested, mappings)
			continue
		}
		mappings[field] = expressionPath(value)
	}
}
