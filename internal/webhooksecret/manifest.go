package webhooksecret

import "crypto/sha256"

const (
	// MaxManifestBytes bounds one secret manifest document.
	MaxManifestBytes = 1 << 20
	// MaxManifestEntries bounds the entries in one secret manifest document.
	MaxManifestEntries = 4096
)

// Bindings enforces that one secret signs for at most one principal. The zero
// value is ready to use.
type Bindings[P comparable] struct {
	principals map[[sha256.Size]byte]P
}

// Bind records secret for principal and reports false when a different
// principal already holds the same secret bytes.
func (b *Bindings[P]) Bind(secret []byte, principal P) bool {
	digest := sha256.Sum256(secret)
	if previous, exists := b.principals[digest]; exists && previous != principal {
		return false
	}
	if b.principals == nil {
		b.principals = make(map[[sha256.Size]byte]P)
	}
	b.principals[digest] = principal
	return true
}
