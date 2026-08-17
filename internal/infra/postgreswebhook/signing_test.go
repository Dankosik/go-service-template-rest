package postgreswebhook

import (
	"encoding/hex"
	"testing"
	"time"
)

func TestWebhookSigningContract(t *testing.T) {
	t.Parallel()
	key := []byte("0123456789abcdef0123456789abcdef")
	header, evidence, err := SignV1("test_delivery_01", time.Unix(1700000000, 0), []byte(`{"event":"order.created","id":"evt_01"}`), []SigningKey{{Reference: "key-1", Bytes: key}})
	if err != nil {
		t.Fatal(err)
	}
	const want = "v1,p75LLTWzwS12ldSqW8PVMMr3coNo6m83PGJu/jVFO0U="
	if header != want {
		t.Fatalf("header = %q, want %q", header, want)
	}
	if got := hex.EncodeToString(evidence[:]); got != "9291ae82facaa14a94ee4b97afcc53014a2cbcb84db650467b2f1c6358d1032f" {
		t.Fatalf("evidence = %s", got)
	}
	if !VerifyV1("test_delivery_01", "1700000000", []byte(`{"event":"order.created","id":"evt_01"}`), header, [][]byte{key}) {
		t.Fatal("signature did not verify")
	}
}
