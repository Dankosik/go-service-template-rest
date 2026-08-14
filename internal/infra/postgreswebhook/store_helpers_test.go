package postgreswebhook

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/example/go-service-template-rest/internal/infra/postgres/sqlcgen"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
)

type webhookQueryStub struct {
	execErrors []error
	execTags   []pgconn.CommandTag
	queryRows  []pgx.Rows
	rowResults []pgx.Row
}

func (stub *webhookQueryStub) Exec(context.Context, string, ...any) (pgconn.CommandTag, error) {
	if len(stub.execErrors) == 0 {
		if len(stub.execTags) == 0 {
			return pgconn.CommandTag{}, nil
		}
		tag := stub.execTags[0]
		stub.execTags = stub.execTags[1:]
		return tag, nil
	}
	err := stub.execErrors[0]
	stub.execErrors = stub.execErrors[1:]
	return pgconn.CommandTag{}, err
}

//nolint:ireturn // sqlcgen.DBTX requires pgx.Rows.
func (stub *webhookQueryStub) Query(context.Context, string, ...any) (pgx.Rows, error) {
	if len(stub.queryRows) > 0 {
		rows := stub.queryRows[0]
		stub.queryRows = stub.queryRows[1:]
		return rows, nil
	}
	return nil, errors.New("unexpected query")
}

//nolint:ireturn // sqlcgen.DBTX requires pgx.Row.
func (stub *webhookQueryStub) QueryRow(context.Context, string, ...any) pgx.Row {
	if len(stub.rowResults) > 0 {
		row := stub.rowResults[0]
		stub.rowResults = stub.rowResults[1:]
		return row
	}
	return webhookRowStub{err: errors.New("unexpected query row")}
}

type webhookRowStub struct{ err error }

func (stub webhookRowStub) Scan(...any) error { return stub.err }

type webhookRowsStub struct {
	pgx.Rows

	rows  []pgx.Row
	index int
}

func (stub *webhookRowsStub) Close() {}

func (stub *webhookRowsStub) Err() error { return nil }

func (stub *webhookRowsStub) Next() bool {
	if stub.index >= len(stub.rows) {
		return false
	}
	stub.index++
	return true
}

func (stub *webhookRowsStub) Scan(destinations ...any) error {
	if err := stub.rows[stub.index-1].Scan(destinations...); err != nil {
		return fmt.Errorf("scan webhook row: %w", err)
	}
	return nil
}

type webhookScanRow func(...any) error

func (row webhookScanRow) Scan(destinations ...any) error { return row(destinations...) }

func setScanValue(destinations []any, index int, value any) error {
	if index >= len(destinations) {
		return errors.New("unexpected scan destination")
	}
	switch destination := destinations[index].(type) {
	case *string:
		value, ok := value.(string)
		if !ok || destination == nil {
			return errors.New("unexpected string scan destination")
		}
		*destination = value
	case *[]byte:
		value, ok := value.([]byte)
		if !ok || destination == nil {
			return errors.New("unexpected bytes scan destination")
		}
		*destination = value
	case *int64:
		value, ok := value.(int64)
		if !ok || destination == nil {
			return errors.New("unexpected int64 scan destination")
		}
		*destination = value
	case *int32:
		value, ok := value.(int32)
		if !ok || destination == nil {
			return errors.New("unexpected int32 scan destination")
		}
		*destination = value
	case *bool:
		value, ok := value.(bool)
		if !ok || destination == nil {
			return errors.New("unexpected bool scan destination")
		}
		*destination = value
	case *pgtype.Timestamptz:
		value, ok := value.(pgtype.Timestamptz)
		if !ok || destination == nil {
			return errors.New("unexpected time scan destination")
		}
		*destination = value
	default:
		return errors.New("unsupported scan destination")
	}
	return nil
}

type scanValue struct {
	index int
	value any
}

func setScanValues(destinations []any, values ...scanValue) error {
	for _, value := range values {
		if err := setScanValue(destinations, value.index, value.value); err != nil {
			return err
		}
	}
	return nil
}

//nolint:ireturn // sqlcgen.DBTX requires pgx.Row.
func webhookActionDeliveryRow(now time.Time, summary string) pgx.Row {
	return webhookScanRow(func(destinations ...any) error {
		return setScanValues(destinations,
			scanValue{0, int64(2)}, scanValue{1, string(DeliveryTerminal)}, scanValue{2, summary},
			scanValue{3, pgtime(now.Add(time.Hour))}, scanValue{4, "destination"}, scanValue{5, int64(1)},
			scanValue{6, false}, scanValue{7, string(OutcomeAttemptsExhausted)},
		)
	})
}

