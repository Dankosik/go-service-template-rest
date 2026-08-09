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
		t.Fatalf("os.Getwd() error = %v", err)
	}
	repositoryRoot := filepath.Clean(filepath.Join(packageDirectory, "..", "..", ".."))
	// The child package must stay below the module root so Go internal-package
	// visibility and the generated import path match a real consumer.
	generatedDirectory, err := os.MkdirTemp( //nolint:usetesting // t.TempDir cannot select an in-module parent.
		packageDirectory,
		"generatedclientcheck-",
	)
	if err != nil {
		t.Fatalf("os.MkdirTemp() error = %v", err)
	}
	t.Cleanup(func() {
		if err := os.RemoveAll(generatedDirectory); err != nil {
			t.Errorf("remove generated client fixture: %v", err)
		}
	})

	generatedPath := filepath.Join(generatedDirectory, "client.gen.go")
	generate := exec.CommandContext(
		t.Context(),
		"bash",
		filepath.Join(repositoryRoot, "scripts", "run-go-tool.sh"),
		"oapi-codegen",
		"-generate",
		"types,client",
		"-package",
		"generatedclient",
		"-o",
		generatedPath,
		filepath.Join(packageDirectory, "testdata", "generated-client.yaml"),
	)
	generate.Dir = repositoryRoot
	if output, err := generate.CombinedOutput(); err != nil {
		t.Fatalf("generate pinned oapi-codegen client: %v\n%s", err, output)
	}

	const consumerTest = `package generatedclient

import (
	"errors"
	"testing"
	"time"

	"github.com/example/go-service-template-rest/internal/infra/httpclient"
)

func TestGeneratedClientUsesBoundedHTTPClient(t *testing.T) {
	cfg := httpclient.Config{
		DependencyName:         "generated-fixture",
		BaseURL:                "https://localhost:443",
		TargetClass:            httpclient.ExternalHTTPS,
		RequestTimeout:         time.Second,
		ResponseHeaderTimeout:  time.Second,
		MaxResponseHeaderBytes: 16 << 10,
		MaxResponseBodyBytes:   1 << 20,
		MaxConnsPerHost:        2,
		Propagation:            httpclient.PropagationTrustedService,
	}
	bounded, err := httpclient.New(cfg, nil)
	if err != nil {
		t.Fatalf("httpclient.New() error = %v", err)
	}
	t.Cleanup(bounded.CloseIdleConnections)

	var _ HttpRequestDoer = bounded
	client, err := NewClient(bounded.BaseURL(), WithHTTPClient(bounded))
	if err != nil {
		t.Fatalf("generated NewClient() error = %v", err)
	}
	_, err = client.GetHealth(t.Context())
	if !errors.Is(err, httpclient.ErrTargetDenied) {
		t.Fatalf("generated GetHealth() error = %v, want shared client address denial", err)
	}
}
`
	consumerPath := filepath.Join(generatedDirectory, "composition_test.go")
	if err := os.WriteFile(consumerPath, []byte(consumerTest), 0o600); err != nil {
		t.Fatalf("write generated consumer test: %v", err)
	}

	relativePackage, err := filepath.Rel(repositoryRoot, generatedDirectory)
	if err != nil {
		t.Fatalf("relative generated package path: %v", err)
	}
	testCommand := exec.CommandContext(
		t.Context(),
		"go",
		"test",
		"-vet=off",
		"./"+filepath.ToSlash(relativePackage),
		"-run",
		"^TestGeneratedClientUsesBoundedHTTPClient$",
		"-count=1",
	)
	testCommand.Dir = repositoryRoot
	if output, err := testCommand.CombinedOutput(); err != nil {
		t.Fatalf("compile and run generated client composition: %v\n%s", err, output)
	}

	entries, err := os.ReadDir(generatedDirectory)
	if err != nil {
		t.Fatalf("read generated directory: %v", err)
	}
	if len(entries) != 2 {
		var names []string
		for _, entry := range entries {
			names = append(names, entry.Name())
		}
		t.Fatalf("generated fixture files = %s, want generated code and one consumer test", strings.Join(names, ", "))
	}
}
