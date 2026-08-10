package failure

import (
	"errors"
	"fmt"
	"strings"
)

// classChainDepth bounds how far ClassChain walks. A chain this long is already
// past the point where another name helps, and the bound is what keeps a cyclic
// or pathological Unwrap from deciding the size of a log record.
const classChainDepth = 8

// opNameLimit bounds one [Op] name in a rendered chain. classChainDepth alone
// does not bound the record: eight links of unbounded text still is, and the
// two together are what keep a chain a fixed cost.
const opNameLimit = 64

// Op names the step err came from, so a record written at a sanitized boundary
// can say which step failed.
//
// It exists because [ClassChain] prints no message text, which leaves every
// fmt.Errorf layer rendering as "*fmt.wrapError": a handler that loads an author,
// renders a body, and stores a row produces the same record whichever of the
// three broke. Deriving the step from the text instead is not available — a
// wrapper's prefix cannot be told apart from a dependency's own, and pgx writes
// the DSN into one.
//
// The name must be a literal written in this repository. That, and not the type,
// is what makes it printable when err.Error() is not; interpolating a slug, an
// identifier, or anything else the request carried puts it in every log this
// boundary refused to leak into.
//
// fmt.Errorf stays correct everywhere else and needs no migration. It renders as
// its type, which is what every layer did before this existed.
func Op(name string, err error) error {
	if err == nil {
		return nil
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return err
	}
	return &opError{op: name, err: err}
}

type opError struct {
	op  string
	err error
}

func (e *opError) Error() string { return e.op + ": " + e.err.Error() }
func (e *opError) Unwrap() error { return e.err }

// ClassChain renders err's unwrap chain for the records a transport writes when
// it answers with a sanitized failure: the Go type of every layer, and the name
// of every layer built by [Op].
//
// The outermost type alone is not enough to act on: every wrapped failure in
// this repository goes through fmt.Errorf, so a record carrying only %T says
// "*fmt.wrapError" for a dead connection pool and for a nil map alike. The chain
// names the dependency underneath — "*fmt.wrapError -> *pgconn.PgError" is a
// database fault — and an operator can route on that.
//
// No message text is included, and that is the whole reason this exists rather
// than err.Error(). A handler's error can carry a credential, a DSN, or a token,
// and the canary tests in internal/infra/grpc assert that none of it reaches a
// log, span, or metric. The cost is real and worth stating: a chain built from
// errors.New renders as "*errors.errorString" and identifies nothing. A package
// whose failures must stay diagnosable publishes typed errors or sentinels,
// and why their faults survive this rendering. [Op] is what covers the rest.
//
// Joined errors are not expanded. errors.Join reports one type for the group,
// and walking every branch would turn a fan-out of cleanup failures into an
// unbounded record; the join's own type still says that is what happened.
func ClassChain(err error) string {
	if err == nil {
		return "nil"
	}

	classes := make([]string, 0, classChainDepth)
	for depth := 0; err != nil && depth < classChainDepth; depth++ {
		classes = append(classes, chainLink(err))
		err = errors.Unwrap(err)
	}
	if err != nil {
		classes = append(classes, "...")
	}
	return strings.Join(classes, " -> ")
}

// chainLink renders the layer err is, and never what it wraps.
//
// The type switch is deliberate where errors.As would be wrong: As searches the
// whole remaining chain, so an outer fmt.Errorf layer would render as the name of
// an [Op] several links below it and the chain would report that step twice.
func chainLink(err error) string {
	switch link := err.(type) { //nolint:errorlint // Rendering one layer is the whole contract; errors.As would search past it and repeat an inner Op's name on every layer above it.
	case *opError:
		// Byte length gates a rune slice: under the limit in bytes is under it in
		// runes too, so the common name costs no allocation, and the one that has
		// to be cut is not cut through a rune.
		if len(link.op) <= opNameLimit {
			return link.op
		}
		return string([]rune(link.op)[:opNameLimit]) + "…"
	default:
		return fmt.Sprintf("%T", err)
	}
}
