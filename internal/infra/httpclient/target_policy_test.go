package httpclient

import (
	"context"
	"errors"
	"net"
	"net/http"
	"sync/atomic"
	"testing"
	"time"

	metricnoop "go.opentelemetry.io/otel/metric/noop"
)

// profile:outbound-auth-oauth2-client-credentials:start
func TestPrivateHTTPSTargetPolicy(t *testing.T) {
	t.Parallel()

	cfg := validExternalConfig()
	cfg.BaseURL = "HTTPS://AUTH.SERVICE.INTERNAL/oauth/token"
	cfg.TargetClass = PrivateHTTPS
	cfg.PrivateHostSuffix = ".service.internal"
	client, err := New(cfg, nil)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	t.Cleanup(client.CloseIdleConnections)
	if client.BaseURL() != "https://auth.service.internal/oauth/token" {
		t.Fatalf("BaseURL() = %q", client.BaseURL())
	}

	for _, test := range []struct {
		name    string
		baseURL string
		suffix  string
	}{
		{name: "plaintext", baseURL: "http://auth.service.internal/oauth/token", suffix: "service.internal"},
		{name: "missing suffix", baseURL: "https://auth.service.internal/oauth/token"},
		{name: "wrong suffix", baseURL: "https://auth.service.internal/oauth/token", suffix: "other.internal"},
		{name: "public hostname", baseURL: "https://example.com/oauth/token", suffix: "service.internal"},
		{name: "private literal", baseURL: "https://10.0.0.1/oauth/token", suffix: "service.internal"},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			invalid := cfg
			invalid.BaseURL = test.baseURL
			invalid.PrivateHostSuffix = test.suffix
			if _, err := New(invalid, nil); err == nil {
				t.Fatal("New() error = nil, want rejection")
			}
		})
	}

	for _, address := range []string{"10.0.0.1:443", "[fd00::1]:443"} {
		if err := enforceDialAddress(PrivateHTTPS, address); err != nil {
			t.Fatalf("enforceDialAddress(%q) error = %v", address, err)
		}
	}
	for _, address := range []string{"8.8.8.8:443", "127.0.0.1:443", "100.64.0.1:443"} {
		if err := enforceDialAddress(PrivateHTTPS, address); !errors.Is(err, ErrTargetDenied) {
			t.Fatalf("enforceDialAddress(%q) error = %v, want %v", address, err, ErrTargetDenied)
		}
	}
}

// profile:outbound-auth-oauth2-client-credentials:end

func TestTargetClassAdmission(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		mutate func(*Config)
	}{
		{name: "external HTTP", mutate: func(cfg *Config) { cfg.BaseURL = "http://example.com" }},
		{name: "external private literal", mutate: func(cfg *Config) { cfg.BaseURL = "https://127.0.0.1" }},
		{name: "private HTTPS under HTTP policy", mutate: func(cfg *Config) {
			cfg.BaseURL = "https://api.railway.internal"
			cfg.TargetClass = PrivateHTTP
			cfg.PrivateHostSuffix = "railway.internal"
		}},
		{name: "private public host", mutate: func(cfg *Config) {
			cfg.BaseURL = "http://example.com"
			cfg.TargetClass = PrivateHTTP
			cfg.PrivateHostSuffix = "railway.internal"
		}},
		{name: "private suffix is required", mutate: func(cfg *Config) {
			cfg.BaseURL = "http://api.railway.internal"
			cfg.TargetClass = PrivateHTTP
			cfg.PrivateHostSuffix = ""
		}},
		{name: "private host outside configured suffix", mutate: func(cfg *Config) {
			cfg.BaseURL = "http://api.railway.internal"
			cfg.TargetClass = PrivateHTTP
			cfg.PrivateHostSuffix = "svc.cluster.local"
		}},
		{name: "private suffix cannot be dot only", mutate: func(cfg *Config) {
			cfg.BaseURL = "http://api.railway.internal"
			cfg.TargetClass = PrivateHTTP
			cfg.PrivateHostSuffix = "."
		}},
		{name: "invalid target class", mutate: func(cfg *Config) { cfg.TargetClass = 99 }},
		{name: "external rejects HTTP even with suffix", mutate: func(cfg *Config) {
			cfg.BaseURL = "http://provider.example"
			cfg.PrivateHostSuffix = "provider.example"
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			cfg := validExternalConfig()
			test.mutate(&cfg)
			if _, err := New(cfg, nil); err == nil {
				t.Fatal("New() error = nil, want target-policy rejection")
			}
		})
	}
}

