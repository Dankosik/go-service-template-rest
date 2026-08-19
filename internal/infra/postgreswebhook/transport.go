package postgreswebhook

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptrace"
	"net/netip"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/example/go-service-template-rest/internal/outboundtrust"
)

const (
	webhookUserAgent       = "go-service-template-webhook/1"
	maxDNSAddresses        = 64
	responseHeaderTimeout  = 15 * time.Second
	maxResponseHeaderBytes = 32 << 10
	maxResponseBodyBytes   = 64 << 10
)

type DeliveryAttempt struct {
	ID                   string
	OwnerScope           string
	ReceiverID           string
	URL                  string
	Body                 []byte
	AttemptedAt          time.Time
	Deadline             time.Time
	KeyReference         string
	PredecessorReference string
}

type PreparedSend struct {
	Attempt         DeliveryAttempt
	URL             *url.URL
	Addresses       []netip.Addr
	SelectedAddress netip.Addr
	Signature       string
}

type SendResult struct {
	Evidence            TransportEvidence
	ResponseHeaderBytes int
	ResponseBodyBytes   int
	RetryAfter          string
	ResponseDate        string
}

func PrepareSend(ctx context.Context, resolver *net.Resolver, attempt DeliveryAttempt, manifest *SecretManifest) (PreparedSend, error) {
	if resolver == nil || manifest == nil || !attemptContextBounded(ctx, attempt.Deadline) {
		return PreparedSend{}, fmt.Errorf("%w: resolver, secret manifest, and bounded attempt deadline are required", ErrConfig)
	}
	parsed, err := parseWebhookURL(attempt.URL)
	if err != nil {
		return PreparedSend{}, err
	}
	addresses, err := resolver.LookupNetIP(ctx, "ip", parsed.Hostname())
	if err != nil {
		return PreparedSend{}, fmt.Errorf("resolve webhook destination: %w", err)
	}
	if len(addresses) == 0 || len(addresses) > maxDNSAddresses {
		return PreparedSend{}, fmt.Errorf("%w: destination returned an invalid address count", ErrDestinationDenied)
	}
	for i := range addresses {
		addresses[i] = addresses[i].Unmap()
		if !outboundtrust.PublicAddress(addresses[i]) {
			return PreparedSend{}, fmt.Errorf("%w: destination answer contains a non-public address", ErrDestinationDenied)
		}
	}
	slices.SortFunc(addresses, func(a, b netip.Addr) int { return bytes.Compare(a.AsSlice(), b.AsSlice()) })
	addresses = slices.Compact(addresses)

	active, err := manifest.Resolve(attempt.OwnerScope, attempt.ReceiverID, attempt.KeyReference)
	if err != nil {
		return PreparedSend{}, err
	}
	keys := []SigningKey{active}
	if attempt.PredecessorReference != "" {
		predecessor, err := manifest.Resolve(attempt.OwnerScope, attempt.ReceiverID, attempt.PredecessorReference)
		if err != nil {
			return PreparedSend{}, err
		}
		keys = append(keys, predecessor)
	}
	signature, err := SignV1(attempt.ID, attempt.AttemptedAt, attempt.Body, keys)
	if err != nil {
		return PreparedSend{}, err
	}
	return PreparedSend{
		Attempt: attempt, URL: parsed, Addresses: slices.Clone(addresses),
		SelectedAddress: addresses[0], Signature: signature,
	}, nil
}

func tryPreparedAddresses(
	ctx context.Context,
	prepared PreparedSend,
	send func(context.Context, PreparedSend) (SendResult, error),
) (SendResult, error) {
	if len(prepared.Addresses) == 0 || send == nil {
		return SendResult{Evidence: TransportEvidence{DefinitelyNotSent: true, LocalDenial: true}}, ErrConfig
	}
	var result SendResult
	var err error
	for _, address := range prepared.Addresses {
		prepared.SelectedAddress = address
		result, err = send(ctx, prepared)
		if err == nil || !result.Evidence.DefinitelyNotSent {
			return result, err
		}
	}
	return result, err
}

func Send(ctx context.Context, prepared PreparedSend) (SendResult, error) {
	if prepared.URL == nil || !prepared.SelectedAddress.IsValid() || !attemptContextBounded(ctx, prepared.Attempt.Deadline) {
		return SendResult{Evidence: TransportEvidence{DefinitelyNotSent: true, LocalDenial: true}}, fmt.Errorf("%w: prepared send is invalid", ErrConfig)
	}
	wroteRequest := false
	trace := &httptrace.ClientTrace{WroteRequest: func(httptrace.WroteRequestInfo) { wroteRequest = true }}
	attemptCtx := httptrace.WithClientTrace(ctx, trace)
	transport := newAttemptTransport(prepared.URL.Hostname(), prepared.SelectedAddress)
	defer transport.CloseIdleConnections()
	return sendWithTransport(attemptCtx, prepared, transport, &wroteRequest)
}

func attemptContextBounded(ctx context.Context, attemptDeadline time.Time) bool {
	deadline, ok := ctx.Deadline()
	return ok && !attemptDeadline.IsZero() && !deadline.After(attemptDeadline)
}

