package background

import (
	"testing"

	"go.uber.org/goleak"
)

// TestMain fails the package when a supervised task outlives its supervisor. The
// join is the whole contract, so a leak here is a defect in the thing under test
// rather than test noise.
func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m)
}
