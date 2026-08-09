package oidcjwt

// Proof for verifier.go: what Verify answers once trust is established — key
// rotation, coalesced and rate-limited recovery, staleness, cancellation, and
// the redaction every one of those paths owes. Startup lives in provider_test.go
// and the lifecycle Run and Close own lives in lifecycle_test.go.

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

func TestKeyMissRefresh(t *testing.T) {
	now := testNow
	first := loadTestRSAKey(t, testSigningKey)
	second := loadTestRSAKey(t, testRotatedKey)
	rotatedJWKS := jwksDocument(t, second, "key-2")
	client := &scriptedClient{responses: append(initialResponses(t, first), scriptedResponse{
		status: http.StatusOK,
		body:   rotatedJWKS,
	})}
	verifier := requireTestVerifier(t, testVerifierOptions{now: newTestClock(now).now, client: client})
	token := signToken(t, second, "key-2", "at+jwt", validClaims(now))
	principal, err := verifier.verify(t.Context(), token, transportHTTP)
	if err != nil || principal.Subject == "" {
		t.Fatalf("Verify(rotated token) = (%+v, %v), want success", principal, err)
	}
	requireProviderCalls(t, client, 1, "after a rotated key id")

	outage := &scriptedClient{responses: append(initialResponses(t, first), scriptedResponse{
		err: errors.New("poison issuer outage"),
	})}
	outageVerifier := requireTestVerifier(t, testVerifierOptions{now: newTestClock(now).now, client: outage})
	unknown := signToken(t, second, "unknown", "at+jwt", validClaims(now))
	_, err = outageVerifier.verify(t.Context(), unknown, transportHTTP)
	requireKind(t, err, KindInvalid)
	valid := signToken(t, first, "key-1", "at+jwt", validClaims(now))
	principal, err = outageVerifier.verify(t.Context(), valid, transportHTTP)
	if err != nil || principal.Subject == "" {
		t.Fatalf("Verify(cached valid token after outage) = (%+v, %v), want success", principal, err)
	}
}

func TestSameKIDRotation(t *testing.T) {
	now := testNow
	first := loadTestRSAKey(t, testSigningKey)
	second := loadTestRSAKey(t, testRotatedKey)
	client := &scriptedClient{responses: append(initialResponses(t, first),
		scriptedResponse{status: http.StatusOK, body: jwksDocument(t, second, "key-1")},
		scriptedResponse{err: errors.New("sequential poison outage")},
		scriptedResponse{err: errors.New("post-cooldown poison outage")},
	)}
	clock := newTestClock(now)
	verifier := requireTestVerifier(t, testVerifierOptions{now: clock.now, client: client})

	rotated := signToken(t, second, "key-1", "at+jwt", validClaims(now))
	if principal, verifyErr := verifier.verify(t.Context(), rotated, transportHTTP); verifyErr != nil ||
		principal.Subject == "" {
		t.Fatalf("Verify(same-kid rotation) = (%+v, %v), want success", principal, verifyErr)
	}
	requireProviderCalls(t, client, 1, "after same-kid rotation")

	unknown := signToken(t, first, "unknown", "at+jwt", validClaims(now))
	_, err := verifier.verify(t.Context(), unknown, transportHTTP)
	requireKind(t, err, KindInvalid)
	requireProviderCalls(t, client, 1, "during rotation cooldown")

	clock.advance(RefreshCooldown)
	_, err = verifier.verify(t.Context(), unknown, transportHTTP)
	requireKind(t, err, KindInvalid)
	requireProviderCalls(t, client, 2, "at the cooldown boundary")
	_, err = verifier.verify(t.Context(), unknown, transportHTTP)
	requireKind(t, err, KindInvalid)
	requireProviderCalls(t, client, 2, "inside the next cooldown")
}

func TestStaleUnknownKIDPerformsRequestDrivenRecovery(t *testing.T) {
	now := testNow
	first := loadTestRSAKey(t, testSigningKey)
	second := loadTestRSAKey(t, testRotatedKey)
	client := &scriptedClient{responses: append(initialResponses(t, first), scriptedResponse{
		status: http.StatusOK,
		body:   jwksDocument(t, second, "key-2"),
	})}
	clock := newTestClock(now)
	verifier := requireTestVerifier(t, testVerifierOptions{now: clock.now, client: client})
	clock.advance(MaxKeySetAge)

	token := signToken(t, second, "key-2", "at+jwt", validClaims(clock.now()))
	principal, err := verifier.verify(t.Context(), token, transportHTTP)
	if err != nil || principal.Subject == "" {
		t.Fatalf("Verify(stale unknown-kid recovery) = (%+v, %v), want success", principal, err)
	}
	requireProviderCalls(t, client, 1, "after request-driven recovery from a stale set")
}

