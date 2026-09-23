//go:build ignore

package main

import (
	"go/parser"
	"go/token"
	"testing"
)

func TestCheckStartupMappingPenultimateConstruction(t *testing.T) {
	startup, err := parser.ParseFile(token.NewFileSet(), "startup.go", `package bootstrap
func initExample() {
	client, err := example.New(example.Config{Value: cfg.Value})
	if err != nil { return nil, err }
}`, 0)
	if err != nil {
		t.Fatal(err)
	}

	got := checkStartupMapping(startup, "startup.go", "example", "initExample", map[string]string{"Value": "cfg.Value"})
	if want := "startup.go: canonical startup flows=0, want 1"; got != want {
		t.Fatalf("checkStartupMapping() = %q, want %q", got, want)
	}
}

func TestCheckStartupMappingThreeStatementFlow(t *testing.T) {
	startup, err := parser.ParseFile(token.NewFileSet(), "startup.go", `package bootstrap
func initExample() {
	client, err := example.New(example.Config{Value: cfg.Value})
	if err != nil { return nil, err }
	return client, nil
}`, 0)
	if err != nil {
		t.Fatal(err)
	}

	if got := checkStartupMapping(startup, "startup.go", "example", "initExample", map[string]string{"Value": "cfg.Value"}); got != "" {
		t.Fatalf("checkStartupMapping() = %q, want no diagnostic", got)
	}
}
