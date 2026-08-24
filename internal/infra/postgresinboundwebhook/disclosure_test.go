// profile:inbound-webhooks-standard:start
package postgresinboundwebhook

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"testing"

	"github.com/example/go-service-template-rest/internal/inboundwebhook"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"
)

const disclosureCanary = "provider-canary-text"

func TestInboundWebhookDisclosureFormats(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name      string
		decodeErr error
		handle    func(context.Context, inboundwebhook.VerifiedDelivery, json.RawMessage) error
		store     *memoryStore
		attempt   int
		max       int
		want      error
	}{
		{
			name: "handler",
			handle: func(context.Context, inboundwebhook.VerifiedDelivery, json.RawMessage) error {
				return errors.New(disclosureCanary)
			},
			store:   &memoryStore{receipt: pendingReceipt()},
			attempt: 1, max: 3, want: errHandlerFailed,
		},
		{
			name:      "decoder",
			decodeErr: errors.New(disclosureCanary),
			handle:    func(context.Context, inboundwebhook.VerifiedDelivery, json.RawMessage) error { return nil },
			store:     &memoryStore{receipt: pendingReceipt()},
			attempt:   1, max: 3, want: errDecoderFailed,
		},
		{
			name: "panic",
			handle: func(context.Context, inboundwebhook.VerifiedDelivery, json.RawMessage) error {
				panic(disclosureCanary)
			},
			store:   &memoryStore{receipt: pendingReceipt()},
			attempt: 1, max: 3, want: errPanicRecovered,
		},
		{
			name: "storage",
			handle: func(context.Context, inboundwebhook.VerifiedDelivery, json.RawMessage) error {
				return nil
			},
			store:   &memoryStore{receipt: pendingReceipt(), failHandled: true},
			attempt: 1, max: 3, want: errStorageUnavailable,
		},
		{
			name:      "quarantine",
			decodeErr: inboundwebhook.ErrDecodeRejected,
			handle:    func(context.Context, inboundwebhook.VerifiedDelivery, json.RawMessage) error { return nil },
			store:     &memoryStore{receipt: pendingReceipt()},
			attempt:   1, max: 3,
		},
		{
			name:    "terminalization",
			handle:  func(context.Context, inboundwebhook.VerifiedDelivery, json.RawMessage) error { return nil },
			store:   &memoryStore{receipt: pendingReceipt(), failTerminal: true},
			attempt: 3, max: 3,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var logged bytes.Buffer
			tc.store.receipt.Payload = []byte(`{"secret":"` + disclosureCanary + `"}`)
			worker, err := newWorker(tc.store, testRegistry(t, tc.handle, tc.decodeErr), newTelemetry(nil, slog.New(slog.NewJSONHandler(&logged, nil))))
			if err != nil {
				t.Fatal(err)
			}
			workErr := worker.Work(context.Background(), &river.Job[receiptJobArgs]{
				Args:   receiptJobArgs{ReceiptID: "rcpt_1"},
				JobRow: &rivertype.JobRow{Attempt: tc.attempt, MaxAttempts: tc.max},
			})
			sinks := []string{logged.String()}
			if workErr != nil {
				sinks = append(sinks,
					fmt.Sprintf("%s", workErr),
					fmt.Sprintf("%v", workErr),
					fmt.Sprintf("%+v", workErr),
					workErr.Error(),
				)
			}
			for _, formatted := range sinks {
				if strings.Contains(formatted, disclosureCanary) {
					t.Fatalf("canary leaked: %q", formatted)
				}
			}
			if tc.want != nil && !errors.Is(workErr, tc.want) {
				t.Fatalf("err = %v, want %v", workErr, tc.want)
			}
		})
	}
}

// profile:inbound-webhooks-standard:end
