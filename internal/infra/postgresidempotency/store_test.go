package postgresidempotency

import (
	"errors"
	"testing"
	"time"

	"github.com/example/go-service-template-rest/internal/httpidempotency"
)

func TestStoreRejectsMissingInputs(t *testing.T) {
	t.Parallel()

	if _, err := NewStore(nil, time.Hour); !errors.Is(err, ErrConfig) {
		t.Fatalf("NewStore(nil) error = %v, want ErrConfig", err)
	}
	store := &Store{}
	if _, _, err := store.Execute(t.Context(), httpidempotency.Request{}, nil); !errors.Is(err, ErrConfig) {
		t.Fatalf("Execute(invalid) error = %v, want ErrConfig", err)
	}
	if _, err := NewExecutor[struct{}, struct{}](nil, nil, httpidempotency.Codec[struct{}]{}); !errors.Is(err, ErrConfig) {
		t.Fatalf("NewExecutor(invalid) error = %v, want ErrConfig", err)
	}
}
