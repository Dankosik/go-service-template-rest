// profile:inbound-webhooks-standard:start
package bootstrap

import (
	"context"
	"strings"
	"testing"
)

func TestInboundWebhookWorkerNilBuilderStillFailsBeforeConfig(t *testing.T) {
	err := run(context.Background(), []string{"--config", "/does/not/exist"}, nil)
	if err == nil || !strings.Contains(err.Error(), "worker builder") {
		t.Fatalf("run() error = %v", err)
	}
}

// profile:inbound-webhooks-standard:end
