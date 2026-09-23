//go:build ignore

package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"path"
	"strconv"
	"strings"
)

func parseFile(filename string) (*ast.File, error) {
	parsed, err := parser.ParseFile(token.NewFileSet(), filename, nil, 0)
	if err != nil {
		return nil, fmt.Errorf("%s: parse: %w", filename, err)
	}
	return parsed, nil
}

func clientLiteralFields(expression ast.Expr) map[string]string {
	return literalIdentFields(pointerCompositeLiteral(expression))
}

func literalIdentFields(literal *ast.CompositeLit) map[string]string {
	fields := map[string]string{}
	for key, value := range keyedFields(literal) {
		if identifier, ok := value.(*ast.Ident); ok {
			fields[key] = identifier.Name
		}
	}
	return fields
}

// keyedFields maps each identifier key of a keyed composite literal to its
// value. A nil literal has no fields.
func keyedFields(literal *ast.CompositeLit) map[string]ast.Expr {
	fields := map[string]ast.Expr{}
	if literal == nil {
		return fields
	}
	for _, element := range literal.Elts {
		pair, pairOK := element.(*ast.KeyValueExpr)
		if !pairOK {
			continue
		}
		if key, keyOK := pair.Key.(*ast.Ident); keyOK {
			fields[key.Name] = pair.Value
		}
	}
	return fields
}

func pointerCompositeLiteral(expression ast.Expr) *ast.CompositeLit {
	pointer, ok := expression.(*ast.UnaryExpr)
	if !ok || pointer.Op != token.AND {
		return nil
	}
	literal, _ := pointer.X.(*ast.CompositeLit)
	return literal
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

// usableImportAlias is importAlias restricted to a name a selector can use; a
// missing, dot, or blank import yields "".
func usableImportAlias(file *ast.File, suffix string) string {
	alias := importAlias(file, suffix)
	if alias == "." || alias == "_" {
		return ""
	}
	return alias
}

// oauthConfigMapping matches alias.New(alias.Config{...}) with the canonical
// OAuth client-credentials field mapping.
func oauthConfigMapping(call *ast.CallExpr, alias string) bool {
	if !selectorNamed(call.Fun, alias, "New") || len(call.Args) != 1 {
		return false
	}
	literal, ok := call.Args[0].(*ast.CompositeLit)
	if !ok || !selectorNamed(literal.Type, alias, "Config") {
		return false
	}
	fields := keyedFields(literal)
	scopes, ok := fields["Scopes"].(*ast.CallExpr)
	if !ok || len(scopes.Args) != 1 || !selectorNamed(scopes.Fun, "strings", "Fields") {
		return false
	}
	return expressionPath(fields["TokenURL"]) == "cfg.OAuth.TokenURL" &&
		expressionPath(fields["ClientID"]) == "cfg.OAuth.ClientID" &&
		expressionPath(fields["ClientSecret"]) == "cfg.OAuth.ClientSecret" &&
		expressionPath(scopes.Args[0]) == "cfg.OAuth.Scopes"
}

func ownedBy(selector *ast.SelectorExpr, ownerName string) bool {
	owner, ok := selector.X.(*ast.Ident)
	return ok && owner.Name == ownerName
}

// selectorNamed reports whether expression is exactly owner.name.
func selectorNamed(expression ast.Expr, ownerName, name string) bool {
	selector, ok := expression.(*ast.SelectorExpr)
	return ok && selector.Sel.Name == name && ownedBy(selector, ownerName)
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