func TestAuthorityTransportAllowsOnlyConfiguredAuthority(t *testing.T) {
	t.Parallel()

	var calls atomic.Int64
	transport := authorityTransport{
		base: roundTripFunc(func(*http.Request) (*http.Response, error) {
			calls.Add(1)
			return responseWithBody("ok"), nil
		}),
		scheme:    "https",
		authority: "api.example.com",
	}

	allowed, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "https://api.example.com/v1/items", http.NoBody)
	if err != nil {
		t.Fatal(err)
	}
	response, err := transport.RoundTrip(allowed)
	if err != nil {
		t.Fatalf("allowed RoundTrip() error = %v", err)
	}
	_ = response.Body.Close()

	for _, target := range []string{
		"http://api.example.com/v1/items",
		"https://other.example.com/v1/items",
		"https://user@api.example.com/v1/items",
	} {
		request, requestErr := http.NewRequestWithContext(context.Background(), http.MethodGet, target, http.NoBody)
		if requestErr != nil {
			t.Fatal(requestErr)
		}
		response, roundTripErr := transport.RoundTrip(request)
		if response != nil {
			_ = response.Body.Close()
		}
		if !errors.Is(roundTripErr, ErrTargetDenied) {
			t.Fatalf("RoundTrip(%q) error = %v, want %v", target, roundTripErr, ErrTargetDenied)
		}
	}

	if got := calls.Load(); got != 1 {
		t.Fatalf("base transport calls = %d, want 1", got)
	}
}

func TestDialAddressGate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		targetClass TargetClass
		address     string
		allowed     bool
	}{
		{name: "external public IPv4", targetClass: ExternalHTTPS, address: "8.8.8.8:443", allowed: true},
		{name: "external public IPv6", targetClass: ExternalHTTPS, address: "[2606:4700:4700::1111]:443", allowed: true},
		{name: "external public NAT64", targetClass: ExternalHTTPS, address: "[64:ff9b::808:808]:443", allowed: true},
		{name: "external public IPv4 special", targetClass: ExternalHTTPS, address: "192.0.0.9:443", allowed: true},
		{name: "external public IPv6 special", targetClass: ExternalHTTPS, address: "[2001:1::1]:443", allowed: true},
		{name: "external private", targetClass: ExternalHTTPS, address: "10.0.0.1:443"},
		{name: "external shared", targetClass: ExternalHTTPS, address: "100.64.0.1:443"},
		{name: "external loopback", targetClass: ExternalHTTPS, address: "127.0.0.1:443"},
		{name: "external metadata", targetClass: ExternalHTTPS, address: "169.254.169.254:80"},
		{name: "external protocol assignment", targetClass: ExternalHTTPS, address: "192.0.0.8:443"},
		{name: "external documentation IPv4", targetClass: ExternalHTTPS, address: "192.0.2.1:443"},
		{name: "external benchmark IPv4", targetClass: ExternalHTTPS, address: "198.18.0.1:443"},
		{name: "external documentation IPv6", targetClass: ExternalHTTPS, address: "[2001:db8::1]:443"},
		{name: "external benchmark IPv6", targetClass: ExternalHTTPS, address: "[2001:2::1]:443"},
		{name: "external documentation IPv6 future", targetClass: ExternalHTTPS, address: "[3fff::1]:443"},
		{name: "external unallocated IPv6", targetClass: ExternalHTTPS, address: "[4000::1]:443"},
		{name: "external NAT64 private", targetClass: ExternalHTTPS, address: "[64:ff9b::a00:1]:443"},
		{name: "external multicast", targetClass: ExternalHTTPS, address: "224.0.0.1:443"},
		{name: "external broadcast", targetClass: ExternalHTTPS, address: "255.255.255.255:443"},
		{name: "external unspecified", targetClass: ExternalHTTPS, address: "[::]:443"},
		{name: "private HTTP RFC1918", targetClass: PrivateHTTP, address: "10.0.0.1:80", allowed: true},
		{name: "private HTTP ULA", targetClass: PrivateHTTP, address: "[fd00::1]:80", allowed: true},
		// profile:outbound-auth-oauth2-client-credentials:start
		{name: "private HTTPS RFC1918", targetClass: PrivateHTTPS, address: "10.0.0.1:443", allowed: true},
		{name: "private HTTPS ULA", targetClass: PrivateHTTPS, address: "[fd00::1]:443", allowed: true},
		// profile:outbound-auth-oauth2-client-credentials:end
		{name: "private public", targetClass: PrivateHTTP, address: "8.8.8.8:80"},
		{name: "private shared", targetClass: PrivateHTTP, address: "100.64.0.1:80"},
		{name: "private loopback", targetClass: PrivateHTTP, address: "127.0.0.1:80"},
		{name: "hostname", targetClass: ExternalHTTPS, address: "example.com:443"},
		{name: "invalid", targetClass: ExternalHTTPS, address: "invalid"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			err := enforceDialAddress(test.targetClass, test.address)
			if test.allowed && err != nil {
				t.Fatalf("enforceDialAddress() error = %v, want nil", err)
			}
			if !test.allowed && !errors.Is(err, ErrTargetDenied) {
				t.Fatalf("enforceDialAddress() error = %v, want %v", err, ErrTargetDenied)
			}
		})
	}
}

