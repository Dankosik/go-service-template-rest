package postgresidempotency

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/example/go-service-template-rest/internal/packagetest"
)

func TestDriverBoundaryMatchesStoreFiles(t *testing.T) {
	configuration, err := os.ReadFile(filepath.Join("..", "..", "..", ".golangci.yml"))
	if err != nil {
		t.Fatalf("read linter configuration: %v", err)
	}
	const exemption = `- "!**/internal/infra/postgresidempotency/store*.go"`
	if got := strings.Count(string(configuration), exemption); got != 2 {
		t.Fatalf("idempotency Store exemption count = %d, want 2", got)
	}

	sawDriver := false
	packagetest.EachGoFile(t, ".", parser.ImportsOnly, func(name string, parsed *ast.File, _ *token.FileSet) {
		if strings.HasSuffix(name, "_test.go") {
			return
		}
		for _, imported := range parsed.Imports {
			path := strings.Trim(imported.Path.Value, `"`)
			if !strings.HasPrefix(path, "github.com/jackc/pgx/v5") &&
				path != "github.com/example/go-service-template-rest/internal/infra/postgres/sqlcgen" {
				continue
			}
			sawDriver = true
			if !strings.HasPrefix(name, "store") {
				t.Errorf("%s imports %s outside the Store boundary", name, path)
			}
		}
	})
	if !sawDriver {
		t.Fatal("no Store file imports the PostgreSQL driver or generated queries")
	}
}
