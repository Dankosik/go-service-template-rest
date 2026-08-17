package postgresoutbox

import (
	"errors"
	"math"
	"testing"

	"github.com/example/go-service-template-rest/internal/infra/postgres/sqlcgen"
	"github.com/jackc/pgx/v5/pgtype"
)

func TestOperatorActionCycle(t *testing.T) {
	t.Parallel()
	falseValue := false
	trueValue := true
	unknownClass := classOutcomeUnknown

	poisoned := sqlcgen.OutboxEvent{PoisonedAt: pgtype.Timestamptz{Valid: true}, PublicationUncertain: &falseValue, RedriveCount: 2}
	unknown := sqlcgen.OutboxEvent{
		PoisonedAt: pgtype.Timestamptz{Valid: true}, PublicationUncertain: &trueValue,
		LastErrorClass: &unknownClass, RedriveCount: 4,
	}

	for name, test := range map[string]struct {
		action string
		row    sqlcgen.OutboxEvent
		cycle  int32
		has    bool
		want   error
	}{
		"redrive poison":      {action: actionRedrivePoison, row: poisoned, cycle: 3, has: true},
		"redrive unknown":     {action: actionRedriveUnknown, row: unknown, cycle: 5, has: true},
		"confirm accepted":    {action: actionConfirmAccepted, row: unknown},
		"poison wrong state":  {action: actionRedrivePoison, row: unknown, want: ErrOperatorStateConflict},
		"unknown wrong state": {action: actionRedriveUnknown, row: poisoned, want: ErrOperatorStateConflict},
		"redrive count exhausted": {action: actionRedriveUnknown, row: sqlcgen.OutboxEvent{
			PoisonedAt: pgtype.Timestamptz{Valid: true}, PublicationUncertain: &trueValue,
			LastErrorClass: &unknownClass, RedriveCount: math.MaxInt32,
		}, want: ErrOperatorStateConflict},
		"unknown action": {action: "other", row: unknown, want: ErrConfig},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			cycle, has, err := operatorActionCycle(test.action, test.row)
			if !errors.Is(err, test.want) || cycle != test.cycle || has != test.has {
				t.Fatalf("operatorActionCycle() = (%d, %t, %v), want (%d, %t, %v)", cycle, has, err, test.cycle, test.has, test.want)
			}
		})
	}
}

func TestOperatorRows(t *testing.T) {
	t.Parallel()
	if err := operatorRows("operator action", 1, nil); err != nil {
		t.Fatalf("operatorRows() error = %v", err)
	}
	if err := operatorRows("operator action", 0, nil); !errors.Is(err, ErrOperatorStateConflict) {
		t.Fatalf("operatorRows(0) error = %v, want ErrOperatorStateConflict", err)
	}
	want := errors.New("database unavailable")
	if err := operatorRows("operator action", 1, want); !errors.Is(err, want) {
		t.Fatalf("operatorRows(database error) = %v, want wrapped error", err)
	}
}
