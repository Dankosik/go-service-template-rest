package logctx

import (
	"fmt"
	"runtime"
)

// PanicAttrs returns the log attributes describing a recovered panic: its class,
// its concrete type, and the stack it came from.
//
// Recovery sites answer their callers differently but must describe the panic
// identically. stack is a parameter because only the recovering deferred
// function still has the panicking goroutine's frames.
//
// Only the panic's type is published, never its value. A panic raised while
// parsing a token, a provider document, or a request body can carry that input
// in its message, and a log is not a safer place for it than a response is.
func PanicAttrs(recovered any, stack []byte) []any {
	return []any{
		"panic.class", PanicClass(recovered),
		"panic.type", fmt.Sprintf("%T", recovered),
		"stack", string(stack),
	}
}

// PanicClass returns the closed class operators group recovered panics by.
func PanicClass(recovered any) string {
	switch recovered.(type) {
	case nil:
		return "none"
	case runtime.Error:
		return "runtime_error"
	case error:
		return "error"
	case string:
		return "string"
	default:
		return "value"
	}
}
