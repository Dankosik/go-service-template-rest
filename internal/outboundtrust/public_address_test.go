package outboundtrust

import (
	"net/netip"
	"testing"
)

func TestPublicAddressCorpus(t *testing.T) {
	t.Parallel()
	if IANASpecialPurposeRegistryRevision != "2025-10-09" {
		t.Fatalf("IANA registry revision = %q", IANASpecialPurposeRegistryRevision)
	}

	tests := map[string]bool{
		"8.8.8.8": true, "2606:4700:4700::1111": true,
		"192.0.0.9": true, "192.0.0.10": true, "192.31.196.1": true, "192.52.193.1": true, "192.175.48.1": true,
		"0.0.0.1": false, "10.0.0.1": false, "100.64.0.1": false, "127.0.0.1": false, "169.254.169.254": false,
		"172.16.0.1": false, "192.0.0.8": false, "192.0.0.170": false, "192.0.2.1": false,
		"192.88.99.2": false, "192.168.0.1": false, "198.18.0.1": false, "198.51.100.1": false,
		"203.0.113.1": false, "240.0.0.1": false, "255.255.255.255": false,
		"2001:1::1": true, "2001:1::2": true, "2001:1::3": true, "2001:3::1": true,
		"2001:4:112::1": true, "2001:20::1": true, "2001:30::1": true, "2620:4f:8000::1": true,
		"64:ff9b::808:808": true, "64:ff9b:1::1": false, "100::1": false, "100:0:0:1::1": false,
		"2001::1": false, "2001:2::1": false, "2001:10::1": false, "2001:db8::1": false,
		"2002::1": false, "3fff::1": false, "5f00::1": false, "fc00::1": false, "fe80::1": false,
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
