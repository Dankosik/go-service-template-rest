package sqlcgen

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
)

func TestGeneratedQueriesScanCutoverContractRows(t *testing.T) {
	t.Parallel()

	db := &fakeGeneratedDBTX{rowCount: 1}
	queries := New(db)

	queryValue := reflect.ValueOf(queries)
	queryType := queryValue.Type()
	for i := 0; i < queryValue.NumMethod(); i++ {
		method := queryType.Method(i)
		args := generatedMethodArgs(method.Type)

		beforeCalls := db.calls
		results := queryValue.Method(i).Call(args)
		if len(results) > 0 {
			if err := generatedMethodError(results); err != nil {
				t.Fatalf("%s() error = %v", method.Name, err)
			}
		}
		if method.Name == "WithTx" {
			continue
		}
		if db.calls != beforeCalls+1 {
			t.Fatalf("%s() DB calls = %d, want one additional call", method.Name, db.calls-beforeCalls)
		}
		if len(results) > 0 && results[0].Kind() == reflect.Slice && results[0].Len() != 1 {
			t.Fatalf("%s() returned %d rows, want one contract row", method.Name, results[0].Len())
		}
	}
	if queries.WithTx(nil) == nil {
		t.Fatal("WithTx(nil) returned nil Queries")
	}
}

func TestGeneratedQueriesPropagateDBTXFailures(t *testing.T) {
	t.Parallel()

	expected := errors.New("db unavailable")
	if _, err := New(&fakeGeneratedDBTX{queryErr: expected}).ClaimBillingOutbox(context.Background(), ClaimBillingOutboxParams{}); !errors.Is(err, expected) {
		t.Fatalf("ClaimBillingOutbox(query error) = %v, want %v", err, expected)
	}
	if _, err := New(&fakeGeneratedDBTX{rowErr: expected}).GetBillingAccountByScope(context.Background(), "user:1"); !errors.Is(err, expected) {
		t.Fatalf("GetBillingAccountByScope(row error) = %v, want %v", err, expected)
	}
	if _, err := New(&fakeGeneratedDBTX{rowCount: 1, scanErr: expected}).ListLedgerEntriesByAccount(context.Background(), ListLedgerEntriesByAccountParams{}); !errors.Is(err, expected) {
		t.Fatalf("ListLedgerEntriesByAccount(scan error) = %v, want %v", err, expected)
	}
	if _, err := New(&fakeGeneratedDBTX{rowCount: 1, rowsErr: expected}).ListReconciliationCasesByAccount(context.Background(), ListReconciliationCasesByAccountParams{}); !errors.Is(err, expected) {
		t.Fatalf("ListReconciliationCasesByAccount(rows error) = %v, want %v", err, expected)
	}
}

type fakeGeneratedDBTX struct {
	calls    int
	rowCount int
	queryErr error
	rowErr   error
	scanErr  error
	rowsErr  error
}

func (db *fakeGeneratedDBTX) Exec(context.Context, string, ...interface{}) (pgconn.CommandTag, error) {
	db.calls++
	return pgconn.CommandTag{}, nil
}

func (db *fakeGeneratedDBTX) Query(context.Context, string, ...interface{}) (pgx.Rows, error) {
	db.calls++
	if db.queryErr != nil {
		return nil, db.queryErr
	}
	return &fakeGeneratedRows{remaining: db.rowCount, scanErr: db.scanErr, rowsErr: db.rowsErr}, nil
}

func (db *fakeGeneratedDBTX) QueryRow(context.Context, string, ...interface{}) pgx.Row {
	db.calls++
	return fakeGeneratedRow{err: db.rowErr}
}

type fakeGeneratedRow struct {
	err error
}

func (row fakeGeneratedRow) Scan(dest ...any) error {
	if row.err != nil {
		return row.err
	}
	fillGeneratedScanDestinations(dest...)
	return nil
}

type fakeGeneratedRows struct {
	remaining int
	scanErr   error
	rowsErr   error
	closed    bool
}

func (rows *fakeGeneratedRows) Close() {
	rows.closed = true
}

func (rows *fakeGeneratedRows) Err() error {
	return rows.rowsErr
}

func (rows *fakeGeneratedRows) CommandTag() pgconn.CommandTag {
	return pgconn.CommandTag{}
}

func (rows *fakeGeneratedRows) FieldDescriptions() []pgconn.FieldDescription {
	return nil
}

func (rows *fakeGeneratedRows) Next() bool {
	if rows.remaining <= 0 {
		rows.Close()
		return false
	}
	rows.remaining--
	return true
}

func (rows *fakeGeneratedRows) Scan(dest ...any) error {
	if rows.scanErr != nil {
		return rows.scanErr
	}
	fillGeneratedScanDestinations(dest...)
	return nil
}

func (rows *fakeGeneratedRows) Values() ([]any, error) {
	return nil, nil
}

func (rows *fakeGeneratedRows) RawValues() [][]byte {
	return nil
}

func (rows *fakeGeneratedRows) Conn() *pgx.Conn {
	return nil
}

func generatedMethodArgs(methodType reflect.Type) []reflect.Value {
	args := make([]reflect.Value, 0, methodType.NumIn()-1)
	contextType := reflect.TypeOf((*context.Context)(nil)).Elem()
	for i := 1; i < methodType.NumIn(); i++ {
		argType := methodType.In(i)
		if argType == contextType {
			args = append(args, reflect.ValueOf(context.Background()))
			continue
		}
		args = append(args, reflect.Zero(argType))
	}
	return args
}

func generatedMethodError(results []reflect.Value) error {
	errorType := reflect.TypeOf((*error)(nil)).Elem()
	for _, result := range results {
		if result.Type().Implements(errorType) && !result.IsNil() {
			return result.Interface().(error)
		}
	}
	return nil
}

func fillGeneratedScanDestinations(destinations ...any) {
	for _, destination := range destinations {
		fillGeneratedValue(reflect.ValueOf(destination))
	}
}

func fillGeneratedValue(value reflect.Value) {
	if !value.IsValid() || value.IsNil() {
		return
	}
	if uuid, ok := value.Interface().(*pgtype.UUID); ok {
		*uuid = pgtype.UUID{Bytes: [16]byte{1, 2, 3, 4, 5, 6}, Valid: true}
		return
	}
	if ts, ok := value.Interface().(*pgtype.Timestamptz); ok {
		*ts = pgtype.Timestamptz{Time: time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC), Valid: true}
		return
	}

	elem := value.Elem()
	switch elem.Kind() {
	case reflect.Bool:
		elem.SetBool(true)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		elem.SetInt(1)
	case reflect.String:
		elem.SetString("value")
	case reflect.Slice:
		if elem.Type().Elem().Kind() == reflect.Uint8 {
			elem.SetBytes([]byte(`{"safe":"metadata"}`))
		}
	case reflect.Ptr:
		nested := reflect.New(elem.Type().Elem())
		elem.Set(nested)
		fillGeneratedValue(nested)
	}
}
