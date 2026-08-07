package natsjs

import "errors"

// Every failure this package produces wraps one of the five sentinels below,
// and both composition roots route on them rather than on message text. The
// "Errors" section of doc.go owns what each one means to a caller and is the
// one place that states it; this file exists so a reader looking for the set
// has somewhere to find it.
var (
	ErrRejected  = errors.New("messaging operation rejected")
	ErrAmbiguous = errors.New("messaging operation outcome ambiguous")
	ErrCapacity  = errors.New("messaging producer capacity exhausted")
	ErrDraining  = errors.New("messaging runtime draining")
	ErrTerminal  = errors.New("messaging runtime terminal failure")
)
