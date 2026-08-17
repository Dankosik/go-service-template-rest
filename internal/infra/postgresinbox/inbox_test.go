package postgresinbox

import (
	"context"
	"errors"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func TestClaimValidation(t *testing.T) {
	tests := []struct {
		name        string
		tx          pgx.Tx
		consumer    string
		message     string
		rows        int64
		wantClaim   bool
		wantErr     error
		wantQueries int
	}{
		{name: "nil transaction", consumer: "c", message: "m", wantErr: errInvalidInput},
		{name: "lower bounds", tx: &recordingTx{rows: 1}, consumer: "c", message: "m", wantClaim: true, wantQueries: 1},
		{name: "upper bounds", tx: &recordingTx{rows: 1}, consumer: strings.Repeat("c", maxConsumerIdentityBytes), message: strings.Repeat("m", maxMessageIDBytes), wantClaim: true, wantQueries: 1},
		{name: "duplicate", tx: &recordingTx{}, consumer: "c", message: "m", wantQueries: 1},
		{name: "empty consumer", tx: &recordingTx{}, message: "m", wantErr: errInvalidInput},
		{name: "empty message", tx: &recordingTx{}, consumer: "c", wantErr: errInvalidInput},
		{name: "consumer control", tx: &recordingTx{}, consumer: "bad\nconsumer", message: "m", wantErr: errInvalidInput},
		{name: "message C1 control", tx: &recordingTx{}, consumer: "c", message: "bad\u0085message", wantErr: errInvalidInput},
		{name: "consumer invalid UTF-8", tx: &recordingTx{}, consumer: string([]byte{0xff}), message: "m", wantErr: errInvalidInput},
		{name: "message invalid UTF-8", tx: &recordingTx{}, consumer: "c", message: string([]byte{0xff}), wantErr: errInvalidInput},
		{name: "consumer over limit", tx: &recordingTx{}, consumer: strings.Repeat("c", maxConsumerIdentityBytes+1), message: "m", wantErr: errInvalidInput},
		{name: "message over limit", tx: &recordingTx{}, consumer: "c", message: strings.Repeat("m", maxMessageIDBytes+1), wantErr: errInvalidInput},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			claimed, err := Claim(t.Context(), test.tx, test.consumer, test.message)
			if claimed != test.wantClaim || !errors.Is(err, test.wantErr) {
				t.Fatalf("Claim() = (%t, %v), want (%t, %v)", claimed, err, test.wantClaim, test.wantErr)
			}
			if tx, ok := test.tx.(*recordingTx); ok && tx.queries != test.wantQueries {
				t.Fatalf("queries = %d, want %d", tx.queries, test.wantQueries)
			}
		})
	}

	databaseErr := errors.New("database unavailable")
	tx := &recordingTx{err: databaseErr}
	if claimed, err := Claim(t.Context(), tx, "consumer", "message"); claimed || !errors.Is(err, databaseErr) {
		t.Fatalf("Claim(database error) = (%t, %v), want wrapped database error", claimed, err)
	}
}

func TestInboxSchemaHasNoExpirySurface(t *testing.T) {
	migration, err := os.ReadFile("../../../migrations/000002_postgres_inbox.sql")
	if err != nil {
		t.Fatalf("read inbox migration: %v", err)
	}
	query, err := os.ReadFile("../postgres/queries/postgres_inbox.sql")
	if err != nil {
		t.Fatalf("read inbox query: %v", err)
	}
	canonical := strings.ToLower(string(migration) + "\n" + string(query))
	for _, forbidden := range []string{"expires", "ttl", "cleanup", "delete from", "timestamp", "status", "attempt"} {
		if strings.Contains(canonical, forbidden) {
			t.Fatalf("inbox canonical SQL contains forbidden expiry/state surface %q", forbidden)
		}
	}
	if !strings.Contains(canonical, "primary key (consumer_identity, message_id)") ||
		!strings.Contains(canonical, "on conflict (consumer_identity, message_id) do nothing") {
		t.Fatal("inbox canonical SQL does not contain the two-key insert-on-conflict authority")
	}
}

type recordingTx struct {
	pgx.Tx

	rows    int64
	err     error
	queries int
}

func (tx *recordingTx) Exec(context.Context, string, ...any) (pgconn.CommandTag, error) {
	tx.queries++
	return pgconn.NewCommandTag("INSERT 0 " + strconv.FormatInt(tx.rows, 10)), tx.err
}
