package postgresjobs

import "errors"

var (
	ErrConfig             = errors.New("postgres jobs configuration")
	ErrSchemaIncompatible = errors.New("postgres jobs schema incompatible")
	ErrOperationTimeout   = errors.New("postgres jobs operation timeout")
	ErrSessionTerminal    = errors.New("postgres jobs control Session terminal")
	ErrUnknownVocabulary  = errors.New("postgres jobs database vocabulary unknown")
)