// TestUnknownKIDCategoryDependsOnTrustCurrentness pins the distinction Verify
// draws once a refresh has not produced a usable set: invalid against a current
// set, unavailable against one the refresh could not replace. [Verifier.Verify]
// states why.
func TestUnknownKIDCategoryDependsOnTrustCurrentness(t *testing.T) {
	first := loadTestRSAKey(t, testSigningKey)
	second := loadTestRSAKey(t, testRotatedKey)
	for _, testCase := range []struct {
		name     string
		stale    bool
		wantKind Kind
	}{
		{name: "current set answers invalid", wantKind: KindInvalid},
		{name: "stale set answers unavailable", stale: true, wantKind: KindUnavailable},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			client := &scriptedClient{responses: append(initialResponses(t, first), scriptedResponse{
				err: errors.New("poison refresh outage"),
			})}
			clock := newTestClock(testNow)
			verifier := requireTestVerifier(t, testVerifierOptions{now: clock.now, client: client})
			if testCase.stale {
				clock.advance(MaxKeySetAge)
			}

			unknown := signToken(t, second, "unknown", "at+jwt", validClaims(clock.now()))
			_, err := verifier.verify(t.Context(), unknown, transportHTTP)
			requireKind(t, err, testCase.wantKind)
			requireProviderCalls(t, client, 1, "after a refresh the provider failed")
		})
	}
}

func TestRefreshCoalescing(t *testing.T) {
	now := testNow
	first := loadTestRSAKey(t, testSigningKey)
	second := loadTestRSAKey(t, testRotatedKey)
	release := make(chan struct{})
	started := make(chan struct{})
	client := &scriptedClient{responses: append(initialResponses(t, first), scriptedResponse{
		status:  http.StatusOK,
		body:    jwksDocument(t, second, "key-2"),
		wait:    release,
		started: started,
	})}
	verifier := requireTestVerifier(t, testVerifierOptions{now: newTestClock(now).now, client: client})
	token := signToken(t, second, "key-2", "at+jwt", validClaims(now))

	var successful atomic.Int64
	var wait sync.WaitGroup
	for range 20 {
		wait.Go(func() {
			if _, err := verifier.verify(context.Background(), token, transportHTTP); err == nil {
				successful.Add(1)
			}
		})
	}
	<-started
	close(release)
	wait.Wait()
	if successful.Load() != 20 {
		t.Fatalf("successful verifications = %d, want 20", successful.Load())
	}
	requireProviderCalls(t, client, 1, "after 20 concurrent key misses")
}

func TestRefreshCancellation(t *testing.T) {
	now := testNow
	first := loadTestRSAKey(t, testSigningKey)
	second := loadTestRSAKey(t, testRotatedKey)
	release := make(chan struct{})
	started := make(chan struct{})
	client := &scriptedClient{responses: append(initialResponses(t, first), scriptedResponse{
		status:  http.StatusOK,
		body:    jwksDocument(t, second, "key-2"),
		wait:    release,
		started: started,
	})}
	verifier := requireTestVerifier(t, testVerifierOptions{now: newTestClock(now).now, client: client})
	token := signToken(t, second, "key-2", "at+jwt", validClaims(now))
	ctx, cancel := context.WithCancel(context.Background())
	result := make(chan error, 1)
	go func() {
		_, err := verifier.verify(ctx, token, transportHTTP)
		result <- err
	}()
	<-started
	cancel()
	if err := <-result; !errors.Is(err, context.Canceled) {
		t.Fatalf("Verify() error = %v, want context cancellation", err)
	}
	close(release)
}

func TestStaleKeySetFailsReadinessAndVerificationClosed(t *testing.T) {
	now := testNow
	key := loadTestRSAKey(t, testSigningKey)
	client := &scriptedClient{responses: initialResponses(t, key)}
	clock := newTestClock(now)
	verifier := requireTestVerifier(t, testVerifierOptions{now: clock.now, client: client})
	clock.advance(MaxKeySetAge)

	requireKind(t, verifier.CheckReady(), KindUnavailable)
	token := signToken(t, key, "key-1", "at+jwt", validClaims(clock.now()))
	_, err := verifier.verify(t.Context(), token, transportHTTP)
	requireKind(t, err, KindUnavailable)
}

