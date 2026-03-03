package ping

import "testing"

func TestServicePong(t *testing.T) {
	svc := New()
	if svc == nil {
		t.Fatal("New() returned nil")
	}

	if got := svc.Pong(); got != "pong" {
		t.Fatalf("Pong() = %q, want %q", got, "pong")
	}
}
