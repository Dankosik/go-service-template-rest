package outboundtrust

import (
	"net/netip"
	"testing"
)

func TestPublicAddressCorpus(t *testing.T) {
	t.Parallel()

	tests := map[string]bool{
		"8.8.8.8": true, "2606:4700:4700::1111": true,
		"192.0.0.9": true, "2001:1::1": true, "64:ff9b::808:808": true,
		"10.0.0.1": false, "100.64.0.1": false, "127.0.0.1": false,
		"169.254.169.254": false, "192.0.2.1": false, "198.18.0.1": false,
		"2001:db8::1": false, "2001:2::1": false, "3fff::1": false,
		"4000::1": false, "64:ff9b::a00:1": false, "::ffff:127.0.0.1": false,
	}
	for raw, want := range tests {
		t.Run(raw, func(t *testing.T) {
			t.Parallel()
			if got := PublicAddress(netip.MustParseAddr(raw)); got != want {
				t.Fatalf("PublicAddress(%s) = %t, want %t", raw, got, want)
			}
		})
	}
}
