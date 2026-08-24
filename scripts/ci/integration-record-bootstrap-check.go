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
	if len(arguments) < 7 {
		fmt.Fprintln(os.Stderr, "usage: integration-record-bootstrap-check STARTUP RUN IMPORT_SUFFIX INIT_FUNC CLIENT_VAR CONFIG_PATH FIELD=VALUE...")
		os.Exit(2)
	}

	startupFile, runFile := arguments[0], arguments[1]
	importSuffix, initFunction := arguments[2], arguments[3]
	clientVariable, configPath := arguments[4], arguments[5]
	expected := make(map[string]string, len(arguments)-6)
	for _, raw := range arguments[6:] {
		field, value, ok := strings.Cut(raw, "=")
		if !ok || field == "" || value == "" {
			fmt.Fprintf(os.Stderr, "invalid mapping %q\n", raw)
			os.Exit(2)
		}
		expected[field] = value
	}

	startup := mustParse(startupFile)
	alias := importAlias(startup, importSuffix)
	if alias == "" || alias == "." || alias == "_" {
		fmt.Fprintf(os.Stderr, "%s: missing usable import ending in %s\n", startupFile, importSuffix)
		os.Exit(1)
	}

	actual := map[string]string{}
	startupFlows := 0
	for _, declaration := range startup.Decls {
		function, ok := declaration.(*ast.FuncDecl)
		if !ok || function.Recv != nil || function.Name.Name != initFunction || function.Body == nil {
			continue
		}
		for index := 0; index+1 < len(function.Body.List); index++ {
			literal, ok := startupConstruction(function.Body.List[index], alias)
			if !ok || !errorReturn(function.Body.List[index+1], true) || !clientReturn(function.Body.List[index+2]) {
				continue
			}
			startupFlows++
			collectMappings("", literal, actual)
		}
	}

	if startupFlows != 1 {
		fmt.Fprintf(os.Stderr, "%s: canonical startup flows=%d, want 1\n", startupFile, startupFlows)
		os.Exit(1)
	}
	for field, want := range expected {
		if got := actual[field]; got != want {
			fmt.Fprintf(os.Stderr, "%s: mapping %s=%q, want %q\n", startupFile, field, got, want)
			os.Exit(1)
		}
	}

	run := mustParse(runFile)
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
				fmt.Fprintf(os.Stderr, "%s: %s flow error=%t closed=%t assignments=%d closes=%d\n", runFile, clientVariable, errorOK, closedOK, assignments, closes)
				continue
			}
			runFlows++
		}
	}
	if runFlows != 1 {
		fmt.Fprintf(os.Stderr, "%s: canonical %s lifecycle flows=%d, want 1\n", runFile, clientVariable, runFlows)
		os.Exit(1)
	}
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
	if !ok || constructor.Sel.Name != "New" || !selectorOwnedBy(constructor, alias) {
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
		if ok && selector.Sel.Name == "Close" && selectorOwnedBy(selector, variable) {
			count++
		}
		return true
	})
	return count
}

func mustParse(filename string) *ast.File {
	parsed, err := parser.ParseFile(token.NewFileSet(), filename, nil, 0)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: parse: %v\n", filename, err)
		os.Exit(1)
	}
	return parsed
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

func selectorTypeIs(expression ast.Expr, ownerName, typeName string) bool {
	selector, ok := expression.(*ast.SelectorExpr)
	if !ok || selector.Sel.Name != typeName {
		return false
	}
	owner, ok := selector.X.(*ast.Ident)
	return ok && owner.Name == ownerName
}

func selectorOwnedBy(selector *ast.SelectorExpr, ownerName string) bool {
	owner, ok := selector.X.(*ast.Ident)
	return ok && owner.Name == ownerName
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

func identifierIs(expression ast.Expr, name string) bool {
	identifier, ok := expression.(*ast.Ident)
	return ok && identifier.Name == name
}