func TestResolveAcceptanceRejectsAbsentRecord(t *testing.T) {
	prepared, err := PrepareAcceptance(goldenAcceptance())
	if err != nil {
		t.Fatal(err)
	}
	queries := sqlcgen.New(&webhookQueryStub{rowResults: []pgx.Row{webhookRowStub{err: pgx.ErrNoRows}, webhookRowStub{err: pgx.ErrNoRows}}})
	receipt, err := resolveAcceptance(t.Context(), queries, prepared)
	if err != nil || receipt.Disposition != AcceptanceRejected || receipt.AcceptanceID != prepared.Acceptance.AcceptanceID {
		t.Fatalf("resolveAcceptance() = %+v, %v", receipt, err)
	}
}

func TestResolveAcceptanceReadbackReturnsDeliveryReceipt(t *testing.T) {
	prepared, err := PrepareAcceptance(goldenAcceptance())
	if err != nil {
		t.Fatal(err)
	}
	acceptedAt := time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC)
	stub := &webhookQueryStub{
		rowResults: []pgx.Row{webhookRowStub{err: pgx.ErrNoRows}, webhookScanRow(func(destinations ...any) error {
			return setScanValues(destinations,
				scanValue{0, prepared.Acceptance.BusinessEventID}, scanValue{1, prepared.Acceptance.AcceptanceID},
				scanValue{2, prepared.Acceptance.FanoutSnapshotID}, scanValue{3, prepared.Fingerprint[:]},
				scanValue{4, int32(len(prepared.Destinations))}, scanValue{5, prepared.Fingerprint[:]},
				scanValue{6, pgtime(acceptedAt)},
			)
		})},
		queryRows: []pgx.Rows{&webhookRowsStub{rows: []pgx.Row{webhookScanRow(func(destinations ...any) error {
			return setScanValues(destinations,
				scanValue{0, prepared.Destinations[0].DeliveryID}, scanValue{1, prepared.Destinations[0].DestinationID},
				scanValue{2, prepared.Destinations[0].Generation}, scanValue{3, int64(0)},
			)
		})}}},
	}
	receipt, err := resolveAcceptance(t.Context(), sqlcgen.New(stub), prepared)
	if err != nil || receipt.Disposition != AcceptanceAccepted || len(receipt.DeliveryIDs) != 1 || receipt.DeliveryIDs[0] != prepared.Destinations[0].DeliveryID || !receipt.AcceptedAt.Equal(acceptedAt) {
		t.Fatalf("resolveAcceptance() = %+v, %v", receipt, err)
	}
}

func TestStoreQueryHelpersPropagateDatabaseFailures(t *testing.T) {
	prepared, err := PrepareAcceptance(goldenAcceptance())
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name       string
		rowResults []pgx.Row
		want       AcceptanceDisposition
	}{
		{name: "tombstone", rowResults: []pgx.Row{webhookRowStub{err: errors.New("read tombstone")}}},
		{name: "acceptance", rowResults: []pgx.Row{webhookRowStub{err: pgx.ErrNoRows}, webhookRowStub{err: errors.New("read acceptance")}}, want: AcceptanceUnknown},
	} {
		t.Run(test.name, func(t *testing.T) {
			receipt, err := resolveAcceptance(t.Context(), sqlcgen.New(&webhookQueryStub{rowResults: test.rowResults}), prepared)
			if err == nil || receipt.Disposition != test.want {
				t.Fatalf("resolveAcceptance() = %+v, %v", receipt, err)
			}
		})
	}

	if err := lockAcceptance(t.Context(), sqlcgen.New(&webhookQueryStub{}), "owner", "event"); err != nil {
		t.Fatalf("lockAcceptance() error = %v", err)
	}
	if err := lockAcceptance(t.Context(), sqlcgen.New(&webhookQueryStub{execErrors: []error{errors.New("lock")}}), "owner", "event"); err == nil {
		t.Fatal("lockAcceptance() error = nil")
	}
	if err := insertAndMatchDestination(t.Context(), sqlcgen.New(&webhookQueryStub{execErrors: []error{errors.New("insert")}}), "owner", prepared.Destinations[0], time.Now(), 1); err == nil {
		t.Fatal("insertAndMatchDestination() error = nil")
	}
	destination := prepared.Destinations[0]
	readback := webhookScanRow(func(destinations ...any) error {
		encoded, encodeErr := encodeDeliveryPolicy(destination.Policy)
		if encodeErr != nil {
			return encodeErr
		}
		digest := canonicalDigest(encoded)
		return setScanValues(destinations,
			scanValue{3, destination.OwnershipVerificationReceipt}, scanValue{4, destination.URL},
			scanValue{5, destination.SelectionRevision}, scanValue{6, destination.PayloadVersionPreference},
			scanValue{7, destination.SignatureProfile}, scanValue{8, destination.SigningAuthorityBinding},
			scanValue{10, digest[:]}, scanValue{11, int32(destination.Policy.DestinationConcurrency)},
			scanValue{12, int32(destination.Policy.GlobalConcurrency)},
		)
	})
	if err := insertAndMatchDestination(t.Context(), sqlcgen.New(&webhookQueryStub{rowResults: []pgx.Row{readback}}), "owner", destination, time.Now(), 1); err != nil {
		t.Fatalf("insertAndMatchDestination() error = %v", err)
	}
}

