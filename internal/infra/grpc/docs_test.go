// Proof for the two claims docs/grpc/runtime-and-streaming.md makes about this package's code: the
// failure.Code to gRPC code table it publishes to service authors, and the list
// of gRPC codes it tells them that table cannot produce.
//
// Each is one half of a pair with [mappedStatus], and nothing in the toolchain
// relates them, so a new arm, a changed code, or a retired failure.Code would
// leave the published contract describing a transport that no longer answers
// that way. config_parity_test.go holds this package's other pair of owners to
// one answer for the same reason.
//
// The document's left column is read as Go identifiers taken from
// internal/failure's own source rather than as prose, so a renamed constant
// fails here instead of matching a string this file also had to spell.

package grpcx

import (
	"context"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/example/go-service-template-rest/internal/failure"
	"github.com/example/go-service-template-rest/internal/packagetest"
	"google.golang.org/grpc/status"
)

// documentedTableHeader is the row the published mapping table starts at. The
// scan runs from there to the first blank line, so the profile-marked row inside
// the table is read like any other and is removed with its profile.
const documentedTableHeader = "| `failure.Code` | gRPC code |"

func TestDocumentedFailureCodeTableMatchesStatusMapping(t *testing.T) {
	documented := documentedProblemCodes(t)
	if len(documented) == 0 {
		t.Fatalf("docs/grpc/runtime-and-streaming.md has no rows under %q", documentedTableHeader)
	}
	constantNames := failureCodeConstantNames(t)

	for code, name := range constantNames {
		want, published := documented[name]
		if !published {
			t.Errorf("docs/grpc/runtime-and-streaming.md publishes no gRPC code for %s", name)
			continue
		}
		delete(documented, name)
		if got := status.Code(mappedStatus(failure.Classification{Code: code}, "")); got.String() != want {
			t.Errorf("mappedStatus(%s) = %s, but docs/grpc/runtime-and-streaming.md publishes %s", name, got, want)
		}
	}

	for name, grpcCode := range documented {
		t.Errorf("docs/grpc/runtime-and-streaming.md maps %s to %s, which internal/failure no longer publishes", name, grpcCode)
	}
}

// documentedProblemCodes reads the published table as constant name to gRPC code
// name. One row may name several constants that share an answer, which is how
// the document groups them.
func documentedProblemCodes(t *testing.T) map[string]string {
	t.Helper()

	documented := make(map[string]string)
	inTable := false
	for line := range strings.SplitSeq(grpcGuide(t), "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == documentedTableHeader {
			inTable = true
			continue
		}
		if !inTable {
			continue
		}
		if trimmed == "" {
			break
		}
		names, grpcCode, isRow := parseDocumentedRow(trimmed)
		if !isRow {
			continue
		}
		for _, name := range names {
			documented[name] = grpcCode
		}
	}
	return documented
}

// parseDocumentedRow splits one table row into the failure.Code constants it
// names and the gRPC code they answer with. The column separator, a profile
// marker, and a row naming no constant are all reported as not a row.
func parseDocumentedRow(line string) ([]string, string, bool) {
	cells := strings.Split(strings.Trim(line, "|"), "|")
	if len(cells) != 2 {
		return nil, "", false
	}
	names := make([]string, 0, 2)
	for field := range strings.SplitSeq(cells[0], ",") {
		// A cell may also carry prose, as the unclassified fallback row does;
		// only the backticked constants are the document's claim about code.
		if name := strings.Trim(strings.TrimSpace(field), "`"); strings.HasPrefix(name, "Code") {
			names = append(names, name)
		}
	}
	grpcCode := strings.Trim(strings.TrimSpace(cells[1]), "`")
	if len(names) == 0 || grpcCode == "" {
		return nil, "", false
	}
	return names, grpcCode, true
}

// documentedUnreachableHeader is the line the published unreachable-code list
// starts at. The scan runs from there to the blank line after the list, the same
// shape [documentedTableHeader] uses.
const documentedUnreachableHeader = "The gRPC codes this table cannot produce:"

