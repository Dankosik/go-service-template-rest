package httpclient

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestGeneratedClientComposition(t *testing.T) {
	packageDirectory, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	repositoryRoot := filepath.Clean(filepath.Join(packageDirectory, "..", "..", ".."))
	generatedDirectory, err := os.MkdirTemp(packageDirectory, "generatedclientcheck-") //nolint:usetesting // Must stay inside the module.
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(generatedDirectory) })

	// #nosec G204 -- fixed Go executable with repository-owned and generated paths, never caller input.
	generate := exec.CommandContext(t.Context(), "go", "tool", "-modfile="+filepath.Join(repositoryRoot, "tools", "go.mod"),
		"oapi-codegen", "-generate", "types,client", "-package", "generatedclient", "-o",
		filepath.Join(generatedDirectory, "client.gen.go"), filepath.Join(packageDirectory, "testdata", "generated-client.yaml"))
	generate.Dir = repositoryRoot
	if output, err := generate.CombinedOutput(); err != nil {
		t.Fatalf("generate client: %v\n%s", err, output)
	}

	consumer := `package generatedclient

import (
	"errors"
	"testing"

	"github.com/example/go-service-template-rest/internal/infra/httpclient"
)

func TestUsesFixedHTTPClient(t *testing.T) {
	bounded, err := httpclient.NewExternalHTTPS("https://localhost:443")
	if err != nil { t.Fatal(err) }
	t.Cleanup(bounded.CloseIdleConnections)
	var _ HttpRequestDoer = bounded
	client, err := NewClient(bounded.BaseURL(), WithHTTPClient(bounded))
	if err != nil { t.Fatal(err) }
	_, err = client.GetHealth(t.Context())
	if !errors.Is(err, httpclient.ErrTargetDenied) { t.Fatalf("GetHealth() error = %v", err) }
}
`
	if err := os.WriteFile(filepath.Join(generatedDirectory, "composition_test.go"), []byte(consumer), 0o600); err != nil {
		t.Fatal(err)
	}
	relativePackage, err := filepath.Rel(repositoryRoot, generatedDirectory)
	if err != nil {
		t.Fatal(err)
	}
	// #nosec G204 -- fixed Go executable with a package path derived from the test-owned temporary directory.
	testCommand := exec.CommandContext(t.Context(), "go", "test", "-vet=off", "./"+filepath.ToSlash(relativePackage), "-run", "^TestUsesFixedHTTPClient$", "-count=1")
	testCommand.Dir = repositoryRoot
	if output, err := testCommand.CombinedOutput(); err != nil {
		t.Fatalf("generated composition: %v\n%s", err, output)
	}
	entries, err := os.ReadDir(generatedDirectory)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 {
		var names []string
		for _, entry := range entries {
			names = append(names, entry.Name())
		}
		t.Fatalf("generated files = %s", strings.Join(names, ", "))
	}
}
