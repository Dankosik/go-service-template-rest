package objectstorage

import (
	"strings"
	"testing"
)

func TestPortContractAndKeyGrammar(t *testing.T) {
	if got := Kind(NewError(ErrorKind("provider-secret"))); got != KindInternal {
		t.Fatalf("Kind(NewError(unknown)) = %q, want %q", got, KindInternal)
	}

	tests := []struct {
		name  string
		key   string
		valid bool
	}{
		{name: "allowed ASCII", key: "letters-012_~.AZ/path", valid: true},
		{name: "one byte", key: "a", valid: true},
		{name: "maximum bytes", key: strings.Repeat("a", 1024), valid: true},
		{name: "wire escaping characters", key: "space~percent%question?hash#", valid: false},
		{name: "empty", key: ""},
		{name: "too long", key: strings.Repeat("a", 1025)},
		{name: "unicode", key: "caf\u00e9"},
		{name: "case-sensitive soap", key: "soap"},
		{name: "soap is case sensitive", key: "SOAP", valid: true},
		{name: "dot segment", key: "a/./b"},
		{name: "dot-dot segment", key: "a/../b"},
		{name: "empty segment", key: "a//b"},
		{name: "leading slash", key: "/a"},
		{name: "trailing slash", key: "a/"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := ValidateKey(test.key)
			if test.valid {
				if err != nil {
					t.Fatalf("ValidateKey(%q) error = %v", test.key, err)
				}
				return
			}
			if Kind(err) != KindInvalid {
				t.Fatalf("Kind(ValidateKey(%q)) = %q, want %q", test.key, Kind(err), KindInvalid)
			}
		})
	}
}

func FuzzValidateKey(f *testing.F) {
	for _, key := range []string{"a", "a/b", "soap", "a//b", "caf\u00e9", strings.Repeat("a", 1024)} {
		f.Add(key)
	}
	f.Fuzz(func(t *testing.T, key string) {
		err := ValidateKey(key)
		if err != nil && Kind(err) != KindInvalid {
			t.Fatalf("Kind(ValidateKey(%q)) = %q, want %q", key, Kind(err), KindInvalid)
		}
	})
}
