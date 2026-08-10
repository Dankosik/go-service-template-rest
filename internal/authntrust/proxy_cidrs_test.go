package authntrust_test

import (
	"net/netip"
	"slices"
	"testing"

	"github.com/example/go-service-template-rest/internal/authntrust"
)

func TestParseProxyCIDRsAccepts(t *testing.T) {
	t.Parallel()

	for _, testCase := range []struct {
		name string
		raw  string
		want []string
	}{
		{name: "single", raw: "10.0.0.0/8", want: []string{"10.0.0.0/8"}},
		{name: "mixed families", raw: "127.0.0.0/8,::1/128", want: []string{"127.0.0.0/8", "::1/128"}},
		{name: "surrounding space", raw: " 10.0.0.0/8 , ::1/128 ", want: []string{"10.0.0.0/8", "::1/128"}},
		{name: "trailing comma", raw: "10.0.0.0/8,", want: []string{"10.0.0.0/8"}},
		// The stored spelling is the masked one, which is what lets the two
		// callers agree: one persists this answer and the other re-parses it.
		{name: "host bits are masked away", raw: "10.20.30.40/8", want: []string{"10.0.0.0/8"}},
		{name: "order is kept", raw: "192.168.0.0/16,10.0.0.0/8", want: []string{"192.168.0.0/16", "10.0.0.0/8"}},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			parsed, err := authntrust.ParseProxyCIDRs(testCase.raw)
			if err != nil {
				t.Fatalf("ParseProxyCIDRs(%q) error = %v, want success", testCase.raw, err)
			}
			got := make([]string, 0, len(parsed))
			for _, prefix := range parsed {
				got = append(got, prefix.String())
			}
			if !slices.Equal(got, testCase.want) {
				t.Fatalf("ParseProxyCIDRs(%q) = %v, want %v", testCase.raw, got, testCase.want)
			}
		})
	}
}

func TestParseProxyCIDRsRejects(t *testing.T) {
	t.Parallel()

	for _, testCase := range []struct {
		name string
		raw  string
	}{
		{name: "empty list", raw: ""},
		{name: "only separators", raw: " , , "},
		{name: "bare address", raw: "127.0.0.1"},
		{name: "not an address", raw: "not-a-cidr/8"},
		{name: "prefix length out of range", raw: "10.0.0.0/33"},
		{name: "IPv4 wildcard", raw: "0.0.0.0/0"},
		{name: "IPv6 wildcard", raw: "::/0"},
		{name: "duplicate", raw: "10.0.0.0/8,10.0.0.0/8"},
		// Masking is what makes these one entry rather than two, so the duplicate
		// is only visible after it.
		{name: "duplicate after masking", raw: "10.0.0.0/8,10.20.30.40/8"},
		{name: "one bad entry poisons the list", raw: "10.0.0.0/8,nope"},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			parsed, err := authntrust.ParseProxyCIDRs(testCase.raw)
			if err == nil {
				t.Fatalf("ParseProxyCIDRs(%q) = %v, nil; want an error", testCase.raw, parsed)
			}
			if parsed != nil {
				t.Fatalf("ParseProxyCIDRs(%q) returned %v alongside an error, want nil", testCase.raw, parsed)
			}
		})
	}
}

// TestParseProxyCIDRsMatchesPeers is the property the trusted-proxy rule exists
// for: the parse a caller stores answers the membership question the other
// caller asks. Unmap is the transport adapter's job, so this covers only the
// prefixes themselves.
func TestParseProxyCIDRsMatchesPeers(t *testing.T) {
	t.Parallel()

	parsed, err := authntrust.ParseProxyCIDRs("10.0.0.0/8,::1/128")
	if err != nil {
		t.Fatalf("ParseProxyCIDRs() error = %v, want success", err)
	}
	for _, testCase := range []struct {
		peer    string
		trusted bool
	}{
		{peer: "10.20.30.40", trusted: true},
		{peer: "::1", trusted: true},
		{peer: "11.0.0.1"},
		{peer: "::2"},
	} {
		address := netip.MustParseAddr(testCase.peer)
		contains := slices.ContainsFunc(parsed, func(prefix netip.Prefix) bool {
			return prefix.Contains(address)
		})
		if contains != testCase.trusted {
			t.Fatalf("prefixes contain %q = %v, want %v", testCase.peer, contains, testCase.trusted)
		}
	}
}