func TestErrorsAndLogsRedactCredentialAndProviderContent(t *testing.T) {
	const sensitive = "sensitive-token-and-claim"
	var output strings.Builder
	client := &scriptedClient{responses: []scriptedResponse{{
		status: http.StatusBadGateway,
		body:   []byte(sensitive),
		err:    errors.New(sensitive),
	}}}
	_, err := buildTestVerifier(t, testVerifierOptions{
		now:    time.Now,
		client: client,
		log:    slog.New(slog.NewTextHandler(&output, nil)),
	})
	if err == nil {
		t.Fatal("newVerifier() error = nil, want provider failure")
	}
	if strings.Contains(err.Error(), sensitive) || strings.Contains(output.String(), sensitive) {
		t.Fatalf("sensitive provider content escaped: error=%q log=%q", err, output.String())
	}

	verifier := newTestVerifier(t, loadTestRSAKey(t, testSigningKey))
	_, err = verifier.verify(t.Context(), sensitive, transportHTTP)
	if err == nil || strings.Contains(err.Error(), sensitive) {
		t.Fatalf("credential escaped through verification error: %v", err)
	}
}

func TestAuthnRedactionCoversRefreshPanicAndTelemetry(t *testing.T) {
	const poison = "poison-authn-marker-7c5e"
	now := testNow
	first := loadTestRSAKey(t, testSigningKey)
	second := loadTestRSAKey(t, testRotatedKey)
	reader := sdkmetric.NewManualReader()
	provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	t.Cleanup(func() {
		if err := provider.Shutdown(t.Context()); err != nil {
			t.Errorf("shutdown metric provider: %v", err)
		}
	})
	var output strings.Builder
	client := &scriptedClient{responses: append(initialResponses(t, first), scriptedResponse{panic: poison})}
	clock := newTestClock(now)
	verifier := requireTestVerifier(t, testVerifierOptions{
		now:      clock.now,
		client:   client,
		provider: provider,
		log:      slog.New(slog.NewJSONHandler(&output, nil)),
	})

	claims := validClaims(now)
	claims.Subject = poison
	claims.JWTID = poison
	token := signToken(t, second, poison, "at+jwt", claims)
	_, err := verifier.verify(t.Context(), token, transportHTTP)
	requireKind(t, err, KindInvalid)
	// Rendered rather than dereferenced: requireKind has already proved this
	// error non-nil, and formatting keeps that proof out of the nil checker's way.
	if strings.Contains(fmt.Sprint(err), poison) || strings.Contains(output.String(), poison) {
		t.Fatalf("refresh panic escaped through error or logs: error=%q logs=%q", err, output.String())
	}

	clock.setFunc(func() time.Time { panic(poison) })
	_, err = verifier.verify(t.Context(), token, transportHTTP)
	// The key-age gauge reads the same clock, and this case is about a panic
	// inside verification rather than inside telemetry, so the failure is scoped
	// to the call above before the collection below.
	clock.set(now)
	requireKind(t, err, KindUnavailable)
	if strings.Contains(fmt.Sprint(err), poison) || strings.Contains(output.String(), poison) {
		t.Fatalf("verification panic escaped through error or logs: error=%q logs=%q", err, output.String())
	}

	var metrics metricdata.ResourceMetrics
	if err := reader.Collect(t.Context(), &metrics); err != nil {
		t.Fatalf("collect authn metrics: %v", err)
	}
	if encoded := fmt.Sprintf("%#v", metrics); strings.Contains(encoded, poison) {
		t.Fatalf("authn metrics contain poison marker: %s", encoded)
	}

	// Redaction is half the contract; the other half is that the conversion is
	// reported at all. The closed categories keep telemetry safe, while the log
	// is the evidence that locates the service defect.
	// logRecoveredPanic owns that argument.
	logged := output.String()
	if !strings.Contains(logged, "authn_panic_recovered") {
		t.Errorf("no recovered panic was reported: %s", logged)
	}
	for _, operation := range []string{"jwks_refresh", "verify"} {
		if !strings.Contains(logged, operation) {
			t.Errorf("the recovered panic in %q was converted silently: %s", operation, logged)
		}
	}
}
