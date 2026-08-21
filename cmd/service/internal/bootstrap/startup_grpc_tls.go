package bootstrap

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"

	"github.com/example/go-service-template-rest/internal/config"
)

// grpcServerTLS builds the settings the gRPC listener presents to callers.
//
// internal/config has already proven that a certificate and key are present and
// that a client CA file accompanies a TLS listener rather than a plaintext one,
// so this function reads those paths rather than re-deciding whether they may be
// set.
func grpcServerTLS(settings config.GRPCTLSConfig, log *slog.Logger) (*tls.Config, error) {
	reloader, err := newCertificateReloader(settings.CertFile, settings.KeyFile, log)
	if err != nil {
		return nil, err
	}

	// TLS 1.3 is a floor rather than a configured value. Every gRPC runtime this
	// listener can serve has supported it for years, docs/grpc/transport-security.md
	// records that floor, the client example in docs/grpc/runtime-and-streaming.md
	// already pins it, and a knob here would exist only to lower it.
	built := &tls.Config{
		MinVersion:     tls.VersionTLS13,
		GetCertificate: reloader.certificate,
	}
	if settings.ClientCAFile == "" {
		return built, nil
	}

	authorities, err := clientCertificateAuthorities(settings.ClientCAFile)
	if err != nil {
		return nil, err
	}
	// Required rather than requested: a configured CA file whose verification a
	// caller may decline is not a trust boundary.
	built.ClientCAs = authorities
	built.ClientAuth = tls.RequireAndVerifyClientCert
	return built, nil
}

// certificateReloader answers each handshake with the newest usable certificate
// pair on disk.
//
// Without it a renewed pair needs a process restart: crypto/tls reads
// Certificates once, so a cert-manager or ACME renewal landing mid-process stays
// invisible until the pod is bounced — and a deployment that does not bounce it
// serves an expired certificate.
//
// The pair is re-stat'ed per handshake rather than watched. A watcher is another
// goroutine with a lifecycle to drain and a platform-specific failure mode,
// while two stats are microseconds against the signature the handshake is about
// to compute anyway.
//
// A rotation writes the two files separately, so a handshake can land between
// them and read a new certificate beside the old key. LoadX509KeyPair refuses
// that pair, and the last good one keeps serving until both halves agree — which
// is also what an unreadable or deleted file gets, because failing every
// handshake would turn one bad write into an outage.
type certificateReloader struct {
	certFile string
	keyFile  string
	log      *slog.Logger

	mu      sync.Mutex
	current *tls.Certificate
	stamp   pairStamp
	// degraded suppresses the record below the first failure, so one bad
	// rotation cannot log at connection rate.
	degraded bool
}

// newCertificateReloader loads the pair once and refuses to build a listener
// that cannot serve, which is the check a deployment relies on: a bad path or an
// unusable key stops the process instead of failing every handshake later.
func newCertificateReloader(certFile, keyFile string, log *slog.Logger) (*certificateReloader, error) {
	// Stamped before the load, never after. A rotation landing between the two
	// then costs one redundant reload on the first handshake; the reverse order
	// would record the new pair's stamp against the old pair's bytes and never
	// reload again.
	stamp, err := stampPair(certFile, keyFile)
	if err != nil {
		return nil, fmt.Errorf("stat gRPC server TLS certificate: %w", err)
	}
	pair, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, fmt.Errorf("load gRPC server TLS certificate: %w", err)
	}

	return &certificateReloader{
		certFile: certFile,
		keyFile:  keyFile,
		log:      log,
		current:  &pair,
		stamp:    stamp,
		degraded: false,
	}, nil
}

// certificate satisfies tls.Config.GetCertificate.
//
// The lock covers the two stats and any reload, not the handshake, so callers
// serialize on filesystem metadata rather than on cryptography. It never returns
// an error: every failure below has a usable pair in hand, and answering with
// one is strictly better for the caller than refusing the connection.
func (r *certificateReloader) certificate(*tls.ClientHelloInfo) (*tls.Certificate, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	stamp, err := stampPair(r.certFile, r.keyFile)
	if err != nil {
		r.degrade("grpc_tls_certificate_unreadable", err)
		return r.current, nil
	}
	if stamp == r.stamp {
		r.markHealthy()
		return r.current, nil
	}
	pair, err := tls.LoadX509KeyPair(r.certFile, r.keyFile)
	if err != nil {
		// The stamp is deliberately not recorded, so a half-written rotation is
		// retried on the next handshake rather than accepted as the current pair.
		r.degrade("grpc_tls_certificate_rejected", err)
		return r.current, nil
	}

	r.current, r.stamp = &pair, stamp
	r.markHealthy()
	return r.current, nil
}

// degrade and markHealthy record only the transitions. A handshake carries no
// request identity — it precedes every RPC on the connection — so these records
// are process-scoped and use the background context deliberately.
func (r *certificateReloader) degrade(event string, err error) {
	if r.degraded {
		return
	}
	r.degraded = true
	r.log.LogAttrs(
		context.Background(),
		slog.LevelError,
		event,
		slog.String("tls.cert_file", r.certFile),
		slog.String("error", err.Error()),
	)
}

func (r *certificateReloader) markHealthy() {
	if !r.degraded {
		return
	}
	r.degraded = false
	r.log.LogAttrs(
		context.Background(),
		slog.LevelInfo,
		"grpc_tls_certificate_recovered",
		slog.String("tls.cert_file", r.certFile),
	)
}

// pairStamp identifies one certificate pair on disk closely enough to decide
// whether reading it again is worth it. Size and modification time are what a
// rotation changes; hashing the bytes would mean the read this exists to avoid.
type pairStamp struct {
	cert fileStamp
	key  fileStamp
}

// fileStamp holds the modification time as Unix nanoseconds so that a pairStamp
// stays comparable with ==. A time.Time carries a location pointer and an
// optional monotonic reading, neither of which belongs in an equality test.
type fileStamp struct {
	size         int64
	modifiedUnix int64
}

func stampPair(certFile, keyFile string) (pairStamp, error) {
	cert, err := stampFile(certFile)
	if err != nil {
		return pairStamp{}, err
	}
	key, err := stampFile(keyFile)
	if err != nil {
		return pairStamp{}, err
	}
	return pairStamp{cert: cert, key: key}, nil
}

func stampFile(name string) (fileStamp, error) {
	info, err := os.Stat(name)
	if err != nil {
		return fileStamp{}, fmt.Errorf("stat %s: %w", name, err)
	}
	return fileStamp{size: info.Size(), modifiedUnix: info.ModTime().UnixNano()}, nil
}

// clientCertificateAuthorities reads the roots a client certificate may chain
// to. A file holding no certificate is refused rather than accepted as an empty
// pool, which would reject every caller at handshake time instead of at startup.
func clientCertificateAuthorities(name string) (*x509.CertPool, error) {
	root, err := os.OpenRoot(filepath.Dir(name))
	if err != nil {
		return nil, fmt.Errorf("open gRPC client CA directory: %w", err)
	}
	defer func() { _ = root.Close() }()

	encoded, err := root.ReadFile(filepath.Base(name))
	if err != nil {
		return nil, fmt.Errorf("read gRPC client CA file: %w", err)
	}
	authorities := x509.NewCertPool()
	if !authorities.AppendCertsFromPEM(encoded) {
		return nil, fmt.Errorf("gRPC client CA file %s contains no certificate", name)
	}
	return authorities, nil
}
