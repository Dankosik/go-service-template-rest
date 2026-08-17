package postgresjobs

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/example/go-service-template-rest/internal/infra/postgres"
)

func TestStoreRejectsInvalidConstructionWithoutAcquiring(t *testing.T) {
	t.Parallel()
	options := StoreOptions{OperationTimeout: time.Second, StatementTimeout: time.Second}
	for _, test := range []struct {
		name    string
		pool    *postgres.Pool
		options StoreOptions
		want    string
	}{
		{name: "nil pool", options: options, want: "postgres pool"},
		{name: "zero pool", pool: &postgres.Pool{}, options: options, want: "postgres pool"},
		{name: "missing operation timeout", pool: &postgres.Pool{}, options: StoreOptions{StatementTimeout: time.Second}, want: "operation timeout"},
		{name: "missing statement timeout", pool: &postgres.Pool{}, options: StoreOptions{OperationTimeout: time.Second}, want: "statement timeout"},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			store, err := NewStore(test.pool, test.options)
			if store != nil || !errors.Is(err, ErrConfig) || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("NewStore() = %v, %v, want nil ErrConfig containing %q", store, err, test.want)
			}
		})
	}
}

func TestStoreZeroValueFailsClosed(t *testing.T) {
	t.Parallel()
	store := &Store{}
	if _, err := store.AcquireSession(context.Background()); !errors.Is(err, ErrConfig) {
		t.Fatalf("AcquireSession() error = %v, want ErrConfig", err)
	}
	if err := store.CheckSchema(context.Background()); !errors.Is(err, ErrConfig) {
		t.Fatalf("CheckSchema() error = %v, want ErrConfig", err)
	}
}

func TestSchemaMismatchKeepsSchemaCause(t *testing.T) {
	t.Parallel()
	err := schemaMismatch("columns", []string{"actual"}, []string{"expected"})
	if !errors.Is(err, ErrSchemaIncompatible) || !strings.Contains(err.Error(), "columns") {
		t.Fatalf("schemaMismatch() = %v, want ErrSchemaIncompatible naming columns", err)
	}
}
