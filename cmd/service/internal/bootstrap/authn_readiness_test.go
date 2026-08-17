package bootstrap

import "testing"

func TestAuthnReadinessComposition(t *testing.T) {
	server := newFakeGRPCRuntimeServer()
	for _, current := range []bool{true, false, true} {
		setGRPCAuthnReady(server, current)
		select {
		case got := <-server.authnReadiness:
			if got != current {
				t.Fatalf("published authn readiness = %t, want %t", got, current)
			}
		default:
			t.Fatalf("authn readiness %t was not published", current)
		}
	}

	setGRPCAuthnReady(nil, false)
}
