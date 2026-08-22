// Package openapi contains generated bindings for the reference service.
package openapi

//go:generate go tool -modfile=../../../../tools/go.mod oapi-codegen -config oapi-codegen.yaml ../../api/openapi.yaml
//go:generate go tool -modfile=../../../../tools/go.mod gofumpt -w openapi.gen.go
