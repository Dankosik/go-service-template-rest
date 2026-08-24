//go:build ignore

package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"go.yaml.in/yaml/v3"
)

func main() {
	args := os.Args[1:]
	if len(args) > 0 && args[0] == "--" {
		args = args[1:]
	}
	if len(args) != 1 {
		fmt.Fprintln(os.Stderr, "usage: openapi-ref-check CONTRACT")
		os.Exit(2)
	}

	contract, err := os.Open(args[0])
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", args[0], err)
		os.Exit(1)
	}
	defer contract.Close()

	decoder := yaml.NewDecoder(contract)
	var document yaml.Node
	if err := decoder.Decode(&document); err != nil {
		fmt.Fprintf(os.Stderr, "%s: decode OpenAPI contract: %v\n", args[0], err)
		os.Exit(1)
	}
	var extra yaml.Node
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		if err == nil {
			err = errors.New("multiple YAML documents are not supported")
		}
		fmt.Fprintf(os.Stderr, "%s: decode OpenAPI contract: %v\n", args[0], err)
		os.Exit(1)
	}
	if err := validateRefs(&document, map[*yaml.Node]bool{}); err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", args[0], err)
		os.Exit(1)
	}
}

func validateRefs(node *yaml.Node, seen map[*yaml.Node]bool) error {
	if node == nil || seen[node] {
		return nil
	}
	seen[node] = true
	if node.Kind == yaml.MappingNode {
		for index := 0; index+1 < len(node.Content); index += 2 {
			key, value := node.Content[index], node.Content[index+1]
			if mappingKey(key, map[*yaml.Node]bool{}) == "$ref" && (value.Kind != yaml.ScalarNode || !strings.HasPrefix(value.Value, "#")) {
				return fmt.Errorf("line %d: external OpenAPI $ref is not supported", value.Line)
			}
		}
	}
	if node.Alias != nil {
		if err := validateRefs(node.Alias, seen); err != nil {
			return err
		}
	}
	for _, child := range node.Content {
		if err := validateRefs(child, seen); err != nil {
			return err
		}
	}
	return nil
}

func mappingKey(node *yaml.Node, seen map[*yaml.Node]bool) string {
	if node == nil || seen[node] {
		return ""
	}
	seen[node] = true
	if node.Kind == yaml.AliasNode {
		return mappingKey(node.Alias, seen)
	}
	if node.Kind == yaml.ScalarNode {
		return node.Value
	}
	return ""
}
