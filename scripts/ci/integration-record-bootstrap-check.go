//go:build ignore

package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
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
	alias := importAlias(startup, importSuffix)
	if alias == "" || alias == "." || alias == "_" {
		return 1, []string{fmt.Sprintf("%s: missing usable import ending in %s", startupFile, importSuffix)}
	}
	if diagnostic := checkStartupMapping(startup, startupFile, alias, initFunction, expected); diagnostic != "" {
		return 1, []string{diagnostic}
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

func checkStartupMapping(startup *ast.File, startupFile, alias, initFunction string, expected map[string]string) string {
	actual := map[string]string{}
	startupFlows := 0
	for _, declaration := range startup.Decls {
		function, ok := declaration.(*ast.FuncDecl)
		if !ok || function.Recv != nil || function.Name.Name != initFunction || function.Body == nil {
			continue
		}
		for index := 0; index+2 < len(function.Body.List); index++ {
			literal, ok := startupConstruction(function.Body.List[index], alias)
			if !ok || !errorReturn(function.Body.List[index+1], true) || !clientReturn(function.Body.List[index+2]) {
				continue
			}
			startupFlows++
			collectMappings("", literal, actual)
		}
	}

	if startupFlows != 1 {
		return fmt.Sprintf("%s: canonical startup flows=%d, want 1", startupFile, startupFlows)
	}
	for field, want := range expected {
		if got := actual[field]; got != want {
			return fmt.Sprintf("%s: mapping %s=%q, want %q", startupFile, field, got, want)
		}
	}
	return ""
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
		for index := 0; index+2 < len(function.Body.List); index++ {
			constructionOK := runConstruction(function.Body.List[index], initFunction, clientVariable, configPath)
			if !constructionOK {
				continue
			}
			errorOK := errorReturn(function.Body.List[index+1], false)
			closedOK := false
			for later := index + 2; later < len(function.Body.List); later++ {
				if falseAssignment(function.Body.List[later], closedVariable) {
					closedOK = true
					break
				}
			}
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
	constructor, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || constructor.Sel.Name != "New" || !ownedBy(constructor, alias) {
		return nil, false
	}
	literal, ok := call.Args[0].(*ast.CompositeLit)
	return literal, ok && selectorTypeIs(literal.Type, alias, "Config")
}

func errorReturn(statement ast.Stmt, twoResults bool) bool {
	branch, ok := statement.(*ast.IfStmt)
	if !ok || !errNotNil(branch.Cond) || len(branch.Body.List) != 1 {
		return false
	}
	returned, ok := branch.Body.List[0].(*ast.ReturnStmt)
	if !ok {
		return false
	}
	if twoResults {
		return len(returned.Results) == 2 && identifierIs(returned.Results[0], "nil") && !identifierIs(returned.Results[1], "nil")
	}
	return len(returned.Results) == 1 && identifierIs(returned.Results[0], "err")
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
		selector, ok := call.Fun.(*ast.SelectorExpr)
		if ok && selector.Sel.Name == "Close" && ownedBy(selector, variable) {
			count++
		}
		return true
	})
	return count
}

func parseFile(filename string) (*ast.File, error) {
	parsed, err := parser.ParseFile(token.NewFileSet(), filename, nil, 0)
	if err != nil {
		return nil, fmt.Errorf("%s: parse: %w", filename, err)
	}
	return parsed, nil
}

func collectMappings(prefix string, literal *ast.CompositeLit, mappings map[string]string) {
	for _, element := range literal.Elts {
		pair, ok := element.(*ast.KeyValueExpr)
		if !ok {
			continue
		}
		key, ok := pair.Key.(*ast.Ident)
		if !ok {
			continue
		}
		field := prefix + key.Name
		if nested, ok := pair.Value.(*ast.CompositeLit); ok {
			collectMappings(field+".", nested, mappings)
			continue
		}
		mappings[field] = expressionPath(pair.Value)
	}
}
