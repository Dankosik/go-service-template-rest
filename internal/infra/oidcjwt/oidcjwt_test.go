package oidcjwt

// Fixtures shared by every test in this package: the scripted provider, the
// movable clock, the verifier constructors, and the token and JWKS builders.
// They live here rather than beside one suite because nearly every test file
// depends on them, and a reader opening any one of them should not have to know
// which other file owns the verifier under test.

import (
	"bytes"
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math/big"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	jose "github.com/go-jose/go-jose/v4"
	"go.opentelemetry.io/otel/metric"
)

const (
	testIssuer   = "https://issuer.example.com"
	testAudience = "service-api"
	testJWKSURI  = "https://keys.example.net/jwks"
)

// The fixed RSA keys every case signs with. Two are enough to state both kinds
// of rotation: a new key id, and a new key under the id already installed.
const (
	testSigningKey = "test-key-1.pem"
	testRotatedKey = "test-key-2.pem"
)

// testNow is the instant cases build tokens and key sets around. Fixing it
// makes an expiry or staleness assertion read as an offset from one known
// point rather than from whenever the suite happened to run.
var testNow = time.Unix(1_900_000_000, 0)

// bootstrapProviderCalls is what initialResponses scripts: one Discovery
// request and one JWKS request. Every provider-call assertion in this package
// is stated as this baseline plus the refreshes the case expects, so a change
// to startup does not have to be re-counted into unrelated tests.
const bootstrapProviderCalls = 2

// testClock is a clock a test can move, or make fail, after the verifier has
// captured it. The indirection is atomic because a refresh goroutine may read
// the clock while the test changes it; a plain time.Time closed over by the
// clock function is a data race as soon as a refresh is in flight.
type testClock struct {
	read atomic.Pointer[func() time.Time]
}

func newTestClock(start time.Time) *testClock {
	clock := &testClock{}
	clock.set(start)
	return clock
}

// now is the clock function handed to newVerifier.
func (c *testClock) now() time.Time {
	return (*c.read.Load())()
}

func (c *testClock) set(value time.Time) {
	c.setFunc(func() time.Time { return value })
}

func (c *testClock) advance(delta time.Duration) {
	c.set(c.now().Add(delta))
}

// setFunc replaces how the clock answers, for a test that needs reading the
// time to fail rather than to return a different instant.
func (c *testClock) setFunc(read func() time.Time) {
	c.read.Store(&read)
}

type scriptedResponse struct {
	status  int
	body    []byte
	err     error
	wait    <-chan struct{}
	started chan<- struct{}
	panic   any
}

type scriptedClient struct {
	mu        sync.Mutex
	responses []scriptedResponse
	calls     int
}

func (c *scriptedClient) Do(request *http.Request) (*http.Response, error) {
	c.mu.Lock()
	c.calls++
	if len(c.responses) == 0 {
		c.mu.Unlock()
		return nil, errors.New("poison network error")
	}
	response := c.responses[0]
	c.responses = c.responses[1:]
	c.mu.Unlock()
	if response.started != nil {
		close(response.started)
	}
	if response.wait != nil {
		select {
		case <-response.wait:
		case <-request.Context().Done():
			return nil, fmt.Errorf("scripted provider request: %w", request.Context().Err())
		}
	}
	select {
	case <-request.Context().Done():
		return nil, fmt.Errorf("scripted provider request: %w", request.Context().Err())
	default:
	}
	if response.panic != nil {
		panic(response.panic)
	}
	if response.err != nil {
		return nil, response.err
	}
	return &http.Response{
		StatusCode: response.status,
		Body:       io.NopCloser(strings.NewReader(string(response.body))),
		Header:     make(http.Header),
	}, nil
}

func (c *scriptedClient) callCount() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.calls
}

// testVerifierOptions varies the parts of a verifier one case cares about. The
// zero value builds a verifier on a fixed clock at testNow, served by a
// provider scripted with one successful bootstrap signed by testSigningKey,
// with telemetry and logs discarded — so a case states only what it changes.
type testVerifierOptions struct {
	// bootstrapCtx supplies the context trust is established under. It is a
	// function rather than a context so these options stay an ordinary value a
	// case can declare inline; a case that does not care leaves it nil and
	// bootstrap runs under t.Context().
	bootstrapCtx func() context.Context
	now          func() time.Time
	client       *scriptedClient
	provider     metric.MeterProvider
	log          *slog.Logger
	// onClose counts or observes the release of an owned provider client. The
	// bootstrap client is released before newVerifier returns and the JWKS
	// client on Close, so a case watching this sees both.
	onClose func()
}