func TestExternalClientRejectsPrivateDNSResolutionBeforeConnect(t *testing.T) {
	t.Parallel()

	var listenConfig net.ListenConfig
	listener, err := listenConfig.Listen(context.Background(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = listener.Close() })

	_, port, err := net.SplitHostPort(listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	cfg := validExternalConfig()
	cfg.BaseURL = "https://localhost:" + port
	client, err := New(cfg, metricnoop.NewMeterProvider())
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	request, err := http.NewRequestWithContext(context.Background(), http.MethodGet, client.BaseURL(), http.NoBody)
	if err != nil {
		t.Fatal(err)
	}
	response, requestErr := client.Do(request)
	if response != nil {
		_ = response.Body.Close()
	}
	if !errors.Is(requestErr, ErrTargetDenied) {
		t.Fatalf("Do() error = %v, want %v", requestErr, ErrTargetDenied)
	}

	tcpListener, ok := listener.(*net.TCPListener)
	if !ok {
		t.Fatalf("listener = %T, want *net.TCPListener", listener)
	}
	if err := tcpListener.SetDeadline(time.Now().Add(100 * time.Millisecond)); err != nil {
		t.Fatal(err)
	}
	if connection, acceptErr := listener.Accept(); acceptErr == nil {
		_ = connection.Close()
		t.Fatal("private target accepted a connection")
	}
}

func TestNewAcceptsCustomPrivateHostSuffix(t *testing.T) {
	t.Parallel()

	for _, test := range []struct {
		name    string
		suffix  string
		baseURL string
	}{
		{name: "kubernetes cluster zone", suffix: "svc.cluster.local", baseURL: "http://billing.default.svc.cluster.local"},
		{name: "leading dot accepted", suffix: ".internal", baseURL: "http://billing.internal"},
		{name: "railway zone", suffix: "railway.internal", baseURL: "http://billing.railway.internal"},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			cfg := validExternalConfig()
			cfg.TargetClass = PrivateHTTP
			cfg.PrivateHostSuffix = test.suffix
			cfg.BaseURL = test.baseURL

			client, err := New(cfg, metricnoop.NewMeterProvider())
			if err != nil {
				t.Fatalf("New() error = %v, want nil", err)
			}
			t.Cleanup(client.CloseIdleConnections)
		})
	}
}