func TestPreparedAcceptanceAndStoreInputGuards(t *testing.T) {
	prepared, err := PrepareAcceptance(goldenAcceptance())
	if err != nil {
		t.Fatal(err)
	}
	if err := validatePrepared(prepared); err != nil {
		t.Fatalf("validatePrepared() error = %v", err)
	}
	prepared.Destinations[0].DeliveryID = ""
	if err := validatePrepared(prepared); !errors.Is(err, ErrConfig) {
		t.Fatalf("validatePrepared(missing delivery) error = %v", err)
	}

	if _, err := NewStore(nil, StoreOptions{}); !errors.Is(err, ErrConfig) {
		t.Fatalf("NewStore(nil) error = %v", err)
	}
	first, err := advisoryKey("owner", "event", "one")
	if err != nil {
		t.Fatal(err)
	}
	second, err := advisoryKey("owner", "event", "two")
	if err != nil {
		t.Fatal(err)
	}
	if first == second {
		t.Fatal("advisory key does not bind resource identity")
	}
	if got := pgtime(time.Date(2024, 1, 2, 3, 4, 5, 0, time.FixedZone("offset", 3600))); !got.Valid || got.Time.Location() != time.UTC {
		t.Fatalf("pgtime() = %+v", got)
	}
}

func TestOperatorMutationInputGuards(t *testing.T) {
	now := time.Unix(1700000000, 0)
	guardQueries := sqlcgen.New(&webhookQueryStub{})
	for _, request := range []ActionRequest{
		{Kind: ActionDestinationState, Expected: "invalid", Values: []string{"disabled", ""}},
		{Kind: ActionKeyRotation, Expected: "0", Values: []string{"bad", "", "", "", "", "", ""}},
		{Kind: ActionRedrive, Expected: "0", DuplicateRisk: true, Values: []string{"bad", "bad"}},
	} {
		if _, _, err := applyActionMutation(t.Context(), guardQueries, request, now); !errors.Is(err, ErrConfig) {
			t.Fatalf("applyActionMutation(%s) error = %v", request.Kind, err)
		}
	}
	result, cycle, err := applyActionMutation(t.Context(), guardQueries, ActionRequest{Kind: ActionRedrive, Expected: "0", Values: []string{"1", "1"}}, now)
	if err != nil || result != "rejected" || cycle != 0 {
		t.Fatalf("unacknowledged redrive = %q, %d, %v", result, cycle, err)
	}
	if result, _, err := actionMutationResult(0, nil); result != "state_conflict" || err != nil {
		t.Fatalf("actionMutationResult(0) = %q, %v", result, err)
	}
	if _, _, err := actionMutationResult(1, errors.New("write failed")); err == nil {
		t.Fatal("actionMutationResult() error = nil")
	}
	request := ActionRequest{Kind: ActionDestinationState, OwnerScope: "owner", TargetID: "destination", TargetGeneration: 1, Expected: "0", Values: []string{"paused", ""}}
	result, cycle, err = applyActionMutation(t.Context(), sqlcgen.New(&webhookQueryStub{execTags: []pgconn.CommandTag{pgconn.NewCommandTag("UPDATE 1")}}), request, now)
	if err != nil || result != "applied" || cycle != 0 {
		t.Fatalf("applyActionMutation(destination) = %q, %d, %v", result, cycle, err)
	}
	rotation := ActionRequest{Kind: ActionKeyRotation, OwnerScope: "owner", TargetID: "destination", TargetGeneration: 1, Expected: "0", Values: []string{"1", "2", "key-new", "key-old", "1699999999", "1700000100", "rotation-receipt"}}
	result, cycle, err = applyActionMutation(t.Context(), sqlcgen.New(&webhookQueryStub{execTags: []pgconn.CommandTag{pgconn.NewCommandTag("UPDATE 1")}}), rotation, now)
	if err != nil || result != "applied" || cycle != 0 {
		t.Fatalf("applyActionMutation(rotation) = %q, %d, %v", result, cycle, err)
	}
	redrive := ActionRequest{Kind: ActionRedrive, OwnerScope: "owner", TargetID: "delivery", Expected: "2", DuplicateRisk: true, Values: []string{"3", "3600000000000"}}
	result, cycle, err = applyActionMutation(t.Context(), sqlcgen.New(&webhookQueryStub{
		rowResults: []pgx.Row{webhookActionDeliveryRow(now, string(OutcomeAttemptsExhausted))},
		execTags:   []pgconn.CommandTag{pgconn.NewCommandTag("INSERT 1"), pgconn.NewCommandTag("UPDATE 1")},
	}), redrive, now)
	if err != nil || result != "applied" || cycle != 3 {
		t.Fatalf("applyActionMutation(redrive) = %q, %d, %v", result, cycle, err)
	}
	closeUnknown := ActionRequest{Kind: ActionCloseUnknown, OwnerScope: "owner", TargetID: "delivery", Expected: "2", Values: []string{"closed_unknown"}}
	result, cycle, err = applyActionMutation(t.Context(), sqlcgen.New(&webhookQueryStub{
		rowResults: []pgx.Row{webhookActionDeliveryRow(now, string(OutcomeUnknown))},
		execTags:   []pgconn.CommandTag{pgconn.NewCommandTag("UPDATE 1"), pgconn.NewCommandTag("UPDATE 1")},
	}), closeUnknown, now)
	if err != nil || result != "applied" || cycle != 0 {
		t.Fatalf("applyActionMutation(close unknown) = %q, %d, %v", result, cycle, err)
	}
}