// buildTestVerifier returns exactly what newVerifier returned. It is the only
// place a test wires the provider-client factory, so a case whose subject is a
// startup failure, a cancelled bootstrap, or a close count states just the one
// input that causes it instead of restating the whole construction.
func buildTestVerifier(t *testing.T, opts testVerifierOptions) (*Verifier, error) {
	t.Helper()
	bootstrapCtx := opts.bootstrapCtx
	if bootstrapCtx == nil {
		bootstrapCtx = t.Context
	}
	now := opts.now
	if now == nil {
		now = newTestClock(testNow).now
	}
	client := opts.client
	if client == nil {
		client = &scriptedClient{responses: initialResponses(t, loadTestRSAKey(t, testSigningKey))}
	}
	onClose := opts.onClose
	if onClose == nil {
		onClose = func() {}
	}
	return newVerifier(
		bootstrapCtx(),
		testPolicy(t),
		func(string) (providerClient, error) {
			return providerClient{request: client, close: onClose}, nil
		},
		now,
		opts.provider,
		opts.log,
	)
}

// requireTestVerifier builds a verifier whose trust is established and closes
// it with the test, failing when bootstrap does not succeed.
func requireTestVerifier(t *testing.T, opts testVerifierOptions) *Verifier {
	t.Helper()
	verifier, err := buildTestVerifier(t, opts)
	if err != nil {
		t.Fatalf("newVerifier() error = %v", err)
	}
	t.Cleanup(verifier.Close)
	return verifier
}

// newTestVerifier is the shorthand for the most common subject: a verifier fixed
// at testNow, served by a provider that answers one successful bootstrap signed
// by key. The clock is not a parameter because a case that has to move it states
// its own options through requireTestVerifier instead, which keeps what varies
// visible at the call site rather than encoded in a constructor name.
func newTestVerifier(t *testing.T, key *rsa.PrivateKey) *Verifier {
	t.Helper()
	return requireTestVerifier(t, testVerifierOptions{
		now:    newTestClock(testNow).now,
		client: &scriptedClient{responses: initialResponses(t, key)},
	})
}

// requireProviderCalls states an expected provider-call count as the bootstrap
// baseline plus the refreshes the case should have caused.
func requireProviderCalls(t *testing.T, client *scriptedClient, refreshes int, when string) {
	t.Helper()
	want := bootstrapProviderCalls + refreshes
	if got := client.callCount(); got != want {
		t.Fatalf(
			"provider calls %s = %d, want %d (%d bootstrap + %d refresh)",
			when, got, want, bootstrapProviderCalls, refreshes,
		)
	}
}

func testPolicy(t *testing.T) Policy {
	t.Helper()
	policy, err := NewPolicy(PolicyInput{
		Issuer:            testIssuer,
		Audience:          testAudience,
		TrustedProxyCIDRs: "127.0.0.0/8,::1/128",
	})
	if err != nil {
		t.Fatalf("NewPolicy() error = %v", err)
	}
	return policy
}

func initialResponses(t *testing.T, key *rsa.PrivateKey) []scriptedResponse {
	t.Helper()
	discovery, err := json.Marshal(discoveryDocument{Issuer: testIssuer, JWKSURI: testJWKSURI})
	if err != nil {
		t.Fatal(err)
	}
	return []scriptedResponse{
		{status: http.StatusOK, body: discovery},
		{status: http.StatusOK, body: jwksDocument(t, key, "key-1")},
	}
}

type jwkEntry struct {
	key   *rsa.PrivateKey
	keyID string
}

func jwksDocument(t *testing.T, key *rsa.PrivateKey, keyID string) []byte {
	t.Helper()
	return jwksDocumentOf(t, jwkEntry{key: key, keyID: keyID})
}

// jwksDocumentOf builds a JWKS from the given entries, so a test can state an
// ambiguous or multi-key set as data rather than by splicing JSON text.
func jwksDocumentOf(t *testing.T, entries ...jwkEntry) []byte {
	t.Helper()
	keys := make([]json.RawMessage, 0, len(entries))
	for _, entry := range entries {
		encoded, err := publicJWK(&entry.key.PublicKey, entry.keyID)
		if err != nil {
			t.Fatal(err)
		}
		keys = append(keys, encoded)
	}
	document, err := json.Marshal(map[string]any{"keys": keys})
	if err != nil {
		t.Fatalf("marshal JWKS document: %v", err)
	}
	return document
}

func publicJWK(key *rsa.PublicKey, keyID string) ([]byte, error) {
	if key == nil {
		return nil, errors.New("nil RSA key")
	}
	exponent := big.NewInt(int64(key.E)).Bytes()
	encoded, err := json.Marshal(map[string]string{
		"kty": "RSA",
		"kid": keyID,
		"alg": AllowedAlgorithm,
		"use": "sig",
		"n":   base64.RawURLEncoding.EncodeToString(key.N.Bytes()),
		"e":   base64.RawURLEncoding.EncodeToString(exponent),
	})
	if err != nil {
		return nil, fmt.Errorf("marshal public JWK: %w", err)
	}
	return encoded, nil
}

