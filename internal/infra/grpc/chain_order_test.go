// Proof that the interceptor order doc.go publishes is the order the chain
// actually runs.
//
// doc.go calls itself the one prose owner of that order, and its own header says
// the comments in this package carry more of the design than the code does. That
// is exactly the pair nothing in the toolchain relates: builtinPolicies names its
// entries, assembleChain fixes what surrounds them, and the published list is
// prose in another file. A position inserted on one side stays invisible on the
// other, and the reader who trusts the list is the one who pays.

package grpcx

import (
	"log/slog"
	"os"
	"slices"
	"strings"
	"testing"
	"time"
)

// documentedOrderHeading starts the published list. The scan runs from there to
// the first line that is neither a bullet nor a bullet's continuation.
const documentedOrderHeading = "// # Interceptor order"

// The three positions assembleChain contributes rather than builtinPolicies: the
// outermost interceptor, the composition root's slot, and the innermost boundary.
// None carries a name constant, because nothing indexes them the way
// builtinPolicy.name indexes the rest, so doc.go's wording for each is pinned
// here instead.
const (
	documentedCorrelation     = "correlation"
	documentedSuppliedPolicy  = "[Options.UnaryPolicy] and [Options.StreamPolicy]"
	documentedHandlerBoundary = "handler error boundary"
)

func TestDocumentedInterceptorOrderMatchesTheChain(t *testing.T) {
	t.Parallel()

	assembled := []string{documentedCorrelation}
	for _, policy := range builtinPolicies(
		slog.New(slog.DiscardHandler),
		accessLogPolicy{},
		newAdmissionPolicy(1, 1, noopLoadRecorder{}),
		time.Second,
	) {
		assembled = append(assembled, strings.ReplaceAll(policy.name, "_", " "))
	}
	assembled = append(assembled, documentedSuppliedPolicy, documentedHandlerBoundary)

	documented := documentedInterceptorOrder(t)
	if !slices.Equal(documented, assembled) {
		t.Fatalf(
			"doc.go publishes the interceptor order\n\t%v\nbut the chain assembles\n\t%v",
			documented, assembled,
		)
	}
}

// documentedInterceptorOrder reads the position names out of doc.go's published
// list, outermost first. A bullet names its position ahead of the em dash that
// introduces what the position buys.
func documentedInterceptorOrder(t *testing.T) []string {
	t.Helper()

	document, err := os.ReadFile("doc.go")
	if err != nil {
		t.Fatalf("read package documentation: %v", err)
	}
	lines := strings.Split(string(document), "\n")
	start := slices.Index(lines, documentedOrderHeading)
	if start < 0 {
		t.Fatalf("doc.go has no %q section", documentedOrderHeading)
	}

	order := make([]string, 0)
	for _, line := range lines[start+1:] {
		bullet, isBullet := strings.CutPrefix(line, "//   - ")
		if !isBullet {
			// A bullet's own continuation is indented past the marker. Anything
			// else, once the list has started, has ended it.
			if len(order) > 0 && !strings.HasPrefix(line, "//     ") {
				break
			}
			continue
		}
		name, _, described := strings.Cut(bullet, " — ")
		if !described {
			t.Fatalf("doc.go bullet %q names no position ahead of an em dash", bullet)
		}
		order = append(order, name)
	}
	if len(order) == 0 {
		t.Fatalf("doc.go's %q section lists no positions", documentedOrderHeading)
	}
	return order
}
