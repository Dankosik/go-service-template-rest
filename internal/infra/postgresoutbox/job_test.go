package postgresoutbox

import (
	"slices"
	"testing"

	"github.com/riverqueue/river/rivertype"
)

func TestPublishJobRemainsUniqueInEveryState(t *testing.T) {
	t.Parallel()

	got := PublishJob{}.InsertOpts().UniqueOpts.ByState
	if want := rivertype.JobStates(); !slices.Equal(got, want) {
		t.Fatalf("unique states = %v, want %v", got, want)
	}
}
