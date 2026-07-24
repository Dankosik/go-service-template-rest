// Package openapi contains generated bindings for the reference service.
package openapi

//go:generate bash ../../../../scripts/run-go-tool.sh oapi-codegen -config oapi-codegen.yaml ../../api/openapi.yaml
//go:generate bash ../../../../scripts/run-go-tool.sh gofumpt -w openapi.gen.go