func sendWithTransport(ctx context.Context, prepared PreparedSend, transport *http.Transport, wroteRequest *bool) (SendResult, error) {
	request, err := webhookRequest(ctx, prepared)
	if err != nil {
		return SendResult{Evidence: TransportEvidence{DefinitelyNotSent: true, LocalDenial: true}}, err
	}
	client := &http.Client{Transport: transport, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	response, err := client.Do(request)
	if err != nil {
		wrote := wroteRequest != nil && *wroteRequest
		if errors.Is(err, http.ErrLineTooLong) || strings.Contains(err.Error(), "server response headers exceeded") {
			err = ErrResponseLimit
		}
		return SendResult{Evidence: TransportEvidence{
			DefinitelyNotSent: !wrote, MayHaveSent: wrote,
			LocalDenial: !wrote && permanentTLSValidationError(err),
		}}, fmt.Errorf("send webhook request: %w", err)
	}
	defer func() { _ = response.Body.Close() }()
	result := SendResult{
		Evidence:   TransportEvidence{StatusCode: response.StatusCode, MayHaveSent: true},
		RetryAfter: response.Header.Get("Retry-After"), ResponseDate: response.Header.Get("Date"),
		ResponseHeaderBytes: responseHeaderBytes(response.Header),
	}
	if result.ResponseHeaderBytes > maxResponseHeaderBytes {
		result.ResponseHeaderBytes = maxResponseHeaderBytes
		return result, ErrResponseLimit
	}
	if response.StatusCode >= 200 && response.StatusCode <= 299 {
		return result, nil
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, maxResponseBodyBytes+1))
	result.ResponseBodyBytes = len(body)
	if err != nil {
		return result, fmt.Errorf("receive webhook response body: %w", err)
	}
	if len(body) > maxResponseBodyBytes {
		result.ResponseBodyBytes = maxResponseBodyBytes
		return result, ErrResponseLimit
	}
	return result, nil
}

func permanentTLSValidationError(err error) bool {
	var verification *tls.CertificateVerificationError
	var unknownAuthority x509.UnknownAuthorityError
	var hostname x509.HostnameError
	var invalidCertificate x509.CertificateInvalidError
	return errors.As(err, &verification) || errors.As(err, &unknownAuthority) ||
		errors.As(err, &hostname) || errors.As(err, &invalidCertificate)
}

func webhookRequest(ctx context.Context, prepared PreparedSend) (*http.Request, error) {
	body := io.NopCloser(bytes.NewReader(prepared.Attempt.Body))
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, prepared.URL.String(), body)
	if err != nil {
		return nil, fmt.Errorf("build webhook request: %w", err)
	}
	request.ContentLength = int64(len(prepared.Attempt.Body))
	request.Header = http.Header{
		"Content-Type":      []string{"application/json"},
		"Accept-Encoding":   []string{"identity"},
		"User-Agent":        []string{webhookUserAgent},
		"Webhook-Id":        []string{prepared.Attempt.ID},
		"Webhook-Timestamp": []string{strconv.FormatInt(prepared.Attempt.AttemptedAt.Unix(), 10)},
		"Webhook-Signature": []string{prepared.Signature},
	}
	return request, nil
}

func newAttemptTransport(serverName string, address netip.Addr) *http.Transport {
	dialer := &net.Dialer{}
	return &http.Transport{
		Proxy: nil, DisableKeepAlives: true, DisableCompression: true, ForceAttemptHTTP2: false,
		MaxConnsPerHost: 1, MaxIdleConns: 0, ResponseHeaderTimeout: responseHeaderTimeout,
		MaxResponseHeaderBytes: maxResponseHeaderBytes,
		TLSHandshakeTimeout:    responseHeaderTimeout,
		TLSClientConfig:        &tls.Config{ServerName: serverName, MinVersion: tls.VersionTLS13},
		TLSNextProto:           map[string]func(string, *tls.Conn) http.RoundTripper{},
		DialContext: func(ctx context.Context, network, _ string) (net.Conn, error) {
			if !outboundtrust.PublicAddress(address) {
				return nil, ErrDestinationDenied
			}
			return dialer.DialContext(ctx, network, net.JoinHostPort(address.String(), "443"))
		},
	}
}

func parseWebhookURL(raw string) (*url.URL, error) {
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Scheme != "https" || parsed.Hostname() == "" || parsed.User != nil ||
		parsed.RawQuery != "" || parsed.Fragment != "" || parsed.Port() != "" && parsed.Port() != "443" || len(raw) > 2048 {
		return nil, fmt.Errorf("%w: destination URL must be absolute HTTPS on port 443", ErrDestinationDenied)
	}
	if parsed.Port() == "" {
		parsed.Host = net.JoinHostPort(parsed.Hostname(), "443")
	}
	return parsed, nil
}

func responseHeaderBytes(headers http.Header) int {
	total := 0
	for name, values := range headers {
		for _, value := range values {
			total += len(name) + len(value) + 4
		}
	}
	return total
}
