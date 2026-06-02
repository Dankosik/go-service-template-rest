package eventsv1

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

var prohibitedEventPayloadTerms = []string{
	"raw_prompt",
	"raw_completion",
	"sse_chunk",
	"bearer_token",
	"api_key",
	"dsn",
	"payment_secret",
	"raw_provider_payload",
	"raw_event_payload",
	"dynamic_proof_url",
	"sensitive_request_body",
}

func TestGeneratedMicroleaseEventDTOsExposeSafeLineage(t *testing.T) {
	t.Parallel()

	event := MicroleaseChildTerminalSubmitted{
		Envelope: MicroleaseEventEnvelope{
			EventID:          "evt_01",
			EventType:        "MicroleaseChildTerminalSubmitted",
			ContractVersion:  "v1",
			SchemaVersion:    "2026-06-01",
			ProducerIdentity: "svc:gonka-proxy",
			EventFingerprint: "fingerprint_01",
		},
		Identity: MicroleaseIdentity{
			MicroleaseID:          "ml_01",
			AccountScopeKey:       "acct_scope_support_safe",
			ProxyAllocatorOwnerID: "owner_01",
			MicroleaseGeneration:  1,
		},
		DebitAuthorizationID:     "debit_01",
		ChildCapUSDAtoms:         100,
		TerminalKind:             "finalize",
		RequestBasisFingerprint:  "request_basis_01",
		TerminalBasisFingerprint: "terminal_basis_01",
		Pricing: PricingSnapshotBasis{
			PricingSnapshotID:   "price_01",
			SnapshotFingerprint: "price_fingerprint_01",
			PolicyVersion:       "pricing_policy_v1",
			SelectorKey:         "pricing_selector_v1:GNK:USDT:spot:quote_per_base_unit",
			UseClass:            "usage_reserve",
			ContractVersion:     "v1",
		},
		SafeExecutionReference: "exec_ref_01",
	}

	if event.Envelope.ProducerIdentity != "svc:gonka-proxy" {
		t.Fatalf("producer identity = %q, want svc:gonka-proxy", event.Envelope.ProducerIdentity)
	}
	if event.Pricing.SnapshotFingerprint == "" {
		t.Fatal("pricing snapshot fingerprint is empty")
	}
}

func TestMicroleaseEventContractsExcludeSensitivePayloadFields(t *testing.T) {
	t.Parallel()

	paths := []string{
		filepath.Join("..", "..", "..", "..", "api", "proto", "events", "v1", "microlease_events.proto"),
		filepath.Join("testdata", "microlease_terminal_submitted.safe.json"),
	}
	for _, path := range paths {
		t.Run(path, func(t *testing.T) {
			t.Parallel()

			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read %s: %v", path, err)
			}
			lower := strings.ToLower(string(data))
			for _, term := range prohibitedEventPayloadTerms {
				if strings.Contains(lower, term) {
					t.Fatalf("%s contains prohibited payload term %q", path, term)
				}
			}
		})
	}
}
