package httpx

import "testing"

func TestOTelServerNameBoundsAuthorityLabels(t *testing.T) {
	t.Parallel()

	for _, tt := range []struct {
		configured string
		want       string
	}{
		{configured: "orders-api", want: "orders-api:0"},
		{configured: " orders-api ", want: "orders-api:0"},
		{want: "service:0"},
	} {
		if got := otelServerName(tt.configured); got != tt.want {
			t.Fatalf("otelServerName(%q) = %q, want %q", tt.configured, got, tt.want)
		}
	}
}
