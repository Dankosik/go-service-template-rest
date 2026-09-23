//go:build ignore

package main

import (
	"go/ast"
	"go/token"
	"path"
	"strconv"
	"strings"
)

func clientLiteralFields(expression ast.Expr) map[string]string {
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

func oauthConfigMapping(call *ast.CallExpr, alias string) bool {
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
	scopes, ok := fields["Scopes"].(*ast.CallExpr)
	if !ok || len(scopes.Args) != 1 {
		return false
	}
	stringsFields, ok := scopes.Fun.(*ast.SelectorExpr)
	return expressionPath(fields["TokenURL"]) == "cfg.OAuth.TokenURL" &&
		expressionPath(fields["ClientID"]) == "cfg.OAuth.ClientID" &&
		expressionPath(fields["ClientSecret"]) == "cfg.OAuth.ClientSecret" &&
		ok && stringsFields.Sel.Name == "Fields" && ownedBy(stringsFields, "strings") &&
		expressionPath(scopes.Args[0]) == "cfg.OAuth.Scopes"
}

func ownedBy(selector *ast.SelectorExpr, ownerName string) bool {
	owner, ok := selector.X.(*ast.Ident)
	return ok && owner.Name == ownerName
}

func selectorTypeIs(expression ast.Expr, ownerName, typeName string) bool {
	selector, ok := expression.(*ast.SelectorExpr)
	return ok && selector.Sel.Name == typeName && ownedBy(selector, ownerName)
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
