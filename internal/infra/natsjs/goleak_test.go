package natsjs

import (
	"fmt"
	"os"
	"testing"

	"github.com/example/go-service-template-rest/internal/infra/natsjs/natsjstest"
	"go.uber.org/goleak"
)

var packageNATSPool natsjstest.Pool

func TestMain(m *testing.M) {
	code := m.Run()
	if err := packageNATSPool.Close(); err != nil {
		fmt.Fprintf(os.Stderr, "terminate shared NATS container: %v\n", err)
		code = 1
	}
	if code == 0 {
		if err := goleak.Find(); err != nil {
			fmt.Fprintf(os.Stderr, "goleak: Errors on successful test run: %v\n", err)
			code = 1
		}
	}
	os.Exit(code)
}