func TestTombstoneReplayRecognizesConflictsAndReplays(t *testing.T) {
	request := ActionRequest{OwnerScope: "owner", Actor: "actor", ActionID: "privacy", Kind: ActionPrivacyDelete, TargetKind: "event", TargetID: "event", Reason: "privacy", Values: []string{"event", "event", "minimal_tombstone", "ticket"}}
	fingerprint, err := request.Fingerprint()
	if err != nil {
		t.Fatal(err)
	}
	if _, handled, err := tombstoneReplay(t.Context(), sqlcgen.New(&webhookQueryStub{rowResults: []pgx.Row{webhookRowStub{err: pgx.ErrNoRows}}}), request, fingerprint); handled || err != nil {
		t.Fatalf("tombstoneReplay(absent) = handled:%t err:%v", handled, err)
	}

	conflictRow := webhookScanRow(func(destinations ...any) error {
		return setScanValue(destinations, 7, []byte("different"))
	})
	if _, handled, err := tombstoneReplay(t.Context(), sqlcgen.New(&webhookQueryStub{rowResults: []pgx.Row{conflictRow}}), request, fingerprint); !handled || !errors.Is(err, ErrConflict) {
		t.Fatalf("tombstoneReplay(conflict) = handled:%t err:%v", handled, err)
	}

	replayRow := webhookScanRow(func(destinations ...any) error {
		return setScanValues(destinations, scanValue{7, fingerprint[:]}, scanValue{8, "completed"})
	})
	receipt, handled, err := tombstoneReplay(t.Context(), sqlcgen.New(&webhookQueryStub{rowResults: []pgx.Row{replayRow}}), request, fingerprint)
	if err != nil || !handled || !receipt.Replay || receipt.Result != "completed" {
		t.Fatalf("tombstoneReplay(replay) = %+v, handled:%t err:%v", receipt, handled, err)
	}
}