func loadTestRSAKey(t *testing.T, name string) *rsa.PrivateKey {
	t.Helper()
	encoded, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatalf("read fixed test key: %v", err)
	}
	block, rest := pem.Decode(encoded)
	if block == nil || block.Type != "PRIVATE KEY" || len(rest) != 0 {
		t.Fatalf("decode fixed test key %q", name)
	}
	parsed, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		t.Fatalf("parse fixed test key: %v", err)
	}
	key, ok := parsed.(*rsa.PrivateKey)
	if !ok || key.N.BitLen() != 2048 {
		t.Fatalf("fixed test key %q is not RSA-2048", name)
	}
	return key
}

type tokenClaims struct {
	Issuer    string   `json:"iss"`
	Subject   string   `json:"sub,omitempty"`
	Audience  any      `json:"aud"`
	ClientID  string   `json:"client_id"`
	JWTID     string   `json:"jti"`
	Expires   int64    `json:"exp"`
	IssuedAt  int64    `json:"iat"`
	NotBefore *int64   `json:"nbf,omitempty"`
	Scope     []string `json:"scope,omitempty"`
	Roles     []string `json:"roles,omitempty"`
}

func validClaims(now time.Time) tokenClaims {
	return tokenClaims{
		Issuer:   testIssuer,
		Subject:  "opaque-subject",
		Audience: []string{"another", testAudience},
		ClientID: "client-1",
		JWTID:    "token-1",
		Expires:  now.Add(5 * time.Minute).Unix(),
		IssuedAt: now.Unix(),
		Scope:    []string{"admin"},
		Roles:    []string{"owner"},
	}
}

// claimsMap renders a claim set as the map a case mutates when it needs a value
// tokenClaims cannot express: a JSON number that is not an integer, or an
// explicit null. Deriving it from validClaims is what keeps one definition of a
// complete claim set — a term added to parseAccessTokenClaims is added to
// tokenClaims once, rather than once per test that builds a payload by hand.
//
// Decoding through UseNumber keeps every integer exactly as it was written, so
// a re-marshalled exp is still the NumericDate the verifier requires.
func claimsMap(t *testing.T, claims tokenClaims) map[string]any {
	t.Helper()
	encoded, err := json.Marshal(claims)
	if err != nil {
		t.Fatalf("marshal claims: %v", err)
	}
	decoder := json.NewDecoder(bytes.NewReader(encoded))
	decoder.UseNumber()
	var rendered map[string]any
	if err := decoder.Decode(&rendered); err != nil {
		t.Fatalf("decode claims into a map: %v", err)
	}
	// Callers index into the result, so a payload that decoded to a JSON null
	// fails here rather than at the assignment that would panic.
	if rendered == nil {
		t.Fatal("claims decoded to a nil map")
	}
	return rendered
}

func signToken(t *testing.T, key *rsa.PrivateKey, keyID, typ string, claims tokenClaims) string {
	t.Helper()
	return signTokenWithHeader(t, key, map[jose.HeaderKey]any{"typ": typ, "kid": keyID}, claims)
}

func signTokenWithHeader(
	t *testing.T,
	key *rsa.PrivateKey,
	headers map[jose.HeaderKey]any,
	claims tokenClaims,
) string {
	t.Helper()
	options := &jose.SignerOptions{}
	for name, value := range headers {
		options.WithHeader(name, value)
	}
	payload, err := json.Marshal(claims)
	if err != nil {
		t.Fatal(err)
	}
	return signPayloadWithOptions(t, key, options, payload)
}

func signPayload(t *testing.T, key *rsa.PrivateKey, payload []byte) string {
	t.Helper()
	options := (&jose.SignerOptions{}).WithHeader("typ", "at+jwt").WithHeader("kid", "key-1")
	return signPayloadWithOptions(t, key, options, payload)
}

func signPayloadWithOptions(
	t *testing.T,
	key *rsa.PrivateKey,
	options *jose.SignerOptions,
	payload []byte,
) string {
	t.Helper()
	signer, err := jose.NewSigner(jose.SigningKey{Algorithm: jose.RS256, Key: key}, options)
	if err != nil {
		t.Fatalf("jose.NewSigner() error = %v", err)
	}
	signed, err := signer.Sign(payload)
	if err != nil {
		t.Fatalf("Sign() error = %v", err)
	}
	compact, err := signed.CompactSerialize()
	if err != nil {
		t.Fatalf("CompactSerialize() error = %v", err)
	}
	return compact
}

func requireKind(t *testing.T, err error, want Kind) {
	t.Helper()
	if want == 0 {
		if err != nil {
			t.Fatalf("error = %v, want nil", err)
		}
		return
	}
	got, ok := KindOf(err)
	if !ok || got != want {
		t.Fatalf("error = %v, kind = (%v, %v), want %v", err, got, ok, want)
	}
}