// TestDocumentedUnreachableCodesAreActuallyUnreachable holds the guide's second
// claim about this mapping to the same source the table is held to.
//
// It is a claim a reader acts on — it is what sends a service author to extend
// internal/failure rather than reach for status.Error — and it had already
// drifted once, naming AlreadyExists while the table two paragraphs above mapped
// CodeAlreadyExists to exactly that.
//
// Only this direction is checked. A code that becomes unreachable and is left
// off the list costs a reader nothing; one that becomes reachable and stays on
// it sends them to write a contract change they no longer need.
func TestDocumentedUnreachableCodesAreActuallyUnreachable(t *testing.T) {
	documented := documentedUnreachableCodes(t)
	if len(documented) == 0 {
		t.Fatalf("docs/grpc/runtime-and-streaming.md has no entries under %q", documentedUnreachableHeader)
	}

	answered := make(map[string]struct{})
	for code := range failureCodeConstantNames(t) {
		answered[status.Code(mappedStatus(failure.Classification{Code: code}, "")).String()] = struct{}{}
	}
	// The two the guide's own precedence paragraph publishes. They answer inside
	// [mapError] before any mapper runs, so no classification reaches them.
	for _, caller := range []error{context.Canceled, context.DeadlineExceeded} {
		mapped, _ := mapError(caller, ownedStatusOnly, errorRendering{})
		answered[status.Code(mapped).String()] = struct{}{}
	}

	for _, name := range documented {
		if _, reachable := answered[name]; reachable {
			t.Errorf("docs/grpc/runtime-and-streaming.md lists %s as unreachable, but this transport answers with it", name)
		}
	}
}

// documentedUnreachableCodes reads the published list as gRPC code names.
//
// The leading blank line between the header and the first item is skipped
// rather than ending the scan, which is the one way this list's shape differs
// from the table's.
func documentedUnreachableCodes(t *testing.T) []string {
	t.Helper()

	names := make([]string, 0, 4)
	inList := false
	for line := range strings.SplitSeq(grpcGuide(t), "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == documentedUnreachableHeader {
			inList = true
			continue
		}
		if !inList {
			continue
		}
		if trimmed == "" {
			if len(names) > 0 {
				break
			}
			continue
		}
		item, isItem := strings.CutPrefix(trimmed, "- ")
		if !isItem {
			break
		}
		names = append(names, strings.Trim(strings.TrimSpace(item), "`"))
	}
	return names
}

func grpcGuide(t *testing.T) string {
	t.Helper()

	document, err := os.ReadFile(filepath.Join("..", "..", "..", "docs", "grpc", "runtime-and-streaming.md"))
	if err != nil {
		t.Fatalf("read gRPC runtime leaf: %v", err)
	}
	return string(document)
}

// failureCodeConstantNames returns the constant name internal/failure declares
// for each published code, read out of that package's source.
//
// The join has to come from the source. The document publishes identifiers and
// failure constants publish values, so a rule this file invented to turn one into
// the other would be a third owner of the same naming decision — and it would
// keep agreeing right up until someone renamed a constant.
func failureCodeConstantNames(t *testing.T) map[failure.Code]string {
	t.Helper()

	directory := filepath.Join("..", "..", "failure")
	names := make(map[failure.Code]string)
	packagetest.EachGoFile(t, directory, parser.SkipObjectResolution, func(name string, parsed *ast.File, _ *token.FileSet) {
		if strings.HasSuffix(name, "_test.go") {
			return
		}
		ast.Inspect(parsed, func(node ast.Node) bool {
			spec, isValue := node.(*ast.ValueSpec)
			if !isValue || len(spec.Names) != len(spec.Values) {
				return true
			}
			// A grouped const block repeats the type on each spec that states
			// one, so an untyped or differently typed neighbour is skipped
			// rather than inherited.
			if named, isIdent := spec.Type.(*ast.Ident); !isIdent || named.Name != "Code" {
				return true
			}
			for index, value := range spec.Values {
				literal, isLiteral := value.(*ast.BasicLit)
				if !isLiteral || literal.Kind != token.STRING {
					continue
				}
				unquoted, err := strconv.Unquote(literal.Value)
				if err != nil {
					t.Fatalf("unquote %s in %s: %v", literal.Value, name, err)
				}
				names[failure.Code(unquoted)] = spec.Names[index].Name
			}
			return true
		})
	})
	return names
}
