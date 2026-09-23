//go:build ignore

package main

import (
	"go/ast"
	"go/parser"
	"testing"
)

func TestTLSCredentialCallPointerLiteralShape(t *testing.T) {
	for _, test := range []struct {
		name string
		want bool
	}{
		{name: "credentials.NewTLS(&tls.Config{ServerName: hostname})", want: true},
		{name: "credentials.NewTLS(&Other{ServerName: hostname})", want: true},
		{name: "credentials.NewTLS(tls.Config{ServerName: hostname})"},
		{name: "credentials.NewTLS(&tls.Config{ServerName: other})"},
	} {
		t.Run(test.name, func(t *testing.T) {
			expression, err := parser.ParseExpr(test.name)
			if err != nil {
				t.Fatal(err)
			}
			call := expression.(*ast.CallExpr)
			if got := tlsCredentialCall(call, "credentials"); got != test.want {
				t.Fatalf("tlsCredentialCall(%s) = %v, want %v", test.name, got, test.want)
			}
		})
	}
}

func TestGRPCClientFieldsDoNotRequireClientType(t *testing.T) {
	expression, err := parser.ParseExpr("&Other{conn: conn, auth: auth}")
	if err != nil {
		t.Fatal(err)
	}
	fields := clientLiteralFields(expression)
	if fields["conn"] != "conn" || fields["auth"] != "auth" {
		t.Fatalf("clientLiteralFields() = %v, want conn and auth bindings", fields)
	}
}
