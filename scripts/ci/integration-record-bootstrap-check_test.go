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

	err = checkStartupMapping(startup, "startup.go", "example", "initExample", map[string]string{"Value": "cfg.Value"})
	if want := "startup.go: canonical startup flows=0, want 1"; err == nil || err.Error() != want {
		t.Fatalf("checkStartupMapping() = %v, want %q", err, want)
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

	if err := checkStartupMapping(startup, "startup.go", "example", "initExample", map[string]string{"Value": "cfg.Value"}); err != nil {
		t.Fatalf("checkStartupMapping() = %v, want no diagnostic", err)
	}
}
