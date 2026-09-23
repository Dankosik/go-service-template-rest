//go:build ignore

package main

import (
	"go/parser"
	"go/token"
	"testing"
)

func TestReturnedClientFieldsRequireExplicitClientPointer(t *testing.T) {
	for _, test := range []struct {
		name string
		want bool
	}{
		{name: "&Client{generated: generated, transport: transport}", want: true},
		{name: "&Other{generated: generated, transport: transport}"},
		{name: "Client{generated: generated, transport: transport}"},
	} {
		t.Run(test.name, func(t *testing.T) {
			expression, err := parser.ParseExpr(test.name)
			if err != nil {
				t.Fatal(err)
			}
			fields := returnedClientFields(expression)
			if got := fields["generated"] == "generated" && fields["transport"] == "transport"; got != test.want {
				t.Fatalf("returnedClientFields(%s) = %v, want %v", test.name, got, test.want)
			}
		})
	}
}

func TestCountForbiddenCallsScansWholeFile(t *testing.T) {
	file, err := parser.ParseFile(token.NewFileSet(), "adapter.go", `package adapter
var eager = transport.NewExternalHTTPS()
func unrelated() {
	transport.NewExternalHTTPS()
	other.NewExternalHTTPS()
	transport.NewPrivateHTTPS()
}`, 0)
	if err != nil {
		t.Fatal(err)
	}
	if got := countForbiddenCalls(file, "transport", "NewExternalHTTPS"); got != 2 {
		t.Fatalf("countForbiddenCalls() = %d, want 2", got)
	}
}
