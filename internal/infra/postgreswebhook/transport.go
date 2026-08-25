package postgreswebhook

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
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
)

type deliveryAttempt struct {
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

type preparedSend struct {
	Attempt         deliveryAttempt
	URL             *url.URL
	Addresses       []netip.Addr
	SelectedAddress netip.Addr
	Signature       string
}

type sendResult struct {
	Evidence     transportEvidence
	RetryAfter   string
	ResponseDate string
}

func prepareSend(ctx context.Context, resolver *net.Resolver, attempt deliveryAttempt, manifest *SecretManifest) (preparedSend, error) {
	if resolver == nil || manifest == nil || !attemptContextBounded(ctx, attempt.Deadline) {
		return preparedSend{}, fmt.Errorf("%w: resolver, secret manifest, and bounded attempt deadline are required", ErrConfig)
	}
	parsed, err := parseWebhookURL(attempt.URL)
	if err != nil {
		return preparedSend{}, err
	}
	addresses, err := resolver.LookupNetIP(ctx, "ip", parsed.Hostname())
	if err != nil {
		return preparedSend{}, fmt.Errorf("resolve webhook destination: %w", err)
	}
	if len(addresses) == 0 || len(addresses) > maxDNSAddresses {
		return preparedSend{}, fmt.Errorf("%w: destination returned an invalid address count", errDestinationDenied)
	}
	for i := range addresses {
		addresses[i] = addresses[i].Unmap()
		if !outboundtrust.PublicAddress(addresses[i]) {
			return preparedSend{}, fmt.Errorf("%w: destination answer contains a non-public address", errDestinationDenied)
		}
	}
	slices.SortFunc(addresses, func(a, b netip.Addr) int { return bytes.Compare(a.AsSlice(), b.AsSlice()) })
	addresses = slices.Compact(addresses)

	active, err := manifest.resolve(attempt.OwnerScope, attempt.ReceiverID, attempt.KeyReference)
	if err != nil {
		return preparedSend{}, err
	}
	keys := [][]byte{active}
	if attempt.PredecessorReference != "" {
		predecessor, err := manifest.resolve(attempt.OwnerScope, attempt.ReceiverID, attempt.PredecessorReference)
		if err != nil {
			return preparedSend{}, err
		}
		keys = append(keys, predecessor)
	}
	signature, err := signV1(attempt.ID, attempt.AttemptedAt, attempt.Body, keys)
	if err != nil {
		return preparedSend{}, err
	}
	return preparedSend{
		Attempt: attempt, URL: parsed, Addresses: slices.Clone(addresses),
		SelectedAddress: addresses[0], Signature: signature,
	}, nil
}

func tryPreparedAddresses(
	ctx context.Context,
	prepared preparedSend,
	send func(context.Context, preparedSend) (sendResult, error),
) (sendResult, error) {
	if len(prepared.Addresses) == 0 || send == nil {
		return sendResult{Evidence: transportEvidence{DefinitelyNotSent: true, LocalDenial: true}}, ErrConfig
	}
	var result sendResult
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

func send(ctx context.Context, prepared preparedSend) (sendResult, error) {
	if prepared.URL == nil || !prepared.SelectedAddress.IsValid() || !attemptContextBounded(ctx, prepared.Attempt.Deadline) {
		return sendResult{Evidence: transportEvidence{DefinitelyNotSent: true, LocalDenial: true}}, fmt.Errorf("%w: prepared send is invalid", ErrConfig)
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

func sendWithTransport(ctx context.Context, prepared preparedSend, transport *http.Transport, wroteRequest *bool) (sendResult, error) {
	request, err := webhookRequest(ctx, prepared)
	if err != nil {
		return sendResult{Evidence: transportEvidence{DefinitelyNotSent: true, LocalDenial: true}}, err
	}
	client := &http.Client{Transport: transport, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	response, err := client.Do(request)
	if err != nil {
		wrote := wroteRequest != nil && *wroteRequest
		if errors.Is(err, http.ErrLineTooLong) || strings.Contains(err.Error(), "server response headers exceeded") {
			err = errResponseLimit
		}
		return sendResult{Evidence: transportEvidence{
			DefinitelyNotSent: !wrote, MayHaveSent: wrote,
			LocalDenial: !wrote && permanentTLSValidationError(err),
		}}, fmt.Errorf("send webhook request: %w", err)
	}
	defer func() { _ = response.Body.Close() }()
	result := sendResult{
		Evidence:   transportEvidence{StatusCode: response.StatusCode, MayHaveSent: true},
		RetryAfter: response.Header.Get("Retry-After"), ResponseDate: response.Header.Get("Date"),
	}
	return result, nil
}

func permanentTLSValidationError(err error) bool {
	_, verification := errors.AsType[*tls.CertificateVerificationError](err)
	_, unknownAuthority := errors.AsType[x509.UnknownAuthorityError](err)
	_, hostname := errors.AsType[x509.HostnameError](err)
	_, invalidCertificate := errors.AsType[x509.CertificateInvalidError](err)
	return verification || unknownAuthority || hostname || invalidCertificate
}

func webhookRequest(ctx context.Context, prepared preparedSend) (*http.Request, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, prepared.URL.String(), bytes.NewReader(prepared.Attempt.Body))
	if err != nil {
		return nil, fmt.Errorf("build webhook request: %w", err)
	}
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
				return nil, errDestinationDenied
			}
			return dialer.DialContext(ctx, network, net.JoinHostPort(address.String(), "443"))
		},
	}
}

func parseWebhookURL(raw string) (*url.URL, error) {
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Scheme != "https" || parsed.Hostname() == "" || parsed.User != nil ||
		parsed.RawQuery != "" || parsed.ForceQuery || parsed.Fragment != "" || parsed.Port() != "" && parsed.Port() != "443" || len(raw) > 2048 {
		return nil, fmt.Errorf("%w: destination URL must be absolute HTTPS on port 443", errDestinationDenied)
	}
	if parsed.Port() == "" {
		parsed.Host = net.JoinHostPort(parsed.Hostname(), "443")
	}
	return parsed, nil
}
